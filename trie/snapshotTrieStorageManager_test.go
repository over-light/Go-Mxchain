package trie

import (
	"strings"
	"testing"

	"github.com/ElrondNetwork/elrond-go/testscommon/trie"
	"github.com/stretchr/testify/assert"
)

func TestNewSnapshotTrieStorageManagerInvalidStorerType(t *testing.T) {
	_, trieStorage := newEmptyTrie()

	stsm, err := newSnapshotTrieStorageManager(trieStorage)
	assert.Nil(t, stsm)
	assert.True(t, strings.Contains(err.Error(), "invalid storer type"))
}

func TestNewSnapshotTrieStorageManager(t *testing.T) {
	_, trieStorage := newEmptyTrie()
	trieStorage.mainStorer = &trie.SnapshotPruningStorerStub{}
	stsm, err := newSnapshotTrieStorageManager(trieStorage)
	assert.Nil(t, err)
	assert.NotNil(t, stsm)
}

func TestNewSnapshotTrieStorageManager_GetFromOldEpochsWithoutCache(t *testing.T) {
	_, trieStorage := newEmptyTrie()
	getFromOldEpochsWithoutCacheCalled := false
	trieStorage.mainStorer = &trie.SnapshotPruningStorerStub{
		GetFromOldEpochsWithoutCacheCalled: func(_ []byte) ([]byte, error) {
			getFromOldEpochsWithoutCacheCalled = true
			return nil, nil
		},
	}
	stsm, _ := newSnapshotTrieStorageManager(trieStorage)

	_, _ = stsm.Get([]byte("key"))
	assert.True(t, getFromOldEpochsWithoutCacheCalled)
}

func TestNewSnapshotTrieStorageManager_PutWithoutCache(t *testing.T) {
	_, trieStorage := newEmptyTrie()
	putWithoutCacheCalled := false
	trieStorage.mainStorer = &trie.SnapshotPruningStorerStub{
		PutWithoutCacheCalled: func(_, _ []byte) error {
			putWithoutCacheCalled = true
			return nil
		},
	}
	stsm, _ := newSnapshotTrieStorageManager(trieStorage)

	_ = stsm.Put([]byte("key"), []byte("data"))
	assert.True(t, putWithoutCacheCalled)
}