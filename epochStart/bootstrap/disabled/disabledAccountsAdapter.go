package disabled

import (
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/state"
)

type accountsAdapter struct {
}

// NewAccountsAdapter returns a nil implementation of accountsAdapter
func NewAccountsAdapter() *accountsAdapter {
	return &accountsAdapter{}
}

// LoadAccount -
func (a *accountsAdapter) LoadAccount(_ state.AddressContainer) (state.AccountHandler, error) {
	return nil, nil
}

// SaveAccount -
func (a *accountsAdapter) SaveAccount(_ state.AccountHandler) error {
	return nil
}

// PruneTrie -
func (a *accountsAdapter) PruneTrie(_ []byte, _ data.TriePruningIdentifier) {
}

// GetExistingAccount -
func (a *accountsAdapter) GetExistingAccount(_ state.AddressContainer) (state.AccountHandler, error) {
	return nil, nil
}

// RemoveAccount -
func (a *accountsAdapter) RemoveAccount(_ state.AddressContainer) error {
	return nil
}

// Commit -
func (a *accountsAdapter) Commit() ([]byte, error) {
	return nil, nil
}

// JournalLen -
func (a *accountsAdapter) JournalLen() int {
	return 0
}

// RevertToSnapshot -
func (a *accountsAdapter) RevertToSnapshot(_ int) error {
	return nil
}

// RootHash -
func (a *accountsAdapter) RootHash() ([]byte, error) {
	return nil, nil
}

// RecreateTrie -
func (a *accountsAdapter) RecreateTrie(_ []byte) error {
	return nil
}

// CancelPrune -
func (a *accountsAdapter) CancelPrune(_ []byte, _ data.TriePruningIdentifier) {
	return
}

// SnapshotState -
func (a *accountsAdapter) SnapshotState(_ []byte) {
	return
}

// SetStateCheckpoint -
func (a *accountsAdapter) SetStateCheckpoint(_ []byte) {
	return
}

// IsPruningEnabled -
func (a *accountsAdapter) IsPruningEnabled() bool {
	return false
}

// ClosePersister -
func (a *accountsAdapter) ClosePersister() error {
	return nil
}

// GetAllLeaves -
func (a *accountsAdapter) GetAllLeaves(_ []byte) (map[string][]byte, error) {
	return nil, nil
}

// RecreateAllTries -
func (a *accountsAdapter) RecreateAllTries(_ []byte) (map[string]data.Trie, error) {
	return nil, nil
}

// IsInterfaceNil -
func (a *accountsAdapter) IsInterfaceNil() bool {
	return a == nil
}
