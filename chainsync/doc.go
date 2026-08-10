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

// Package chainsync provides an observable server-callback test harness for the
// Ouroboros chain-sync mini-protocol.
//
// The harness stands up a real gouroboros chain-sync server (the system under
// test), configured with the caller's own [github.com/blinklabs-io/gouroboros/protocol/chainsync.Config]
// callbacks, and connects it to a lightweight mock client driver over an
// in-memory pipe. Tests drive the server by sending FindIntersect and
// RequestNext, and observe every message the server sends back — RollForward,
// RollBackward, AwaitReply, IntersectFound, and IntersectNotFound — decoded and
// delivered over a channel. All synchronization is channel-based; the harness
// never sleeps.
//
// Both node-to-node (NtN) and node-to-client (NtC) modes are supported via
// [Config.Mode].
//
// # Composition with builders
//
// The harness deliberately exposes only standard gouroboros protocol types
// (points, tips, block CBOR) so it composes with the message/conversation
// builders and block builders tracked in this repository. The block helpers in
// this package ([BuildChain], [PointOf], [TipOf]) wrap
// [github.com/blinklabs-io/ouroboros-mock/fixtures.GenerateConwayChain] to
// produce chains whose points and tips line up with what a server callback
// would report. Callers who need richer, multi-era blocks can substitute the
// block builders and continue to use [PointOf]/[TipOf] to derive the matching
// points and tips.
//
// # Consumer connection registries
//
// Some server callbacks do more than reply through their CallbackContext: they
// resolve the peer connection out of a registry the consumer owns — a
// connection manager, peer-governance table, or metrics map — usually to watch
// for peer errors while a read blocks. A callback like that cannot run at all
// until the connection is registered, so register the harness's connection
// with [Harness.ServerConnection] before driving it:
//
//	h, err := csmock.New(csmock.Config{ChainSync: cfg})
//	if err != nil { ... }
//	defer h.Close()
//	myConnManager.Add(h.ServerConnection())
//
// The connection's Id matches CallbackContext.ConnectionId, so a registry keyed
// on that ID resolves. The harness owns the connection's lifecycle and always
// closes it in [Harness.Close]; a consumer registry that also closes its
// connections composes safely in either order. See [Harness.ServerConnection].
package chainsync
