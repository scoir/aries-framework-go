/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package verifiable

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// UniversityDegree university degree
type UniversityDegree struct {
	Type       string `json:"type,omitempty"`
	Name       string `json:"name,omitempty"`
	College    string `json:"college,omitempty"`
	University string `json:"university,omitempty"`
}

// UniversityDegreeSubject subject of university degree
type UniversityDegreeSubject struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Spouse string `json:"spouse,omitempty"`

	Degree UniversityDegree `json:"degree,omitempty"`
}

// UniversityDegreeCredential University Degree credential, from examples of https://w3c.github.io/vc-data-model
type UniversityDegreeCredential struct {
	Base Credential `json:"-"`

	Subject *UniversityDegreeSubject `json:"credentialSubject,omitempty"`
}

func NewUniversityDegreeCredential(vcData []byte, opts ...CredentialOpt) (*UniversityDegreeCredential, error) {
	cred, err := parseTestCredential(vcData, opts...)
	if err != nil {
		return nil, fmt.Errorf("new university degree credential: %w", err)
	}

	udc := UniversityDegreeCredential{
		Base: *cred,
	}

	credBytes, err := json.Marshal(cred)
	if err != nil {
		return nil, fmt.Errorf("new university degree credential: %w", err)
	}

	err = json.Unmarshal(credBytes, &udc)
	if err != nil {
		return nil, fmt.Errorf("new university degree credential: %w", err)
	}

	return &udc, nil
}

func TestCredentialExtensibility(t *testing.T) {
	//nolint:lll
	udCredential := `

{
  "@context": [
    "https://www.w3.org/2018/credentials/v1",
    "https://www.w3.org/2018/credentials/examples/v1"
  ],
  "id": "http://example.edu/credentials/1872",
  "type": [
    "VerifiableCredential",
    "UniversityDegreeCredential"
  ],
  "credentialSubject": {
    "id": "did:example:ebfeb1f712ebc6f1c276e12ec21",
    "degree": {
      "type": "BachelorDegree"
    },
    "name": "Jayden Doe",
    "spouse": "did:example:c276e12ec21ebfeb1f712ebc6f1"
  },

  "issuer": {
    "id": "did:example:76e12ec712ebc6f1c221ebfeb1f",
    "name": "Example University"
  },

  "issuanceDate": "2010-01-01T19:23:24Z",

  "expirationDate": "2020-01-01T19:23:24Z",

  "credentialStatus": {
    "id": "https://example.edu/status/24",
    "type": "CredentialStatusList2017"
  },

  "evidence": [{
    "id": "https://example.edu/evidence/f2aeec97-fc0d-42bf-8ca7-0548192d4231",
    "type": ["DocumentVerification"],
    "verifier": "https://example.edu/issuers/14",
    "evidenceDocument": "DriversLicense",
    "subjectPresence": "Physical",
    "documentPresence": "Physical"
  },{
    "id": "https://example.edu/evidence/f2aeec97-fc0d-42bf-8ca7-0548192dxyzab",
    "type": ["SupportingActivity"],
    "verifier": "https://example.edu/issuers/14",
    "evidenceDocument": "Fluid Dynamics Focus",
    "subjectPresence": "Digital",
    "documentPresence": "Digital"
  }],

  "termsOfUse": [
    {
      "type": "IssuerPolicy",
      "id": "http://example.com/policies/credential/4",
      "profile": "http://example.com/profiles/credential",
      "prohibition": [
        {
          "assigner": "https://example.edu/issuers/14",
          "assignee": "AllVerifiers",
          "target": "http://example.edu/credentials/3732",
          "action": [
            "Archival"
          ]
        }
      ]
    }
  ],

  "refreshService": {
    "id": "https://example.edu/refresh/3732",
    "type": "ManualRefreshService2018"
  }
}
`

	cred, err := parseTestCredential([]byte(udCredential))
	require.NoError(t, err)
	require.NotNil(t, cred)

	udc, err := NewUniversityDegreeCredential([]byte(udCredential))
	require.NoError(t, err)

	// base Credential part is the same
	require.Equal(t, *cred, udc.Base)

	// default issuer credential decoder is applied (i.e. not re-written by new custom decoder)
	require.NotNil(t, cred.Issuer)
	require.Equal(t, cred.Issuer.ID, "did:example:76e12ec712ebc6f1c221ebfeb1f")
	require.Equal(t, cred.Issuer.CustomFields["name"], "Example University")

	// new mapping is applied
	subj := udc.Subject
	require.Equal(t, "did:example:ebfeb1f712ebc6f1c276e12ec21", subj.ID)
	require.Equal(t, "BachelorDegree", subj.Degree.Type)
	require.Equal(t, "Jayden Doe", subj.Name)
	require.Equal(t, "did:example:c276e12ec21ebfeb1f712ebc6f1", subj.Spouse)
}
