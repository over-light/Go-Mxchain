package processor

import (
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
)

// TxInterceptorProcessor is the processor used when intercepting transactions
// (smart contract results, receipts, transaction) structs which satisfy TransactionHandler interface.
type TxInterceptorProcessor struct {
	shardedPool ShardedPool
	txValidator process.TxValidator
}

// NewTxInterceptorProcessor creates a new TxInterceptorProcessor instance
func NewTxInterceptorProcessor(argument *ArgTxInterceptorProcessor) (*TxInterceptorProcessor, error) {
	if argument == nil {
		return nil, process.ErrNilArgumentStruct
	}
	if check.IfNil(argument.ShardedDataCache) {
		return nil, process.ErrNilDataPoolHolder
	}
	if check.IfNil(argument.TxValidator) {
		return nil, process.ErrNilTxValidator
	}

	return &TxInterceptorProcessor{
		shardedPool: argument.ShardedDataCache,
		txValidator: argument.TxValidator,
	}, nil
}

// Validate checks if the intercepted data can be processed
func (txip *TxInterceptorProcessor) Validate(data process.InterceptedData, _ p2p.PeerID) error {
	interceptedTx, ok := data.(InterceptedTransactionHandler)
	if !ok {
		return process.ErrWrongTypeAssertion
	}

	return txip.txValidator.CheckTxValidity(interceptedTx)
}

// Save will save the received data into the cacher
func (txip *TxInterceptorProcessor) Save(data process.InterceptedData, _ p2p.PeerID) error {
	interceptedTx, ok := data.(InterceptedTransactionHandler)
	if !ok {
		return process.ErrWrongTypeAssertion
	}

	cacherIdentifier := process.ShardCacherIdentifier(interceptedTx.SenderShardId(), interceptedTx.ReceiverShardId())
	txip.shardedPool.AddData(
		data.Hash(),
		interceptedTx.Transaction(),
		cacherIdentifier,
	)

	return nil
}

// SignalEndOfProcessing signals the end of processing
func (txip *TxInterceptorProcessor) SignalEndOfProcessing(_ []process.InterceptedData) {
}

// IsInterfaceNil returns true if there is no value under the interface
func (txip *TxInterceptorProcessor) IsInterfaceNil() bool {
	return txip == nil
}
