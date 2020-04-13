package indy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hyperledger/aries-framework-go/pkg/kms/legacykms"
	"github.com/pkg/errors"
)

const (
	// StoreNamespace store name space for DID Store
	DefaultVDRHost = "0.0.0.0"
	DefaulVDRPort  = 4300
)

type provider interface {
	LegacyKMS() legacykms.KeyManager
	Signer() legacykms.Signer
}

// VDRI implements building new peer did
type VDRI struct {
	didMethod  string
	prefix     string
	vdrAddress string
	genesis    []Record
	clientIP   string
	clientPort int
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

	resp, err := http.Get(genesisURL)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to retrieve genesis file from %s", genesisURL)
	}
	defer klose(resp.Body)
	gread := bufio.NewScanner(resp.Body)

	var genesis []Record
	for gread.Scan() {
		rec := Record{}
		err = json.Unmarshal(gread.Bytes(), &rec)
		if err != nil {
			return nil, errors.Wrap(err, "error decoding genesis")
		}
		genesis = append(genesis, rec)
	}

	node1Data := genesis[0].Txn.Data.Data
	vdri.genesis = genesis
	vdri.clientIP = node1Data["client_ip"].(string)
	vdri.clientPort = int(node1Data["client_port"].(float64))

	return vdri, nil
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
