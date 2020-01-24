package state_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/mock"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/stretchr/testify/assert"
)

//------- BaseJournalEntryCreation

func TestNewBaseJournalEntryCreation_NilUpdaterShouldErr(t *testing.T) {
	t.Parallel()

	entry, err := state.NewBaseJournalEntryCreation([]byte("key"), nil)

	assert.Nil(t, entry)
	assert.Equal(t, state.ErrNilUpdater, err)
}

func TestNewBaseJournalEntryCreation_NilKeyShouldErr(t *testing.T) {
	t.Parallel()

	entry, err := state.NewBaseJournalEntryCreation(nil, &mock.UpdaterStub{})

	assert.Nil(t, entry)
	assert.Equal(t, state.ErrNilOrEmptyKey, err)
}

func TestNewBaseJournalEntryCreation_ShouldWork(t *testing.T) {
	t.Parallel()

	entry, err := state.NewBaseJournalEntryCreation([]byte("key"), &mock.UpdaterStub{})

	assert.Nil(t, err)
	assert.False(t, check.IfNil(entry))
}

func TestBaseJournalEntryCreation_RevertOkValsShouldWork(t *testing.T) {
	t.Parallel()

	wasCalled := false

	updater := &mock.UpdaterStub{
		UpdateCalled: func(key, value []byte) error {
			wasCalled = true
			return nil
		},
	}

	entry, _ := state.NewBaseJournalEntryCreation([]byte("key"), updater)
	_, err := entry.Revert()

	assert.Nil(t, err)
	assert.True(t, wasCalled)
}

//------- BaseJournalEntryCodeHash

func TestNewBaseJournalEntryCodeHash_NilAccountShouldErr(t *testing.T) {
	t.Parallel()

	entry, err := state.NewBaseJournalEntryCodeHash(nil, nil)

	assert.Nil(t, entry)
	assert.Equal(t, state.ErrNilAccountHandler, err)
}

func TestNewBaseJournalEntryCodeHash_ShouldWork(t *testing.T) {
	t.Parallel()

	entry, err := state.NewBaseJournalEntryCodeHash(mock.NewAccountWrapMock(nil, nil), nil)

	assert.NotNil(t, entry)
	assert.Nil(t, err)
	assert.False(t, entry.IsInterfaceNil())
}

func TestBaseJournalEntryCodeHash_RevertOkValsShouldWork(t *testing.T) {
	t.Parallel()

	testCodeHash := []byte("code hash to revert")
	accnt := mock.NewAccountWrapMock(nil, nil)

	entry, _ := state.NewBaseJournalEntryCodeHash(accnt, testCodeHash)
	_, err := entry.Revert()

	assert.Nil(t, err)
	assert.Equal(t, testCodeHash, accnt.GetCodeHash())
}

//------- BaseJournalEntryNonce

func TestNewBaseJournalEntryNonce_NilAccountShouldErr(t *testing.T) {
	t.Parallel()

	entry, err := state.NewBaseJournalEntryNonce(nil, 0)

	assert.Nil(t, entry)
	assert.Equal(t, state.ErrNilAccountHandler, err)
}

func TestNewBaseJournalEntryNonce_ShouldWork(t *testing.T) {
	t.Parallel()

	accnt, _ := state.NewAccount(mock.NewAddressMock(), &mock.AccountTrackerStub{})
	entry, err := state.NewBaseJournalEntryNonce(accnt, 0)

	assert.NotNil(t, entry)
	assert.Nil(t, err)
	assert.False(t, entry.IsInterfaceNil())
}

func TestBaseJournalEntryNonce_RevertOkValsShouldWork(t *testing.T) {
	t.Parallel()

	nonce := uint64(445)
	accnt, _ := state.NewAccount(mock.NewAddressMock(), &mock.AccountTrackerStub{})
	entry, _ := state.NewBaseJournalEntryNonce(accnt, nonce)
	_, err := entry.Revert()

	assert.Nil(t, err)
	assert.Equal(t, nonce, accnt.Nonce)
}
