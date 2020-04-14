package indy

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type AttribRequest struct {
	Operation `json:",inline"`
	Dest      string `json:"dest"`
	Raw       string `json:"raw,omitempty"`
	Hash      string `json:"hash,omitempty"`
	Enc       string `json:"enc,omitempty"`
}

type Attrib struct {
	Operation `json:",inline"`
	Dest      string                 `json:"dest"`
	Raw       interface{}            `json:"raw,omitempty"`
	Hash      string                 `json:"hash,omitempty"`
	Enc       string                 `json:"enc,omitempty"`
	Data      map[string]interface{} `json:"-"`
}

func NewRawAttribRequest(did, raw, from string) *Request {
	return newAttribRequest(AttribRequest{Operation: Operation{Type: GET_ATTR}, Dest: did, Raw: raw}, from)
}

func NewHashAttribRequest(did, data, from string) *Request {
	hash := sha256.New().Sum([]byte(data))
	return newAttribRequest(AttribRequest{Operation: Operation{Type: GET_ATTR}, Dest: did, Hash: string(hash)}, from)
}

func NewEncAttribRequest(did, data, from string) *Request {
	enc := data //TODO, figure out how to encrypt
	return newAttribRequest(AttribRequest{Operation: Operation{Type: GET_ATTR}, Dest: did, Enc: enc}, from)
}

func newAttribRequest(attrReq AttribRequest, from string) *Request {
	return &Request{
		Operation:       attrReq,
		Identifier:      from,
		ProtocolVersion: protocolVersion,
		ReqID:           uuid.New().ID(),
	}
}


func (r *VDRI) GetAttrib(did, raw string) (*Attrib, error) {
	attribRequest := NewRawAttribRequest(did, raw, AgencyDID)

	resp, err := r.read(attribRequest)
	if err != nil {
		return nil, fmt.Errorf("unable to get attribute: %v", err)
	}

	mdata := map[string]interface{}{}
	err = json.Unmarshal([]byte(resp.Data), &mdata)
	if err != nil {
		return nil, fmt.Errorf("invalid attrib data: %v", err)
	}

	attrib := &Attrib{Data: mdata}
	return attrib, nil

}