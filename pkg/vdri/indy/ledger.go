package indy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/btcsuite/btcutil/base58"
	"github.com/pkg/errors"
)

const (
	AgencyDID    = "Tx2uToSkAKQj9TzvDYknhn"
	AgencyVerkey = "2ic4nKVNbkYwK3MgcyRLBcrUWJ8kByHLPMgzdERdovH3"
)

func (r *VDRI) endpoint() string {
	return fmt.Sprintf("tcp://%s:%d", r.clientIP, r.clientPort)
}

func klose(r io.ReadCloser, msg ...string) {
	err := r.Close()
	if err != nil {
		log.Printf("error closing %s: (%v)\n", msg, err)
	}
}

func (r *VDRI) write(req *Request, verkey string) (*WriteReply, error) {
	d, _ := json.MarshalIndent(req, " ", "")
	m := map[string]interface{}{}
	_ = json.Unmarshal(d, &m)

	ser, err := SerializeSignature(m)
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate signature")
	}

	sig, err := r.kms.SignMessage([]byte(ser), verkey)
	if err != nil {
		return nil, errors.Wrap(err, "unable to sign write request")
	}

	req.Signature = base58.Encode(sig)
	d, err = json.MarshalIndent(req, " ", "")
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal write request")
	}

	resp, err := r.send(d)
	if err != nil {
		return nil, errors.Wrap(err, "unable to write")
	}

	rply := &WriteReply{}
	err = json.Unmarshal(resp, rply)
	if err != nil {
		return nil, errors.Wrap(err, "invalid response to write request")
	}

	return rply, nil
}

func (r *VDRI) read(req *Request) (*ReadReply, error) {
	d, err := json.MarshalIndent(req, " ", " ")
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal nym_request")
	}

	resp, err := r.send(d)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read")
	}

	rply := &ReadReply{}
	err = json.Unmarshal(resp, rply)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal read reply")
	}
	return rply, nil
}

func (r *VDRI) send(req []byte) ([]byte, error) {
	buf := bytes.NewReader(req)
	resp, err := http.Post(fmt.Sprintf("http://%s/submit", r.vdrAddress), "application/json", buf)
	if err != nil {
		return nil, errors.Wrap(err, "unable to post to vdr")
	}
	defer klose(resp.Body)

	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read body of response")
	}

	if resp.StatusCode != http.StatusOK {
		er := &ErrorReply{}
		_ = json.Unmarshal(d, er)
		return nil, errors.Errorf("error %s from ledger reading attrib: %s", er.Op, er.Reason)
	}

	return d, nil
}
