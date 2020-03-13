package interceptorscontainer_test

import (
	"strings"
	"testing"

	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/factory/interceptorscontainer"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/stretchr/testify/assert"
)

func createShardStubTopicHandler(matchStrToErrOnCreate string, matchStrToErrOnRegister string) process.TopicHandler {
	return &mock.TopicHandlerStub{
		CreateTopicCalled: func(name string, createChannelForTopic bool) error {
			if matchStrToErrOnCreate == "" {
				return nil
			}
			if strings.Contains(name, matchStrToErrOnCreate) {
				return errExpected
			}

			return nil
		},
		RegisterMessageProcessorCalled: func(topic string, handler p2p.MessageProcessor) error {
			if matchStrToErrOnRegister == "" {
				return nil
			}
			if strings.Contains(topic, matchStrToErrOnRegister) {
				return errExpected
			}

			return nil
		},
	}
}

func createShardDataPools() dataRetriever.PoolsHolder {
	pools := &mock.PoolsHolderStub{}
	pools.TransactionsCalled = func() dataRetriever.ShardedDataCacherNotifier {
		return &mock.ShardedDataStub{}
	}
	pools.HeadersCalled = func() dataRetriever.HeadersPool {
		return &mock.HeadersCacherStub{}
	}
	pools.MiniBlocksCalled = func() storage.Cacher {
		return &mock.CacherStub{}
	}
	pools.PeerChangesBlocksCalled = func() storage.Cacher {
		return &mock.CacherStub{}
	}
	pools.MetaBlocksCalled = func() storage.Cacher {
		return &mock.CacherStub{}
	}
	pools.UnsignedTransactionsCalled = func() dataRetriever.ShardedDataCacherNotifier {
		return &mock.ShardedDataStub{}
	}
	pools.RewardTransactionsCalled = func() dataRetriever.ShardedDataCacherNotifier {
		return &mock.ShardedDataStub{}
	}
	pools.TrieNodesCalled = func() storage.Cacher {
		return &mock.CacherStub{}
	}
	pools.CurrBlockTxsCalled = func() dataRetriever.TransactionCacher {
		return &mock.TxForCurrentBlockStub{}
	}
	return pools
}

func createShardStore() *mock.ChainStorerMock {
	return &mock.ChainStorerMock{
		GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
			return &mock.StorerStub{}
		},
	}
}

//------- NewInterceptorsContainerFactory
func TestNewShardInterceptorsContainerFactory_NilAccountsAdapter(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.Accounts = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilAccountsAdapter, err)
}

func TestNewShardInterceptorsContainerFactory_NilShardCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.ShardCoordinator = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilShardCoordinator, err)
}

func TestNewShardInterceptorsContainerFactory_NilNodesCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.NodesCoordinator = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilNodesCoordinator, err)
}

func TestNewShardInterceptorsContainerFactory_NilMessengerShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.Messenger = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilMessenger, err)
}

func TestNewShardInterceptorsContainerFactory_NilStoreShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.Store = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilStore, err)
}

func TestNewShardInterceptorsContainerFactory_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.ProtoMarshalizer = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilMarshalizer, err)
}

func TestNewShardInterceptorsContainerFactory_NilMarshalizerAndSizeCheckShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.ProtoMarshalizer = nil
	args.SizeCheckDelta = 1
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilMarshalizer, err)
}

func TestNewShardInterceptorsContainerFactory_NilHasherShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.Hasher = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilHasher, err)
}

func TestNewShardInterceptorsContainerFactory_NilKeyGenShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.KeyGen = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilKeyGen, err)
}

func TestNewShardInterceptorsContainerFactory_NilSingleSignerShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.SingleSigner = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilSingleSigner, err)
}

func TestNewShardInterceptorsContainerFactory_NilMultiSignerShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.MultiSigner = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilMultiSigVerifier, err)
}

func TestNewShardInterceptorsContainerFactory_NilDataPoolShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.DataPool = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilDataPoolHolder, err)
}

func TestNewShardInterceptorsContainerFactory_NilAddrConverterShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.AddrConverter = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilAddressConverter, err)
}

func TestNewShardInterceptorsContainerFactory_NilTxFeeHandlerShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.TxFeeHandler = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilEconomicsFeeHandler, err)
}

func TestNewShardInterceptorsContainerFactory_NilBlackListHandlerShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.BlackList = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilBlackListHandler, err)
}

func TestNewShardInterceptorsContainerFactory_EmptyChainIDShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.ChainID = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrInvalidChainID, err)
}

func TestNewShardInterceptorsContainerFactory_NilValidityAttesterShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.ValidityAttester = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilValidityAttester, err)
}

func TestNewShardInterceptorsContainerFactory_EmptyEpochStartTriggerShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.EpochStartTrigger = nil
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilEpochStartTrigger, err)
}

func TestNewShardInterceptorsContainerFactory_ShouldWork(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.NotNil(t, icf)
	assert.Nil(t, err)
	assert.False(t, icf.IsInterfaceNil())
}
func TestNewShardInterceptorsContainerFactory_ShouldWorkWithSizeCheck(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.SizeCheckDelta = 1
	icf, err := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	assert.NotNil(t, icf)
	assert.Nil(t, err)
}

//------- Create

func TestShardInterceptorsContainerFactory_CreateTopicCreationTxFailsShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.Messenger = createShardStubTopicHandler(factory.TransactionTopic, "")
	icf, _ := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestShardInterceptorsContainerFactory_CreateTopicCreationHdrFailsShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.Messenger = createShardStubTopicHandler(factory.ShardBlocksTopic, "")
	icf, _ := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestShardInterceptorsContainerFactory_CreateTopicCreationMiniBlocksFailsShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.Messenger = createShardStubTopicHandler(factory.MiniBlocksTopic, "")
	icf, _ := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestShardInterceptorsContainerFactory_CreateTopicCreationMetachainHeadersFailsShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.Messenger = createShardStubTopicHandler(factory.MetachainBlocksTopic, "")
	icf, _ := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestShardInterceptorsContainerFactory_CreateRegisterTxFailsShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.Messenger = createShardStubTopicHandler("", factory.TransactionTopic)
	icf, _ := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestShardInterceptorsContainerFactory_CreateRegisterHdrFailsShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.Messenger = createShardStubTopicHandler("", factory.ShardBlocksTopic)
	icf, _ := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestShardInterceptorsContainerFactory_CreateRegisterMiniBlocksFailsShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.Messenger = createShardStubTopicHandler("", factory.MiniBlocksTopic)
	icf, _ := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestShardInterceptorsContainerFactory_CreateRegisterMetachainHeadersShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.Messenger = createShardStubTopicHandler("", factory.MetachainBlocksTopic)
	icf, _ := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestShardInterceptorsContainerFactory_CreateRegisterTrieNodesShouldErr(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.Messenger = createShardStubTopicHandler("", factory.AccountTrieNodesTopic)
	icf, _ := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestShardInterceptorsContainerFactory_CreateShouldWork(t *testing.T) {
	t.Parallel()

	args := getArgumentsShard()
	args.Messenger = &mock.TopicHandlerStub{
		CreateTopicCalled: func(name string, createChannelForTopic bool) error {
			return nil
		},
		RegisterMessageProcessorCalled: func(topic string, handler p2p.MessageProcessor) error {
			return nil
		},
	}
	icf, _ := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	container, err := icf.Create()

	assert.NotNil(t, container)
	assert.Nil(t, err)
}

func TestShardInterceptorsContainerFactory_With4ShardsShouldWork(t *testing.T) {
	t.Parallel()

	noOfShards := 4

	shardCoordinator := mock.NewMultipleShardsCoordinatorMock()
	shardCoordinator.SetNoShards(uint32(noOfShards))
	shardCoordinator.CurrentShard = 1

	nodesCoordinator := &mock.NodesCoordinatorMock{
		ShardId:            1,
		ShardConsensusSize: 1,
		MetaConsensusSize:  1,
		NbShards:           uint32(noOfShards),
	}

	mesenger := &mock.TopicHandlerStub{
		CreateTopicCalled: func(name string, createChannelForTopic bool) error {
			return nil
		},
		RegisterMessageProcessorCalled: func(topic string, handler p2p.MessageProcessor) error {
			return nil
		},
	}

	args := getArgumentsShard()
	args.ShardCoordinator = shardCoordinator
	args.NodesCoordinator = nodesCoordinator
	args.Messenger = mesenger

	icf, _ := interceptorscontainer.NewShardInterceptorsContainerFactory(args)

	container, err := icf.Create()

	numInterceptorTxs := noOfShards + 1
	numInterceptorsUnsignedTxs := numInterceptorTxs
	numInterceptorsRewardTxs := 1
	numInterceptorHeaders := 1
	numInterceptorMiniBlocks := noOfShards + 1
	numInterceptorMetachainHeaders := 1
	numInterceptorTrieNodes := 3
	totalInterceptors := numInterceptorTxs + numInterceptorsUnsignedTxs + numInterceptorsRewardTxs +
		numInterceptorHeaders + numInterceptorMiniBlocks + numInterceptorMetachainHeaders + numInterceptorTrieNodes

	assert.Nil(t, err)
	assert.Equal(t, totalInterceptors, container.Len())
}

func getArgumentsShard() interceptorscontainer.ShardInterceptorsContainerFactoryArgs {
	return interceptorscontainer.ShardInterceptorsContainerFactoryArgs{
		Accounts:               &mock.AccountsStub{},
		ShardCoordinator:       mock.NewOneShardCoordinatorMock(),
		NodesCoordinator:       mock.NewNodesCoordinatorMock(),
		Messenger:              &mock.TopicHandlerStub{},
		Store:                  createShardStore(),
		ProtoMarshalizer:       &mock.MarshalizerMock{},
		TxSignMarshalizer:      &mock.MarshalizerMock{},
		Hasher:                 &mock.HasherMock{},
		KeyGen:                 &mock.SingleSignKeyGenMock{},
		BlockSignKeyGen:        &mock.SingleSignKeyGenMock{},
		SingleSigner:           &mock.SignerMock{},
		BlockSingleSigner:      &mock.SignerMock{},
		MultiSigner:            mock.NewMultiSigner(),
		DataPool:               createShardDataPools(),
		AddrConverter:          &mock.AddressConverterMock{},
		MaxTxNonceDeltaAllowed: maxTxNonceDeltaAllowed,
		TxFeeHandler:           &mock.FeeHandlerStub{},
		BlackList:              &mock.BlackListHandlerStub{},
		HeaderSigVerifier:      &mock.HeaderSigVerifierStub{},
		ChainID:                chainID,
		SizeCheckDelta:         0,
		ValidityAttester:       &mock.ValidityAttesterStub{},
		EpochStartTrigger:      &mock.EpochStartTriggerStub{},
	}
}
