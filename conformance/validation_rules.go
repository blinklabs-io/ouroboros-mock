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
	"strings"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
)

// ValidationRulesForVector selects rules applicable to the era named in a
// Blueprint record. The protocol-parameter loader intentionally returns the
// Conway representation, so pre-Conway records use the common Conway rule
// implementations with Conway-only rules omitted.
func ValidationRulesForVector(title string) []common.UtxoValidationRuleFunc {
	if strings.Contains(title, "AlonzoImpSpec") {
		return AlonzoConformanceValidationRules
	}
	return ConformanceValidationRules
}

var AlonzoConformanceValidationRules = []common.UtxoValidationRuleFunc{
	conway.UtxoValidateMetadata,
	conway.UtxoValidateIsValidFlag,
	conway.UtxoValidateRequiredVKeyWitnesses,
	conway.UtxoValidateCollateralVKeyWitnesses,
	conway.UtxoValidateRedeemerAndScriptWitnesses,
	conway.UtxoValidateSignatures,
	conway.UtxoValidateCostModelsPresent,
	conway.UtxoValidateScriptDataHash,
	conway.UtxoValidateOutsideValidityIntervalUtxo,
	conway.UtxoValidateConwayFeaturesWithPlutusV1V2,
	conway.UtxoValidateInputSetEmptyUtxo,
	conway.UtxoValidateInsufficientCollateral,
	conway.UtxoValidateCollateralContainsNonAda,
	conway.UtxoValidateCollateralEqBalance,
	conway.UtxoValidateNoCollateralInputs,
	conway.UtxoValidateBadInputsUtxo,
	conway.UtxoValidateScriptWitnesses,
	conway.UtxoValidateValueNotConservedUtxo,
	conway.UtxoValidateOutputTooSmallUtxo,
	conway.UtxoValidateOutputTooBigUtxo,
	conway.UtxoValidateWrongNetwork,
	conway.UtxoValidateTransactionNetworkId,
	conway.UtxoValidateExUnitsTooBigUtxo,
	conway.UtxoValidateExtraneousRedeemers,
	conway.UtxoValidateSupplementalDatums,
	conway.UtxoValidatePlutusScripts,
	conway.UtxoValidateNativeScripts,
	conway.UtxoValidateDelegation,
	conway.UtxoValidateWithdrawals,
}

// ConformanceValidationRules is a custom validation rule set for conformance tests.
// It excludes fee validation (UtxoValidateFeeTooSmallUtxo) because test vectors
// from Haskell have pre-computed fees that may differ due to CBOR encoding differences.
// It also excludes max transaction size validation for the same reason.
var ConformanceValidationRules = []common.UtxoValidationRuleFunc{
	conway.UtxoValidateMetadata,
	conway.UtxoValidateProposalProcedures,
	conway.UtxoValidateProposalNetworkIds,
	conway.UtxoValidateEmptyTreasuryWithdrawals,
	conway.UtxoValidateIsValidFlag,
	conway.UtxoValidateRequiredVKeyWitnesses,
	conway.UtxoValidateCollateralVKeyWitnesses,
	conway.UtxoValidateRedeemerAndScriptWitnesses,
	conway.UtxoValidateSignatures,
	conway.UtxoValidateCostModelsPresent,
	conway.UtxoValidateScriptDataHash,
	conway.UtxoValidateInlineDatumsWithPlutusV1,
	conway.UtxoValidateConwayFeaturesWithPlutusV1V2,
	conway.UtxoValidateDisjointRefInputs,
	conway.UtxoValidateOutsideValidityIntervalUtxo,
	conway.UtxoValidateInputSetEmptyUtxo,
	// UtxoValidateFeeTooSmallUtxo is EXCLUDED - test vectors have pre-computed fees
	conway.UtxoValidateInsufficientCollateral,
	conway.UtxoValidateCollateralContainsNonAda,
	conway.UtxoValidateCollateralEqBalance,
	conway.UtxoValidateNoCollateralInputs,
	conway.UtxoValidateBadInputsUtxo,
	conway.UtxoValidateScriptWitnesses,
	conway.UtxoValidateValueNotConservedUtxo,
	conway.UtxoValidateOutputTooSmallUtxo,
	conway.UtxoValidateOutputTooBigUtxo,
	conway.UtxoValidateOutputBootAddrAttrsTooBig,
	conway.UtxoValidateWrongNetwork,
	conway.UtxoValidateWrongNetworkWithdrawal,
	conway.UtxoValidateTransactionNetworkId,
	// UtxoValidateMaxTxSizeUtxo is EXCLUDED - test vectors have pre-computed sizes
	conway.UtxoValidateExUnitsTooBigUtxo,
	conway.UtxoValidateTooManyCollateralInputs,
	conway.UtxoValidateSupplementalDatums,
	conway.UtxoValidateExtraneousRedeemers,
	conway.UtxoValidatePlutusScripts,
	conway.UtxoValidateNativeScripts,
	conway.UtxoValidateDelegation,
	conway.UtxoValidateWithdrawals,
	// The exact credential-aware Validator phase runs before this core rule.
	// Retain both so conformance exercises the upstream ledger rule while the
	// local phase preserves credential identity and sequential certificate state.
	utxoValidateCommitteeCertificates,
	conway.UtxoValidateMalformedReferenceScripts,
}

type committeeCredentialMemberState interface {
	CommitteeStateAvailable() (bool, error)
	CommitteeCredentialMember(
		common.Credential,
	) (*common.CommitteeMember, error)
	CommitteeHotCredentialMember(
		common.Credential,
	) (*common.CommitteeMember, error)
}

type committeeCertificateTransaction struct {
	common.Transaction
	certificate common.Certificate
}

func (t committeeCertificateTransaction) Certificates() []common.Certificate {
	return []common.Certificate{t.certificate}
}

type committeeCertificateLedgerState struct {
	common.LedgerState
	credential common.Credential
	provider   committeeCredentialMemberState
}

func (s committeeCertificateLedgerState) CommitteeMember(
	coldKey common.Blake2b224,
) (*common.CommitteeMember, error) {
	if coldKey == s.credential.Credential {
		return s.provider.CommitteeCredentialMember(s.credential)
	}
	return s.LedgerState.CommitteeMember(coldKey)
}

func (s committeeCertificateLedgerState) CommitteeStateAvailable() (bool, error) {
	return s.provider.CommitteeStateAvailable()
}

func (s committeeCertificateLedgerState) CommitteeCredentialMember(
	coldCredential common.Credential,
) (*common.CommitteeMember, error) {
	return s.provider.CommitteeCredentialMember(coldCredential)
}

func (s committeeCertificateLedgerState) CommitteeHotCredentialMember(
	hotCredential common.Credential,
) (*common.CommitteeMember, error) {
	return s.provider.CommitteeHotCredentialMember(hotCredential)
}

// utxoValidateCommitteeCertificates retains the standard Conway rule while
// projecting each certificate's full cold credential through its hash-only
// state query. Running one committee certificate at a time prevents key and
// script credentials with the same hash from aliasing within a transaction.
func utxoValidateCommitteeCertificates(
	tx common.Transaction,
	slot uint64,
	state common.LedgerState,
	params common.ProtocolParameters,
) error {
	credentialState, ok := state.(committeeCredentialMemberState)
	if !ok {
		return conway.UtxoValidateCommitteeCertificates(tx, slot, state, params)
	}

	for _, certificate := range tx.Certificates() {
		var coldCredential common.Credential
		switch typedCertificate := certificate.(type) {
		case *common.AuthCommitteeHotCertificate:
			coldCredential = typedCertificate.ColdCredential
		case *common.ResignCommitteeColdCertificate:
			coldCredential = typedCertificate.ColdCredential
		default:
			continue
		}
		transactionView := committeeCertificateTransaction{
			Transaction: tx,
			certificate: certificate,
		}
		stateView := committeeCertificateLedgerState{
			LedgerState: state,
			credential:  coldCredential,
			provider:    credentialState,
		}
		if err := conway.UtxoValidateCommitteeCertificates(
			transactionView,
			slot,
			stateView,
			params,
		); err != nil {
			return err
		}
	}
	return nil
}
