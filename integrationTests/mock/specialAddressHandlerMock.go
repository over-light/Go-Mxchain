package mock

import (
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type SpecialAddressHandlerMock struct {
	ElrondCommunityAddressCalled func() []byte
	LeaderAddressCalled          func() []byte
	BurnAddressCalled            func() []byte
	ShardIdForAddressCalled      func([]byte) (uint32, error)
	AdrConv                      state.AddressConverter
	ShardCoordinator             sharding.Coordinator

	shardConsensusData *data.ConsensusRewardData
	metaConsensusData  []*data.ConsensusRewardData
}

func (sh *SpecialAddressHandlerMock) SetElrondCommunityAddress(elrond []byte) {
}

func (sh *SpecialAddressHandlerMock) SetConsensusData(consensusRewardAddresses []string, round uint64, epoch uint32) {
	sh.shardConsensusData = &data.ConsensusRewardData{
		Round:     round,
		Epoch:     epoch,
		Addresses: consensusRewardAddresses,
	}
}

func (sh *SpecialAddressHandlerMock) ConsensusShardRewardData() *data.ConsensusRewardData {
	return sh.shardConsensusData
}

func (sh *SpecialAddressHandlerMock) SetMetaConsensusData(rewardAddresses []string, round uint64, epoch uint32) {
	if sh.metaConsensusData == nil {
		sh.metaConsensusData = make([]*data.ConsensusRewardData, 0)
	}

	sh.metaConsensusData = append(sh.metaConsensusData, &data.ConsensusRewardData{
		Round:     round,
		Epoch:     epoch,
		Addresses: rewardAddresses,
	})
}

func (sh *SpecialAddressHandlerMock) ClearMetaConsensusData() {
	sh.metaConsensusData = make([]*data.ConsensusRewardData, 0)
}

func (sh *SpecialAddressHandlerMock) ConsensusMetaRewardData() []*data.ConsensusRewardData {
	return sh.metaConsensusData
}

func (sh *SpecialAddressHandlerMock) BurnAddress() []byte {
	if sh.BurnAddressCalled == nil {
		return []byte("burn0000000000000000000000000000")
	}

	return sh.BurnAddressCalled()
}

func (sh *SpecialAddressHandlerMock) ElrondCommunityAddress() []byte {
	if sh.ElrondCommunityAddressCalled == nil {
		return []byte("elrond00000000000000000000000000")
	}

	return sh.ElrondCommunityAddressCalled()
}

func (sh *SpecialAddressHandlerMock) LeaderAddress() []byte {
	if sh.LeaderAddressCalled == nil {
		return []byte("leader0000000000000000000000000000")
	}

	return sh.LeaderAddressCalled()
}

func (sh *SpecialAddressHandlerMock) Round() uint64 {
	return sh.shardConsensusData.Round
}

func (sh *SpecialAddressHandlerMock) Epoch() uint32 {
	return sh.shardConsensusData.Epoch
}

func (sh *SpecialAddressHandlerMock) ShardIdForAddress(addr []byte) (uint32, error) {
	convAdr, err := sh.AdrConv.CreateAddressFromPublicKeyBytes(addr)
	if err != nil {
		return 0, err
	}

	return sh.ShardCoordinator.ComputeId(convAdr), nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (sh *SpecialAddressHandlerMock) IsInterfaceNil() bool {
	if sh == nil {
		return true
	}
	return false
}
