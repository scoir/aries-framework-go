package indy

import (
	"fmt"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	vdriapi "github.com/hyperledger/aries-framework-go/pkg/framework/aries/api/vdri"
)

func (r *VDRI) Build(pubKey *vdriapi.PubKey, opts ...vdriapi.DocOpts) (*did.Doc, error) {

	docOpts := &vdriapi.CreateDIDOpts{}
	// Apply options
	for _, opt := range opts {
		opt(docOpts)
	}

	didDoc, err := build(pubKey, docOpts, r.didMethod)
	if err != nil {
		return nil, fmt.Errorf("create peer DID : %w", err)
	}

	return didDoc, nil
}

func build(pubKey *vdriapi.PubKey, docOpts *vdriapi.CreateDIDOpts, didMethod string) (*did.Doc, error) {
	publicKey := did.PublicKey{
		ID:         pubKey.Value[0:7],
		Type:       pubKey.Type,
		Controller: "#id",
		// TODO fix hardcode base58 https://github.com/hyperledger/aries-framework-go/issues/1207
		Value: base58.Decode(pubKey.Value),
	}

	// Service model to be included only if service type is provided through opts
	var service []did.Service

	if docOpts.ServiceType != "" {
		s := did.Service{
			ID:              "#agent",
			Type:            docOpts.ServiceType,
			ServiceEndpoint: docOpts.ServiceEndpoint,
			RoutingKeys:     docOpts.RoutingKeys,
		}

		if docOpts.ServiceType == vdriapi.DIDCommServiceType {
			s.RecipientKeys = []string{publicKey.ID}
			s.Priority = 0
		}

		service = append(service, s)
	}

	// Created/Updated time
	t := time.Now()

	return NewDoc(
		[]did.PublicKey{publicKey},
		[]did.VerificationMethod{
			{PublicKey: publicKey},
		},
		didMethod,
		did.WithService(service),
		did.WithCreatedTime(t),
		did.WithUpdatedTime(t),
	)
}
