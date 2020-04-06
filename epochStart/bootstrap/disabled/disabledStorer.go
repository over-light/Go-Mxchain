package disabled

import (
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/memorydb"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
)

const defaultMemDBSize = 10000
const defaultNumShards = 1

// CreateMemUnit creates an in-memory storer unit using maps
func CreateMemUnit() storage.Storer {
	cache, err := storageUnit.NewCache(storageUnit.LRUCache, defaultMemDBSize, defaultNumShards)
	if err != nil {
		return nil
	}

	unit, err := storageUnit.NewStorageUnit(cache, memorydb.New())
	if err != nil {
		return nil
	}

	return unit
}
