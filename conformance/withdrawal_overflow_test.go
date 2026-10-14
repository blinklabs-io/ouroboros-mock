// Copyright 2026 Blink Labs Software
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

package conformance

import (
	"bytes"
	"math"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/stretchr/testify/assert"
)

func TestDecodeWithdrawalsRejectsAggregateOverflow(t *testing.T) {
	t.Parallel()
	stake := bytes.Repeat([]byte{0x22}, 28)
	account := func(payment byte) cbor.ByteString {
		raw := append([]byte{0x01}, bytes.Repeat([]byte{payment}, 28)...)
		return cbor.NewByteString(append(raw, stake...))
	}
	// Two base addresses with different payment credentials reduce to one
	// stake credential, so their amounts aggregate.
	assert.Nil(t, decodeWithdrawals(map[cbor.ByteString]uint64{
		account(0x11): math.MaxUint64,
		account(0x33): 2,
	}))
	assert.Len(t, decodeWithdrawals(map[cbor.ByteString]uint64{
		account(0x11): math.MaxUint64 - 2,
		account(0x33): 2,
	}), 1)
}
