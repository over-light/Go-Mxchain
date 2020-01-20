package state

import (
	"errors"
	"strconv"
	"sync"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
)

// AccountsDB is the struct used for accessing accounts
type AccountsDB struct {
	mainTrie       data.Trie
	hasher         hashing.Hasher
	marshalizer    marshal.Marshalizer
	accountFactory AccountFactory

	dataTries  TriesHolder
	entries    []JournalEntry
	mutEntries sync.RWMutex
}

// NewAccountsDB creates a new account manager
func NewAccountsDB(
	trie data.Trie,
	hasher hashing.Hasher,
	marshalizer marshal.Marshalizer,
	accountFactory AccountFactory,
) (*AccountsDB, error) {
	if trie == nil || trie.IsInterfaceNil() {
		return nil, ErrNilTrie
	}
	if hasher == nil || hasher.IsInterfaceNil() {
		return nil, ErrNilHasher
	}
	if marshalizer == nil || marshalizer.IsInterfaceNil() {
		return nil, ErrNilMarshalizer
	}
	if accountFactory == nil || accountFactory.IsInterfaceNil() {
		return nil, ErrNilAccountFactory
	}

	return &AccountsDB{
		mainTrie:       trie,
		hasher:         hasher,
		marshalizer:    marshalizer,
		accountFactory: accountFactory,
		entries:        make([]JournalEntry, 0),
		mutEntries:     sync.RWMutex{},
		dataTries:      NewDataTriesHolder(),
	}, nil
}

// PutCode sets the SC plain code in AccountHandler object and trie, code hash in AccountState.
// Errors if something went wrong
func (adb *AccountsDB) PutCode(accountHandler AccountHandler, code []byte) error {
	if code == nil {
		return ErrNilCode
	}
	if check.IfNil(accountHandler) {
		return ErrNilAccountHandler
	}

	codeHash := adb.hasher.Compute(string(code))

	err := adb.addCodeToTrieIfMissing(codeHash, code)
	if err != nil {
		return err
	}

	err = accountHandler.SetCodeHashWithJournal(codeHash)
	if err != nil {
		return err
	}
	accountHandler.SetCode(code)

	return nil
}

func (adb *AccountsDB) addCodeToTrieIfMissing(codeHash []byte, code []byte) error {
	val, err := adb.mainTrie.Get(codeHash)
	if err != nil {
		return err
	}
	if val == nil {
		//append a journal entry as the code needs to be inserted in the trie
		entry, err := NewBaseJournalEntryCreation(codeHash, adb.mainTrie)
		if err != nil {
			return err
		}
		adb.Journalize(entry)
		return adb.mainTrie.Update(codeHash, code)
	}

	return nil
}

// RemoveCode deletes the code from the trie. It writes an empty byte slice at codeHash "address"
func (adb *AccountsDB) RemoveCode(codeHash []byte) error {
	return adb.mainTrie.Update(codeHash, make([]byte, 0))
}

// LoadDataTrie retrieves and saves the SC data inside accountHandler object.
// Errors if something went wrong
func (adb *AccountsDB) loadDataTrie(accountHandler AccountHandler) error {
	if accountHandler.GetRootHash() == nil {
		//do nothing, the account is either SC library or transfer account
		return nil
	}
	if len(accountHandler.GetRootHash()) != HashLength {
		return NewErrorTrieNotNormalized(HashLength, len(accountHandler.GetRootHash()))
	}

	dataTrie := adb.dataTries.Get(accountHandler.AddressContainer().Bytes())
	if dataTrie != nil {
		accountHandler.SetDataTrie(dataTrie)
		return nil
	}

	dataTrie, err := adb.mainTrie.Recreate(accountHandler.GetRootHash())
	if err != nil {
		//error as there is an inconsistent state:
		//account has data root hash but does not contain the actual trie
		return NewErrMissingTrie(accountHandler.GetRootHash())
	}

	accountHandler.SetDataTrie(dataTrie)
	adb.dataTries.Put(accountHandler.AddressContainer().Bytes(), dataTrie)
	return nil
}

// SaveDataTrie is used to save the data trie (not committing it) and to recompute the new Root value
// If data is not dirtied, method will not create its JournalEntries to keep track of data modification
func (adb *AccountsDB) SaveDataTrie(accountHandler AccountHandler) error {
	flagHasDirtyData := false

	if check.IfNil(accountHandler.DataTrie()) {
		newDataTrie, err := adb.mainTrie.Recreate(make([]byte, 0))
		if err != nil {
			return err
		}

		accountHandler.SetDataTrie(newDataTrie)
		adb.dataTries.Put(accountHandler.AddressContainer().Bytes(), newDataTrie)
	}

	trackableDataTrie := accountHandler.DataTrieTracker()
	if trackableDataTrie == nil {
		return ErrNilTrackableDataTrie
	}

	dataTrie := trackableDataTrie.DataTrie()
	oldValues := make(map[string][]byte)

	for k, v := range trackableDataTrie.DirtyData() {
		//TODO: delete the next verification when delete from trie bug is repaired
		if len(v) == 0 {
			continue
		}

		flagHasDirtyData = true

		val, err := dataTrie.Get([]byte(k))
		if err != nil {
			return err
		}

		oldValues[k] = val

		err = dataTrie.Update([]byte(k), v)
		if err != nil {
			return err
		}
	}

	if !flagHasDirtyData {
		//do not need to save, return
		return nil
	}

	entry, err := NewJournalEntryDataTrieUpdates(oldValues, accountHandler)
	if err != nil {
		return err
	}
	adb.Journalize(entry)

	rootHash, err := trackableDataTrie.DataTrie().Root()
	if err != nil {
		return err
	}

	accountHandler.SetRootHash(rootHash)
	trackableDataTrie.ClearDataCaches()

	return adb.SaveAccount(accountHandler)
}

// HasAccount searches for an account based on the address. Errors if something went wrong and
// outputs if the account exists or not
func (adb *AccountsDB) HasAccount(addressContainer AddressContainer) (bool, error) {
	val, err := adb.mainTrie.Get(addressContainer.Bytes())
	if err != nil {
		return false, err
	}

	return val != nil, nil
}

func (adb *AccountsDB) getAccount(addressContainer AddressContainer) (AccountHandler, error) {
	addrBytes := addressContainer.Bytes()

	val, err := adb.mainTrie.Get(addrBytes)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil
	}

	acnt, err := adb.accountFactory.CreateAccount(addressContainer, adb)
	if err != nil {
		return nil, err
	}

	err = adb.marshalizer.Unmarshal(acnt, val)
	if err != nil {
		return nil, err
	}

	return acnt, nil
}

// SaveAccount saves the account WITHOUT data trie inside main trie. Errors if something went wrong
func (adb *AccountsDB) SaveAccount(accountHandler AccountHandler) error {
	if accountHandler == nil || accountHandler.IsInterfaceNil() {
		return errors.New("can not save nil account state")
	}

	//pass the reference to marshalizer, otherwise it will fail marshalizing balance
	buff, err := adb.marshalizer.Marshal(accountHandler)
	if err != nil {
		return err
	}

	return adb.mainTrie.Update(accountHandler.AddressContainer().Bytes(), buff)
}

// RemoveAccount removes the account data from underlying trie.
// It basically calls Update with empty slice
func (adb *AccountsDB) RemoveAccount(addressContainer AddressContainer) error {
	return adb.mainTrie.Update(addressContainer.Bytes(), make([]byte, 0))
}

// GetAccountWithJournal fetches the account based on the address. Creates an empty account if the account is missing.
func (adb *AccountsDB) GetAccountWithJournal(addressContainer AddressContainer) (AccountHandler, error) {
	acnt, err := adb.getAccount(addressContainer)
	if err != nil {
		return nil, err
	}
	if acnt != nil {
		return adb.loadAccountHandler(acnt)
	}

	return adb.newAccountHandler(addressContainer)
}

// GetExistingAccount returns an existing account if exists or nil if missing
func (adb *AccountsDB) GetExistingAccount(addressContainer AddressContainer) (AccountHandler, error) {
	acnt, err := adb.getAccount(addressContainer)
	if err != nil {
		return nil, err
	}
	if acnt == nil {
		return nil, ErrAccNotFound
	}

	err = adb.loadCodeAndDataIntoAccountHandler(acnt)
	if err != nil {
		return nil, err
	}

	return acnt, nil
}

func (adb *AccountsDB) loadAccountHandler(accountHandler AccountHandler) (AccountHandler, error) {
	err := adb.loadCodeAndDataIntoAccountHandler(accountHandler)
	if err != nil {
		return nil, err
	}

	return accountHandler, nil
}

func (adb *AccountsDB) loadCodeAndDataIntoAccountHandler(accountHandler AccountHandler) error {
	err := adb.loadCode(accountHandler)
	if err != nil {
		return err
	}

	err = adb.loadDataTrie(accountHandler)
	if err != nil {
		return err
	}

	return nil
}

func (adb *AccountsDB) newAccountHandler(address AddressContainer) (AccountHandler, error) {
	acnt, err := adb.accountFactory.CreateAccount(address, adb)
	if err != nil {
		return nil, err
	}

	entry, err := NewBaseJournalEntryCreation(address.Bytes(), adb.mainTrie)
	if err != nil {
		return nil, err
	}

	adb.Journalize(entry)
	err = adb.SaveAccount(acnt)
	if err != nil {
		return nil, err
	}

	return acnt, nil
}

// RevertToSnapshot apply Revert method over accounts object and removes entries from the list
// If snapshot > len(entries) will do nothing, return will be nil
// 0 index based. Calling this method with negative value will do nothing. Calling with 0 revert everything.
// Concurrent safe.
func (adb *AccountsDB) RevertToSnapshot(snapshot int) error {
	if snapshot > len(adb.entries) || snapshot < 0 {
		//outside of bounds array, not quite error, just return
		return nil
	}

	adb.mutEntries.Lock()
	defer adb.mutEntries.Unlock()

	for i := len(adb.entries) - 1; i >= snapshot; i-- {
		account, err := adb.entries[i].Revert()
		if err != nil {
			return err
		}

		if account != nil {
			err = adb.SaveAccount(account)
		}
	}

	adb.entries = adb.entries[:snapshot]

	return nil
}

// JournalLen will return the number of entries
// Concurrent safe.
func (adb *AccountsDB) JournalLen() int {
	adb.mutEntries.RLock()
	length := len(adb.entries)
	adb.mutEntries.RUnlock()

	return length
}

// Commit will persist all data inside the trie
func (adb *AccountsDB) Commit() ([]byte, error) {
	adb.mutEntries.RLock()
	jEntries := make([]JournalEntry, len(adb.entries))
	copy(jEntries, adb.entries)
	adb.mutEntries.RUnlock()

	oldHashes := make([][]byte, 0)
	//Step 1. commit all data tries
	dataTries := adb.dataTries.GetAll()
	for i := 0; i < len(dataTries); i++ {
		oldTrieHashes := dataTries[i].ResetOldHashes()
		err := dataTries[i].Commit()
		if err != nil {
			return nil, err
		}

		oldHashes = append(oldHashes, oldTrieHashes...)
	}
	adb.dataTries.Reset()

	//step 2. clean the journal
	adb.clearJournal()

	//Step 3. commit main trie
	adb.mainTrie.AppendToOldHashes(oldHashes)
	err := adb.mainTrie.Commit()
	if err != nil {
		return nil, err
	}

	root, err := adb.mainTrie.Root()
	if err != nil {
		return nil, err
	}

	return root, nil
}

// loadCode retrieves and saves the SC code inside AccountState object. Errors if something went wrong
func (adb *AccountsDB) loadCode(accountHandler AccountHandler) error {
	if accountHandler.GetCodeHash() == nil || len(accountHandler.GetCodeHash()) == 0 {
		return nil
	}
	if len(accountHandler.GetCodeHash()) != HashLength {
		return errors.New("attempt to search a hash not normalized to" +
			strconv.Itoa(HashLength) + "bytes")
	}

	val, err := adb.mainTrie.Get(accountHandler.GetCodeHash())
	if err != nil {
		return err
	}

	accountHandler.SetCode(val)
	return nil
}

// RootHash returns the main trie's root hash
func (adb *AccountsDB) RootHash() ([]byte, error) {
	return adb.mainTrie.Root()
}

// RecreateTrie is used to reload the trie based on an existing rootHash
func (adb *AccountsDB) RecreateTrie(rootHash []byte) error {
	newTrie, err := adb.mainTrie.Recreate(rootHash)
	if err != nil {
		return err
	}
	if newTrie == nil {
		return ErrNilTrie
	}

	adb.mainTrie = newTrie
	return nil
}

// Journalize adds a new object to entries list. Concurrent safe.
func (adb *AccountsDB) Journalize(entry JournalEntry) {
	if entry == nil || entry.IsInterfaceNil() {
		return
	}

	adb.mutEntries.Lock()
	adb.entries = append(adb.entries, entry)
	adb.mutEntries.Unlock()
}

// Clear clears the data from this journal.
func (adb *AccountsDB) clearJournal() {
	adb.mutEntries.Lock()
	adb.entries = make([]JournalEntry, 0)
	adb.mutEntries.Unlock()
}

// PruneTrie removes old values from the trie database
func (adb *AccountsDB) PruneTrie(rootHash []byte) error {
	return adb.mainTrie.Prune(rootHash, data.OldRoot)
}

// CancelPrune clears the trie's evictionWaitingList
func (adb *AccountsDB) CancelPrune(rootHash []byte) {
	adb.mainTrie.CancelPrune(rootHash, data.NewRoot)
}

// SnapshotState triggers the snapshotting process of the state trie
func (adb *AccountsDB) SnapshotState(rootHash []byte) {
	adb.mainTrie.TakeSnapshot(rootHash)
}

// SetStateCheckpoint sets a checkpoint for the state trie
func (adb *AccountsDB) SetStateCheckpoint(rootHash []byte) {
	adb.mainTrie.SetCheckpoint(rootHash)
}

// IsPruningEnabled returns true if state pruning is enabled
func (adb *AccountsDB) IsPruningEnabled() bool {
	return adb.mainTrie.IsPruningEnabled()
}

// IsInterfaceNil returns true if there is no value under the interface
func (adb *AccountsDB) IsInterfaceNil() bool {
	return adb == nil
}
