package indy

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
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


func (r *VDRI) GetNym(did string) (*Nym, error) {
	nymRequest := NewNymRequest(did, AgencyDID)
	resp, err := r.read(nymRequest)
	if err != nil {
		return nil, fmt.Errorf("unable to get nym: %v", err)
	}

	nym := &Nym{}
	err = json.Unmarshal([]byte(resp.Data), nym)
	if err != nil {
		return nil, fmt.Errorf("invalid nym data: %v", err)
	}
	return nym, nil
}
