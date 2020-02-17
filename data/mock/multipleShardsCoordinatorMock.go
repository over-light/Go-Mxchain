package mock

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go/data/state"
)

// MultipleShardsCoordinatorMock -
type MultipleShardsCoordinatorMock struct {
	ComputeIdCalled func(address state.AddressContainer) uint32
	NoShards        uint32
	CurrentShard    uint32
}

// NewMultiShardsCoordinatorMock -
func NewMultiShardsCoordinatorMock(nrShard uint32) *MultipleShardsCoordinatorMock {
	return &MultipleShardsCoordinatorMock{NoShards: nrShard}
}

// NumberOfShards -
func (scm *MultipleShardsCoordinatorMock) NumberOfShards() uint32 {
	return scm.NoShards
}

// ComputeId -
func (scm *MultipleShardsCoordinatorMock) ComputeId(address state.AddressContainer) uint32 {
	if scm.ComputeIdCalled == nil {
		return scm.SelfId()
	}
	return scm.ComputeIdCalled(address)
}

// SelfId -
func (scm *MultipleShardsCoordinatorMock) SelfId() uint32 {
	return scm.CurrentShard
}

// SetSelfId -
func (scm *MultipleShardsCoordinatorMock) SetSelfId(shardId uint32) error {
	return nil
}

// SameShard -
func (scm *MultipleShardsCoordinatorMock) SameShard(firstAddress, secondAddress state.AddressContainer) bool {
	return true
}

// SetNoShards -
func (scm *MultipleShardsCoordinatorMock) SetNoShards(noShards uint32) {
	scm.NoShards = noShards
}

// CommunicationIdentifier returns the identifier between current shard ID and destination shard ID
// identifier is generated such as the first shard from identifier is always smaller than the last
func (scm *MultipleShardsCoordinatorMock) CommunicationIdentifier(destShardID uint32) string {
	if destShardID == scm.CurrentShard {
		return fmt.Sprintf("_%d", scm.CurrentShard)
	}

	if destShardID < scm.CurrentShard {
		return fmt.Sprintf("_%d_%d", destShardID, scm.CurrentShard)
	}

	return fmt.Sprintf("_%d_%d", scm.CurrentShard, destShardID)
}

// IsInterfaceNil returns true if there is no value under the interface
func (scm *MultipleShardsCoordinatorMock) IsInterfaceNil() bool {
	if scm == nil {
		return true
	}
	return false
}
