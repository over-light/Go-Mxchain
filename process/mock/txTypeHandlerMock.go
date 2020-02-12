package mock

import (
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/process"
)

// TxTypeHandlerMock -
type TxTypeHandlerMock struct {
	ComputeTransactionTypeCalled func(tx data.TransactionHandler) (process.TransactionType, error)
}

// ComputeTransactionType -
func (th *TxTypeHandlerMock) ComputeTransactionType(tx data.TransactionHandler) (process.TransactionType, error) {
	if th.ComputeTransactionTypeCalled == nil {
		return process.MoveBalance, nil
	}

	return th.ComputeTransactionTypeCalled(tx)
}

// IsInterfaceNil returns true if there is no value under the interface
func (th *TxTypeHandlerMock) IsInterfaceNil() bool {
	if th == nil {
		return true
	}
	return false
}
