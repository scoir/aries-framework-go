/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package verifiable

import (
	"errors"
	"fmt"

	"github.com/hyperledger/aries-framework-go/pkg/doc/verifiable"
	"github.com/hyperledger/aries-framework-go/pkg/storage"
)

const (
	// NameSpace for vc store
	NameSpace = "verifiable"

	credentialNameKey            = "vcname_"
	credentialNameDataKeyPattern = credentialNameKey + "%s"

	// limitPattern for the iterator
	limitPattern = "%s~"
)

// ErrNotFound signals that the entry for the given DID and key is not present in the store.
var ErrNotFound = errors.New("did not found under given key")

// Store stores vc
type Store struct {
	store storage.Store
}

type provider interface {
	StorageProvider() storage.Provider
}

// New returns a new vc store
func New(ctx provider) (*Store, error) {
	store, err := ctx.StorageProvider().OpenStore(NameSpace)
	if err != nil {
		return nil, fmt.Errorf("failed to open vc store: %w", err)
	}

	return &Store{store: store}, nil
}

// SaveCredential saves a verifiable credential.
func (s *Store) SaveCredential(name string, vc *verifiable.Credential) error {
	if name == "" {
		return errors.New("credential name is mandatory")
	}

	id, err := s.GetCredentialIDByName(name)
	if err != nil && !errors.Is(err, storage.ErrDataNotFound) {
		return fmt.Errorf("get credential id using name : %w", err)
	}

	if id != "" {
		return fmt.Errorf("credential name %s already exists", name)
	}

	vcBytes, err := vc.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal vc: %w", err)
	}

	if err := s.store.Put(vc.ID, vcBytes); err != nil {
		return fmt.Errorf("failed to put vc: %w", err)
	}

	if err := s.store.Put(credentialNameDataKey(name), []byte(vc.ID)); err != nil {
		return fmt.Errorf("store vc name to id map : %w", err)
	}

	return nil
}

// GetCredential retrieves a verifiable credential based on ID.
func (s *Store) GetCredential(id string) (*verifiable.Credential, error) {
	vcBytes, err := s.store.Get(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get vc: %w", err)
	}

	vc, err := verifiable.NewUnverifiedCredential(vcBytes)
	if err != nil {
		return nil, fmt.Errorf("new credential failed: %w", err)
	}

	return vc, nil
}

// GetCredentialIDByName retrieves verifiable credential id based on name.
func (s *Store) GetCredentialIDByName(name string) (string, error) {
	idBytes, err := s.store.Get(credentialNameDataKey(name))
	if err != nil {
		return "", fmt.Errorf("fetch credential id based on name : %w", err)
	}

	return string(idBytes), nil
}

// GetCredentials retrieves the verifiable credential records containing name and vcID.
func (s *Store) GetCredentials() []*CredentialRecord {
	searchKey := credentialNameDataKey("")

	itr := s.store.Iterator(searchKey, fmt.Sprintf(limitPattern, searchKey))
	defer itr.Release()

	var records []*CredentialRecord

	for itr.Next() {
		record := &CredentialRecord{
			Name: getCredentialName(string(itr.Key())),
			ID:   string(itr.Value()),
		}

		records = append(records, record)
	}

	return records
}

func credentialNameDataKey(name string) string {
	return fmt.Sprintf(credentialNameDataKeyPattern, name)
}

func getCredentialName(dataKey string) string {
	return dataKey[len(credentialNameKey):]
}
