package indy

type Record struct {
	ReqSignature ReqSignature `json:"reqSignature"`
	Txn          Transaction  `json:"txn"`
	TxnMetadata  Metadata     `json:"txnMetadata"`
	Ver          string       `json:"ver"`
}

type TxData struct {
	Data map[string]interface{}
}

type Transaction struct {
	Data TxData `json:"data"`
	Dest string `json:"dest"`
}
