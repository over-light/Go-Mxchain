package integrationTests

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
)

// ArgIndexHashedNodesCoordinatorFactory -
type ArgIndexHashedNodesCoordinatorFactory struct {
	nodesPerShard           int
	nbMetaNodes             int
	shardConsensusGroupSize int
	metaConsensusGroupSize  int
	shardId                 uint32
	nbShards                int
	validatorsMap           map[uint32][]sharding.Validator
	waitingMap              map[uint32][]sharding.Validator
	keyIndex                int
	cp                      *CryptoParams
	epochStartSubscriber    sharding.EpochStartSubscriber
	hasher                  hashing.Hasher
	consensusGroupCache     sharding.Cacher
	bootStorer              storage.Storer
}

// IndexHashedNodesCoordinatorFactory -
type IndexHashedNodesCoordinatorFactory struct {
}

// CreateNodesCoordinator -
func (tpn *IndexHashedNodesCoordinatorFactory) CreateNodesCoordinator(arg ArgIndexHashedNodesCoordinatorFactory) sharding.NodesCoordinator {

	nodeKeys := arg.cp.Keys[arg.shardId][arg.keyIndex]
	pubKeyBytes, _ := nodeKeys.Pk.ToByteArray()

	nodeShuffler := sharding.NewXorValidatorsShuffler(uint32(arg.nodesPerShard), uint32(arg.nbMetaNodes), 0.2, false)
	argumentsNodesCoordinator := sharding.ArgNodesCoordinator{
		ShardConsensusGroupSize: arg.shardConsensusGroupSize,
		MetaConsensusGroupSize:  arg.metaConsensusGroupSize,
		Hasher:                  arg.hasher,
		Shuffler:                nodeShuffler,
		EpochStartSubscriber:    arg.epochStartSubscriber,
		ShardId:                 arg.shardId,
		NbShards:                uint32(arg.nbShards),
		EligibleNodes:           arg.validatorsMap,
		WaitingNodes:            arg.waitingMap,
		SelfPublicKey:           pubKeyBytes,
		ConsensusGroupCache:     arg.consensusGroupCache,
		BootStorer:              arg.bootStorer,
	}
	nodesCoordinator, err := sharding.NewIndexHashedNodesCoordinator(argumentsNodesCoordinator)
	if err != nil {
		fmt.Println("Error creating node coordinator")
	}

	return nodesCoordinator
}

// IndexHashedNodesCoordinatorWithRaterFactory -
type IndexHashedNodesCoordinatorWithRaterFactory struct {
	sharding.RaterHandler
}

// CreateNodesCoordinator is used for creating a nodes coordinator in the integration tests
// based on the provided parameters
func (ihncrf *IndexHashedNodesCoordinatorWithRaterFactory) CreateNodesCoordinator(
	arg ArgIndexHashedNodesCoordinatorFactory,
) sharding.NodesCoordinator {
	nodeKeys := arg.cp.Keys[arg.shardId][arg.keyIndex]
	pubKeyBytes, _ := nodeKeys.Pk.ToByteArray()

	nodeShuffler := sharding.NewXorValidatorsShuffler(uint32(arg.nodesPerShard), uint32(arg.nbMetaNodes), 0.2, false)
	argumentsNodesCoordinator := sharding.ArgNodesCoordinator{
		ShardConsensusGroupSize: arg.shardConsensusGroupSize,
		MetaConsensusGroupSize:  arg.metaConsensusGroupSize,
		Hasher:                  arg.hasher,
		Shuffler:                nodeShuffler,
		EpochStartSubscriber:    arg.epochStartSubscriber,
		ShardId:                 arg.shardId,
		NbShards:                uint32(arg.nbShards),
		EligibleNodes:           arg.validatorsMap,
		WaitingNodes:            arg.waitingMap,
		SelfPublicKey:           pubKeyBytes,
		ConsensusGroupCache:     arg.consensusGroupCache,
		BootStorer:              arg.bootStorer,
	}

	baseCoordinator, err := sharding.NewIndexHashedNodesCoordinator(argumentsNodesCoordinator)
	if err != nil {
		log.Debug("Error creating node coordinator")
	}

	nodesCoordinator, err := sharding.NewIndexHashedNodesCoordinatorWithRater(baseCoordinator, ihncrf.RaterHandler)
	if err != nil {
		log.Debug("Error creating node coordinator")
	}

	return &NodesWithRater{
		NodesCoordinator: nodesCoordinator,
		RaterHandler:     ihncrf.RaterHandler,
	}
}

// NodesWithRater -
type NodesWithRater struct {
	sharding.NodesCoordinator
	sharding.RaterHandler
}

// IsInterfaceNil -
func (nwr *NodesWithRater) IsInterfaceNil() bool {
	return nwr == nil
}
