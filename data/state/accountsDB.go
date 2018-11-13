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
}

// NewAccountsDB creates a new account manager
func NewAccountsDB(tr trie.PatriciaMerkelTree, hasher hashing.Hasher, marsh marshal.Marshalizer) *AccountsDB {
	adb := AccountsDB{MainTrie: tr, hasher: hasher, marsh: marsh}

	adb.mutUsedAccounts = sync.RWMutex{}
	adb.usedStateAccounts = make(map[string]*AccountState, 0)

	return &adb
}

// RetrieveCode retrieves and saves the SC code inside AccountState object. Errors if something went wrong
func (adb *AccountsDB) RetrieveCode(state *AccountState) error {
	if state.CodeHash == nil {
		state.Code = nil
		return nil
	}

	if adb.MainTrie == nil {
		return errors.New("attempt to search on a nil trie")
	}

	if len(state.CodeHash) != encoding.HashLength {
		return errors.New("attempt to search a hash not normalized to" +
			strconv.Itoa(encoding.HashLength) + "bytes")
	}

	val, err := adb.MainTrie.Get(state.CodeHash)

	if err != nil {
		return err
	}

	state.Code = val
	return nil
}

// RetrieveData retrieves and saves the SC data inside AccountState object. Errors if something went wrong
func (adb *AccountsDB) RetrieveData(state *AccountState) error {
	if state.Root == nil {
		state.Data = nil
		return nil
	}

	if adb.MainTrie == nil {
		return errors.New("attempt to search on a nil trie")
	}

	if len(state.Root) != encoding.HashLength {
		return errors.New("attempt to search a hash not normalized to" +
			strconv.Itoa(encoding.HashLength) + "bytes")
	}

	dataTrie, err := adb.MainTrie.Recreate(state.Root, adb.MainTrie.DBW())
	if err != nil {
		//node does not exists, create new one
		dataTrie, err = adb.MainTrie.Recreate(make([]byte, 0), adb.MainTrie.DBW())
		if err != nil {
			return err
		}

		state.Root = dataTrie.Root()
	}

	state.Data = dataTrie
	return nil
}

// PutCode sets the SC plain code in AccountState object and trie, code hash in AccountState. Errors if something went wrong
func (adb *AccountsDB) PutCode(state *AccountState, code []byte) error {
	if (code == nil) || (len(code) == 0) {
		state.Reset()
		return nil
	}

	if adb.MainTrie == nil {
		return errors.New("attempt to search on a nil trie")
	}

	state.CodeHash = adb.hasher.Compute(string(code))
	state.Code = code

	err := adb.MainTrie.Update(state.CodeHash, state.Code)
	if err != nil {
		state.Reset()
		return err
	}

	dataTrie, err := adb.MainTrie.Recreate(make([]byte, 0), adb.MainTrie.DBW())
	if err != nil {
		state.Reset()
		return err
	}

	state.Data = dataTrie
	return nil
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

	buff, err := adb.marsh.Marshal(state.Account)

	if err != nil {
		return err
	}

	adb.trackAccountState(state)

	err = adb.MainTrie.Update(state.Addr.Hash(adb.hasher), buff)
	if err != nil {
		return err
	}

	return nil
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

		acnt := Account{}

		err = adb.marsh.Unmarshal(&acnt, val)
		if err != nil {
			return nil, err
		}

		state := NewAccountState(address, acnt)

		adb.trackAccountState(state)

		return state, nil
	}

	state := NewAccountState(address, Account{})

	err = adb.SaveAccountState(state)
	if err != nil {
		return nil, err
	}

	return state, nil
}

// Commit will persist all data inside
func (adb *AccountsDB) Commit() ([]byte, error) {
	adb.mutUsedAccounts.Lock()
	defer adb.mutUsedAccounts.Unlock()

	//Step 1. iterate through tracked account states map and commits the new tries
	for _, v := range adb.usedStateAccounts {
		if v.Dirty() {
			hash, err := v.Data.Commit(nil)
			if err != nil {
				return nil, err
			}

			v.Root = hash
			v.PrevRoot = hash
		}

		buff, err := adb.marsh.Marshal(v.Account)
		if err != nil {
			return nil, err
		}
		adb.MainTrie.Update(v.Addr.Hash(adb.hasher), buff)
	}

	//step 2. clean used accounts map
	adb.usedStateAccounts = make(map[string]*AccountState, 0)

	//Step 3. commit main trie
	hash, err := adb.MainTrie.Commit(nil)
	if err != nil {
		return nil, err
	}

	return hash, nil
}

func (adb *AccountsDB) trackAccountState(state *AccountState) {
	//add account to used accounts to track the modifications in data trie for committing/undoing
	adb.mutUsedAccounts.Lock()
	_, ok := adb.usedStateAccounts[string(state.Addr.Hash(adb.hasher))]
	if !ok {
		adb.usedStateAccounts[string(state.Addr.Hash(adb.hasher))] = state
	}
	adb.mutUsedAccounts.Unlock()
}
