package indy

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	TRUSTEE         = "0"
	STEWARD         = "2"
	ENDORSER        = "101"
	NETWORK_MONITOR = "201"
)

type NymRequest struct {
	Operation `json:",inline"`
	Dest      string `json:"dest"`
}

type Nym struct {
	Operation `json:",inline"`
	Dest      string `json:"dest"`
	Role      string `json:"role,omitempty"`
	Verkey    string `json:"verkey,omitempty"`
}

func NewNymRequest(did, from string) *Request {
	return &Request{
		Operation: NymRequest{
			Operation: Operation{Type: GET_NYM},
			Dest:      did,
		},
		Identifier:      from,
		ProtocolVersion: protocolVersion,
		ReqID:           uuid.New().ID(),
	}
}

func NewNym(did, verkey, from, role string) *Request {
	return &Request{
		Operation: Nym{
			Operation: Operation{Type: NYM},
			Dest:      did,
			Verkey:    verkey,
			Role:      role,
		},
		Identifier:      from,
		ReqID:           uuid.New().ID(),
		ProtocolVersion: protocolVersion,
	}
}

func (r *VDRI) GetNym(did string) (*Nym, error) {
	nymRequest := NewNymRequest(did, AgencyDID)
	resp, err := r.read(nymRequest)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get nym")
	}

	nym := &Nym{}
	err = json.Unmarshal([]byte(resp.Data), nym)
	if err != nil {
		return nil, errors.Wrap(err, "invalid nym data")
	}
	return nym, nil
}

func (r *VDRI) CreateNym(did, verkey string) error {
	fmt.Println("the ver key", verkey)
	nymRequest := NewNym(r.strip(did), verkey, AgencyDID, "")
	_, err := r.write(nymRequest, AgencyVerkey)
	if err != nil {
		return errors.Wrap(err, "unable to create nym")
	}

	return nil
}

//enckey, verkey, err := r.kms.CreateKeySet()
//if err != nil {
//	return nil, errors.Wrap(err, "error creating keyset")
//}
//did := base58.Encode(base58.Decode(enckey)[0:16])
//fmt.Println(verkey)
//fmt.Println(did)
//
