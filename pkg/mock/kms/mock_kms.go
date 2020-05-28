/*
 Copyright SecureKey Technologies Inc. All Rights Reserved.

 SPDX-License-Identifier: Apache-2.0
*/

package kms

import (
	"fmt"

	"github.com/google/tink/go/keyset"
	tinkpb "github.com/google/tink/go/proto/tink_go_proto"
	"github.com/google/tink/go/testkeyset"
	"github.com/google/tink/go/testutil"

	kmsservice "github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/pkg/secretlock"
	"github.com/hyperledger/aries-framework-go/pkg/storage"
)

// KeyManager mocks a local Key Management Service + ExportableKeyManager
type KeyManager struct {
	CreateKeyID              string
	CreateKeyValue           *keyset.Handle
	CreateKeyErr             error
	GetKeyValue              *keyset.Handle
	GetKeyErr                error
	RotateKeyID              string
	RotateKeyValue           *keyset.Handle
	RotateKeyErr             error
	ExportPubKeyBytesErr     error
	ExportPubKeyBytesValue   []byte
	PubKeyBytesToHandleErr   error
	PubKeyBytesToHandleValue *keyset.Handle
	ImportPrivateKeyErr      error
	ImportPrivateKeyID       string
	ImportPrivateKeyValue    *keyset.Handle
}

// Create a new mock ey/keyset/key handle for the type kt
func (k *KeyManager) Create(kt kmsservice.KeyType) (string, interface{}, error) {
	if k.CreateKeyErr != nil {
		return "", nil, k.CreateKeyErr
	}

	return k.CreateKeyID, k.CreateKeyValue, nil
}

// Get a mock key handle for the given keyID
func (k *KeyManager) Get(keyID string) (interface{}, error) {
	if k.GetKeyErr != nil {
		return nil, k.GetKeyErr
	}

	return k.GetKeyValue, nil
}

// Rotate returns a mocked rotated keyset handle and its ID
func (k *KeyManager) Rotate(kt kmsservice.KeyType, keyID string) (string, interface{}, error) {
	if k.RotateKeyErr != nil {
		return "", nil, k.RotateKeyErr
	}

	return k.RotateKeyID, k.RotateKeyValue, nil
}

// ExportPubKeyBytes will return a mocked []bytes public key
func (k *KeyManager) ExportPubKeyBytes(keyID string) ([]byte, error) {
	if k.ExportPubKeyBytesErr != nil {
		return nil, k.ExportPubKeyBytesErr
	}

	return k.ExportPubKeyBytesValue, nil
}

// PubKeyBytesToHandle will return a mocked keyset.Handle representing a public key handle
func (k *KeyManager) PubKeyBytesToHandle(pubKey []byte, keyType kmsservice.KeyType) (interface{}, error) {
	if k.PubKeyBytesToHandleErr != nil {
		return nil, k.PubKeyBytesToHandleErr
	}

	return k.PubKeyBytesToHandleValue, nil
}

// ImportPrivateKey will emulate importing a private key and returns a mocked keyID, private key handle
func (k *KeyManager) ImportPrivateKey(privKey interface{}, keyType kmsservice.KeyType,
	opts ...kmsservice.PrivateKeyOpts) (string, interface{}, error) {
	if k.ImportPrivateKeyErr != nil {
		return "", nil, k.ImportPrivateKeyErr
	}

	return k.ImportPrivateKeyID, k.ImportPrivateKeyValue, nil
}

// CreateMockKeyHandle is a utility function that returns a mock key (for tests only. ie: not registered in Tink)
func CreateMockKeyHandle() (*keyset.Handle, error) {
	ks := testutil.NewTestAESGCMKeyset(tinkpb.OutputPrefixType_TINK)
	primaryKey := ks.Key[0]

	if primaryKey.OutputPrefixType == tinkpb.OutputPrefixType_RAW {
		return nil, fmt.Errorf("expect a non-raw key")
	}

	return testkeyset.NewHandle(ks)
}

// Provider provides mock Provider implementation.
type Provider struct {
	storeProvider storage.Provider
	secretLock    secretlock.Service
}

// StorageProvider return a storage provider.
func (p *Provider) StorageProvider() storage.Provider {
	return p.storeProvider
}

// SecretLock returns a secret lock service.
func (p *Provider) SecretLock() secretlock.Service {
	return p.secretLock
}

// NewProviderForKMS creates a new mock Provider to create a KMS.
func NewProviderForKMS(storeProvider storage.Provider, secretLock secretlock.Service) *Provider {
	return &Provider{
		storeProvider: storeProvider,
		secretLock:    secretLock,
	}
}
