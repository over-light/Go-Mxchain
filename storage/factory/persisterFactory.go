package factory

import (
	"errors"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/badgerdb"
	"github.com/ElrondNetwork/elrond-go/storage/boltdb"
	"github.com/ElrondNetwork/elrond-go/storage/leveldb"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
)

// PersisterFactory is the factory which will handle creating new databases
type PersisterFactory struct {
	dbType            string
	batchDelaySeconds int
	maxBatchSize      int
	maxOpenFiles      int
}

// NewPersisterFactory will return a new instance of a PersisterFactory
func NewPersisterFactory(config config.DBConfig) *PersisterFactory {
	return &PersisterFactory{
		dbType:            config.Type,
		batchDelaySeconds: config.BatchDelaySeconds,
		maxBatchSize:      config.MaxBatchSize,
		maxOpenFiles:      config.MaxOpenFiles,
	}
}

// Create will return a new instance of a DB with a given path
func (df *PersisterFactory) Create(path string) (storage.Persister, error) {
	if len(path) < 0 {
		return nil, errors.New("invalid file path")
	}

	switch storageUnit.DBType(df.dbType) {
	case storageUnit.LvlDB:
		return leveldb.NewDB(path, df.batchDelaySeconds, df.maxBatchSize, df.maxOpenFiles)
	case storageUnit.LvlDbSerial:
		return leveldb.NewSerialDB(path, df.batchDelaySeconds, df.maxBatchSize, df.maxOpenFiles)
	case storageUnit.BadgerDB:
		return badgerdb.NewDB(path, df.batchDelaySeconds, df.maxBatchSize)
	case storageUnit.BoltDB:
		return boltdb.NewDB(path, df.batchDelaySeconds, df.maxBatchSize)
	default:
		return nil, storage.ErrNotSupportedDBType
	}
}

// IsInterfaceNil returns true if there is no value under the interface
func (pf *PersisterFactory) IsInterfaceNil() bool {
	return pf == nil
}
