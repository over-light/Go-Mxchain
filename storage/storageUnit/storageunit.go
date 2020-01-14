package storageUnit

import (
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/hashing/blake2b"
	"github.com/ElrondNetwork/elrond-go/hashing/fnv"
	"github.com/ElrondNetwork/elrond-go/hashing/keccak"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/badgerdb"
	"github.com/ElrondNetwork/elrond-go/storage/bloom"
	"github.com/ElrondNetwork/elrond-go/storage/boltdb"
	"github.com/ElrondNetwork/elrond-go/storage/fifocache"
	"github.com/ElrondNetwork/elrond-go/storage/leveldb"
	"github.com/ElrondNetwork/elrond-go/storage/lrucache"
)

// CacheType represents the type of the supported caches
type CacheType string

// DBType represents the type of the supported databases
type DBType string

// HasherType represents the type of the supported hash functions
type HasherType string

// LRUCache is currently the only supported Cache type
const (
	LRUCache         CacheType = "LRU"
	FIFOShardedCache CacheType = "FIFOSharded"
)

// LvlDB currently the only supported DBs
// More to be added
const (
	LvlDB       DBType = "LvlDB"
	LvlDbSerial DBType = "LvlDBSerial"
	BadgerDB    DBType = "BadgerDB"
	BoltDB      DBType = "BoltDB"
)

const (
	// Keccak is the string representation of the keccak hashing function
	Keccak HasherType = "Keccak"
	// Blake2b is the string representation of the blake2b hashing function
	Blake2b HasherType = "Blake2b"
	// Fnv is the string representation of the fnv hashing function
	Fnv HasherType = "Fnv"
)

// UnitConfig holds the configurable elements of the storage unit
type UnitConfig struct {
	CacheConf CacheConfig
	DBConf    DBConfig
	BloomConf BloomConfig
}

// CacheConfig holds the configurable elements of a cache
type CacheConfig struct {
	Size   uint32
	Type   CacheType
	Shards uint32
}

// DBConfig holds the configurable elements of a database
type DBConfig struct {
	FilePath          string
	Type              DBType
	BatchDelaySeconds int
	MaxBatchSize      int
	MaxOpenFiles      int
}

// BloomConfig holds the configurable elements of a bloom filter
type BloomConfig struct {
	Size     uint
	HashFunc []HasherType
}

// Unit represents a storer's data bank
// holding the cache, persistence unit and bloom filter
type Unit struct {
	lock        sync.RWMutex
	batcher     storage.Batcher
	persister   storage.Persister
	cacher      storage.Cacher
	bloomFilter storage.BloomFilter
}

// Put adds data to both cache and persistence medium and updates the bloom filter
func (u *Unit) Put(key, data []byte) error {
	u.lock.Lock()
	defer u.lock.Unlock()

	u.cacher.Put(key, data)

	err := u.persister.Put(key, data)
	if err != nil {
		u.cacher.Remove(key)
		return err
	}

	if u.bloomFilter != nil {
		u.bloomFilter.Add(key)
	}

	return err
}

// Get searches the key in the cache. In case it is not found, it searches
// for the key in bloom filter first and if found
// it further searches it in the associated database.
// In case it is found in the database, the cache is updated with the value as well.
func (u *Unit) Get(key []byte) ([]byte, error) {
	u.lock.Lock()
	defer u.lock.Unlock()

	v, ok := u.cacher.Get(key)
	var err error

	if !ok {
		// not found in cache
		// search it in second persistence medium
		if u.bloomFilter == nil || u.bloomFilter.MayContain(key) {
			v, err = u.persister.Get(key)

			if err != nil {
				return nil, err
			}

			// if found in persistence unit, add it in cache
			u.cacher.Put(key, v)
		} else {
			return nil, errors.New(fmt.Sprintf("key: %s not found", base64.StdEncoding.EncodeToString(key)))
		}
	}

	return v.([]byte), nil
}

// Has checks if the key is in the Unit.
// It first checks the cache. If it is not found, it checks the bloom filter
// and if present it checks the db
func (u *Unit) Has(key []byte) error {
	u.lock.RLock()
	defer u.lock.RUnlock()

	has := u.cacher.Has(key)
	if has {
		return nil
	}

	if u.bloomFilter == nil || u.bloomFilter.MayContain(key) {
		return u.persister.Has(key)
	}

	return storage.ErrKeyNotFound
}

// Remove removes the data associated to the given key from both cache and persistence medium
func (u *Unit) Remove(key []byte) error {
	u.lock.Lock()
	defer u.lock.Unlock()

	u.cacher.Remove(key)
	err := u.persister.Remove(key)

	return err
}

// ClearCache cleans up the entire cache
func (u *Unit) ClearCache() {
	u.cacher.Clear()
}

// DestroyUnit cleans up the bloom filter, the cache, and the db
func (u *Unit) DestroyUnit() error {
	u.lock.Lock()
	defer u.lock.Unlock()

	if u.bloomFilter != nil {
		u.bloomFilter.Clear()
	}

	u.cacher.Clear()
	return u.persister.Destroy()
}

// GetFromEpoch will call the Get method as this storer doesn't handle epochs
func (u *Unit) GetFromEpoch(key []byte, _ uint32) ([]byte, error) {
	return u.Get(key)
}

// SearchFirst will call the Get method as this storer doesn't handle epochs
func (u *Unit) SearchFirst(key []byte) ([]byte, error) {
	return u.Get(key)
}

// HasInEpoch will call the Has method as this storer doesn't handle epochs
func (u *Unit) HasInEpoch(key []byte, _ uint32) error {
	return u.Has(key)
}

// IsInterfaceNil returns true if there is no value under the interface
func (u *Unit) IsInterfaceNil() bool {
	return u == nil
}

// NewStorageUnit is the constructor for the storage unit, creating a new storage unit
// from the given cacher and persister.
func NewStorageUnit(c storage.Cacher, p storage.Persister) (*Unit, error) {
	if p == nil || p.IsInterfaceNil() {
		return nil, storage.ErrNilPersister
	}
	if c == nil || c.IsInterfaceNil() {
		return nil, storage.ErrNilCacher
	}

	sUnit := &Unit{
		persister:   p,
		cacher:      c,
		bloomFilter: nil,
	}

	err := sUnit.persister.Init()
	if err != nil {
		return nil, err
	}

	return sUnit, nil
}

// NewStorageUnitWithBloomFilter is the constructor for the storage unit, creating a new storage unit
// from the given cacher, persister and bloom filter.
func NewStorageUnitWithBloomFilter(c storage.Cacher, p storage.Persister, b storage.BloomFilter) (*Unit, error) {
	if p == nil || p.IsInterfaceNil() {
		return nil, storage.ErrNilPersister
	}
	if c == nil || c.IsInterfaceNil() {
		return nil, storage.ErrNilCacher
	}
	if b == nil || b.IsInterfaceNil() {
		return nil, storage.ErrNilBloomFilter
	}

	sUnit := &Unit{
		persister:   p,
		cacher:      c,
		bloomFilter: b,
	}

	err := sUnit.persister.Init()
	if err != nil {
		return nil, err
	}

	return sUnit, nil
}

// NewStorageUnitFromConf creates a new storage unit from a storage unit config
func NewStorageUnitFromConf(cacheConf CacheConfig, dbConf DBConfig, bloomFilterConf BloomConfig) (*Unit, error) {
	var cache storage.Cacher
	var db storage.Persister
	var bf storage.BloomFilter
	var err error

	defer func() {
		if err != nil && db != nil {
			_ = db.Destroy()
		}
	}()

	cache, err = NewCache(cacheConf.Type, cacheConf.Size, cacheConf.Shards)
	if err != nil {
		return nil, err
	}

	db, err = NewDB(dbConf.Type, dbConf.FilePath, dbConf.BatchDelaySeconds, dbConf.MaxBatchSize, dbConf.MaxOpenFiles)
	if err != nil {
		return nil, err
	}

	if reflect.DeepEqual(bloomFilterConf, BloomConfig{}) {
		return NewStorageUnit(cache, db)
	}

	bf, err = NewBloomFilter(bloomFilterConf)
	if err != nil {
		return nil, err
	}

	return NewStorageUnitWithBloomFilter(cache, db, bf)
}

// NewShardedStorageUnitFromConf creates a new sharded storage unit from a storage unit config
func NewShardedStorageUnitFromConf(cacheConf CacheConfig, dbConf DBConfig, bloomFilterConf BloomConfig, shardId uint32) (*Unit, error) {
	var cache storage.Cacher
	var db storage.Persister
	var bf storage.BloomFilter
	var err error

	defer func() {
		if err != nil && db != nil {
			_ = db.Destroy()
		}
	}()

	cache, err = NewCache(cacheConf.Type, cacheConf.Size, cacheConf.Shards)
	if err != nil {
		return nil, err
	}

	filePath := fmt.Sprintf("%s%d", dbConf.FilePath, shardId)
	db, err = NewDB(dbConf.Type, filePath, dbConf.BatchDelaySeconds, dbConf.MaxBatchSize, dbConf.MaxOpenFiles)
	if err != nil {
		return nil, err
	}

	if reflect.DeepEqual(bloomFilterConf, BloomConfig{}) {
		return NewStorageUnit(cache, db)
	}

	bf, err = NewBloomFilter(bloomFilterConf)
	if err != nil {
		return nil, err
	}

	return NewStorageUnitWithBloomFilter(cache, db, bf)
}

//NewCache creates a new cache from a cache config
//TODO: add a cacher factory or a cacheConfig param instead
func NewCache(cacheType CacheType, size uint32, shards uint32) (storage.Cacher, error) {
	var cacher storage.Cacher
	var err error

	switch cacheType {
	case LRUCache:
		cacher, err = lrucache.NewCache(int(size))
	case FIFOShardedCache:
		cacher, err = fifocache.NewShardedCache(int(size), int(shards))
		if err != nil {
			return nil, err
		}
		// add other implementations if required
	default:
		return nil, storage.ErrNotSupportedCacheType
	}

	if err != nil {
		return nil, err
	}

	return cacher, nil
}

// NewDB creates a new database from database config
func NewDB(dbType DBType, path string, batchDelaySeconds int, maxBatchSize int, maxOpenFiles int) (storage.Persister, error) {
	var db storage.Persister
	var err error

	switch dbType {
	case LvlDB:
		db, err = leveldb.NewDB(path, batchDelaySeconds, maxBatchSize, maxOpenFiles)
	case LvlDbSerial:
		db, err = leveldb.NewSerialDB(path, batchDelaySeconds, maxBatchSize, maxOpenFiles)
	case BadgerDB:
		db, err = badgerdb.NewDB(path, batchDelaySeconds, maxBatchSize)
	case BoltDB:
		db, err = boltdb.NewDB(path, batchDelaySeconds, maxBatchSize)
	default:
		return nil, storage.ErrNotSupportedDBType
	}

	if err != nil {
		return nil, err
	}

	return db, nil
}

// NewBloomFilter creates a new bloom filter from bloom filter config
func NewBloomFilter(conf BloomConfig) (storage.BloomFilter, error) {
	var bf storage.BloomFilter
	var err error
	var hashers []hashing.Hasher

	for _, hashString := range conf.HashFunc {
		hasher, err := hashString.NewHasher()
		if err == nil {
			hashers = append(hashers, hasher)
		} else {
			return nil, err
		}
	}

	bf, err = bloom.NewFilter(conf.Size, hashers)
	if err != nil {
		return nil, err
	}

	return bf, nil
}

// NewHasher will return a hasher implementation form the string HasherType
func (h HasherType) NewHasher() (hashing.Hasher, error) {
	switch h {
	case Keccak:
		return keccak.Keccak{}, nil
	case Blake2b:
		return blake2b.Blake2b{}, nil
	case Fnv:
		return fnv.Fnv{}, nil
	default:
		return nil, storage.ErrNotSupportedHashType
	}
}
