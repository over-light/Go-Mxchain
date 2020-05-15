package bootstrap

import (
	"context"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	triesFactory "github.com/ElrondNetwork/elrond-go/data/trie/factory"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/epochStart/mock"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/stretchr/testify/assert"
)

func createPkBytes(numShards uint32) map[uint32][]byte {
	pksbytes := make(map[uint32][]byte, numShards+1)
	for i := uint32(0); i < numShards; i++ {
		pksbytes[i] = make([]byte, 128)
		pksbytes[i] = []byte("afafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafaf")
		pksbytes[i][0] = byte(i)
	}

	pksbytes[core.MetachainShardId] = make([]byte, 128)
	pksbytes[core.MetachainShardId] = []byte("afafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafaf")
	pksbytes[core.MetachainShardId][0] = byte(numShards)

	return pksbytes
}

func createMockEpochStartBootstrapArgs() ArgsEpochStartBootstrap {
	return ArgsEpochStartBootstrap{
		CoreComponentsHolder: &mock.CoreComponentsMock{
			IntMarsh:            &mock.MarshalizerMock{},
			Marsh:               &mock.MarshalizerMock{},
			Hash:                &mock.HasherMock{},
			UInt64ByteSliceConv: &mock.Uint64ByteSliceConverterMock{},
			AddrPubKeyConv:      &mock.PubkeyConverterMock{},
			PathHdl:             &mock.PathManagerStub{},
		},
		CryptoComponentsHolder: &mock.CryptoComponentsMock{
			PubKey:   &mock.PublicKeyMock{},
			BlockSig: &mock.SignerStub{},
			TxSig:    &mock.SignerStub{},
			BlKeyGen: &mock.KeyGenMock{},
			TxKeyGen: &mock.KeyGenMock{},
		},
		Messenger: &mock.MessengerStub{},
		GeneralConfig: config.Config{
			WhiteListPool: config.CacheConfig{
				Type:        "LRU",
				Size:        10,
				SizeInBytes: 1000,
				Shards:      10,
			},
			EpochStartConfig: config.EpochStartConfig{
				MinNumConnectedPeersToStart:       2,
				MinNumOfPeersToConsiderBlockValid: 2,
			},
		},
		EconomicsData:              &economics.EconomicsData{},
		GenesisNodesConfig:         &mock.NodesSetupStub{},
		GenesisShardCoordinator:    mock.NewMultipleShardsCoordinatorMock(),
		Rater:                      &mock.RaterStub{},
		DestinationShardAsObserver: 0,
		TrieContainer: &mock.TriesHolderMock{
			GetCalled: func(bytes []byte) data.Trie {
				return &mock.TrieStub{}
			},
		},
		TrieStorageManagers: map[string]data.StorageManager{
			triesFactory.UserAccountTrie: &mock.StorageManagerStub{},
			triesFactory.PeerAccountTrie: &mock.StorageManagerStub{},
		},
		NodeShuffler:              &mock.NodeShufflerMock{},
		Rounder:                   &mock.RounderStub{},
		LatestStorageDataProvider: &mock.LatestStorageDataProviderStub{},
		StorageUnitOpener:         &mock.UnitOpenerStub{},
	}
}

func TestNewEpochStartBootstrap(t *testing.T) {
	t.Parallel()

	args := createMockEpochStartBootstrapArgs()

	epochStartProvider, err := NewEpochStartBootstrap(args)
	assert.Nil(t, err)
	assert.False(t, check.IfNil(epochStartProvider))
}

func TestIsStartInEpochZero(t *testing.T) {
	t.Parallel()

	args := createMockEpochStartBootstrapArgs()
	args.GenesisNodesConfig = &mock.NodesSetupStub{
		GetStartTimeCalled: func() int64 {
			return 1000
		},
	}

	epochStartProvider, _ := NewEpochStartBootstrap(args)

	result := epochStartProvider.isStartInEpochZero()
	assert.False(t, result)
}

func TestEpochStartBootstrap_BootstrapStartInEpochNotEnabled(t *testing.T) {
	args := createMockEpochStartBootstrapArgs()

	epochStartProvider, _ := NewEpochStartBootstrap(args)

	params, err := epochStartProvider.Bootstrap()
	assert.Nil(t, err)
	assert.NotNil(t, params)
}

func TestEpochStartBootstrap_Bootstrap(t *testing.T) {
	roundsPerEpoch := int64(100)
	roundDuration := uint64(60000)
	args := createMockEpochStartBootstrapArgs()
	args.GenesisNodesConfig = &mock.NodesSetupStub{
		GetRoundDurationCalled: func() uint64 {
			return roundDuration
		},
	}
	args.GeneralConfig = getGeneralConfig()
	args.GeneralConfig.EpochStartConfig.RoundsPerEpoch = roundsPerEpoch
	epochStartProvider, _ := NewEpochStartBootstrap(args)

	done := make(chan bool, 1)

	go func() {
		_, _ = epochStartProvider.Bootstrap()
		<-done
	}()

	for {
		select {
		case <-done:
			assert.Fail(t, "should not be reach")
		case <-time.After(time.Second):
			assert.True(t, true, "pass with timeout")
			return
		}
	}
}

func TestPrepareForEpochZero(t *testing.T) {
	args := createMockEpochStartBootstrapArgs()

	epochStartProvider, _ := NewEpochStartBootstrap(args)

	params, err := epochStartProvider.prepareEpochZero()
	assert.Nil(t, err)
	assert.Equal(t, uint32(0), params.Epoch)
}

func TestCreateSyncers(t *testing.T) {
	args := createMockEpochStartBootstrapArgs()

	epochStartProvider, _ := NewEpochStartBootstrap(args)
	epochStartProvider.shardCoordinator = mock.NewMultipleShardsCoordinatorMock()
	epochStartProvider.dataPool = &mock.PoolsHolderStub{
		HeadersCalled: func() dataRetriever.HeadersPool {
			return &mock.HeadersCacherStub{}
		},
		TransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{}
		},
		UnsignedTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{}
		},
		RewardTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{}
		},
		MiniBlocksCalled: func() storage.Cacher {
			return &mock.CacherStub{}
		},
		TrieNodesCalled: func() storage.Cacher {
			return &mock.CacherStub{}
		},
	}
	epochStartProvider.whiteListHandler = &mock.WhiteListHandlerStub{}
	epochStartProvider.whiteListerVerifiedTxs = &mock.WhiteListHandlerStub{}
	epochStartProvider.requestHandler = &mock.RequestHandlerStub{}

	err := epochStartProvider.createSyncers()
	assert.Nil(t, err)
}

func TestSyncHeadersFrom_MockHeadersSyncerShouldSyncHeaders(t *testing.T) {
	hdrHash1 := []byte("hdrHash1")
	hdrHash2 := []byte("hdrHash2")
	header1 := &block.Header{}
	header2 := &block.MetaBlock{}

	args := createMockEpochStartBootstrapArgs()
	epochStartProvider, _ := NewEpochStartBootstrap(args)
	epochStartProvider.headersSyncer = &mock.HeadersByHashSyncerStub{
		SyncMissingHeadersByHashCalled: func(shardIDs []uint32, headersHashes [][]byte, ctx context.Context) error {
			return nil
		},
		GetHeadersCalled: func() (m map[string]data.HeaderHandler, err error) {
			return map[string]data.HeaderHandler{
				string(hdrHash1): header1,
				string(hdrHash2): header2,
			}, nil
		},
	}

	metaBlock := &block.MetaBlock{
		Epoch: 2,
		EpochStart: block.EpochStart{
			LastFinalizedHeaders: []block.EpochStartShardData{
				{HeaderHash: hdrHash1, ShardID: 0},
			},
			Economics: block.Economics{
				PrevEpochStartHash: hdrHash2,
			},
		},
	}

	headers, err := epochStartProvider.syncHeadersFrom(metaBlock)
	assert.Nil(t, err)
	assert.Equal(t, header1, headers[string(hdrHash1)])
	assert.Equal(t, header2, headers[string(hdrHash2)])
}

func TestSyncPeerAccountsState_NilRequestHandlerErr(t *testing.T) {
	args := createMockEpochStartBootstrapArgs()
	epochStartProvider, _ := NewEpochStartBootstrap(args)
	epochStartProvider.dataPool = &mock.PoolsHolderStub{
		TrieNodesCalled: func() storage.Cacher {
			return &mock.CacherStub{
				GetCalled: func(key []byte) (value interface{}, ok bool) {
					return nil, true
				},
			}
		},
	}

	rootHash := []byte("rootHash")
	err := epochStartProvider.syncPeerAccountsState(rootHash)
	assert.Equal(t, state.ErrNilRequestHandler, err)
}

func TestCreateTriesForNewShardID(t *testing.T) {
	args := createMockEpochStartBootstrapArgs()
	args.GeneralConfig = getGeneralConfig()
	epochStartProvider, _ := NewEpochStartBootstrap(args)

	err := epochStartProvider.createTriesForNewShardId(1)
	assert.Nil(t, err)
}

func TestSyncUserAccountsState(t *testing.T) {
	args := createMockEpochStartBootstrapArgs()

	epochStartProvider, _ := NewEpochStartBootstrap(args)
	epochStartProvider.shardCoordinator = mock.NewMultipleShardsCoordinatorMock()
	epochStartProvider.dataPool = &mock.PoolsHolderStub{
		TrieNodesCalled: func() storage.Cacher {
			return &mock.CacherStub{
				GetCalled: func(key []byte) (value interface{}, ok bool) {
					return nil, true
				},
			}
		},
	}

	rootHash := []byte("rootHash")
	err := epochStartProvider.syncUserAccountsState(rootHash)
	assert.Equal(t, state.ErrNilRequestHandler, err)
}

func TestRequestAndProcessForShard(t *testing.T) {
	args := createMockEpochStartBootstrapArgs()

	hdrHash1 := []byte("hdrHash1")
	header1 := &block.Header{}
	metaBlock := &block.MetaBlock{
		Epoch: 2,
		EpochStart: block.EpochStart{
			LastFinalizedHeaders: []block.EpochStartShardData{
				{HeaderHash: hdrHash1, ShardID: 0},
			},
		},
	}

	shardCoordinator := mock.NewMultipleShardsCoordinatorMock()
	shardCoordinator.CurrentShard = 0

	epochStartProvider, _ := NewEpochStartBootstrap(args)
	epochStartProvider.syncedHeaders = make(map[string]data.HeaderHandler)
	epochStartProvider.miniBlocksSyncer = &mock.PendingMiniBlockSyncHandlerStub{}
	epochStartProvider.headersSyncer = &mock.HeadersByHashSyncerStub{
		GetHeadersCalled: func() (m map[string]data.HeaderHandler, err error) {
			return map[string]data.HeaderHandler{
				string(hdrHash1): header1,
			}, nil
		},
	}
	epochStartProvider.dataPool = &mock.PoolsHolderStub{
		TrieNodesCalled: func() storage.Cacher {
			return &mock.CacherStub{
				GetCalled: func(key []byte) (value interface{}, ok bool) {
					return nil, true
				},
			}
		},
	}

	epochStartProvider.shardCoordinator = shardCoordinator
	epochStartProvider.epochStartMeta = metaBlock
	err := epochStartProvider.requestAndProcessForShard()
	assert.Equal(t, state.ErrNilRequestHandler, err)
}

func getNodesConfigMock(numOfShards uint32) sharding.GenesisNodesSetupHandler {
	pksBytes := createPkBytes(numOfShards)
	address := make([]byte, 32)
	address = []byte("afafafafafafafafafafafafafafafaf")

	roundDurationMillis := 4000
	epochDurationMillis := 50 * int64(roundDurationMillis)

	nodesConfig := &mock.NodesSetupStub{
		InitialNodesInfoCalled: func() (m map[uint32][]sharding.GenesisNodeInfoHandler, m2 map[uint32][]sharding.GenesisNodeInfoHandler) {
			oneMap := make(map[uint32][]sharding.GenesisNodeInfoHandler)
			for i := uint32(0); i < numOfShards; i++ {
				oneMap[i] = append(oneMap[i], mock.NewNodeInfo(address, pksBytes[i], i))
			}
			oneMap[core.MetachainShardId] = append(oneMap[core.MetachainShardId], mock.NewNodeInfo(address, pksBytes[core.MetachainShardId], core.MetachainShardId))
			return oneMap, nil
		},
		GetStartTimeCalled: func() int64 {
			return time.Now().Add(-time.Duration(epochDurationMillis) * time.Millisecond).Unix()
		},
		GetRoundDurationCalled: func() uint64 {
			return 4000
		},
		GetChainIdCalled: func() string {
			return "chainId"
		},
		GetShardConsensusGroupSizeCalled: func() uint32 {
			return 1
		},
		GetMetaConsensusGroupSizeCalled: func() uint32 {
			return 1
		},
		NumberOfShardsCalled: func() uint32 {
			return numOfShards
		},
	}

	return nodesConfig
}

func TestRequestAndProcessing(t *testing.T) {
	args := createMockEpochStartBootstrapArgs()
	args.GenesisNodesConfig = getNodesConfigMock(1)

	hdrHash1 := []byte("hdrHash1")
	hdrHash2 := []byte("hdrHash2")
	header1 := &block.Header{}
	header2 := &block.MetaBlock{
		Epoch: 1,
		EpochStart: block.EpochStart{
			LastFinalizedHeaders: []block.EpochStartShardData{
				{HeaderHash: hdrHash1, ShardID: 0},
			},
			Economics: block.Economics{
				PrevEpochStartHash: hdrHash1,
			},
		},
	}
	metaBlock := &block.MetaBlock{
		Epoch: 0,
		EpochStart: block.EpochStart{
			LastFinalizedHeaders: []block.EpochStartShardData{
				{HeaderHash: hdrHash1, ShardID: 0},
			},
			Economics: block.Economics{
				PrevEpochStartHash: hdrHash2,
			},
		},
	}

	epochStartProvider, _ := NewEpochStartBootstrap(args)
	epochStartProvider.epochStartMeta = metaBlock
	epochStartProvider.headersSyncer = &mock.HeadersByHashSyncerStub{
		GetHeadersCalled: func() (m map[string]data.HeaderHandler, err error) {
			return map[string]data.HeaderHandler{
				string(hdrHash1): header1,
				string(hdrHash2): header2,
			}, nil
		},
	}
	epochStartProvider.dataPool = &mock.PoolsHolderStub{
		MiniBlocksCalled: func() storage.Cacher {
			return &mock.CacherStub{}
		},
		TrieNodesCalled: func() storage.Cacher {
			return &mock.CacherStub{
				GetCalled: func(key []byte) (value interface{}, ok bool) {
					return nil, true
				},
			}
		},
	}
	epochStartProvider.requestHandler = &mock.RequestHandlerStub{}
	epochStartProvider.miniBlocksSyncer = &mock.PendingMiniBlockSyncHandlerStub{}

	params, err := epochStartProvider.requestAndProcessing()
	assert.Equal(t, Parameters{}, params)
	assert.Equal(t, storage.ErrInvalidNumberOfEpochsToSave, err)
}
