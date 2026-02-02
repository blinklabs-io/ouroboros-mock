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
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
)

// ParsedUtxo contains a UTxO parsed from test vectors.
// It stores the full TransactionOutput for proper validation.
type ParsedUtxo struct {
	TxHash []byte
	Index  uint32
	// Output is the decoded transaction output (e.g., BabbageTransactionOutput)
	Output common.TransactionOutput
}

// ParsedInitialState contains the extracted state from a test vector's initial_state.
type ParsedInitialState struct {
	// CurrentEpoch is the epoch number at the start of the test.
	CurrentEpoch uint64

	// Utxos maps UtxoId (as "txHash#index") to the UTxO output.
	Utxos map[string]ParsedUtxo

	// StakeRegistrations tracks which stake credentials are registered.
	StakeRegistrations map[common.Blake2b224]bool

	// RewardAccounts maps stake credentials to their reward balances.
	RewardAccounts map[common.Blake2b224]uint64

	// PoolRegistrations tracks which pools are registered (by pool key hash).
	PoolRegistrations map[common.Blake2b224]bool

	// CommitteeMembers contains the current constitutional committee (cold key -> expiry).
	CommitteeMembers map[common.Blake2b224]uint64

	// DRepRegistrations contains registered DReps (credential hash).
	DRepRegistrations []common.Blake2b224

	// HotKeyAuthorizations maps cold keys to hot keys for committee members.
	HotKeyAuthorizations map[common.Blake2b224]common.Blake2b224

	// Proposals maps GovActionId (as "txHash#index") to proposal info.
	Proposals map[string]GovActionInfo

	// ProposalRoots tracks the last enacted proposal for each governance purpose.
	ProposalRoots ProposalRoots

	// Constitution contains the current constitution if present.
	Constitution *ConstitutionInfo

	// PParamsHash is the hash of the current protocol parameters.
	PParamsHash []byte

	// CostModels maps Plutus version to cost model array.
	CostModels map[uint][]int64
}

// GovActionInfo contains metadata about a governance proposal.
type GovActionInfo struct {
	ActionType      common.GovActionType
	ExpiresAfter    uint64
	SubmittedEpoch  uint64
	RatifiedEpoch   *uint64
	ParentActionId  *string
	Votes           map[string]uint8                      // "voterType:credHash" -> vote (0=No, 1=Yes, 2=Abstain)
	ProposedMembers map[common.Blake2b224]uint64          // For UpdateCommittee: cold key -> expiry
	ProtocolVersion *ProtocolVersionInfo                  // For HardFork
	PolicyHash      []byte                                // For NewConstitution
	ParameterUpdate *conway.ConwayProtocolParameterUpdate // For ParameterChange
}

// ProtocolVersionInfo contains protocol version for HardFork proposals.
type ProtocolVersionInfo struct {
	Major uint
	Minor uint
}

// ProposalRoots tracks the last enacted proposal for each governance purpose.
type ProposalRoots struct {
	ProtocolParameters      *string // GovActionId for last enacted ParameterChange
	HardFork                *string // GovActionId for last enacted HardFork
	ConstitutionalCommittee *string // GovActionId for last enacted NoConfidence/UpdateCommittee
	Constitution            *string // GovActionId for last enacted NewConstitution
}

// ConstitutionInfo contains constitution details.
type ConstitutionInfo struct {
	AnchorURL  string
	AnchorHash []byte
	PolicyHash []byte
}

// stakeCredential is used for CBOR decoding of credentials.
type stakeCredential struct {
	cbor.StructAsArray
	Type uint64
	Hash common.Blake2b224
}

// ParseInitialState extracts state from a test vector's InitialState field.
func ParseInitialState(raw cbor.RawMessage) (*ParsedInitialState, error) {
	var v cbor.Value
	if _, err := cbor.Decode(raw, &v); err != nil {
		return nil, fmt.Errorf("failed to decode initial_state: %w", err)
	}

	stateArr, ok := v.Value().([]any)
	if !ok || len(stateArr) < 4 {
		return nil, errors.New("unexpected initial_state shape")
	}

	state := &ParsedInitialState{
		Utxos:                make(map[string]ParsedUtxo),
		StakeRegistrations:   make(map[common.Blake2b224]bool),
		RewardAccounts:       make(map[common.Blake2b224]uint64),
		PoolRegistrations:    make(map[common.Blake2b224]bool),
		CommitteeMembers:     make(map[common.Blake2b224]uint64),
		HotKeyAuthorizations: make(map[common.Blake2b224]common.Blake2b224),
		Proposals:            make(map[string]GovActionInfo),
		CostModels:           make(map[uint][]int64),
	}

	// Extract current epoch from stateArr[0]
	if epoch, ok := stateArr[0].(uint64); ok {
		state.CurrentEpoch = epoch
	}

	// Navigate to begin_epoch_state[1] (ledger_state)
	bes, ok := stateArr[3].([]any)
	if !ok || len(bes) < 2 {
		return nil, errors.New("unexpected begin_epoch_state shape")
	}

	ls, ok := bes[1].([]any)
	if !ok || len(ls) < 2 {
		return nil, errors.New("unexpected ledger_state shape")
	}

	// Parse cert_state (ls[0])
	if err := parseCertState(state, ls[0]); err != nil {
		return nil, fmt.Errorf("failed to parse cert_state: %w", err)
	}

	// Parse utxo_state (ls[1]) for governance and cost models
	if err := parseUtxoState(state, ls[1]); err != nil {
		return nil, fmt.Errorf("failed to parse utxo_state: %w", err)
	}

	// Parse UTxOs from raw CBOR using typed decoders (like gouroboros)
	// This is more reliable than the generic parseUtxos approach
	utxos, err := parseUtxosFromRawCBOR(raw)
	if err == nil && len(utxos) > 0 {
		// Fully replace UTxOs to avoid stale entries from generic parsing
		state.Utxos = utxos
	}

	// Parse committee members from raw CBOR using typed decoders
	// This avoids the circular pointer issue with cbor.Value
	// Non-fatal: some vectors don't have committee data
	_ = parseCommitteeFromRawCBOR(state, raw)

	// Extract pparams hash from gov_state (search in ledger_state)
	state.PParamsHash = extractPParamsHash(ls)

	return state, nil
}

// parseCommitteeFromRawCBOR parses committee members from raw CBOR using typed decoders.
// This follows the same approach as gouroboros conformance tests to avoid circular pointer issues.
func parseCommitteeFromRawCBOR(
	state *ParsedInitialState,
	raw cbor.RawMessage,
) error {
	// Decode the top-level array
	var arr []cbor.RawMessage
	if _, err := cbor.Decode(raw, &arr); err != nil {
		return fmt.Errorf("failed to decode initial_state: %w", err)
	}
	if len(arr) < 4 {
		return errors.New("initial_state array too short")
	}

	// arr[3] = begin_epoch_state
	var bes []cbor.RawMessage
	if _, err := cbor.Decode(arr[3], &bes); err != nil {
		return fmt.Errorf("failed to decode begin_epoch_state: %w", err)
	}
	if len(bes) < 2 {
		return errors.New("begin_epoch_state array too short")
	}

	// bes[1] = begin_ledger_state
	var bls []cbor.RawMessage
	if _, err := cbor.Decode(bes[1], &bls); err != nil {
		return fmt.Errorf("failed to decode begin_ledger_state: %w", err)
	}
	if len(bls) < 2 {
		return errors.New("begin_ledger_state array too short")
	}

	// bls[1] = utxo_state
	var utxoState []cbor.RawMessage
	if _, err := cbor.Decode(bls[1], &utxoState); err != nil {
		return fmt.Errorf("failed to decode utxo_state: %w", err)
	}
	if len(utxoState) < 4 {
		return errors.New("utxo_state array too short for gov_state")
	}

	// utxoState[3] = gov_state
	var govState []cbor.RawMessage
	if _, err := cbor.Decode(utxoState[3], &govState); err != nil {
		return fmt.Errorf("failed to decode gov_state: %w", err)
	}
	if len(govState) < 2 {
		return errors.New("gov_state array too short for committee")
	}

	// govState[1] = committee array [[members_map, quorum]]
	var committeeArr []cbor.RawMessage
	if _, err := cbor.Decode(govState[1], &committeeArr); err != nil {
		return fmt.Errorf("failed to decode committee: %w", err)
	}
	if len(committeeArr) < 1 {
		return errors.New("committee array too short")
	}

	// committeeArr[0] = [members_map, quorum]
	var committeeData []cbor.RawMessage
	if _, err := cbor.Decode(committeeArr[0], &committeeData); err != nil {
		return fmt.Errorf("failed to decode committee data: %w", err)
	}
	if len(committeeData) < 1 {
		return errors.New("committee data array too short")
	}

	// committeeData[0] = map of cold credentials -> expiry epoch
	var members map[stakeCredential]uint64
	if _, err := cbor.Decode(committeeData[0], &members); err != nil {
		return fmt.Errorf("failed to decode committee members: %w", err)
	}

	for cred, expiryEpoch := range members {
		state.CommitteeMembers[cred.Hash] = expiryEpoch
	}

	return nil
}

// parseUtxosFromRawCBOR parses UTxOs from raw CBOR using typed decoders.
// This follows the same approach as gouroboros conformance tests for reliable decoding.
func parseUtxosFromRawCBOR(raw cbor.RawMessage) (map[string]ParsedUtxo, error) {
	result := make(map[string]ParsedUtxo)

	// Decode the top-level array
	var arr []cbor.RawMessage
	if _, err := cbor.Decode(raw, &arr); err != nil {
		return result, fmt.Errorf("failed to decode initial_state: %w", err)
	}
	if len(arr) < 4 {
		return result, errors.New("initial_state array too short")
	}

	// arr[3] = begin_epoch_state
	var bes []cbor.RawMessage
	if _, err := cbor.Decode(arr[3], &bes); err != nil {
		return result, fmt.Errorf("failed to decode begin_epoch_state: %w", err)
	}
	if len(bes) < 2 {
		return result, errors.New("begin_epoch_state array too short")
	}

	// bes[1] = begin_ledger_state
	var bls []cbor.RawMessage
	if _, err := cbor.Decode(bes[1], &bls); err != nil {
		return result, fmt.Errorf(
			"failed to decode begin_ledger_state: %w",
			err,
		)
	}
	if len(bls) < 2 {
		return result, errors.New("begin_ledger_state array too short")
	}

	// bls[1] = utxo_state
	var utxoState []cbor.RawMessage
	if _, err := cbor.Decode(bls[1], &utxoState); err != nil {
		return result, fmt.Errorf("failed to decode utxo_state: %w", err)
	}
	if len(utxoState) < 1 {
		return result, errors.New("utxo_state array too short")
	}

	// Try to decode UTxOs from each element in utxoState
	for _, utxoData := range utxoState {
		// Try direct map[UtxoId]BabbageTransactionOutput format
		var utxosMapDirect map[localstatequery.UtxoId]babbage.BabbageTransactionOutput
		if _, err := cbor.Decode(utxoData, &utxosMapDirect); err == nil &&
			len(utxosMapDirect) > 0 {
			for utxoId, output := range utxosMapDirect {
				key := fmt.Sprintf("%x#%d", utxoId.Hash[:], utxoId.Idx)
				outputCopy := output // Copy to avoid pointer issues
				// Copy hash to avoid aliasing the underlying array
				txHashCopy := append([]byte(nil), utxoId.Hash[:]...)
				result[key] = ParsedUtxo{
					TxHash: txHashCopy,
					//nolint:gosec // idx from trusted CBOR test data
					Index:  uint32(utxoId.Idx),
					Output: &outputCopy,
				}
			}
			continue
		}

		// Try array of [UtxoId, Output] pairs
		var utxoPairs [][]cbor.RawMessage
		if _, err := cbor.Decode(utxoData, &utxoPairs); err == nil &&
			len(utxoPairs) > 0 {
			for _, pair := range utxoPairs {
				if len(pair) != 2 {
					continue
				}
				var utxoId localstatequery.UtxoId
				if _, err := cbor.Decode(pair[0], &utxoId); err != nil {
					continue
				}
				var output babbage.BabbageTransactionOutput
				if _, err := cbor.Decode(pair[1], &output); err != nil {
					continue
				}
				key := fmt.Sprintf("%x#%d", utxoId.Hash[:], utxoId.Idx)
				// Copy hash to avoid aliasing the underlying array
				txHashCopy := append([]byte(nil), utxoId.Hash[:]...)
				// Copy output to avoid pointer aliasing across iterations
				outputCopy := output
				result[key] = ParsedUtxo{
					TxHash: txHashCopy,
					//nolint:gosec // idx from trusted CBOR test data
					Index:  uint32(utxoId.Idx),
					Output: &outputCopy,
				}
			}
			continue
		}

		// Try map with string keys (hex-encoded hashes)
		var utxosMapString map[string]babbage.BabbageTransactionOutput
		if _, err := cbor.Decode(utxoData, &utxosMapString); err == nil &&
			len(utxosMapString) > 0 {
			for key, output := range utxosMapString {
				parts := strings.Split(key, "#")
				if len(parts) != 2 {
					continue
				}
				hashBytes, err := hex.DecodeString(parts[0])
				if err != nil {
					continue
				}
				idx, err := strconv.ParseUint(parts[1], 10, 32)
				if err != nil {
					continue
				}
				outputCopy := output
				result[key] = ParsedUtxo{
					TxHash: hashBytes,
					Index:  uint32(idx),
					Output: &outputCopy,
				}
			}
			continue
		}

		// Try using cbor.Value for complex key structures
		var val cbor.Value
		if _, err := cbor.Decode(utxoData, &val); err == nil {
			if m, ok := val.Value().(map[any]any); ok && len(m) > 0 {
				for k, v := range m {
					// Dereference pointer if needed
					var key any
					if ptr, ok := k.(*any); ok && ptr != nil {
						key = *ptr
					} else {
						key = k
					}

					// Extract hash and index from key
					var hash ledger.Blake2b256
					var index uint32
					keyOk := false

					if arr, ok := key.([]any); ok && len(arr) == 2 {
						if h, ok := arr[0].([]byte); ok {
							copy(hash[:], h)
							keyOk = true
						}
						if i, ok := arr[1].(uint64); ok {
							//nolint:gosec // idx from trusted CBOR test data
							index = uint32(i)
						}
					} else {
						// Try encoding key and decoding as UtxoId
						if keyData, err := cbor.Encode(key); err == nil {
							var utxoId localstatequery.UtxoId
							if _, err := cbor.Decode(keyData, &utxoId); err == nil {
								hash = utxoId.Hash
								//nolint:gosec // idx from trusted CBOR test data
								index = uint32(utxoId.Idx)
								keyOk = true
							}
						}
					}

					if !keyOk {
						continue
					}

					// Decode output
					var output babbage.BabbageTransactionOutput
					if outData, err := cbor.Encode(v); err == nil {
						if _, err := cbor.Decode(outData, &output); err == nil {
							utxoKey := fmt.Sprintf("%x#%d", hash[:], index)
							// Copy hash to avoid aliasing the underlying array
							txHashCopy := append([]byte(nil), hash[:]...)
							// Copy output to avoid pointer aliasing across iterations
							outputCopy := output
							result[utxoKey] = ParsedUtxo{
								TxHash: txHashCopy,
								Index:  index,
								Output: &outputCopy,
							}
						}
					}
				}
			}
		}
	}

	return result, nil
}

// parseCertState extracts voting, pool, and delegation state.
func parseCertState(state *ParsedInitialState, certStateRaw any) error {
	certState, ok := certStateRaw.([]any)
	if !ok || len(certState) < 3 {
		return nil // No cert state to parse
	}

	// voting_state = certState[0]
	if err := parseVotingState(state, certState[0]); err != nil {
		return fmt.Errorf("voting_state: %w", err)
	}

	// pool_state = certState[1]
	if err := parsePoolState(state, certState[1]); err != nil {
		return fmt.Errorf("pool_state: %w", err)
	}

	// delegation_state = certState[2]
	if err := parseDelegationState(state, certState[2]); err != nil {
		return fmt.Errorf("delegation_state: %w", err)
	}

	return nil
}

// parseVotingState extracts DReps and hot key authorizations.
//
//nolint:unparam // error return for consistency with other parse functions
func parseVotingState(state *ParsedInitialState, votingStateRaw any) error {
	votingState, ok := votingStateRaw.([]any)
	if !ok || len(votingState) < 2 {
		return nil
	}

	// DReps map at votingState[0]
	if drepsMap, ok := votingState[0].(map[any]any); ok {
		for k := range drepsMap {
			cred := extractCredentialHash(k)
			if cred != nil {
				state.DRepRegistrations = append(
					state.DRepRegistrations,
					cred.Credential,
				)
			}
		}
	}

	// Hot key authorizations at votingState[1]
	if hotKeyMap, ok := votingState[1].(map[any]any); ok {
		for k, v := range hotKeyMap {
			coldKey := extractBlake2b224(k)
			if coldKey == nil {
				continue
			}
			// Value is [hotKeyCred, memberStatus]
			if vArr, ok := v.([]any); ok && len(vArr) >= 1 {
				hotKey := extractCredentialHash(vArr[0])
				if hotKey != nil {
					state.HotKeyAuthorizations[*coldKey] = hotKey.Credential
				}
			}
		}
	}

	return nil
}

// parsePoolState extracts pool registrations.
//
//nolint:unparam // error return for consistency with other parse functions
func parsePoolState(state *ParsedInitialState, poolStateRaw any) error {
	poolState, ok := poolStateRaw.([]any)
	if !ok || len(poolState) < 1 {
		return nil
	}

	// stakePoolParams at poolState[0]
	if poolParams, ok := poolState[0].(map[any]any); ok {
		for k := range poolParams {
			poolId := extractBlake2b224(k)
			if poolId != nil {
				state.PoolRegistrations[*poolId] = true
			}
		}
	}

	return nil
}

// parseDelegationState extracts stake registrations and reward balances.
//
//nolint:unparam // error return for consistency with other parse functions
func parseDelegationState(
	state *ParsedInitialState,
	delegationStateRaw any,
) error {
	delegationState, ok := delegationStateRaw.([]any)
	if !ok || len(delegationState) < 1 {
		return nil
	}

	// unified_map at delegationState[0] contains stake credentials
	if unifiedMap, ok := delegationState[0].(map[any]any); ok {
		for k, v := range unifiedMap {
			cred := extractCredentialHash(k)
			if cred == nil {
				continue
			}
			state.StakeRegistrations[cred.Credential] = true

			// Extract reward balance if present in value
			// Value structure: [rewards_map, deposit, drep_delegatee, pool_delegatee]
			if vArr, ok := v.([]any); ok && len(vArr) >= 1 {
				// rewards_map contains [[epoch, balance], ...]
				if rewardsMap, ok := vArr[0].(map[any]any); ok {
					for _, reward := range rewardsMap {
						if rewardPair, ok := reward.([]any); ok &&
							len(rewardPair) >= 2 {
							if balance, ok := rewardPair[1].(uint64); ok {
								state.RewardAccounts[cred.Credential] = balance
								break
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// parseUtxoState extracts UTxOs, governance state, and cost models.
func parseUtxoState(state *ParsedInitialState, utxoStateRaw any) error {
	utxoState, ok := utxoStateRaw.([]any)
	if !ok || len(utxoState) < 2 {
		return nil
	}

	// UTxOs at utxoState[0]
	if err := parseUtxos(state, utxoState[0]); err != nil {
		return fmt.Errorf("utxos: %w", err)
	}

	// Embedded pparams at utxoState[1] for cost models
	// Non-fatal: some vectors don't have cost models
	_ = parseCostModels(state, utxoState[1])

	// Gov state at utxoState[3] if present
	if len(utxoState) > 3 {
		if err := parseGovState(state, utxoState[3]); err != nil {
			return fmt.Errorf("gov_state: %w", err)
		}
	}

	return nil
}

// parseUtxos extracts UTxOs from various encoding formats.
//
//nolint:unparam // error return for consistency with other parse functions
func parseUtxos(state *ParsedInitialState, utxosRaw any) error {
	switch utxos := utxosRaw.(type) {
	case map[any]any:
		// Direct map encoding
		for k, v := range utxos {
			utxoId := extractUtxoId(k)
			if utxoId == "" {
				continue
			}
			parsedUtxo := extractParsedUtxo(utxoId, v)
			state.Utxos[utxoId] = parsedUtxo
		}
	case []any:
		// Array of [UtxoId, Output] pairs
		for _, pair := range utxos {
			if pairArr, ok := pair.([]any); ok && len(pairArr) >= 2 {
				utxoId := extractUtxoId(pairArr[0])
				if utxoId == "" {
					continue
				}
				parsedUtxo := extractParsedUtxo(utxoId, pairArr[1])
				state.Utxos[utxoId] = parsedUtxo
			}
		}
	}
	return nil
}

// extractUtxoId extracts a UtxoId string from various encodings.
func extractUtxoId(raw any) string {
	switch v := raw.(type) {
	case string:
		return v
	case []byte:
		// Could be raw CBOR or hex-encoded
		if len(v) == 34 {
			// [32-byte txHash, 2-byte index] packed
			return fmt.Sprintf("%x#%d", v[:32], binary.BigEndian.Uint16(v[32:]))
		}
		return hex.EncodeToString(v)
	case []any:
		// [txHash, index] array
		if len(v) >= 2 {
			var txHash string
			switch h := v[0].(type) {
			case []byte:
				txHash = hex.EncodeToString(h)
			case cbor.ByteString:
				txHash = hex.EncodeToString(h.Bytes())
			case string:
				txHash = h
			}
			var idx uint64
			switch i := v[1].(type) {
			case uint64:
				idx = i
			case int64:
				//nolint:gosec // idx from trusted CBOR data
				idx = uint64(i)
			}
			if txHash != "" {
				return fmt.Sprintf("%s#%d", txHash, idx)
			}
		}
	}
	return ""
}

// extractParsedUtxo extracts a ParsedUtxo from raw CBOR value.
func extractParsedUtxo(utxoId string, raw any) ParsedUtxo {
	utxo := ParsedUtxo{}

	// Parse utxoId to get txHash and index
	parts := strings.Split(utxoId, "#")
	if len(parts) == 2 {
		utxo.TxHash, _ = hex.DecodeString(parts[0])
		if idx, err := strconv.ParseUint(parts[1], 10, 32); err == nil {
			utxo.Index = uint32(idx) //nolint:gosec // checked by ParseUint
		}
	}

	// Try to decode as a BabbageTransactionOutput from raw CBOR
	// The raw value may be a cbor.Value that we need to re-encode first
	var outputCbor []byte
	switch v := raw.(type) {
	case cbor.RawMessage:
		outputCbor = v
	case cbor.Value:
		// Re-encode the Value to get raw CBOR for proper decoding
		var err error
		outputCbor, err = cbor.Encode(v.Value())
		if err != nil {
			return utxo
		}
	default:
		// Try to encode whatever we got
		var err error
		outputCbor, err = cbor.Encode(v)
		if err != nil {
			return utxo
		}
	}

	// Try to decode as BabbageTransactionOutput (supports both map and array formats)
	var babbageOutput babbage.BabbageTransactionOutput
	if _, err := cbor.Decode(outputCbor, &babbageOutput); err == nil {
		utxo.Output = babbageOutput
		return utxo
	}

	return utxo
}

// parseCostModels extracts cost models from embedded pparams.
//
//nolint:unparam // error return for consistency with other parse functions
func parseCostModels(state *ParsedInitialState, pparamsRaw any) error {
	// pparams is an array of 4 maps: [current, prev, future, proposed]
	pparamsArr, ok := pparamsRaw.([]any)
	if !ok || len(pparamsArr) < 1 {
		return nil
	}

	// Current pparams at index 0
	currentPParams, ok := pparamsArr[0].(map[any]any)
	if !ok {
		return nil
	}

	// Cost models are at key 15 (Conway) or similar
	costModelsKey := uint64(15)
	costModelsRaw, ok := currentPParams[costModelsKey]
	if !ok {
		return nil
	}

	costModelsMap, ok := costModelsRaw.(map[any]any)
	if !ok {
		return nil
	}

	for k, v := range costModelsMap {
		version, ok := k.(uint64)
		if !ok {
			continue
		}
		if arr, ok := v.([]any); ok {
			costs := make([]int64, len(arr))
			for i, c := range arr {
				switch cv := c.(type) {
				case uint64:
					//nolint:gosec // cost model values from trusted CBOR
					costs[i] = int64(cv)
				case int64:
					costs[i] = cv
				}
			}
			state.CostModels[uint(version)] = costs
		}
	}

	return nil
}

// parseGovState extracts governance state (proposals, committee, constitution).
func parseGovState(state *ParsedInitialState, govStateRaw any) error {
	govState, ok := govStateRaw.([]any)
	if !ok || len(govState) < 3 {
		return nil
	}

	// Parse proposals from govState[0]
	if err := parseProposals(state, govState[0]); err != nil {
		return fmt.Errorf("proposals: %w", err)
	}

	// Parse committee from govState[1]
	if err := parseCommittee(state, govState[1]); err != nil {
		return fmt.Errorf("committee: %w", err)
	}

	// Parse constitution from govState[2]
	if err := parseConstitution(state, govState[2]); err != nil {
		return fmt.Errorf("constitution: %w", err)
	}

	return nil
}

// parseProposals extracts governance proposals.
//
//nolint:unparam // error return for consistency with other parse functions
func parseProposals(state *ParsedInitialState, proposalsRaw any) error {
	// Proposals can be in different structures:
	// Option A: [[proposals_tree, root_params, root_hf, root_constitution], root_cc]
	// Option B: [proposals_tree, root_params, root_hf, root_cc, root_constitution]

	proposalsArr, ok := proposalsRaw.([]any)
	if !ok {
		return nil
	}

	var proposalsTree any
	var rootParams, rootHF, rootCC, rootConstitution any

	if len(proposalsArr) >= 2 {
		if inner, ok := proposalsArr[0].([]any); ok && len(inner) >= 4 {
			// Option A
			proposalsTree = inner[0]
			rootParams = inner[1]
			rootHF = inner[2]
			rootConstitution = inner[3]
			rootCC = proposalsArr[1]
		} else if len(proposalsArr) >= 5 {
			// Option B
			proposalsTree = proposalsArr[0]
			rootParams = proposalsArr[1]
			rootHF = proposalsArr[2]
			rootCC = proposalsArr[3]
			rootConstitution = proposalsArr[4]
		} else {
			proposalsTree = proposalsArr[0]
		}
	}

	// Extract proposal roots
	state.ProposalRoots.ProtocolParameters = extractEnactedRoot(rootParams)
	state.ProposalRoots.HardFork = extractEnactedRoot(rootHF)
	state.ProposalRoots.ConstitutionalCommittee = extractEnactedRoot(rootCC)
	state.ProposalRoots.Constitution = extractEnactedRoot(rootConstitution)

	// Parse proposals from tree
	if proposalsMap, ok := proposalsTree.(map[any]any); ok {
		for k, v := range proposalsMap {
			govActionId := extractGovActionId(k)
			if govActionId == "" {
				continue
			}
			info := extractProposalInfo(v)
			state.Proposals[govActionId] = info
		}
	}

	return nil
}

// extractEnactedRoot extracts a GovActionId from a root structure.
func extractEnactedRoot(raw any) *string {
	if raw == nil {
		return nil
	}
	// Root can be [txHash, idx] or wrapped in additional structure
	if arr, ok := raw.([]any); ok && len(arr) >= 2 {
		govActionId := extractGovActionId(arr)
		if govActionId != "" {
			return &govActionId
		}
	}
	return nil
}

// extractGovActionId extracts a GovActionId string from [txHash, idx].
func extractGovActionId(raw any) string {
	switch v := raw.(type) {
	case []any:
		if len(v) >= 2 {
			var txHash string
			switch h := v[0].(type) {
			case []byte:
				txHash = hex.EncodeToString(h)
			case cbor.ByteString:
				txHash = hex.EncodeToString(h.Bytes())
			}
			var idx uint64
			switch i := v[1].(type) {
			case uint64:
				idx = i
			case int64:
				//nolint:gosec // idx from trusted CBOR data
				idx = uint64(i)
			}
			if txHash != "" {
				return fmt.Sprintf("%s#%d", txHash, idx)
			}
		}
	case string:
		return v
	}
	return ""
}

// extractProposalInfo extracts GovActionInfo from a proposal entry.
func extractProposalInfo(raw any) GovActionInfo {
	info := GovActionInfo{
		Votes:           make(map[string]uint8),
		ProposedMembers: make(map[common.Blake2b224]uint64),
	}

	// Proposal structure: [id, cc_votes, drep_votes, pool_votes, procedure, proposed_in, expires_after]
	arr, ok := raw.([]any)
	if !ok || len(arr) < 7 {
		return info
	}

	// Extract votes from cc_votes (arr[1]), drep_votes (arr[2]), pool_votes (arr[3])
	extractVotes(info.Votes, arr[1], 0) // CC votes
	extractVotes(info.Votes, arr[2], 2) // DRep votes
	extractVotes(info.Votes, arr[3], 4) // Pool/SPO votes

	// procedure at arr[4]
	if procedure, ok := arr[4].([]any); ok {
		info.ActionType, info.ProposedMembers = extractActionTypeAndMembers(
			procedure,
		)
	}

	// proposed_in at arr[5]
	if epoch, ok := arr[5].(uint64); ok {
		info.SubmittedEpoch = epoch
	}

	// expires_after at arr[6]
	if epoch, ok := arr[6].(uint64); ok {
		info.ExpiresAfter = epoch
	}

	return info
}

// extractVotes extracts votes from a vote map.
// voterTypeBase is the base voter type for this category:
//   - 0 for CC votes (0=hot key hash, 1=hot script hash)
//   - 2 for DRep votes (2=key hash, 3=script hash)
//   - 4 for SPO votes (4=pool key hash, no script variant)
//
// For CC and DRep, the credential type (0=key, 1=script) is added to the base.
// For SPO, the type is always 4 (pools use key hashes only).
func extractVotes(votes map[string]uint8, votesRaw any, voterTypeBase uint8) {
	votesMap, ok := votesRaw.(map[any]any)
	if !ok {
		return
	}

	for k, v := range votesMap {
		var credHash string
		var credType uint8 // 0=key hash, 1=script hash

		switch key := k.(type) {
		case []byte:
			// Raw bytes - assume key hash (type 0)
			credHash = hex.EncodeToString(key)
			credType = 0
		case cbor.ByteString:
			// Raw bytes - assume key hash (type 0)
			credHash = hex.EncodeToString(key.Bytes())
			credType = 0
		case []any:
			// [type, hash] credential - extract actual credential type
			if cred := extractCredentialHash(key); cred != nil {
				credHash = hex.EncodeToString(cred.Credential.Bytes())
				//nolint:gosec // CredType is 0 or 1
				credType = uint8(cred.CredType)
			}
		}
		if credHash == "" {
			continue
		}

		// Vote value is typically just a uint (0=No, 1=Yes, 2=Abstain)
		var vote uint8
		var voteOk bool
		switch vv := v.(type) {
		case uint64:
			//nolint:gosec // vote values are 0-2
			vote = uint8(vv)
			voteOk = true
		case []any:
			// Could be wrapped in array
			if len(vv) > 0 {
				if voteVal, ok := vv[0].(uint64); ok {
					//nolint:gosec // vote values are 0-2
					vote = uint8(voteVal)
					voteOk = true
				}
			}
		}

		// Only record the vote if we successfully parsed it
		if !voteOk {
			continue
		}

		// Compute actual voter type:
		// - CC: base 0 + credType (0 or 1) = 0 or 1
		// - DRep: base 2 + credType (0 or 1) = 2 or 3
		// - SPO: base 4, credType ignored (always key hash)
		voterType := voterTypeBase
		if voterTypeBase < 4 {
			// CC and DRep support both key and script credentials
			voterType = voterTypeBase + credType
		}

		voterKey := fmt.Sprintf("%d:%s", voterType, credHash)
		votes[voterKey] = vote
	}
}

// extractActionTypeAndMembers extracts action type and proposed members from procedure.
func extractActionTypeAndMembers(
	procedure []any,
) (common.GovActionType, map[common.Blake2b224]uint64) {
	members := make(map[common.Blake2b224]uint64)

	// Find action array in procedure (typically at indices 2-4)
	var actionArr []any
	for i := 2; i < len(procedure) && i <= 4; i++ {
		if arr, ok := procedure[i].([]any); ok && len(arr) >= 1 {
			if _, ok := arr[0].(uint64); ok {
				actionArr = arr
				break
			}
		}
	}

	if len(actionArr) < 1 {
		return 0, members
	}

	actionType, _ := actionArr[0].(uint64)

	// For UpdateCommittee (type 4), extract proposed members from actionArr[3]
	if actionType == 4 && len(actionArr) > 3 {
		if membersMap, ok := actionArr[3].(map[any]any); ok {
			for k, v := range membersMap {
				cred := extractCredentialHash(k)
				if cred == nil {
					continue
				}
				if expiry, ok := v.(uint64); ok {
					members[cred.Credential] = expiry
				}
			}
		}
	}

	return common.GovActionType(actionType), members
}

// parseCommittee extracts committee members.
// NOTE: This function is kept for backwards compatibility but the actual committee parsing
// now happens in parseCommitteeFromRawCBOR which uses typed decoders to avoid circular pointer issues.
//
//nolint:unparam // error return for consistency with other parse functions
func parseCommittee(state *ParsedInitialState, committeeRaw any) error {
	// Committee parsing is now handled by parseCommitteeFromRawCBOR which uses
	// typed decoders from raw CBOR to avoid the circular pointer issue.
	// This function is kept for any cases where the raw CBOR parser fails.
	committeeArr, ok := committeeRaw.([]any)
	if !ok || len(committeeArr) < 1 {
		return nil
	}

	// Try to find the members map, handling different nesting levels
	var membersMap map[any]any

	// Check if committeeArr[0] is the members map directly
	if m, ok := committeeArr[0].(map[any]any); ok {
		membersMap = m
	} else if inner, ok := committeeArr[0].([]any); ok && len(inner) > 0 {
		// Nested array format: [[{members_map}, threshold]]
		if m, ok := inner[0].(map[any]any); ok {
			membersMap = m
		}
	}

	if membersMap == nil {
		return nil
	}

	for k, v := range membersMap {
		// Handle pointer wrapper that Go's cbor decoder uses for non-hashable keys
		key := unwrapPointer(k)

		cred := extractCredentialHash(key)
		if cred == nil {
			continue
		}

		var expiry uint64
		if exp, ok := v.(uint64); ok {
			expiry = exp
		}

		state.CommitteeMembers[cred.Credential] = expiry
	}

	return nil
}

// parseConstitution extracts constitution details.
//
//nolint:unparam // error return for consistency with other parse functions
func parseConstitution(state *ParsedInitialState, constitutionRaw any) error {
	// Constitution structure: [anchor, policy_hash] or [[url, hash], policy_hash]
	constitutionArr, ok := constitutionRaw.([]any)
	if !ok || len(constitutionArr) < 1 {
		return nil
	}

	state.Constitution = &ConstitutionInfo{}

	// Anchor at index 0
	if anchor, ok := constitutionArr[0].([]any); ok && len(anchor) >= 2 {
		if url, ok := anchor[0].(string); ok {
			state.Constitution.AnchorURL = url
		}
		switch h := anchor[1].(type) {
		case []byte:
			state.Constitution.AnchorHash = h
		case cbor.ByteString:
			state.Constitution.AnchorHash = h.Bytes()
		}
	}

	// Policy hash at index 1 (optional)
	if len(constitutionArr) > 1 {
		switch h := constitutionArr[1].(type) {
		case []byte:
			state.Constitution.PolicyHash = h
		case cbor.ByteString:
			state.Constitution.PolicyHash = h.Bytes()
		}
	}

	return nil
}

// extractPParamsHash searches ledger_state for protocol parameters hash.
func extractPParamsHash(ls []any) []byte {
	// Search through ledger state items for pparams hash
	// It's typically in gov_state[3] (the current pparams hash)
	for _, item := range ls {
		sub, ok := item.([]any)
		if !ok || len(sub) <= 3 {
			continue
		}

		// Check direct byte value at index 3
		switch v := sub[3].(type) {
		case []byte:
			if len(v) > 0 {
				return v
			}
		case cbor.ByteString:
			b := v.Bytes()
			if len(b) > 0 {
				return b
			}
		case []any:
			// Nested array: check sub[3][3]
			if len(v) > 3 {
				switch hashv := v[3].(type) {
				case []byte:
					if len(hashv) > 0 {
						return hashv
					}
				case cbor.ByteString:
					b := hashv.Bytes()
					if len(b) > 0 {
						return b
					}
				}
			}
		}
	}
	return nil
}

// unwrapPointer attempts to extract a usable value from CBOR's pointer wrappers.
// For non-hashable map keys, CBOR wraps the value in &keyValue (see cbor/value.go).
// We simply dereference the pointer to get the actual value.
func unwrapPointer(v any) any {
	// If it's a pointer to any, dereference once
	if ptr, ok := v.(*any); ok && ptr != nil {
		return *ptr
	}
	return v
}

// extractCredentialHash extracts a Credential from various encodings.
func extractCredentialHash(raw any) *common.Credential {
	switch v := raw.(type) {
	case []any:
		// [type, hash] format
		if len(v) >= 2 {
			cred := &common.Credential{}
			if credType, ok := v[0].(uint64); ok {
				cred.CredType = uint(credType)
			}
			switch h := v[1].(type) {
			case []byte:
				if len(h) == 28 {
					copy(cred.Credential[:], h)
				} else if len(h) == 56 {
					// Handle hex-encoded bytes
					if decoded, err := hex.DecodeString(string(h)); err == nil && len(decoded) == 28 {
						copy(cred.Credential[:], decoded)
					}
				}
			case cbor.ByteString:
				b := h.Bytes()
				if len(b) == 28 {
					copy(cred.Credential[:], b)
				} else if len(b) == 56 {
					// Handle hex-encoded bytes
					if decoded, err := hex.DecodeString(string(b)); err == nil && len(decoded) == 28 {
						copy(cred.Credential[:], decoded)
					}
				}
			case string:
				// Handle hex-encoded string
				if len(h) == 56 {
					if decoded, err := hex.DecodeString(h); err == nil && len(decoded) == 28 {
						copy(cred.Credential[:], decoded)
					}
				}
			}
			// Only return the credential if we successfully extracted a valid hash.
			// A zeroed credential indicates extraction failed.
			var zeroHash common.Blake2b224
			if cred.Credential == zeroHash {
				return nil
			}
			return cred
		}
	case stakeCredential:
		cred := &common.Credential{}
		cred.CredType = uint(v.Type)
		cred.Credential = v.Hash
		return cred
	case string:
		// Handle hex-encoded credential hash (56 chars = 28 bytes)
		if len(v) == 56 {
			if decoded, err := hex.DecodeString(v); err == nil && len(decoded) == 28 {
				cred := &common.Credential{}
				copy(cred.Credential[:], decoded)
				return cred
			}
		}
	}
	return nil
}

// extractBlake2b224 extracts a Blake2b224 hash from various encodings.
func extractBlake2b224(raw any) *common.Blake2b224 {
	var hash common.Blake2b224
	switch v := raw.(type) {
	case []byte:
		if len(v) == 28 {
			copy(hash[:], v[:28])
			return &hash
		}
		// Handle hex-encoded bytes (56 chars = 28 bytes when decoded)
		if len(v) == 56 {
			if decoded, err := hex.DecodeString(string(v)); err == nil && len(decoded) == 28 {
				copy(hash[:], decoded)
				return &hash
			}
		}
	case cbor.ByteString:
		b := v.Bytes()
		if len(b) == 28 {
			copy(hash[:], b[:28])
			return &hash
		}
		// Handle hex-encoded bytes
		if len(b) == 56 {
			if decoded, err := hex.DecodeString(string(b)); err == nil && len(decoded) == 28 {
				copy(hash[:], decoded)
				return &hash
			}
		}
	case string:
		// Handle hex-encoded string
		if len(v) == 56 {
			if decoded, err := hex.DecodeString(v); err == nil && len(decoded) == 28 {
				copy(hash[:], decoded)
				return &hash
			}
		}
	}
	return nil
}
