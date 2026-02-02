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
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
)

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
	conway.UtxoValidateCommitteeCertificates,
	conway.UtxoValidateMalformedReferenceScripts,
}
