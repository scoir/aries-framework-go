package indy

const (
	NODE          = "0"
	NYM           = "1"
	ATTRIB        = "100"
	SCHEMA        = "101"
	CLAIM_DEF     = "102"
	POOL_UPGRADE  = "109"
	NODE_UPGRADE  = "110"
	POOL_CONFIG   = "111"
	GET_TXN       = "3"
	GET_ATTR      = "104"
	GET_NYM       = "105"
	GET_SCHEMA    = "107"
	GET_CLAIM_DEF = "108"

	protocolVersion = 2
)

type Request struct {
	Operation       interface{} `json:"operation"`
	Identifier      string      `json:"identifier,omitempty"`
	Endorser        string      `json:"endorser,omitempty"`
	ReqID           uint32      `json:"reqId"`
	ProtocolVersion int         `json:"protocolVersion"`
	Signature       string      `json:"signature,omitempty"`
}

type Operation struct {
	Type string `json:"type"`
}
