/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package indy

import (
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	vdriapi "github.com/hyperledger/aries-framework-go/pkg/framework/aries/api/vdri"
)

// Read implements didresolver.DidMethod.Read interface (https://w3c-ccg.github.io/did-resolution/#resolving-input)
func (r *VDRI) Read(didID string, _ ...vdriapi.ResolveOpts) (*did.Doc, error) {
	// get the document from the store

	var service []did.Service
	var endpoint string

	short := r.strip(didID)
	nym, err := r.GetNym(short)
	if err != nil {
		return nil, fmt.Errorf("unable to get nym resolving DID: %v", err)
	}
	attrib, err := r.GetAttrib(short, "endpoint")
	if err != nil {
		return nil, fmt.Errorf("unable to get attrib resolving DID: %v", err)
	}

	mm, ok := attrib.Data["endpoint"].(map[string]interface{})
	if ok {
		endpoint = mm["endpoint"].(string)
	}

	publicKey := did.PublicKey{
		ID:         nym.Verkey,
		Type:       "Ed25519VerificationKey2018",
		Controller: "#id",
		// TODO fix hardcode base58 https://github.com/hyperledger/aries-framework-go/issues/1207
		Value: base58.Decode(nym.Verkey),
	}

	s := did.Service{
		ID:              "#agent",
		Type:            vdriapi.DIDCommServiceType,
		ServiceEndpoint: endpoint,
		RecipientKeys:   []string{nym.Verkey},
		Priority:        0,
	}

	service = append(service, s)

	doc := did.BuildDoc(did.WithService(service), did.WithPublicKey([]did.PublicKey{publicKey}))
	doc.ID = didID

	return doc, nil
}
