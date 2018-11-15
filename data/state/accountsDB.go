package state

import (
	"errors"
	"strconv"
	"sync"

	"github.com/ElrondNetwork/elrond-go-sandbox/data/trie"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/trie/encoding"
	"github.com/ElrondNetwork/elrond-go-sandbox/hashing"
	"github.com/ElrondNetwork/elrond-go-sandbox/marshal"
)

// AccountsDB is the struct used for accessing accounts
type AccountsDB struct {
	//should use a concurrent trie
	MainTrie trie.PatriciaMerkelTree
	hasher   hashing.Hasher
	marsh    marshal.Marshalizer

	mutUsedAccounts   sync.RWMutex
	usedStateAccounts map[string]*AccountState
	jurnal            *Jurnal
}

// NewAccountsDB creates a new account manager
func NewAccountsDB(tr trie.PatriciaMerkelTree, hasher hashing.Hasher, marsh marshal.Marshalizer) *AccountsDB {
	adb := AccountsDB{MainTrie: tr, hasher: hasher, marsh: marsh}

	adb.mutUsedAccounts = sync.RWMutex{}
	adb.usedStateAccounts = make(map[string]*AccountState, 0)
	adb.jurnal = NewJurnal(&adb)

	return &adb
}

// RetrieveCode retrieves and saves the SC code inside AccountState object. Errors if something went wrong
func (adb *AccountsDB) RetrieveCode(state *AccountState) error {
	if state.CodeHash() == nil {
		state.Code = nil
		return nil
	}

	if adb.MainTrie == nil {
		return errors.New("attempt to search on a nil trie")
	}

	if len(state.CodeHash()) != encoding.HashLength {
		return errors.New("attempt to search a hash not normalized to" +
			strconv.Itoa(encoding.HashLength) + "bytes")
	}

	val, err := adb.MainTrie.Get(state.CodeHash())

	if err != nil {
		return err
	}

	state.Code = val
	return nil
}

// RetrieveDataTrie retrieves and saves the SC data inside AccountState object. Errors if something went wrong
func (adb *AccountsDB) RetrieveDataTrie(state *AccountState) error {
	if state.Root() == nil {
		//do nothing, the account is either SC library or transfer account
		return nil
	}

	if adb.MainTrie == nil {
		return ErrNilTrie
	}

	if len(state.Root()) != encoding.HashLength {
		return NewErrorTrieNotNormalized(encoding.HashLength, len(state.Root()))
	}

	dataTrie, err := adb.MainTrie.Recreate(state.Root(), adb.MainTrie.DBW())
	if err != nil {
		//error as there is an inconsistent state:
		//account has data root but does not contain the actual trie
		return NewErrMissingTrie(state.Root())
	}

	state.DataTrie = dataTrie
	return nil
}

// PutCode sets the SC plain code in AccountState object and trie, code hash in AccountState. Errors if something went wrong
func (adb *AccountsDB) PutCode(state *AccountState, code []byte) error {
	if (code == nil) || (len(code) == 0) {
		return nil
	}

	if adb.MainTrie == nil {
		return errors.New("attempt to search on a nil trie")
	}

	codeHash := adb.hasher.Compute(string(code))

	if state != nil {
		state.SetCodeHash(adb, codeHash)
		state.Code = code
	}

	//test that we add the code, or reuse an existing code as to be able to correctly revert the addition
	val, err := adb.MainTrie.Get(codeHash)
	if err != nil {
		return err
	}

	if val == nil {
		//append a jurnal entry as the code needs to be inserted in the trie
		adb.Jurnal().AddEntry(NewJurnalEntryCode(codeHash))
	}

	return adb.MainTrie.Update(codeHash, code)
}

// RemoveCode deletes the code from the trie. It writes an empty byte slice at codeHash "address"
func (adb *AccountsDB) RemoveCode(codeHash []byte) error {
	return adb.MainTrie.Update(codeHash, make([]byte, 0))
}

// HasAccount searches for an account based on the address. Errors if something went wrong and outputs if the account exists or not
func (adb *AccountsDB) HasAccount(address Address) (bool, error) {
	if adb.MainTrie == nil {
		return false, errors.New("attempt to search on a nil trie")
	}

	val, err := adb.MainTrie.Get(address.Hash(adb.hasher))

	if err != nil {
		return false, err
	}

	return val != nil, nil
}

// SaveAccountState saves the account WITHOUT data trie inside main trie. Errors if something went wrong
func (adb *AccountsDB) SaveAccountState(state *AccountState) error {
	if state == nil {
		return errors.New("can not save nil account state")
	}

	if adb.MainTrie == nil {
		return errors.New("attempt to search on a nil trie")
	}

	buff, err := adb.marsh.Marshal(state.Account())

	if err != nil {
		return err
	}

	err = adb.MainTrie.Update(state.Addr.Hash(adb.hasher), buff)
	if err != nil {
		return err
	}

	return nil
}

// RemoveAccount removes the account data from underlying trie.
// It basically calls Update with empty slice
func (adb *AccountsDB) RemoveAccount(address Address) error {
	if adb.MainTrie == nil {
		return errors.New("attempt to search on a nil trie")
	}

	return adb.MainTrie.Update(address.Hash(adb.hasher), make([]byte, 0))
}

// GetOrCreateAccount fetches the account based on the address. Creates an empty account if the account is missing.
func (adb *AccountsDB) GetOrCreateAccount(address Address) (*AccountState, error) {
	has, err := adb.HasAccount(address)
	if err != nil {
		return nil, err
	}

	if has {
		val, err := adb.MainTrie.Get(address.Hash(adb.hasher))
		if err != nil {
			return nil, err
		}

		acnt := NewAccount()

		err = adb.marsh.Unmarshal(acnt, val)
		if err != nil {
			return nil, err
		}

		state := NewAccountState(address, *acnt, adb.hasher)

		return state, nil
	}

	state := NewAccountState(address, *NewAccount(), adb.hasher)
	adb.jurnal.AddEntry(NewJurnalEntryCreation(&address, state))

	err = adb.SaveAccountState(state)
	if err != nil {
		return nil, err
	}

	return state, nil
}

// Commit will persist all data inside the trie
func (adb *AccountsDB) Commit() ([]byte, error) {
	adb.mutUsedAccounts.Lock()
	defer adb.mutUsedAccounts.Unlock()

	//Step 1. iterate through tracked account states map and commits the new tries
	//TODO
	//for _, v := range adb.usedStateAccounts {
	//	if v.Dirty() {
	//		hash, err := v.Data.Commit(nil)
	//		if err != nil {
	//			return nil, err
	//		}
	//
	//		v.Root = hash
	//		v.prevRoot = hash
	//	}
	//
	//	buff, err := adb.marsh.Marshal(v.Account)
	//	if err != nil {
	//		return nil, err
	//	}
	//	adb.MainTrie.Update(v.Addr.Hash(adb.hasher), buff)
	//}

	//step 2. clean used accounts map
	adb.usedStateAccounts = make(map[string]*AccountState, 0)

	//Step 3. commit main trie
	hash, err := adb.MainTrie.Commit(nil)
	if err != nil {
		return nil, err
	}

	return hash, nil
}

// Jurnal returns the Jurnal pointer which is used at taking snapshots and reverting tries past states
func (adb *AccountsDB) Jurnal() *Jurnal {
	return adb.jurnal
}
