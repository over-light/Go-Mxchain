package block_test

import (
	"bytes"
	"errors"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/blockchain"
	"github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	blproc "github.com/ElrondNetwork/elrond-go/process/block"
	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/memorydb"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/stretchr/testify/assert"
)

func haveTime() time.Duration {
	return 2000 * time.Millisecond
}

func createTestBlockchain() *mock.BlockChainMock {
	return &mock.BlockChainMock{GetGenesisHeaderCalled: func() data.HeaderHandler {
		return &block.Header{Nonce: 0}
	}}
}

func generateTestCache() storage.Cacher {
	cache, _ := storageUnit.NewCache(storageUnit.LRUCache, 1000, 1)
	return cache
}

func generateTestUnit() storage.Storer {
	storer, _ := storageUnit.NewStorageUnit(
		generateTestCache(),
		memorydb.New(),
	)

	return storer
}

func createShardedDataChacherNotifier(
	handler data.TransactionHandler,
	testHash []byte,
) func() dataRetriever.ShardedDataCacherNotifier {
	return func() dataRetriever.ShardedDataCacherNotifier {
		return &mock.ShardedDataStub{
			RegisterHandlerCalled: func(i func(key []byte, value interface{})) {},
			ShardDataStoreCalled: func(id string) (c storage.Cacher) {
				return &mock.CacherStub{
					PeekCalled: func(key []byte) (value interface{}, ok bool) {
						if reflect.DeepEqual(key, testHash) {
							return handler, true
						}
						return nil, false
					},
					KeysCalled: func() [][]byte {
						return [][]byte{[]byte("key1"), []byte("key2")}
					},
					LenCalled: func() int {
						return 0
					},
					MaxSizeCalled: func() int {
						return 1000
					},
				}
			},
			RemoveSetOfDataFromPoolCalled: func(keys [][]byte, id string) {},
			SearchFirstDataCalled: func(key []byte) (value interface{}, ok bool) {
				if reflect.DeepEqual(key, []byte("tx1_hash")) {
					return handler, true
				}
				return nil, false
			},
			AddDataCalled: func(key []byte, data interface{}, cacheId string) {
			},
		}
	}
}

func initDataPool(testHash []byte) *mock.PoolsHolderStub {
	rwdTx := &rewardTx.RewardTx{
		Round:   1,
		Epoch:   0,
		Value:   big.NewInt(10),
		RcvAddr: []byte("receiver"),
	}
	txCalled := createShardedDataChacherNotifier(&transaction.Transaction{Nonce: 10}, testHash)
	unsignedTxCalled := createShardedDataChacherNotifier(&transaction.Transaction{Nonce: 10}, testHash)
	rewardTransactionsCalled := createShardedDataChacherNotifier(rwdTx, testHash)

	sdp := &mock.PoolsHolderStub{
		TransactionsCalled:         txCalled,
		UnsignedTransactionsCalled: unsignedTxCalled,
		RewardTransactionsCalled:   rewardTransactionsCalled,
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
				MaxSizeCalled: func() int {
					return 1000
				},
				PeekCalled: func(key []byte) (value interface{}, ok bool) {
					if reflect.DeepEqual(key, []byte("tx1_hash")) {
						return &transaction.Transaction{Nonce: 10}, true
					}
					return nil, false
				},
				RegisterHandlerCalled: func(i func(key []byte, value interface{})) {},
				RemoveCalled:          func(key []byte) {},
			}
		},
		MiniBlocksCalled: func() storage.Cacher {
			cs := &mock.CacherStub{}
			cs.RegisterHandlerCalled = func(i func(key []byte, value interface{})) {
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
			cs.RegisterHandlerCalled = func(i func(key []byte, value interface{})) {}
			cs.RemoveCalled = func(key []byte) {}
			cs.LenCalled = func() int {
				return 0
			}
			cs.MaxSizeCalled = func() int {
				return 300
			}
			cs.KeysCalled = func() [][]byte {
				return nil
			}
			return cs
		},
		HeadersCalled: func() dataRetriever.HeadersPool {
			cs := &mock.HeadersCacherStub{}
			cs.RegisterHandlerCalled = func(i func(header data.HeaderHandler, key []byte)) {
			}
			cs.GetHeaderByHashCalled = func(hash []byte) (data.HeaderHandler, error) {
				return nil, process.ErrMissingHeader
			}
			cs.RemoveHeaderByHashCalled = func(key []byte) {
			}
			cs.LenCalled = func() int {
				return 0
			}
			cs.MaxSizeCalled = func() int {
				return 1000
			}
			cs.NoncesCalled = func(shardId uint32) []uint64 {
				return nil
			}
			return cs
		},
	}

	return sdp
}

func initStore() *dataRetriever.ChainStorer {
	store := dataRetriever.NewChainStorer()
	store.AddStorer(dataRetriever.TransactionUnit, generateTestUnit())
	store.AddStorer(dataRetriever.MiniBlockUnit, generateTestUnit())
	store.AddStorer(dataRetriever.RewardTransactionUnit, generateTestUnit())
	store.AddStorer(dataRetriever.MetaBlockUnit, generateTestUnit())
	store.AddStorer(dataRetriever.PeerChangesUnit, generateTestUnit())
	store.AddStorer(dataRetriever.BlockHeaderUnit, generateTestUnit())
	store.AddStorer(dataRetriever.ShardHdrNonceHashDataUnit, generateTestUnit())
	store.AddStorer(dataRetriever.MetaHdrNonceHashDataUnit, generateTestUnit())
	return store
}

func createDummyMetaBlock(destShardId uint32, senderShardId uint32, miniBlockHashes ...[]byte) *block.MetaBlock {
	metaBlock := &block.MetaBlock{
		ShardInfo: []block.ShardData{
			{
				ShardID:               senderShardId,
				ShardMiniBlockHeaders: make([]block.MiniBlockHeader, len(miniBlockHashes)),
			},
		},
	}

	for idx, mbHash := range miniBlockHashes {
		metaBlock.ShardInfo[0].ShardMiniBlockHeaders[idx].ReceiverShardID = destShardId
		metaBlock.ShardInfo[0].ShardMiniBlockHeaders[idx].SenderShardID = senderShardId
		metaBlock.ShardInfo[0].ShardMiniBlockHeaders[idx].Hash = mbHash
	}

	return metaBlock
}

func createDummyMiniBlock(
	txHash string,
	marshalizer marshal.Marshalizer,
	hasher hashing.Hasher,
	destShardId uint32,
	senderShardId uint32) (*block.MiniBlock, []byte) {

	miniblock := &block.MiniBlock{
		TxHashes:        [][]byte{[]byte(txHash)},
		ReceiverShardID: destShardId,
		SenderShardID:   senderShardId,
	}

	buff, _ := marshalizer.Marshal(miniblock)
	hash := hasher.Compute(string(buff))

	return miniblock, hash
}

func isInTxHashes(searched []byte, list [][]byte) bool {
	for _, txHash := range list {
		if bytes.Equal(txHash, searched) {
			return true
		}
	}
	return false
}

type wrongBody struct {
}

func (wr *wrongBody) Clone() data.BodyHandler {
	wrCopy := *wr

	return &wrCopy
}

func (wr *wrongBody) IntegrityAndValidity() error {
	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (wr *wrongBody) IsInterfaceNil() bool {
	return wr == nil
}

func createComponentHolderMocks() (*mock.CoreComponentsMock, *mock.DataComponentsMock) {
	blkc := blockchain.NewBlockChain()
	_ = blkc.SetGenesisHeader(&block.Header{Nonce: 0})

	coreComponents := &mock.CoreComponentsMock{
		IntMarsh:            &mock.MarshalizerMock{},
		Hash:                &mock.HasherStub{},
		UInt64ByteSliceConv: &mock.Uint64ByteSliceConverterMock{},
	}

	dataComponents := &mock.DataComponentsMock{
		Storage:    initStore(),
		DataPool:   initDataPool([]byte("")),
		BlockChain: blkc,
	}
	return coreComponents, dataComponents
}

func CreateMockArguments(
	coreComponents *mock.CoreComponentsMock,
	dataComponents *mock.DataComponentsMock,
) blproc.ArgShardProcessor {
	nodesCoordinator := mock.NewNodesCoordinatorMock()
	shardCoordinator := mock.NewOneShardCoordinatorMock()
	argsHeaderValidator := blproc.ArgsHeaderValidator{
		Hasher:      &mock.HasherMock{},
		Marshalizer: &mock.MarshalizerMock{},
	}
	headerValidator, _ := blproc.NewHeaderValidator(argsHeaderValidator)

	startHeaders := createGenesisBlocks(mock.NewOneShardCoordinatorMock())

	accountsDb := make(map[state.AccountsDbIdentifier]state.AccountsAdapter)
	accountsDb[state.UserAccountsState] = &mock.AccountsStub{}

	arguments := blproc.ArgShardProcessor{
		ArgBaseProcessor: blproc.ArgBaseProcessor{
			CoreComponents:    coreComponents,
			DataComponents:    dataComponents,
			AccountsDB:        accountsDb,
			ForkDetector:      &mock.ForkDetectorMock{},
			ShardCoordinator:  shardCoordinator,
			NodesCoordinator:  nodesCoordinator,
			FeeHandler:        &mock.FeeAccumulatorStub{},
			RequestHandler:    &mock.RequestHandlerStub{},
			Core:              &mock.ServiceContainerMock{},
			BlockChainHook:    &mock.BlockChainHookHandlerMock{},
			TxCoordinator:     &mock.TransactionCoordinatorMock{},
			EpochStartTrigger: &mock.EpochStartTriggerStub{},
			HeaderValidator:   headerValidator,
			Rounder:           &mock.RounderMock{},
			BootStorer: &mock.BoostrapStorerMock{
				PutCalled: func(round int64, bootData bootstrapStorage.BootstrapData) error {
					return nil
				},
			},
			BlockTracker:       mock.NewBlockTrackerMock(shardCoordinator, startHeaders),
			BlockSizeThrottler: &mock.BlockSizeThrottlerStub{},
			Version:            "softwareVersion",
		},
	}

	return arguments
}

func TestBlockProcessor_CheckBlockValidity(t *testing.T) {
	t.Parallel()

	coreComponents, dataComponents := createComponentHolderMocks()
	coreComponents.Hash = &mock.HasherMock{}
	blkc := createTestBlockchain()
	dataComponents.BlockChain = blkc
	arguments := CreateMockArguments(coreComponents, dataComponents)
	bp, _ := blproc.NewShardProcessor(arguments)

	body := &block.Body{}
	hdr := &block.Header{}
	hdr.Nonce = 1
	hdr.Round = 1
	hdr.TimeStamp = 0
	hdr.PrevHash = []byte("X")
	err := bp.CheckBlockValidity(hdr, body)
	assert.Equal(t, process.ErrBlockHashDoesNotMatch, err)

	hdr.PrevHash = []byte("")
	err = bp.CheckBlockValidity(hdr, body)
	assert.Nil(t, err)

	hdr.Nonce = 2
	err = bp.CheckBlockValidity(hdr, body)
	assert.Equal(t, process.ErrWrongNonceInBlock, err)

	blkc.GetCurrentBlockHeaderCalled = func() data.HeaderHandler {
		return &block.Header{Round: 1, Nonce: 1}
	}
	prevHash := []byte("X")
	blkc.GetCurrentBlockHeaderHashCalled = func() []byte {
		return prevHash
	}
	hdr = &block.Header{}

	err = bp.CheckBlockValidity(hdr, body)
	assert.Equal(t, process.ErrLowerRoundInBlock, err)

	hdr.Round = 2
	hdr.Nonce = 1
	err = bp.CheckBlockValidity(hdr, body)
	assert.Equal(t, process.ErrWrongNonceInBlock, err)

	hdr.Nonce = 2
	hdr.PrevHash = []byte("XX")
	err = bp.CheckBlockValidity(hdr, body)
	assert.Equal(t, process.ErrBlockHashDoesNotMatch, err)

	hdr.PrevHash = blkc.GetCurrentBlockHeaderHash()
	hdr.PrevRandSeed = []byte("X")
	err = bp.CheckBlockValidity(hdr, body)
	assert.Equal(t, process.ErrRandSeedDoesNotMatch, err)

	hdr.PrevRandSeed = []byte("")
	err = bp.CheckBlockValidity(hdr, body)
	assert.Nil(t, err)
}

func TestVerifyStateRoot_ShouldWork(t *testing.T) {
	t.Parallel()
	rootHash := []byte("root hash to be tested")
	accounts := &mock.AccountsStub{
		RootHashCalled: func() ([]byte, error) {
			return rootHash, nil
		},
	}

	coreComponents, dataComponents := createComponentHolderMocks()
	arguments := CreateMockArguments(coreComponents, dataComponents)
	arguments.AccountsDB[state.UserAccountsState] = accounts
	bp, _ := blproc.NewShardProcessor(arguments)

	assert.True(t, bp.VerifyStateRoot(rootHash))
}

//------- SetAppStatusHandler
func TestBaseProcessor_SetAppStatusHandlerNilHandlerShouldErr(t *testing.T) {
	t.Parallel()

	coreComponents, dataComponents := createComponentHolderMocks()
	arguments := CreateMockArguments(coreComponents, dataComponents)
	bp, _ := blproc.NewShardProcessor(arguments)

	err := bp.SetAppStatusHandler(nil)
	assert.Equal(t, process.ErrNilAppStatusHandler, err)
}

func TestBaseProcessor_SetAppStatusHandlerOkHandlerShouldWork(t *testing.T) {
	t.Parallel()

	coreComponents, dataComponents := createComponentHolderMocks()
	arguments := CreateMockArguments(coreComponents, dataComponents)
	bp, _ := blproc.NewShardProcessor(arguments)

	ash := &mock.AppStatusHandlerStub{}
	err := bp.SetAppStatusHandler(ash)
	assert.Nil(t, err)
}

//------- RevertState
func TestBaseProcessor_RevertStateRecreateTrieFailsShouldErr(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("err")
	coreComponents, dataComponents := createComponentHolderMocks()
	arguments := CreateMockArguments(coreComponents, dataComponents)
	arguments.AccountsDB[state.UserAccountsState] = &mock.AccountsStub{
		RecreateTrieCalled: func(rootHash []byte) error {
			return expectedErr
		},
	}

	bp, _ := blproc.NewShardProcessor(arguments)

	hdr := block.Header{Nonce: 37}
	err := bp.RevertStateToBlock(&hdr)
	assert.Equal(t, expectedErr, err)
}

// removeHeadersBehindNonceFromPools
func TestBaseProcessor_RemoveHeadersBehindNonceFromPools(t *testing.T) {
	t.Parallel()

	removeFromDataPoolWasCalled := false
	dataPool := initDataPool([]byte(""))
	dataPool.HeadersCalled = func() dataRetriever.HeadersPool {
		cs := &mock.HeadersCacherStub{}
		cs.RegisterHandlerCalled = func(i func(header data.HeaderHandler, key []byte)) {
		}
		cs.GetHeaderByHashCalled = func(hash []byte) (handler data.HeaderHandler, err error) {
			return nil, err
		}
		cs.RemoveHeaderByHashCalled = func(key []byte) {
		}
		cs.LenCalled = func() int {
			return 0
		}
		cs.MaxSizeCalled = func() int {
			return 1000
		}
		cs.NoncesCalled = func(shardId uint32) []uint64 {
			return []uint64{1, 2, 3}
		}
		cs.GetHeaderByNonceAndShardIdCalled = func(hdrNonce uint64, shardId uint32) ([]data.HeaderHandler, [][]byte, error) {
			hdrs := make([]data.HeaderHandler, 0)
			hdrs = append(hdrs, &block.Header{Nonce: 2})
			return hdrs, nil, nil
		}

		return cs
	}

	coreComponents, dataComponents := createComponentHolderMocks()
	dataComponents.DataPool = dataPool
	arguments := CreateMockArguments(coreComponents, dataComponents)
	arguments.TxCoordinator = &mock.TransactionCoordinatorMock{
		RemoveBlockDataFromPoolCalled: func(body *block.Body) error {
			removeFromDataPoolWasCalled = true
			return nil
		},
	}
	bp, _ := blproc.NewShardProcessor(arguments)

	bp.RemoveHeadersBehindNonceFromPools(true, 0, 4)

	assert.True(t, removeFromDataPoolWasCalled)
}

//------- ComputeNewNoncePrevHash

func TestBlockProcessor_computeHeaderHashMarshalizerFail1ShouldErr(t *testing.T) {
	t.Parallel()
	marshalizer := &mock.MarshalizerStub{}

	coreComponents, dataComponents := createComponentHolderMocks()
	coreComponents.IntMarsh = marshalizer
	arguments := CreateMockArguments(coreComponents, dataComponents)
	bp, _ := blproc.NewShardProcessor(arguments)
	hdr, txBlock := createTestHdrTxBlockBody()
	expectedError := errors.New("marshalizer fail")
	marshalizer.MarshalCalled = func(obj interface{}) (bytes []byte, e error) {
		if hdr == obj {
			return nil, expectedError
		}

		if reflect.DeepEqual(txBlock, obj) {
			return []byte("txBlockBodyMarshalized"), nil
		}
		return nil, nil
	}
	_, err := bp.ComputeHeaderHash(hdr)
	assert.Equal(t, expectedError, err)
}

func TestBlockPorcessor_ComputeNewNoncePrevHashShouldWork(t *testing.T) {
	t.Parallel()
	marshalizer := &mock.MarshalizerStub{}
	hasher := &mock.HasherStub{}

	coreComponents, dataComponents := createComponentHolderMocks()
	coreComponents.IntMarsh = marshalizer
	coreComponents.Hash = hasher
	arguments := CreateMockArguments(coreComponents, dataComponents)
	bp, _ := blproc.NewShardProcessor(arguments)
	hdr, txBlock := createTestHdrTxBlockBody()
	marshalizer.MarshalCalled = func(obj interface{}) (bytes []byte, e error) {
		if hdr == obj {
			return []byte("hdrHeaderMarshalized"), nil
		}
		if reflect.DeepEqual(txBlock, obj) {
			return []byte("txBlockBodyMarshalized"), nil
		}
		return nil, nil
	}
	hasher.ComputeCalled = func(s string) []byte {
		if s == "hdrHeaderMarshalized" {
			return []byte("hdr hash")
		}
		if s == "txBlockBodyMarshalized" {
			return []byte("tx block body hash")
		}
		return nil
	}
	_, err := bp.ComputeHeaderHash(hdr)
	assert.Nil(t, err)
}

func createShardProcessHeadersToSaveLastNotarized(
	highestNonce uint64,
	genesisHdr data.HeaderHandler,
	hasher hashing.Hasher,
	marshalizer marshal.Marshalizer,
) []data.HeaderHandler {
	rootHash := []byte("roothash")
	processedHdrs := make([]data.HeaderHandler, 0)

	headerMarsh, _ := marshalizer.Marshal(genesisHdr)
	headerHash := hasher.Compute(string(headerMarsh))

	for i := uint64(1); i <= highestNonce; i++ {
		hdr := &block.Header{
			Nonce:         i,
			Round:         i,
			Signature:     rootHash,
			RandSeed:      rootHash,
			PrevRandSeed:  rootHash,
			PubKeysBitmap: rootHash,
			RootHash:      rootHash,
			PrevHash:      headerHash}
		processedHdrs = append(processedHdrs, hdr)

		headerMarsh, _ = marshalizer.Marshal(hdr)
		headerHash = hasher.Compute(string(headerMarsh))
	}

	return processedHdrs
}

func createMetaProcessHeadersToSaveLastNoterized(
	highestNonce uint64,
	genesisHdr data.HeaderHandler,
	hasher hashing.Hasher,
	marshalizer marshal.Marshalizer,
) []data.HeaderHandler {
	rootHash := []byte("roothash")
	processedHdrs := make([]data.HeaderHandler, 0)

	headerMarsh, _ := marshalizer.Marshal(genesisHdr)
	headerHash := hasher.Compute(string(headerMarsh))

	for i := uint64(1); i <= highestNonce; i++ {
		hdr := &block.MetaBlock{
			Nonce:         i,
			Round:         i,
			Signature:     rootHash,
			RandSeed:      rootHash,
			PrevRandSeed:  rootHash,
			PubKeysBitmap: rootHash,
			RootHash:      rootHash,
			PrevHash:      headerHash}
		processedHdrs = append(processedHdrs, hdr)

		headerMarsh, _ = marshalizer.Marshal(hdr)
		headerHash = hasher.Compute(string(headerMarsh))
	}

	return processedHdrs
}

func TestBaseProcessor_SaveLastNotarizedInOneShardHdrsSliceForShardIsNil(t *testing.T) {
	t.Parallel()

	coreComponents, dataComponents := createComponentHolderMocks()
	coreComponents.Hash = &mock.HasherMock{}
	coreComponents.IntMarsh = &mock.MarshalizerMock{}
	arguments := CreateMockArguments(coreComponents, dataComponents)
	sp, _ := blproc.NewShardProcessor(arguments)
	prHdrs := createShardProcessHeadersToSaveLastNotarized(10, &block.Header{}, mock.HasherMock{}, &mock.MarshalizerMock{})

	err := sp.SaveLastNotarizedHeader(2, prHdrs)

	assert.Equal(t, process.ErrNotarizedHeadersSliceForShardIsNil, err)
}

func TestBaseProcessor_SaveLastNotarizedInMultiShardHdrsSliceForShardIsNil(t *testing.T) {
	t.Parallel()

	shardCoordinator := mock.NewMultiShardsCoordinatorMock(5)
	coreComponents, dataComponents := createComponentHolderMocks()
	coreComponents.Hash = &mock.HasherMock{}
	coreComponents.IntMarsh = &mock.MarshalizerMock{}
	arguments := CreateMockArguments(coreComponents, dataComponents)
	arguments.ShardCoordinator = shardCoordinator
	sp, _ := blproc.NewShardProcessor(arguments)

	prHdrs := createShardProcessHeadersToSaveLastNotarized(10, &block.Header{}, mock.HasherMock{}, &mock.MarshalizerMock{})

	err := sp.SaveLastNotarizedHeader(6, prHdrs)

	assert.Equal(t, process.ErrNotarizedHeadersSliceForShardIsNil, err)
}

func TestBaseProcessor_SaveLastNotarizedHdrShardGood(t *testing.T) {
	t.Parallel()

	shardCoordinator := mock.NewMultiShardsCoordinatorMock(5)
	coreComponents, dataComponents := createComponentHolderMocks()
	coreComponents.Hash = &mock.HasherMock{}
	coreComponents.IntMarsh = &mock.MarshalizerMock{}
	arguments := CreateMockArguments(coreComponents, dataComponents)
	arguments.ShardCoordinator = shardCoordinator
	sp, _ := blproc.NewShardProcessor(arguments)
	argsHeaderValidator := blproc.ArgsHeaderValidator{
		Hasher:      coreComponents.Hasher(),
		Marshalizer: coreComponents.InternalMarshalizer(),
	}
	headerValidator, _ := blproc.NewHeaderValidator(argsHeaderValidator)
	sp.SetHeaderValidator(headerValidator)

	genesisBlcks := createGenesisBlocks(shardCoordinator)

	highestNonce := uint64(10)
	shardId := uint32(0)
	prHdrs := createShardProcessHeadersToSaveLastNotarized(
		highestNonce,
		genesisBlcks[shardId],
		coreComponents.Hasher(),
		coreComponents.InternalMarshalizer())

	err := sp.SaveLastNotarizedHeader(shardId, prHdrs)
	assert.Nil(t, err)

	assert.Equal(t, highestNonce, sp.LastNotarizedHdrForShard(shardId).GetNonce())
}

func TestBaseProcessor_SaveLastNotarizedHdrMetaGood(t *testing.T) {
	t.Parallel()

	shardCoordinator := mock.NewMultiShardsCoordinatorMock(5)
	coreComponents, dataComponents := createComponentHolderMocks()
	coreComponents.Hash = &mock.HasherMock{}
	coreComponents.IntMarsh = &mock.MarshalizerMock{}
	arguments := CreateMockArguments(coreComponents, dataComponents)
	arguments.ShardCoordinator = shardCoordinator
	sp, _ := blproc.NewShardProcessor(arguments)

	argsHeaderValidator := blproc.ArgsHeaderValidator{
		Hasher:      coreComponents.Hasher(),
		Marshalizer: coreComponents.InternalMarshalizer(),
	}
	headerValidator, _ := blproc.NewHeaderValidator(argsHeaderValidator)
	sp.SetHeaderValidator(headerValidator)

	genesisBlcks := createGenesisBlocks(shardCoordinator)

	highestNonce := uint64(10)
	prHdrs := createMetaProcessHeadersToSaveLastNoterized(
		highestNonce,
		genesisBlcks[core.MetachainShardId],
		coreComponents.Hasher(),
		coreComponents.InternalMarshalizer())

	err := sp.SaveLastNotarizedHeader(core.MetachainShardId, prHdrs)
	assert.Nil(t, err)

	assert.Equal(t, highestNonce, sp.LastNotarizedHdrForShard(core.MetachainShardId).GetNonce())
}

func TestShardProcessor_ProcessBlockEpochDoesNotMatchShouldErr(t *testing.T) {
	t.Parallel()
	blockChain := &mock.BlockChainMock{
		GetCurrentBlockHeaderCalled: func() data.HeaderHandler {
			return &block.Header{
				Epoch: 2,
			}
		},
		GetGenesisHeaderCalled: func() data.HeaderHandler {
			return &block.Header{Nonce: 0}
		},
	}
	coreComponents, dataComponents := CreateCoreComponentsMultiShard()
	dataComponents.BlockChain = blockChain
	arguments := CreateMockArgumentsMultiShard(coreComponents, dataComponents)
	sp, _ := blproc.NewShardProcessor(arguments)
	header := &block.Header{Round: 10, Nonce: 1}

	blk := &block.Body{}
	err := sp.ProcessBlock(header, blk, func() time.Duration { return time.Second })

	assert.True(t, errors.Is(err, process.ErrEpochDoesNotMatch))
}

func TestShardProcessor_ProcessBlockEpochDoesNotMatchShouldErr2(t *testing.T) {
	t.Parallel()

	randSeed := []byte("randseed")
	blockChain := &mock.BlockChainMock{
		GetCurrentBlockHeaderCalled: func() data.HeaderHandler {
			return &block.Header{
				Epoch:           1,
				RandSeed:        randSeed,
				AccumulatedFees: big.NewInt(0),
				DeveloperFees:   big.NewInt(0),
			}
		},
		GetGenesisHeaderCalled: func() data.HeaderHandler {
			return &block.Header{Nonce: 0}
		},
	}

	coreComponents, dataComponents := CreateCoreComponentsMultiShard()
	dataComponents.BlockChain = blockChain
	arguments := CreateMockArgumentsMultiShard(coreComponents, dataComponents)
	arguments.EpochStartTrigger = &mock.EpochStartTriggerStub{
		EpochCalled: func() uint32 {
			return 1
		},
	}

	sp, _ := blproc.NewShardProcessor(arguments)
	header := &block.Header{Round: 10, Nonce: 1, Epoch: 5, RandSeed: randSeed, PrevRandSeed: randSeed}

	blk := &block.Body{}
	err := sp.ProcessBlock(header, blk, func() time.Duration { return time.Second })

	assert.True(t, errors.Is(err, process.ErrEpochDoesNotMatch))
}

func TestShardProcessor_ProcessBlockEpochDoesNotMatchShouldErr3(t *testing.T) {
	t.Parallel()

	randSeed := []byte("randseed")
	blockChain := &mock.BlockChainMock{
		GetCurrentBlockHeaderCalled: func() data.HeaderHandler {
			return &block.Header{
				Epoch:    3,
				RandSeed: randSeed,
			}
		},
		GetGenesisHeaderCalled: func() data.HeaderHandler {
			return &block.Header{Nonce: 0}
		},
	}

	coreComponents, dataComponents := CreateCoreComponentsMultiShard()
	dataComponents.BlockChain = blockChain
	arguments := CreateMockArgumentsMultiShard(coreComponents, dataComponents)
	arguments.EpochStartTrigger = &mock.EpochStartTriggerStub{
		EpochCalled: func() uint32 {
			return 2
		},
		IsEpochStartCalled: func() bool {
			return true
		},
	}

	sp, _ := blproc.NewShardProcessor(arguments)
	header := &block.Header{Round: 10, Nonce: 1, Epoch: 5, RandSeed: randSeed, PrevRandSeed: randSeed}

	blk := &block.Body{}
	err := sp.ProcessBlock(header, blk, func() time.Duration { return time.Second })

	assert.True(t, errors.Is(err, process.ErrEpochDoesNotMatch))
}

func TestShardProcessor_ProcessBlockEpochDoesNotMatchShouldErrMetaHashDoesNotMatch(t *testing.T) {
	t.Parallel()

	randSeed := []byte("randseed")
	chain := &mock.BlockChainMock{
		GetCurrentBlockHeaderCalled: func() data.HeaderHandler {
			return &block.Header{
				Epoch:    2,
				RandSeed: randSeed,
			}
		},
		GetGenesisHeaderCalled: func() data.HeaderHandler {
			return &block.Header{Nonce: 0}
		},
	}
	hasher := &mock.HasherStub{ComputeCalled: func(s string) []byte {
		return nil
	}}

	coreComponents, dataComponents := CreateCoreComponentsMultiShard()
	dataComponents.BlockChain = chain
	coreComponents.Hash = hasher
	arguments := CreateMockArgumentsMultiShard(coreComponents, dataComponents)

	epochStartTrigger := &mock.EpochStartTriggerStub{
		EpochCalled: func() uint32 {
			return 2
		},
		MetaEpochCalled: func() uint32 {
			return 3
		},
		IsEpochStartCalled: func() bool {
			return true
		},
		EpochFinalityAttestingRoundCalled: func() uint64 {
			return 100
		},
	}
	arguments.EpochStartTrigger = epochStartTrigger

	sp, _ := blproc.NewShardProcessor(arguments)
	rootHash, _ := arguments.AccountsDB[state.UserAccountsState].RootHash()
	epochStartHash := []byte("epochStartHash")
	header := &block.Header{
		Round:              10,
		Nonce:              1,
		Epoch:              3,
		RandSeed:           randSeed,
		PrevRandSeed:       randSeed,
		EpochStartMetaHash: epochStartHash,
		RootHash:           rootHash,
		AccumulatedFees:    big.NewInt(0),
		DeveloperFees:      big.NewInt(0),
	}

	blk := &block.Body{}
	err := sp.ProcessBlock(header, blk, func() time.Duration { return time.Second })
	assert.True(t, errors.Is(err, process.ErrEpochDoesNotMatch))

	epochStartTrigger.EpochStartMetaHdrHashCalled = func() []byte {
		return header.EpochStartMetaHash
	}
	err = sp.ProcessBlock(header, blk, func() time.Duration { return time.Second })
	assert.Nil(t, err)
}

func TestShardProcessor_ProcessBlockEpochDoesNotMatchShouldErrMetaHashDoesNotMatchForOldEpoch(t *testing.T) {
	t.Parallel()

	randSeed := []byte("randseed")
	chain := &mock.BlockChainMock{
		GetCurrentBlockHeaderCalled: func() data.HeaderHandler {
			return &block.Header{
				Epoch:    2,
				RandSeed: randSeed,
			}
		},
		GetGenesisHeaderCalled: func() data.HeaderHandler {
			return &block.Header{Nonce: 0}
		},
	}
	hasher := &mock.HasherStub{ComputeCalled: func(s string) []byte {
		return nil
	}}

	coreComponents, dataComponents := CreateCoreComponentsMultiShard()
	coreComponents.Hash = hasher
	dataComponents.BlockChain = chain
	arguments := CreateMockArgumentsMultiShard(coreComponents, dataComponents)

	arguments.EpochStartTrigger = &mock.EpochStartTriggerStub{
		EpochCalled: func() uint32 {
			return 5
		},
		MetaEpochCalled: func() uint32 {
			return 6
		},
		IsEpochStartCalled: func() bool {
			return true
		},
		EpochFinalityAttestingRoundCalled: func() uint64 {
			return 100
		},
	}

	sp, _ := blproc.NewShardProcessor(arguments)
	rootHash, _ := arguments.AccountsDB[state.UserAccountsState].RootHash()
	epochStartHash := []byte("epochStartHash")
	header := &block.Header{
		Round:              10,
		Nonce:              1,
		Epoch:              3,
		RandSeed:           randSeed,
		PrevRandSeed:       randSeed,
		EpochStartMetaHash: epochStartHash,
		RootHash:           rootHash,
		AccumulatedFees:    big.NewInt(0),
		DeveloperFees:      big.NewInt(0),
	}

	blk := &block.Body{}
	err := sp.ProcessBlock(header, blk, func() time.Duration { return time.Second })
	assert.True(t, errors.Is(err, process.ErrMissingHeader))

	metaHdr := &block.MetaBlock{}
	metaHdrData, _ := coreComponents.InternalMarshalizer().Marshal(metaHdr)
	_ = dataComponents.StorageService().Put(dataRetriever.MetaBlockUnit, header.EpochStartMetaHash, metaHdrData)

	err = sp.ProcessBlock(header, blk, func() time.Duration { return time.Second })
	assert.True(t, errors.Is(err, process.ErrEpochDoesNotMatch))

	metaHdr = &block.MetaBlock{Epoch: 3, EpochStart: block.EpochStart{
		LastFinalizedHeaders: []block.EpochStartShardData{{}},
		Economics:            block.Economics{},
	}}
	metaHdrData, _ = coreComponents.InternalMarshalizer().Marshal(metaHdr)
	_ = dataComponents.StorageService().Put(dataRetriever.MetaBlockUnit, header.EpochStartMetaHash, metaHdrData)

	err = sp.ProcessBlock(header, blk, func() time.Duration { return time.Second })
	assert.Nil(t, err)
}

func TestBlockProcessor_PruneStateOnRollbackPrunesPeerTrieIfAccPruneIsDisabled(t *testing.T) {
	t.Parallel()

	pruningCalled := 0
	peerAccDb := &mock.AccountsStub{
		PruneTrieCalled: func(rootHash []byte, identifier data.TriePruningIdentifier) {
			pruningCalled++
		},
		CancelPruneCalled: func(rootHash []byte, identifier data.TriePruningIdentifier) {
			pruningCalled++
		},
		IsPruningEnabledCalled: func() bool {
			return true
		},
	}

	coreComponents, dataComponents := createComponentHolderMocks()
	arguments := CreateMockArguments(coreComponents, dataComponents)
	arguments.AccountsDB[state.PeerAccountsState] = peerAccDb
	bp, _ := blproc.NewShardProcessor(arguments)

	prevHeader := &block.MetaBlock{
		RootHash:               []byte("prevRootHash"),
		ValidatorStatsRootHash: []byte("prevValidatorRootHash"),
	}
	currHeader := &block.MetaBlock{
		RootHash:               []byte("prevRootHash"),
		ValidatorStatsRootHash: []byte("currValidatorRootHash"),
	}

	bp.PruneStateOnRollback(currHeader, prevHeader)
	assert.Equal(t, 2, pruningCalled)
}

func TestBlockProcessor_PruneStateOnRollbackPrunesPeerTrieIfSameRootHashButDifferentValidatorRootHash(t *testing.T) {
	t.Parallel()

	pruningCalled := 0
	peerAccDb := &mock.AccountsStub{
		PruneTrieCalled: func(rootHash []byte, identifier data.TriePruningIdentifier) {
			pruningCalled++
		},
		CancelPruneCalled: func(rootHash []byte, identifier data.TriePruningIdentifier) {
			pruningCalled++
		},
		IsPruningEnabledCalled: func() bool {
			return true
		},
	}

	accDb := &mock.AccountsStub{
		PruneTrieCalled: func(rootHash []byte, identifier data.TriePruningIdentifier) {
			pruningCalled++
		},
		CancelPruneCalled: func(rootHash []byte, identifier data.TriePruningIdentifier) {
			pruningCalled++
		},
		IsPruningEnabledCalled: func() bool {
			return true
		},
	}

	coreComponents, dataComponents := createComponentHolderMocks()
	arguments := CreateMockArguments(coreComponents, dataComponents)
	arguments.AccountsDB[state.PeerAccountsState] = peerAccDb
	arguments.AccountsDB[state.UserAccountsState] = accDb
	bp, _ := blproc.NewShardProcessor(arguments)

	prevHeader := &block.MetaBlock{
		RootHash:               []byte("prevRootHash"),
		ValidatorStatsRootHash: []byte("prevValidatorRootHash"),
	}
	currHeader := &block.MetaBlock{
		RootHash:               []byte("prevRootHash"),
		ValidatorStatsRootHash: []byte("currValidatorRootHash"),
	}

	bp.PruneStateOnRollback(currHeader, prevHeader)
	assert.Equal(t, 2, pruningCalled)
}
