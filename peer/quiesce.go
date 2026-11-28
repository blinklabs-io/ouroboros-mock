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

package peer

import (
	"context"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/protocol"
)

func waitProtocols(ctx context.Context, protocols ...*protocol.Protocol) error {
	for _, p := range protocols {
		if err := ctx.Err(); err != nil {
			return err
		}
		select {
		case <-p.DoneChan():
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return ctx.Err()
}

func waitServerProtocols(ctx context.Context, conn *ouroboros.Connection) error {
	return waitProtocols(ctx, conn.ChainSync().Server.ProtocolInstance(), conn.BlockFetch().Server.ProtocolInstance(), conn.TxSubmission().Server.ProtocolInstance())
}
