package indy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	AgencyDID    = "Tx2uToSkAKQj9TzvDYknhn"
	AgencyVerkey = "2ic4nKVNbkYwK3MgcyRLBcrUWJ8kByHLPMgzdERdovH3"
)

func klose(r io.ReadCloser, msg ...string) {
	err := r.Close()
	if err != nil {
		log.Printf("error closing %s: (%v)\n", msg, err)
	}
}

func (r *VDRI) read(req *Request) (*ReadReply, error) {
	d, err := json.MarshalIndent(req, " ", " ")
	if err != nil {
		return nil, fmt.Errorf("unable to marshal nym_request: %v", err)
	}

	resp, err := r.send(d)
	if err != nil {
		return nil, fmt.Errorf("unable to read: %v", err)
	}

	rply := &ReadReply{}
	err = json.Unmarshal(resp, rply)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal read reply: %v", err)
	}
	return rply, nil
}

func (r *VDRI) send(req []byte) ([]byte, error) {
	buf := bytes.NewReader(req)
	resp, err := http.Post(fmt.Sprintf("http://%s/submit", r.vdrAddress), "application/json", buf)
	if err != nil {
		return nil, fmt.Errorf("unable to post to vdr: %v", err)
	}
	defer klose(resp.Body)

	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read body of response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		er := &ErrorReply{}
		_ = json.Unmarshal(d, er)
		return nil, fmt.Errorf("error %s from ledger reading attrib: %s", er.Op, er.Reason)
	}

	return d, nil
}
