/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ecdhes

import (
	"fmt"

	"github.com/google/tink/go/core/cryptofmt"
	"github.com/google/tink/go/core/primitiveset"
	"github.com/google/tink/go/core/registry"
	"github.com/google/tink/go/keyset"

	"github.com/hyperledger/aries-framework-go/pkg/crypto/tinkcrypto/primitive/composite/api"
)

// NewECDHESDecrypt returns an CompositeDecrypt primitive from the given keyset handle.
func NewECDHESDecrypt(h *keyset.Handle) (api.CompositeDecrypt, error) {
	return NewECDHESDecryptWithKeyManager(h, nil /*keyManager*/)
}

// NewECDHESDecryptWithKeyManager returns an CompositeDecrypt primitive from the given keyset handle and custom key
// manager.
func NewECDHESDecryptWithKeyManager(h *keyset.Handle, km registry.KeyManager) (api.CompositeDecrypt, error) {
	ps, err := h.PrimitivesWithKeyManager(km)
	if err != nil {
		return nil, fmt.Errorf("ecdhes_factory: cannot obtain primitive set: %s", err)
	}

	return newDecryptPrimitiveSet(ps)
}

// decryptPrimitiveSet is an CompositeDecrypt implementation that uses the underlying primitive set for
// decryption.
type decryptPrimitiveSet struct {
	ps *primitiveset.PrimitiveSet
}

// Asserts that primitiveSet implements the CompositeDecrypt interface.
var _ api.CompositeDecrypt = (*decryptPrimitiveSet)(nil)

func newDecryptPrimitiveSet(ps *primitiveset.PrimitiveSet) (*decryptPrimitiveSet, error) {
	if _, ok := (ps.Primary.Primitive).(api.CompositeDecrypt); !ok {
		return nil, fmt.Errorf("ecdhes_factory: not an CompositeDecrypt primitive")
	}

	for _, primitives := range ps.Entries {
		for _, p := range primitives {
			if _, ok := (p.Primitive).(api.CompositeDecrypt); !ok {
				return nil, fmt.Errorf("ecdhes_factory: not an CompositeDecrypt primitive")
			}
		}
	}

	ret := new(decryptPrimitiveSet)
	ret.ps = ps

	return ret, nil
}

// Decrypt decrypts the given ciphertext and authenticates it with the given
// additional authenticated data. It returns the corresponding plaintext if the
// ciphertext is authenticated.
func (a *decryptPrimitiveSet) Decrypt(ct, aad []byte) ([]byte, error) {
	// try non-raw keys
	prefixSize := cryptofmt.NonRawPrefixSize
	if len(ct) > prefixSize {
		prefix := ct[:prefixSize]
		ctNoPrefix := ct[prefixSize:]

		entries, err := a.ps.EntriesForPrefix(string(prefix))
		if err == nil {
			for i := 0; i < len(entries); i++ {
				p, ok := (entries[i].Primitive).(api.CompositeDecrypt)
				if !ok {
					return nil, fmt.Errorf("ecdhes_factory: not an CompositeDecrypt primitive")
				}

				pt, e := p.Decrypt(ctNoPrefix, aad)
				if e == nil {
					return pt, nil
				}
			}
		}
	}

	// try raw keys
	entries, err := a.ps.RawEntries()
	if err == nil {
		for i := 0; i < len(entries); i++ {
			p, ok := (entries[i].Primitive).(api.CompositeDecrypt)
			if !ok {
				return nil, fmt.Errorf("ecdhes_factory: not an CompositeDecrypt primitive")
			}

			pt, e := p.Decrypt(ct, aad)
			if e == nil {
				return pt, nil
			}
		}
	}

	// nothing worked
	return nil, fmt.Errorf("ecdhes_factory: decryption failed")
}
