package mock

import (
	"github.com/ElrondNetwork/elrond-go/data/state"
)

// AccountsStub -
type AccountsStub struct {
	AddJournalEntryCalled    func(je state.JournalEntry)
	CommitCalled             func() ([]byte, error)
	GetExistingAccountCalled func(addressContainer state.AddressContainer) (state.AccountHandler, error)
	HasAccountStateCalled    func(addressContainer state.AddressContainer) (bool, error)
	JournalLenCalled         func() int
	PutCodeCalled            func(accountHandler state.AccountHandler, code []byte) error
	RemoveAccountCalled      func(addressContainer state.AddressContainer) error
	RemoveCodeCalled         func(codeHash []byte) error
	RevertToSnapshotCalled   func(snapshot int) error
	SaveAccountStateCalled   func(acountWrapper state.AccountHandler) error
	SaveDataTrieCalled       func(acountWrapper state.AccountHandler) error
	RootHashCalled           func() ([]byte, error)
	RecreateTrieCalled       func(rootHash []byte) error
	PruneTrieCalled          func(rootHash []byte) error
	SnapshotStateCalled      func(rootHash []byte)
	SetStateCheckpointCalled func(rootHash []byte)
	CancelPruneCalled        func(rootHash []byte)
	IsPruningEnabledCalled   func() bool
	GetAllLeavesCalled       func(rootHash []byte) (map[string][]byte, error)
	LoadAccountCalled        func(container state.AddressContainer) (state.AccountHandler, error)
	SaveAccountCalled        func(account state.AccountHandler) error
}

func (as *AccountsStub) LoadAccount(address state.AddressContainer) (state.AccountHandler, error) {
	if as.LoadAccountCalled != nil {
		return as.LoadAccountCalled(address)
	}
	return nil, nil
}

func (as *AccountsStub) SaveAccount(account state.AccountHandler) error {
	if as.SaveAccountCalled != nil {
		return as.SaveAccountCalled(account)
	}
	return nil
}

// GetAllLeaves -
func (as *AccountsStub) GetAllLeaves(rootHash []byte) (map[string][]byte, error) {
	if as.GetAllLeavesCalled != nil {
		return as.GetAllLeavesCalled(rootHash)
	}
	return nil, nil
}

// ClosePersister -
func (as *AccountsStub) ClosePersister() error {
	return nil
}

// AddJournalEntry -
func (as *AccountsStub) AddJournalEntry(je state.JournalEntry) {
	as.AddJournalEntryCalled(je)
}

// Commit -
func (as *AccountsStub) Commit() ([]byte, error) {
	return as.CommitCalled()
}

// GetExistingAccount -
func (as *AccountsStub) GetExistingAccount(addressContainer state.AddressContainer) (state.AccountHandler, error) {
	return as.GetExistingAccountCalled(addressContainer)
}

// HasAccount -
func (as *AccountsStub) HasAccount(addressContainer state.AddressContainer) (bool, error) {
	return as.HasAccountStateCalled(addressContainer)
}

// JournalLen -
func (as *AccountsStub) JournalLen() int {
	return as.JournalLenCalled()
}

// PutCode -
func (as *AccountsStub) PutCode(accountHandler state.AccountHandler, code []byte) error {
	return as.PutCodeCalled(accountHandler, code)
}

// RemoveAccount -
func (as *AccountsStub) RemoveAccount(addressContainer state.AddressContainer) error {
	return as.RemoveAccountCalled(addressContainer)
}

// RemoveCode -
func (as *AccountsStub) RemoveCode(codeHash []byte) error {
	return as.RemoveCodeCalled(codeHash)
}

// RevertToSnapshot -
func (as *AccountsStub) RevertToSnapshot(snapshot int) error {
	return as.RevertToSnapshotCalled(snapshot)
}

// SaveJournalizedAccount -
func (as *AccountsStub) SaveJournalizedAccount(journalizedAccountHandler state.AccountHandler) error {
	return as.SaveAccountStateCalled(journalizedAccountHandler)
}

// SaveDataTrie -
func (as *AccountsStub) SaveDataTrie(journalizedAccountHandler state.AccountHandler) error {
	return as.SaveDataTrieCalled(journalizedAccountHandler)
}

// RootHash -
func (as *AccountsStub) RootHash() ([]byte, error) {
	return as.RootHashCalled()
}

// RecreateTrie -
func (as *AccountsStub) RecreateTrie(rootHash []byte) error {
	return as.RecreateTrieCalled(rootHash)
}

// PruneTrie -
func (as *AccountsStub) PruneTrie(rootHash []byte) error {
	return as.PruneTrieCalled(rootHash)
}

// CancelPrune -
func (as *AccountsStub) CancelPrune(rootHash []byte) {
	as.CancelPruneCalled(rootHash)
}

// SnapshotState -
func (as *AccountsStub) SnapshotState(rootHash []byte) {
	as.SnapshotStateCalled(rootHash)
}

// SetStateCheckpoint -
func (as *AccountsStub) SetStateCheckpoint(rootHash []byte) {
	as.SetStateCheckpointCalled(rootHash)
}

// IsPruningEnabled -
func (as *AccountsStub) IsPruningEnabled() bool {
	return as.IsPruningEnabledCalled()
}

// IsInterfaceNil returns true if there is no value under the interface
func (as *AccountsStub) IsInterfaceNil() bool {
	return as == nil
}
