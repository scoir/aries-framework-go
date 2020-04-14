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
