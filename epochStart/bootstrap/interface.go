package bootstrap

import (
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// StartOfEpochNodesConfigHandler defines the methods to process nodesConfig from epoch start metablocks
type StartOfEpochNodesConfigHandler interface {
	NodesConfigFromMetaBlock(
		currMetaBlock *block.MetaBlock,
		prevMetaBlock *block.MetaBlock,
		publicKey []byte,
	) (*sharding.NodesCoordinatorRegistry, uint32, error)
	IsInterfaceNil() bool
}

// EpochStartInterceptor -
type EpochStartInterceptor interface {
	process.Interceptor
	GetEpochStartMetaBlock(target int, epoch uint32) (*block.MetaBlock, error)
}

// EpochStartNodesCoordinator -
type EpochStartNodesCoordinator interface {
	ComputeNodesConfigFor(
		metaBlock *block.MetaBlock,
		validatorInfos []*state.ValidatorInfo,
	) (*sharding.EpochValidators, error)
	ComputeNodesConfigForGenesis(genesis *sharding.NodesSetup) (*sharding.EpochValidators, error)
	ComputeShardForSelfPublicKey(epoch uint32, pubKey []byte) uint32
	IsInterfaceNil() bool
}

// Messenger defines which methods a p2p messenger should implement
type Messenger interface {
	dataRetriever.MessageHandler
	dataRetriever.TopicHandler
	UnregisterMessageProcessor(topic string) error
	UnregisterAllMessageProcessors() error
	ConnectedPeers() []p2p.PeerID
}

// RequestHandler defines which methods a request handler should implement
type RequestHandler interface {
	RequestStartOfEpochMetaBlock(epoch uint32)
	SetNumPeersToQuery(topic string, intra int, cross int) error
	GetNumPeersToQuery(topic string) (int, int, error)
	IsInterfaceNil() bool
}
