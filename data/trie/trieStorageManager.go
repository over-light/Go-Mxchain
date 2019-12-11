package trie

import (
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/mock"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
)

// trieStorageManager manages all the storage operations of the trie (commit, snapshot, checkpoint, pruning)
type trieStorageManager struct {
	db            data.DBWriteCacher
	pruningBuffer [][]byte

	snapshots       []storage.Persister
	snapshotId      int
	snapshotDbCfg   config.DBConfig
	snapshotsBuffer snapshotsBuffer

	dbEvictionWaitingList data.DBRemoveCacher
	storageOperationMutex sync.RWMutex
}

// NewTrieStorageManager creates a new instance of trieStorageManager
func NewTrieStorageManager(db data.DBWriteCacher, snapshotDbCfg config.DBConfig, ewl data.DBRemoveCacher) (*trieStorageManager, error) {
	if check.IfNil(db) {
		return nil, ErrNilDatabase
	}
	if check.IfNil(ewl) {
		return nil, ErrNilEvictionWaitingList
	}

	return &trieStorageManager{
		db:                    db,
		pruningBuffer:         make([][]byte, 0),
		snapshots:             make([]storage.Persister, 0),
		snapshotId:            0,
		snapshotDbCfg:         snapshotDbCfg,
		snapshotsBuffer:       newSnapshotsQueue(),
		dbEvictionWaitingList: ewl,
	}, nil
}

// Database returns the main database
func (tsm *trieStorageManager) Database() data.DBWriteCacher {
	tsm.storageOperationMutex.Lock()
	defer tsm.storageOperationMutex.Unlock()

	return tsm.db
}

// SetDatabase sets the provided database as the main database
func (tsm *trieStorageManager) SetDatabase(db data.DBWriteCacher) {
	tsm.storageOperationMutex.Lock()
	tsm.db = db
	tsm.storageOperationMutex.Unlock()
}

// Clone returns a new instance of trieStorageManager
func (tsm *trieStorageManager) Clone() data.StorageManager {
	tsm.storageOperationMutex.Lock()
	defer tsm.storageOperationMutex.Unlock()

	return &trieStorageManager{
		db:                    tsm.db,
		pruningBuffer:         tsm.pruningBuffer,
		snapshots:             tsm.snapshots,
		snapshotId:            tsm.snapshotId,
		snapshotDbCfg:         tsm.snapshotDbCfg,
		snapshotsBuffer:       tsm.snapshotsBuffer.clone(),
		dbEvictionWaitingList: tsm.dbEvictionWaitingList,
	}
}

// Prune removes the given hash from db
func (tsm *trieStorageManager) Prune(rootHash []byte) error {
	tsm.storageOperationMutex.Lock()
	defer tsm.storageOperationMutex.Unlock()

	if tsm.snapshotsBuffer.len() != 0 {
		tsm.pruningBuffer = append(tsm.pruningBuffer, rootHash)
		return nil
	}

	err := tsm.removeFromDb(rootHash)
	if err != nil {
		return err
	}

	return nil
}

// CancelPrune removes the given hash from the eviction waiting list
func (tsm *trieStorageManager) CancelPrune(rootHash []byte) {
	tsm.storageOperationMutex.Lock()
	defer tsm.storageOperationMutex.Unlock()

	_, _ = tsm.dbEvictionWaitingList.Evict(rootHash)
}

func (tsm *trieStorageManager) removeFromDb(hash []byte) error {
	hashes, err := tsm.dbEvictionWaitingList.Evict(hash)
	if err != nil {
		return err
	}

	for i := range hashes {
		err = tsm.db.Remove(hashes[i])
		if err != nil {
			return err
		}
	}

	return nil
}

// MarkForEviction adds the given hashes in the eviction waiting list at the provided key
func (tsm *trieStorageManager) MarkForEviction(root []byte, hashes [][]byte) error {
	tsm.storageOperationMutex.Lock()
	defer tsm.storageOperationMutex.Unlock()

	return tsm.dbEvictionWaitingList.Put(root, hashes)
}

// GetDbThatContainsHash returns the database that contains the given hash
func (tsm *trieStorageManager) GetDbThatContainsHash(rootHash []byte) data.DBWriteCacher {
	tsm.storageOperationMutex.Lock()
	defer tsm.storageOperationMutex.Unlock()

	_, err := tsm.db.Get(rootHash)

	hashPresent := err == nil
	if hashPresent {
		return tsm.db
	}

	for i := range tsm.snapshots {
		_, err = tsm.snapshots[i].Get(rootHash)

		hashPresent = err == nil
		if hashPresent {
			return tsm.snapshots[i]
		}
	}

	return nil
}

// Snapshot creates a new snapshot, or if there is another snapshot or checkpoint in progress,
// it adds this snapshot in the queue.
func (tsm *trieStorageManager) Snapshot(rootHash []byte, msh marshal.Marshalizer, hsh hashing.Hasher) {
	tsm.storageOperationMutex.Lock()
	defer tsm.storageOperationMutex.Unlock()

	tsm.snapshotsBuffer.add(rootHash, true)
	if tsm.snapshotsBuffer.len() > 1 {
		return
	}

	go tsm.snapshot(msh, hsh)
}

// Checkpoint creates a new checkpoint, or if there is another snapshot or checkpoint in progress,
// it adds this checkpoint in the queue. The checkpoint operation creates a new snapshot file
// only if there was no snapshot done prior to this
func (tsm *trieStorageManager) Checkpoint(rootHash []byte, msh marshal.Marshalizer, hsh hashing.Hasher) {
	tsm.storageOperationMutex.Lock()
	defer tsm.storageOperationMutex.Unlock()

	tsm.snapshotsBuffer.add(rootHash, false)
	if tsm.snapshotsBuffer.len() > 1 {
		return
	}

	go tsm.snapshot(msh, hsh)
}

func (tsm *trieStorageManager) snapshot(msh marshal.Marshalizer, hsh hashing.Hasher) {
	var keys [][]byte
	for {
		tsm.storageOperationMutex.Lock()

		snapshot := tsm.snapshotsBuffer.getFirst()
		tr, err := newSnapshotTrie(tsm.db, msh, hsh, snapshot.rootHash)
		if err != nil {
			log.Error(err.Error())
			return
		}
		db := tsm.getSnapshotDb(snapshot.newDb)

		tsm.storageOperationMutex.Unlock()

		err = tr.root.commit(true, 0, tsm.db, db)
		if err != nil {
			log.Error(err.Error())
			return
		}

		tsm.storageOperationMutex.Lock()
		isEmpty := tsm.snapshotsBuffer.removeFirst()
		if isEmpty {
			keys = tsm.pruningBuffer
			tsm.pruningBuffer = make([][]byte, 0)
			tsm.storageOperationMutex.Unlock()
			break
		}
		tsm.storageOperationMutex.Unlock()
	}

	tsm.removeKeysFromDb(keys)
}

func (tsm *trieStorageManager) removeKeysFromDb(keys [][]byte) {
	for i := range keys {
		tsm.storageOperationMutex.Lock()
		err := tsm.removeFromDb(keys[i])
		if err != nil {
			log.Error(err.Error())
		}
		tsm.storageOperationMutex.Unlock()
	}
}

func (tsm *trieStorageManager) getSnapshotDb(newDb bool) data.DBWriteCacher {
	createNewDb := newDb || len(tsm.snapshots) == 0
	if !createNewDb {
		return tsm.snapshots[len(tsm.snapshots)-1]
	}

	db, err := tsm.newSnapshotDb()
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	if len(tsm.snapshots) > maxSnapshots {
		tsm.removeSnapshot()
	}

	return db
}

func (tsm *trieStorageManager) removeSnapshot() {
	dbUniqueId := strconv.Itoa(tsm.snapshotId - len(tsm.snapshots))

	err := tsm.snapshots[0].Close()
	if err != nil {
		log.Error(err.Error())
		return
	}
	tsm.snapshots = tsm.snapshots[1:]

	removePath := path.Join(tsm.snapshotDbCfg.FilePath, dbUniqueId)
	go removeDirectory(removePath)
}

func removeDirectory(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		log.Error(err.Error())
	}
}

func newSnapshotTrie(
	db data.DBWriteCacher,
	msh marshal.Marshalizer,
	hsh hashing.Hasher,
	rootHash []byte,
) (*patriciaMerkleTrie, error) {
	newRoot, err := getNodeFromDBAndDecode(rootHash, db, msh, hsh)
	if err != nil {
		return nil, err
	}

	trieStorage, err := NewTrieStorageManager(db, config.DBConfig{}, &mock.EvictionWaitingList{})
	if err != nil {
		return nil, err
	}

	return &patriciaMerkleTrie{
		root:        newRoot,
		trieStorage: trieStorage,
		marshalizer: msh,
		hasher:      hsh,
	}, nil
}

func (tsm *trieStorageManager) newSnapshotDb() (storage.Persister, error) {
	snapshotPath := path.Join(tsm.snapshotDbCfg.FilePath, strconv.Itoa(tsm.snapshotId))
	_, err := os.Stat(snapshotPath)
	for err == nil {
		tsm.snapshotId++
		snapshotPath = path.Join(tsm.snapshotDbCfg.FilePath, strconv.Itoa(tsm.snapshotId))
		_, err = os.Stat(snapshotPath)
	}

	db, err := storageUnit.NewDB(
		storageUnit.DBType(tsm.snapshotDbCfg.Type),
		snapshotPath,
		tsm.snapshotDbCfg.BatchDelaySeconds,
		tsm.snapshotDbCfg.MaxBatchSize,
		tsm.snapshotDbCfg.MaxOpenFiles,
	)
	if err != nil {
		return nil, err
	}

	tsm.snapshotId++
	tsm.snapshots = append(tsm.snapshots, db)

	return db, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (tsm *trieStorageManager) IsInterfaceNil() bool {
	return tsm == nil
}
