package transactionLog

import (
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/process"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

type nilTxLogProcessor struct {}

// NewNilTxLogProcessor returns a new txLogProcessor with no functionality for nodes
//  that don't want to process transaction logs
func NewNilTxLogProcessor() process.TransactionLogProcessor {
	return &nilTxLogProcessor{}
}

// GetLog retreives a log generated by a transaction
func (tlp *nilTxLogProcessor) GetLog(_ []byte) (data.LogHandler, error) {
	return nil, nil
}

// SaveLog takes the VM logs and saves them into the correct format in storage
func (tlp *nilTxLogProcessor) SaveLog(_ []byte, _ data.TransactionHandler, _ []*vmcommon.LogEntry) error {
	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (tlp *nilTxLogProcessor) IsInterfaceNil() bool {
	return tlp == nil
}
