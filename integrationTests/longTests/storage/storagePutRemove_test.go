package storage

import (
	"crypto/rand"
	"io/ioutil"
	"testing"
	"time"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/leveldb"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
)

var log = logger.GetOrCreate("integrationTests/longTests/storage")

func TestPutRemove(t *testing.T) {
	t.Skip("this is a long test")

	cache, _ := storageUnit.NewCache(storageUnit.LRUCache, 5000, 16, 0)
	dir, _ := ioutil.TempDir("", "leveldb_temp")
	log.Info("opened in", "directory", dir)
	lvdb1, err := leveldb.NewDB(dir, 2, 1000, 10)

	defer func() {
		_ = lvdb1.Close()
	}()

	store, err := storageUnit.NewStorageUnit(cache, lvdb1)
	log.LogIfError(err)

	numPuts := 800
	valuePayloadSize := 2048
	rmv := make(map[int][][]byte)
	max := int64(0)
	iterations := 0
	go func() {
		for {
			time.Sleep(time.Second)
			log.Info("Operation stats", "max", time.Duration(max), "iterations", iterations)
		}
	}()

	for i := 0; i < 100000; i++ {
		values := generateValues(numPuts, valuePayloadSize)

		start := time.Now().UnixNano()
		putValues(store, values, rmv, i)
		removeOld(store, rmv, i)
		stop := time.Now().UnixNano()

		crt := stop - start
		if crt > max {
			max = crt
		}
		iterations++
	}
}

func generateValues(numPuts int, valuesPayloadSize int) map[string][]byte {
	m := make(map[string][]byte)
	for i := 0; i < numPuts; i++ {
		hash := make([]byte, 32)
		_, _ = rand.Read(hash)

		value := make([]byte, valuesPayloadSize)
		_, _ = rand.Read(value)

		m[string(hash)] = value
	}

	return m
}

func putValues(store storage.Storer, values map[string][]byte, rmv map[int][][]byte, idx int) {
	hashes := make([][]byte, 0, len(rmv))
	for key, val := range values {
		hashes = append(hashes, []byte(key))
		err := store.Put([]byte(key), val)
		log.LogIfError(err)
	}

	rmv[idx] = hashes
}

func removeOld(store storage.Storer, rmv map[int][][]byte, idx int) {
	hashes, found := rmv[idx-2]
	if !found {
		return
	}

	for _, hash := range hashes {
		err := store.Remove(hash)
		log.LogIfError(err)
	}

	delete(rmv, idx-2)
}
