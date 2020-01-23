package preprocess

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/crypto/signing"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/kyber"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/txpool"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/stretchr/testify/assert"
)

const MaxGasLimitPerBlock = uint64(100000)

func feeHandlerMock() *mock.FeeHandlerStub {
	return &mock.FeeHandlerStub{
		ComputeGasLimitCalled: func(tx process.TransactionWithFeeHandler) uint64 {
			return 0
		},
		MaxGasLimitPerBlockCalled: func() uint64 {
			return MaxGasLimitPerBlock
		},
	}
}

func miniBlocksCompacterMock() *mock.MiniBlocksCompacterMock {
	return &mock.MiniBlocksCompacterMock{
		CompactCalled: func(miniBlocks block.MiniBlockSlice, mpaHashesAndTxs map[string]data.TransactionHandler) block.MiniBlockSlice {
			return miniBlocks
		},
		ExpandCalled: func(miniBlocks block.MiniBlockSlice, mapHashesAntTxs map[string]data.TransactionHandler) (block.MiniBlockSlice, error) {
			return miniBlocks, nil
		},
	}
}

func initDataPool() *mock.PoolsHolderStub {
	sdp := &mock.PoolsHolderStub{
		TransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{
				RegisterHandlerCalled: func(i func(key []byte)) {},
				ShardDataStoreCalled: func(id string) (c storage.Cacher) {
					return &mock.CacherStub{
						PeekCalled: func(key []byte) (value interface{}, ok bool) {
							if reflect.DeepEqual(key, []byte("tx1_hash")) {
								return &transaction.Transaction{Nonce: 10}, true
							}
							return nil, false
						},
						KeysCalled: func() [][]byte {
							return [][]byte{[]byte("key1"), []byte("key2")}
						},
						LenCalled: func() int {
							return 0
						},
					}
				},
				AddDataCalled:                 func(key []byte, data interface{}, cacheId string) {},
				RemoveSetOfDataFromPoolCalled: func(keys [][]byte, id string) {},
				SearchFirstDataCalled: func(key []byte) (value interface{}, ok bool) {
					if reflect.DeepEqual(key, []byte("tx1_hash")) {
						return &transaction.Transaction{Nonce: 10}, true
					}
					return nil, false
				},
			}
		},
		UnsignedTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{
				RegisterHandlerCalled: func(i func(key []byte)) {},
				ShardDataStoreCalled: func(id string) (c storage.Cacher) {
					return &mock.CacherStub{
						PeekCalled: func(key []byte) (value interface{}, ok bool) {
							if reflect.DeepEqual(key, []byte("tx1_hash")) {
								return &smartContractResult.SmartContractResult{Nonce: 10}, true
							}
							return nil, false
						},
						KeysCalled: func() [][]byte {
							return [][]byte{[]byte("key1"), []byte("key2")}
						},
						LenCalled: func() int {
							return 0
						},
					}
				},
				AddDataCalled:                 func(key []byte, data interface{}, cacheId string) {},
				RemoveSetOfDataFromPoolCalled: func(keys [][]byte, id string) {},
				SearchFirstDataCalled: func(key []byte) (value interface{}, ok bool) {
					if reflect.DeepEqual(key, []byte("tx1_hash")) {
						return &smartContractResult.SmartContractResult{Nonce: 10}, true
					}
					return nil, false
				},
			}
		},
		RewardTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{
				RegisterHandlerCalled: func(i func(key []byte)) {},
				ShardDataStoreCalled: func(id string) (c storage.Cacher) {
					return &mock.CacherStub{
						PeekCalled: func(key []byte) (value interface{}, ok bool) {
							if reflect.DeepEqual(key, []byte("tx1_hash")) {
								return &rewardTx.RewardTx{Value: big.NewInt(100)}, true
							}
							return nil, false
						},
						KeysCalled: func() [][]byte {
							return [][]byte{[]byte("key1"), []byte("key2")}
						},
						LenCalled: func() int {
							return 0
						},
					}
				},
				AddDataCalled:                 func(key []byte, data interface{}, cacheId string) {},
				RemoveSetOfDataFromPoolCalled: func(keys [][]byte, id string) {},
				SearchFirstDataCalled: func(key []byte) (value interface{}, ok bool) {
					if reflect.DeepEqual(key, []byte("tx1_hash")) {
						return &rewardTx.RewardTx{Value: big.NewInt(100)}, true
					}
					return nil, false
				},
			}
		},
		MetaBlocksCalled: func() storage.Cacher {
			return &mock.CacherStub{
				GetCalled: func(key []byte) (value interface{}, ok bool) {
					if reflect.DeepEqual(key, []byte("tx1_hash")) {
						return &transaction.Transaction{Nonce: 10}, true
					}
					return nil, false
				},
				KeysCalled: func() [][]byte {
					return nil
				},
				LenCalled: func() int {
					return 0
				},
				PeekCalled: func(key []byte) (value interface{}, ok bool) {
					if reflect.DeepEqual(key, []byte("tx1_hash")) {
						return &transaction.Transaction{Nonce: 10}, true
					}
					return nil, false
				},
				RegisterHandlerCalled: func(i func(key []byte)) {},
			}
		},
		MiniBlocksCalled: func() storage.Cacher {
			cs := &mock.CacherStub{}
			cs.RegisterHandlerCalled = func(i func(key []byte)) {
			}
			cs.GetCalled = func(key []byte) (value interface{}, ok bool) {
				if bytes.Equal([]byte("bbb"), key) {
					return make(block.MiniBlockSlice, 0), true
				}

				return nil, false
			}
			cs.PeekCalled = func(key []byte) (value interface{}, ok bool) {
				if bytes.Equal([]byte("bbb"), key) {
					return make(block.MiniBlockSlice, 0), true
				}

				return nil, false
			}
			cs.RegisterHandlerCalled = func(i func(key []byte)) {}
			cs.RemoveCalled = func(key []byte) {}
			return cs
		},
		HeadersCalled: func() dataRetriever.HeadersPool {
			cs := &mock.HeadersCacherStub{}
			cs.RegisterHandlerCalled = func(i func(header data.HeaderHandler, key []byte)) {
			}
			return cs
		},
	}
	return sdp
}

func TestFzxc(t *testing.T) {
	a := uint32(7)
	b := a - uint32(14)
	fmt.Println(b)
}

func TestTxsPreprocessor_NewTransactionPreprocessorNilPool(t *testing.T) {
	t.Parallel()

	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewTransactionPreprocessor(
		nil,
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilTransactionPool, err)
}

func TestTxsPreprocessor_NewTransactionPreprocessorNilStore(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewTransactionPreprocessor(
		tdp.Transactions(),
		nil,
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilTxStorage, err)
}

func TestTxsPreprocessor_NewTransactionPreprocessorNilHasher(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewTransactionPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		nil,
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilHasher, err)
}

func TestTxsPreprocessor_NewTransactionPreprocessorNilMarsalizer(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewTransactionPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		nil,
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilMarshalizer, err)
}

func TestTxsPreprocessor_NewTransactionPreprocessorNilTxProce(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewTransactionPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		nil,
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilTxProcessor, err)
}

func TestTxsPreprocessor_NewTransactionPreprocessorNilShardCoord(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewTransactionPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		nil,
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilShardCoordinator, err)
}

func TestTxsPreprocessor_NewTransactionPreprocessorNilAccounts(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewTransactionPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		nil,
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilAccountsAdapter, err)
}

func TestTxsPreprocessor_NewTransactionPreprocessorNilRequestFunc(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	txs, err := NewTransactionPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		nil,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilRequestHandler, err)
}

func TestTxsPreprocessor_NewTransactionPreprocessorNilFeeHandler(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewTransactionPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		nil,
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilEconomicsFeeHandler, err)
}

func TestTxsPreprocessor_NewTransactionPreprocessorNilMiniBlocksCompacter(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewTransactionPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		nil,
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilMiniBlocksCompacter, err)
}

func TestTxsPreprocessor_NewTransactionPreprocessorNilGasHandler(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewTransactionPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		nil,
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilGasHandler, err)
}

func TestTxsPreprocessor_NewTransactionPreprocessorNilBlockTracker(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewTransactionPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		nil,
		block.TxBlock,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilBlockTracker, err)
}

func TestTxsPreProcessor_GetTransactionFromPool(t *testing.T) {
	t.Parallel()
	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewTransactionPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)
	txHash := []byte("tx1_hash")
	tx, _ := process.GetTransactionHandlerFromPool(1, 1, txHash, tdp.Transactions(), false)
	assert.NotNil(t, txs)
	assert.NotNil(t, tx)
	assert.Equal(t, uint64(10), tx.(*transaction.Transaction).Nonce)
}

func TestTransactionPreprocessor_RequestTransactionFromNetwork(t *testing.T) {
	t.Parallel()
	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewTransactionPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)
	shardId := uint32(1)
	txHash1 := []byte("tx_hash1")
	txHash2 := []byte("tx_hash2")
	body := make(block.Body, 0)
	txHashes := make([][]byte, 0)
	txHashes = append(txHashes, txHash1)
	txHashes = append(txHashes, txHash2)
	mBlk := block.MiniBlock{ReceiverShardID: shardId, TxHashes: txHashes}
	body = append(body, &mBlk)
	txsRequested := txs.RequestBlockTransactions(body)
	assert.Equal(t, 2, txsRequested)
}

func TestTransactionPreprocessor_RequestBlockTransactionFromMiniBlockFromNetwork(t *testing.T) {
	t.Parallel()
	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewTransactionPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)

	shardId := uint32(1)
	txHash1 := []byte("tx_hash1")
	txHash2 := []byte("tx_hash2")
	txHashes := make([][]byte, 0)
	txHashes = append(txHashes, txHash1)
	txHashes = append(txHashes, txHash2)
	mb := &block.MiniBlock{ReceiverShardID: shardId, TxHashes: txHashes}
	txsRequested := txs.RequestTransactionsForMiniBlock(mb)
	assert.Equal(t, 2, txsRequested)
}

func TestTransactionPreprocessor_ReceivedTransactionShouldEraseRequested(t *testing.T) {
	t.Parallel()

	dataPool := mock.NewPoolsHolderMock()

	shardedDataStub := &mock.ShardedDataStub{
		ShardDataStoreCalled: func(cacheId string) (c storage.Cacher) {
			return &mock.CacherStub{
				PeekCalled: func(key []byte) (value interface{}, ok bool) {
					return &transaction.Transaction{}, true
				},
			}
		},
		RegisterHandlerCalled: func(i func(key []byte)) {
		},
	}

	dataPool.SetTransactions(shardedDataStub)

	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewTransactionPreprocessor(
		dataPool.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)

	//add 3 tx hashes on requested list
	txHash1 := []byte("tx hash 1")
	txHash2 := []byte("tx hash 2")
	txHash3 := []byte("tx hash 3")

	txs.AddTxHashToRequestedList(txHash1)
	txs.AddTxHashToRequestedList(txHash2)
	txs.AddTxHashToRequestedList(txHash3)

	txs.SetMissingTxs(3)

	//received txHash2
	txs.ReceivedTransaction(txHash2)

	assert.True(t, txs.IsTxHashRequested(txHash1))
	assert.False(t, txs.IsTxHashRequested(txHash2))
	assert.True(t, txs.IsTxHashRequested(txHash3))
}

//------- GetAllTxsFromMiniBlock

func computeHash(data interface{}, marshalizer marshal.Marshalizer, hasher hashing.Hasher) []byte {
	buff, _ := marshalizer.Marshal(data)
	return hasher.Compute(string(buff))
}

func TestTransactionPreprocessor_GetAllTxsFromMiniBlockShouldWork(t *testing.T) {
	t.Parallel()

	hasher := mock.HasherMock{}
	marshalizer := &mock.MarshalizerMock{}
	dataPool := mock.NewPoolsHolderMock()
	senderShardId := uint32(0)
	destinationShardId := uint32(1)

	transactions := []*transaction.Transaction{
		{Nonce: 1},
		{Nonce: 2},
		{Nonce: 3},
	}
	transactionsHashes := make([][]byte, len(transactions))

	//add defined transactions to sender-destination cacher
	for idx, tx := range transactions {
		transactionsHashes[idx] = computeHash(tx, marshalizer, hasher)

		dataPool.Transactions().AddData(
			transactionsHashes[idx],
			tx,
			process.ShardCacherIdentifier(senderShardId, destinationShardId),
		)
	}

	//add some random data
	txRandom := &transaction.Transaction{Nonce: 4}
	dataPool.Transactions().AddData(
		computeHash(txRandom, marshalizer, hasher),
		txRandom,
		process.ShardCacherIdentifier(3, 4),
	)

	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewTransactionPreprocessor(
		dataPool.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)

	mb := &block.MiniBlock{
		SenderShardID:   senderShardId,
		ReceiverShardID: destinationShardId,
		TxHashes:        transactionsHashes,
	}

	txsRetrieved, txHashesRetrieved, err := txs.getAllTxsFromMiniBlock(mb, func() bool { return true })

	assert.Nil(t, err)
	assert.Equal(t, len(transactions), len(txsRetrieved))
	assert.Equal(t, len(transactions), len(txHashesRetrieved))
	for idx, tx := range transactions {
		//txReceived should be all txs in the same order
		assert.Equal(t, txsRetrieved[idx], tx)
		//verify corresponding transaction hashes
		assert.Equal(t, txHashesRetrieved[idx], computeHash(tx, marshalizer, hasher))
	}
}

func TestTransactionPreprocessor_RemoveBlockTxsFromPoolNilBlockShouldErr(t *testing.T) {
	t.Parallel()
	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewTransactionPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)
	err := txs.RemoveTxBlockFromPools(nil, tdp.MiniBlocks())
	assert.NotNil(t, err)
	assert.Equal(t, err, process.ErrNilTxBlockBody)
}

func TestTransactionPreprocessor_RemoveBlockTxsFromPoolOK(t *testing.T) {
	t.Parallel()
	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewTransactionPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)
	body := make(block.Body, 0)
	txHash := []byte("txHash")
	txHashes := make([][]byte, 0)
	txHashes = append(txHashes, txHash)
	miniblock := block.MiniBlock{
		ReceiverShardID: 0,
		SenderShardID:   0,
		TxHashes:        txHashes,
	}
	body = append(body, &miniblock)
	err := txs.RemoveTxBlockFromPools(body, tdp.MiniBlocks())
	assert.Nil(t, err)
}

func TestTransactions_CreateAndProcessMiniBlockCrossShardGasLimitAddAll(t *testing.T) {
	t.Parallel()

	txPool, _ := txpool.NewShardedTxPool(storageUnit.CacheConfig{Size: 100000, Shards: 1})
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	hasher := &mock.HasherMock{}
	marshalizer := &mock.MarshalizerMock{}

	totalGasConsumed := uint64(0)
	txs, _ := NewTransactionPreprocessor(
		txPool,
		&mock.ChainStorerMock{},
		hasher,
		marshalizer,
		&mock.TxProcessorMock{ProcessTransactionCalled: func(transaction *transaction.Transaction) error {
			return nil
		}},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{
			SetGasConsumedCalled: func(gasConsumed uint64, hash []byte) {
				totalGasConsumed += gasConsumed
			},
			TotalGasConsumedCalled: func() uint64 {
				return totalGasConsumed
			},
			ComputeGasConsumedByTxCalled: func(txSenderShardId uint32, txReceiverShardId uint32, txHandler data.TransactionHandler) (uint64, uint64, error) {
				return 0, 0, nil
			},
			SetGasRefundedCalled: func(gasRefunded uint64, hash []byte) {},
			TotalGasRefundedCalled: func() uint64 {
				return 0
			},
		},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)
	assert.NotNil(t, txs)

	sndShardId := uint32(0)
	dstShardId := uint32(1)
	strCache := process.ShardCacherIdentifier(sndShardId, dstShardId)

	addedTxs := make([]*transaction.Transaction, 0)
	for i := 0; i < 10; i++ {
		newTx := &transaction.Transaction{GasLimit: uint64(i)}

		txHash, _ := core.CalculateHash(marshalizer, hasher, newTx)
		txPool.AddData(txHash, newTx, strCache)

		addedTxs = append(addedTxs, newTx)
	}

	mb, err := txs.createAndProcessMiniBlock(sndShardId, dstShardId, process.MaxItemsInBlock, haveTimeTrue)
	assert.Nil(t, err)

	assert.Equal(t, len(addedTxs), len(mb.TxHashes))
}

func TestTransactions_CreateAndProcessMiniBlockCrossShardGasLimitAddAllAsNoSCCalls(t *testing.T) {
	t.Parallel()

	txPool, _ := txpool.NewShardedTxPool(storageUnit.CacheConfig{Size: 100000, Shards: 1})
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	hasher := &mock.HasherMock{}
	marshalizer := &mock.MarshalizerMock{}

	totalGasConsumed := uint64(0)
	txs, _ := NewTransactionPreprocessor(
		txPool,
		&mock.ChainStorerMock{},
		hasher,
		marshalizer,
		&mock.TxProcessorMock{ProcessTransactionCalled: func(transaction *transaction.Transaction) error {
			return nil
		}},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{
			SetGasConsumedCalled: func(gasConsumed uint64, hash []byte) {
				totalGasConsumed += gasConsumed
			},
			TotalGasConsumedCalled: func() uint64 {
				return totalGasConsumed
			},
			ComputeGasConsumedByTxCalled: func(txSenderShardId uint32, txReceiverShardId uint32, txHandler data.TransactionHandler) (uint64, uint64, error) {
				return 0, 0, nil
			},
			SetGasRefundedCalled: func(gasRefunded uint64, hash []byte) {},
			TotalGasRefundedCalled: func() uint64 {
				return 0
			},
		},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)
	assert.NotNil(t, txs)

	sndShardId := uint32(0)
	dstShardId := uint32(1)
	strCache := process.ShardCacherIdentifier(sndShardId, dstShardId)

	gasLimit := MaxGasLimitPerBlock / uint64(5)

	addedTxs := make([]*transaction.Transaction, 0)
	for i := 0; i < 10; i++ {
		newTx := &transaction.Transaction{GasLimit: gasLimit, GasPrice: uint64(i), RcvAddr: []byte("012345678910")}

		txHash, _ := core.CalculateHash(marshalizer, hasher, newTx)
		txPool.AddData(txHash, newTx, strCache)

		addedTxs = append(addedTxs, newTx)
	}

	mb, err := txs.createAndProcessMiniBlock(sndShardId, dstShardId, process.MaxItemsInBlock, haveTimeTrue)
	assert.Nil(t, err)

	assert.Equal(t, len(addedTxs), len(mb.TxHashes))
}

func TestTransactions_CreateAndProcessMiniBlockCrossShardGasLimitAddOnly5asSCCall(t *testing.T) {
	t.Parallel()

	txPool, _ := txpool.NewShardedTxPool(storageUnit.CacheConfig{Size: 100000, Shards: 1})
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	hasher := &mock.HasherMock{}
	marshalizer := &mock.MarshalizerMock{}

	numTxsToAdd := 5
	gasLimit := MaxGasLimitPerBlock / uint64(numTxsToAdd)

	totalGasConsumed := uint64(0)
	txs, _ := NewTransactionPreprocessor(
		txPool,
		&mock.ChainStorerMock{},
		hasher,
		marshalizer,
		&mock.TxProcessorMock{ProcessTransactionCalled: func(transaction *transaction.Transaction) error {
			return nil
		}},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{
			SetGasConsumedCalled: func(gasConsumed uint64, hash []byte) {
				totalGasConsumed += gasConsumed
			},
			ComputeGasConsumedByTxCalled: func(txSenderShardId uint32, txReceiverShardId uint32, txHandler data.TransactionHandler) (uint64, uint64, error) {
				return gasLimit, gasLimit, nil
			},
			TotalGasConsumedCalled: func() uint64 {
				return totalGasConsumed
			},
			SetGasRefundedCalled: func(gasRefunded uint64, hash []byte) {},
			GasRefundedCalled: func(hash []byte) uint64 {
				return 0
			},
			RemoveGasConsumedCalled: func(hashes [][]byte) {
				totalGasConsumed = 0
			},
			RemoveGasRefundedCalled: func(hashes [][]byte) {
			},
		},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)
	assert.NotNil(t, txs)

	sndShardId := uint32(0)
	dstShardId := uint32(1)
	strCache := process.ShardCacherIdentifier(sndShardId, dstShardId)

	scAddress, _ := hex.DecodeString("000000000000000000005fed9c659422cd8429ce92f8973bba2a9fb51e0eb3a1")
	addedTxs := make([]*transaction.Transaction, 0)
	for i := 0; i < 10; i++ {
		newTx := &transaction.Transaction{GasLimit: gasLimit, GasPrice: uint64(i), RcvAddr: scAddress}

		txHash, _ := core.CalculateHash(marshalizer, hasher, newTx)
		txPool.AddData(txHash, newTx, strCache)

		addedTxs = append(addedTxs, newTx)
	}

	mb, err := txs.createAndProcessMiniBlock(sndShardId, dstShardId, process.MaxItemsInBlock, haveTimeTrue)
	assert.Nil(t, err)

	assert.Equal(t, numTxsToAdd, len(mb.TxHashes))
}

func TestMiniBlocksCompaction_CompactAndExpandMiniBlocksShouldResultTheSameMiniBlocks(t *testing.T) {
	t.Parallel()

	totalGasConsumed := uint64(0)
	txPool, _ := txpool.NewShardedTxPool(storageUnit.CacheConfig{Size: 100000, Shards: 1})
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewTransactionPreprocessor(
		txPool,
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{
			ProcessTransactionCalled: func(transaction *transaction.Transaction) error {
				return nil
			},
		},
		mock.NewMultiShardsCoordinatorMock(2),
		&mock.AccountsStub{},
		requestTransaction,
		feeHandlerMock(),
		miniBlocksCompacterMock(),
		&mock.GasHandlerMock{
			InitCalled: func() {
				totalGasConsumed = 0
			},
			SetGasConsumedCalled: func(gasConsumed uint64, hash []byte) {
				totalGasConsumed += gasConsumed
			},
			ComputeGasConsumedByTxCalled: func(txSenderShardId uint32, txReceiverSharedId uint32, txHandler data.TransactionHandler) (uint64, uint64, error) {
				return 0, 0, nil
			},
			TotalGasConsumedCalled: func() uint64 {
				return totalGasConsumed
			},
			SetGasRefundedCalled: func(gasRefunded uint64, hash []byte) {},
			GasRefundedCalled: func(hash []byte) uint64 {
				return 0
			},
		},
		&mock.BlockTrackerMock{},
		block.TxBlock,
	)

	keygen := signing.NewKeyGenerator(kyber.NewBlakeSHA256Ed25519())
	_, accPk := keygen.GeneratePair()
	pkBytes, _ := accPk.ToByteArray()

	strCache00 := process.ShardCacherIdentifier(0, 0)
	strCache01 := process.ShardCacherIdentifier(0, 1)

	txHashesInMb1 := [][]byte{[]byte("tx00"), []byte("tx01"), []byte("tx02")}
	txHashesInMb2 := [][]byte{[]byte("tx10"), []byte("tx11"), []byte("tx12")}
	txHashesInMb3 := [][]byte{[]byte("tx20"), []byte("tx21"), []byte("tx22")}
	txHashesInMb4 := [][]byte{[]byte("tx30"), []byte("tx31"), []byte("tx32")}

	mapHashesAndTxs := map[string]data.TransactionHandler{
		string(txHashesInMb1[0]): &transaction.Transaction{Nonce: 0, SndAddr: pkBytes},
		string(txHashesInMb1[1]): &transaction.Transaction{Nonce: 1, SndAddr: pkBytes},
		string(txHashesInMb1[2]): &transaction.Transaction{Nonce: 2, SndAddr: pkBytes},
		string(txHashesInMb2[0]): &transaction.Transaction{Nonce: 3, SndAddr: pkBytes},
		string(txHashesInMb2[1]): &transaction.Transaction{Nonce: 4, SndAddr: pkBytes},
		string(txHashesInMb2[2]): &transaction.Transaction{Nonce: 5, SndAddr: pkBytes},
		string(txHashesInMb3[0]): &transaction.Transaction{Nonce: 6, SndAddr: pkBytes},
		string(txHashesInMb3[1]): &transaction.Transaction{Nonce: 7, SndAddr: pkBytes},
		string(txHashesInMb3[2]): &transaction.Transaction{Nonce: 8, SndAddr: pkBytes},
		string(txHashesInMb4[0]): &transaction.Transaction{Nonce: 9, SndAddr: pkBytes},
		string(txHashesInMb4[1]): &transaction.Transaction{Nonce: 10, SndAddr: pkBytes},
		string(txHashesInMb4[2]): &transaction.Transaction{Nonce: 11, SndAddr: pkBytes},
	}

	txPool.AddData(txHashesInMb1[0], mapHashesAndTxs[string(txHashesInMb1[0])], strCache00)
	txPool.AddData(txHashesInMb1[1], mapHashesAndTxs[string(txHashesInMb1[1])], strCache00)
	txPool.AddData(txHashesInMb1[2], mapHashesAndTxs[string(txHashesInMb1[2])], strCache00)
	mb1 := block.MiniBlock{
		TxHashes:        txHashesInMb1,
		ReceiverShardID: 0,
		SenderShardID:   0,
		Type:            0,
	}

	txPool.AddData(txHashesInMb2[0], mapHashesAndTxs[string(txHashesInMb2[0])], strCache01)
	txPool.AddData(txHashesInMb2[1], mapHashesAndTxs[string(txHashesInMb2[1])], strCache01)
	txPool.AddData(txHashesInMb2[2], mapHashesAndTxs[string(txHashesInMb2[2])], strCache01)
	mb2 := block.MiniBlock{
		TxHashes:        txHashesInMb2,
		ReceiverShardID: 1,
		SenderShardID:   0,
		Type:            0,
	}

	txPool.AddData(txHashesInMb3[0], mapHashesAndTxs[string(txHashesInMb3[0])], strCache00)
	txPool.AddData(txHashesInMb3[1], mapHashesAndTxs[string(txHashesInMb3[1])], strCache00)
	txPool.AddData(txHashesInMb3[2], mapHashesAndTxs[string(txHashesInMb3[2])], strCache00)
	mb3 := block.MiniBlock{
		TxHashes:        txHashesInMb3,
		ReceiverShardID: 0,
		SenderShardID:   0,
		Type:            0,
	}

	txPool.AddData(txHashesInMb4[0], mapHashesAndTxs[string(txHashesInMb4[0])], strCache01)
	txPool.AddData(txHashesInMb4[1], mapHashesAndTxs[string(txHashesInMb4[1])], strCache01)
	txPool.AddData(txHashesInMb4[2], mapHashesAndTxs[string(txHashesInMb4[2])], strCache01)
	mb4 := block.MiniBlock{
		TxHashes:        txHashesInMb4,
		ReceiverShardID: 1,
		SenderShardID:   0,
		Type:            0,
	}

	txs.gasHandler.Init()
	_ = txs.ProcessMiniBlock(&mb1, haveTimeTrue)
	txs.gasHandler.Init()
	_ = txs.ProcessMiniBlock(&mb2, haveTimeTrue)
	txs.gasHandler.Init()
	_ = txs.ProcessMiniBlock(&mb3, haveTimeTrue)
	txs.gasHandler.Init()
	_ = txs.ProcessMiniBlock(&mb4, haveTimeTrue)

	mbsOrig := block.MiniBlockSlice{}
	mbsOrig = append(mbsOrig, &mb1, &mb2, &mb3, &mb4)

	mbsValues := make([]block.MiniBlock, 0)
	for _, mb := range mbsOrig {
		mbsValues = append(mbsValues, *mb)
	}

	compactedMbs := txs.miniBlocksCompacter.Compact(mbsOrig, mapHashesAndTxs)
	expandedMbs, err := txs.miniBlocksCompacter.Expand(compactedMbs, mapHashesAndTxs)
	assert.Nil(t, err)

	assert.Equal(t, len(mbsValues), len(expandedMbs))
	for i := 0; i < len(mbsValues); i++ {
		assert.True(t, reflect.DeepEqual(mbsValues[i], *expandedMbs[i]))
	}
}
