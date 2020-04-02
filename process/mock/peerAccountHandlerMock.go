package mock

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/state"
)

// PeerAccountHandlerMock -
type PeerAccountHandlerMock struct {
	IncreaseLeaderSuccessRateCalled    func(uint32)
	DecreaseLeaderSuccessRateCalled    func(uint32)
	IncreaseValidatorSuccessRateCalled func(uint32)
	DecreaseValidatorSuccessRateCalled func(uint32)
	SetTempRatingCalled                func(uint32)
	GetTempRatingCalled                func() uint32
	SetAccumulatedFeesCalled           func(*big.Int)
	GetAccumulatedFeesCalled           func() *big.Int
	GetConsecutiveProposerMissesCalled func() uint32
	SetConsecutiveProposerMissesCalled func(rating uint32)
	SetListAndIndexCalled              func(shardID uint32, list string, index uint32)
}

// GetList -
func (p *PeerAccountHandlerMock) GetList() string {
	return ""
}

// GetIndex -
func (p *PeerAccountHandlerMock) GetIndex() uint32 {
	return 0
}

// GetBLSPublicKey -
func (p *PeerAccountHandlerMock) GetBLSPublicKey() []byte {
	return nil
}

// SetBLSPublicKey -
func (p *PeerAccountHandlerMock) SetBLSPublicKey([]byte) error {
	return nil
}

// GetSchnorrPublicKey -
func (p *PeerAccountHandlerMock) GetSchnorrPublicKey() []byte {
	return nil
}

// SetSchnorrPublicKey -
func (p *PeerAccountHandlerMock) SetSchnorrPublicKey([]byte) error {
	return nil
}

// GetRewardAddress -
func (p *PeerAccountHandlerMock) GetRewardAddress() []byte {
	return nil
}

// SetRewardAddress -
func (p *PeerAccountHandlerMock) SetRewardAddress([]byte) error {
	return nil
}

// GetStake -
func (p *PeerAccountHandlerMock) GetStake() *big.Int {
	return nil
}

// SetStake -
func (p *PeerAccountHandlerMock) SetStake(*big.Int) error {
	return nil
}

// GetAccumulatedFees -
func (p *PeerAccountHandlerMock) GetAccumulatedFees() *big.Int {
	if p.GetAccumulatedFeesCalled != nil {
		p.GetAccumulatedFeesCalled()
	}
	return big.NewInt(0)
}

// AddToAccumulatedFees -
func (p *PeerAccountHandlerMock) AddToAccumulatedFees(val *big.Int) {
	if p.SetAccumulatedFeesCalled != nil {
		p.SetAccumulatedFeesCalled(val)
	}
}

// GetJailTime -
func (p *PeerAccountHandlerMock) GetJailTime() state.TimePeriod {
	return state.TimePeriod{}
}

// SetJailTime -
func (p *PeerAccountHandlerMock) SetJailTime(state.TimePeriod) {

}

// GetCurrentShardId -
func (p *PeerAccountHandlerMock) GetCurrentShardId() uint32 {
	return 0
}

// SetCurrentShardId -
func (p *PeerAccountHandlerMock) SetCurrentShardId(uint32) {

}

// GetNextShardId -
func (p *PeerAccountHandlerMock) GetNextShardId() uint32 {
	return 0
}

// SetNextShardId -
func (p *PeerAccountHandlerMock) SetNextShardId(uint32) {

}

// GetNodeInWaitingList -
func (p *PeerAccountHandlerMock) GetNodeInWaitingList() bool {
	return false
}

// SetNodeInWaitingList -
func (p *PeerAccountHandlerMock) SetNodeInWaitingList(bool) {

}

// GetUnStakedNonce -
func (p *PeerAccountHandlerMock) GetUnStakedNonce() uint64 {
	return 0
}

// SetUnStakedNonce -
func (p *PeerAccountHandlerMock) SetUnStakedNonce(uint64) {

}

// IncreaseLeaderSuccessRate -
func (p *PeerAccountHandlerMock) IncreaseLeaderSuccessRate(val uint32) {
	if p.IncreaseLeaderSuccessRateCalled != nil {
		p.IncreaseLeaderSuccessRateCalled(val)
	}
}

// DecreaseLeaderSuccessRate -
func (p *PeerAccountHandlerMock) DecreaseLeaderSuccessRate(val uint32) {
	if p.DecreaseLeaderSuccessRateCalled != nil {
		p.DecreaseLeaderSuccessRateCalled(val)
	}
}

// IncreaseValidatorSuccessRate -
func (p *PeerAccountHandlerMock) IncreaseValidatorSuccessRate(val uint32) {
	if p.IncreaseValidatorSuccessRateCalled != nil {
		p.IncreaseValidatorSuccessRateCalled(val)
	}
}

// DecreaseValidatorSuccessRate -
func (p *PeerAccountHandlerMock) DecreaseValidatorSuccessRate(val uint32) {
	if p.DecreaseValidatorSuccessRateCalled != nil {
		p.DecreaseValidatorSuccessRateCalled(val)
	}
}

// GetNumSelectedInSuccessBlocks -
func (p *PeerAccountHandlerMock) GetNumSelectedInSuccessBlocks() uint32 {
	return 0
}

// IncreaseNumSelectedInSuccessBlocks -
func (p *PeerAccountHandlerMock) IncreaseNumSelectedInSuccessBlocks() {

}

// GetLeaderSuccessRate -
func (p *PeerAccountHandlerMock) GetLeaderSuccessRate() state.SignRate {
	return state.SignRate{}
}

// GetValidatorSuccessRate -
func (p *PeerAccountHandlerMock) GetValidatorSuccessRate() state.SignRate {
	return state.SignRate{}
}

// GetLeaderSuccessRate -
func (p *PeerAccountHandlerMock) GetTotalLeaderSuccessRate() state.SignRate {
	return state.SignRate{}
}

// GetValidatorSuccessRate -
func (p *PeerAccountHandlerMock) GetTotalValidatorSuccessRate() state.SignRate {
	return state.SignRate{}
}

// GetRating -
func (p *PeerAccountHandlerMock) GetRating() uint32 {
	return 0
}

// SetRating -
func (p *PeerAccountHandlerMock) SetRating(uint32) {

}

// GetTempRating -
func (p *PeerAccountHandlerMock) GetTempRating() uint32 {
	if p.GetTempRatingCalled != nil {
		return p.GetTempRatingCalled()
	}
	return 0
}

// SetTempRating -
func (p *PeerAccountHandlerMock) SetTempRating(val uint32) {
	if p.SetTempRatingCalled != nil {
		p.SetTempRatingCalled(val)
	}
}

// ResetAtNewEpoch -
func (p *PeerAccountHandlerMock) ResetAtNewEpoch() {
}

// AddressContainer -
func (p *PeerAccountHandlerMock) AddressContainer() state.AddressContainer {
	return nil
}

// IncreaseNonce -
func (p *PeerAccountHandlerMock) IncreaseNonce(_ uint64) {
}

// GetNonce -
func (p *PeerAccountHandlerMock) GetNonce() uint64 {
	return 0
}

// SetCode -
func (p *PeerAccountHandlerMock) SetCode(_ []byte) {

}

// GetCode -
func (p *PeerAccountHandlerMock) GetCode() []byte {
	return nil
}

// SetCodeHash -
func (p *PeerAccountHandlerMock) SetCodeHash([]byte) {

}

// GetCodeHash -
func (p *PeerAccountHandlerMock) GetCodeHash() []byte {
	return nil
}

// SetRootHash -
func (p *PeerAccountHandlerMock) SetRootHash([]byte) {

}

// GetRootHash -
func (p *PeerAccountHandlerMock) GetRootHash() []byte {
	return nil
}

// SetDataTrie -
func (p *PeerAccountHandlerMock) SetDataTrie(_ data.Trie) {

}

// DataTrie -
func (p *PeerAccountHandlerMock) DataTrie() data.Trie {
	return nil
}

// DataTrieTracker -
func (p *PeerAccountHandlerMock) DataTrieTracker() state.DataTrieTracker {
	return nil
}

// GetConsecutiveProposerMisses -
func (pahm *PeerAccountHandlerMock) GetConsecutiveProposerMisses() uint32 {
	if pahm.GetConsecutiveProposerMissesCalled != nil {
		return pahm.GetConsecutiveProposerMissesCalled()
	}
	return 0
}

// SetConsecutiveProposerMissesWithJournal -
func (pahm *PeerAccountHandlerMock) SetConsecutiveProposerMisses(consecutiveMisses uint32) {
	if pahm.SetConsecutiveProposerMissesCalled != nil {
		pahm.SetConsecutiveProposerMissesCalled(consecutiveMisses)
	}
}

// SetListAndIndex -
func (pahm *PeerAccountHandlerMock) SetListAndIndex(shardID uint32, list string, index uint32) {
	if pahm.SetListAndIndexCalled != nil {
		pahm.SetListAndIndexCalled(shardID, list, index)
	}
}

// IsInterfaceNil -
func (p *PeerAccountHandlerMock) IsInterfaceNil() bool {
	return false
}
