/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package subtle

import (
	"crypto/aes"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/google/tink/go/subtle/hybrid"
	josecipher "github.com/square/go-jose/v3/cipher"
)

// ECDHESConcatKDFRecipientKW represents concat KDF based ECDH-ES KW (key wrapping)
// for ECDH-ES recipient's unwrapping of CEK
type ECDHESConcatKDFRecipientKW struct {
	recipientPrivateKey *hybrid.ECPrivateKey
}

// unwrapKey will do ECDH-ES key unwrapping
func (s *ECDHESConcatKDFRecipientKW) unwrapKey(recWK *RecipientWrappedKey, keySize int) ([]byte, error) {
	if recWK == nil {
		return nil, fmt.Errorf("unwrapKey: RecipientWrappedKey is empty")
	}

	recPrivKey := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: s.recipientPrivateKey.PublicKey.Curve,
			X:     s.recipientPrivateKey.PublicKey.Point.X,
			Y:     s.recipientPrivateKey.PublicKey.Point.Y,
		},
		D: s.recipientPrivateKey.D,
	}

	epkCurve, err := hybrid.GetCurve(recWK.EPK.Curve)
	if err != nil {
		return nil, err
	}

	epkPubKey := &ecdsa.PublicKey{
		Curve: epkCurve,
		X:     new(big.Int).SetBytes(recWK.EPK.X),
		Y:     new(big.Int).SetBytes(recWK.EPK.Y),
	}

	// DeriveECDHES checks if keys are on the same curve
	kek := josecipher.DeriveECDHES(recWK.Alg, []byte{}, []byte{}, recPrivKey, epkPubKey, keySize)

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}

	return josecipher.KeyUnwrap(block, recWK.EncryptedCEK)
}
