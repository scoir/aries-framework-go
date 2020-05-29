/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/
package httpbinding

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	vdriapi "github.com/hyperledger/aries-framework-go/pkg/framework/aries/api/vdri"
)

//nolint:lll
const doc = `{
  "@context": ["https://w3id.org/did/v1","https://w3id.org/did/v2"],
  "id": "did:peer:21tDAKCERh95uGgKbJNHYp",
  "publicKey": [
    {
      "id": "did:peer:123456789abcdefghi#keys-1",
      "type": "Secp256k1VerificationKey2018",
      "controller": "did:peer:123456789abcdefghi",
      "publicKeyBase58": "H3C2AVvLMv6gmMNam3uVAjZpfkcJCwDwnZn6z3wXmqPV"
    },
    {
      "id": "did:peer:123456789abcdefghw#key2",
      "type": "RsaVerificationKey2018",
      "controller": "did:peer:123456789abcdefghw",
      "publicKeyPem": "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAryQICCl6NZ5gDKrnSztO\n3Hy8PEUcuyvg/ikC+VcIo2SFFSf18a3IMYldIugqqqZCs4/4uVW3sbdLs/6PfgdX\n7O9D22ZiFWHPYA2k2N744MNiCD1UE+tJyllUhSblK48bn+v1oZHCM0nYQ2NqUkvS\nj+hwUU3RiWl7x3D2s9wSdNt7XUtW05a/FXehsPSiJfKvHJJnGOX0BgTvkLnkAOTd\nOrUZ/wK69Dzu4IvrN4vs9Nes8vbwPa/ddZEzGR0cQMt0JBkhk9kU/qwqUseP1QRJ\n5I1jR4g8aYPL/ke9K35PxZWuDp3U0UPAZ3PjFAh+5T+fc7gzCs9dPzSHloruU+gl\nFQIDAQAB\n-----END PUBLIC KEY-----"
    }
  ]
}`

//nolint:lll
const didResolutionData = `{
  "@context": "https://www.w3.org/ns/did-resolution/v1",
  "didDocument": ` + doc + `,
  "resolverMetadata": {
    "driverId": "did:example",
    "driver": "HttpDriver",
    "retrieved": "2019-06-01T19:73:24Z",
    "duration": 1015
  },
  "methodMetadata": {
    "nymResponse": {
      "result": {
        "type": "105",
        "txnTime": 1524055264,
        "seqNo": 11,
        "reqId": 1527256870802314800,
        "identifier": "HixkhyA4dXGz9yxmLQC4PU",
        "dest": "WRfXPg8dantKVubE3HX8pw"
      },
      "op": "REPLY"
    },
    "attrResponse": {
      "result": {
        "identifier": "HixkhyA4dXGz9yxmLQC4PU",
        "seqNo": 12,
        "raw": "endpoint",
        "dest": "WRfXPg8dantKVubE3HX8pw",
        "data": "{\"endpoint\":{\"xdi\":\"http://127.0.0.1:8080/xdi\"}}",
        "txnTime": 1524055265,
        "type": "104",
        "reqId": 1527256870925570600
      },
      "op": "REPLY"
    }
  }
}`

func TestWithOutboundOpts(t *testing.T) {
	opt := WithTimeout(1 * time.Second)
	require.NotNil(t, opt)

	clOpts := &VDRI{}
	// opt.client is nil, so setting timeout should panic
	require.Panics(t, func() { opt(clOpts) })

	opt = WithTLSConfig(nil)
	require.NotNil(t, opt)

	clOpts = &VDRI{}
	// opt.client is nil, so setting TLS config should panic
	require.Panics(t, func() { opt(clOpts) })
}

func TestNew(t *testing.T) {
	t.Run("test new with no options", func(t *testing.T) {
		var err error
		// OK with no options
		_, err = New("https://uniresolver.io/")
		require.NoError(t, err)
	})

	t.Run("test new with all options are applied", func(t *testing.T) {
		// All options are applied
		i := 0
		_, err := New("https://uniresolver.io/",
			func(opts *VDRI) {
				i += 1 // nolint
			},
			func(opts *VDRI) {
				i += 2
			},
		)
		require.NoError(t, err)
		require.Equal(t, 1+2, i)
	})

	t.Run("test new with invalid url", func(t *testing.T) {
		// Invalid URL
		_, err := New("invalid url")
		require.Error(t, err)
		require.Contains(t, err.Error(), "base URL invalid")

		r, err := New("https://uniresolver.io/", WithAccept(func(method string) bool {
			return false
		}))
		require.NoError(t, err)
		require.False(t, r.Accept("w"))
	})
}

func TestRead_DIDDoc(t *testing.T) {
	t.Run("test success return did doc", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			require.Equal(t, "/did:example:334455", req.URL.String())
			res.Header().Add("Content-type", "application/did+ld+json")
			res.WriteHeader(http.StatusOK)
			_, err := res.Write([]byte(doc))
			require.NoError(t, err)
		}))

		defer func() { testServer.Close() }()

		resolver, err := New(testServer.URL, WithResolveAuthToken("tk1"))
		require.NoError(t, err)
		gotDocument, err := resolver.Read("did:example:334455")
		require.NoError(t, err)
		didDoc, err := did.ParseDocument([]byte(doc))
		require.NoError(t, err)
		require.Equal(t, didDoc.ID, gotDocument.ID)
	})

	t.Run("test success return did resolution", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			require.Equal(t, "/did:example:334455", req.URL.String())
			res.Header().Add("Content-type", "application/did+ld+json")
			res.WriteHeader(http.StatusOK)
			_, err := res.Write([]byte(didResolutionData))
			require.NoError(t, err)
		}))

		defer func() { testServer.Close() }()

		resolver, err := New(testServer.URL)
		require.NoError(t, err)
		gotDocument, err := resolver.Read("did:example:334455")
		require.NoError(t, err)
		didDoc, err := did.ParseDocument([]byte(doc))
		require.NoError(t, err)
		require.Equal(t, didDoc.ID, gotDocument.ID)
	})
	t.Run("test empty doc", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			require.Equal(t, "/did:example:334455", req.URL.String())
			res.Header().Add("Content-type", "application/did+ld+json")
			res.WriteHeader(http.StatusOK)
			_, err := res.Write(nil)
			require.NoError(t, err)
		}))

		defer func() { testServer.Close() }()

		resolver, err := New(testServer.URL)
		require.NoError(t, err)
		_, err = resolver.Read("did:example:334455")
		require.Error(t, err)
		require.True(t, errors.Is(err, vdriapi.ErrNotFound))
	})
}

func TestRead_DIDDocWithBasePath(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/document/did:example:334455", req.URL.String())
		res.Header().Add("Content-type", "application/did+ld+json")
		res.WriteHeader(http.StatusOK)
		_, err := res.Write([]byte(doc))
		require.NoError(t, err)
	}))

	defer func() { testServer.Close() }()

	resolver, err := New(testServer.URL + "/document")
	require.NoError(t, err)
	gotDocument, err := resolver.Read("did:example:334455")
	require.NoError(t, err)
	didDoc, err := did.ParseDocument([]byte(doc))
	require.NoError(t, err)
	require.Equal(t, didDoc.ID, gotDocument.ID)
}

func TestRead_DIDDocWithBasePathWithSlashes(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/document/did:example:334455", req.URL.String())
		res.Header().Add("Content-type", "application/did+ld+json")
		res.WriteHeader(http.StatusOK)
		_, err := res.Write([]byte(doc))
		require.NoError(t, err)
	}))

	defer func() { testServer.Close() }()

	resolver, err := New(testServer.URL + "/document/")
	require.NoError(t, err)
	gotDocument, err := resolver.Read("did:example:334455")
	require.NoError(t, err)
	didDoc, err := did.ParseDocument([]byte(doc))
	require.NoError(t, err)
	require.Equal(t, didDoc.ID, gotDocument.ID)
}

func TestRead_DIDDocNotFound(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/did:example:334455", req.URL.String())
		res.WriteHeader(http.StatusNotFound)
		_, err := res.Write([]byte("did doc body"))
		require.NoError(t, err)
	}))

	defer func() { testServer.Close() }()

	resolver, err := New(testServer.URL)
	require.NoError(t, err)
	_, err = resolver.Read("did:example:334455")
	require.Error(t, err)
	require.Contains(t, err.Error(), "DID does not exist")
}

func TestRead_UnsupportedStatus(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusForbidden)
	}))

	defer func() { testServer.Close() }()

	resolver, err := New(testServer.URL)
	require.NoError(t, err)
	_, err = resolver.Read("did:example:334455")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported response from DID resolver")
}

func TestRead_HTTPGetFailed(t *testing.T) {
	// HTTP GET failed
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusSeeOther)
	}))

	defer func() { testServer.Close() }()

	resolver, err := New(testServer.URL)
	require.NoError(t, err)
	_, err = resolver.Read("did:example:334455")
	require.Error(t, err)
	require.Contains(t, err.Error(), "HTTP Get request failed")
}

func TestDIDResolver_Accept(t *testing.T) {
	resolver, err := New("localhost:8080")
	require.NoError(t, err)
	require.True(t, resolver.accept("example"))

	resolver, err = New("localhost:8080", WithAccept(func(method string) bool {
		return false
	}))
	require.NoError(t, err)
	require.False(t, resolver.accept("example"))
}
