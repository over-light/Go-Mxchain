package transaction

import "github.com/ElrondNetwork/elrond-go/core"

// ApiTransactionResult is the data transfer object which will be returned on the get transaction by hash endpoint
type ApiTransactionResult struct {
	Type      string                 `json:"type"`
	Nonce     uint64                 `json:"nonce,omitempty"`
	Round     uint64                 `json:"round,omitempty"`
	Epoch     uint32                 `json:"epoch,omitempty"`
	Value     string                 `json:"value,omitempty"`
	Receiver  string                 `json:"receiver,omitempty"`
	Sender    string                 `json:"sender,omitempty"`
	GasPrice  uint64                 `json:"gasPrice,omitempty"`
	GasLimit  uint64                 `json:"gasLimit,omitempty"`
	Data      []byte                 `json:"data,omitempty"`
	Code      string                 `json:"code,omitempty"`
	Signature string                 `json:"signature,omitempty"`
	Status    core.TransactionStatus `json:"status,omitempty"`
}
