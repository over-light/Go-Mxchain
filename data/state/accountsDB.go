package state

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/logger"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/storage"
)

// Type defines account types to save in accounts trie
type Type uint8

const (
	// UserAccount identifies an account holding balance, storage updates, code
	UserAccount Type = 0
	// ValidatorAccount identifies an account holding stake, crypto public keys, assigned shard, rating
	ValidatorAccount Type = 1
	// DataTrie identifies the data trie kept under a specific account
	DataTrie Type = 2
)

// SupportedAccountTypes is the list to describe the possible account types in the accounts DB
var SupportedAccountTypes = []Type{UserAccount, ValidatorAccount, DataTrie}

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

var log = logger.GetOrCreate("state")

// NewAccountsDB creates a new account manager
//TODO refactor this: remove the mutex from patricia merkle trie impl. Make all these methods concurrent safe
func NewAccountsDB(
	trie data.Trie,
	hasher hashing.Hasher,
	marshalizer marshal.Marshalizer,
	accountFactory AccountFactory,
) (*AccountsDB, error) {
	if check.IfNil(trie) {
		return nil, ErrNilTrie
	}
	if check.IfNil(hasher) {
		return nil, ErrNilHasher
	}
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(accountFactory) {
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
	log.Trace("accountsDB.PutCode", "code hash", codeHash)

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
		var entry *BaseJournalEntryCreation
		entry, err = NewBaseJournalEntryCreation(codeHash, adb.mainTrie)
		if err != nil {
			return err
		}
		adb.Journalize(entry)
		return adb.mainTrie.Update(codeHash, code)
	}

	return nil
}

// ClosePersister will close trie persister
func (adb *AccountsDB) ClosePersister() error {
	log.Trace("accountsDB.ClosePersister")

	closedSuccessfully := true

	err := adb.mainTrie.ClosePersister()
	log.LogIfError(err)
	if err != nil {
		closedSuccessfully = false
	}

	trees := adb.dataTries.GetAll()
	for _, trie := range trees {
		err = trie.ClosePersister()
		if err != nil {
			log.Error("cannot close accounts trie persister", err)
			closedSuccessfully = false
		}
	}

	if closedSuccessfully {
		return nil
	}

	return storage.ErrClosingPersisters
}

// RemoveCode deletes the code from the trie. It writes an empty byte slice at codeHash "address"
func (adb *AccountsDB) RemoveCode(codeHash []byte) error {
	log.Trace("accountsDB.RemoveCode", "code hash", codeHash)

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
	if check.IfNil(accountHandler) {
		return fmt.Errorf("%w in SaveDataTrie", ErrNilAccountHandler)
	}

	log.Trace("accountsDB.SaveDataTrie",
		"address", hex.EncodeToString(accountHandler.AddressContainer().Bytes()),
		"nonce", accountHandler.GetNonce(),
	)

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
	if check.IfNil(addressContainer) {
		return false, fmt.Errorf("%w in HasAccount", ErrNilAddressContainer)
	}

	log.Trace("accountsDB.HasAccount",
		"address", hex.EncodeToString(addressContainer.Bytes()),
	)

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
	if check.IfNil(accountHandler) {
		return fmt.Errorf("%w in SaveAccount", ErrNilAccountHandler)
	}

	log.Trace("accountsDB.SaveAccount",
		"address", hex.EncodeToString(accountHandler.AddressContainer().Bytes()),
		"nonce", accountHandler.GetNonce(),
	)

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
	if check.IfNil(addressContainer) {
		return fmt.Errorf("%w in RemoveAccount", ErrNilAddressContainer)
	}

	log.Trace("accountsDB.RemoveAccount",
		"address", hex.EncodeToString(addressContainer.Bytes()),
	)

	return adb.mainTrie.Update(addressContainer.Bytes(), make([]byte, 0))
}

// GetAccountWithJournal fetches the account based on the address. Creates an empty account if the account is missing.
func (adb *AccountsDB) GetAccountWithJournal(addressContainer AddressContainer) (AccountHandler, error) {
	if check.IfNil(addressContainer) {
		return nil, fmt.Errorf("%w in GetAccountWithJournal", ErrNilAddressContainer)
	}

	log.Trace("accountsDB.GetAccountWithJournal",
		"address", hex.EncodeToString(addressContainer.Bytes()),
	)

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
	if check.IfNil(addressContainer) {
		return nil, fmt.Errorf("%w in GetExistingAccount", ErrNilAddressContainer)
	}

	log.Trace("accountsDB.GetExistingAccount",
		"address", hex.EncodeToString(addressContainer.Bytes()),
	)

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
	log.Trace("accountsDB.RevertToSnapshot started",
		"snapshot", snapshot,
	)

	adb.mutEntries.Lock()
	defer func() {
		log.Trace("accountsDB.RevertToSnapshot ended")
		adb.mutEntries.Unlock()
	}()

	if snapshot > len(adb.entries) || snapshot < 0 {
		//outside of bounds array, not quite error, just return
		return nil
	}

	for i := len(adb.entries) - 1; i >= snapshot; i-- {
		account, err := adb.entries[i].Revert()
		if err != nil {
			return err
		}

		if account != nil {
			err = adb.SaveAccount(account)
			if err != nil {
				return err
			}
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

	log.Trace("accountsDB.JournalLen",
		"length", length,
	)

	return length
}

// Commit will persist all data inside the trie
func (adb *AccountsDB) Commit() ([]byte, error) {
	log.Trace("accountsDB.Commit started")

	adb.mutEntries.Lock()
	jEntries := make([]JournalEntry, len(adb.entries))
	copy(jEntries, adb.entries)
	adb.entries = make([]JournalEntry, 0)
	adb.mutEntries.Unlock()

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

	//TODO apply clean code here
	//Step 2. commit main trie
	adb.mainTrie.AppendToOldHashes(oldHashes)
	err := adb.mainTrie.Commit()
	if err != nil {
		return nil, err
	}

	root, err := adb.mainTrie.Root()
	if err != nil {
		log.Trace("accountsDB.Commit ended", "error", err.Error())
		return nil, err
	}

	log.Trace("accountsDB.Commit ended", "root hash", root)

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
	rootHash, err := adb.mainTrie.Root()

	log.Trace("accountsDB.RootHash",
		"root hash", rootHash,
		"err", err,
	)

	return rootHash, err
}

// RecreateTrie is used to reload the trie based on an existing rootHash
func (adb *AccountsDB) RecreateTrie(rootHash []byte) error {
	log.Trace("accountsDB.RecreateTrie", "root hash", rootHash)
	defer func() {
		log.Trace("accountsDB.RecreateTrie ended")
	}()

	newTrie, err := adb.mainTrie.Recreate(rootHash)
	if err != nil {
		return err
	}
	if check.IfNil(newTrie) {
		return ErrNilTrie
	}

	adb.mainTrie = newTrie
	return nil
}

// RecreateAllTries recreates all the tries from the accounts DB
func (adb *AccountsDB) RecreateAllTries(rootHash []byte) (map[string]data.Trie, error) {
	recreatedTrie, err := adb.mainTrie.Recreate(rootHash)
	if err != nil {
		return nil, err
	}

	leafs, err := recreatedTrie.GetAllLeaves()
	if err != nil {
		return nil, err
	}

	allTries := make(map[string]data.Trie)
	allTries[string(rootHash)] = recreatedTrie

	for _, leaf := range leafs {
		account := &Account{}
		err = adb.marshalizer.Unmarshal(account, leaf)
		if err != nil {
			log.Trace("this must be a leaf with code", "err", err)
			continue
		}

		if len(account.RootHash) > 0 {
			dataTrie, err := adb.mainTrie.Recreate(account.RootHash)
			if err != nil {
				return nil, err
			}

			allTries[string(account.RootHash)] = dataTrie
		}
	}

	return allTries, nil
}

// Journalize adds a new object to entries list. Concurrent safe.
func (adb *AccountsDB) Journalize(entry JournalEntry) {
	if check.IfNil(entry) {
		return
	}

	adb.mutEntries.Lock()
	adb.entries = append(adb.entries, entry)

	log.Trace("accountsDB.Journalize", "new length", len(adb.entries))
	adb.mutEntries.Unlock()
}

// PruneTrie removes old values from the trie database
func (adb *AccountsDB) PruneTrie(rootHash []byte, identifier data.TriePruningIdentifier) error {
	log.Trace("accountsDB.PruneTrie", "root hash", rootHash)

	return adb.mainTrie.Prune(rootHash, identifier)
}

// CancelPrune clears the trie's evictionWaitingList
func (adb *AccountsDB) CancelPrune(rootHash []byte, identifier data.TriePruningIdentifier) {
	log.Trace("accountsDB.CancelPrune", "root hash", rootHash)

	adb.mainTrie.CancelPrune(rootHash, identifier)
}

// SnapshotState triggers the snapshotting process of the state trie
func (adb *AccountsDB) SnapshotState(rootHash []byte) {
	log.Trace("accountsDB.SnapshotState", "root hash", rootHash)
	adb.mainTrie.EnterSnapshotMode()
	adb.mainTrie.TakeSnapshot(rootHash)

	go func() {
		adb.snapshotUserAccountDataTrie(rootHash)
		adb.mainTrie.ExitSnapshotMode()
	}()
}

func (adb *AccountsDB) snapshotUserAccountDataTrie(rootHash []byte) {
	leafs, err := adb.GetAllLeaves(rootHash)
	if err != nil {
		log.Error("incomplete snapshot as getAllLeaves error", "error", err)
		return
	}

	for _, leaf := range leafs {
		account := &Account{}
		err = adb.marshalizer.Unmarshal(account, leaf)
		if err != nil {
			log.Trace("this must be a leaf with code", "err", err)
			continue
		}

		if len(account.RootHash) > 0 {
			adb.mainTrie.TakeSnapshot(account.RootHash)
		}
	}
}

// SetStateCheckpoint sets a checkpoint for the state trie
func (adb *AccountsDB) SetStateCheckpoint(rootHash []byte) {
	log.Trace("accountsDB.SetStateCheckpoint", "root hash", rootHash)
	adb.mainTrie.EnterSnapshotMode()
	adb.mainTrie.SetCheckpoint(rootHash)
	adb.mainTrie.ExitSnapshotMode()
}

// IsPruningEnabled returns true if state pruning is enabled
func (adb *AccountsDB) IsPruningEnabled() bool {
	return adb.mainTrie.IsPruningEnabled()
}

// GetAllLeaves returns all the leaves from a given rootHash
func (adb *AccountsDB) GetAllLeaves(rootHash []byte) (map[string][]byte, error) {
	newTrie, err := adb.mainTrie.Recreate(rootHash)
	if err != nil {
		return nil, err
	}
	if newTrie == nil {
		return nil, ErrNilTrie
	}

	allAccounts, err := newTrie.GetAllLeaves()
	if err != nil {
		return nil, err
	}

	return allAccounts, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (adb *AccountsDB) IsInterfaceNil() bool {
	return adb == nil
}
