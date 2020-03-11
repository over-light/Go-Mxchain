package mock

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/state"
)

// UserAccountStub -
type UserAccountStub struct {
	AddToBalanceCalled func(value *big.Int) error
}

// AddToBalance -
func (u *UserAccountStub) AddToBalance(value *big.Int) error {
	if u.AddToBalanceCalled != nil {
		return u.AddToBalanceCalled(value)
	}
	return nil
}

// SubFromBalance -
func (u *UserAccountStub) SubFromBalance(_ *big.Int) error {
	return nil
}

// GetBalance -
func (u *UserAccountStub) GetBalance() *big.Int {
	return nil
}

// ClaimDeveloperRewards -
func (u *UserAccountStub) ClaimDeveloperRewards([]byte) (*big.Int, error) {
	return nil, nil
}

// AddToDeveloperReward -
func (u *UserAccountStub) AddToDeveloperReward(*big.Int) {

}

// GetDeveloperReward -
func (u *UserAccountStub) GetDeveloperReward() *big.Int {
	return nil
}

// ChangeOwnerAddress -
func (u *UserAccountStub) ChangeOwnerAddress([]byte, []byte) error {
	return nil
}

// SetOwnerAddress -
func (u *UserAccountStub) SetOwnerAddress([]byte) {

}

// GetOwnerAddress -
func (u *UserAccountStub) GetOwnerAddress() []byte {
	return nil
}

// AddressContainer -
func (u *UserAccountStub) AddressContainer() state.AddressContainer {
	return nil
}

// SetNonce -
func (u *UserAccountStub) SetNonce(_ uint64) {

}

// GetNonce -
func (u *UserAccountStub) GetNonce() uint64 {
	return 0
}

// SetCode -
func (u *UserAccountStub) SetCode(_ []byte) {

}

// GetCode -
func (u *UserAccountStub) GetCode() []byte {
	return nil
}

// SetCodeHash -
func (u *UserAccountStub) SetCodeHash([]byte) {

}

// GetCodeHash -
func (u *UserAccountStub) GetCodeHash() []byte {
	return nil
}

// SetRootHash -
func (u *UserAccountStub) SetRootHash([]byte) {

}

// GetRootHash -
func (u *UserAccountStub) GetRootHash() []byte {
	return nil
}

// SetDataTrie -
func (u *UserAccountStub) SetDataTrie(_ data.Trie) {

}

// DataTrie -
func (u *UserAccountStub) DataTrie() data.Trie {
	return nil
}

// DataTrieTracker -
func (u *UserAccountStub) DataTrieTracker() state.DataTrieTracker {
	return nil
}

// IsInterfaceNil -
func (u *UserAccountStub) IsInterfaceNil() bool {
	return false
}
