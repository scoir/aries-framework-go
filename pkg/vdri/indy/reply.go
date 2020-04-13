package indy

type ErrorReply struct {
	Op     string `json:"op"`
	Reason string `json:"reason"`
}

type ReadReply struct {
	Type       string     `json:"type"`
	Identifier string     `json:"identifier,omitempty"`
	ReqID      uint32     `json:"reqId"`
	SeqNo      uint32     `json:"seqNo"`
	TxnTime    uint32     `json:"TxnTime"`
	StateProof StateProof `json:"state_proof"`
	Data       string     `json:"data"`
}

type StateProof struct {
	RootHash        string     `json:"root_hash"`
	ProofNodes      string     `json:"proof_nodes"`
	MultiSignatures Signatures `json:"multi_signatures"`
}

type Signatures struct {
	Value        SigValue `json:"value"`
	Signature    string   `json:"signature"`
	Participants []string `json:"participants"`
}

type SigValue struct {
	Timestamp         int    `json:"timestamp"`
	LedgerID          int    `json:"ledger_id"`
	TxnRootHash       string `json:"txn_root_hash"`
	PoolStateRootHash string `json:"pool_state_root_hash"`
	StateRootHash     string `json:"state_root_hash"`
}

type WriteReply struct {
	Ver          string                      `json:"ver"`
	Txn          WriteReplyResultTxn         `json:"txn"`
	TxnMetadata  WriteReplyResultTxnMetadata `json:"txnMetadata"`
	ReqSignature ReqSignature                `json:"reqSignature"`
	RootHash     string                      `json:"rootHash"`
	AuditPath    []string                    `json:"auditPath"`
}

type WriteReplyResultTxn struct {
	Type            string                 `json:"type"`
	ProtocolVersion int                    `json:"protocolVersion"`
	Data            map[string]interface{} `json:"data"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type WriteReplyResultTxnMetadata struct {
}

type Metadata struct {
	TxnTime int    `json:"txnTime"`
	SeqNo   int    `json:"seqNo"`
	TxnID   string `json:"txnId"`
}
type ReqSignature struct {
	Type   string        `json:"type"`
	Values []ReqSigValue `json:"values"`
}

type ReqSigValue struct {
	From  string `json:"from"`
	Value string `json:"value"`
}
