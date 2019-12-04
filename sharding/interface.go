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
	IsInterfaceNil() bool
}

// Validator defines a node that can be allocated to a shard for participation in a consensus group as validator
// or block proposer
type Validator interface {
	Stake() *big.Int
	Rating() int32
	PubKey() []byte
	Address() []byte
}

// NodesCoordinator defines the behaviour of a struct able to do validator group selection
type NodesCoordinator interface {
	PublicKeysSelector
	SetNodesPerShards(nodes map[uint32][]Validator) error
	ComputeValidatorsGroup(randomness []byte, round uint64, shardId uint32) (validatorsGroup []Validator, err error)
	GetValidatorWithPublicKey(publicKey []byte) (validator Validator, shardId uint32, err error)
	IsInterfaceNil() bool
}

// PublicKeysSelector allows retrieval of eligible validators public keys
type PublicKeysSelector interface {
	GetValidatorsIndexes(publicKeys []string) []uint64
	GetAllValidatorsPublicKeys() map[uint32][][]byte
	GetSelectedPublicKeys(selection []byte, shardId uint32) (publicKeys []string, err error)
	GetValidatorsPublicKeys(randomness []byte, round uint64, shardId uint32) ([]string, error)
	GetValidatorsRewardsAddresses(randomness []byte, round uint64, shardId uint32) ([]string, error)
	GetOwnPublicKey() []byte
}

type RaterHandler interface {
	RatingReader
	//ComputeRating computes the current rating
	ComputeRating(string, uint32) uint32
	//GetRatingOptionKeys gets all the ratings option keys
	GetRatingOptionKeys() []string
	//GetStartRating gets the start rating values
	GetStartRating() uint32
}

type RatingReader interface {
	//GetRating gets the rating for the public key
	GetRating(string) uint32
	//GetRatings gets all the ratings as a map[pk] ratingValue
	GetRatings([]string) map[string]uint32
	//IsInterfaceNil verifies if the interface is nil
	IsInterfaceNil() bool
}

type RatingReaderSetter interface {
	//GetRating gets the rating for the public key
	SetRatingReader(RatingReader)
	//IsInterfaceNil verifies if the interface is nil
	IsInterfaceNil() bool
}
