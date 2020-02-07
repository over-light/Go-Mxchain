//go:generate protoc -I=proto -I=$GOPATH/src -I=$GOPATH/src/github.com/gogo/protobuf/protobuf  --gogoslick_out=. accountData.proto
package state

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go/data"
)

// Account is the struct used in serialization/deserialization
type Account struct {
	AccountData

	addressContainer AddressContainer
	code             []byte
	accountTracker   AccountTracker
	dataTrieTracker  DataTrieTracker
}

// NewAccount creates new simple account wrapper for an AccountContainer (that has just been initialized)
func NewAccount(addressContainer AddressContainer, tracker AccountTracker) (*Account, error) {
	if addressContainer == nil || addressContainer.IsInterfaceNil() {
		return nil, ErrNilAddressContainer
	}
	if tracker == nil || tracker.IsInterfaceNil() {
		return nil, ErrNilAccountTracker
	}

	return &Account{
		AccountData: AccountData{
			Balance: big.NewInt(0),
		},
		addressContainer: addressContainer,
		accountTracker:   tracker,
		dataTrieTracker:  NewTrackableDataTrie(nil),
	}, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (a *Account) IsInterfaceNil() bool {
	if a == nil {
		return true
	}
	return false
}

// AddressContainer returns the address associated with the account
func (a *Account) AddressContainer() AddressContainer {
	return a.addressContainer
}

// SetNonceWithJournal sets the account's nonce, saving the old nonce before changing
func (a *Account) SetNonceWithJournal(nonce uint64) error {
	entry, err := NewBaseJournalEntryNonce(a, a.Nonce)
	if err != nil {
		return err
	}

	a.accountTracker.Journalize(entry)
	a.Nonce = nonce

	return a.accountTracker.SaveAccount(a)
}

//SetNonce saves the nonce to the account
func (a *Account) SetNonce(nonce uint64) {
	a.Nonce = nonce
}

// GetNonce gets the nonce of the account
func (a *Account) GetNonce() uint64 {
	return a.Nonce
}

// SetBalanceWithJournal sets the account's balance, saving the old balance before changing
func (a *Account) SetBalanceWithJournal(balance *big.Int) error {
	entry, err := NewJournalEntryBalance(a, a.Balance)
	if err != nil {
		return err
	}

	a.accountTracker.Journalize(entry)
	a.Balance.Set(balance)

	return a.accountTracker.SaveAccount(a)
}

//------- code / code hash

// GetCodeHash returns the code hash associated with this account
func (a *Account) GetCodeHash() []byte {
	return a.CodeHash
}

// SetCodeHash sets the code hash associated with the account
func (a *Account) SetCodeHash(codeHash []byte) {
	a.CodeHash = codeHash
}

// SetCodeHashWithJournal sets the account's code hash, saving the old code hash before changing
func (a *Account) SetCodeHashWithJournal(codeHash []byte) error {
	entry, err := NewBaseJournalEntryCodeHash(a, a.CodeHash)
	if err != nil {
		return err
	}

	a.accountTracker.Journalize(entry)
	a.CodeHash = codeHash

	return a.accountTracker.SaveAccount(a)
}

// GetCode gets the actual code that needs to be run in the VM
func (a *Account) GetCode() []byte {
	return a.code
}

// SetCode sets the actual code that needs to be run in the VM
func (a *Account) SetCode(code []byte) {
	a.code = code
}

//------- data trie / root hash

// GetRootHash returns the root hash associated with this account
func (a *Account) GetRootHash() []byte {
	return a.RootHash
}

// SetRootHash sets the root hash associated with the account
func (a *Account) SetRootHash(roothash []byte) {
	a.RootHash = roothash
}

// DataTrie returns the trie that holds the current account's data
func (a *Account) DataTrie() data.Trie {
	return a.dataTrieTracker.DataTrie()
}

// SetDataTrie sets the trie that holds the current account's data
func (a *Account) SetDataTrie(trie data.Trie) {
	a.dataTrieTracker.SetDataTrie(trie)
}

// DataTrieTracker returns the trie wrapper used in managing the SC data
func (a *Account) DataTrieTracker() DataTrieTracker {
	return a.dataTrieTracker
}

//TODO add Cap'N'Proto converter funcs
