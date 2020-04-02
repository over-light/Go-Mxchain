package mock

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/state"
)

// AccountWrapMock -
type AccountWrapMock struct {
	MockValue         int
	dataTrie          data.Trie
	nonce             uint64
	rating            uint32
	tempRating        uint32
	consecutiveMisses uint32
	code              []byte
	codeMetadata      []byte
	codeHash          []byte
	rootHash          []byte
	address           state.AddressContainer
	trackableDataTrie state.DataTrieTracker

	SetNonceWithJournalCalled    func(nonce uint64) error    `json:"-"`
	SetCodeHashWithJournalCalled func(codeHash []byte) error `json:"-"`
	SetCodeWithJournalCalled     func(codeHash []byte) error `json:"-"`
}

// NewAccountWrapMock -
func NewAccountWrapMock(adr state.AddressContainer) *AccountWrapMock {
	return &AccountWrapMock{
		address:           adr,
		trackableDataTrie: state.NewTrackableDataTrie([]byte("identifier"), nil),
	}
}

// AddToBalance -
func (awm *AccountWrapMock) AddToBalance(_ *big.Int) error {
	return nil
}

// SubFromBalance -
func (awm *AccountWrapMock) SubFromBalance(_ *big.Int) error {
	return nil
}

// GetBalance -
func (awm *AccountWrapMock) GetBalance() *big.Int {
	return nil
}

// ClaimDeveloperRewards -
func (awm *AccountWrapMock) ClaimDeveloperRewards([]byte) (*big.Int, error) {
	return nil, nil
}

// AddToDeveloperReward -
func (awm *AccountWrapMock) AddToDeveloperReward(*big.Int) {

}

// GetDeveloperReward -
func (awm *AccountWrapMock) GetDeveloperReward() *big.Int {
	return nil
}

// ChangeOwnerAddress -
func (awm *AccountWrapMock) ChangeOwnerAddress([]byte, []byte) error {
	return nil
}

// SetOwnerAddress -
func (awm *AccountWrapMock) SetOwnerAddress([]byte) {

}

// GetOwnerAddress -
func (awm *AccountWrapMock) GetOwnerAddress() []byte {
	return nil
}

// GetCodeHash -
func (awm *AccountWrapMock) GetCodeHash() []byte {
	return awm.codeHash
}

// SetCodeHash -
func (awm *AccountWrapMock) SetCodeHash(codeHash []byte) {
	awm.codeHash = codeHash
}

// SetCode -
func (awm *AccountWrapMock) SetCode(code []byte) {
	awm.code = code
}

// GetCode -
func (awm *AccountWrapMock) GetCode() []byte {
	return awm.code
}

// SetCodeMetadata -
func (awm *AccountWrapMock) SetCodeMetadata(codeMetadata []byte) {
	awm.codeMetadata = codeMetadata
}

// GetCodeMetadata -
func (awm *AccountWrapMock) GetCodeMetadata() []byte {
	return awm.codeMetadata
}

// GetRootHash -
func (awm *AccountWrapMock) GetRootHash() []byte {
	return awm.rootHash
}

// SetRootHash -
func (awm *AccountWrapMock) SetRootHash(rootHash []byte) {
	awm.rootHash = rootHash
}

// AddressContainer -
func (awm *AccountWrapMock) AddressContainer() state.AddressContainer {
	return awm.address
}

// DataTrie -
func (awm *AccountWrapMock) DataTrie() data.Trie {
	return awm.dataTrie
}

// SetDataTrie -
func (awm *AccountWrapMock) SetDataTrie(trie data.Trie) {
	awm.dataTrie = trie
	awm.trackableDataTrie.SetDataTrie(trie)
}

// DataTrieTracker -
func (awm *AccountWrapMock) DataTrieTracker() state.DataTrieTracker {
	return awm.trackableDataTrie
}

//IncreaseNonce -
func (awm *AccountWrapMock) IncreaseNonce(val uint64) {
	awm.nonce = awm.nonce + val
}

// GetNonce -
func (awm *AccountWrapMock) GetNonce() uint64 {
	return awm.nonce
}

// GetConsecutiveProposerMisses -
func (awm *AccountWrapMock) GetConsecutiveProposerMisses() uint32 {
	return awm.consecutiveMisses
}

// SetConsecutiveProposerMissesWithJournal -
func (awm *AccountWrapMock) SetConsecutiveProposerMisses(consecutiveMisses uint32) {
	awm.consecutiveMisses = consecutiveMisses
}

// IsInterfaceNil -
func (awm *AccountWrapMock) IsInterfaceNil() bool {
	return awm == nil
}
