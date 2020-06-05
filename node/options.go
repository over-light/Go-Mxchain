package node

import (
	"fmt"
	"time"

	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/consensus/spos"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/indexer"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/endProcess"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/ntp"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// WithMessenger sets up the messenger option for the Node
func WithMessenger(mes P2PMessenger) Option {
	return func(n *Node) error {
		if check.IfNil(mes) {
			return ErrNilMessenger
		}
		n.messenger = mes
		return nil
	}
}

// WithInternalMarshalizer sets up the marshalizer option for the Node
func WithInternalMarshalizer(marshalizer marshal.Marshalizer, sizeCheckDelta uint32) Option {
	return func(n *Node) error {
		if check.IfNil(marshalizer) {
			return ErrNilMarshalizer
		}
		n.sizeCheckDelta = sizeCheckDelta
		n.internalMarshalizer = marshalizer
		return nil
	}
}

// WithVmMarshalizer sets up the marshalizer used in Vm communication (SC)
func WithVmMarshalizer(marshalizer marshal.Marshalizer) Option {
	return func(n *Node) error {
		if check.IfNil(marshalizer) {
			return ErrNilMarshalizer
		}
		n.vmMarshalizer = marshalizer
		return nil
	}
}

// WithTxSignMarshalizer sets up the marshalizer used in transaction singning
func WithTxSignMarshalizer(marshalizer marshal.Marshalizer) Option {
	return func(n *Node) error {
		if check.IfNil(marshalizer) {
			return ErrNilMarshalizer
		}
		n.txSignMarshalizer = marshalizer
		return nil
	}
}

// WithHasher sets up the hasher option for the Node
func WithHasher(hasher hashing.Hasher) Option {
	return func(n *Node) error {
		if check.IfNil(hasher) {
			return ErrNilHasher
		}
		n.hasher = hasher
		return nil
	}
}

// WithTxFeeHandler sets up the tx fee handler for the Node
func WithTxFeeHandler(feeHandler process.FeeHandler) Option {
	return func(n *Node) error {
		if check.IfNil(feeHandler) {
			return ErrNilTxFeeHandler
		}
		n.feeHandler = feeHandler
		return nil
	}
}

// WithAccountsAdapter sets up the accounts adapter option for the Node
func WithAccountsAdapter(accounts state.AccountsAdapter) Option {
	return func(n *Node) error {
		if check.IfNil(accounts) {
			return ErrNilAccountsAdapter
		}
		n.accounts = accounts
		return nil
	}
}

// WithAddressPubkeyConverter sets up the address public key converter adapter option for the Node
func WithAddressPubkeyConverter(pubkeyConverter core.PubkeyConverter) Option {
	return func(n *Node) error {
		if check.IfNil(pubkeyConverter) {
			return fmt.Errorf("%w in WithAddressPubkeyConverter", ErrNilPubkeyConverter)
		}
		n.addressPubkeyConverter = pubkeyConverter
		return nil
	}
}

// WithValidatorPubkeyConverter sets up the validator public key converter adapter option for the Node
func WithValidatorPubkeyConverter(pubkeyConverter core.PubkeyConverter) Option {
	return func(n *Node) error {
		if check.IfNil(pubkeyConverter) {
			return fmt.Errorf("%w in WithValidatorPubkeyConverter", ErrNilPubkeyConverter)
		}
		n.validatorPubkeyConverter = pubkeyConverter
		return nil
	}
}

// WithBlockChain sets up the blockchain option for the Node
func WithBlockChain(blkc data.ChainHandler) Option {
	return func(n *Node) error {
		if check.IfNil(blkc) {
			return ErrNilBlockchain
		}
		n.blkc = blkc
		return nil
	}
}

// WithDataStore sets up the storage options for the Node
func WithDataStore(store dataRetriever.StorageService) Option {
	return func(n *Node) error {
		if store == nil || store.IsInterfaceNil() {
			return ErrNilStore
		}
		n.store = store
		return nil
	}
}

// WithPubKey sets up the multi sign pub key option for the Node
func WithPubKey(pk crypto.PublicKey) Option {
	return func(n *Node) error {
		if check.IfNil(pk) {
			return ErrNilPublicKey
		}
		n.pubKey = pk
		return nil
	}
}

// WithPrivKey sets up the multi sign private key option for the Node
func WithPrivKey(sk crypto.PrivateKey) Option {
	return func(n *Node) error {
		if check.IfNil(sk) {
			return ErrNilPrivateKey
		}
		n.privKey = sk
		return nil
	}
}

// WithKeyGen sets up the single sign key generator option for the Node
func WithKeyGen(keyGen crypto.KeyGenerator) Option {
	return func(n *Node) error {
		if check.IfNil(keyGen) {
			return ErrNilSingleSignKeyGen
		}
		n.keyGen = keyGen
		return nil
	}
}

// WithKeyGenForAccounts sets up the balances key generator option for the Node
func WithKeyGenForAccounts(keyGenForAccounts crypto.KeyGenerator) Option {
	return func(n *Node) error {
		if check.IfNil(keyGenForAccounts) {
			return ErrNilKeyGenForBalances
		}
		n.keyGenForAccounts = keyGenForAccounts
		return nil
	}
}

// WithInitialNodesPubKeys sets up the initial nodes public key option for the Node
func WithInitialNodesPubKeys(pubKeys map[uint32][]string) Option {
	return func(n *Node) error {
		n.initialNodesPubkeys = pubKeys
		return nil
	}
}

// WithRoundDuration sets up the round duration option for the Node
func WithRoundDuration(roundDuration uint64) Option {
	return func(n *Node) error {
		if roundDuration == 0 {
			return ErrZeroRoundDurationNotSupported
		}
		n.roundDuration = roundDuration
		return nil
	}
}

// WithConsensusGroupSize sets up the consensus group size option for the Node
func WithConsensusGroupSize(consensusGroupSize int) Option {
	return func(n *Node) error {
		if consensusGroupSize < 1 {
			return ErrNegativeOrZeroConsensusGroupSize
		}
		n.consensusGroupSize = consensusGroupSize
		return nil
	}
}

// WithSyncer sets up the syncTimer option for the Node
func WithSyncer(syncer ntp.SyncTimer) Option {
	return func(n *Node) error {
		if check.IfNil(syncer) {
			return ErrNilSyncTimer
		}
		n.syncTimer = syncer
		return nil
	}
}

// WithRounder sets up the rounder option for the Node
func WithRounder(rounder consensus.Rounder) Option {
	return func(n *Node) error {
		if check.IfNil(rounder) {
			return ErrNilRounder
		}
		n.rounder = rounder
		return nil
	}
}

// WithBlockProcessor sets up the block processor option for the Node
func WithBlockProcessor(blockProcessor process.BlockProcessor) Option {
	return func(n *Node) error {
		if check.IfNil(blockProcessor) {
			return ErrNilBlockProcessor
		}
		n.blockProcessor = blockProcessor
		return nil
	}
}

// WithGenesisTime sets up the genesis time option for the Node
func WithGenesisTime(genesisTime time.Time) Option {
	return func(n *Node) error {
		n.genesisTime = genesisTime
		return nil
	}
}

// WithDataPool sets up the data pools option for the Node
func WithDataPool(dataPool dataRetriever.PoolsHolder) Option {
	return func(n *Node) error {
		if check.IfNil(dataPool) {
			return ErrNilDataPool
		}
		n.dataPool = dataPool
		return nil
	}
}

// WithShardCoordinator sets up the shard coordinator for the Node
func WithShardCoordinator(shardCoordinator sharding.Coordinator) Option {
	return func(n *Node) error {
		if check.IfNil(shardCoordinator) {
			return ErrNilShardCoordinator
		}
		n.shardCoordinator = shardCoordinator
		return nil
	}
}

// WithNodesCoordinator sets up the nodes coordinator
func WithNodesCoordinator(nodesCoordinator sharding.NodesCoordinator) Option {
	return func(n *Node) error {
		if check.IfNil(nodesCoordinator) {
			return ErrNilNodesCoordinator
		}
		n.nodesCoordinator = nodesCoordinator
		return nil
	}
}

// WithUint64ByteSliceConverter sets up the uint64 <-> []byte converter
func WithUint64ByteSliceConverter(converter typeConverters.Uint64ByteSliceConverter) Option {
	return func(n *Node) error {
		if check.IfNil(converter) {
			return ErrNilUint64ByteSliceConverter
		}
		n.uint64ByteSliceConverter = converter
		return nil
	}
}

// WithSingleSigner sets up a singleSigner option for the Node
func WithSingleSigner(singleSigner crypto.SingleSigner) Option {
	return func(n *Node) error {
		if check.IfNil(singleSigner) {
			return ErrNilSingleSig
		}
		n.singleSigner = singleSigner
		return nil
	}
}

// WithTxSingleSigner sets up a txSingleSigner option for the Node
func WithTxSingleSigner(txSingleSigner crypto.SingleSigner) Option {
	return func(n *Node) error {
		if check.IfNil(txSingleSigner) {
			return ErrNilSingleSig
		}
		n.txSingleSigner = txSingleSigner
		return nil
	}
}

// WithMultiSigner sets up the multiSigner option for the Node
func WithMultiSigner(multiSigner crypto.MultiSigner) Option {
	return func(n *Node) error {
		if check.IfNil(multiSigner) {
			return ErrNilMultiSig
		}
		n.multiSigner = multiSigner
		return nil
	}
}

// WithForkDetector sets up the multiSigner option for the Node
func WithForkDetector(forkDetector process.ForkDetector) Option {
	return func(n *Node) error {
		if check.IfNil(forkDetector) {
			return ErrNilForkDetector
		}
		n.forkDetector = forkDetector
		return nil
	}
}

// WithInterceptorsContainer sets up the interceptors container option for the Node
func WithInterceptorsContainer(interceptorsContainer process.InterceptorsContainer) Option {
	return func(n *Node) error {
		if check.IfNil(interceptorsContainer) {
			return ErrNilInterceptorsContainer
		}
		n.interceptorsContainer = interceptorsContainer
		return nil
	}
}

// WithResolversFinder sets up the resolvers finder option for the Node
func WithResolversFinder(resolversFinder dataRetriever.ResolversFinder) Option {
	return func(n *Node) error {
		if check.IfNil(resolversFinder) {
			return ErrNilResolversFinder
		}
		n.resolversFinder = resolversFinder
		return nil
	}
}

// WithConsensusType sets up the consensus type option for the Node
func WithConsensusType(consensusType string) Option {
	return func(n *Node) error {
		n.consensusType = consensusType
		return nil
	}
}

// WithBootstrapRoundIndex sets up a bootstrapRoundIndex option for the Node
func WithBootstrapRoundIndex(bootstrapRoundIndex uint64) Option {
	return func(n *Node) error {
		n.bootstrapRoundIndex = bootstrapRoundIndex
		return nil
	}
}

// WithEpochStartTrigger sets up an start of epoch trigger option for the node
func WithEpochStartTrigger(epochStartTrigger epochStart.TriggerHandler) Option {
	return func(n *Node) error {
		if check.IfNil(epochStartTrigger) {
			return ErrNilEpochStartTrigger
		}
		n.epochStartTrigger = epochStartTrigger
		return nil
	}
}

// WithEpochStartEventNotifier sets up the notifier for the epoch start event
func WithEpochStartEventNotifier(epochStartEventNotifier epochStart.RegistrationHandler) Option {
	return func(n *Node) error {
		if epochStartEventNotifier == nil {
			return ErrNilEpochStartTrigger
		}
		n.epochStartRegistrationHandler = epochStartEventNotifier
		return nil
	}
}

// WithAppStatusHandler sets up which handler will monitor the status of the node
func WithAppStatusHandler(aph core.AppStatusHandler) Option {
	return func(n *Node) error {
		if check.IfNil(aph) {
			return ErrNilStatusHandler

		}
		n.appStatusHandler = aph
		return nil
	}
}

// WithIndexer sets up a indexer for the Node
func WithIndexer(indexer indexer.Indexer) Option {
	return func(n *Node) error {
		n.indexer = indexer
		return nil
	}
}

// WithBlockBlackListHandler sets up a block black list handler for the Node
func WithBlockBlackListHandler(blackListHandler process.BlackListHandler) Option {
	return func(n *Node) error {
		if check.IfNil(blackListHandler) {
			return fmt.Errorf("%w for WithBlockBlackListHandler", ErrNilBlackListHandler)
		}
		n.blocksBlackListHandler = blackListHandler
		return nil
	}
}

// WithPeerBlackListHandler sets up a block black list handler for the Node
func WithPeerBlackListHandler(blackListHandler process.BlackListHandler) Option {
	return func(n *Node) error {
		if check.IfNil(blackListHandler) {
			return fmt.Errorf("%w for WithPeerBlackListHandler", ErrNilBlackListHandler)
		}
		n.peerBlackListHandler = blackListHandler
		return nil
	}
}

// WithBootStorer sets up a boot storer for the Node
func WithBootStorer(bootStorer process.BootStorer) Option {
	return func(n *Node) error {
		if check.IfNil(bootStorer) {
			return ErrNilBootStorer
		}
		n.bootStorer = bootStorer
		return nil
	}
}

// WithRequestedItemsHandler sets up a requested items handler for the Node
func WithRequestedItemsHandler(requestedItemsHandler dataRetriever.RequestedItemsHandler) Option {
	return func(n *Node) error {
		if check.IfNil(requestedItemsHandler) {
			return ErrNilRequestedItemsHandler
		}
		n.requestedItemsHandler = requestedItemsHandler
		return nil
	}
}

// WithHeaderSigVerifier sets up a header sig verifier for the Node
func WithHeaderSigVerifier(headerSigVerifier spos.RandSeedVerifier) Option {
	return func(n *Node) error {
		if check.IfNil(headerSigVerifier) {
			return ErrNilHeaderSigVerifier
		}
		n.headerSigVerifier = headerSigVerifier
		return nil
	}
}

// WithHeaderIntegrityVerifier sets up a header integrity verifier for the Node
func WithHeaderIntegrityVerifier(headerIntegrityVerifier spos.HeaderIntegrityVerifier) Option {
	return func(n *Node) error {
		if check.IfNil(headerIntegrityVerifier) {
			return ErrNilHeaderIntegrityVerifier
		}
		n.headerIntegrityVerifier = headerIntegrityVerifier
		return nil
	}
}

// WithValidatorStatistics sets up the validator statistics for the node
func WithValidatorStatistics(validatorStatistics process.ValidatorStatisticsProcessor) Option {
	return func(n *Node) error {
		if check.IfNil(validatorStatistics) {
			return ErrNilValidatorStatistics
		}
		n.validatorStatistics = validatorStatistics
		return nil
	}
}

// WithValidatorsProvider sets up the validators provider for the node
func WithValidatorsProvider(validatorsProvider process.ValidatorsProvider) Option {
	return func(n *Node) error {
		if check.IfNil(validatorsProvider) {
			return ErrNilValidatorStatistics
		}
		n.validatorsProvider = validatorsProvider
		return nil
	}
}

// WithChainID sets up the chain ID on which the current node is supposed to work on
func WithChainID(chainID []byte) Option {
	return func(n *Node) error {
		if len(chainID) == 0 {
			return ErrInvalidChainID
		}
		n.chainID = chainID

		return nil
	}
}

// WithBlockTracker sets up the block tracker for the Node
func WithBlockTracker(blockTracker process.BlockTracker) Option {
	return func(n *Node) error {
		if check.IfNil(blockTracker) {
			return ErrNilBlockTracker
		}
		n.blockTracker = blockTracker
		return nil
	}
}

// WithPendingMiniBlocksHandler sets up the pending miniblocks handler for the Node
func WithPendingMiniBlocksHandler(pendingMiniBlocksHandler process.PendingMiniBlocksHandler) Option {
	return func(n *Node) error {
		if check.IfNil(pendingMiniBlocksHandler) {
			return ErrNilPendingMiniBlocksHandler
		}
		n.pendingMiniBlocksHandler = pendingMiniBlocksHandler
		return nil
	}
}

// WithRequestHandler sets up the request handler for the Node
func WithRequestHandler(requestHandler process.RequestHandler) Option {
	return func(n *Node) error {
		if check.IfNil(requestHandler) {
			return ErrNilRequestHandler
		}
		n.requestHandler = requestHandler
		return nil
	}
}

// WithNetworkShardingCollector sets up a network sharding updater for the Node
func WithNetworkShardingCollector(networkShardingCollector NetworkShardingCollector) Option {
	return func(n *Node) error {
		if check.IfNil(networkShardingCollector) {
			return ErrNilNetworkShardingCollector
		}
		n.networkShardingCollector = networkShardingCollector
		return nil
	}
}

// WithInputAntifloodHandler sets up an antiflood handler for the Node on the input side
func WithInputAntifloodHandler(antifloodHandler P2PAntifloodHandler) Option {
	return func(n *Node) error {
		if check.IfNil(antifloodHandler) {
			return fmt.Errorf("%w on input", ErrNilAntifloodHandler)
		}
		n.inputAntifloodHandler = antifloodHandler
		return nil
	}
}

// WithTxAccumulator sets up a transaction accumulator handler for the Node
func WithTxAccumulator(accumulator Accumulator) Option {
	return func(n *Node) error {
		if check.IfNil(accumulator) {
			return ErrNilTxAccumulator
		}
		n.txAcumulator = accumulator

		go n.sendFromTxAccumulator()
		go n.printTxSentCounter()

		return nil
	}
}

// WithHardforkTrigger sets up a hardfork trigger
func WithHardforkTrigger(hardforkTrigger HardforkTrigger) Option {
	return func(n *Node) error {
		if check.IfNil(hardforkTrigger) {
			return ErrNilHardforkTrigger
		}

		n.hardforkTrigger = hardforkTrigger

		return nil
	}
}

// WithWhiteListHandler sets up a white list handler option
func WithWhiteListHandler(whiteListHandler process.WhiteListHandler) Option {
	return func(n *Node) error {
		if check.IfNil(whiteListHandler) {
			return ErrNilWhiteListHandler
		}

		n.whiteListRequest = whiteListHandler

		return nil
	}
}

// WithWhiteListHandlerVerified sets up a white list handler option
func WithWhiteListHandlerVerified(whiteListHandler process.WhiteListHandler) Option {
	return func(n *Node) error {
		if check.IfNil(whiteListHandler) {
			return ErrNilWhiteListHandler
		}

		n.whiteListerVerifiedTxs = whiteListHandler

		return nil
	}
}

// WithSignatureSize sets up a signatureSize option for the Node
func WithSignatureSize(signatureSize int) Option {
	return func(n *Node) error {
		n.signatureSize = signatureSize
		return nil
	}
}

// WithPublicKeySize sets up a publicKeySize option for the Node
func WithPublicKeySize(publicKeySize int) Option {
	return func(n *Node) error {
		n.publicKeySize = publicKeySize
		return nil
	}
}

// WithNodeStopChannel sets up the channel which will handle closing the node
func WithNodeStopChannel(channel chan endProcess.ArgEndProcess) Option {
	return func(n *Node) error {
		if channel == nil {
			return ErrNilNodeStopChannel
		}
		n.chanStopNodeProcess = channel

		return nil
	}
}

// WithApiTransactionByHashThrottler sets up the api transaction by hash throttler
func WithApiTransactionByHashThrottler(throttler Throttler) Option {
	return func(n *Node) error {
		if throttler == nil {
			return ErrNilApiTransactionByHashThrottler
		}
		n.apiTransactionByHashThrottler = throttler
		return nil
	}
}
