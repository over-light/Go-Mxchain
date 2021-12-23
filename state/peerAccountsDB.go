package state

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	"github.com/ElrondNetwork/elrond-go/common"
)

// PeerAccountsDB will save and synchronize data from peer processor, plus will synchronize with nodesCoordinator
type PeerAccountsDB struct {
	*AccountsDB
}

// NewPeerAccountsDB creates a new account manager
func NewPeerAccountsDB(
	trie common.Trie,
	hasher hashing.Hasher,
	marshalizer marshal.Marshalizer,
	accountFactory AccountFactory,
	storagePruningManager StoragePruningManager,
) (*PeerAccountsDB, error) {
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
	if check.IfNil(storagePruningManager) {
		return nil, ErrNilStoragePruningManager
	}

	numCheckpoints := getNumCheckpoints(trie.GetStorageManager())
	return &PeerAccountsDB{
		&AccountsDB{
			mainTrie:       trie,
			hasher:         hasher,
			marshalizer:    marshalizer,
			accountFactory: accountFactory,
			entries:        make([]JournalEntry, 0),
			dataTries:      NewDataTriesHolder(),
			mutOp:          sync.RWMutex{},
			numCheckpoints: numCheckpoints,
			loadCodeMeasurements: &loadingMeasurements{
				identifier: "load code",
			},
			storagePruningManager: storagePruningManager,
		},
	}, nil
}

// SnapshotState triggers the snapshotting process of the state trie
func (adb *PeerAccountsDB) SnapshotState(rootHash []byte) {
	log.Trace("peerAccountsDB.SnapshotState", "root hash", rootHash)
	trieStorageManager := adb.mainTrie.GetStorageManager()
	if !trieStorageManager.ShouldTakeSnapshot() {
		log.Debug("skipping snapshot for rootHash", "hash", rootHash)
		return
	}

	stats := newSnapshotStatistics(0)

	trieStorageManager.EnterPruningBufferingMode()
	stats.NewSnapshotStarted()
	trieStorageManager.TakeSnapshot(rootHash, nil, stats)
	trieStorageManager.ExitPruningBufferingMode()

	go func() {
		printStats(stats, "snapshotState peer trie", rootHash)

		err := trieStorageManager.Put([]byte(common.ActiveDBKey), []byte(common.ActiveDBVal))
		if err != nil {
			log.Warn("error while putting active DB value into main storer", "error", err)
		}
	}()

	if adb.processingMode == common.ImportDb {
		stats.WaitForSnapshotsToFinish()
	}

	adb.increaseNumCheckpoints()
}

// SetStateCheckpoint triggers the checkpointing process of the state trie
func (adb *PeerAccountsDB) SetStateCheckpoint(rootHash []byte) {
	log.Trace("peerAccountsDB.SetStateCheckpoint", "root hash", rootHash)
	trieStorageManager := adb.mainTrie.GetStorageManager()

	stats := newSnapshotStatistics(0)

	trieStorageManager.EnterPruningBufferingMode()
	stats.NewSnapshotStarted()
	trieStorageManager.SetCheckpoint(rootHash, nil, stats)
	trieStorageManager.ExitPruningBufferingMode()

	go printStats(stats, "snapshotState peer trie", rootHash)

	if adb.processingMode == common.ImportDb {
		stats.WaitForSnapshotsToFinish()
	}

	adb.increaseNumCheckpoints()
}

// RecreateAllTries recreates all the tries from the accounts DB
func (adb *PeerAccountsDB) RecreateAllTries(rootHash []byte) (map[string]common.Trie, error) {
	recreatedTrie, err := adb.mainTrie.Recreate(rootHash)
	if err != nil {
		return nil, err
	}

	allTries := make(map[string]common.Trie)
	allTries[string(rootHash)] = recreatedTrie

	return allTries, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (adb *PeerAccountsDB) IsInterfaceNil() bool {
	return adb == nil
}
