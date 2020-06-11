package mock

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
)

// TxProcessorMock -
type TxProcessorMock struct {
	ProcessTransactionCalled         func(transaction *transaction.Transaction) error
	SetBalancesToTrieCalled          func(accBalance map[string]*big.Int) (rootHash []byte, err error)
	ProcessSmartContractResultCalled func(scr *smartContractResult.SmartContractResult) error
}

// ProcessTransaction -
func (tp *TxProcessorMock) ProcessTransaction(transaction *transaction.Transaction) error {
	return tp.ProcessTransactionCalled(transaction)
}

// SetBalancesToTrie -
func (tp *TxProcessorMock) SetBalancesToTrie(accBalance map[string]*big.Int) (rootHash []byte, err error) {
	return tp.SetBalancesToTrieCalled(accBalance)
}

// ProcessSmartContractResult -
func (tp *TxProcessorMock) ProcessSmartContractResult(scr *smartContractResult.SmartContractResult) (ReturnCode, error) {
	return 0, tp.ProcessSmartContractResultCalled(scr)
}

// IsInterfaceNil returns true if there is no value under the interface
func (tp *TxProcessorMock) IsInterfaceNil() bool {
	return tp == nil
}
