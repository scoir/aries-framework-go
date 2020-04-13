package indy

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/multiformats/go-multihash"
)

const (
	peerPrefix = "did:%s:"
)

// nolint:gochecknoglobals
var (
	didRegex = regexp.MustCompile(`did:scoir:([1-9a-km-zA-HJ-NP-Z]{22})`)
)

// NewDoc returns the resolved variant of the genesis version of the peer DID document
func NewDoc(publicKey []did.PublicKey, authentication []did.VerificationMethod, didMethod string,
	opts ...did.DocOption) (*did.Doc, error) {
	// build DID Doc
	doc := did.BuildDoc(opts...)

	// Create a did doc based on the mandatory value: publicKeys & authentication
	doc.PublicKey = publicKey
	doc.Authentication = authentication

	id, err := computeDid(doc, didMethod)
	if err != nil {
		return nil, err
	}

	doc.ID = id

	return doc, nil
}

// computeDid creates the peer DID.
// For example: did:peer:1zQmZMygzYqNwU6Uhmewx5Xepf2VLp5S4HLSwwgf2aiKZuwa
func computeDid(doc *did.Doc, didMethod string) (string, error) {
	if doc.PublicKey == nil || doc.Authentication == nil {
		return "", errors.New("the genesis version must include public keys and authentication")
	}

	encNumBasis, err := calculateEncNumBasis(doc)
	if err != nil {
		return "", err
	}

	messageIdentifier := []string{fmt.Sprintf(peerPrefix, didMethod), encNumBasis}

	return strings.Join(messageIdentifier, ""), nil
}

// validateDID checks the format of the doc's DID and checks that the DID's 'namestring' matches against its enc numeric
// basis as per the Namestring Generation Method.
// Reference: https://openssi.github.io/peer-did-method-spec/index.html#method-specific-identifier
//
// Note: this check should be done only on the resolved variant of the genesis version of Peer DID documents.
func validateDID(doc *did.Doc) error {
	peerDid := doc.ID

	matched := didRegex.MatchString(peerDid)
	if !matched {
		return fmt.Errorf("validate did : %w", errors.New("did doesnt follow matching regex"))
	}

	// extracting numbasis from the validated did
	splitDid := strings.FieldsFunc(peerDid, func(r rune) bool { return r == ':' })
	encnumbasis := splitDid[2]

	// genesis version(no did) of the peer DID doc
	genesisDoc := &did.Doc{
		Context:        doc.Context,
		ID:             "",
		PublicKey:      doc.PublicKey,
		Service:        doc.Service,
		Authentication: doc.Authentication,
		Created:        doc.Created,
		Updated:        doc.Updated,
		Proof:          doc.Proof,
	}

	// calculate the encnumbasis of the genesis version of the peer DID doc
	numBas, err := calculateEncNumBasis(genesisDoc)
	if err != nil {
		return fmt.Errorf("validate did : %w", err)
	}

	if !(numBas == encnumbasis) {
		return fmt.Errorf("validate did : %w", errors.New("multiHash of the doc doesnt match the computed multiHash"))
	}

	return nil
}

// calculateEncNumBasis is multicodec numeric basis.
func calculateEncNumBasis(doc *did.Doc) (string, error) {
	docBytes, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}

	hash, err := multihash.Sum(docBytes, multihash.SHA2_256, 32)
	if err != nil {
		return "", err
	}

	hash = hash[0:16]
	return hash.B58String(), nil
}
