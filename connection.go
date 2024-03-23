// Copyright 2024 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ouroboros_mock

import (
	"bytes"
	"fmt"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/muxer"
)

// ProtocolRole is an enum of the protocol roles
type ProtocolRole uint

// Protocol roles
const (
	ProtocolRoleNone   ProtocolRole = 0 // Default (invalid) protocol role
	ProtocolRoleClient ProtocolRole = 1 // Client protocol role
	ProtocolRoleServer ProtocolRole = 2 // Server protocol role
)

// Connection mocks an Ouroboros connection
type Connection struct {
	mockConn      net.Conn
	conn          net.Conn
	conversation  []ConversationEntry
	muxer         *muxer.Muxer
	muxerRecvChan chan *muxer.Segment
	doneChan      chan any
	onceClose     sync.Once
	errorChan     chan error
}

// NewConnection returns a new Connection with the provided conversation entries
func NewConnection(
	protocolRole ProtocolRole,
	conversation []ConversationEntry,
) net.Conn {
	c := &Connection{
		conversation: conversation,
		doneChan:     make(chan any),
		errorChan:    make(chan error, 1),
	}
	c.conn, c.mockConn = net.Pipe()
	// Start a muxer on the mocked side of the connection
	c.muxer = muxer.New(c.mockConn)
	// The muxer is for the opposite end of the connection, so we flip the protocol role
	muxerProtocolRole := muxer.ProtocolRoleResponder
	if protocolRole == ProtocolRoleServer {
		muxerProtocolRole = muxer.ProtocolRoleInitiator
	}
	// We use ProtocolUnknown to catch all inbound messages when no other protocols are registered
	_, c.muxerRecvChan, _ = c.muxer.RegisterProtocol(
		muxer.ProtocolUnknown,
		muxerProtocolRole,
	)
	c.muxer.Start()
	// Start async muxer error handler
	go func() {
		err, ok := <-c.muxer.ErrorChan()
		if !ok {
			return
		}
		c.errorChan <- fmt.Errorf("muxer error: %w", err)
		c.Close()
	}()
	// Start async conversation handler
	go c.asyncLoop()
	return c
}

func (c *Connection) ErrorChan() <-chan error {
	return c.errorChan
}

// Read provides a proxy to the client-side connection's Read function. This is needed to satisfy the net.Conn interface
func (c *Connection) Read(b []byte) (n int, err error) {
	return c.conn.Read(b)
}

// Write provides a proxy to the client-side connection's Write function. This is needed to satisfy the net.Conn interface
func (c *Connection) Write(b []byte) (n int, err error) {
	return c.conn.Write(b)
}

// Close closes both sides of the connection. This is needed to satisfy the net.Conn interface
func (c *Connection) Close() error {
	var retErr error
	c.onceClose.Do(func() {
		close(c.doneChan)
		c.muxer.Stop()
		if err := c.conn.Close(); err != nil {
			retErr = err
			return
		}
		if err := c.mockConn.Close(); err != nil {
			retErr = err
			return
		}
	})
	return retErr
}

// LocalAddr provides a proxy to the client-side connection's LocalAddr function. This is needed to satisfy the net.Conn interface
func (c *Connection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr provides a proxy to the client-side connection's RemoteAddr function. This is needed to satisfy the net.Conn interface
func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// SetDeadline provides a proxy to the client-side connection's SetDeadline function. This is needed to satisfy the net.Conn interface
func (c *Connection) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline provides a proxy to the client-side connection's SetReadDeadline function. This is needed to satisfy the net.Conn interface
func (c *Connection) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline provides a proxy to the client-side connection's SetWriteDeadline function. This is needed to satisfy the net.Conn interface
func (c *Connection) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func (c *Connection) sendError(err error) {
	select {
	case c.errorChan <- err:
		_ = c.Close()
	default:
	}
}

func (c *Connection) asyncLoop() {
	defer func() {
		close(c.errorChan)
	}()
	for _, entry := range c.conversation {
		select {
		case <-c.doneChan:
			return
		default:
		}
		switch entry := entry.(type) {
		case ConversationEntryInput:
			if err := c.processInputEntry(entry); err != nil {
				c.sendError(fmt.Errorf("input error: %w", err))
				return
			}
		case ConversationEntryOutput:
			if err := c.processOutputEntry(entry); err != nil {
				c.sendError(fmt.Errorf("output error: %w", err))
				return
			}
		case ConversationEntryClose:
			c.Close()
		case ConversationEntrySleep:
			time.Sleep(entry.Duration)
		default:
			c.sendError(
				fmt.Errorf(
					"unknown conversation entry type: %T: %#v",
					entry,
					entry,
				),
			)
			return
		}
	}
}

func (c *Connection) processInputEntry(entry ConversationEntryInput) error {
	// Wait for segment to be received from muxer
	segment, ok := <-c.muxerRecvChan
	if !ok {
		return nil
	}
	if segment.GetProtocolId() != entry.ProtocolId {
		return fmt.Errorf(
			"input message protocol ID did not match expected value: expected %d, got %d",
			entry.ProtocolId,
			segment.GetProtocolId(),
		)
	}
	if segment.IsResponse() != entry.IsResponse {
		return fmt.Errorf(
			"input message response flag did not match expected value: expected %v, got %v",
			entry.IsResponse,
			segment.IsResponse(),
		)
	}
	// Determine message type
	msgType, err := cbor.DecodeIdFromList(segment.Payload)
	if err != nil {
		return fmt.Errorf("decode error: %s", err)
	}
	if entry.Message != nil {
		// Create Message object from CBOR
		msg, err := entry.MsgFromCborFunc(uint(msgType), segment.Payload)
		if err != nil {
			return fmt.Errorf("message from CBOR error: %s", err)
		}
		if msg == nil {
			return fmt.Errorf("received unknown message type: %d", msgType)
		}

		// Compare received message to expected message, excluding the cbor content
		//
		// As changing the CBOR of the expected message is not thread-safe, we instead clear the
		// CBOR of the received message
		msg.SetCbor(nil)
		if !reflect.DeepEqual(msg, entry.Message) {
			return fmt.Errorf(
				"parsed message does not match expected value: got %#v, expected %#v",
				msg,
				entry.Message,
			)
		}
	} else {
		if entry.MessageType == uint(msgType) {
			return nil
		}
		return fmt.Errorf("input message is not of expected type: expected %d, got %d", entry.MessageType, msgType)
	}
	return nil
}

func (c *Connection) processOutputEntry(entry ConversationEntryOutput) error {
	payloadBuf := bytes.NewBuffer(nil)
	for _, msg := range entry.Messages {
		// Get raw CBOR from message
		data := msg.Cbor()
		// If message has no raw CBOR, encode the message
		if data == nil {
			var err error
			data, err = cbor.Encode(msg)
			if err != nil {
				return err
			}
		}
		payloadBuf.Write(data)
	}
	segment := muxer.NewSegment(
		entry.ProtocolId,
		payloadBuf.Bytes(),
		entry.IsResponse,
	)
	if err := c.muxer.Send(segment); err != nil {
		return err
	}
	return nil
}
