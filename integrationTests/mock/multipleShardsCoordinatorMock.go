package mock

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go/data/state"
)

type multipleShardsCoordinatorMock struct {
	noShards        uint32
	ComputeIdCalled func(address state.AddressContainer) uint32
	CurrentShard    uint32
}

func NewMultipleShardsCoordinatorMock() *multipleShardsCoordinatorMock {
	return &multipleShardsCoordinatorMock{noShards: 1}
}

func NewMultiShardsCoordinatorMock(nrShard uint32) *multipleShardsCoordinatorMock {
	return &multipleShardsCoordinatorMock{noShards: nrShard}
}

func (scm *multipleShardsCoordinatorMock) NumberOfShards() uint32 {
	return scm.noShards
}

func (scm *multipleShardsCoordinatorMock) ComputeId(address state.AddressContainer) uint32 {
	if scm.ComputeIdCalled != nil {
		return scm.ComputeIdCalled(address)
	}

	return uint32(0)
}

func (scm *multipleShardsCoordinatorMock) SelfId() uint32 {
	return scm.CurrentShard
}

func (scm *multipleShardsCoordinatorMock) SetSelfId(shardId uint32) error {
	return nil
}

func (scm *multipleShardsCoordinatorMock) SameShard(firstAddress, secondAddress state.AddressContainer) bool {
	return true
}

func (scm *multipleShardsCoordinatorMock) SetNoShards(noShards uint32) {
	scm.noShards = noShards
}

// CommunicationIdentifier returns the identifier between current shard ID and destination shard ID
// identifier is generated such as the first shard from identifier is always smaller than the last
func (scm *multipleShardsCoordinatorMock) CommunicationIdentifier(destShardID uint32) string {
	if destShardID == scm.CurrentShard {
		return fmt.Sprintf("_%d", scm.CurrentShard)
	}

	if destShardID < scm.CurrentShard {
		return fmt.Sprintf("_%d_%d", destShardID, scm.CurrentShard)
	}

	return fmt.Sprintf("_%d_%d", scm.CurrentShard, destShardID)
}

// IsInterfaceNil returns true if there is no value under the interface
func (scm *multipleShardsCoordinatorMock) IsInterfaceNil() bool {
	if scm == nil {
		return true
	}
	return false
}
