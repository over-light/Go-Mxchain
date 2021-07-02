package sharding

import (
	"github.com/ElrondNetwork/elrond-go/data/endProcess"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/storage"
)

// ArgNodesCoordinator holds all dependencies required by the nodes coordinator in order to create new instances
type ArgNodesCoordinator struct {
	ShardConsensusGroupSize    int
	MetaConsensusGroupSize     int
	Marshalizer                marshal.Marshalizer
	Hasher                     hashing.Hasher
	Shuffler                   NodesShuffler
	EpochStartNotifier         EpochStartEventNotifier
	BootStorer                 storage.Storer
	ShardIDAsObserver          uint32
	NbShards                   uint32
	EligibleNodes              map[uint32][]Validator
	WaitingNodes               map[uint32][]Validator
	SelfPublicKey              []byte
	Epoch                      uint32
	StartEpoch                 uint32
	ConsensusGroupCache        Cacher
	ShuffledOutHandler         ShuffledOutHandler
	WaitingListFixEnabledEpoch uint32
	ChanStopNode               chan endProcess.ArgEndProcess
	IsFullArchive              bool
}
