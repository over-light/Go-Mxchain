package preprocess

import (
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/gin-gonic/gin/json"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"time"
)

func TestScrsPreprocessor_NewSmartContractResultPreprocessorNilPool(t *testing.T) {
	t.Parallel()

	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewSmartContractResultPreprocessor(
		nil,
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilUTxDataPool, err)
}

func TestScrsPreprocessor_NewSmartContractResultPreprocessorNilStore(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewSmartContractResultPreprocessor(
		tdp.Transactions(),
		nil,
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilUTxStorage, err)
}

func TestScrsPreprocessor_NewSmartContractResultPreprocessorNilHasher(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewSmartContractResultPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		nil,
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilHasher, err)
}

func TestScrsPreprocessor_NewSmartContractResultPreprocessorNilMarsalizer(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewSmartContractResultPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		nil,
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilMarshalizer, err)
}

func TestScrsPreprocessor_NewSmartContractResultPreprocessorNilTxProce(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewSmartContractResultPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		nil,
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilTxProcessor, err)
}

func TestScrsPreprocessor_NewSmartContractResultPreprocessorNilShardCoord(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewSmartContractResultPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		nil,
		&mock.AccountsStub{},
		requestTransaction,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilShardCoordinator, err)
}

func TestScrsPreprocessor_NewSmartContractResultPreprocessorNilAccounts(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, err := NewSmartContractResultPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		nil,
		requestTransaction,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilAccountsAdapter, err)
}

func TestScrsPreprocessor_NewSmartContractResultPreprocessorNilRequestFunc(t *testing.T) {
	t.Parallel()

	tdp := initDataPool()
	txs, err := NewSmartContractResultPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		nil,
	)

	assert.Nil(t, txs)
	assert.Equal(t, process.ErrNilRequestHandler, err)
}

func TestScrsPreProcessor_GetTransactionFromPool(t *testing.T) {
	t.Parallel()
	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewSmartContractResultPreprocessor(
		tdp.UnsignedTransactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
	)
	txHash := []byte("tx1_hash")
	tx := txs.getTransactionFromPool(1, 1, txHash, tdp.UnsignedTransactions())
	assert.NotNil(t, tx)
	assert.Equal(t, uint64(10), tx.(*smartContractResult.SmartContractResult).Nonce)
}

func TestScrsPreprocessor_RequestTransactionFromNetwork(t *testing.T) {
	t.Parallel()
	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewSmartContractResultPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
	)
	shardId := uint32(1)
	txHash1 := []byte("tx_hash1")
	txHash2 := []byte("tx_hash2")
	body := make(block.Body, 0)
	txHashes := make([][]byte, 0)
	txHashes = append(txHashes, txHash1)
	txHashes = append(txHashes, txHash2)
	mBlk := block.MiniBlock{ReceiverShardID: shardId, TxHashes: txHashes, Type: block.SmartContractResultBlock}
	body = append(body, &mBlk)
	txsRequested := txs.RequestBlockTransactions(body)
	assert.Equal(t, 2, txsRequested)
}

func TestScrsPreprocessor_RequestBlockTransactionFromMiniBlockFromNetwork(t *testing.T) {
	t.Parallel()
	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewSmartContractResultPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
	)

	shardId := uint32(1)
	txHash1 := []byte("tx_hash1")
	txHash2 := []byte("tx_hash2")
	txHashes := make([][]byte, 0)
	txHashes = append(txHashes, txHash1)
	txHashes = append(txHashes, txHash2)
	mb := block.MiniBlock{ReceiverShardID: shardId, TxHashes: txHashes, Type: block.SmartContractResultBlock}
	txsRequested := txs.RequestTransactionsForMiniBlock(mb)
	assert.Equal(t, 2, txsRequested)
}

func TestScrsPreprocessor_ReceivedTransactionShouldEraseRequested(t *testing.T) {
	t.Parallel()

	dataPool := mock.NewPoolsHolderFake()

	shardedDataStub := &mock.ShardedDataStub{
		ShardDataStoreCalled: func(cacheId string) (c storage.Cacher) {
			return &mock.CacherStub{
				PeekCalled: func(key []byte) (value interface{}, ok bool) {
					return &smartContractResult.SmartContractResult{}, true
				},
			}
		},
		RegisterHandlerCalled: func(i func(key []byte)) {
		},
	}

	dataPool.SetUnsignedTransactions(shardedDataStub)

	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewSmartContractResultPreprocessor(
		dataPool.UnsignedTransactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
	)

	//add 3 tx hashes on requested list
	txHash1 := []byte("tx hash 1")
	txHash2 := []byte("tx hash 2")
	txHash3 := []byte("tx hash 3")

	txs.AddScrHashToRequestedList(txHash1)
	txs.AddScrHashToRequestedList(txHash2)
	txs.AddScrHashToRequestedList(txHash3)

	txs.SetMissingScr(3)

	//received txHash2
	txs.receivedSmartContractResult(txHash2)

	assert.True(t, txs.IsScrHashRequested(txHash1))
	assert.False(t, txs.IsScrHashRequested(txHash2))
	assert.True(t, txs.IsScrHashRequested(txHash3))
}

func TestScrsPreprocessor_GetAllTxsFromMiniBlockShouldWork(t *testing.T) {
	t.Parallel()

	hasher := mock.HasherMock{}
	marshalizer := &mock.MarshalizerMock{}
	dataPool := mock.NewPoolsHolderFake()
	senderShardId := uint32(0)
	destinationShardId := uint32(1)

	transactions := []*smartContractResult.SmartContractResult{
		{Nonce: 1},
		{Nonce: 2},
		{Nonce: 3},
	}
	transactionsHashes := make([][]byte, len(transactions))

	//add defined transactions to sender-destination cacher
	for idx, tx := range transactions {
		transactionsHashes[idx] = computeHash(tx, marshalizer, hasher)

		dataPool.UnsignedTransactions().AddData(
			transactionsHashes[idx],
			tx,
			process.ShardCacherIdentifier(senderShardId, destinationShardId),
		)
	}

	//add some random data
	txRandom := &smartContractResult.SmartContractResult{Nonce: 4}
	dataPool.UnsignedTransactions().AddData(
		computeHash(txRandom, marshalizer, hasher),
		txRandom,
		process.ShardCacherIdentifier(3, 4),
	)

	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewSmartContractResultPreprocessor(
		dataPool.UnsignedTransactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
	)

	mb := &block.MiniBlock{
		SenderShardID:   senderShardId,
		ReceiverShardID: destinationShardId,
		TxHashes:        transactionsHashes,
		Type:            block.SmartContractResultBlock,
	}

	txsRetrieved, txHashesRetrieved, err := txs.getAllScrsFromMiniBlock(mb, func() bool { return true })

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

func TestScrsPreprocessor_RemoveBlockTxsFromPoolNilBlockShouldErr(t *testing.T) {
	t.Parallel()
	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewSmartContractResultPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
	)
	err := txs.RemoveTxBlockFromPools(nil, tdp.MiniBlocks())
	assert.NotNil(t, err)
	assert.Equal(t, err, process.ErrNilTxBlockBody)
}

func TestScrsPreprocessor_RemoveBlockTxsFromPoolOK(t *testing.T) {
	t.Parallel()
	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewSmartContractResultPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
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

func TestScrsPreprocessor_IsDataPreparedErr(t *testing.T) {
	t.Parallel()
	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewSmartContractResultPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
	)
	err := txs.IsDataPrepared(1, func() time.Duration { return 1000 })
	assert.NotNil(t, err)
	assert.Equal(t, process.ErrTimeIsOut, err)
}

func TestScrsPreprocessor_IsDataPrepared(t *testing.T) {
	t.Parallel()
	txDataPool := mock.NewStardedDataMock(storageUnit.CacheConfig{Size: 10000, Type: storageUnit.LRUCache})
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}

	marshalizer := &mock.MarshalizerMock{}
	hasher := mock.HasherMock{}

	crtShardId := uint32(3)

	txs, _ := NewSmartContractResultPreprocessor(
		txDataPool,
		&mock.ChainStorerMock{},
		hasher,
		marshalizer,
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(crtShardId),
		&mock.AccountsStub{},
		requestTransaction,
	)

	body, hashes, scrs := createTestBodyAndTxs(hasher, marshalizer)
	//body, _, _ := createTestBodyAndTxs(hasher, marshalizer)
	txs.RequestBlockTransactions(body)

	//call before is data prepare
	for idx, hash := range hashes {
		txDataPool.AddData(hash, scrs[idx], process.ShardCacherIdentifier(crtShardId, body[0].ReceiverShardID))

	}
	time.Sleep(500 * time.Millisecond)
	err := txs.IsDataPrepared(len(scrs), func() time.Duration { return time.Second })
	assert.Nil(t, err)
}

func createTestBodyAndTxs(hasher hashing.Hasher, marshalizer marshal.Marshalizer) (block.Body, [][]byte, []*smartContractResult.SmartContractResult) {
	noOfScrs := 2

	scrs := make([]*smartContractResult.SmartContractResult, noOfScrs)
	hashes := make([][]byte, noOfScrs)

	miniblock := &block.MiniBlock{}

	for i := 0; i < noOfScrs; i++ {
		g
		scr, hash := createScResultAndHash(uint64(i), hasher, marshalizer)

		scrs[i] = scr
		hashes[i] = hash
		miniblock.TxHashes = append(miniblock.TxHashes, hash)
	}

	body := []*block.MiniBlock{miniblock}

	return body, hashes, scrs
}

func createScResultAndHash(nonce uint64, hasher hashing.Hasher, marshalizer marshal.Marshalizer) (*smartContractResult.SmartContractResult, []byte) {
	scr := &smartContractResult.SmartContractResult{
		Nonce: nonce,
	}
	hash, _ := core.CalculateHash(marshalizer, hasher, scr)

	return scr, hash
}

func TestScrsPreprocessor_SaveTxBlockToStorage(t *testing.T) {
	t.Parallel()
	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewSmartContractResultPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
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
	err := txs.SaveTxBlockToStorage(body)

	assert.Nil(t, err)

}

func TestScrsPreprocessor_SaveTxBlockToStorageErr(t *testing.T) {
	t.Parallel()
	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	txs, _ := NewSmartContractResultPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
	)
	body := make(block.Body, 0)
	txHash := []byte(nil)
	txHashes := make([][]byte, 0)
	txHashes = append(txHashes, txHash)
	miniblock := block.MiniBlock{
		ReceiverShardID: 0,
		SenderShardID:   0,
		TxHashes:        txHashes,
		Type:            block.SmartContractResultBlock,
	}

	body = append(body, &miniblock)
	err := txs.SaveTxBlockToStorage(body)

	assert.Equal(t, process.ErrMissingTransaction, err)
}

func TestScrsPreprocessor_ProcessBlockTransactions(t *testing.T) {
	t.Parallel()
	tdp := initDataPool()
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	scr, _ := NewSmartContractResultPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{
			ProcessSmartContractResultCalled: func(scr *smartContractResult.SmartContractResult) error {
				return nil
			},
		},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
	)
	body := make(block.Body, 0)
	txHash := []byte("txHash")
	txHashes := make([][]byte, 0)
	txHashes = append(txHashes, txHash)
	miniblock := block.MiniBlock{
		ReceiverShardID: 0,
		SenderShardID:   0,
		TxHashes:        txHashes,
		Type:            block.SmartContractResultBlock,
	}
	body = append(body, &miniblock)
	scr.AddScrHashToRequestedList([]byte("txHash"))
	txshardInfo := txShardInfo{0, 0}
	smartcr := smartContractResult.SmartContractResult{
		Nonce: 1,
		Data:  "tx",
	}
	scr.scrForBlock.txHashAndInfo["txHash"] = &txInfo{&smartcr, &txshardInfo}
	err := scr.ProcessBlockTransactions(body, 1, func() time.Duration { return time.Second })
	assert.Nil(t, err)

}

func TestScrsPreprocessor_ProcessMiniBlock(t *testing.T) {
	t.Parallel()
	tdp := initDataPool()
	tdp.TransactionsCalled = func() dataRetriever.ShardedDataCacherNotifier {
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
				}
			},
		}
	}
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	scr, _ := NewSmartContractResultPreprocessor(
		tdp.Transactions(),
		&mock.ChainStorerMock{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{
			ProcessSmartContractResultCalled: func(scr *smartContractResult.SmartContractResult) error {
				return nil
			},
		},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
	)
	body := make(block.Body, 0)
	txHash := []byte("tx1_hash")
	txHashes := make([][]byte, 0)
	txHashes = append(txHashes, txHash)
	miniblock := block.MiniBlock{
		ReceiverShardID: 0,
		SenderShardID:   0,
		TxHashes:        txHashes,
		Type:            block.SmartContractResultBlock,
	}
	body = append(body, &miniblock)
	err := scr.ProcessMiniBlock(&miniblock, func() bool { return true }, 1)
	assert.Nil(t, err)

}

func TestScrs_Preprocess_RestoreTxBlockIntoPools(t *testing.T) {
	t.Parallel()
	txHash := []byte("txHash")
	scrstorage := mock.ChainStorerMock{}
	scrstorage.AddStorer(1, &mock.StorerStub{})
	scrstorage.Put(1, txHash, txHash)
	scrstorage.GetAllCalled = func(unitType dataRetriever.UnitType, keys [][]byte) (bytes map[string][]byte, e error) {
		par := make(map[string][]byte)
		tx := smartContractResult.SmartContractResult{}
		par["txHash"], _ = json.Marshal(tx)
		return par, nil
	}
	dataPool := mock.NewPoolsHolderFake()
	shardedDataStub := &mock.ShardedDataStub{
		AddDataCalled: func(key []byte, data interface{}, cacheId string) {
			return
		},
		RegisterHandlerCalled: func(i func(key []byte)) {
		},
	}
	dataPool.SetUnsignedTransactions(shardedDataStub)
	requestTransaction := func(shardID uint32, txHashes [][]byte) {}
	scr, _ := NewSmartContractResultPreprocessor(
		dataPool.UnsignedTransactions(),
		&scrstorage,
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.TxProcessorMock{},
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
		requestTransaction,
	)
	body := make(block.Body, 0)
	txHashes := make([][]byte, 0)
	txHashes = append(txHashes, txHash)
	miniblock := block.MiniBlock{
		ReceiverShardID: 0,
		SenderShardID:   0,
		TxHashes:        txHashes,
		Type:            block.SmartContractResultBlock,
	}
	body = append(body, &miniblock)
	miniblockPool := mock.NewCacherMock()
	scrRestored, _, err := scr.RestoreTxBlockIntoPools(body, miniblockPool)
	assert.Equal(t, scrRestored, 1)
	assert.Nil(t, err)
}
