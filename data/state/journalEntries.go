package state

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go/core/check"
)

//------- JournalEntryBalance

// JournalEntryBalance is used to revert a balance change
type JournalEntryBalance struct {
	account    *Account
	oldBalance *big.Int
}

// NewJournalEntryBalance outputs a new JournalEntry implementation used to revert a balance change
func NewJournalEntryBalance(account *Account, oldBalance *big.Int) (*JournalEntryBalance, error) {
	if account == nil {
		return nil, ErrNilAccountHandler
	}

	return &JournalEntryBalance{
		account:    account,
		oldBalance: big.NewInt(0).Set(oldBalance),
	}, nil
}

// Revert applies undo operation
func (jeb *JournalEntryBalance) Revert() (AccountHandler, error) {
	jeb.account.Balance = big.NewInt(0).Set(jeb.oldBalance)

	return jeb.account, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (jeb *JournalEntryBalance) IsInterfaceNil() bool {
	return jeb == nil
}

//------- JournalEntryDeveloperReward

// JournalEntryDeveloperReward is used to revert a developer reward change
type JournalEntryDeveloperReward struct {
	account            *Account
	oldDeveloperReward *big.Int
}

// NewJournalEntryDeveloperReward outputs a new JournalEntry implementation used to revert a developer reward change
func NewJournalEntryDeveloperReward(account *Account, oldDeveloperReward *big.Int) (*JournalEntryDeveloperReward, error) {
	if account == nil {
		return nil, ErrNilAccountHandler
	}

	return &JournalEntryDeveloperReward{
		account:            account,
		oldDeveloperReward: big.NewInt(0).Set(oldDeveloperReward),
	}, nil
}

// Revert applies undo operation
func (jed *JournalEntryDeveloperReward) Revert() (AccountHandler, error) {
	jed.account.DeveloperReward = big.NewInt(0).Set(jed.oldDeveloperReward)

	return jed.account, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (jed *JournalEntryDeveloperReward) IsInterfaceNil() bool {
	return jed == nil
}

//------- JournalEntryDataTrieUpdates

// JournalEntryDataTrieUpdates stores all the updates done to the account's data trie,
// so it can be reverted in case of rollback
type JournalEntryDataTrieUpdates struct {
	trieUpdates map[string][]byte
	account     AccountHandler
}

// NewJournalEntryDataTrieUpdates outputs a new JournalEntryDataTrieUpdates implementation used to revert an account's data trie
func NewJournalEntryDataTrieUpdates(trieUpdates map[string][]byte, account AccountHandler) (*JournalEntryDataTrieUpdates, error) {
	if check.IfNil(account) {
		return nil, ErrNilAccountHandler
	}
	if len(trieUpdates) == 0 {
		return nil, ErrNilOrEmptyDataTrieUpdates
	}

	return &JournalEntryDataTrieUpdates{
		trieUpdates: trieUpdates,
		account:     account,
	}, nil
}

// Revert applies undo operation
func (jedtu *JournalEntryDataTrieUpdates) Revert() (AccountHandler, error) {
	for key := range jedtu.trieUpdates {
		err := jedtu.account.DataTrie().Update([]byte(key), jedtu.trieUpdates[key])
		if err != nil {
			return nil, err
		}
	}

	rootHash, err := jedtu.account.DataTrie().Root()
	if err != nil {
		return nil, err
	}

	jedtu.account.SetRootHash(rootHash)

	return jedtu.account, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (jedtu *JournalEntryDataTrieUpdates) IsInterfaceNil() bool {
	return jedtu == nil
}

//------- JournalEntryOwnerAddress

// JournalEntryOwnerAddress is used to revert an owner address change
type JournalEntryOwnerAddress struct {
	account         *Account
	oldOwnerAddress []byte
}

// NewJournalEntryOwnerAddress outputs a new JournalEntry implementation used to revert an owner address change
func NewJournalEntryOwnerAddress(account *Account, ownerAddress []byte) (*JournalEntryOwnerAddress, error) {
	if account == nil {
		return nil, ErrNilAccountHandler
	}

	return &JournalEntryOwnerAddress{
		account:         account,
		oldOwnerAddress: ownerAddress,
	}, nil
}

// Revert applies undo operation
func (jeo *JournalEntryOwnerAddress) Revert() (AccountHandler, error) {
	jeo.account.OwnerAddress = jeo.oldOwnerAddress

	return jeo.account, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (jeo *JournalEntryOwnerAddress) IsInterfaceNil() bool {
	return jeo == nil
}
