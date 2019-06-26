package sharding

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go/data/state"
)

// MetachainShardId will be used to identify a shard ID as metachain
const MetachainShardId = uint32(0xFFFFFFFF)

// Coordinator defines what a shard state coordinator should hold
type Coordinator interface {
	NumberOfShards() uint32
	ComputeId(address state.AddressContainer) uint32
	SelfId() uint32
	SameShard(firstAddress, secondAddress state.AddressContainer) bool
	CommunicationIdentifier(destShardID uint32) string
}

// Validator defines a node that can be allocated to a shard for participation in a consensus group as validator
// or block proposer
type Validator interface {
	Stake() *big.Int
	Rating() int32
	PubKey() []byte
}

// NodesCoordinator defines the behaviour of a struct able to do validator group selection
type NodesCoordinator interface {
	PublicKeysSelector
	LoadNodesPerShards(nodes map[uint32][]Validator) error
	ComputeValidatorsGroup(randomness []byte) (validatorsGroup []Validator, err error)
	ConsensusGroupSize() int
	SetConsensusGroupSize(int) error
}

// PublicKeysSelector allows retrieval of eligible validators public keys
type PublicKeysSelector interface {
	GetSelectedPublicKeys(selection []byte) (publicKeys []string, err error)
	GetValidatorsPublicKeys(randomness []byte) ([]string, error)
}
