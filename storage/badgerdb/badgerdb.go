package badgerdb

import (
	"os"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go/logger"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/options"
)

// read + write + execute for owner only
const rwxOwner = 0700

var log = logger.GetOrCreate("storage/badgerdb")

// DB holds a pointer to the badger database and the path to where it is stored.
type DB struct {
	db                *badger.DB
	path              string
	maxBatchSize      int
	batchDelaySeconds int
	sizeBatch         int
	batch             storage.Batcher
	mutBatch          sync.RWMutex
	dbClosed          chan struct{}
}

// NewDB is a constructor for the badger persister
// It creates the files in the location given as parameter
func NewDB(path string, batchDelaySeconds int, maxBatchSize int) (s *DB, err error) {
	opts := badger.DefaultOptions(path)
	opts.Dir = path
	opts.ValueDir = path
	opts.ValueLogLoadingMode = options.FileIO

	err = os.MkdirAll(path, rwxOwner)
	if err != nil {
		return nil, err
	}

	db, err := badger.Open(opts)
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

	dbStore.batch = dbStore.createBatch()

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
				log.Warn("badger putBatch", "error", err.Error())
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
		log.Warn("badger putBatch", "error", err.Error())
		return err
	}

	s.batch.Reset()
	s.sizeBatch = 0

	return err
}

// CreateBatch returns a batcher to be used for batch writing data to the database
func (s *DB) createBatch() storage.Batcher {
	return NewBatch(s)
}

// putBatch writes the Batch data into the database
func (s *DB) putBatch(b storage.Batcher) error {
	batch, ok := b.(*batch)
	if !ok {
		return storage.ErrInvalidBatch
	}

	return batch.batch.Commit()
}

// Get returns the value associated to the key
func (s *DB) Get(key []byte) ([]byte, error) {
	var value []byte

	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		value, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return value, nil
}

// Has returns true if the given key is present in the persistence medium
func (s *DB) Has(key []byte) error {
	err := s.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

// Init initializes the storage medium and prepares it for usage
func (s *DB) Init() error {
	// no special initialization needed
	return nil
}

// Close closes the files/resources associated to the storage medium
func (s *DB) Close() error {
	s.mutBatch.Lock()
	err := s.putBatch(s.batch)
	s.mutBatch.Unlock()
	if err != nil {
		return err
	}

	s.dbClosed <- struct{}{}

	return s.db.Close()
}

// Remove removes the data associated to the given key
func (s *DB) Remove(key []byte) error {
	s.mutBatch.Lock()
	_ = s.batch.Delete(key)
	s.mutBatch.Unlock()

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

// Destroy removes the storage medium stored data
func (s *DB) Destroy() error {
	err := s.db.Close()
	if err != nil {
		return err
	}

	s.dbClosed <- struct{}{}
	err = os.RemoveAll(s.path)

	return err
}

// DestroyClosed removes the already closed storage medium stored data
func (s *DB) DestroyClosed() error {
	return os.RemoveAll(s.path)
}

// IsInterfaceNil returns true if there is no value under the interface
func (s *DB) IsInterfaceNil() bool {
	if s == nil {
		return true
	}
	return false
}
