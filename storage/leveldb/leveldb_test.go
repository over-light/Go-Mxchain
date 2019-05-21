package leveldb_test

import (
	"io/ioutil"
	"testing"

	"github.com/ElrondNetwork/elrond-go-sandbox/storage/leveldb"
	"github.com/stretchr/testify/assert"
)

func createLevelDb(t *testing.T) (p *leveldb.DB) {
	dir, err := ioutil.TempDir("", "leveldb_temp")
	lvdb, err := leveldb.NewDB(dir)

	assert.Nil(t, err, "Failed creating leveldb database file")
	return lvdb
}

func TestInitNoError(t *testing.T) {
	ldb := createLevelDb(t)

	err := ldb.Init()

	assert.Nil(t, err, "error initializing db")
}

func TestPutNoError(t *testing.T) {
	key, val := []byte("key"), []byte("value")
	ldb := createLevelDb(t)

	err := ldb.Put(key, val)

	assert.Nil(t, err, "error saving in db")
}

func TestGetPresent(t *testing.T) {
	key, val := []byte("key1"), []byte("value1")
	ldb := createLevelDb(t)

	err := ldb.Put(key, val)

	assert.Nil(t, err, "error saving in db")

	v, err := ldb.Get(key)

	assert.Nil(t, err, "error not expected, but got %s", err)
	assert.Equalf(t, v, val, "read:%s but expected: %s", v, val)
}

func TestGetNotPresent(t *testing.T) {
	key := []byte("key2")
	ldb := createLevelDb(t)

	v, err := ldb.Get(key)

	assert.NotNil(t, err, "error expected but got nil, value %s", v)
}

func TestHasPresent(t *testing.T) {
	key, val := []byte("key3"), []byte("value3")
	ldb := createLevelDb(t)

	err := ldb.Put(key, val)

	assert.Nil(t, err, "error saving in db")

	err = ldb.Has(key)

	assert.Nil(t, err)
}

func TestHasNotPresent(t *testing.T) {
	key := []byte("key4")
	ldb := createLevelDb(t)

	err := ldb.Has(key)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Key not found")
}

func TestRemovePresent(t *testing.T) {
	key, val := []byte("key5"), []byte("value5")
	ldb := createLevelDb(t)

	err := ldb.Put(key, val)

	assert.Nil(t, err, "error saving in db")

	err = ldb.Remove(key)

	assert.Nil(t, err, "no error expected but got %s", err)

	err = ldb.Has(key)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Key not found")
}

func TestRemoveNotPresent(t *testing.T) {
	key := []byte("key6")
	ldb := createLevelDb(t)

	err := ldb.Remove(key)

	assert.Nil(t, err, "no error expected but got %s", err)
}

func TestClose(t *testing.T) {
	ldb := createLevelDb(t)

	err := ldb.Close()

	assert.Nil(t, err, "no error expected but got %s", err)
}

func TestDestroy(t *testing.T) {
	ldb := createLevelDb(t)

	err := ldb.Destroy()

	assert.Nil(t, err, "no error expected but got %s", err)
}
