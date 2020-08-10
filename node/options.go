package node

import (
	"fmt"
	"time"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/fullHistory"
	"github.com/ElrondNetwork/elrond-go/data/endProcess"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	mainFactory "github.com/ElrondNetwork/elrond-go/factory"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
)

// WithBootstrapComponents sets up the Node bootstrap components
func WithBootstrapComponents(bootstrapComponents mainFactory.BootstrapComponentsHandler) Option {
	return func(n *Node) error {
		if check.IfNil(bootstrapComponents) {
			return ErrNilBootstrapComponents
		}
		err := bootstrapComponents.CheckSubcomponents()
		if err != nil {
			return err
		}
		n.bootstrapComponents = bootstrapComponents
		return nil
	}
}

// WithCoreComponents sets up the Node core components
func WithCoreComponents(coreComponents mainFactory.CoreComponentsHandler) Option {
	return func(n *Node) error {
		if check.IfNil(coreComponents) {
			return ErrNilCoreComponents
		}
		err := coreComponents.CheckSubcomponents()
		if err != nil {
			return err
		}
		n.coreComponents = coreComponents
		return nil
	}
}

// WithCryptoComponents sets up Node crypto components
func WithCryptoComponents(cryptoComponents mainFactory.CryptoComponentsHandler) Option {
	return func(n *Node) error {
		if check.IfNil(cryptoComponents) {
			return ErrNilCryptoComponents
		}
		err := cryptoComponents.CheckSubcomponents()
		if err != nil {
			return err
		}
		n.cryptoComponents = cryptoComponents
		return nil
	}
}

// WithDataComponents sets up the Node data components
func WithDataComponents(dataComponents mainFactory.DataComponentsHandler) Option {
	return func(n *Node) error {
		if check.IfNil(dataComponents) {
			return ErrNilDataComponents
		}
		err := dataComponents.CheckSubcomponents()
		if err != nil {
			return err
		}
		n.dataComponents = dataComponents
		return nil
	}
}

// WithNetworkComponents sets up the Node network components
func WithNetworkComponents(networkComponents mainFactory.NetworkComponentsHandler) Option {
	return func(n *Node) error {
		if check.IfNil(networkComponents) {
			return ErrNilNetworkComponents
		}
		err := networkComponents.CheckSubcomponents()
		if err != nil {
			return err
		}
		n.networkComponents = networkComponents
		return nil
	}
}

// WithProcessComponents sets up the Node process components
func WithProcessComponents(processComponents mainFactory.ProcessComponentsHandler) Option {
	return func(n *Node) error {
		if check.IfNil(processComponents) {
			return ErrNilProcessComponents
		}
		err := processComponents.CheckSubcomponents()
		if err != nil {
			return err
		}
		n.processComponents = processComponents
		return nil
	}
}

// WithStateComponents sets up the Node state components
func WithStateComponents(stateComponents mainFactory.StateComponentsHandler) Option {
	return func(n *Node) error {
		if check.IfNil(stateComponents) {
			return ErrNilStateComponents
		}
		err := stateComponents.CheckSubcomponents()
		if err != nil {
			return err
		}
		n.stateComponents = stateComponents
		return nil
	}
}

// WithStatusComponents sets up the Node status components
func WithStatusComponents(statusComponents mainFactory.StatusComponentsHandler) Option {
	return func(n *Node) error {
		if check.IfNil(statusComponents) {
			return ErrNilStatusComponents
		}
		err := statusComponents.CheckSubcomponents()
		if err != nil {
			return err
		}
		n.statusComponents = statusComponents
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

// WithGenesisTime sets up the genesis time option for the Node
func WithGenesisTime(genesisTime time.Time) Option {
	return func(n *Node) error {
		n.genesisTime = genesisTime
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

// WithPeerDenialEvaluator sets up a peer denial evaluator for the Node
func WithPeerDenialEvaluator(handler p2p.PeerDenialEvaluator) Option {
	return func(n *Node) error {
		if check.IfNil(handler) {
			return fmt.Errorf("%w for WithPeerDenialEvaluator", ErrNilPeerDenialEvaluator)
		}
		n.peerDenialEvaluator = handler
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

// WithHistoryRepository sets up a history repository for the node
func WithHistoryRepository(historyRepo fullHistory.HistoryRepository) Option {
	return func(n *Node) error {
		if check.IfNil(historyRepo) {
			return ErrNilHistoryRepository
		}
		n.historyRepository = historyRepo
		return nil
	}
}
