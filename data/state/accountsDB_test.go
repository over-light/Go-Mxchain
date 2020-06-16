package state_test

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core/atomic"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/mock"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/state/factory"
	"github.com/ElrondNetwork/elrond-go/data/trie"
	"github.com/stretchr/testify/assert"
)

func generateAccountDBFromTrie(trie data.Trie) *state.AccountsDB {
	accnt, _ := state.NewAccountsDB(trie, &mock.HasherMock{}, &mock.MarshalizerMock{}, &mock.AccountsFactoryStub{
		CreateAccountCalled: func(address []byte) (state.AccountHandler, error) {
			return mock.NewAccountWrapMock(address), nil
		},
	})
	return accnt
}

func generateAccount() *mock.AccountWrapMock {
	return mock.NewAccountWrapMock(make([]byte, 32))
}

func generateAddressAccountAccountsDB(trie data.Trie) ([]byte, *mock.AccountWrapMock, *state.AccountsDB) {
	adr := make([]byte, 32)
	account := mock.NewAccountWrapMock(adr)

	adb := generateAccountDBFromTrie(trie)

	return adr, account, adb
}

//------- NewAccountsDB

func TestNewAccountsDB_WithNilTrieShouldErr(t *testing.T) {
	t.Parallel()

	adb, err := state.NewAccountsDB(
		nil,
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.AccountsFactoryStub{},
	)

	assert.True(t, check.IfNil(adb))
	assert.Equal(t, state.ErrNilTrie, err)
}

func TestNewAccountsDB_WithNilHasherShouldErr(t *testing.T) {
	t.Parallel()

	adb, err := state.NewAccountsDB(
		&mock.TrieStub{},
		nil,
		&mock.MarshalizerMock{},
		&mock.AccountsFactoryStub{},
	)

	assert.True(t, check.IfNil(adb))
	assert.Equal(t, state.ErrNilHasher, err)
}

func TestNewAccountsDB_WithNilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	adb, err := state.NewAccountsDB(
		&mock.TrieStub{},
		&mock.HasherMock{},
		nil,
		&mock.AccountsFactoryStub{},
	)

	assert.True(t, check.IfNil(adb))
	assert.Equal(t, state.ErrNilMarshalizer, err)
}

func TestNewAccountsDB_WithNilAddressFactoryShouldErr(t *testing.T) {
	t.Parallel()

	adb, err := state.NewAccountsDB(
		&mock.TrieStub{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		nil,
	)

	assert.True(t, check.IfNil(adb))
	assert.Equal(t, state.ErrNilAccountFactory, err)
}

func TestNewAccountsDB_OkValsShouldWork(t *testing.T) {
	t.Parallel()

	adb, err := state.NewAccountsDB(
		&mock.TrieStub{},
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
		&mock.AccountsFactoryStub{},
	)

	assert.Nil(t, err)
	assert.False(t, check.IfNil(adb))
}

//------- SaveAccount

func TestAccountsDB_SaveAccountNilAccountShouldErr(t *testing.T) {
	t.Parallel()

	adb := generateAccountDBFromTrie(&mock.TrieStub{})

	err := adb.SaveAccount(nil)
	assert.True(t, errors.Is(err, state.ErrNilAccountHandler))
}

func TestAccountsDB_SaveAccountErrWhenGettingOldAccountShouldErr(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("trie get err")
	adb := generateAccountDBFromTrie(&mock.TrieStub{
		GetCalled: func(key []byte) (i []byte, err error) {
			return nil, expectedErr
		},
	})

	err := adb.SaveAccount(generateAccount())
	assert.Equal(t, expectedErr, err)
}

func TestAccountsDB_SaveAccountNilOldAccount(t *testing.T) {
	t.Parallel()

	adb := generateAccountDBFromTrie(&mock.TrieStub{
		GetCalled: func(key []byte) (i []byte, err error) {
			return nil, nil
		},
		UpdateCalled: func(key, value []byte) error {
			return nil
		},
	})

	err := adb.SaveAccount(generateAccount())
	assert.Nil(t, err)
	assert.Equal(t, 1, adb.JournalLen())
}

func TestAccountsDB_SaveAccountExistingOldAccount(t *testing.T) {
	t.Parallel()

	adb := generateAccountDBFromTrie(&mock.TrieStub{
		GetCalled: func(key []byte) (i []byte, err error) {
			return (&mock.MarshalizerMock{}).Marshal(generateAccount())
		},
		UpdateCalled: func(key, value []byte) error {
			return nil
		},
	})

	err := adb.SaveAccount(generateAccount())
	assert.Nil(t, err)
	assert.Equal(t, 1, adb.JournalLen())
}

func TestAccountsDB_SaveAccountSavesCodeAndDataTrieForUserAccount(t *testing.T) {
	t.Parallel()

	updateCalled := 0
	trieStub := &mock.TrieStub{
		GetCalled: func(key []byte) (i []byte, err error) {
			return nil, nil
		},
		UpdateCalled: func(key, value []byte) error {
			return nil
		},
		RootCalled: func() (i []byte, err error) {
			return []byte("rootHash"), nil
		},
	}

	adb := generateAccountDBFromTrie(&mock.TrieStub{
		GetCalled: func(key []byte) (i []byte, err error) {
			return nil, nil
		},
		UpdateCalled: func(key, value []byte) error {
			updateCalled++
			return nil
		},
		RecreateCalled: func(root []byte) (d data.Trie, err error) {
			return trieStub, nil
		},
	})

	accCode := []byte("code")
	acc := generateAccount()
	acc.SetCode(accCode)
	acc.DataTrieTracker().SaveKeyValue([]byte("key"), []byte("value"))

	err := adb.SaveAccount(acc)
	assert.Nil(t, err)
	assert.Equal(t, 2, updateCalled)
	assert.NotNil(t, acc.GetCodeHash())
	assert.NotNil(t, acc.GetRootHash())
}

func TestAccountsDB_SaveAccountMalfunctionMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	account := generateAccount()
	mockTrie := &mock.TrieStub{}
	marshalizer := &mock.MarshalizerMock{}
	adb, _ := state.NewAccountsDB(mockTrie, &mock.HasherMock{}, marshalizer, &mock.AccountsFactoryStub{
		CreateAccountCalled: func(address []byte) (state.AccountHandler, error) {
			return mock.NewAccountWrapMock(address), nil
		},
	})

	marshalizer.Fail = true

	//should return error
	err := adb.SaveAccount(account)

	assert.NotNil(t, err)
}

func TestAccountsDB_SaveAccountWithSomeValuesShouldWork(t *testing.T) {
	t.Parallel()

	ts := &mock.TrieStub{
		GetCalled: func(key []byte) (i []byte, err error) {
			return nil, nil
		},
		UpdateCalled: func(key, value []byte) error {
			return nil
		},
	}
	_, account, adb := generateAddressAccountAccountsDB(ts)

	//should return error
	err := adb.SaveAccount(account)
	assert.Nil(t, err)
}

//------- RemoveAccount

func TestAccountsDB_RemoveAccountShouldWork(t *testing.T) {
	t.Parallel()

	wasCalled := false
	marsh := &mock.MarshalizerMock{}
	trieStub := &mock.TrieStub{
		GetCalled: func(key []byte) (i []byte, err error) {
			return marsh.Marshal(mock.AccountWrapMock{})
		},
		UpdateCalled: func(key, value []byte) error {
			wasCalled = true
			return nil
		},
	}

	adr := make([]byte, 32)
	adb := generateAccountDBFromTrie(trieStub)

	err := adb.RemoveAccount(adr)
	assert.Nil(t, err)
	assert.True(t, wasCalled)
	assert.Equal(t, 1, adb.JournalLen())
}

//------- LoadAccount

func TestAccountsDB_LoadAccountMalfunctionTrieShouldErr(t *testing.T) {
	t.Parallel()

	trieStub := &mock.TrieStub{}
	adr := make([]byte, 32)
	adb := generateAccountDBFromTrie(trieStub)

	_, err := adb.LoadAccount(adr)
	assert.NotNil(t, err)
}

func TestAccountsDB_LoadAccountNotFoundShouldCreateEmpty(t *testing.T) {
	t.Parallel()

	trieStub := &mock.TrieStub{
		GetCalled: func(key []byte) (i []byte, e error) {
			return nil, nil
		},
		UpdateCalled: func(key, value []byte) error {
			return nil
		},
	}

	adr := make([]byte, 32)
	adb := generateAccountDBFromTrie(trieStub)

	accountExpected := mock.NewAccountWrapMock(adr)
	accountRecovered, err := adb.LoadAccount(adr)

	assert.Equal(t, accountExpected, accountRecovered)
	assert.Nil(t, err)
}

func TestAccountsDB_LoadAccountExistingShouldLoadCodeAndDataTrie(t *testing.T) {
	t.Parallel()

	acc := generateAccount()
	acc.SetRootHash([]byte("root hash"))
	codeHash := []byte("code hash")
	acc.SetCodeHash(codeHash)
	code := []byte("code")
	dataTrie := &mock.TrieStub{}

	trieStub := &mock.TrieStub{
		GetCalled: func(key []byte) (i []byte, e error) {
			if bytes.Equal(key, acc.AddressBytes()) {
				return (&mock.MarshalizerMock{}).Marshal(acc)
			}
			if bytes.Equal(key, codeHash) {
				return code, nil
			}
			return nil, nil
		},
		RecreateCalled: func(root []byte) (d data.Trie, err error) {
			return dataTrie, nil
		},
	}

	adb := generateAccountDBFromTrie(trieStub)
	retrievedAccount, err := adb.LoadAccount(acc.AddressBytes())
	assert.Nil(t, err)

	account, _ := retrievedAccount.(state.UserAccountHandler)
	assert.Equal(t, code, account.GetCode())
	assert.Equal(t, dataTrie, account.DataTrie())
}

//------- GetExistingAccount

func TestAccountsDB_GetExistingAccountMalfunctionTrieShouldErr(t *testing.T) {
	t.Parallel()

	trieStub := &mock.TrieStub{}
	adr := make([]byte, 32)
	adb := generateAccountDBFromTrie(trieStub)

	_, err := adb.GetExistingAccount(adr)
	assert.NotNil(t, err)
}

func TestAccountsDB_GetExistingAccountNotFoundShouldRetNil(t *testing.T) {
	t.Parallel()

	trieStub := &mock.TrieStub{
		GetCalled: func(key []byte) (i []byte, e error) {
			return nil, nil
		},
	}

	adr := make([]byte, 32)
	adb := generateAccountDBFromTrie(trieStub)

	account, err := adb.GetExistingAccount(adr)
	assert.Equal(t, state.ErrAccNotFound, err)
	assert.Nil(t, account)
	//no journal entry shall be created
	assert.Equal(t, 0, adb.JournalLen())
}

func TestAccountsDB_GetExistingAccountFoundShouldRetAccount(t *testing.T) {
	t.Parallel()

	acc := generateAccount()
	acc.SetRootHash([]byte("root hash"))
	codeHash := []byte("code hash")
	acc.SetCodeHash(codeHash)
	code := []byte("code")
	dataTrie := &mock.TrieStub{}

	trieStub := &mock.TrieStub{
		GetCalled: func(key []byte) (i []byte, e error) {
			if bytes.Equal(key, acc.AddressBytes()) {
				return (&mock.MarshalizerMock{}).Marshal(acc)
			}
			if bytes.Equal(key, codeHash) {
				return code, nil
			}
			return nil, nil
		},
		RecreateCalled: func(root []byte) (d data.Trie, err error) {
			return dataTrie, nil
		},
	}

	adb := generateAccountDBFromTrie(trieStub)
	retrievedAccount, err := adb.GetExistingAccount(acc.AddressBytes())
	assert.Nil(t, err)

	account, _ := retrievedAccount.(state.UserAccountHandler)
	assert.Equal(t, code, account.GetCode())
	assert.Equal(t, dataTrie, account.DataTrie())
}

//------- getAccount

func TestAccountsDB_GetAccountAccountNotFound(t *testing.T) {
	t.Parallel()

	trieMock := mock.TrieStub{}
	adr, _, adb := generateAddressAccountAccountsDB(&mock.TrieStub{})

	//Step 1. Create an account + its DbAccount representation
	testAccount := mock.NewAccountWrapMock(adr)
	testAccount.MockValue = 45

	//Step 2. marshalize the DbAccount
	marshalizer := mock.MarshalizerMock{}
	buff, err := marshalizer.Marshal(testAccount)
	assert.Nil(t, err)

	trieMock.GetCalled = func(key []byte) (bytes []byte, e error) {
		//whatever the key is, return the same marshalized DbAccount
		return buff, nil
	}

	adb, _ = state.NewAccountsDB(&trieMock, &mock.HasherMock{}, &marshalizer, &mock.AccountsFactoryStub{
		CreateAccountCalled: func(address []byte) (state.AccountHandler, error) {
			return mock.NewAccountWrapMock(address), nil
		},
	})

	//Step 3. call get, should return a copy of DbAccount, recover an Account object
	recoveredAccount, err := adb.GetAccount(adr)
	assert.Nil(t, err)

	//Step 4. Let's test
	assert.Equal(t, testAccount.MockValue, recoveredAccount.(*mock.AccountWrapMock).MockValue)
}

//------- loadCode

func TestAccountsDB_LoadCodeWrongHashLengthShouldErr(t *testing.T) {
	t.Parallel()

	_, account, adb := generateAddressAccountAccountsDB(&mock.TrieStub{})

	account.SetCodeHash([]byte("AAAA"))

	err := adb.LoadCode(account)
	assert.NotNil(t, err)
}

func TestAccountsDB_LoadCodeMalfunctionTrieShouldErr(t *testing.T) {
	t.Parallel()

	adr := make([]byte, 32)
	account := generateAccount()
	mockTrie := &mock.TrieStub{}
	adb := generateAccountDBFromTrie(mockTrie)

	//just search a hash. Any hash will do
	account.SetCodeHash(adr)

	err := adb.LoadCode(account)
	assert.NotNil(t, err)
}

func TestAccountsDB_LoadCodeOkValsShouldWork(t *testing.T) {
	t.Parallel()

	adr, account, _ := generateAddressAccountAccountsDB(&mock.TrieStub{})

	trieStub := mock.TrieStub{}
	trieStub.GetCalled = func(key []byte) (bytes []byte, e error) {
		//will return adr.Bytes() so its hash will correspond to adr.Hash()
		return adr, nil
	}
	marshalizer := mock.MarshalizerMock{}
	adb, _ := state.NewAccountsDB(&trieStub, &mock.HasherMock{}, &marshalizer, &mock.AccountsFactoryStub{
		CreateAccountCalled: func(address []byte) (state.AccountHandler, error) {
			return mock.NewAccountWrapMock(address), nil
		},
	})

	//just search a hash. Any hash will do
	account.SetCodeHash(adr)

	err := adb.LoadCode(account)
	assert.Nil(t, err)
	assert.Equal(t, adr, account.GetCode())
}

//------- RetrieveData

func TestAccountsDB_LoadDataNilRootShouldRetNil(t *testing.T) {
	t.Parallel()

	_, account, adb := generateAddressAccountAccountsDB(&mock.TrieStub{})

	//since root is nil, result should be nil and data trie should be nil
	err := adb.LoadDataTrie(account)
	assert.Nil(t, err)
	assert.Nil(t, account.DataTrie())
}

func TestAccountsDB_LoadDataBadLengthShouldErr(t *testing.T) {
	t.Parallel()

	_, account, adb := generateAddressAccountAccountsDB(&mock.TrieStub{})

	account.SetRootHash([]byte("12345"))

	//should return error
	err := adb.LoadDataTrie(account)
	assert.NotNil(t, err)
}

func TestAccountsDB_LoadDataMalfunctionTrieShouldErr(t *testing.T) {
	t.Parallel()

	account := generateAccount()
	account.SetRootHash([]byte("12345"))

	mockTrie := &mock.TrieStub{}
	adb := generateAccountDBFromTrie(mockTrie)

	//should return error
	err := adb.LoadDataTrie(account)
	assert.NotNil(t, err)
}

func TestAccountsDB_LoadDataNotFoundRootShouldReturnErr(t *testing.T) {
	t.Parallel()

	_, account, adb := generateAddressAccountAccountsDB(&mock.TrieStub{})

	rootHash := make([]byte, mock.HasherMock{}.Size())
	rootHash[0] = 1
	account.SetRootHash(rootHash)

	//should return error
	err := adb.LoadDataTrie(account)
	assert.NotNil(t, err)
	fmt.Println(err.Error())
}

func TestAccountsDB_LoadDataWithSomeValuesShouldWork(t *testing.T) {
	t.Parallel()

	rootHash := make([]byte, mock.HasherMock{}.Size())
	rootHash[0] = 1
	keyRequired := []byte{65, 66, 67}
	val := []byte{32, 33, 34}

	trieVal := append(val, keyRequired...)
	trieVal = append(trieVal, []byte("identifier")...)

	dataTrie := &mock.TrieStub{
		GetCalled: func(key []byte) (i []byte, e error) {
			if bytes.Equal(key, keyRequired) {
				return trieVal, nil
			}

			return nil, nil
		},
	}

	account := generateAccount()
	mockTrie := &mock.TrieStub{
		RecreateCalled: func(root []byte) (trie data.Trie, e error) {
			if !bytes.Equal(root, rootHash) {
				return nil, errors.New("bad root hash")
			}

			return dataTrie, nil
		},
	}
	adb := generateAccountDBFromTrie(mockTrie)

	account.SetRootHash(rootHash)

	//should not return error
	err := adb.LoadDataTrie(account)
	assert.Nil(t, err)

	//verify data
	dataRecov, err := account.DataTrieTracker().RetrieveValue(keyRequired)
	assert.Nil(t, err)
	assert.Equal(t, val, dataRecov)
}

//------- Commit

func TestAccountsDB_CommitShouldCallCommitFromTrie(t *testing.T) {
	t.Parallel()

	commitCalled := 0
	marsh := &mock.MarshalizerMock{}
	serializedAccount, _ := marsh.Marshal(mock.AccountWrapMock{})
	trieStub := mock.TrieStub{
		CommitCalled: func() error {
			commitCalled++

			return nil
		},
		RootCalled: func() (i []byte, e error) {
			return nil, nil
		},
		GetCalled: func(key []byte) (i []byte, err error) {
			return serializedAccount, nil
		},
		RecreateCalled: func(root []byte) (trie data.Trie, err error) {
			return &mock.TrieStub{
				GetCalled: func(key []byte) (i []byte, err error) {
					return []byte("doge"), nil
				},
				UpdateCalled: func(key, value []byte) error {
					return nil
				},
				CommitCalled: func() error {
					commitCalled++

					return nil
				},
			}, nil
		},
	}

	adb := generateAccountDBFromTrie(&trieStub)

	state2, _ := adb.LoadAccount(make([]byte, 32))
	state2.(state.UserAccountHandler).DataTrieTracker().SaveKeyValue([]byte("dog"), []byte("puppy"))
	_ = adb.SaveAccount(state2)

	_, err := adb.Commit()
	assert.Nil(t, err)
	//one commit for the JournalEntryData and one commit for the main trie
	assert.Equal(t, 2, commitCalled)
}

//------- RecreateTrie

func TestAccountsDB_RecreateTrieMalfunctionTrieShouldErr(t *testing.T) {
	t.Parallel()

	wasCalled := false

	errExpected := errors.New("failure")
	trieStub := mock.TrieStub{}
	trieStub.RecreateCalled = func(root []byte) (tree data.Trie, e error) {
		wasCalled = true
		return nil, errExpected
	}

	adb := generateAccountDBFromTrie(&trieStub)

	err := adb.RecreateTrie(nil)
	assert.Equal(t, errExpected, err)
	assert.True(t, wasCalled)
}

func TestAccountsDB_RecreateTrieOutputsNilTrieShouldErr(t *testing.T) {
	t.Parallel()

	wasCalled := false

	trieStub := mock.TrieStub{}
	trieStub.RecreateCalled = func(root []byte) (tree data.Trie, e error) {
		wasCalled = true
		return nil, nil
	}

	adb := generateAccountDBFromTrie(&trieStub)
	err := adb.RecreateTrie(nil)

	assert.Equal(t, state.ErrNilTrie, err)
	assert.True(t, wasCalled)

}

func TestAccountsDB_RecreateTrieOkValsShouldWork(t *testing.T) {
	t.Parallel()

	wasCalled := false

	trieStub := mock.TrieStub{}
	trieStub.RecreateCalled = func(root []byte) (tree data.Trie, e error) {
		wasCalled = true
		return &mock.TrieStub{}, nil
	}

	adb := generateAccountDBFromTrie(&trieStub)
	err := adb.RecreateTrie(nil)

	assert.Nil(t, err)
	assert.True(t, wasCalled)

}

func TestAccountsDB_CancelPrune(t *testing.T) {
	t.Parallel()

	cancelPruneWasCalled := false
	trieStub := &mock.TrieStub{
		CancelPruneCalled: func(rootHash []byte, identifier data.TriePruningIdentifier) {
			cancelPruneWasCalled = true
		},
	}
	adb := generateAccountDBFromTrie(trieStub)
	adb.CancelPrune([]byte("roothash"), data.OldRoot)

	assert.True(t, cancelPruneWasCalled)
}

func TestAccountsDB_PruneTrie(t *testing.T) {
	t.Parallel()

	pruneTrieWasCalled := atomic.Flag{}
	trieStub := &mock.TrieStub{
		PruneCalled: func(rootHash []byte, identifier data.TriePruningIdentifier) {
			pruneTrieWasCalled.Set()
		},
	}
	adb := generateAccountDBFromTrie(trieStub)
	adb.PruneTrie([]byte("roothash"), data.OldRoot)
	assert.True(t, pruneTrieWasCalled.IsSet())
}

func TestAccountsDB_SnapshotState(t *testing.T) {
	t.Parallel()

	takeSnapshotWasCalled := false
	snapshotMut := sync.Mutex{}
	trieStub := &mock.TrieStub{
		TakeSnapshotCalled: func(rootHash []byte) {
			snapshotMut.Lock()
			takeSnapshotWasCalled = true
			snapshotMut.Unlock()
		},
	}
	adb := generateAccountDBFromTrie(trieStub)
	adb.SnapshotState([]byte("roothash"))
	time.Sleep(time.Second)

	snapshotMut.Lock()
	assert.True(t, takeSnapshotWasCalled)
	snapshotMut.Unlock()
}

func TestAccountsDB_SetStateCheckpoint(t *testing.T) {
	t.Parallel()

	setCheckPointWasCalled := false
	snapshotMut := sync.Mutex{}
	trieStub := &mock.TrieStub{
		SetCheckpointCalled: func(rootHash []byte) {
			snapshotMut.Lock()
			setCheckPointWasCalled = true
			snapshotMut.Unlock()
		},
	}
	adb := generateAccountDBFromTrie(trieStub)
	adb.SetStateCheckpoint([]byte("roothash"))
	time.Sleep(time.Second)

	snapshotMut.Lock()
	assert.True(t, setCheckPointWasCalled)
	snapshotMut.Unlock()
}

func TestAccountsDB_IsPruningEnabled(t *testing.T) {
	t.Parallel()

	trieStub := &mock.TrieStub{
		IsPruningEnabledCalled: func() bool {
			return true
		},
	}
	adb := generateAccountDBFromTrie(trieStub)
	res := adb.IsPruningEnabled()

	assert.Equal(t, true, res)
}

func TestAccountsDB_RevertToSnapshotOutOfBounds(t *testing.T) {
	t.Parallel()

	trieStub := &mock.TrieStub{}
	adb := generateAccountDBFromTrie(trieStub)

	err := adb.RevertToSnapshot(1)
	assert.Equal(t, state.ErrSnapshotValueOutOfBounds, err)
}

func TestAccountsDB_RevertToSnapshotShouldWork(t *testing.T) {
	t.Parallel()

	marsh := &mock.MarshalizerMock{}
	hsh := mock.HasherMock{}
	accFactory := factory.NewAccountCreator()
	storageManager, _ := trie.NewTrieStorageManagerWithoutPruning(mock.NewMemDbMock())
	maxTrieLevelInMemory := uint(5)
	tr, _ := trie.NewTrie(storageManager, marsh, hsh, maxTrieLevelInMemory)

	adb, _ := state.NewAccountsDB(tr, hsh, marsh, accFactory)

	acc, _ := adb.LoadAccount(make([]byte, 32))
	acc.(state.UserAccountHandler).SetCode([]byte("code"))
	_ = adb.SaveAccount(acc)

	err := adb.RevertToSnapshot(0)
	assert.Nil(t, err)

	expectedRoot := make([]byte, 32)
	root, err := adb.RootHash()
	assert.Nil(t, err)
	assert.Equal(t, expectedRoot, root)
}

func TestAccountsDB_RevertToSnapshotWithoutLastRootHashSet(t *testing.T) {
	t.Parallel()

	marsh := &mock.MarshalizerMock{}
	hsh := mock.HasherMock{}
	accFactory := factory.NewAccountCreator()
	storageManager, _ := trie.NewTrieStorageManagerWithoutPruning(mock.NewMemDbMock())
	maxTrieLevelInMemory := uint(5)
	tr, _ := trie.NewTrie(storageManager, marsh, hsh, maxTrieLevelInMemory)
	adb, _ := state.NewAccountsDB(tr, hsh, marsh, accFactory)

	err := adb.RevertToSnapshot(0)
	assert.Nil(t, err)

	rootHash, err := adb.RootHash()
	assert.Nil(t, err)
	assert.Equal(t, make([]byte, 32), rootHash)
}

func TestAccountsDB_RecreateTrieInvalidatesJournalEntries(t *testing.T) {
	t.Parallel()

	marsh := &mock.MarshalizerMock{}
	hsh := mock.HasherMock{}
	accFactory := factory.NewAccountCreator()
	storageManager, _ := trie.NewTrieStorageManagerWithoutPruning(mock.NewMemDbMock())
	maxTrieLevelInMemory := uint(5)
	tr, _ := trie.NewTrie(storageManager, marsh, hsh, maxTrieLevelInMemory)
	adb, _ := state.NewAccountsDB(tr, hsh, marsh, accFactory)

	address := make([]byte, 32)
	key := []byte("key")
	value := []byte("value")

	acc, _ := adb.LoadAccount(address)
	_ = adb.SaveAccount(acc)
	rootHash, _ := adb.Commit()

	acc, _ = adb.LoadAccount(address)
	acc.(state.UserAccountHandler).SetCode([]byte("code"))
	_ = adb.SaveAccount(acc)

	acc, _ = adb.LoadAccount(address)
	acc.(state.UserAccountHandler).IncreaseNonce(1)
	_ = adb.SaveAccount(acc)

	acc, _ = adb.LoadAccount(address)
	acc.(state.UserAccountHandler).DataTrieTracker().SaveKeyValue(key, value)
	_ = adb.SaveAccount(acc)

	assert.Equal(t, 5, adb.JournalLen())
	err := adb.RecreateTrie(rootHash)
	assert.Nil(t, err)
	assert.Equal(t, 0, adb.JournalLen())
}

func TestAccountsDB_RootHash(t *testing.T) {
	t.Parallel()

	rootHashCalled := false
	rootHash := []byte("root hash")
	trieStub := &mock.TrieStub{
		RootCalled: func() (i []byte, err error) {
			rootHashCalled = true
			return rootHash, nil
		},
	}
	adb := generateAccountDBFromTrie(trieStub)
	res, err := adb.RootHash()
	assert.Nil(t, err)
	assert.True(t, rootHashCalled)
	assert.Equal(t, rootHash, res)
}

func TestAccountsDB_GetAllLeavesWrongRootHash(t *testing.T) {
	t.Parallel()

	trieStub := &mock.TrieStub{
		RecreateCalled: func(root []byte) (d data.Trie, err error) {
			return nil, nil
		},
	}

	adb := generateAccountDBFromTrie(trieStub)
	res, err := adb.GetAllLeaves([]byte("root hash"))
	assert.Equal(t, state.ErrNilTrie, err)
	assert.Nil(t, res)
}

func TestAccountsDB_GetAllLeaves(t *testing.T) {
	t.Parallel()

	recreateCalled := false
	getAllLeavesCalled := false
	trieStub := &mock.TrieStub{
		RecreateCalled: func(root []byte) (d data.Trie, err error) {
			recreateCalled = true
			return &mock.TrieStub{
				GetAllLeavesCalled: func() (m map[string][]byte, err error) {
					getAllLeavesCalled = true
					return nil, nil
				},
			}, nil
		},
	}

	adb := generateAccountDBFromTrie(trieStub)
	_, err := adb.GetAllLeaves([]byte("root hash"))
	assert.Nil(t, err)
	assert.True(t, recreateCalled)
	assert.True(t, getAllLeavesCalled)
}

func TestAccountsDB_SaveAccountSavesCodeIfCodeHashIsSet(t *testing.T) {
	t.Parallel()

	marsh := &mock.MarshalizerMock{}
	hsh := mock.HasherMock{}
	accFactory := factory.NewAccountCreator()
	storageManager, _ := trie.NewTrieStorageManagerWithoutPruning(mock.NewMemDbMock())
	maxTrieLevelInMemory := uint(5)
	tr, _ := trie.NewTrie(storageManager, marsh, hsh, maxTrieLevelInMemory)
	adb, _ := state.NewAccountsDB(tr, hsh, marsh, accFactory)

	addr := make([]byte, 32)
	acc, _ := adb.LoadAccount(addr)
	userAcc := acc.(state.UserAccountHandler)

	code := []byte("code")
	userAcc.SetCode(code)
	codeHash := mock.HasherMock{}.Compute(string(code))
	userAcc.SetCodeHash(codeHash)

	_ = adb.SaveAccount(acc)

	codeFromTrie, err := tr.Get(codeHash)
	assert.Nil(t, err)
	assert.Equal(t, code, codeFromTrie)
}
