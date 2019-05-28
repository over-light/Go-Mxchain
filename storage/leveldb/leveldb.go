package leveldb

import (
	"os"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go-sandbox/core/logger"
	"github.com/ElrondNetwork/elrond-go-sandbox/storage"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const maxOpenFilesPerTable = 50

// read + write + execute for owner only
const rwxOwner = 0700

var log = logger.DefaultLogger()

// DB holds a pointer to the leveldb database and the path to where it is stored.
type DB struct {
	db                *leveldb.DB
	path              string
	maxBatchSize      int
	batchDelaySeconds int
	sizeBatch         int
	batch             storage.Batcher
	mutBatch          sync.RWMutex
	dbClosed          chan struct{}
}

// NewDB is a constructor for the leveldb persister
// It creates the files in the location given as parameter
func NewDB(path string, batchDelaySeconds int, maxBatchSize int) (s *DB, err error) {
	err = os.MkdirAll(path, rwxOwner)
	if err != nil {
		return nil, err
	}

	options := &opt.Options{
		OpenFilesCacheCapacity: maxOpenFilesPerTable,
	}

	db, err := leveldb.OpenFile(path, options)
	if err != nil {
		return nil, err
	}

	dbStore := &DB{
		db:                db,
		path:              path,
		maxBatchSize:      maxBatchSize,
		batchDelaySeconds: batchDelaySeconds,
		sizeBatch:         0,
		dbClosed:          make(chan struct{}),
	}

	dbStore.batch = s.createBatch()

	go dbStore.batchTimeoutHandle()

	return dbStore, nil
}

func (s *DB) batchTimeoutHandle() {
	for {
		select {
		case <-time.After(time.Duration(s.batchDelaySeconds) * time.Second):
			s.mutBatch.Lock()
			err := s.putBatch(s.batch)
			if err != nil {
				log.Error(err.Error())
				s.mutBatch.Unlock()
				continue
			}
			s.batch.Reset()
			s.sizeBatch = 0
			s.mutBatch.Unlock()
		case <-s.dbClosed:
			return
		}
	}
}

// Put adds the value to the (key, val) storage medium
func (s *DB) Put(key, val []byte) error {
	err := s.batch.Put(key, val)
	if err != nil {
		return err
	}

	s.mutBatch.Lock()
	defer s.mutBatch.Unlock()

	s.sizeBatch++
	if s.sizeBatch < s.maxBatchSize {
		return nil
	}

	err = s.putBatch(s.batch)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	s.batch.Reset()
	s.sizeBatch = 0

	return nil
}

// Get returns the value associated to the key
func (s *DB) Get(key []byte) ([]byte, error) {
	has, err := s.db.Has(key, nil)
	if err != nil || !has {
		return nil, storage.ErrKeyNotFound
	}

	data, err := s.db.Get(key, nil)
	if err == leveldb.ErrNotFound {
		return nil, storage.ErrKeyNotFound
	}

	return data, nil
}

// Has returns true if the given key is present in the persistance medium
func (s *DB) Has(key []byte) error {
	has, err := s.db.Has(key, nil)
	if err != nil {
		return err
	}

	if has {
		return nil
	}

	return storage.ErrKeyNotFound
}

// Init initializes the storage medium and prepares it for usage
func (s *DB) Init() error {
	// no special initialization needed
	return nil
}

// CreateBatch returns a batcher to be used for batch writing data to the database
func (s *DB) createBatch() storage.Batcher {
	return NewBatch()
}

// PutBatch writes the Batch data into the database
func (s *DB) putBatch(b storage.Batcher) error {
	batch, ok := b.(*Batch)
	if !ok {
		return storage.ErrInvalidBatch
	}

	return s.db.Write(batch.batch, nil)
}

// Close closes the files/resources associated to the storage medium
func (s *DB) Close() error {
	s.dbClosed <- struct{}{}
	return s.db.Close()
}

// Remove removes the data associated to the given key
func (s *DB) Remove(key []byte) error {
	_ = s.batch.Delete(key)

	return s.db.Delete(key, nil)
}

// Destroy removes the storage medium stored data
func (s *DB) Destroy() error {
	s.dbClosed <- struct{}{}
	_ = s.db.Close()
	err := os.RemoveAll(s.path)
	return err
}
