package state_test

import (
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/data/mock"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/stretchr/testify/assert"
)

func TestMetaAccount_MarshalUnmarshal_ShouldWork(t *testing.T) {
	t.Parallel()

	addr := &mock.AddressMock{}
	addrTr := &mock.AccountTrackerStub{}
	acnt, _ := state.NewMetaAccount(addr, addrTr)

	marshalizer := mock.MarshalizerMock{}
	buff, _ := marshalizer.Marshal(&acnt)

	acntRecovered, _ := state.NewMetaAccount(addr, addrTr)
	_ = marshalizer.Unmarshal(acntRecovered, buff)

	assert.Equal(t, acnt, acntRecovered)
}

func TestMetaAccount_NewAccountNilAddress(t *testing.T) {
	t.Parallel()

	acc, err := state.NewMetaAccount(nil, &mock.AccountTrackerStub{})

	assert.Nil(t, acc)
	assert.Equal(t, err, state.ErrNilAddressContainer)
}

func TestMetaAccount_NewMetaAccountNilAaccountTracker(t *testing.T) {
	t.Parallel()

	acc, err := state.NewMetaAccount(&mock.AddressMock{}, nil)

	assert.Nil(t, acc)
	assert.Equal(t, err, state.ErrNilAccountTracker)
}

func TestMetaAccount_NewMetaAccountOk(t *testing.T) {
	t.Parallel()

	acc, err := state.NewMetaAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.False(t, acc.IsInterfaceNil())
}

func TestMetaAccount_AddressContainer(t *testing.T) {
	t.Parallel()

	addr := &mock.AddressMock{}
	acc, err := state.NewMetaAccount(addr, &mock.AccountTrackerStub{})

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, addr, acc.AddressContainer())
}

func TestMetaAccount_GetCode(t *testing.T) {
	t.Parallel()

	acc, err := state.NewMetaAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})
	assert.Nil(t, err)

	code := []byte("code")
	acc.SetCode(code)

	assert.NotNil(t, acc)
	assert.Equal(t, code, acc.GetCode())
}

func TestMetaAccount_GetCodeHash(t *testing.T) {
	t.Parallel()

	acc, err := state.NewMetaAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})
	assert.Nil(t, err)

	code := []byte("code")
	acc.CodeHash = code

	assert.NotNil(t, acc)
	assert.Equal(t, code, acc.GetCodeHash())
}

func TestMetaAccount_SetCodeHash(t *testing.T) {
	t.Parallel()

	acc, err := state.NewMetaAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})
	assert.Nil(t, err)

	code := []byte("code")
	acc.SetCodeHash(code)

	assert.NotNil(t, acc)
	assert.Equal(t, code, acc.GetCodeHash())
}

func TestMetaAccount_GetRootHash(t *testing.T) {
	t.Parallel()

	acc, err := state.NewMetaAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})
	assert.Nil(t, err)

	root := []byte("root")
	acc.RootHash = root

	assert.NotNil(t, acc)
	assert.Equal(t, root, acc.GetRootHash())
}

func TestMetaAccount_SetRootHash(t *testing.T) {
	t.Parallel()

	acc, err := state.NewMetaAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})
	assert.Nil(t, err)

	root := []byte("root")
	acc.SetRootHash(root)

	assert.NotNil(t, acc)
	assert.Equal(t, root, acc.GetRootHash())
}

func TestMetaAccount_DataTrie(t *testing.T) {
	t.Parallel()

	acc, err := state.NewMetaAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})
	assert.Nil(t, err)

	trie := &mock.TrieStub{}
	acc.SetDataTrie(trie)

	assert.NotNil(t, acc)
	assert.Equal(t, trie, acc.DataTrie())
}

func TestMetaAccount_SetRoundWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewMetaAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	round := uint64(0)
	err = acc.SetRoundWithJournal(round)

	assert.NotNil(t, acc)
	assert.Equal(t, round, acc.Round)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestMetaAccount_SetGetNonce(t *testing.T) {
	t.Parallel()

	acc, _ := state.NewMetaAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})

	nonce := uint64(37)
	acc.SetNonce(nonce)
	assert.Equal(t, nonce, acc.GetNonce())
}

func TestMetaAccount_SetNonceWithJournal(t *testing.T) {
	t.Parallel()

	nonce := uint64(5)
	journalizeWasCalled := false
	saveAccountWasCalled := false
	address := &mock.AddressMock{}
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeWasCalled = true
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountWasCalled = true
			assert.Equal(t, nonce, accountHandler.GetNonce())
			return nil
		},
	}
	acc, _ := state.NewMetaAccount(address, tracker)
	acc.SetNonce(nonce)

	err := acc.SetNonceWithJournal(nonce)
	assert.Nil(t, err)
	assert.True(t, journalizeWasCalled)
	assert.True(t, saveAccountWasCalled)
}

func TestMetaAccount_DataTrieTracker(t *testing.T) {
	t.Parallel()

	acc, _ := state.NewMetaAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})

	dtt := acc.DataTrieTracker()
	assert.NotNil(t, dtt)
}

func TestMetaAccount_SetTxCountWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewMetaAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	txCount := big.NewInt(15)
	err = acc.SetTxCountWithJournal(txCount)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, txCount, acc.TxCount)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestMetaAccount_SetCodeHashWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewMetaAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	codeHash := []byte("codehash")
	err = acc.SetCodeHashWithJournal(codeHash)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, codeHash, acc.CodeHash)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestMetaAccount_SetMiniBlocksDataWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewMetaAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	mbs := make([]*state.MiniBlockData, 2)
	err = acc.SetMiniBlocksDataWithJournal(mbs)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, mbs, acc.MiniBlocks)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestMetaAccount_SetShardRootHashWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewMetaAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	shardRootHash := []byte("shardroothash")
	err = acc.SetShardRootHashWithJournal(shardRootHash)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, shardRootHash, acc.ShardRootHash)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}
