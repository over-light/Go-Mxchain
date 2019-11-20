package preprocess_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/node/mock"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block/preprocess"
	"github.com/stretchr/testify/assert"
)

func TestNewGasConsumption_NilEconomicsFeeHandlerShouldErr(t *testing.T) {
	t.Parallel()

	gc, err := preprocess.NewGasComputation(
		nil,
	)

	assert.Nil(t, gc)
	assert.Equal(t, process.ErrNilEconomicsFeeHandler, err)
}

func TestNewGasConsumption_ShouldWork(t *testing.T) {
	t.Parallel()

	gc, err := preprocess.NewGasComputation(
		&mock.FeeHandlerStub{},
	)

	assert.NotNil(t, gc)
	assert.Nil(t, err)
}

func TestGasConsumed_ShouldWork(t *testing.T) {
	t.Parallel()

	gc, _ := preprocess.NewGasComputation(
		&mock.FeeHandlerStub{},
	)

	gc.SetGasConsumed(2)
	assert.Equal(t, uint64(2), gc.GasConsumed())

	gc.AddGasConsumed(1)
	assert.Equal(t, uint64(3), gc.GasConsumed())

	gc.Init()
	assert.Equal(t, uint64(0), gc.GasConsumed())
}

func TestGasRefunded_ShouldWork(t *testing.T) {
	t.Parallel()

	gc, _ := preprocess.NewGasComputation(
		&mock.FeeHandlerStub{},
	)

	gc.SetGasRefunded(2)
	assert.Equal(t, uint64(2), gc.GasRefunded())

	gc.AddGasRefunded(1)
	assert.Equal(t, uint64(3), gc.GasRefunded())

	gc.Init()
	assert.Equal(t, uint64(0), gc.GasRefunded())
}

func TestComputeGasConsumedByTx_ShouldErrWrongTypeAssertion(t *testing.T) {
	t.Parallel()

	gc, _ := preprocess.NewGasComputation(
		&mock.FeeHandlerStub{},
	)

	_, _, err := gc.ComputeGasConsumedByTx(0, 1, nil)
	assert.Equal(t, process.ErrWrongTypeAssertion, err)
}

func TestComputeGasConsumedByTx_ShouldErrInsufficientGasLimitInTx(t *testing.T) {
	t.Parallel()

	gc, _ := preprocess.NewGasComputation(
		&mock.FeeHandlerStub{
			ComputeGasLimitCalled: func(tx process.TransactionWithFeeHandler) uint64 {
				return 6
			},
		},
	)

	tx := transaction.Transaction{GasLimit: 5}

	_, _, err := gc.ComputeGasConsumedByTx(0, 1, &tx)
	assert.Equal(t, process.ErrInsufficientGasLimitInTx, err)
}

func TestComputeGasConsumedByTx_ShouldWorkWhenTxReceiverAddressIsNotASmartContract(t *testing.T) {
	t.Parallel()

	gc, _ := preprocess.NewGasComputation(
		&mock.FeeHandlerStub{
			ComputeGasLimitCalled: func(tx process.TransactionWithFeeHandler) uint64 {
				return 6
			},
		},
	)

	tx := transaction.Transaction{GasLimit: 7}

	gasInSnd, gasInRcv, _ := gc.ComputeGasConsumedByTx(0, 1, &tx)
	assert.Equal(t, uint64(6), gasInSnd)
	assert.Equal(t, uint64(6), gasInRcv)
}

func TestComputeGasConsumedByTx_ShouldWorkWhenTxReceiverAddressIsASmartContractInShard(t *testing.T) {
	t.Parallel()

	gc, _ := preprocess.NewGasComputation(
		&mock.FeeHandlerStub{
			ComputeGasLimitCalled: func(tx process.TransactionWithFeeHandler) uint64 {
				return 6
			},
		},
	)

	tx := transaction.Transaction{GasLimit: 7, RcvAddr: make([]byte, core.NumInitCharactersForScAddress+1)}

	gasInSnd, gasInRcv, _ := gc.ComputeGasConsumedByTx(0, 0, &tx)
	assert.Equal(t, uint64(7), gasInSnd)
	assert.Equal(t, uint64(7), gasInRcv)
}

func TestComputeGasConsumedByTx_ShouldWorkWhenTxReceiverAddressIsASmartContractCrossShard(t *testing.T) {
	t.Parallel()

	gc, _ := preprocess.NewGasComputation(
		&mock.FeeHandlerStub{
			ComputeGasLimitCalled: func(tx process.TransactionWithFeeHandler) uint64 {
				return 6
			},
		},
	)

	tx := transaction.Transaction{GasLimit: 7, RcvAddr: make([]byte, core.NumInitCharactersForScAddress+1)}

	gasInSnd, gasInRcv, _ := gc.ComputeGasConsumedByTx(0, 1, &tx)
	assert.Equal(t, uint64(6), gasInSnd)
	assert.Equal(t, uint64(1), gasInRcv)
}

func TestComputeGasConsumedByMiniBlock_ShouldErrMissingTransaction(t *testing.T) {
	t.Parallel()

	gc, _ := preprocess.NewGasComputation(
		&mock.FeeHandlerStub{
			ComputeGasLimitCalled: func(tx process.TransactionWithFeeHandler) uint64 {
				return 6
			},
		},
	)

	txHashes := make([][]byte, 0)
	txHashes = append(txHashes, []byte("hash1"))
	txHashes = append(txHashes, []byte("hash2"))

	miniBlock := block.MiniBlock{
		SenderShardID:   0,
		ReceiverShardID: 1,
		TxHashes:        txHashes,
	}

	mapHashTx := make(map[string]data.TransactionHandler)

	_, _, err := gc.ComputeGasConsumedByMiniBlock(&miniBlock, mapHashTx)
	assert.Equal(t, process.ErrMissingTransaction, err)
}

func TestComputeGasConsumedByMiniBlock_ShouldReturnZeroWhenOneTxIsMissing(t *testing.T) {
	t.Parallel()

	gc, _ := preprocess.NewGasComputation(
		&mock.FeeHandlerStub{
			ComputeGasLimitCalled: func(tx process.TransactionWithFeeHandler) uint64 {
				return 6
			},
		},
	)

	txHashes := make([][]byte, 0)
	txHashes = append(txHashes, []byte("hash1"))
	txHashes = append(txHashes, []byte("hash2"))

	miniBlock := block.MiniBlock{
		SenderShardID:   0,
		ReceiverShardID: 1,
		TxHashes:        txHashes,
	}

	mapHashTx := make(map[string]data.TransactionHandler)
	mapHashTx["hash1"] = nil
	mapHashTx["hash2"] = nil

	gasInSnd, gasInRcv, _ := gc.ComputeGasConsumedByMiniBlock(&miniBlock, mapHashTx)
	assert.Equal(t, uint64(0), gasInSnd)
	assert.Equal(t, uint64(0), gasInRcv)
}

func TestComputeGasConsumedByMiniBlock_ShouldWork(t *testing.T) {
	t.Parallel()

	gc, _ := preprocess.NewGasComputation(
		&mock.FeeHandlerStub{
			ComputeGasLimitCalled: func(tx process.TransactionWithFeeHandler) uint64 {
				return 6
			},
		},
	)

	txHashes := make([][]byte, 0)
	txHashes = append(txHashes, []byte("hash1"))
	txHashes = append(txHashes, []byte("hash2"))
	txHashes = append(txHashes, []byte("hash3"))

	miniBlock := block.MiniBlock{
		SenderShardID:   0,
		ReceiverShardID: 1,
		TxHashes:        txHashes,
	}

	mapHashTx := make(map[string]data.TransactionHandler)
	mapHashTx["hash1"] = &transaction.Transaction{GasLimit: 7}
	mapHashTx["hash2"] = &transaction.Transaction{GasLimit: 20, RcvAddr: make([]byte, core.NumInitCharactersForScAddress+1)}
	mapHashTx["hash3"] = &transaction.Transaction{GasLimit: 30, RcvAddr: make([]byte, core.NumInitCharactersForScAddress+1)}

	gasInSnd, gasInRcv, _ := gc.ComputeGasConsumedByMiniBlock(&miniBlock, mapHashTx)
	assert.Equal(t, uint64(18), gasInSnd)
	assert.Equal(t, uint64(44), gasInRcv)
}
