package mock

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go-sandbox/data/state"
)

type multipleShardsCoordinatorMock struct {
	noShards                     uint32
	ComputeShardForAddressCalled func(address state.AddressContainer, addressConverter state.AddressConverter) uint32
	CurrentShard                 uint32
}

func NewMultipleShardsCoordinatorMock() *multipleShardsCoordinatorMock {
	return &multipleShardsCoordinatorMock{noShards: 1}
}

func (scm *multipleShardsCoordinatorMock) NoShards() uint32 {
	return scm.noShards
}

func (scm *multipleShardsCoordinatorMock) SetNoShards(shards uint32) {
	scm.noShards = shards
}

func (scm *multipleShardsCoordinatorMock) ComputeShardForAddress(
	address state.AddressContainer,
	addressConverter state.AddressConverter) uint32 {

	return scm.ComputeShardForAddressCalled(address, addressConverter)
}

func (scm *multipleShardsCoordinatorMock) ShardForCurrentNode() uint32 {
	return scm.CurrentShard
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
