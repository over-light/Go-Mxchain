package trie

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
)

// trieStorageManager manages all the storage operations of the trie (commit, snapshot, checkpoint, pruning)
type trieStorageManager struct {
	db       data.DBWriteCacher
	pruneReq chan []byte

	snapshots          []storage.Persister
	snapshotId         int
	snapshotDbCfg      config.DBConfig
	snapshotReq        chan snapshotsQueueEntry
	pruningBuffer      atomicBuffer
	snapshotInProgress uint32
	maxSnapshots       uint8

	dbEvictionWaitingList data.DBRemoveCacher
	storageOperationMutex sync.RWMutex
}

type snapshotsQueueEntry struct {
	rootHash []byte
	newDb    bool
}

// NewTrieStorageManager creates a new instance of trieStorageManager
func NewTrieStorageManager(
	db data.DBWriteCacher,
	marshalizer marshal.Marshalizer,
	hasher hashing.Hasher,
	snapshotDbCfg config.DBConfig,
	ewl data.DBRemoveCacher,
	generalConfig config.TrieStorageManagerConfig,
) (*trieStorageManager, error) {
	if check.IfNil(db) {
		return nil, ErrNilDatabase
	}
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(hasher) {
		return nil, ErrNilHasher
	}
	if check.IfNil(ewl) {
		return nil, ErrNilEvictionWaitingList
	}

	snapshots, snapshotId, err := getSnapshotsAndSnapshotId(snapshotDbCfg)
	if err != nil {
		log.Debug("get snapshot", "error", err.Error())
	}

	tsm := &trieStorageManager{
		db:                    db,
		snapshots:             snapshots,
		snapshotId:            snapshotId,
		snapshotDbCfg:         snapshotDbCfg,
		pruningBuffer:         newPruningBuffer(generalConfig.PruningBufferLen),
		dbEvictionWaitingList: ewl,
		snapshotReq:           make(chan snapshotsQueueEntry, generalConfig.SnapshotsBufferLen),
		pruneReq:              make(chan []byte, generalConfig.PruningBufferLen),
		snapshotInProgress:    0,
		maxSnapshots:          generalConfig.MaxSnapshots,
	}

	go tsm.storageProcessLoop(marshalizer, hasher)
	return tsm, nil
}

func (tsm *trieStorageManager) storageProcessLoop(msh marshal.Marshalizer, hsh hashing.Hasher) {
	for {
		select {
		case snapshot := <-tsm.snapshotReq:
			tsm.takeSnapshot(snapshot, msh, hsh)
		default:
			select {
			case snapshot := <-tsm.snapshotReq:
				tsm.takeSnapshot(snapshot, msh, hsh)
			case rootHash := <-tsm.pruneReq:
				err := tsm.removeFromDb(rootHash)
				if err != nil {
					log.Error("trie storage manager remove from db", "error", err, "rootHash", hex.EncodeToString(rootHash))
				}
			}
		}
	}
}

func getSnapshotsAndSnapshotId(snapshotDbCfg config.DBConfig) ([]storage.Persister, int, error) {
	snapshots := make([]storage.Persister, 0)
	snapshotId := 0

	if !directoryExists(snapshotDbCfg.FilePath) {
		return snapshots, snapshotId, nil
	}

	files, err := ioutil.ReadDir(snapshotDbCfg.FilePath)
	if err != nil {
		return snapshots, snapshotId, err
	}

	for _, f := range files {
		if !f.IsDir() {
			continue
		}

		var snapshotName int
		snapshotName, err = strconv.Atoi(f.Name())
		if err != nil {
			return snapshots, snapshotId, err
		}

		var db storage.Persister
		arg := storageUnit.ArgDB{
			DBType:            storageUnit.DBType(snapshotDbCfg.Type),
			Path:              path.Join(snapshotDbCfg.FilePath, f.Name()),
			BatchDelaySeconds: snapshotDbCfg.BatchDelaySeconds,
			MaxBatchSize:      snapshotDbCfg.MaxBatchSize,
			MaxOpenFiles:      snapshotDbCfg.MaxOpenFiles,
		}
		db, err = storageUnit.NewDB(arg)
		if err != nil {
			return snapshots, snapshotId, err
		}

		if snapshotName > snapshotId {
			snapshotId = snapshotName
		}

		snapshots = append(snapshots, db)
	}

	if len(snapshots) != 0 {
		snapshotId++
	}

	return snapshots, snapshotId, nil
}

// Database returns the main database
func (tsm *trieStorageManager) Database() data.DBWriteCacher {
	return tsm.db
}

// EnterSnapshotMode sets the snapshot mode on
func (tsm *trieStorageManager) EnterSnapshotMode() {
	tsm.storageOperationMutex.Lock()
	defer tsm.storageOperationMutex.Unlock()

	tsm.snapshotInProgress++

	log.Trace("enter snapshot mode", "snapshots in progress", tsm.snapshotInProgress)
}

// ExitSnapshotMode sets the snapshot mode off
func (tsm *trieStorageManager) ExitSnapshotMode() {
	tsm.storageOperationMutex.Lock()
	defer tsm.storageOperationMutex.Unlock()

	if tsm.snapshotInProgress < 1 {
		log.Error("ExitSnapshotMode called too many times")
	}

	if tsm.snapshotInProgress > 0 {
		tsm.snapshotInProgress--
	}

	if tsm.snapshotInProgress == 0 {
		tsm.prune(tsm.pruningBuffer.removeAll())
	}

	log.Trace("exit snapshot mode", "snapshots in progress", tsm.snapshotInProgress)
}

// Prune removes the given hash from db
func (tsm *trieStorageManager) Prune(rootHash []byte) {
	tsm.storageOperationMutex.Lock()
	defer tsm.storageOperationMutex.Unlock()

	log.Trace("trie storage manager prune", "root", rootHash)

	if tsm.snapshotInProgress > 0 {
		tsm.pruningBuffer.add(rootHash)
		return
	}

	oldHashes := tsm.pruningBuffer.removeAll()
	oldHashes[string(rootHash)] = struct{}{}
	tsm.prune(oldHashes)
}

func (tsm *trieStorageManager) prune(oldHashes map[string]struct{}) {
	for key := range oldHashes {
		select {
		case tsm.pruneReq <- []byte(key):
			log.Trace("root hash will be pruned", "rootHash", []byte(key))
		default:
			log.Trace("pruning buffer is full, hash won't be removed", "hash", []byte(key))
		}
	}
}

// CancelPrune removes the given hash from the eviction waiting list
func (tsm *trieStorageManager) CancelPrune(rootHash []byte) {
	tsm.storageOperationMutex.Lock()
	defer tsm.storageOperationMutex.Unlock()

	log.Trace("trie storage manager cancel prune", "root", rootHash)
	_, _ = tsm.dbEvictionWaitingList.Evict(rootHash)
	tsm.pruningBuffer.remove(rootHash)
}

func (tsm *trieStorageManager) removeFromDb(rootHash []byte) error {
	hashes, err := tsm.dbEvictionWaitingList.Evict(rootHash)
	if err != nil {
		return err
	}

	log.Debug("trie removeFromDb", "rootHash", rootHash)

	var hash []byte
	var present bool
	for key := range hashes {
		present, err = tsm.dbEvictionWaitingList.PresentInNewHashes(key)
		if err != nil {
			return err
		}
		if present {
			continue
		}

		hash, err = hex.DecodeString(key)
		if err != nil {
			return err
		}

		log.Trace("remove hash from trie db", "hash", hex.EncodeToString(hash))
		err = tsm.db.Remove(hash)
		if err != nil {
			return err
		}
	}

	return nil
}

// MarkForEviction adds the given hashes in the eviction waiting list at the provided key
func (tsm *trieStorageManager) MarkForEviction(root []byte, hashes data.ModifiedHashes) error {
	log.Trace("trie storage manager: mark for eviction", "root", root)
	return tsm.dbEvictionWaitingList.Put(root, hashes)
}

// GetDbThatContainsHash returns the database that contains the given hash
func (tsm *trieStorageManager) GetDbThatContainsHash(rootHash []byte) data.DBWriteCacher {
	_, err := tsm.db.Get(rootHash)

	hashPresent := err == nil
	if hashPresent {
		return tsm.db
	}

	return tsm.getSnapshotDbThatContainsHash(rootHash)
}

func (tsm *trieStorageManager) getSnapshotDbThatContainsHash(rootHash []byte) data.DBWriteCacher {
	tsm.storageOperationMutex.Lock()
	defer tsm.storageOperationMutex.Unlock()

	for i := range tsm.snapshots {
		_, err := tsm.snapshots[i].Get(rootHash)

		hashPresent := err == nil
		if hashPresent {
			return tsm.snapshots[i]
		}
	}

	return nil
}

// TakeSnapshot creates a new snapshot, or if there is another snapshot or checkpoint in progress,
// it adds this snapshot in the queue.
func (tsm *trieStorageManager) TakeSnapshot(rootHash []byte) {
	snapshotEntry := snapshotsQueueEntry{rootHash: rootHash, newDb: true}
	tsm.writeOnChan(snapshotEntry)
}

// SetCheckpoint creates a new checkpoint, or if there is another snapshot or checkpoint in progress,
// it adds this checkpoint in the queue. The checkpoint operation creates a new snapshot file
// only if there was no snapshot done prior to this
func (tsm *trieStorageManager) SetCheckpoint(rootHash []byte) {
	checkpointEntry := snapshotsQueueEntry{rootHash: rootHash, newDb: false}
	tsm.writeOnChan(checkpointEntry)
}

func (tsm *trieStorageManager) writeOnChan(entry snapshotsQueueEntry) {
	select {
	case tsm.snapshotReq <- entry:
		return
	default:
		log.Debug("snapshots buffer is full")
		return
	}
}

func (tsm *trieStorageManager) takeSnapshot(snapshot snapshotsQueueEntry, msh marshal.Marshalizer, hsh hashing.Hasher) {
	if tsm.getSnapshotDbThatContainsHash(snapshot.rootHash) != nil {
		log.Trace("snapshot for rootHash already taken", "rootHash", snapshot.rootHash)
		return
	}

	log.Debug("trie snapshot started", "rootHash", snapshot.rootHash)

	newRoot, err := newSnapshotNode(tsm.db, msh, hsh, snapshot.rootHash)
	if err != nil {
		log.Error("trie storage manager: newSnapshotTrie", "error", err.Error())
		return
	}
	db := tsm.getSnapshotDb(snapshot.newDb)
	if check.IfNil(db) {
		return
	}

	err = newRoot.commit(true, 0, tsm.db, db)
	if err != nil {
		log.Error("trie storage manager: commit", "error", err.Error())
		return
	}

	log.Debug("trie snapshot finished", "rootHash", snapshot.rootHash)
}

func (tsm *trieStorageManager) getSnapshotDb(newDb bool) data.DBWriteCacher {
	tsm.storageOperationMutex.Lock()
	defer tsm.storageOperationMutex.Unlock()

	createNewDb := newDb || len(tsm.snapshots) == 0
	if !createNewDb {
		return tsm.snapshots[len(tsm.snapshots)-1]
	}

	db, err := tsm.newSnapshotDb()
	if err != nil {
		log.Error("trie storage manager: getSnapshotDb", "error", err.Error())
		return nil
	}

	if uint8(len(tsm.snapshots)) > tsm.maxSnapshots {
		tsm.removeSnapshot()
	}

	return db
}

func (tsm *trieStorageManager) removeSnapshot() {
	dbUniqueId := strconv.Itoa(tsm.snapshotId - len(tsm.snapshots))

	err := tsm.snapshots[0].Close()
	if err != nil {
		log.Error("trie storage manager: removeSnapshot", "error", err.Error())
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

func newSnapshotNode(
	db data.DBWriteCacher,
	msh marshal.Marshalizer,
	hsh hashing.Hasher,
	rootHash []byte,
) (snapshotNode, error) {
	newRoot, err := getNodeFromDBAndDecode(rootHash, db, msh, hsh)
	if err != nil {
		return nil, err
	}

	trieStorage := &trieStorageManager{
		db: db,
	}

	snapshotPmt := &patriciaMerkleTrie{
		root:        newRoot,
		trieStorage: trieStorage,
		marshalizer: msh,
		hasher:      hsh,
	}

	return snapshotPmt.root, nil
}

func (tsm *trieStorageManager) newSnapshotDb() (storage.Persister, error) {
	snapshotPath := path.Join(tsm.snapshotDbCfg.FilePath, strconv.Itoa(tsm.snapshotId))
	for directoryExists(snapshotPath) {
		tsm.snapshotId++
		snapshotPath = path.Join(tsm.snapshotDbCfg.FilePath, strconv.Itoa(tsm.snapshotId))
	}

	arg := storageUnit.ArgDB{
		DBType:            storageUnit.DBType(tsm.snapshotDbCfg.Type),
		Path:              snapshotPath,
		BatchDelaySeconds: tsm.snapshotDbCfg.BatchDelaySeconds,
		MaxBatchSize:      tsm.snapshotDbCfg.MaxBatchSize,
		MaxOpenFiles:      tsm.snapshotDbCfg.MaxOpenFiles,
	}
	db, err := storageUnit.NewDB(arg)
	if err != nil {
		return nil, err
	}

	tsm.snapshotId++
	tsm.snapshots = append(tsm.snapshots, db)

	return db, nil
}

func directoryExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// IsPruningEnabled returns true if the trie pruning is enabled
func (tsm *trieStorageManager) IsPruningEnabled() bool {
	return true
}

// IsInterfaceNil returns true if there is no value under the interface
func (tsm *trieStorageManager) IsInterfaceNil() bool {
	return tsm == nil
}
