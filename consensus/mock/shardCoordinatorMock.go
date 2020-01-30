package mock

import (
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/state"
)

type ShardCoordinatorMock struct {
}

func (scm ShardCoordinatorMock) NumberOfShards() uint32 {
	return uint32(2)
}

func (scm ShardCoordinatorMock) ComputeId(address state.AddressContainer) uint32 {
	panic("implement me")
}

func (scm ShardCoordinatorMock) SetSelfId(shardId uint32) error {
	panic("implement me")
}

func (scm ShardCoordinatorMock) SelfId() uint32 {
	return 0
}

func (scm ShardCoordinatorMock) SameShard(firstAddress, secondAddress state.AddressContainer) bool {
	return true
}

func (scm ShardCoordinatorMock) CommunicationIdentifier(destShardID uint32) string {
	if destShardID == core.MetachainShardId {
		return "_0_META"
	}

	return "_0"
}

// IsInterfaceNil returns true if there is no value under the interface
func (scm ShardCoordinatorMock) IsInterfaceNil() bool {
	if &scm == nil {
		return true
	}
	return false
}
