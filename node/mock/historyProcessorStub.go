package mock

import (
	"github.com/ElrondNetwork/elrond-go/core/fullHistory"
)

// HistoryProcessorStub -
type HistoryProcessorStub struct {
	PutTransactionsDataCalled func(htd *fullHistory.HistoryTransactionsData) error
	GetTransactionCalled      func(hash []byte) (*fullHistory.HistoryTransactionWithEpoch, error)
	IsEnabledCalled           func() bool
}

// PutTransactionsData will save in storage information about history transactions
func (hp *HistoryProcessorStub) PutTransactionsData(historyTxsData *fullHistory.HistoryTransactionsData) error {
	if hp.PutTransactionsDataCalled != nil {
		return hp.PutTransactionsDataCalled(historyTxsData)
	}
	return nil
}

// GetTransaction will return a history transaction for the given hash from storage
func (hp *HistoryProcessorStub) GetTransaction(hash []byte) (*fullHistory.HistoryTransactionWithEpoch, error) {
	if hp.GetTransactionCalled != nil {
		return hp.GetTransactionCalled(hash)
	}
	return nil, nil
}

// IsEnabled will always return true because this is implementation of a history processor
func (hp *HistoryProcessorStub) IsEnabled() bool {
	if hp.IsEnabledCalled != nil {
		return hp.IsEnabledCalled()
	}
	return true
}

// IsInterfaceNil returns true if there is no value under the interface
func (hp *HistoryProcessorStub) IsInterfaceNil() bool {
	return hp == nil
}
