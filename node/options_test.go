package node

import (
	"errors"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/data/blockchain"
	"github.com/ElrondNetwork/elrond-go/node/mock"
	"github.com/ElrondNetwork/elrond-go/statusHandler"
	"github.com/stretchr/testify/assert"
)

const testSizeCheckDelta = 100

func TestWithMessenger_NilMessengerShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithMessenger(nil)
	err := opt(node)

	assert.Nil(t, node.messenger)
	assert.Equal(t, ErrNilMessenger, err)
}

func TestWithMessenger_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	messenger := &mock.MessengerStub{}

	opt := WithMessenger(messenger)
	err := opt(node)

	assert.True(t, node.messenger == messenger)
	assert.Nil(t, err)
}

func TestWithInternalMarshalizer_NilProtoMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithInternalMarshalizer(nil, testSizeCheckDelta)
	err := opt(node)

	assert.Nil(t, node.internalMarshalizer)
	assert.Equal(t, ErrNilMarshalizer, err)
}

func TestWithInternalMarshalizerr_NilVmMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithVmMarshalizer(nil)
	err := opt(node)

	assert.Nil(t, node.vmMarshalizer)
	assert.Equal(t, ErrNilMarshalizer, err)
}

func TestWithMarshalizer_NilTxSignMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithTxSignMarshalizer(nil)
	err := opt(node)

	assert.Nil(t, node.txSignMarshalizer)
	assert.Equal(t, ErrNilMarshalizer, err)
}

func TestWithProtoMarshalizer_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	marshalizer := &mock.MarshalizerMock{}

	opt := WithInternalMarshalizer(marshalizer, testSizeCheckDelta)
	err := opt(node)

	assert.True(t, node.internalMarshalizer == marshalizer)
	assert.True(t, node.sizeCheckDelta == testSizeCheckDelta)
	assert.Nil(t, err)
}

func TestWithVmMarshalizer_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	marshalizer := &mock.MarshalizerMock{}

	opt := WithVmMarshalizer(marshalizer)
	err := opt(node)

	assert.True(t, node.vmMarshalizer == marshalizer)
	assert.Nil(t, err)
}

func TestWithTxSignMarshalizer_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	marshalizer := &mock.MarshalizerMock{}

	opt := WithTxSignMarshalizer(marshalizer)
	err := opt(node)

	assert.True(t, node.txSignMarshalizer == marshalizer)
	assert.Nil(t, err)
}

func TestWithHasher_NilHasherShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithHasher(nil)
	err := opt(node)

	assert.Nil(t, node.hasher)
	assert.Equal(t, ErrNilHasher, err)
}

func TestWithHasher_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	hasher := &mock.HasherMock{}

	opt := WithHasher(hasher)
	err := opt(node)

	assert.True(t, node.hasher == hasher)
	assert.Nil(t, err)
}

func TestWithAccountsAdapter_NilAccountsShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithAccountsAdapter(nil)
	err := opt(node)

	assert.Nil(t, node.accounts)
	assert.Equal(t, ErrNilAccountsAdapter, err)
}

func TestWithAccountsAdapter_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	accounts := &mock.AccountsStub{}

	opt := WithAccountsAdapter(accounts)
	err := opt(node)

	assert.True(t, node.accounts == accounts)
	assert.Nil(t, err)
}

func TestWithPubkeyConverter_NilConverterShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithPubkeyConverter(nil)
	err := opt(node)

	assert.Nil(t, node.pubkeyConverter)
	assert.Equal(t, ErrNilPubkeyConverter, err)
}

func TestWithPubkeyConverter_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	converter := &mock.PubkeyConverterStub{}

	opt := WithPubkeyConverter(converter)
	err := opt(node)

	assert.True(t, node.pubkeyConverter == converter)
	assert.Nil(t, err)
}

func TestWithBlockChain_NilBlockchainrShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithBlockChain(nil)
	err := opt(node)

	assert.Nil(t, node.blkc)
	assert.Equal(t, ErrNilBlockchain, err)
}

func TestWithBlockChain_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	blkc := blockchain.NewBlockChain()

	opt := WithBlockChain(blkc)
	err := opt(node)

	assert.True(t, node.blkc == blkc)
	assert.Nil(t, err)
}

func TestWithDataStore_NilStoreShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithDataStore(nil)
	err := opt(node)

	assert.Nil(t, node.store)
	assert.Equal(t, ErrNilStore, err)
}

func TestWithDataStore_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	store := &mock.ChainStorerMock{}

	opt := WithDataStore(store)
	err := opt(node)

	assert.True(t, node.store == store)
	assert.Nil(t, err)
}

func TestWithPrivateKey_NilBlsPrivateKeyShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithPrivKey(nil)
	err := opt(node)

	assert.Nil(t, node.privKey)
	assert.Equal(t, ErrNilPrivateKey, err)
}

func TestWithBlsPrivateKey_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	sk := &mock.PrivateKeyStub{}

	opt := WithPrivKey(sk)
	err := opt(node)

	assert.True(t, node.privKey == sk)
	assert.Nil(t, err)
}

func TestWithSingleSignKeyGenerator_NilPrivateKeyShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithKeyGen(nil)
	err := opt(node)

	assert.Nil(t, node.keyGen)
	assert.Equal(t, ErrNilSingleSignKeyGen, err)
}

func TestWithSingleSignKeyGenerator_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	keyGen := &mock.KeyGenMock{}

	opt := WithKeyGen(keyGen)
	err := opt(node)

	assert.True(t, node.keyGen == keyGen)
	assert.Nil(t, err)
}

func TestWithInitialNodesPubKeys(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	pubKeys := make(map[uint32][]string, 1)
	pubKeys[0] = []string{"pk1", "pk2", "pk3"}

	opt := WithInitialNodesPubKeys(pubKeys)
	err := opt(node)

	assert.Equal(t, pubKeys, node.initialNodesPubkeys)
	assert.Nil(t, err)
}

func TestWithPublicKey(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	pubKeys := make(map[uint32][]string, 1)
	pubKeys[0] = []string{"pk1", "pk2", "pk3"}

	opt := WithInitialNodesPubKeys(pubKeys)
	err := opt(node)

	assert.Equal(t, pubKeys, node.initialNodesPubkeys)
	assert.Nil(t, err)
}

func TestWithRoundDuration_ZeroDurationShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithRoundDuration(0)
	err := opt(node)

	assert.Equal(t, uint64(0), node.roundDuration)
	assert.Equal(t, ErrZeroRoundDurationNotSupported, err)
}

func TestWithRoundDuration_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	duration := uint64(5664)

	opt := WithRoundDuration(duration)
	err := opt(node)

	assert.True(t, node.roundDuration == duration)
	assert.Nil(t, err)
}

func TestWithConsensusGroupSize_NegativeGroupSizeShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithConsensusGroupSize(-1)
	err := opt(node)

	assert.Equal(t, 0, node.consensusGroupSize)
	assert.Equal(t, ErrNegativeOrZeroConsensusGroupSize, err)
}

func TestWithConsensusGroupSize_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	groupSize := 567

	opt := WithConsensusGroupSize(groupSize)
	err := opt(node)

	assert.True(t, node.consensusGroupSize == groupSize)
	assert.Nil(t, err)
}

func TestWithSyncer_NilSyncerShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithSyncer(nil)
	err := opt(node)

	assert.Nil(t, node.syncTimer)
	assert.Equal(t, ErrNilSyncTimer, err)
}

func TestWithSyncer_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	sync := &mock.SyncStub{}

	opt := WithSyncer(sync)
	err := opt(node)

	assert.True(t, node.syncTimer == sync)
	assert.Nil(t, err)
}

func TestWithRounder_NilRounderShouldErr(t *testing.T) {
	t.Parallel()
	node, _ := NewNode()
	opt := WithRounder(nil)
	err := opt(node)
	assert.Nil(t, node.rounder)
	assert.Equal(t, ErrNilRounder, err)
}

func TestWithRounder_ShouldWork(t *testing.T) {
	t.Parallel()
	node, _ := NewNode()
	rnd := &mock.RounderMock{}
	opt := WithRounder(rnd)
	err := opt(node)
	assert.True(t, node.rounder == rnd)
	assert.Nil(t, err)
}

func TestWithBlockProcessor_NilProcessorShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithBlockProcessor(nil)
	err := opt(node)

	assert.Nil(t, node.syncTimer)
	assert.Equal(t, ErrNilBlockProcessor, err)
}

func TestWithBlockProcessor_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	bp := &mock.BlockProcessorStub{}

	opt := WithBlockProcessor(bp)
	err := opt(node)

	assert.True(t, node.blockProcessor == bp)
	assert.Nil(t, err)
}

func TestWithGenesisTime(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	aTime := time.Time{}.Add(time.Duration(uint64(78)))

	opt := WithGenesisTime(aTime)
	err := opt(node)

	assert.Equal(t, node.genesisTime, aTime)
	assert.Nil(t, err)
}

func TestWithDataPool_NilDataPoolShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithDataPool(nil)
	err := opt(node)

	assert.Nil(t, node.dataPool)
	assert.Equal(t, ErrNilDataPool, err)
}

func TestWithDataPool_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	dataPool := &mock.PoolsHolderStub{}

	opt := WithDataPool(dataPool)
	err := opt(node)

	assert.True(t, node.dataPool == dataPool)
	assert.Nil(t, err)
}

func TestWithShardCoordinator_NilShardCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithShardCoordinator(nil)
	err := opt(node)

	assert.Nil(t, node.shardCoordinator)
	assert.Equal(t, ErrNilShardCoordinator, err)
}

func TestWithShardCoordinator_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	shardCoordinator := mock.NewOneShardCoordinatorMock()

	opt := WithShardCoordinator(shardCoordinator)
	err := opt(node)

	assert.True(t, node.shardCoordinator == shardCoordinator)
	assert.Nil(t, err)
}

func TestWithBlockTracker_NilBlockTrackerShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithBlockTracker(nil)
	err := opt(node)

	assert.Nil(t, node.blockTracker)
	assert.Equal(t, ErrNilBlockTracker, err)
}

func TestWithBlockTracker_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	blockTracker := &mock.BlockTrackerStub{}

	opt := WithBlockTracker(blockTracker)
	err := opt(node)

	assert.True(t, node.blockTracker == blockTracker)
	assert.Nil(t, err)
}

func TestWithPendingMiniBlocksHandler_NilPendingMiniBlocksHandlerShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithPendingMiniBlocksHandler(nil)
	err := opt(node)

	assert.Nil(t, node.pendingMiniBlocksHandler)
	assert.Equal(t, ErrNilPendingMiniBlocksHandler, err)
}

func TestWithPendingMiniBlocksHandler_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	pendingMiniBlocksHandler := &mock.PendingMiniBlocksHandlerStub{}

	opt := WithPendingMiniBlocksHandler(pendingMiniBlocksHandler)
	err := opt(node)

	assert.True(t, node.pendingMiniBlocksHandler == pendingMiniBlocksHandler)
	assert.Nil(t, err)
}

func TestWithRequestHandler_NilRequestHandlerShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithRequestHandler(nil)
	err := opt(node)

	assert.Nil(t, node.requestHandler)
	assert.Equal(t, ErrNilRequestHandler, err)
}

func TestWithRequestHandler_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	requestHandler := &mock.RequestHandlerStub{}

	opt := WithRequestHandler(requestHandler)
	err := opt(node)

	assert.True(t, node.requestHandler == requestHandler)
	assert.Nil(t, err)
}

func TestWithNodesCoordinator_NilNodesCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithNodesCoordinator(nil)
	err := opt(node)

	assert.Nil(t, node.nodesCoordinator)
	assert.Equal(t, ErrNilNodesCoordinator, err)
}

func TestWithNodesCoordinator_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	nodesCoordinator := &mock.NodesCoordinatorMock{}

	opt := WithNodesCoordinator(nodesCoordinator)
	err := opt(node)

	assert.True(t, node.nodesCoordinator == nodesCoordinator)
	assert.Nil(t, err)
}

func TestWithUint64ByteSliceConverter_NilConverterShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithUint64ByteSliceConverter(nil)
	err := opt(node)

	assert.Nil(t, node.uint64ByteSliceConverter)
	assert.Equal(t, ErrNilUint64ByteSliceConverter, err)
}

func TestWithUint64ByteSliceConverter_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	converter := mock.NewNonceHashConverterMock()

	opt := WithUint64ByteSliceConverter(converter)
	err := opt(node)

	assert.True(t, node.uint64ByteSliceConverter == converter)
	assert.Nil(t, err)
}

func TestWithSinglesig_NilBlsSinglesigShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithSingleSigner(nil)
	err := opt(node)

	assert.Nil(t, node.singleSigner)
	assert.Equal(t, ErrNilSingleSig, err)
}

func TestWithSinglesig_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	singlesigner := &mock.SinglesignMock{}

	opt := WithSingleSigner(singlesigner)
	err := opt(node)

	assert.True(t, node.singleSigner == singlesigner)
	assert.Nil(t, err)
}

func TestWithMultisig_NilMultisigShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithMultiSigner(nil)
	err := opt(node)

	assert.Nil(t, node.multiSigner)
	assert.Equal(t, ErrNilMultiSig, err)
}

func TestWithMultisig_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	multisigner := &mock.MultisignMock{}

	opt := WithMultiSigner(multisigner)
	err := opt(node)

	assert.True(t, node.multiSigner == multisigner)
	assert.Nil(t, err)
}

func TestWithForkDetector_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	forkDetector := &mock.ForkDetectorMock{}
	opt := WithForkDetector(forkDetector)
	err := opt(node)

	assert.True(t, node.forkDetector == forkDetector)
	assert.Nil(t, err)
}

func TestWithForkDetector_NilForkDetectorShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithForkDetector(nil)
	err := opt(node)

	assert.Nil(t, node.forkDetector)
	assert.Equal(t, ErrNilForkDetector, err)
}

func TestWithInterceptorsContainer_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	interceptorsContainer := &mock.InterceptorsContainerStub{}
	opt := WithInterceptorsContainer(interceptorsContainer)

	err := opt(node)

	assert.True(t, node.interceptorsContainer == interceptorsContainer)
	assert.Nil(t, err)
}

func TestWithInterceptorsContainer_NilContainerShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithInterceptorsContainer(nil)
	err := opt(node)

	assert.Nil(t, node.interceptorsContainer)
	assert.Equal(t, ErrNilInterceptorsContainer, err)
}

func TestWithResolversFinder_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	resolversFinder := &mock.ResolversFinderStub{}
	opt := WithResolversFinder(resolversFinder)

	err := opt(node)

	assert.True(t, node.resolversFinder == resolversFinder)
	assert.Nil(t, err)
}

func TestWithResolversContainer_NilContainerShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithResolversFinder(nil)
	err := opt(node)

	assert.Nil(t, node.resolversFinder)
	assert.Equal(t, ErrNilResolversFinder, err)
}

func TestWithConsensusBls_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	consensusType := "bls"
	opt := WithConsensusType(consensusType)
	err := opt(node)

	assert.Equal(t, consensusType, node.consensusType)
	assert.Nil(t, err)
}

func TestWithAppStatusHandler_NilAshShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithAppStatusHandler(nil)
	err := opt(node)

	assert.Equal(t, ErrNilStatusHandler, err)
}

func TestWithAppStatusHandler_OkAshShouldPass(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithAppStatusHandler(statusHandler.NewNilStatusHandler())
	err := opt(node)

	assert.IsType(t, &statusHandler.NilStatusHandler{}, node.appStatusHandler)
	assert.Nil(t, err)
}

func TestWithIndexer_ShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	indexer := &mock.IndexerMock{}
	opt := WithIndexer(indexer)
	err := opt(node)

	assert.Equal(t, indexer, node.indexer)
	assert.Nil(t, err)
}

func TestWithKeyGenForAccounts_NilKeygenShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithKeyGenForAccounts(nil)
	err := opt(node)

	assert.Equal(t, ErrNilKeyGenForBalances, err)
}

func TestWithKeyGenForAccounts_OkKeygenShouldPass(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	keyGen := &mock.KeyGenMock{}
	opt := WithKeyGenForAccounts(keyGen)
	err := opt(node)

	assert.True(t, node.keyGenForAccounts == keyGen)
	assert.Nil(t, err)
}

func TestWithTxFeeHandler_NilTxFeeHandlerShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithTxFeeHandler(nil)
	err := opt(node)

	assert.Equal(t, ErrNilTxFeeHandler, err)
}

func TestWithTxFeeHandler_NilBootStorerShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithBootStorer(nil)
	err := opt(node)

	assert.Equal(t, ErrNilBootStorer, err)
}

func TestWithTxFeeHandler_OkStorerShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	bootStorer := &mock.BoostrapStorerMock{}
	opt := WithBootStorer(bootStorer)
	err := opt(node)

	assert.Nil(t, err)
}

func TestWithTxFeeHandler_OkHandlerShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	txFeeHandler := &mock.FeeHandlerStub{}
	opt := WithTxFeeHandler(txFeeHandler)
	err := opt(node)

	assert.True(t, node.feeHandler == txFeeHandler)
	assert.Nil(t, err)
}

func TestWithRequestedItemsHandler_NilRequestedItemsHandlerShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithRequestedItemsHandler(nil)
	err := opt(node)

	assert.Equal(t, ErrNilRequestedItemsHandler, err)
}

func TestWithHeaderSigVerifier_NilHeaderSigVerifierShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithHeaderSigVerifier(nil)
	err := opt(node)

	assert.Equal(t, ErrNilHeaderSigVerifier, err)
}

func TestWithHeaderSigVerifier_OkHeaderSigVerfierShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithHeaderSigVerifier(&mock.HeaderSigVerifierStub{})
	err := opt(node)

	assert.Nil(t, err)
}

func TestWithRequestedItemsHandler_OkRequestedItemsHandlerShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	requestedItemsHeanlder := &mock.RequestedItemsHandlerStub{}
	opt := WithRequestedItemsHandler(requestedItemsHeanlder)
	err := opt(node)

	assert.True(t, node.requestedItemsHandler == requestedItemsHeanlder)
	assert.Nil(t, err)
}

func TestWithValidatorStatistics_NilValidatorStatisticsShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithValidatorStatistics(nil)
	err := opt(node)

	assert.Equal(t, ErrNilValidatorStatistics, err)
}

func TestWithValidatorStatistics_OkValidatorStatisticsShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithValidatorStatistics(&mock.ValidatorStatisticsProcessorStub{})
	err := opt(node)

	assert.Nil(t, err)
}

func TestWithChainID_InvalidShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()
	opt := WithChainID(nil)

	err := opt(node)
	assert.Equal(t, ErrInvalidChainID, err)
}

func TestWithChainID_OkValueShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()
	chainId := []byte("chain ID")
	opt := WithChainID(chainId)

	err := opt(node)
	assert.Equal(t, node.chainID, chainId)
	assert.Nil(t, err)
}

func TestWithBootstrapRoundIndex(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()
	roundIndex := uint64(0)
	opt := WithBootstrapRoundIndex(roundIndex)

	err := opt(node)
	assert.Equal(t, roundIndex, node.bootstrapRoundIndex)
	assert.Nil(t, err)
}

func TestWithTxStorageSize(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()
	txStorageSize := uint32(100)
	opt := WithTxStorageSize(txStorageSize)

	err := opt(node)
	assert.Equal(t, txStorageSize, node.txStorageSize)
	assert.Nil(t, err)
}

func TestWithBlackListHandler_NilBlackListHandler(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()
	opt := WithBlackListHandler(nil)

	err := opt(node)
	assert.Equal(t, ErrNilBlackListHandler, err)
}

func TestWithEpochStartTrigger_NilEpoch(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()
	opt := WithEpochStartTrigger(nil)

	err := opt(node)
	assert.Equal(t, ErrNilEpochStartTrigger, err)
}

func TestWithTxSingleSigner_NilTxSingleSigner(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()
	opt := WithTxSingleSigner(nil)

	err := opt(node)
	assert.Equal(t, ErrNilSingleSig, err)
}

func TestWithPubKey_NilPublicKey(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()
	opt := WithPubKey(nil)

	err := opt(node)
	assert.Equal(t, ErrNilPublicKey, err)
}

func TestWithBlackListHandler_NilBlackListHandlerShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithBlackListHandler(nil)
	err := opt(node)

	assert.Equal(t, ErrNilBlackListHandler, err)
}

func TestWithBlackListHandler_OkHandlerShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	blackListHandler := &mock.BlackListHandlerStub{}
	opt := WithBlackListHandler(blackListHandler)
	err := opt(node)

	assert.True(t, node.blackListHandler == blackListHandler)
	assert.Nil(t, err)
}

func TestWithNetworkShardingCollector_NilNetworkShardingCollectorShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithNetworkShardingCollector(nil)
	err := opt(node)

	assert.Equal(t, ErrNilNetworkShardingCollector, err)
}

func TestWithNetworkShardingCollector_OkHandlerShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	networkShardingCollector := &mock.NetworkShardingCollectorStub{}
	opt := WithNetworkShardingCollector(networkShardingCollector)
	err := opt(node)

	assert.True(t, node.networkShardingCollector == networkShardingCollector)
	assert.Nil(t, err)
}

func TestWithInputAntifloodHandler_NilAntifloodHandlerShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithInputAntifloodHandler(nil)
	err := opt(node)

	assert.True(t, errors.Is(err, ErrNilAntifloodHandler))
}

func TestWithInputAntifloodHandler_OkAntifloodHandlerShouldWork(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	antifloodHandler := &mock.P2PAntifloodHandlerStub{}
	opt := WithInputAntifloodHandler(antifloodHandler)
	err := opt(node)

	assert.True(t, node.inputAntifloodHandler == antifloodHandler)
	assert.Nil(t, err)
}

func TestWithTxAccumulator_NilAccumulatorShouldErr(t *testing.T) {
	t.Parallel()

	node, _ := NewNode()

	opt := WithTxAccumulator(nil)
	err := opt(node)

	assert.Equal(t, ErrNilTxAccumulator, err)
}
