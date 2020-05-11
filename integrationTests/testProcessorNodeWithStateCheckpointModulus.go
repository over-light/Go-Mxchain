package integrationTests

import (
	"context"

	"github.com/ElrondNetwork/elrond-go/consensus/spos/sposFactory"
	"github.com/ElrondNetwork/elrond-go/integrationTests/mock"
	"github.com/ElrondNetwork/elrond-go/process/smartContract"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// NewTestProcessorNodeWithStateCheckpointModulus creates a new testNodeProcessor with custom state checkpoint modulus
func NewTestProcessorNodeWithStateCheckpointModulus(
	maxShards uint32,
	nodeShardId uint32,
	txSignPrivKeyShardId uint32,
	initialNodeAddr string,
	stateCheckpointModulus uint,
) *TestProcessorNode {

	shardCoordinator, _ := sharding.NewMultiShardCoordinator(maxShards, nodeShardId)

	kg := &mock.KeyGenMock{}
	sk, pk := kg.GeneratePair()

	pkBytes := make([]byte, 128)
	pkBytes = []byte("afafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafaf")
	address := make([]byte, 32)
	address = []byte("afafafafafafafafafafafafafafafaf")

	nodesSetup := &mock.NodesSetupStub{
		InitialNodesInfoCalled: func() (m map[uint32][]sharding.GenesisNodeInfoHandler, m2 map[uint32][]sharding.GenesisNodeInfoHandler) {
			oneMap := make(map[uint32][]sharding.GenesisNodeInfoHandler)
			oneMap[0] = append(oneMap[0], mock.NewNodeInfo(address, pkBytes, 0))
			return oneMap, nil
		},
		InitialNodesInfoForShardCalled: func(shardId uint32) (handlers []sharding.GenesisNodeInfoHandler, handlers2 []sharding.GenesisNodeInfoHandler, err error) {
			list := make([]sharding.GenesisNodeInfoHandler, 0)
			list = append(list, mock.NewNodeInfo(address, pkBytes, 0))
			return list, nil, nil
		},
	}

	nodesCoordinator := &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validators []sharding.Validator, err error) {
			v, _ := sharding.NewValidator(pkBytes, defaultChancesSelection, 1)
			return []sharding.Validator{v}, nil
		},
		GetAllValidatorsPublicKeysCalled: func() (map[uint32][][]byte, error) {
			keys := make(map[uint32][][]byte)
			keys[0] = make([][]byte, 0)
			keys[0] = append(keys[0], pkBytes)
			return keys, nil
		},
		GetValidatorWithPublicKeyCalled: func(publicKey []byte) (sharding.Validator, uint32, error) {
			validator, _ := sharding.NewValidator(publicKey, defaultChancesSelection, 1)
			return validator, 0, nil
		},
	}

	messenger := CreateMessengerWithKadDht(context.Background(), initialNodeAddr)
	tpn := &TestProcessorNode{
		ShardCoordinator:  shardCoordinator,
		Messenger:         messenger,
		NodesCoordinator:  nodesCoordinator,
		HeaderSigVerifier: &mock.HeaderSigVerifierStub{},
		ChainID:           ChainID,
	}
	tpn.NodesSetup = nodesSetup

	tpn.NodeKeys = &TestKeyPair{
		Sk: sk,
		Pk: pk,
	}
	tpn.MultiSigner = TestMultiSig
	tpn.OwnAccount = CreateTestWalletAccount(shardCoordinator, txSignPrivKeyShardId)
	tpn.initDataPools()
	tpn.initHeaderValidator()
	tpn.initRounder()
	tpn.NetworkShardingCollector = mock.NewNetworkShardingCollectorMock()
	tpn.initStorage()
	tpn.initAccountDBs()
	tpn.initChainHandler()
	tpn.initEconomicsData()
	tpn.initRatingsData()
	tpn.initRequestedItemsHandler()
	tpn.initResolvers()
	tpn.initValidatorStatistics()
	tpn.GenesisBlocks = CreateGenesisBlocks(
		tpn.AccntState,
		tpn.PeerState,
		TestAddressPubkeyConverter,
		tpn.NodesSetup,
		tpn.ShardCoordinator,
		tpn.Storage,
		tpn.BlockChain,
		TestMarshalizer,
		TestHasher,
		TestUint64Converter,
		tpn.DataPool,
		tpn.EconomicsData.EconomicsData,
	)
	tpn.initBlockTracker()
	tpn.initInterceptors()
	tpn.initInnerProcessors()
	tpn.SCQueryService, _ = smartContract.NewSCQueryService(tpn.VMContainer, tpn.EconomicsData)
	tpn.initBlockProcessor(stateCheckpointModulus)
	tpn.BroadcastMessenger, _ = sposFactory.GetBroadcastMessenger(
		TestMarshalizer,
		tpn.Messenger,
		tpn.ShardCoordinator,
		tpn.OwnAccount.SkTxSign,
		tpn.OwnAccount.SingleSigner,
		tpn.DataPool.Headers(),
	)
	tpn.setGenesisBlock()
	tpn.initNode()
	tpn.addHandlersForCounters()
	tpn.addGenesisBlocksIntoStorage()

	return tpn
}
