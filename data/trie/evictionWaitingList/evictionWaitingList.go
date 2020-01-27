package evictionWaitingList

import (
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/storage"
)

// evictionWaitingList is a structure that caches keys that need to be removed from a certain database.
// If the cache is full, the keys will be stored in the underlying database. Writing at the same key in
// cacher and db will overwrite the previous values. This structure is not concurrent safe.
type evictionWaitingList struct {
	cache       map[string][][]byte
	cacheSize   uint
	db          storage.Persister
	marshalizer marshal.Marshalizer
}

// NewEvictionWaitingList creates a new instance of evictionWaitingList
func NewEvictionWaitingList(size uint, db storage.Persister, marshalizer marshal.Marshalizer) (*evictionWaitingList, error) {
	if size < 1 {
		return nil, data.ErrInvalidCacheSize
	}
	if check.IfNil(db) {
		return nil, data.ErrNilDatabase
	}
	if check.IfNil(marshalizer) {
		return nil, data.ErrNilMarshalizer
	}

	return &evictionWaitingList{
		cache:       make(map[string][][]byte),
		cacheSize:   size,
		db:          db,
		marshalizer: marshalizer,
	}, nil
}

// Put stores the given hashes in the eviction waiting list, in the position given by the root hash
func (ewl *evictionWaitingList) Put(rootHash []byte, hashes [][]byte) error {
	if uint(len(ewl.cache)) < ewl.cacheSize {
		ewl.cache[string(rootHash)] = hashes
		return nil
	}

	marshalizedHashes, err := ewl.marshalizer.Marshal(hashes)
	if err != nil {
		return err
	}

	err = ewl.db.Put(rootHash, marshalizedHashes)
	if err != nil {
		return err
	}

	return nil
}

// Evict returns and removes from the waiting list all the hashes from the position given by the root hash
func (ewl *evictionWaitingList) Evict(rootHash []byte) ([][]byte, error) {
	hashes, ok := ewl.cache[string(rootHash)]
	if ok {
		delete(ewl.cache, string(rootHash))
		return hashes, nil
	}

	marshalizedHashes, err := ewl.db.Get(rootHash)
	if err != nil {
		return nil, err
	}

	err = ewl.marshalizer.Unmarshal(&hashes, marshalizedHashes)
	if err != nil {
		return nil, err
	}

	err = ewl.db.Remove(rootHash)
	if err != nil {
		return nil, err
	}

	return hashes, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (ewl *evictionWaitingList) IsInterfaceNil() bool {
	return ewl == nil
}

// GetSize returns the size of the cache
func (ewl *evictionWaitingList) GetSize() uint {
	return ewl.cacheSize
}
