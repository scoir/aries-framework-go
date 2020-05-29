/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package verifiable

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/aries-framework-go/pkg/doc/signature/suite"
	"github.com/hyperledger/aries-framework-go/pkg/doc/signature/suite/ed25519signature2018"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
)

func TestNewPresentationFromLinkedDataProof(t *testing.T) {
	r := require.New(t)

	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	r.NoError(err)

	ss := ed25519signature2018.New(suite.WithSigner(getEd25519TestSigner(privKey)))

	ldpContext := &LinkedDataProofContext{
		SignatureType:           "Ed25519Signature2018",
		SignatureRepresentation: SignatureJWS,
		Suite:                   ss,
	}

	vc, err := NewPresentation([]byte(validPresentation))
	r.NoError(err)

	err = vc.AddLinkedDataProof(ldpContext)
	r.NoError(err)

	vcBytes, err := json.Marshal(vc)
	r.NoError(err)

	vcWithLdp, err := NewPresentation(vcBytes,
		WithPresEmbeddedSignatureSuites(ss),
		WithPresPublicKeyFetcher(SingleKey(pubKey, kms.ED25519)))
	r.NoError(err)

	r.NoError(err)
	r.Equal(vc, vcWithLdp)
}

func TestPresentation_AddLinkedDataProof(t *testing.T) {
	r := require.New(t)

	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	r.NoError(err)

	ldpContext := &LinkedDataProofContext{
		SignatureType:           "Ed25519Signature2018",
		SignatureRepresentation: SignatureProofValue,
		Suite:                   ed25519signature2018.New(suite.WithSigner(getEd25519TestSigner(privKey))),
	}

	t.Run("Add a valid Linked Data proof to VC", func(t *testing.T) {
		vp, err := NewPresentation([]byte(validPresentation))
		r.NoError(err)

		err = vp.AddLinkedDataProof(ldpContext)
		r.NoError(err)

		vpJSON, err := vp.MarshalJSON()
		r.NoError(err)

		vpMap, err := toMap(vpJSON)
		r.NoError(err)

		r.Contains(vpMap, "proof")
		vpProof := vpMap["proof"]
		vpProofs, ok := vpProof.([]interface{})
		r.True(ok)
		r.Len(vpProofs, 2)
		newVPProof, ok := vpProofs[1].(map[string]interface{})
		r.True(ok)
		r.Contains(newVPProof, "created")
		r.Contains(newVPProof, "proofValue")
		r.Equal("Ed25519Signature2018", newVPProof["type"])
	})
}
