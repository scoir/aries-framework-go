package indy

import (
	"fmt"

	"github.com/hyperledger/aries-framework-go/pkg/common/log"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	vdriapi "github.com/hyperledger/aries-framework-go/pkg/framework/aries/api/vdri"
	"github.com/hyperledger/aries-framework-go/pkg/kms/legacykms"
)

const (
	// StoreNamespace store name space for DID Store
	DefaultVDRHost = "0.0.0.0"
	DefaulVDRPort  = 4300
)

var logger = log.New("aries-framework/vdri/indy")

type provider interface {
	LegacyKMS() legacykms.KeyManager
	Signer() legacykms.Signer
}

// VDRI implements building new peer did
type VDRI struct {
	didMethod  string
	prefix     string
	vdrAddress string
	kms        legacykms.KMS
}

// New return new instance of indy vdri
func New(didMethod, genesisURL string, kms legacykms.KMS, opts ...Option) (*VDRI, error) {
	vdri := &VDRI{
		didMethod:  didMethod,
		prefix:     fmt.Sprintf("did:%s:", didMethod),
		vdrAddress: fmt.Sprintf("%s:%d", DefaultVDRHost, DefaulVDRPort),
		kms:        kms,
	}

	for _, opt := range opts {
		opt(vdri)
	}

	return vdri, nil
}

// Store did doc
func (r *VDRI) Store(doc *did.Doc, by *[]vdriapi.ModifiedBy) error {
	logger.Warnf(" store not supported in http binding vdri")
	return nil
}

// Close frees resources being maintained by vdri.
func (r *VDRI) Close() error {
	return nil
}

// Accept did method
func (r *VDRI) Accept(method string) bool {
	return method == r.didMethod
}

func (r *VDRI) strip(didID string) string {
	return didID[len(r.prefix):]
}

type Option func(opts *VDRI)

// WithTimeout option is for definition of HTTP(s) timeout value of DID Resolver
func WithVDRAddress(vdrAddress string) Option {
	return func(opts *VDRI) {
		opts.vdrAddress = vdrAddress
	}
}
