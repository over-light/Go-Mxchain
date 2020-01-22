package shard_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/factory/shard"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/stretchr/testify/assert"
)

var errExpected = errors.New("expected error")
var chainID = []byte("chain ID")

const maxTxNonceDeltaAllowed = 100

func createStubTopicHandler(matchStrToErrOnCreate string, matchStrToErrOnRegister string) process.TopicHandler {
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

func createDataPools() dataRetriever.PoolsHolder {
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

func createStore() *mock.ChainStorerMock {
	return &mock.ChainStorerMock{
		GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
			return &mock.StorerStub{}
		},
	}
}

//------- NewInterceptorsContainerFactory
func TestNewInterceptorsContainerFactory_NilAccountsAdapter(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		nil,
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilAccountsAdapter, err)
}

func TestNewInterceptorsContainerFactory_NilShardCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		nil,
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilShardCoordinator, err)
}

func TestNewInterceptorsContainerFactory_NilNodesCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		nil,
		&mock.TopicHandlerStub{},
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilNodesCoordinator, err)
}

func TestNewInterceptorsContainerFactory_NilTopicHandlerShouldErr(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		nil,
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilMessenger, err)
}

func TestNewInterceptorsContainerFactory_NilBlockchainShouldErr(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		nil,
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilBlockChain, err)
}

func TestNewInterceptorsContainerFactory_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		createStore(),
		nil,
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilMarshalizer, err)
}

func TestNewInterceptorsContainerFactory_NilMarshalizerAndSizeCheckShouldErr(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		createStore(),
		nil,
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		1,
		&mock.EpochStartTriggerStub{},
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilMarshalizer, err)
}

func TestNewInterceptorsContainerFactory_NilHasherShouldErr(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		createStore(),
		&mock.MarshalizerMock{},
		nil,
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilHasher, err)
}

func TestNewInterceptorsContainerFactory_NilKeyGenShouldErr(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		nil,
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilKeyGen, err)
}

func TestNewInterceptorsContainerFactory_NilSingleSignerShouldErr(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		nil,
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilSingleSigner, err)
}

func TestNewInterceptorsContainerFactory_NilMultiSignerShouldErr(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		nil,
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilMultiSigVerifier, err)
}

func TestNewInterceptorsContainerFactory_NilDataPoolShouldErr(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		nil,
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilDataPoolHolder, err)
}

func TestNewInterceptorsContainerFactory_NilAddrConverterShouldErr(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		nil,
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilAddressConverter, err)
}

func TestNewInterceptorsContainerFactory_NilTxFeeHandlerShouldErr(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		nil,
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilEconomicsFeeHandler, err)
}

func TestNewInterceptorsContainerFactory_NilBlackListHandlerShouldErr(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		nil,
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilBlackListHandler, err)
}

func TestNewInterceptorsContainerFactory_EmptyChainIDShouldErr(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		nil,
		0,
		&mock.EpochStartTriggerStub{},
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrInvalidChainID, err)
}

func TestNewInterceptorsContainerFactory_EmptyEpochStartTriggerShouldErr(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		nil,
	)

	assert.Nil(t, icf)
	assert.Equal(t, process.ErrNilEpochStartTrigger, err)
}

func TestNewInterceptorsContainerFactory_ShouldWork(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	assert.NotNil(t, icf)
	assert.Nil(t, err)
}
func TestNewInterceptorsContainerFactory_ShouldWorkWithSizeCheck(t *testing.T) {
	t.Parallel()

	icf, err := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{},
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		1,
		&mock.EpochStartTriggerStub{},
	)

	assert.NotNil(t, icf)
	assert.Nil(t, err)
}

//------- Create

func TestInterceptorsContainerFactory_CreateTopicCreationTxFailsShouldErr(t *testing.T) {
	t.Parallel()

	icf, _ := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		createStubTopicHandler(factory.TransactionTopic, ""),
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestInterceptorsContainerFactory_CreateTopicCreationHdrFailsShouldErr(t *testing.T) {
	t.Parallel()

	icf, _ := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		createStubTopicHandler(factory.ShardBlocksTopic, ""),
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestInterceptorsContainerFactory_CreateTopicCreationMiniBlocksFailsShouldErr(t *testing.T) {
	t.Parallel()

	icf, _ := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		createStubTopicHandler(factory.MiniBlocksTopic, ""),
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestInterceptorsContainerFactory_CreateTopicCreationMetachainHeadersFailsShouldErr(t *testing.T) {
	t.Parallel()

	icf, _ := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		createStubTopicHandler(factory.MetachainBlocksTopic, ""),
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestInterceptorsContainerFactory_CreateRegisterTxFailsShouldErr(t *testing.T) {
	t.Parallel()

	icf, _ := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		createStubTopicHandler("", factory.TransactionTopic),
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestInterceptorsContainerFactory_CreateRegisterHdrFailsShouldErr(t *testing.T) {
	t.Parallel()

	icf, _ := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		createStubTopicHandler("", factory.ShardBlocksTopic),
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestInterceptorsContainerFactory_CreateRegisterMiniBlocksFailsShouldErr(t *testing.T) {
	t.Parallel()

	icf, _ := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		createStubTopicHandler("", factory.MiniBlocksTopic),
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestInterceptorsContainerFactory_CreateRegisterMetachainHeadersShouldErr(t *testing.T) {
	t.Parallel()

	icf, _ := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		createStubTopicHandler("", factory.MetachainBlocksTopic),
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestInterceptorsContainerFactory_CreateRegisterTrieNodesShouldErr(t *testing.T) {
	t.Parallel()

	icf, _ := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		createStubTopicHandler("", factory.AccountTrieNodesTopic),
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	container, err := icf.Create()

	assert.Nil(t, container)
	assert.Equal(t, errExpected, err)
}

func TestInterceptorsContainerFactory_CreateShouldWork(t *testing.T) {
	t.Parallel()

	icf, _ := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		mock.NewOneShardCoordinatorMock(),
		mock.NewNodesCoordinatorMock(),
		&mock.TopicHandlerStub{
			CreateTopicCalled: func(name string, createChannelForTopic bool) error {
				return nil
			},
			RegisterMessageProcessorCalled: func(topic string, handler p2p.MessageProcessor) error {
				return nil
			},
		},
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	container, err := icf.Create()

	assert.NotNil(t, container)
	assert.Nil(t, err)
}

func TestInterceptorsContainerFactory_With4ShardsShouldWork(t *testing.T) {
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

	icf, _ := shard.NewInterceptorsContainerFactory(
		&mock.AccountsStub{},
		shardCoordinator,
		nodesCoordinator,
		&mock.TopicHandlerStub{
			CreateTopicCalled: func(name string, createChannelForTopic bool) error {
				return nil
			},
			RegisterMessageProcessorCalled: func(topic string, handler p2p.MessageProcessor) error {
				return nil
			},
		},
		createStore(),
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SingleSignKeyGenMock{},
		&mock.SignerMock{},
		&mock.SignerMock{},
		mock.NewMultiSigner(),
		createDataPools(),
		&mock.AddressConverterMock{},
		maxTxNonceDeltaAllowed,
		&mock.FeeHandlerStub{},
		&mock.BlackListHandlerStub{},
		&mock.HeaderSigVerifierStub{},
		chainID,
		0,
		&mock.EpochStartTriggerStub{},
	)

	container, err := icf.Create()

	numInterceptorTxs := noOfShards + 1
	numInterceptorsUnsignedTxs := numInterceptorTxs
	numInterceptorsRewardTxs := numInterceptorTxs
	numInterceptorHeaders := 1
	numInterceptorMiniBlocks := noOfShards + 1
	numInterceptorMetachainHeaders := 1
	numInterceptorTrieNodes := 2
	totalInterceptors := numInterceptorTxs + numInterceptorsUnsignedTxs + numInterceptorsRewardTxs +
		numInterceptorHeaders + numInterceptorMiniBlocks + numInterceptorMetachainHeaders + numInterceptorTrieNodes

	assert.Nil(t, err)
	assert.Equal(t, totalInterceptors, container.Len())
}
