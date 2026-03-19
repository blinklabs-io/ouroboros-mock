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

package fixtures

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	gcommon "github.com/blinklabs-io/gouroboros/ledger/common"
)

type drepMetadata struct {
	Context       any              `json:"@context"`
	HashAlgorithm string           `json:"hashAlgorithm"`
	Body          drepMetadataBody `json:"body"`
}

type drepMetadataBody struct {
	PaymentAddress string                `json:"paymentAddress"`
	GivenName      string                `json:"givenName"`
	Image          metadataImage         `json:"image"`
	Objectives     string                `json:"objectives"`
	Motivations    string                `json:"motivations"`
	Qualifications string                `json:"qualifications"`
	References     []governanceReference `json:"references"`
}

type noConfidenceMetadata struct {
	Context       any                         `json:"@context"`
	HashAlgorithm string                      `json:"hashAlgorithm"`
	Body          noConfidenceMetadataBody    `json:"body"`
	Authors       []metadataAuthorWithWitness `json:"authors"`
}

type noConfidenceMetadataBody struct {
	Title      string                `json:"title"`
	Abstract   string                `json:"abstract"`
	Motivation string                `json:"motivation"`
	Rationale  string                `json:"rationale"`
	References []governanceReference `json:"references"`
}

type governanceReference struct {
	Type          string                 `json:"@type"`
	Label         string                 `json:"label"`
	URI           string                 `json:"uri"`
	ReferenceHash *metadataReferenceHash `json:"referenceHash"`
}

type metadataReferenceHash struct {
	HashDigest    string `json:"hashDigest"`
	HashAlgorithm string `json:"hashAlgorithm"`
}

type metadataImage struct {
	Type       string `json:"@type"`
	ContentURL string `json:"contentUrl"`
	SHA256     string `json:"sha256"`
}

type metadataAuthorWithWitness struct {
	Witness metadataWitness `json:"witness"`
}

type metadataWitness struct {
	WitnessAlgorithm string `json:"witnessAlgorithm"`
	PublicKey        string `json:"publicKey"`
	Signature        string `json:"signature"`
}

func validateGovernanceMetadataFixture(f Fixture) error {
	data, err := f.Read()
	if err != nil {
		return err
	}

	switch f.Name {
	case "valid-drep-metadata.jsonld":
		return validateDRepMetadata(data)
	case "invalid-drep-metadata.jsonld":
		if err := validateDRepMetadata(data); err == nil {
			return fmt.Errorf("expected invalid governance metadata for %s", f.RelPath)
		}
		return nil
	case "no-confidence.jsonld":
		return validateNoConfidenceMetadata(data)
	default:
		return fmt.Errorf("unsupported governance metadata fixture: %s", f.RelPath)
	}
}

func validateDRepMetadata(data []byte) error {
	var metadata drepMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}

	if metadata.Context == nil {
		return errors.New("missing @context")
	}
	if metadata.HashAlgorithm == "" {
		return errors.New("missing hashAlgorithm")
	}
	if metadata.Body.GivenName == "" {
		return errors.New("missing body.givenName")
	}
	if metadata.Body.PaymentAddress == "" {
		return errors.New("missing body.paymentAddress")
	}
	if _, err := gcommon.NewAddress(metadata.Body.PaymentAddress); err != nil {
		return fmt.Errorf("invalid paymentAddress: %w", err)
	}
	if metadata.Body.Objectives == "" || metadata.Body.Motivations == "" || metadata.Body.Qualifications == "" {
		return errors.New("incomplete DRep body text fields")
	}
	if err := validateImage(metadata.Body.Image); err != nil {
		return err
	}
	if len(metadata.Body.References) == 0 {
		return errors.New("missing references")
	}
	for _, reference := range metadata.Body.References {
		if err := validateReference(reference, false); err != nil {
			return err
		}
	}
	return nil
}

func validateNoConfidenceMetadata(data []byte) error {
	var metadata noConfidenceMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}

	if metadata.Context == nil {
		return errors.New("missing @context")
	}
	if metadata.HashAlgorithm == "" {
		return errors.New("missing hashAlgorithm")
	}
	if metadata.Body.Title == "" || metadata.Body.Abstract == "" || metadata.Body.Motivation == "" || metadata.Body.Rationale == "" {
		return errors.New("incomplete governance action body")
	}
	for _, reference := range metadata.Body.References {
		if err := validateReference(reference, true); err != nil {
			return err
		}
	}
	if len(metadata.Authors) == 0 {
		return errors.New("missing authors")
	}
	for _, author := range metadata.Authors {
		if author.Witness.WitnessAlgorithm == "" {
			return errors.New("missing author witness algorithm")
		}
		if err := validateHexString(author.Witness.PublicKey, 32, "author public key"); err != nil {
			return err
		}
		if author.Witness.Signature == "" {
			return errors.New("missing author signature")
		}
		if err := validateHexString(author.Witness.Signature, -1, "author signature"); err != nil {
			return err
		}
	}
	return nil
}

func validateImage(image metadataImage) error {
	if image.Type != "ImageObject" {
		return fmt.Errorf("unexpected image @type %q", image.Type)
	}
	if err := validateURI(image.ContentURL, "image.contentUrl"); err != nil {
		return err
	}
	return validateHexString(image.SHA256, 32, "image.sha256")
}

func validateReference(reference governanceReference, expectHash bool) error {
	if reference.Type == "" {
		return errors.New("missing reference @type")
	}
	if reference.Label == "" {
		return errors.New("missing reference label")
	}
	if err := validateURI(reference.URI, "reference.uri"); err != nil {
		return err
	}
	if expectHash {
		if reference.ReferenceHash == nil {
			return errors.New("missing referenceHash")
		}
		if reference.ReferenceHash.HashAlgorithm == "" {
			return errors.New("missing referenceHash.hashAlgorithm")
		}
		if err := validateHexString(reference.ReferenceHash.HashDigest, 32, "referenceHash.hashDigest"); err != nil {
			return err
		}
	}
	return nil
}

func validateURI(raw string, label string) error {
	if raw == "" {
		return fmt.Errorf("missing %s", label)
	}
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return fmt.Errorf("invalid %s: %w", label, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("invalid %s: expected absolute URI", label)
	}
	return nil
}

func validateHexString(raw string, expectedBytes int, label string) error {
	decoded, err := hex.DecodeString(raw)
	if err != nil {
		return fmt.Errorf("invalid %s: %w", label, err)
	}
	if expectedBytes >= 0 && len(decoded) != expectedBytes {
		return fmt.Errorf("unexpected %s length: got %d want %d", label, len(decoded), expectedBytes)
	}
	return nil
}
