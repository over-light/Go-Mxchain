package state

import (
	"fmt"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/state/mock"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/trie/encoding"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

var testHasher = mock.HasherMock{}
var testMarshalizer = mock.MarshalizerMock{}

func createAccountsDB(t *testing.T) *AccountsDB {
	adb := NewAccountsDB(mock.NewMockTrie(), &testHasher, &testMarshalizer)

	return adb
}

func createAddress(t *testing.T, buff []byte) *Address {
	adr, err := FromPubKeyBytes(buff)
	assert.Nil(t, err)

	return adr
}

func createAccountState(_ *testing.T, address Address) *AccountState {
	return NewAccountState(address, Account{}, testHasher)
}

func TestAccountsDB_RetrieveCode_NilCodeHash_ShouldRetNil(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)

	adb := createAccountsDB(t)

	//since code hash is nil, result should be nil and code should be nil
	err := adb.RetrieveCode(state)
	assert.Nil(t, err)
	assert.Nil(t, state.Code)
}

func TestAccountsDB_RetrieveCode_NilTrie_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.CodeHash = []byte("12345")

	adb := createAccountsDB(t)

	adb.MainTrie = nil

	err := adb.RetrieveCode(state)
	assert.NotNil(t, err)
}

func TestAccountsDB_RetrieveCode_BadLength_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.CodeHash = []byte("12345")

	adb := createAccountsDB(t)

	//should return error
	err := adb.RetrieveCode(state)
	assert.NotNil(t, err)
}

func TestAccountsDB_RetrieveCode_MalfunctionTrie_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.CodeHash = testHasher.Compute("12345")

	adb := createAccountsDB(t)
	adb.MainTrie.(*mock.TrieMock).Fail = true

	//should return error
	err := adb.RetrieveCode(state)
	assert.NotNil(t, err)
}

func TestAccountsDB_RetrieveCode_NotFound_ShouldRetNil(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.CodeHash = testHasher.Compute("12345")

	adb := createAccountsDB(t)

	//should return nil
	err := adb.RetrieveCode(state)
	assert.Nil(t, err)
	assert.Nil(t, state.Code)
}

func TestAccountsDB_RetrieveCode_Found_ShouldWork(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.CodeHash = testHasher.Compute("12345")

	adb := createAccountsDB(t)
	//manually put it in the trie
	adb.MainTrie.Update(state.CodeHash, []byte{68, 69, 70})

	//should return nil
	err := adb.RetrieveCode(state)
	assert.Nil(t, err)
	assert.Equal(t, state.Code, []byte{68, 69, 70})
}

func TestAccountsDB_RetrieveData_NilRoot_ShouldRetNil(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)

	adb := createAccountsDB(t)

	//since root is nil, result should be nil and data trie should be nil
	err := adb.RetrieveData(state)
	assert.Nil(t, err)
	assert.Nil(t, state.Data)
}

func TestAccountsDB_RetrieveData_NilTrie_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = []byte("12345")

	adb := createAccountsDB(t)

	adb.MainTrie = nil

	err := adb.RetrieveData(state)
	assert.NotNil(t, err)
}

func TestAccountsDB_RetrieveData_BadLength_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = []byte("12345")

	adb := createAccountsDB(t)

	//should return error
	err := adb.RetrieveData(state)
	assert.NotNil(t, err)
}

func TestAccountsDB_RetrieveData_MalfunctionTrie_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = testHasher.Compute("12345")

	adb := createAccountsDB(t)
	adb.MainTrie.(*mock.TrieMock).Fail = true

	//should return error
	err := adb.RetrieveData(state)
	assert.NotNil(t, err)
}

func TestAccountsDB_RetrieveData_NotFoundRoot_ShouldReturnEmpty(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = testHasher.Compute("12345")

	adb := createAccountsDB(t)

	//should return error
	err := adb.RetrieveData(state)
	assert.Nil(t, err)
	assert.Equal(t, make([]byte, encoding.HashLength), state.Data.Root())
}

func TestAccountsDB_RetrieveData_WithSomeValues_ShouldWork(t *testing.T) {
	//test simulates creation of a new account, data trie retrieval,
	//adding a (key, value) pair in that data trie, commiting changes
	//and then reloading the data trie based on the root hash generated before
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = testHasher.Compute("12345")

	adb := createAccountsDB(t)

	//should not return error
	err := adb.RetrieveData(state)
	assert.Nil(t, err)
	assert.NotNil(t, state.Data)

	fmt.Printf("Root: %v\n", state.Root)

	//add something to the data trie
	state.Data.Update([]byte{65, 66, 67}, []byte{68, 69, 70})
	state.Root = state.Data.Root()
	fmt.Printf("state.Root is %v\n", state.Root)
	hashResult, err := state.Data.Commit(nil)

	assert.Equal(t, hashResult, state.Root)

	//make data trie null and then retrieve the trie and search the data
	state.Data = nil
	err = adb.RetrieveData(state)
	assert.Nil(t, err)
	assert.NotNil(t, state.Data)

	val, err := state.Data.Get([]byte{65, 66, 67})
	assert.Nil(t, err)
	assert.Equal(t, []byte{68, 69, 70}, val)
}

func TestAccountsDB_PutCode_NilCodeHash_ShouldRetNil(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = state.AddrHash

	adb := createAccountsDB(t)

	err := adb.PutCode(state, nil)
	assert.Nil(t, err)
	assert.Nil(t, state.CodeHash)
	assert.Nil(t, state.Code)
}

func TestAccountsDB_PutCode_EmptyCodeHash_ShouldRetNil(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = state.AddrHash

	adb := createAccountsDB(t)

	err := adb.PutCode(state, make([]byte, 0))
	assert.Nil(t, err)
	assert.Nil(t, state.CodeHash)
	assert.Nil(t, state.Code)
}

func TestAccountsDB_PutCode_NilTrie_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = []byte("12345")

	adb := createAccountsDB(t)

	adb.MainTrie = nil

	err := adb.PutCode(state, []byte{65})
	assert.NotNil(t, err)
}

func TestAccountsDB_PutCode_MalfunctionTrie_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = testHasher.Compute("12345")

	adb := createAccountsDB(t)
	adb.MainTrie.(*mock.TrieMock).Fail = true

	//should return error
	err := adb.PutCode(state, []byte{65})
	assert.NotNil(t, err)
}

func TestAccountsDB_PutCode_Malfunction2Trie_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = testHasher.Compute("12345")

	adb := createAccountsDB(t)
	adb.MainTrie.(*mock.TrieMock).FailRecreate = true

	//should return error
	err := adb.PutCode(state, []byte{65})
	assert.NotNil(t, err)
}

func TestAccountsDB_PutCode_WithSomeValues_ShouldWork(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = testHasher.Compute("12345")

	adb := createAccountsDB(t)

	snapshotRoot := adb.MainTrie.Root()

	err := adb.PutCode(state, []byte("Smart contract code"))
	assert.Nil(t, err)
	assert.NotNil(t, state.CodeHash)
	assert.Equal(t, []byte("Smart contract code"), state.Code)

	fmt.Printf("SC code is at address: %v\n", state.CodeHash)

	//retrieve directly from the trie
	data, err := adb.MainTrie.Get(state.CodeHash)
	assert.Nil(t, err)
	assert.Equal(t, data, state.Code)

	fmt.Printf("SC code is: %v\n", string(data))

	//main root trie should have been modified
	assert.NotEqual(t, snapshotRoot, adb.MainTrie.Root())
}

func TestAccountsDB_HasAccount_NilTrie_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = []byte("12345")

	adb := createAccountsDB(t)
	adb.MainTrie = nil

	val, err := adb.HasAccount(*adr)
	assert.NotNil(t, err)
	assert.False(t, val)
}

func TestAccountsDB_HasAccount_MalfunctionTrie_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = testHasher.Compute("12345")

	adb := createAccountsDB(t)
	adb.MainTrie.(*mock.TrieMock).Fail = true

	//should return error
	val, err := adb.HasAccount(*adr)
	assert.NotNil(t, err)
	assert.False(t, val)
}

func TestAccountsDB_HasAccount_NotFound_ShouldRetFalse(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = testHasher.Compute("12345")

	adb := createAccountsDB(t)

	//should return false
	val, err := adb.HasAccount(*adr)
	assert.Nil(t, err)
	assert.False(t, val)
}

func TestAccountsDB_HasAccount_Found_ShouldRetTrue(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = testHasher.Compute("12345")

	adb := createAccountsDB(t)
	adb.MainTrie.Update(testHasher.Compute(string(adr.Bytes())), []byte{65})

	//should return true
	val, err := adb.HasAccount(*adr)
	assert.Nil(t, err)
	assert.True(t, val)
}

func TestAccountsDB_SaveAccountState_NilTrie_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = []byte("12345")

	adb := createAccountsDB(t)
	adb.MainTrie = nil

	err := adb.SaveAccountState(state)
	assert.NotNil(t, err)
}

func TestAccountsDB_SaveAccountState_NilState_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = []byte("12345")

	adb := createAccountsDB(t)
	adb.MainTrie = nil

	err := adb.SaveAccountState(nil)
	assert.NotNil(t, err)
}

func TestAccountsDB_SaveAccountState_MalfunctionTrie_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = testHasher.Compute("12345")

	adb := createAccountsDB(t)
	adb.MainTrie.(*mock.TrieMock).Fail = true

	//should return error
	err := adb.SaveAccountState(state)
	assert.NotNil(t, err)
}

func TestAccountsDB_SaveAccountState_MalfunctionMarshalizer_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = testHasher.Compute("12345")

	adb := createAccountsDB(t)
	adb.marsh.(*mock.MarshalizerMock).Fail = true
	defer func() {
		//call defer to reset the fail pointer so other tests will work as expected
		adb.marsh.(*mock.MarshalizerMock).Fail = false
	}()

	//should return error
	err := adb.SaveAccountState(state)

	assert.NotNil(t, err)
}

func TestAccountsDB_SaveAccountState_WithSomeValues_ShouldWork(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Root = testHasher.Compute("12345")

	adb := createAccountsDB(t)

	//should return error
	err := adb.SaveAccountState(state)
	assert.Nil(t, err)
}

func TestAccountsDB_GetOrCreateAccount_NilTrie_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)

	adb := createAccountsDB(t)
	adb.MainTrie = nil

	acnt, err := adb.GetOrCreateAccount(*adr)
	assert.NotNil(t, err)
	assert.Nil(t, acnt)

}

func TestAccountsDB_GetOrCreateAccount_MalfunctionTrie_ShouldErr(t *testing.T) {
	//test when there is no such account

	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)

	adb := createAccountsDB(t)
	adb.MainTrie.(*mock.TrieMock).FailGet = true
	adb.MainTrie.(*mock.TrieMock).TTFGet = 1
	adb.MainTrie.(*mock.TrieMock).FailUpdate = true
	adb.MainTrie.(*mock.TrieMock).TTFUpdate = 0

	acnt, err := adb.GetOrCreateAccount(*adr)
	assert.NotNil(t, err)
	assert.Nil(t, acnt)

}

func TestAccountsDB_GetOrCreateAccount_Malfunction2Trie_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)

	adb := createAccountsDB(t)
	adb.SaveAccountState(state)

	adb.MainTrie.(*mock.TrieMock).FailGet = true
	adb.MainTrie.(*mock.TrieMock).TTFGet = 1

	acnt, err := adb.GetOrCreateAccount(*adr)
	assert.NotNil(t, err)
	assert.Nil(t, acnt)

}

func TestAccountsDB_GetOrCreateAccount_MalfunctionMarshalizer_ShouldErr(t *testing.T) {
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)

	adb := createAccountsDB(t)
	adb.SaveAccountState(state)

	adb.marsh.(*mock.MarshalizerMock).Fail = true
	defer func() {
		adb.marsh.(*mock.MarshalizerMock).Fail = false
	}()

	acnt, err := adb.GetOrCreateAccount(*adr)
	assert.NotNil(t, err)
	assert.Nil(t, acnt)

}

func TestAccountsDB_GetOrCreateAccount_ReturnExistingAccnt_ShouldWork(t *testing.T) {
	//test when the account exists
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Balance = big.NewInt(40)

	adb := createAccountsDB(t)
	adb.SaveAccountState(state)

	acnt, err := adb.GetOrCreateAccount(*adr)
	assert.Nil(t, err)
	assert.NotNil(t, acnt)
	assert.Equal(t, acnt.Balance, big.NewInt(40))
}

func TestAccountsDB_GetOrCreateAccount_ReturnNotFoundAccnt_ShouldWork(t *testing.T) {
	//test when the account does not exists
	testHash := testHasher.Compute("ABCDEFGHJIJKLMNOP")

	adr := createAddress(t, testHash)
	state := createAccountState(t, *adr)
	state.Balance = big.NewInt(40)

	adb := createAccountsDB(t)

	//same address of the unsaved account
	acnt, err := adb.GetOrCreateAccount(*adr)
	assert.Nil(t, err)
	assert.NotNil(t, acnt)
	assert.Equal(t, acnt.Balance, big.NewInt(0))
}

func TestAccountsDB_Commit_MalfunctionTrie_ShouldErr(t *testing.T) {
	//test creates 2 accounts (one with a data root)
	//verifies that commit saves the new tries and that can be loaded back
	//second data trie malfunctions

	testHash1 := testHasher.Compute("ABCDEFGHJIJKLMNOP")
	adr1 := createAddress(t, testHash1)

	testHash2 := testHasher.Compute("ABCDEFGHJIJKLMNOPQ")
	adr2 := createAddress(t, testHash2)

	adb := createAccountsDB(t)

	//first account has the balance of 40
	state1, err := adb.GetOrCreateAccount(*adr1)
	assert.Nil(t, err)
	state1.Balance = big.NewInt(40)

	//second account has the balance of 50 and something in data root
	state2, err := adb.GetOrCreateAccount(*adr2)
	assert.Nil(t, err)

	state2.Balance = big.NewInt(50)
	state2.Root = make([]byte, encoding.HashLength)
	err = adb.RetrieveData(state2)
	assert.Nil(t, err)

	state2.Data.Update([]byte{65, 66, 67}, []byte{68, 69, 70})

	//states are now prepared, committing

	state2.Data.(*mock.TrieMock).Fail = true

	err = adb.Commit()
	assert.NotNil(t, err)
}

func TestAccountsDB_Commit_MalfunctionMainTrie_ShouldErr(t *testing.T) {
	//test creates 2 accounts (one with a data root)
	//verifies that commit saves the new tries and that can be loaded back
	//main trie malfunctions

	testHash1 := testHasher.Compute("ABCDEFGHJIJKLMNOP")
	adr1 := createAddress(t, testHash1)

	testHash2 := testHasher.Compute("ABCDEFGHJIJKLMNOPQ")
	adr2 := createAddress(t, testHash2)

	adb := createAccountsDB(t)

	//first account has the balance of 40
	state1, err := adb.GetOrCreateAccount(*adr1)
	assert.Nil(t, err)
	state1.Balance = big.NewInt(40)

	//second account has the balance of 50 and something in data root
	state2, err := adb.GetOrCreateAccount(*adr2)
	assert.Nil(t, err)

	state2.Balance = big.NewInt(50)
	state2.Root = make([]byte, encoding.HashLength)
	err = adb.RetrieveData(state2)
	assert.Nil(t, err)

	state2.Data.Update([]byte{65, 66, 67}, []byte{68, 69, 70})

	//states are now prepared, committing

	adb.MainTrie.(*mock.TrieMock).Fail = true

	err = adb.Commit()
	assert.NotNil(t, err)
}

func TestAccountsDB_Commit_MalfunctionMarshalizer_ShouldErr(t *testing.T) {
	//test creates 2 accounts (one with a data root)
	//verifies that commit saves the new tries and that can be loaded back
	//marshalizer fails

	testHash1 := testHasher.Compute("ABCDEFGHJIJKLMNOP")
	adr1 := createAddress(t, testHash1)

	testHash2 := testHasher.Compute("ABCDEFGHJIJKLMNOPQ")
	adr2 := createAddress(t, testHash2)

	adb := createAccountsDB(t)

	//first account has the balance of 40
	state1, err := adb.GetOrCreateAccount(*adr1)
	assert.Nil(t, err)
	state1.Balance = big.NewInt(40)

	//second account has the balance of 50 and something in data root
	state2, err := adb.GetOrCreateAccount(*adr2)
	assert.Nil(t, err)

	state2.Balance = big.NewInt(50)
	state2.Root = make([]byte, encoding.HashLength)
	err = adb.RetrieveData(state2)
	assert.Nil(t, err)

	state2.Data.Update([]byte{65, 66, 67}, []byte{68, 69, 70})

	//states are now prepared, committing

	adb.marsh.(*mock.MarshalizerMock).Fail = true
	defer func() {
		adb.marsh.(*mock.MarshalizerMock).Fail = false
	}()

	err = adb.Commit()
	assert.NotNil(t, err)
}

func TestAccountsDB_Commit_2okAccounts_ShouldWork(t *testing.T) {
	//test creates 2 accounts (one with a data root)
	//verifies that commit saves the new tries and that can be loaded back

	testHash1 := testHasher.Compute("ABCDEFGHJIJKLMNOP")
	adr1 := createAddress(t, testHash1)

	testHash2 := testHasher.Compute("ABCDEFGHJIJKLMNOPQ")
	adr2 := createAddress(t, testHash2)

	adb := createAccountsDB(t)

	//first account has the balance of 40
	state1, err := adb.GetOrCreateAccount(*adr1)
	assert.Nil(t, err)
	state1.Balance = big.NewInt(40)

	//second account has the balance of 50 and something in data root
	state2, err := adb.GetOrCreateAccount(*adr2)
	assert.Nil(t, err)

	state2.Balance = big.NewInt(50)
	state2.Root = make([]byte, encoding.HashLength)
	err = adb.RetrieveData(state2)
	assert.Nil(t, err)

	state2.Data.Update([]byte{65, 66, 67}, []byte{68, 69, 70})

	//states are now prepared, committing

	err = adb.Commit()
	assert.Nil(t, err)

	fmt.Printf("Data committed! Root: %d\n", adb.MainTrie.Root())

	//reloading a new trie to test if data is inside
	newTrie, err := adb.MainTrie.Recreate(adb.MainTrie.Root(), adb.MainTrie.DBW())
	assert.Nil(t, err)
	newAdb := NewAccountsDB(newTrie, testHasher, &testMarshalizer)

	//checking state1
	newState1, err := newAdb.GetOrCreateAccount(*adr1)
	assert.Nil(t, err)
	assert.Equal(t, newState1.Balance, big.NewInt(40))

	//checking state2
	newState2, err := newAdb.GetOrCreateAccount(*adr2)
	assert.Nil(t, err)
	assert.Equal(t, newState2.Balance, big.NewInt(50))
	assert.NotNil(t, newState2.Root)
	//get data
	err = adb.RetrieveData(newState2)
	assert.Nil(t, err)
	assert.NotNil(t, newState2.Data)
	val, err := newState2.Data.Get([]byte{65, 66, 67})
	assert.Nil(t, err)
	assert.Equal(t, val, []byte{68, 69, 70})
}

func TestAccountsDB_CommitUndo_2okAccounts_ShouldWork(t *testing.T) {
	//test creates 2 accounts (one with a data root)
	//verifies that commit saves the new tries and that can be loaded back
	//the states retrieved alters data
	//undo should not persist the modifications

	testHash1 := testHasher.Compute("ABCDEFGHJIJKLMNOP")
	adr1 := createAddress(t, testHash1)

	testHash2 := testHasher.Compute("ABCDEFGHJIJKLMNOPQ")
	adr2 := createAddress(t, testHash2)

	adb := createAccountsDB(t)

	//first account has the balance of 40
	state1, err := adb.GetOrCreateAccount(*adr1)
	assert.Nil(t, err)
	state1.Balance = big.NewInt(40)

	//second account has the balance of 50 and something in data root
	state2, err := adb.GetOrCreateAccount(*adr2)
	assert.Nil(t, err)

	state2.Balance = big.NewInt(50)
	state2.Root = make([]byte, encoding.HashLength)
	err = adb.RetrieveData(state2)
	assert.Nil(t, err)

	state2.Data.Update([]byte{65, 66, 67}, []byte{68, 69, 70})

	//states are now prepared, committing

	err = adb.Commit()
	assert.Nil(t, err)

	fmt.Printf("Data committed! Root: %d\n", adb.MainTrie.Root())

	//reloading a new trie to test if data is inside
	newTrie, err := adb.MainTrie.Recreate(adb.MainTrie.Root(), adb.MainTrie.DBW())
	assert.Nil(t, err)
	newAdb := NewAccountsDB(newTrie, testHasher, &testMarshalizer)

	//checking state1
	newState1, err := newAdb.GetOrCreateAccount(*adr1)
	assert.Nil(t, err)
	assert.Equal(t, newState1.Balance, big.NewInt(40))

	//checking state2
	newState2, err := newAdb.GetOrCreateAccount(*adr2)
	assert.Nil(t, err)
	assert.Equal(t, newState2.Balance, big.NewInt(50))
	assert.NotNil(t, newState2.Root)
	//get data
	err = adb.RetrieveData(state2)
	assert.Nil(t, err)
	assert.NotNil(t, state2.Data)
	val, err := state2.Data.Get([]byte{65, 66, 67})
	assert.Nil(t, err)
	assert.Equal(t, val, []byte{68, 69, 70})
	//snapshot data root for state 2
	snapshotDataRoot := state2.Root
	fmt.Printf("data ROOT: %v\n", snapshotDataRoot)

	//snapshot root hash
	snapshotRoot1 := adb.MainTrie.Root()

	//so far we have received both states
	//we change data in both of them
	//but call undo, checking that the modifications did not persist

	state1.Balance = big.NewInt(41)
	state2.Balance = big.NewInt(51)
	err = state2.Data.Update([]byte{1, 2, 3}, []byte{4, 5, 6})
	assert.Nil(t, err)

	//modifications complete
	//check dirty on state2
	assert.True(t, state2.Dirty())

	adb.SaveAccountState(state1)
	adb.SaveAccountState(state2)
	fmt.Printf("data ROOT as state2.ROOT: %v\n", state2.Root)
	fmt.Printf("data ROOT as ROOT(): %v\n", state2.Data.Root())

	//snapshot root hash
	snapshotRoot2 := adb.MainTrie.Root()

	//call undo
	err = adb.Undo()

	//reload state1 and state2

	//checking state1
	newNewState1, err := newAdb.GetOrCreateAccount(*adr1)
	assert.Nil(t, err)
	assert.Equal(t, newNewState1.Balance, big.NewInt(40))

	//checking state2
	newNewState2, err := newAdb.GetOrCreateAccount(*adr2)
	assert.Nil(t, err)
	assert.Equal(t, newNewState2.Balance, big.NewInt(50))
	assert.NotNil(t, newNewState2.Root)
	assert.Equal(t, newState2.Root, snapshotDataRoot)
	//get data
	err = adb.RetrieveData(newNewState2)
	assert.Nil(t, err)
	assert.NotNil(t, newNewState2.Data)
	val, err = newNewState2.Data.Get([]byte{65, 66, 67})
	assert.Nil(t, err)
	assert.Equal(t, val, []byte{68, 69, 70})
	//this will not be found
	val, err = newNewState2.Data.Get([]byte{1, 2, 3})
	fmt.Printf("data ROOT as ROOT 2: %v\n", newNewState2.Root)
	fmt.Printf("data ROOT as ROOT() 2: %v\n", newNewState2.Data.Root())
	assert.Nil(t, err)
	//known issue in mock trie
	//TODO(iulian)
	//assert.Nil(t, val)

	//test roots
	assert.Equal(t, snapshotRoot1, adb.MainTrie.Root())
	assert.NotEqual(t, snapshotRoot2, adb.MainTrie.Root())
}
