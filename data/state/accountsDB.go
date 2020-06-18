package state

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sync"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
)

// AccountsDB is the struct used for accessing accounts. This struct is concurrent safe.
type AccountsDB struct {
	mainTrie       data.Trie
	hasher         hashing.Hasher
	marshalizer    marshal.Marshalizer
	accountFactory AccountFactory

	lastRootHash []byte
	dataTries    TriesHolder
	entries      []JournalEntry
	mutOp        sync.RWMutex
}

var log = logger.GetOrCreate("state")

// NewAccountsDB creates a new account manager
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
		mutOp:          sync.RWMutex{},
		dataTries:      NewDataTriesHolder(),
	}, nil
}

// SaveAccount saves in the trie all changes made to the account.
func (adb *AccountsDB) SaveAccount(account AccountHandler) error {
	adb.mutOp.Lock()
	defer adb.mutOp.Unlock()

	if check.IfNil(account) {
		return fmt.Errorf("%w in accountsDB SaveAccount", ErrNilAccountHandler)
	}

	oldAccount, err := adb.getAccount(account.AddressBytes())
	if err != nil {
		return err
	}

	var entry JournalEntry
	if check.IfNil(oldAccount) {
		entry, err = NewJournalEntryAccountCreation(account.AddressBytes(), adb.mainTrie)
		if err != nil {
			return err
		}
		adb.journalize(entry)
	} else {
		entry, err = NewJournalEntryAccount(oldAccount)
		if err != nil {
			return err
		}
		adb.journalize(entry)
	}

	baseAcc, ok := account.(baseAccountHandler)
	if ok {
		err = adb.saveCode(baseAcc, oldAccount)
		if err != nil {
			return err
		}

		err = adb.saveDataTrie(baseAcc)
		if err != nil {
			return err
		}
	}

	return adb.saveAccountToTrie(account)
}

func (adb *AccountsDB) getOldCodeHash(oldAcc AccountHandler) ([]byte, error) {
	if check.IfNil(oldAcc) {
		return nil, nil
	}

	oldAccount, ok := oldAcc.(baseAccountHandler)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return oldAccount.GetCodeHash(), nil
}

func (adb *AccountsDB) saveCode(accountHandler baseAccountHandler, oldAcc AccountHandler) error {
	oldCodeHash, err := adb.getOldCodeHash(oldAcc)
	if err != nil {
		return err
	}

	newCode := accountHandler.GetCode()
	var newCodeHash []byte
	if len(newCode) != 0 {
		newCodeHash = adb.hasher.Compute(string(newCode))
	}

	if bytes.Equal(oldCodeHash, newCodeHash) {
		accountHandler.SetCodeHash(newCodeHash)
		return nil
	}

	unmodifiedOldCodeEntry, err := adb.updateOldCodeEntry(oldCodeHash)
	if err != nil {
		return err
	}

	err = adb.updateNewCodeEntry(newCodeHash, newCode)
	if err != nil {
		return err
	}

	entry, err := NewJournalEntryCode(unmodifiedOldCodeEntry, oldCodeHash, newCodeHash, adb.mainTrie, adb.marshalizer)
	if err != nil {
		return err
	}
	adb.journalize(entry)

	accountHandler.SetCodeHash(newCodeHash)
	return nil
}

func (adb *AccountsDB) updateOldCodeEntry(oldCodeHash []byte) (*CodeEntry, error) {
	oldCodeEntry, err := getCodeEntry(oldCodeHash, adb.mainTrie, adb.marshalizer)
	if err != nil {
		return nil, err
	}

	if oldCodeEntry == nil {
		return nil, nil
	}

	unmodifiedOldCodeEntry := &CodeEntry{
		Code:          oldCodeEntry.Code,
		NumReferences: oldCodeEntry.NumReferences,
	}

	if oldCodeEntry.NumReferences <= 1 {
		err = adb.mainTrie.Update(oldCodeHash, nil)
		if err != nil {
			return nil, err
		}

		return unmodifiedOldCodeEntry, nil
	}

	oldCodeEntry.NumReferences--
	err = saveCodeEntry(oldCodeHash, oldCodeEntry, adb.mainTrie, adb.marshalizer)
	if err != nil {
		return nil, err
	}

	return unmodifiedOldCodeEntry, nil
}

func (adb *AccountsDB) updateNewCodeEntry(newCodeHash []byte, newCode []byte) error {
	if len(newCode) == 0 {
		return nil
	}

	newCodeEntry, err := getCodeEntry(newCodeHash, adb.mainTrie, adb.marshalizer)
	if err != nil {
		return err
	}

	if newCodeEntry == nil {
		newCodeEntry = &CodeEntry{
			Code: newCode,
		}
	}
	newCodeEntry.NumReferences++

	err = saveCodeEntry(newCodeHash, newCodeEntry, adb.mainTrie, adb.marshalizer)
	if err != nil {
		return err
	}

	return nil
}

func getCodeEntry(codeHash []byte, trie Updater, marshalizer marshal.Marshalizer) (*CodeEntry, error) {
	val, err := trie.Get(codeHash)
	if err != nil {
		return nil, err
	}

	if len(val) == 0 {
		return nil, nil
	}

	var codeEntry CodeEntry
	err = marshalizer.Unmarshal(&codeEntry, val)
	if err != nil {
		return nil, err
	}

	return &codeEntry, nil
}

func saveCodeEntry(codeHash []byte, entry *CodeEntry, trie Updater, marshalizer marshal.Marshalizer) error {
	codeEntry, err := marshalizer.Marshal(entry)
	if err != nil {
		return err
	}

	err = trie.Update(codeHash, codeEntry)
	if err != nil {
		return err
	}

	return nil
}

// LoadDataTrie retrieves and saves the SC data inside accountHandler object.
// Errors if something went wrong
func (adb *AccountsDB) loadDataTrie(accountHandler baseAccountHandler) error {
	if len(accountHandler.GetRootHash()) == 0 {
		return nil
	}

	dataTrie := adb.dataTries.Get(accountHandler.AddressBytes())
	if dataTrie != nil {
		accountHandler.SetDataTrie(dataTrie)
		return nil
	}

	dataTrie, err := adb.mainTrie.Recreate(accountHandler.GetRootHash())
	if err != nil {
		return NewErrMissingTrie(accountHandler.GetRootHash())
	}

	accountHandler.SetDataTrie(dataTrie)
	adb.dataTries.Put(accountHandler.AddressBytes(), dataTrie)
	return nil
}

// SaveDataTrie is used to save the data trie (not committing it) and to recompute the new Root value
// If data is not dirtied, method will not create its JournalEntries to keep track of data modification
func (adb *AccountsDB) saveDataTrie(accountHandler baseAccountHandler) error {
	if check.IfNil(accountHandler.DataTrieTracker()) {
		return ErrNilTrackableDataTrie
	}
	if len(accountHandler.DataTrieTracker().DirtyData()) == 0 {
		return nil
	}

	log.Trace("accountsDB.SaveDataTrie",
		"address", hex.EncodeToString(accountHandler.AddressBytes()),
		"nonce", accountHandler.GetNonce(),
	)

	if check.IfNil(accountHandler.DataTrie()) {
		newDataTrie, err := adb.mainTrie.Recreate(make([]byte, 0))
		if err != nil {
			return err
		}

		accountHandler.SetDataTrie(newDataTrie)
		adb.dataTries.Put(accountHandler.AddressBytes(), newDataTrie)
	}

	trackableDataTrie := accountHandler.DataTrieTracker()
	dataTrie := trackableDataTrie.DataTrie()
	oldValues := make(map[string][]byte)

	for k, v := range trackableDataTrie.DirtyData() {
		//TODO use trackableDataTrie.originalData() instead of getting from the trie
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

	entry, err := NewJournalEntryDataTrieUpdates(oldValues, accountHandler)
	if err != nil {
		return err
	}
	adb.journalize(entry)

	rootHash, err := trackableDataTrie.DataTrie().Root()
	if err != nil {
		return err
	}

	accountHandler.SetRootHash(rootHash)
	trackableDataTrie.ClearDataCaches()

	log.Trace("accountsDB.SaveDataTrie",
		"address", hex.EncodeToString(accountHandler.AddressBytes()),
		"new root hash", accountHandler.GetRootHash(),
	)

	return nil
}

func (adb *AccountsDB) saveAccountToTrie(accountHandler AccountHandler) error {
	log.Trace("accountsDB.saveAccountToTrie",
		"address", hex.EncodeToString(accountHandler.AddressBytes()),
	)

	//pass the reference to marshalizer, otherwise it will fail marshalizing balance
	buff, err := adb.marshalizer.Marshal(accountHandler)
	if err != nil {
		return err
	}

	return adb.mainTrie.Update(accountHandler.AddressBytes(), buff)
}

// RemoveAccount removes the account data from underlying trie.
// It basically calls Update with empty slice
func (adb *AccountsDB) RemoveAccount(address []byte) error {
	adb.mutOp.Lock()
	defer adb.mutOp.Unlock()

	if len(address) == 0 {
		return fmt.Errorf("%w in RemoveAccount", ErrNilAddress)
	}

	acnt, err := adb.getAccount(address)
	if err != nil {
		return err
	}
	if acnt == nil {
		return fmt.Errorf("%w in RemoveAccount for address %s", ErrAccNotFound, address)
	}

	entry, err := NewJournalEntryAccount(acnt)
	if err != nil {
		return err
	}
	adb.journalize(entry)

	log.Trace("accountsDB.RemoveAccount",
		"address", hex.EncodeToString(address),
	)

	return adb.mainTrie.Update(address, make([]byte, 0))
}

// LoadAccount fetches the account based on the address. Creates an empty account if the account is missing.
func (adb *AccountsDB) LoadAccount(address []byte) (AccountHandler, error) {
	adb.mutOp.Lock()
	defer adb.mutOp.Unlock()

	if len(address) == 0 {
		return nil, fmt.Errorf("%w in LoadAccount", ErrNilAddress)
	}

	log.Trace("accountsDB.LoadAccount",
		"address", hex.EncodeToString(address),
	)

	acnt, err := adb.getAccount(address)
	if err != nil {
		return nil, err
	}
	if acnt == nil {
		return adb.accountFactory.CreateAccount(address)
	}

	baseAcc, ok := acnt.(baseAccountHandler)
	if ok {
		err = adb.loadCode(baseAcc)
		if err != nil {
			return nil, err
		}

		err = adb.loadDataTrie(baseAcc)
		if err != nil {
			return nil, err
		}
	}

	return acnt, nil
}

func (adb *AccountsDB) getAccount(address []byte) (AccountHandler, error) {
	val, err := adb.mainTrie.Get(address)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil
	}

	acnt, err := adb.accountFactory.CreateAccount(address)
	if err != nil {
		return nil, err
	}

	err = adb.marshalizer.Unmarshal(acnt, val)
	if err != nil {
		return nil, err
	}

	return acnt, nil
}

// GetExistingAccount returns an existing account if exists or nil if missing
func (adb *AccountsDB) GetExistingAccount(address []byte) (AccountHandler, error) {
	adb.mutOp.Lock()
	defer adb.mutOp.Unlock()

	if len(address) == 0 {
		return nil, fmt.Errorf("%w in GetExistingAccount", ErrNilAddress)
	}

	log.Trace("accountsDB.GetExistingAccount",
		"address", hex.EncodeToString(address),
	)

	acnt, err := adb.getAccount(address)
	if err != nil {
		return nil, err
	}
	if acnt == nil {
		return nil, ErrAccNotFound
	}

	baseAcc, ok := acnt.(baseAccountHandler)
	if ok {
		err = adb.loadCode(baseAcc)
		if err != nil {
			return nil, err
		}

		err = adb.loadDataTrie(baseAcc)
		if err != nil {
			return nil, err
		}
	}

	return acnt, nil
}

// loadCode retrieves and saves the SC code inside AccountState object. Errors if something went wrong
func (adb *AccountsDB) loadCode(accountHandler baseAccountHandler) error {
	if len(accountHandler.GetCodeHash()) == 0 {
		return nil
	}

	val, err := adb.mainTrie.Get(accountHandler.GetCodeHash())
	if err != nil {
		return err
	}

	var codeEntry CodeEntry
	err = adb.marshalizer.Unmarshal(&codeEntry, val)
	if err != nil {
		return err
	}

	accountHandler.SetCode(codeEntry.Code)
	return nil
}

// RevertToSnapshot apply Revert method over accounts object and removes entries from the list
// Calling with 0 will revert everything. If the snapshot value is out of bounds, an err will be returned
func (adb *AccountsDB) RevertToSnapshot(snapshot int) error {
	log.Trace("accountsDB.RevertToSnapshot started",
		"snapshot", snapshot,
	)

	adb.mutOp.Lock()

	defer func() {
		log.Trace("accountsDB.RevertToSnapshot ended")
		adb.mutOp.Unlock()
	}()

	if snapshot > len(adb.entries) || snapshot < 0 {
		return ErrSnapshotValueOutOfBounds
	}

	if snapshot == 0 {
		log.Trace("revert snapshot to adb.lastRootHash", "hash", adb.lastRootHash)
		return adb.recreateTrie(adb.lastRootHash)
	}

	for i := len(adb.entries) - 1; i >= snapshot; i-- {
		account, err := adb.entries[i].Revert()
		if err != nil {
			return err
		}

		if !check.IfNil(account) {
			err = adb.saveAccountToTrie(account)
			if err != nil {
				return err
			}
		}
	}

	adb.entries = adb.entries[:snapshot]

	return nil
}

// JournalLen will return the number of entries
func (adb *AccountsDB) JournalLen() int {
	adb.mutOp.Lock()
	defer adb.mutOp.Unlock()

	length := len(adb.entries)
	log.Trace("accountsDB.JournalLen",
		"length", length,
	)

	return length
}

// Commit will persist all data inside the trie
func (adb *AccountsDB) Commit() ([]byte, error) {
	adb.mutOp.Lock()
	defer adb.mutOp.Unlock()

	log.Trace("accountsDB.Commit started")
	adb.entries = make([]JournalEntry, 0)

	oldHashes := make([][]byte, 0)
	newHashes := make(data.ModifiedHashes)
	//Step 1. commit all data tries
	dataTries := adb.dataTries.GetAll()
	for i := 0; i < len(dataTries); i++ {
		oldTrieHashes := dataTries[i].ResetOldHashes()
		newTrieHashes, err := dataTries[i].GetDirtyHashes()
		if err != nil {
			return nil, err
		}

		err = dataTries[i].Commit()
		if err != nil {
			return nil, err
		}

		oldHashes = append(oldHashes, oldTrieHashes...)
		for hash := range newTrieHashes {
			newHashes[hash] = struct{}{}
		}
	}
	adb.dataTries.Reset()

	newTrieHashes, err := adb.mainTrie.GetDirtyHashes()
	if err != nil {
		return nil, err
	}
	for hash := range newTrieHashes {
		newHashes[hash] = struct{}{}
	}

	//Step 2. commit main trie
	adb.mainTrie.SetNewHashes(newHashes)
	adb.mainTrie.AppendToOldHashes(oldHashes)
	err = adb.mainTrie.Commit()
	if err != nil {
		return nil, err
	}

	root, err := adb.mainTrie.Root()
	if err != nil {
		log.Trace("accountsDB.Commit ended", "error", err.Error())
		return nil, err
	}
	adb.lastRootHash = root

	log.Trace("accountsDB.Commit ended", "root hash", root)

	return root, nil
}

// RootHash returns the main trie's root hash
func (adb *AccountsDB) RootHash() ([]byte, error) {
	adb.mutOp.Lock()
	defer adb.mutOp.Unlock()

	rootHash, err := adb.mainTrie.Root()
	log.Trace("accountsDB.RootHash",
		"root hash", rootHash,
		"err", err,
	)

	return rootHash, err
}

// RecreateTrie is used to reload the trie based on an existing rootHash
func (adb *AccountsDB) RecreateTrie(rootHash []byte) error {
	adb.mutOp.Lock()
	defer adb.mutOp.Unlock()

	err := adb.recreateTrie(rootHash)
	if err != nil {
		return err
	}
	adb.lastRootHash = rootHash

	return nil
}

func (adb *AccountsDB) recreateTrie(rootHash []byte) error {
	log.Trace("accountsDB.RecreateTrie", "root hash", rootHash)
	defer func() {
		log.Trace("accountsDB.RecreateTrie ended")
	}()

	adb.dataTries.Reset()
	adb.entries = make([]JournalEntry, 0)
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
		account := &userAccount{}
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

// Journalize adds a new object to entries list.
func (adb *AccountsDB) journalize(entry JournalEntry) {
	if check.IfNil(entry) {
		return
	}

	adb.entries = append(adb.entries, entry)
	log.Trace("accountsDB.Journalize", "new length", len(adb.entries))
}

// PruneTrie removes old values from the trie database
func (adb *AccountsDB) PruneTrie(rootHash []byte, identifier data.TriePruningIdentifier) {
	adb.mutOp.Lock()
	defer adb.mutOp.Unlock()

	log.Trace("accountsDB.PruneTrie", "root hash", rootHash)

	adb.mainTrie.Prune(rootHash, identifier)
}

// CancelPrune clears the trie's evictionWaitingList
func (adb *AccountsDB) CancelPrune(rootHash []byte, identifier data.TriePruningIdentifier) {
	adb.mutOp.Lock()
	defer adb.mutOp.Unlock()

	log.Trace("accountsDB.CancelPrune", "root hash", rootHash)

	adb.mainTrie.CancelPrune(rootHash, identifier)
}

// SnapshotState triggers the snapshotting process of the state trie
func (adb *AccountsDB) SnapshotState(rootHash []byte) {
	adb.mutOp.Lock()
	defer adb.mutOp.Unlock()

	log.Trace("accountsDB.SnapshotState", "root hash", rootHash)
	adb.mainTrie.EnterSnapshotMode()

	go func() {
		adb.mainTrie.TakeSnapshot(rootHash)
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
		account := &userAccount{}
		err = adb.marshalizer.Unmarshal(account, leaf)
		if err != nil {
			log.Trace("this must be a leaf with code", "err", err)
			continue
		}

		if len(account.RootHash) > 0 {
			adb.mainTrie.SetCheckpoint(account.RootHash)
		}
	}
}

// SetStateCheckpoint sets a checkpoint for the state trie
func (adb *AccountsDB) SetStateCheckpoint(rootHash []byte) {
	adb.mutOp.Lock()
	defer adb.mutOp.Unlock()

	log.Trace("accountsDB.SetStateCheckpoint", "root hash", rootHash)
	adb.mainTrie.EnterSnapshotMode()

	go func() {
		adb.mainTrie.SetCheckpoint(rootHash)
		adb.snapshotUserAccountDataTrie(rootHash)
		adb.mainTrie.ExitSnapshotMode()
	}()
}

// IsPruningEnabled returns true if state pruning is enabled
func (adb *AccountsDB) IsPruningEnabled() bool {
	return adb.mainTrie.IsPruningEnabled()
}

// GetAllLeaves returns all the leaves from a given rootHash
func (adb *AccountsDB) GetAllLeaves(rootHash []byte) (map[string][]byte, error) {
	adb.mutOp.Lock()
	defer adb.mutOp.Unlock()

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
