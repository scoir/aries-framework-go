/*
 Copyright SecureKey Technologies Inc. All Rights Reserved.

 SPDX-License-Identifier: Apache-2.0
*/

package localkms

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/google/tink/go/subtle/random"

	"github.com/hyperledger/aries-framework-go/pkg/storage"
)

const maxKeyIDLen = 20

// newWriter creates a new instance of local storage key storeWriter in the given store and for masterKeyURI
func newWriter(kmsStore storage.Store, opts ...PrivateKeyOpts) *storeWriter {
	pOpts := &privateKeyOpts{}

	for _, opt := range opts {
		opt(pOpts)
	}

	return &storeWriter{
		storage:           kmsStore,
		requestedKeysetID: pOpts.ksID,
	}
}

// storeWriter struct to store a keyset in a local store
type storeWriter struct {
	storage storage.Store
	//
	requestedKeysetID string
	// KeysetID is set when Write() is called
	KeysetID string
}

// Write a marshaled keyset p in localstore with masterKeyURI prefix + randomly generated KeysetID
func (l *storeWriter) Write(p []byte) (int, error) {
	var err error

	ksID := ""

	if l.requestedKeysetID != "" {
		ksID, err = l.verifyRequestedID()
		if err != nil {
			return 0, err
		}
	} else {
		ksID, err = l.newKeysetID()
		if err != nil {
			return 0, err
		}
	}

	err = l.storage.Put(ksID, p)
	if err != nil {
		return 0, err
	}

	l.KeysetID = ksID

	return len(p), nil
}

func (l *storeWriter) verifyRequestedID() (string, error) {
	if len(l.requestedKeysetID) > maxKeyIDLen {
		return "", fmt.Errorf("requested ID '%s' is longer than max allowed length of %d", l.requestedKeysetID,
			maxKeyIDLen)
	}

	_, err := l.storage.Get(l.requestedKeysetID)
	if errors.Is(err, storage.ErrDataNotFound) {
		return l.requestedKeysetID, nil
	}

	if err != nil {
		return "", fmt.Errorf("got error while verifying requested ID: %w", err)
	}

	return "", fmt.Errorf("requested ID '%s' already exists, cannot write keyset", l.requestedKeysetID)
}

func (l *storeWriter) newKeysetID() (string, error) {
	keySetIDLength := base64.RawURLEncoding.DecodedLen(maxKeyIDLen)
	ksID := ""

	for {
		// generate random ID
		ksID = base64.RawURLEncoding.EncodeToString(random.GetRandomBytes(uint32(keySetIDLength)))

		// skip IDs starting with '_' as some storage types reserve them for indexes (eg couchdb)
		if ksID[0] == '_' {
			continue
		}

		// ensure ksID is not already used
		_, err := l.storage.Get(ksID)
		if err != nil {
			if err == storage.ErrDataNotFound {
				break
			}

			return "", err
		}
	}

	return ksID, nil
}
