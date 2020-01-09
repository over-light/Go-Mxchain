package mock

import (
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/state"
)

type oneShardCoordinatorMock struct {
	noShards uint32
}

func NewOneShardCoordinatorMock() *oneShardCoordinatorMock {
	return &oneShardCoordinatorMock{noShards: 1}
}

func (scm *oneShardCoordinatorMock) NumberOfShards() uint32 {
	return scm.noShards
}

func (scm *oneShardCoordinatorMock) SetNoShards(shards uint32) {
	scm.noShards = shards
}

func (scm *oneShardCoordinatorMock) ComputeId(address state.AddressContainer) uint32 {

	return uint32(0)
}

func (scm *oneShardCoordinatorMock) SelfId() uint32 {
	return 0
}

func (scm *oneShardCoordinatorMock) SetSelfId(shardId uint32) error {
	return nil
}

func (scm *oneShardCoordinatorMock) SameShard(firstAddress, secondAddress state.AddressContainer) bool {
	return true
}

func (scm *oneShardCoordinatorMock) CommunicationIdentifier(destShardID uint32) string {
	if destShardID == core.MetachainShardId {
		return "_0_META"
	}

	return "_0"
}

// IsInterfaceNil returns true if there is no value under the interface
func (scm *oneShardCoordinatorMock) IsInterfaceNil() bool {
	if scm == nil {
		return true
	}
	return false
}
