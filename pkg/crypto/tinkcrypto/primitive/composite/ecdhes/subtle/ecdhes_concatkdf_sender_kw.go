/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package subtle

import (
	"crypto/aes"
	"crypto/ecdsa"
	"crypto/rand"
	"math/big"

	"github.com/google/tink/go/subtle/hybrid"
	josecipher "github.com/square/go-jose/v3/cipher"

	ecdhespb "github.com/hyperledger/aries-framework-go/pkg/crypto/tinkcrypto/primitive/proto/ecdhes_aead_go_proto"
)

// A256KWAlg is the ECDH-ES key wrapping algorithm
const A256KWAlg = "ECDH-ES+A256KW"

// ECDHESConcatKDFSenderKW represents concat KDF based ECDH-ES KW (key wrapping)
// for ECDH-ES sender
type ECDHESConcatKDFSenderKW struct {
	recipientPublicKey *PublicKey
	cek                []byte
}

// wrapKey will do ECDH-ES key wrapping
func (s *ECDHESConcatKDFSenderKW) wrapKey(kwAlg string, keySize int) (*RecipientWrappedKey, error) {
	// TODO: add support for 25519 key wrapping https://github.com/hyperledger/aries-framework-go/issues/1637
	keyType := ecdhespb.KeyType_EC.String()

	c, err := hybrid.GetCurve(s.recipientPublicKey.Curve)
	if err != nil {
		return nil, err
	}

	recPubKey := &ecdsa.PublicKey{
		Curve: c,
		X:     new(big.Int).SetBytes(s.recipientPublicKey.X),
		Y:     new(big.Int).SetBytes(s.recipientPublicKey.Y),
	}

	ephemeralPriv, err := ecdsa.GenerateKey(recPubKey.Curve, rand.Reader)
	if err != nil {
		return nil, err
	}

	kek := josecipher.DeriveECDHES(kwAlg, []byte{}, []byte{}, ephemeralPriv, recPubKey, keySize)

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}

	wk, err := josecipher.KeyWrap(block, s.cek)
	if err != nil {
		return nil, err
	}

	return &RecipientWrappedKey{
		KID:          s.recipientPublicKey.KID,
		EncryptedCEK: wk,
		EPK: PublicKey{
			X:     ephemeralPriv.PublicKey.X.Bytes(),
			Y:     ephemeralPriv.PublicKey.Y.Bytes(),
			Curve: ephemeralPriv.PublicKey.Curve.Params().Name,
			Type:  keyType,
		},
		Alg: kwAlg,
	}, nil
}
