/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package indy

import (
	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	vdriapi "github.com/hyperledger/aries-framework-go/pkg/framework/aries/api/vdri"
	"github.com/pkg/errors"
)

type endpoint struct {
	Endpoint string `json:"endpoint"`
}

// Store saves Peer DID Document along with user key/signature.
func (r *VDRI) Store(doc *did.Doc, by *[]vdriapi.ModifiedBy) error {
	if doc == nil || doc.ID == "" {
		return errors.New("DID and document are mandatory")
	}

	verkey := base58.Encode(doc.PublicKey[0].Value)
	err := r.CreateNym(doc.ID, verkey)
	if err != nil {
		return errors.Wrap(err, "unable to write nym to ledger")
	}

	endpoint := endpoint{doc.Service[0].ServiceEndpoint}
	err = r.CreateAttrib(doc.ID, verkey, map[string]interface{}{"endpoint": endpoint})
	if err != nil {
		return errors.Wrap(err, "unable to write nym to ledger")
	}

	return nil

}

// Close frees resources being maintained by vdri.
func (r *VDRI) Close() error {
	return nil
}
