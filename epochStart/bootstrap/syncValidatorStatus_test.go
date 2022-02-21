package bootstrap

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/ElrondNetwork/elrond-go-core/data/endProcess"
	"github.com/ElrondNetwork/elrond-go/epochStart/mock"
	"github.com/ElrondNetwork/elrond-go/sharding/nodesCoordinator"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	dataRetrieverMock "github.com/ElrondNetwork/elrond-go/testscommon/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/testscommon/hashingMocks"
	"github.com/ElrondNetwork/elrond-go/testscommon/nodeTypeProviderMock"
	"github.com/ElrondNetwork/elrond-go/testscommon/shardingMocks"
	"github.com/stretchr/testify/require"
)

const initRating = uint32(50)

func TestNewSyncValidatorStatus_ShouldWork(t *testing.T) {
	t.Parallel()

	args := getSyncValidatorStatusArgs()
	svs, err := NewSyncValidatorStatus(args)
	require.NoError(t, err)
	require.False(t, check.IfNil(svs))
}

func TestSyncValidatorStatus_NodesConfigFromMetaBlock(t *testing.T) {
	t.Parallel()

	args := getSyncValidatorStatusArgs()
	svs, _ := NewSyncValidatorStatus(args)

	currMb := &block.MetaBlock{
		Nonce: 37,
		Epoch: 0,
		MiniBlockHeaders: []block.MiniBlockHeader{
			{
				Hash:            []byte("mb0-hash"),
				ReceiverShardID: 0,
				SenderShardID:   0,
				Type:            block.TxBlock,
				TxCount:         0,
			},
		},
		EpochStart: block.EpochStart{
			LastFinalizedHeaders: []block.EpochStartShardData{
				{
					ShardID:                 0,
					Epoch:                   0,
					Round:                   0,
					Nonce:                   0,
					HeaderHash:              []byte("hash"),
					RootHash:                []byte("rootHash"),
					FirstPendingMetaBlock:   []byte("hash"),
					LastFinishedMetaBlock:   []byte("hash"),
					PendingMiniBlockHeaders: nil,
				},
			},
		}}
	prevMb := &block.MetaBlock{
		Nonce: 36,
		Epoch: 0,
		MiniBlockHeaders: []block.MiniBlockHeader{
			{
				Hash:            []byte("mb0-hash"),
				ReceiverShardID: 0,
				SenderShardID:   0,
				Type:            block.TxBlock,
				TxCount:         0,
			},
		},
		EpochStart: block.EpochStart{
			LastFinalizedHeaders: []block.EpochStartShardData{
				{
					ShardID:                 0,
					Epoch:                   0,
					Round:                   0,
					Nonce:                   0,
					HeaderHash:              []byte("hash"),
					RootHash:                []byte("rootHash"),
					FirstPendingMetaBlock:   []byte("hash"),
					LastFinishedMetaBlock:   []byte("hash"),
					PendingMiniBlockHeaders: nil,
				},
			},
		},
	}

	registry, _, err := svs.NodesConfigFromMetaBlock(currMb, prevMb)
	require.NoError(t, err)
	require.NotNil(t, registry)
}

func getSyncValidatorStatusArgs() ArgsNewSyncValidatorStatus {
	return ArgsNewSyncValidatorStatus{
		DataPool: &dataRetrieverMock.PoolsHolderStub{
			MiniBlocksCalled: func() storage.Cacher {
				return testscommon.NewCacherStub()
			},
		},
		Marshalizer:    &mock.MarshalizerMock{},
		Hasher:         &hashingMocks.HasherMock{},
		RequestHandler: &testscommon.RequestHandlerStub{},
		ChanceComputer: &shardingMocks.NodesCoordinatorStub{},
		GenesisNodesConfig: &mock.NodesSetupStub{
			NumberOfShardsCalled: func() uint32 {
				return 1
			},
			InitialNodesInfoForShardCalled: func(shardID uint32) ([]nodesCoordinator.GenesisNodeInfoHandler, []nodesCoordinator.GenesisNodeInfoHandler, error) {
				if shardID == core.MetachainShardId {
					return []nodesCoordinator.GenesisNodeInfoHandler{
							mock.NewNodeInfo([]byte("addr0"), []byte("pubKey0"), core.MetachainShardId, initRating),
							mock.NewNodeInfo([]byte("addr1"), []byte("pubKey1"), core.MetachainShardId, initRating),
						},
						[]nodesCoordinator.GenesisNodeInfoHandler{&mock.NodeInfoMock{}},
						nil
				}
				return []nodesCoordinator.GenesisNodeInfoHandler{
						mock.NewNodeInfo([]byte("addr0"), []byte("pubKey0"), 0, initRating),
						mock.NewNodeInfo([]byte("addr1"), []byte("pubKey1"), 0, initRating),
					},
					[]nodesCoordinator.GenesisNodeInfoHandler{&mock.NodeInfoMock{}},
					nil
			},
			InitialNodesInfoCalled: func() (map[uint32][]nodesCoordinator.GenesisNodeInfoHandler, map[uint32][]nodesCoordinator.GenesisNodeInfoHandler) {
				return map[uint32][]nodesCoordinator.GenesisNodeInfoHandler{
						0: {
							mock.NewNodeInfo([]byte("addr0"), []byte("pubKey0"), 0, initRating),
							mock.NewNodeInfo([]byte("addr1"), []byte("pubKey1"), 0, initRating),
						},
						core.MetachainShardId: {
							mock.NewNodeInfo([]byte("addr0"), []byte("pubKey0"), core.MetachainShardId, initRating),
							mock.NewNodeInfo([]byte("addr1"), []byte("pubKey1"), core.MetachainShardId, initRating),
						},
					}, map[uint32][]nodesCoordinator.GenesisNodeInfoHandler{0: {
						mock.NewNodeInfo([]byte("addr2"), []byte("pubKey2"), 0, initRating),
						mock.NewNodeInfo([]byte("addr3"), []byte("pubKey3"), 0, initRating),
					}}
			},
			GetShardConsensusGroupSizeCalled: func() uint32 {
				return 2
			},
			GetMetaConsensusGroupSizeCalled: func() uint32 {
				return 2
			},
		},
		NodeShuffler:      &shardingMocks.NodeShufflerMock{},
		PubKey:            []byte("public key"),
		ShardIdAsObserver: 0,
		ChanNodeStop:      endProcess.GetDummyEndProcessChannel(),
		NodeTypeProvider:  &nodeTypeProviderMock.NodeTypeProviderStub{},
		IsFullArchive:     false,
	}
}
