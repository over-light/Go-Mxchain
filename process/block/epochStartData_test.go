package block_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	blproc "github.com/ElrondNetwork/elrond-go/process/block"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/memorydb"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/stretchr/testify/assert"
)

func createMockEpochStartCreatorArguments() blproc.ArgsNewEpochStartDataCreator {
	shardCoordinator := mock.NewOneShardCoordinatorMock()
	startHeaders := createGenesisBlocks(shardCoordinator)
	argsNewEpochStartDataCreator := blproc.ArgsNewEpochStartDataCreator{
		Marshalizer:       &mock.MarshalizerMock{},
		Hasher:            &mock.HasherStub{},
		Store:             createMetaStore(),
		DataPool:          initMetaDataPool(),
		BlockTracker:      mock.NewBlockTrackerMock(shardCoordinator, startHeaders),
		ShardCoordinator:  shardCoordinator,
		EpochStartTrigger: &mock.EpochStartTriggerStub{},
	}
	return argsNewEpochStartDataCreator
}

func createMemUnit() storage.Storer {
	cache, _ := storageUnit.NewCache(storageUnit.LRUCache, 10, 1)
	persist, _ := memorydb.NewlruDB(100000)
	unit, _ := storageUnit.NewStorageUnit(cache, persist)

	return unit
}

func createMetaStore() dataRetriever.StorageService {
	store := dataRetriever.NewChainStorer()
	store.AddStorer(dataRetriever.MetaBlockUnit, createMemUnit())
	store.AddStorer(dataRetriever.BlockHeaderUnit, createMemUnit())

	return store
}

func TestEpochStartCreator_getLastFinalizedMetaHashForShardMetaHashNotReturnsGenesis(t *testing.T) {
	t.Parallel()

	arguments := createMockEpochStartCreatorArguments()
	epoch, _ := blproc.NewEpochStartDataCreator(arguments)
	round := uint64(10)

	shardHdr := &block.Header{Round: round}
	last, lastFinal, shardHdrs, err := epoch.GetLastFinalizedMetaHashForShard(shardHdr)
	assert.Nil(t, last)
	assert.True(t, bytes.Equal(lastFinal, []byte(core.EpochStartIdentifier(0))))
	assert.Equal(t, shardHdr, shardHdrs[0])
	assert.Nil(t, err)
}

func TestEpochStartCreator_getLastFinalizedMetaHashForShardShouldWork(t *testing.T) {
	t.Parallel()

	arguments := createMockEpochStartCreatorArguments()
	arguments.EpochStartTrigger = &mock.EpochStartTriggerStub{
		IsEpochStartCalled: func() bool {
			return false
		},
	}

	dPool := initMetaDataPool()
	dPool.TransactionsCalled = func() dataRetriever.ShardedDataCacherNotifier {
		return &mock.ShardedDataStub{}
	}
	metaHash1 := []byte("hash1")
	metaHash2 := []byte("hash2")
	mbHash1 := []byte("mb_hash1")
	dPool.HeadersCalled = func() dataRetriever.HeadersPool {
		cs := &mock.HeadersCacherStub{}
		cs.GetHeaderByHashCalled = func(hash []byte) (handler data.HeaderHandler, e error) {
			return &block.Header{
				PrevHash:         []byte("hash1"),
				Nonce:            2,
				Round:            2,
				PrevRandSeed:     []byte("roothash"),
				MiniBlockHeaders: []block.MiniBlockHeader{{Hash: mbHash1, SenderShardID: 1}},
				MetaBlockHashes:  [][]byte{metaHash1, metaHash2},
			}, nil
		}
		return cs
	}

	arguments.DataPool = dPool

	epoch, _ := blproc.NewEpochStartDataCreator(arguments)
	round := uint64(10)
	nonce := uint64(1)

	shardHdr := &block.Header{
		Round:           round,
		Nonce:           nonce,
		MetaBlockHashes: [][]byte{mbHash1},
	}
	last, lastFinal, shardHdrs, err := epoch.GetLastFinalizedMetaHashForShard(shardHdr)
	assert.NotNil(t, last)
	assert.NotNil(t, lastFinal)
	assert.NotNil(t, shardHdrs)
	assert.Nil(t, err)
}

func TestEpochStartCreator_CreateEpochStartFromMetaBlockEpochIsNotStarted(t *testing.T) {
	t.Parallel()

	arguments := createMockEpochStartCreatorArguments()
	arguments.EpochStartTrigger = &mock.EpochStartTriggerStub{
		IsEpochStartCalled: func() bool {
			return false
		},
	}

	epoch, _ := blproc.NewEpochStartDataCreator(arguments)

	epStart, err := epoch.CreateEpochStartForMetablock()
	assert.Nil(t, err)

	emptyEpochStart := block.EpochStart{}
	assert.Equal(t, emptyEpochStart, *epStart)
}

func TestEpochStartCreator_CreateEpochStartFromMetaBlockHashComputeIssueShouldErr(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("err computing hash")

	arguments := createMockEpochStartCreatorArguments()
	arguments.Marshalizer = &mock.MarshalizerStub{
		// trigger an error on the Marshal method called from core's ComputeHash
		MarshalCalled: func(obj interface{}) (i []byte, e error) {
			return nil, expectedErr
		},
	}
	arguments.EpochStartTrigger = &mock.EpochStartTriggerStub{
		IsEpochStartCalled: func() bool {
			return true
		},
	}

	epoch, _ := blproc.NewEpochStartDataCreator(arguments)

	epStart, err := epoch.CreateEpochStartForMetablock()
	assert.Nil(t, epStart)
	assert.Equal(t, expectedErr, err)
}

func TestMetaProcessor_CreateEpochStartFromMetaBlockShouldWork(t *testing.T) {
	t.Parallel()

	arguments := createMockEpochStartCreatorArguments()
	arguments.EpochStartTrigger = &mock.EpochStartTriggerStub{
		IsEpochStartCalled: func() bool {
			return true
		},
	}

	hash1 := []byte("hash1")
	hash2 := []byte("hash2")

	startHeaders := createGenesisBlocks(arguments.ShardCoordinator)
	arguments.BlockTracker = mock.NewBlockTrackerMock(arguments.ShardCoordinator, startHeaders)

	hdr := startHeaders[0].(*block.Header)
	hdr.MetaBlockHashes = [][]byte{hash1, hash2}
	hdr.Nonce = 1
	startHeaders[0] = hdr

	dPool := initMetaDataPool()
	dPool.TransactionsCalled = func() dataRetriever.ShardedDataCacherNotifier {
		return &mock.ShardedDataStub{}
	}
	metaHash1 := []byte("hash1")
	metaHash2 := []byte("hash2")
	mbHash1 := []byte("mb_hash1")
	dPool.HeadersCalled = func() dataRetriever.HeadersPool {
		cs := &mock.HeadersCacherStub{}
		cs.GetHeaderByHashCalled = func(hash []byte) (handler data.HeaderHandler, e error) {
			return &block.Header{
				PrevHash:         []byte("hash1"),
				Nonce:            1,
				Round:            1,
				PrevRandSeed:     []byte("roothash"),
				MiniBlockHeaders: []block.MiniBlockHeader{{Hash: mbHash1, SenderShardID: 1}},
				MetaBlockHashes:  [][]byte{metaHash1, metaHash2},
			}, nil
		}

		return cs
	}
	arguments.DataPool = dPool
	metaHdrStorage := arguments.Store.GetStorer(dataRetriever.MetaBlockUnit)
	meta1 := &block.MetaBlock{Nonce: 100}

	var hdrs []block.ShardMiniBlockHeader
	hdrs = append(hdrs, block.ShardMiniBlockHeader{
		Hash:            hash1,
		ReceiverShardID: 0,
		SenderShardID:   1,
		TxCount:         2,
	})
	hdrs = append(hdrs, block.ShardMiniBlockHeader{
		Hash:            mbHash1,
		ReceiverShardID: 0,
		SenderShardID:   1,
		TxCount:         2,
	})
	shardData := block.ShardData{ShardID: 1, ShardMiniBlockHeaders: hdrs}
	meta2 := &block.MetaBlock{Nonce: 101, PrevHash: metaHash1, ShardInfo: []block.ShardData{shardData}}

	marshaledData, _ := arguments.Marshalizer.Marshal(meta1)
	_ = metaHdrStorage.Put(metaHash1, marshaledData)

	marshaledData, _ = arguments.Marshalizer.Marshal(meta2)
	_ = metaHdrStorage.Put(metaHash2, marshaledData)

	epoch, _ := blproc.NewEpochStartDataCreator(arguments)

	epStart, err := epoch.CreateEpochStartForMetablock()
	assert.Nil(t, err)
	assert.NotNil(t, epStart)
	assert.Equal(t, hash1, epStart.LastFinalizedHeaders[0].LastFinishedMetaBlock)
	assert.Equal(t, hash2, epStart.LastFinalizedHeaders[0].FirstPendingMetaBlock)
	assert.Equal(t, 1, len(epStart.LastFinalizedHeaders[0].PendingMiniBlockHeaders))
}
