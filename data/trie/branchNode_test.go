package trie

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/mock"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/storage/lrucache"
	"github.com/ElrondNetwork/elrond-go/storage/memorydb"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/stretchr/testify/assert"
)

func getTestMarshAndHasher() (marshal.Marshalizer, hashing.Hasher) {
	marsh := &marshal.GogoProtoMarshalizer{}
	hasher := &mock.KeccakMock{}
	return marsh, hasher
}

func getBnAndCollapsedBn(marshalizer marshal.Marshalizer, hasher hashing.Hasher) (*branchNode, *branchNode) {
	var children [nrOfChildren]node
	EncodedChildren := make([][]byte, nrOfChildren)

	children[2], _ = newLeafNode([]byte("dog"), []byte("dog"), marshalizer, hasher)
	children[6], _ = newLeafNode([]byte("doe"), []byte("doe"), marshalizer, hasher)
	children[13], _ = newLeafNode([]byte("doge"), []byte("doge"), marshalizer, hasher)
	bn, _ := newBranchNode(marshalizer, hasher)
	bn.children = children

	EncodedChildren[2], _ = encodeNodeAndGetHash(children[2])
	EncodedChildren[6], _ = encodeNodeAndGetHash(children[6])
	EncodedChildren[13], _ = encodeNodeAndGetHash(children[13])
	collapsedBn, _ := newBranchNode(marshalizer, hasher)
	collapsedBn.EncodedChildren = EncodedChildren

	return bn, collapsedBn
}

func newEmptyTrie() (*patriciaMerkleTrie, *trieStorageManager, *mock.EvictionWaitingList) {
	db := memorydb.New()
	marsh, hsh := getTestMarshAndHasher()
	evictionWaitListSize := uint(100)
	evictionWaitList, _ := mock.NewEvictionWaitingList(evictionWaitListSize, mock.NewMemDbMock(), marsh)

	// TODO change this initialization of the persister  (and everywhere in this package)
	// by using a persister factory
	tempDir, _ := ioutil.TempDir("", "leveldb_temp")
	cfg := config.DBConfig{
		FilePath:          tempDir,
		Type:              string(storageUnit.LvlDBSerial),
		BatchDelaySeconds: 1,
		MaxBatchSize:      1,
		MaxOpenFiles:      10,
	}
	generalCfg := config.TrieStorageManagerConfig{
		PruningBufferLen:   1000,
		SnapshotsBufferLen: 10,
		MaxSnapshots:       2,
	}

	trieStorage, _ := NewTrieStorageManager(db, marsh, hsh, cfg, evictionWaitList, generalCfg)
	tr := &patriciaMerkleTrie{
		trieStorage:          trieStorage,
		marshalizer:          marsh,
		hasher:               hsh,
		oldHashes:            make([][]byte, 0),
		oldRoot:              make([]byte, 0),
		maxTrieLevelInMemory: 5,
	}

	return tr, trieStorage, evictionWaitList
}

func initTrie() *patriciaMerkleTrie {
	tr, _, _ := newEmptyTrie()
	_ = tr.Update([]byte("doe"), []byte("reindeer"))
	_ = tr.Update([]byte("dog"), []byte("puppy"))
	_ = tr.Update([]byte("ddog"), []byte("cat"))

	return tr
}

func getEncodedTrieNodesAndHashes(tr data.Trie) ([][]byte, [][]byte) {
	it, _ := NewIterator(tr)
	encNode, _ := it.MarshalizedNode()

	nodes := make([][]byte, 0)
	nodes = append(nodes, encNode)

	hashes := make([][]byte, 0)
	hash, _ := it.GetHash()
	hashes = append(hashes, hash)

	for it.HasNext() {
		_ = it.Next()
		encNode, _ = it.MarshalizedNode()

		nodes = append(nodes, encNode)
		hash, _ = it.GetHash()
		hashes = append(hashes, hash)
	}

	return nodes, hashes
}

func TestBranchNode_getHash(t *testing.T) {
	t.Parallel()

	bn := &branchNode{baseNode: &baseNode{hash: []byte("test hash")}}
	assert.Equal(t, bn.hash, bn.getHash())
}

func TestBranchNode_isDirty(t *testing.T) {
	t.Parallel()

	bn := &branchNode{baseNode: &baseNode{dirty: true}}
	assert.Equal(t, true, bn.isDirty())

	bn = &branchNode{baseNode: &baseNode{dirty: false}}
	assert.Equal(t, false, bn.isDirty())
}

func TestBranchNode_getCollapsed(t *testing.T) {
	t.Parallel()

	bn, collapsedBn := getBnAndCollapsedBn(getTestMarshAndHasher())
	collapsedBn.dirty = true

	collapsed, err := bn.getCollapsed()
	assert.Nil(t, err)
	assert.Equal(t, collapsedBn, collapsed)
}

func TestBranchNode_getCollapsedEmptyNode(t *testing.T) {
	t.Parallel()

	bn := emptyDirtyBranchNode()

	collapsed, err := bn.getCollapsed()
	assert.True(t, errors.Is(err, ErrEmptyBranchNode))
	assert.Nil(t, collapsed)
}

func TestBranchNode_getCollapsedNilNode(t *testing.T) {
	t.Parallel()

	var bn *branchNode

	collapsed, err := bn.getCollapsed()
	assert.True(t, errors.Is(err, ErrNilBranchNode))
	assert.Nil(t, collapsed)
}

func TestBranchNode_getCollapsedCollapsedNode(t *testing.T) {
	t.Parallel()

	_, collapsedBn := getBnAndCollapsedBn(getTestMarshAndHasher())

	collapsed, err := collapsedBn.getCollapsed()
	assert.Nil(t, err)
	assert.Equal(t, collapsedBn, collapsed)
}

func TestBranchNode_setHash(t *testing.T) {
	t.Parallel()

	bn, collapsedBn := getBnAndCollapsedBn(getTestMarshAndHasher())
	hash, _ := encodeNodeAndGetHash(collapsedBn)

	err := bn.setHash()
	assert.Nil(t, err)
	assert.Equal(t, hash, bn.hash)
}

func TestBranchNode_setRootHash(t *testing.T) {
	t.Parallel()

	cfg := config.DBConfig{}
	db := mock.NewMemDbMock()
	marsh, hsh := getTestMarshAndHasher()
	trieStorage1, _ := NewTrieStorageManager(db, marsh, hsh, cfg, &mock.EvictionWaitingList{}, config.TrieStorageManagerConfig{})
	trieStorage2, _ := NewTrieStorageManager(db, marsh, hsh, cfg, &mock.EvictionWaitingList{}, config.TrieStorageManagerConfig{})
	maxTrieLevelInMemory := uint(5)

	tr1, _ := NewTrie(trieStorage1, marsh, hsh, maxTrieLevelInMemory)
	tr2, _ := NewTrie(trieStorage2, marsh, hsh, maxTrieLevelInMemory)

	maxIterations := 10000
	for i := 0; i < maxIterations; i++ {
		val := hsh.Compute(string(i))
		_ = tr1.Update(val, val)
		_ = tr2.Update(val, val)
	}

	err := tr1.root.setRootHash()
	_ = tr2.root.setHash()
	assert.Nil(t, err)
	assert.Equal(t, tr1.root.getHash(), tr2.root.getHash())
}

func TestBranchNode_setRootHashCollapsedNode(t *testing.T) {
	t.Parallel()

	_, collapsedBn := getBnAndCollapsedBn(getTestMarshAndHasher())
	hash, _ := encodeNodeAndGetHash(collapsedBn)

	err := collapsedBn.setRootHash()
	assert.Nil(t, err)
	assert.Equal(t, hash, collapsedBn.hash)
}

func TestBranchNode_setHashEmptyNode(t *testing.T) {
	t.Parallel()

	bn := emptyDirtyBranchNode()

	err := bn.setHash()
	assert.True(t, errors.Is(err, ErrEmptyBranchNode))
	assert.Nil(t, bn.hash)
}

func TestBranchNode_setHashNilNode(t *testing.T) {
	t.Parallel()

	var bn *branchNode

	err := bn.setHash()
	assert.True(t, errors.Is(err, ErrNilBranchNode))
	assert.Nil(t, bn)
}

func TestBranchNode_setHashCollapsedNode(t *testing.T) {
	t.Parallel()

	_, collapsedBn := getBnAndCollapsedBn(getTestMarshAndHasher())
	hash, _ := encodeNodeAndGetHash(collapsedBn)

	err := collapsedBn.setHash()
	assert.Nil(t, err)
	assert.Equal(t, hash, collapsedBn.hash)
}

func TestBranchNode_setGivenHash(t *testing.T) {
	t.Parallel()

	bn := &branchNode{baseNode: &baseNode{}}
	expectedHash := []byte("node hash")

	bn.setGivenHash(expectedHash)
	assert.Equal(t, expectedHash, bn.hash)
}

func TestBranchNode_hashChildren(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())

	for i := range bn.children {
		if bn.children[i] != nil {
			assert.Nil(t, bn.children[i].getHash())
		}
	}
	err := bn.hashChildren()
	assert.Nil(t, err)

	for i := range bn.children {
		if bn.children[i] != nil {
			childHash, _ := encodeNodeAndGetHash(bn.children[i])
			assert.Equal(t, childHash, bn.children[i].getHash())
		}
	}
}

func TestBranchNode_hashChildrenEmptyNode(t *testing.T) {
	t.Parallel()

	bn := emptyDirtyBranchNode()

	err := bn.hashChildren()
	assert.True(t, errors.Is(err, ErrEmptyBranchNode))
}

func TestBranchNode_hashChildrenNilNode(t *testing.T) {
	t.Parallel()

	var bn *branchNode

	err := bn.hashChildren()
	assert.True(t, errors.Is(err, ErrNilBranchNode))
}

func TestBranchNode_hashChildrenCollapsedNode(t *testing.T) {
	t.Parallel()

	_, collapsedBn := getBnAndCollapsedBn(getTestMarshAndHasher())

	err := collapsedBn.hashChildren()
	assert.Nil(t, err)

	_, collapsedBn2 := getBnAndCollapsedBn(getTestMarshAndHasher())
	assert.Equal(t, collapsedBn2, collapsedBn)
}

func TestBranchNode_hashNode(t *testing.T) {
	t.Parallel()

	_, collapsedBn := getBnAndCollapsedBn(getTestMarshAndHasher())
	expectedHash, _ := encodeNodeAndGetHash(collapsedBn)

	hash, err := collapsedBn.hashNode()
	assert.Nil(t, err)
	assert.Equal(t, expectedHash, hash)
}

func TestBranchNode_hashNodeEmptyNode(t *testing.T) {
	t.Parallel()

	bn := emptyDirtyBranchNode()

	hash, err := bn.hashNode()
	assert.True(t, errors.Is(err, ErrEmptyBranchNode))
	assert.Nil(t, hash)
}

func TestBranchNode_hashNodeNilNode(t *testing.T) {
	t.Parallel()

	var bn *branchNode

	hash, err := bn.hashNode()
	assert.True(t, errors.Is(err, ErrNilBranchNode))
	assert.Nil(t, hash)
}

func TestBranchNode_commit(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	marsh, hasher := getTestMarshAndHasher()
	bn, collapsedBn := getBnAndCollapsedBn(marsh, hasher)

	hash, _ := encodeNodeAndGetHash(collapsedBn)
	_ = bn.setHash()

	err := bn.commit(false, 0, 5, db, db)
	assert.Nil(t, err)

	encNode, _ := db.Get(hash)
	node, _ := decodeNode(encNode, marsh, hasher)
	h1, _ := encodeNodeAndGetHash(collapsedBn)
	h2, _ := encodeNodeAndGetHash(node)
	assert.Equal(t, h1, h2)
}

func TestBranchNode_commitEmptyNode(t *testing.T) {
	t.Parallel()

	bn := emptyDirtyBranchNode()

	err := bn.commit(false, 0, 5, nil, nil)
	assert.True(t, errors.Is(err, ErrEmptyBranchNode))
}

func TestBranchNode_commitNilNode(t *testing.T) {
	t.Parallel()

	var bn *branchNode

	err := bn.commit(false, 0, 5, nil, nil)
	assert.True(t, errors.Is(err, ErrNilBranchNode))
}

func TestBranchNode_getEncodedNode(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())

	expectedEncodedNode, _ := bn.marsh.Marshal(bn)
	expectedEncodedNode = append(expectedEncodedNode, branch)

	encNode, err := bn.getEncodedNode()
	assert.Nil(t, err)
	assert.Equal(t, expectedEncodedNode, encNode)
}

func TestBranchNode_getEncodedNodeEmpty(t *testing.T) {
	t.Parallel()

	bn := emptyDirtyBranchNode()

	encNode, err := bn.getEncodedNode()
	assert.True(t, errors.Is(err, ErrEmptyBranchNode))
	assert.Nil(t, encNode)
}

func TestBranchNode_getEncodedNodeNil(t *testing.T) {
	t.Parallel()

	var bn *branchNode

	encNode, err := bn.getEncodedNode()
	assert.True(t, errors.Is(err, ErrNilBranchNode))
	assert.Nil(t, encNode)
}

func TestBranchNode_resolveCollapsed(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	bn, collapsedBn := getBnAndCollapsedBn(getTestMarshAndHasher())
	childPos := byte(2)

	_ = bn.setHash()
	_ = bn.commit(false, 0, 5, db, db)
	resolved, _ := newLeafNode([]byte("dog"), []byte("dog"), bn.marsh, bn.hasher)
	resolved.dirty = false
	resolved.hash = bn.EncodedChildren[childPos]

	err := collapsedBn.resolveCollapsed(childPos, db)
	assert.Nil(t, err)
	assert.Equal(t, resolved, collapsedBn.children[childPos])
}

func TestBranchNode_resolveCollapsedEmptyNode(t *testing.T) {
	t.Parallel()

	bn := emptyDirtyBranchNode()

	err := bn.resolveCollapsed(2, nil)
	assert.True(t, errors.Is(err, ErrEmptyBranchNode))
}

func TestBranchNode_resolveCollapsedENilNode(t *testing.T) {
	t.Parallel()

	var bn *branchNode

	err := bn.resolveCollapsed(2, nil)
	assert.True(t, errors.Is(err, ErrNilBranchNode))
}

func TestBranchNode_resolveCollapsedPosOutOfRange(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())

	err := bn.resolveCollapsed(17, nil)
	assert.Equal(t, ErrChildPosOutOfRange, err)
}

func TestBranchNode_isCollapsed(t *testing.T) {
	t.Parallel()

	bn, collapsedBn := getBnAndCollapsedBn(getTestMarshAndHasher())

	assert.True(t, collapsedBn.isCollapsed())
	assert.False(t, bn.isCollapsed())

	collapsedBn.children[2], _ = newLeafNode([]byte("dog"), []byte("dog"), bn.marsh, bn.hasher)
	assert.False(t, collapsedBn.isCollapsed())
}

func TestBranchNode_tryGet(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	childPos := byte(2)
	key := append([]byte{childPos}, []byte("dog")...)

	val, err := bn.tryGet(key, nil)
	assert.Equal(t, []byte("dog"), val)
	assert.Nil(t, err)
}

func TestBranchNode_tryGetEmptyKey(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	var key []byte

	val, err := bn.tryGet(key, nil)
	assert.Nil(t, err)
	assert.Nil(t, val)
}

func TestBranchNode_tryGetChildPosOutOfRange(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	key := []byte("dog")

	val, err := bn.tryGet(key, nil)
	assert.Equal(t, ErrChildPosOutOfRange, err)
	assert.Nil(t, val)
}

func TestBranchNode_tryGetNilChild(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	nilChildKey := []byte{3}

	val, err := bn.tryGet(nilChildKey, nil)
	assert.Nil(t, err)
	assert.Nil(t, val)
}

func TestBranchNode_tryGetCollapsedNode(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	bn, collapsedBn := getBnAndCollapsedBn(getTestMarshAndHasher())

	_ = bn.setHash()
	_ = bn.commit(false, 0, 5, db, db)

	childPos := byte(2)
	key := append([]byte{childPos}, []byte("dog")...)

	val, err := collapsedBn.tryGet(key, db)
	assert.Equal(t, []byte("dog"), val)
	assert.Nil(t, err)
}

func TestBranchNode_tryGetEmptyNode(t *testing.T) {
	t.Parallel()

	bn := emptyDirtyBranchNode()
	childPos := byte(2)
	key := append([]byte{childPos}, []byte("dog")...)

	val, err := bn.tryGet(key, nil)
	assert.True(t, errors.Is(err, ErrEmptyBranchNode))
	assert.Nil(t, val)
}

func TestBranchNode_tryGetNilNode(t *testing.T) {
	t.Parallel()

	var bn *branchNode
	childPos := byte(2)
	key := append([]byte{childPos}, []byte("dog")...)

	val, err := bn.tryGet(key, nil)
	assert.True(t, errors.Is(err, ErrNilBranchNode))
	assert.Nil(t, val)
}

func TestBranchNode_getNext(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	nextNode, _ := newLeafNode([]byte("dog"), []byte("dog"), bn.marsh, bn.hasher)
	childPos := byte(2)
	key := append([]byte{childPos}, []byte("dog")...)

	node, key, err := bn.getNext(key, nil)

	h1, _ := encodeNodeAndGetHash(nextNode)
	h2, _ := encodeNodeAndGetHash(node)
	assert.Equal(t, h1, h2)
	assert.Equal(t, []byte("dog"), key)
	assert.Nil(t, err)
}

func TestBranchNode_getNextWrongKey(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	key := []byte("dog")

	node, key, err := bn.getNext(key, nil)
	assert.Nil(t, node)
	assert.Nil(t, key)
	assert.Equal(t, ErrChildPosOutOfRange, err)
}

func TestBranchNode_getNextNilChild(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	nilChildPos := byte(4)
	key := append([]byte{nilChildPos}, []byte("dog")...)

	node, key, err := bn.getNext(key, nil)
	assert.Nil(t, node)
	assert.Nil(t, key)
	assert.Equal(t, ErrNodeNotFound, err)
}

func TestBranchNode_insert(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	nodeKey := []byte{0, 2, 3}
	node, _ := newLeafNode(nodeKey, []byte("dogs"), bn.marsh, bn.hasher)

	dirty, newBn, _, err := bn.insert(node, nil)
	nodeKeyRemainder := nodeKey[1:]

	bn.children[0], _ = newLeafNode(nodeKeyRemainder, []byte("dogs"), bn.marsh, bn.hasher)
	assert.True(t, dirty)
	assert.Nil(t, err)
	assert.Equal(t, bn, newBn)
}

func TestBranchNode_insertEmptyKey(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	node, _ := newLeafNode([]byte{}, []byte("dogs"), bn.marsh, bn.hasher)

	dirty, newBn, _, err := bn.insert(node, nil)
	assert.False(t, dirty)
	assert.Equal(t, ErrValueTooShort, err)
	assert.Nil(t, newBn)
}

func TestBranchNode_insertChildPosOutOfRange(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	node, _ := newLeafNode([]byte("dog"), []byte("dogs"), bn.marsh, bn.hasher)

	dirty, newBn, _, err := bn.insert(node, nil)
	assert.False(t, dirty)
	assert.Equal(t, ErrChildPosOutOfRange, err)
	assert.Nil(t, newBn)
}

func TestBranchNode_insertCollapsedNode(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	bn, collapsedBn := getBnAndCollapsedBn(getTestMarshAndHasher())
	childPos := byte(2)
	key := append([]byte{childPos}, []byte("dog")...)
	node, _ := newLeafNode(key, []byte("dogs"), bn.marsh, bn.hasher)

	_ = bn.setHash()
	_ = bn.commit(false, 0, 5, db, db)

	dirty, newBn, _, err := collapsedBn.insert(node, db)
	assert.True(t, dirty)
	assert.Nil(t, err)

	val, _ := newBn.tryGet(key, db)
	assert.Equal(t, []byte("dogs"), val)
}

func TestBranchNode_insertInStoredBnOnExistingPos(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	childPos := byte(2)
	key := append([]byte{childPos}, []byte("dog")...)
	node, _ := newLeafNode(key, []byte("dogs"), bn.marsh, bn.hasher)

	_ = bn.commit(false, 0, 5, db, db)
	bnHash := bn.getHash()
	ln, _, _ := bn.getNext(key, db)
	lnHash := ln.getHash()
	expectedHashes := [][]byte{lnHash, bnHash}

	dirty, _, oldHashes, err := bn.insert(node, db)
	assert.True(t, dirty)
	assert.Nil(t, err)
	assert.Equal(t, expectedHashes, oldHashes)
}

func TestBranchNode_insertInStoredBnOnNilPos(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	nilChildPos := byte(11)
	key := append([]byte{nilChildPos}, []byte("dog")...)
	node, _ := newLeafNode(key, []byte("dogs"), bn.marsh, bn.hasher)

	_ = bn.commit(false, 0, 5, db, db)
	bnHash := bn.getHash()
	expectedHashes := [][]byte{bnHash}

	dirty, _, oldHashes, err := bn.insert(node, db)
	assert.True(t, dirty)
	assert.Nil(t, err)
	assert.Equal(t, expectedHashes, oldHashes)
}

func TestBranchNode_insertInDirtyBnOnNilPos(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	nilChildPos := byte(11)
	key := append([]byte{nilChildPos}, []byte("dog")...)
	node, _ := newLeafNode(key, []byte("dogs"), bn.marsh, bn.hasher)

	dirty, _, oldHashes, err := bn.insert(node, nil)
	assert.True(t, dirty)
	assert.Nil(t, err)
	assert.Equal(t, [][]byte{}, oldHashes)
}

func TestBranchNode_insertInDirtyBnOnExistingPos(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	childPos := byte(2)
	key := append([]byte{childPos}, []byte("dog")...)
	node, _ := newLeafNode(key, []byte("dogs"), bn.marsh, bn.hasher)

	dirty, _, oldHashes, err := bn.insert(node, nil)
	assert.True(t, dirty)
	assert.Nil(t, err)
	assert.Equal(t, [][]byte{}, oldHashes)
}

func TestBranchNode_insertInNilNode(t *testing.T) {
	t.Parallel()

	var bn *branchNode

	dirty, newBn, _, err := bn.insert(&leafNode{}, nil)
	assert.False(t, dirty)
	assert.True(t, errors.Is(err, ErrNilBranchNode))
	assert.Nil(t, newBn)
}

func TestBranchNode_delete(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	var children [nrOfChildren]node
	children[6], _ = newLeafNode([]byte("doe"), []byte("doe"), bn.marsh, bn.hasher)
	children[13], _ = newLeafNode([]byte("doge"), []byte("doge"), bn.marsh, bn.hasher)
	expectedBn, _ := newBranchNode(bn.marsh, bn.hasher)
	expectedBn.children = children

	childPos := byte(2)
	key := append([]byte{childPos}, []byte("dog")...)

	dirty, newBn, _, err := bn.delete(key, nil)
	assert.True(t, dirty)
	assert.Nil(t, err)

	_ = expectedBn.setHash()
	_ = newBn.setHash()
	assert.Equal(t, expectedBn.getHash(), newBn.getHash())
}

func TestBranchNode_deleteFromStoredBn(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	childPos := byte(2)
	lnKey := append([]byte{childPos}, []byte("dog")...)

	_ = bn.commit(false, 0, 5, db, db)
	bnHash := bn.getHash()
	ln, _, _ := bn.getNext(lnKey, db)
	lnHash := ln.getHash()
	expectedHashes := [][]byte{lnHash, bnHash}

	dirty, _, oldHashes, err := bn.delete(lnKey, db)
	assert.True(t, dirty)
	assert.Nil(t, err)
	assert.Equal(t, expectedHashes, oldHashes)
}

func TestBranchNode_deleteFromDirtyBn(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	childPos := byte(2)
	lnKey := append([]byte{childPos}, []byte("dog")...)

	dirty, _, oldHashes, err := bn.delete(lnKey, nil)
	assert.True(t, dirty)
	assert.Nil(t, err)
	assert.Equal(t, [][]byte{}, oldHashes)
}

func TestBranchNode_deleteEmptyNode(t *testing.T) {
	t.Parallel()

	bn := emptyDirtyBranchNode()
	childPos := byte(2)
	key := append([]byte{childPos}, []byte("dog")...)

	dirty, newBn, _, err := bn.delete(key, nil)
	assert.False(t, dirty)
	assert.True(t, errors.Is(err, ErrEmptyBranchNode))
	assert.Nil(t, newBn)
}

func TestBranchNode_deleteNilNode(t *testing.T) {
	t.Parallel()

	var bn *branchNode
	childPos := byte(2)
	key := append([]byte{childPos}, []byte("dog")...)

	dirty, newBn, _, err := bn.delete(key, nil)
	assert.False(t, dirty)
	assert.True(t, errors.Is(err, ErrNilBranchNode))
	assert.Nil(t, newBn)
}

func TestBranchNode_deleteNonexistentNodeFromChild(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())

	childPos := byte(2)
	key := append([]byte{childPos}, []byte("butterfly")...)

	dirty, newBn, _, err := bn.delete(key, nil)
	assert.False(t, dirty)
	assert.Nil(t, err)
	assert.Equal(t, bn, newBn)
}

func TestBranchNode_deleteEmptykey(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())

	dirty, newBn, _, err := bn.delete([]byte{}, nil)
	assert.False(t, dirty)
	assert.Equal(t, ErrValueTooShort, err)
	assert.Nil(t, newBn)
}

func TestBranchNode_deleteCollapsedNode(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	bn, collapsedBn := getBnAndCollapsedBn(getTestMarshAndHasher())
	_ = bn.setHash()
	_ = bn.commit(false, 0, 5, db, db)

	childPos := byte(2)
	key := append([]byte{childPos}, []byte("dog")...)

	dirty, newBn, _, err := collapsedBn.delete(key, db)
	assert.True(t, dirty)
	assert.Nil(t, err)

	val, err := newBn.tryGet(key, db)
	assert.Nil(t, val)
	assert.Nil(t, err)
}

func TestBranchNode_deleteAndReduceBn(t *testing.T) {
	t.Parallel()

	bn, _ := newBranchNode(getTestMarshAndHasher())
	var children [nrOfChildren]node
	firstChildPos := byte(2)
	secondChildPos := byte(6)
	children[firstChildPos], _ = newLeafNode([]byte("dog"), []byte("dog"), bn.marsh, bn.hasher)
	children[secondChildPos], _ = newLeafNode([]byte("doe"), []byte("doe"), bn.marsh, bn.hasher)
	bn.children = children

	key := append([]byte{firstChildPos}, []byte("dog")...)
	ln, _ := newLeafNode(key, []byte("dog"), bn.marsh, bn.hasher)

	key = append([]byte{secondChildPos}, []byte("doe")...)
	dirty, newBn, _, err := bn.delete(key, nil)
	assert.True(t, dirty)
	assert.Nil(t, err)
	assert.Equal(t, ln, newBn)
}

func TestBranchNode_reduceNode(t *testing.T) {
	t.Parallel()

	bn, _ := newBranchNode(getTestMarshAndHasher())
	var children [nrOfChildren]node
	childPos := byte(2)
	children[childPos], _ = newLeafNode([]byte("dog"), []byte("dog"), bn.marsh, bn.hasher)
	bn.children = children

	key := append([]byte{childPos}, []byte("dog")...)
	ln, _ := newLeafNode(key, []byte("dog"), bn.marsh, bn.hasher)

	node, err := bn.children[childPos].reduceNode(int(childPos))
	assert.Equal(t, ln, node)
	assert.Nil(t, err)
}

func TestBranchNode_getChildPosition(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	nr, pos := getChildPosition(bn)
	assert.Equal(t, 3, nr)
	assert.Equal(t, 13, pos)
}

func TestBranchNode_clone(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	clone := bn.clone()
	assert.False(t, bn == clone)
	assert.Equal(t, bn, clone)
}

func TestBranchNode_isEmptyOrNil(t *testing.T) {
	t.Parallel()

	bn := emptyDirtyBranchNode()
	assert.Equal(t, ErrEmptyBranchNode, bn.isEmptyOrNil())

	bn = nil
	assert.Equal(t, ErrNilBranchNode, bn.isEmptyOrNil())
}

func TestReduceBranchNodeWithExtensionNodeChildShouldWork(t *testing.T) {
	t.Parallel()

	tr, _, _ := newEmptyTrie()
	expectedTr, _, _ := newEmptyTrie()

	_ = expectedTr.Update([]byte("dog"), []byte("dog"))
	_ = expectedTr.Update([]byte("doll"), []byte("doll"))

	_ = tr.Update([]byte("dog"), []byte("dog"))
	_ = tr.Update([]byte("doll"), []byte("doll"))
	_ = tr.Update([]byte("wolf"), []byte("wolf"))
	_ = tr.Delete([]byte("wolf"))

	expectedHash, _ := expectedTr.Root()
	hash, _ := tr.Root()
	assert.Equal(t, expectedHash, hash)
}

func TestReduceBranchNodeWithBranchNodeChildShouldWork(t *testing.T) {
	t.Parallel()

	tr, _, _ := newEmptyTrie()
	expectedTr, _, _ := newEmptyTrie()

	_ = expectedTr.Update([]byte("dog"), []byte("puppy"))
	_ = expectedTr.Update([]byte("dogglesworth"), []byte("cat"))

	_ = tr.Update([]byte("doe"), []byte("reindeer"))
	_ = tr.Update([]byte("dog"), []byte("puppy"))
	_ = tr.Update([]byte("dogglesworth"), []byte("cat"))
	_ = tr.Delete([]byte("doe"))

	expectedHash, _ := expectedTr.Root()
	hash, _ := tr.Root()
	assert.Equal(t, expectedHash, hash)
}

func TestReduceBranchNodeWithLeafNodeChildShouldWork(t *testing.T) {
	t.Parallel()

	tr, _, _ := newEmptyTrie()
	expectedTr, _, _ := newEmptyTrie()

	_ = expectedTr.Update([]byte("doe"), []byte("reindeer"))
	_ = expectedTr.Update([]byte("dogglesworth"), []byte("cat"))

	_ = tr.Update([]byte("doe"), []byte("reindeer"))
	_ = tr.Update([]byte("dog"), []byte("puppy"))
	_ = tr.Update([]byte("dogglesworth"), []byte("cat"))
	_ = tr.Delete([]byte("dog"))

	expectedHash, _ := expectedTr.Root()
	hash, _ := tr.Root()
	assert.Equal(t, expectedHash, hash)
}

func TestReduceBranchNodeWithLeafNodeValueShouldWork(t *testing.T) {
	t.Parallel()

	tr, _, _ := newEmptyTrie()
	expectedTr, _, _ := newEmptyTrie()

	_ = expectedTr.Update([]byte("doe"), []byte("reindeer"))
	_ = expectedTr.Update([]byte("dog"), []byte("puppy"))

	_ = tr.Update([]byte("doe"), []byte("reindeer"))
	_ = tr.Update([]byte("dog"), []byte("puppy"))
	_ = tr.Update([]byte("dogglesworth"), []byte("cat"))
	_ = tr.Delete([]byte("dogglesworth"))

	expectedHash, _ := expectedTr.Root()
	hash, _ := tr.Root()

	assert.Equal(t, expectedHash, hash)
}

func TestBranchNode_getChildren(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())

	children, err := bn.getChildren(nil)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(children))
}

func TestBranchNode_getChildrenCollapsedBn(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	bn, collapsedBn := getBnAndCollapsedBn(getTestMarshAndHasher())
	_ = bn.commit(true, 0, 5, db, db)

	children, err := collapsedBn.getChildren(db)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(children))
}

func TestBranchNode_isValid(t *testing.T) {
	t.Parallel()

	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	assert.True(t, bn.isValid())

	bn.children[2] = nil
	bn.children[6] = nil
	assert.False(t, bn.isValid())
}

func TestBranchNode_setDirty(t *testing.T) {
	t.Parallel()

	bn := &branchNode{baseNode: &baseNode{}}
	bn.setDirty(true)

	assert.True(t, bn.dirty)
}

func TestBranchNode_loadChildren(t *testing.T) {
	t.Parallel()

	marsh, hasher := getTestMarshAndHasher()
	tr := initTrie()
	_ = tr.root.setRootHash()
	nodes, _ := getEncodedTrieNodesAndHashes(tr)
	nodesCacher, _ := lrucache.NewCache(100)
	for i := range nodes {
		node, _ := NewInterceptedTrieNode(nodes[i], marsh, hasher)
		nodesCacher.Put(node.hash, node, len(node.EncodedNode()))
	}

	firstChildIndex := 5
	secondChildIndex := 7

	bn := getCollapsedBn(t, tr.root)

	getNode := func(hash []byte) (node, error) {
		cacheData, _ := nodesCacher.Get(hash)
		return trieNode(cacheData)
	}

	missing, _, err := bn.loadChildren(getNode)
	assert.Nil(t, err)
	assert.NotNil(t, bn.children[firstChildIndex])
	assert.NotNil(t, bn.children[secondChildIndex])
	assert.Equal(t, 0, len(missing))
	assert.Equal(t, 6, nodesCacher.Len())
}

func getCollapsedBn(t *testing.T, n node) *branchNode {
	bn, ok := n.(*branchNode)
	assert.True(t, ok)
	for i := 0; i < nrOfChildren; i++ {
		bn.children[i] = nil
	}
	return bn
}

//------- deepClone

func TestBranchNode_deepCloneWithNilHashShouldWork(t *testing.T) {
	t.Parallel()

	bn := &branchNode{baseNode: &baseNode{}}
	bn.dirty = true
	bn.hash = nil
	bn.EncodedChildren = make([][]byte, len(bn.children))
	bn.EncodedChildren[4] = getRandomByteSlice()
	bn.EncodedChildren[5] = getRandomByteSlice()
	bn.EncodedChildren[12] = getRandomByteSlice()
	bn.children[4] = &leafNode{baseNode: &baseNode{}}
	bn.children[5] = &leafNode{baseNode: &baseNode{}}
	bn.children[12] = &leafNode{baseNode: &baseNode{}}

	cloned := bn.deepClone().(*branchNode)

	testSameBranchNodeContent(t, bn, cloned)
}

func TestBranchNode_deepCloneShouldWork(t *testing.T) {
	t.Parallel()

	bn := &branchNode{baseNode: &baseNode{}}
	bn.dirty = true
	bn.hash = getRandomByteSlice()
	bn.EncodedChildren = make([][]byte, len(bn.children))
	bn.EncodedChildren[4] = getRandomByteSlice()
	bn.EncodedChildren[5] = getRandomByteSlice()
	bn.EncodedChildren[12] = getRandomByteSlice()
	bn.children[4] = &leafNode{baseNode: &baseNode{}}
	bn.children[5] = &leafNode{baseNode: &baseNode{}}
	bn.children[12] = &leafNode{baseNode: &baseNode{}}

	cloned := bn.deepClone().(*branchNode)

	testSameBranchNodeContent(t, bn, cloned)
}

func TestPatriciaMerkleTrie_CommitCollapsedDirtyTrieShouldWork(t *testing.T) {
	t.Parallel()

	tr, _, _ := newEmptyTrie()
	_ = tr.Update([]byte("aaa"), []byte("aaa"))
	_ = tr.Update([]byte("nnn"), []byte("nnn"))
	_ = tr.Update([]byte("zzz"), []byte("zzz"))
	_ = tr.Commit()

	tr.root, _ = tr.root.getCollapsed()
	_ = tr.Delete([]byte("zzz"))

	assert.True(t, tr.root.isDirty())
	assert.True(t, tr.root.isCollapsed())

	_ = tr.Commit()

	assert.False(t, tr.root.isDirty())
	assert.True(t, tr.root.isCollapsed())
}

func testSameBranchNodeContent(t *testing.T, expected *branchNode, actual *branchNode) {
	if !reflect.DeepEqual(expected, actual) {
		assert.Fail(t, "not equal content")
		fmt.Printf(
			"expected:\n %s, got: \n%s",
			getBranchNodeContents(expected),
			getBranchNodeContents(actual),
		)
	}
	assert.False(t, expected == actual)
}

func getBranchNodeContents(bn *branchNode) string {
	encodedChildsString := ""
	for i := 0; i < len(bn.EncodedChildren); i++ {
		if i > 0 {
			encodedChildsString += ", "
		}

		if bn.EncodedChildren[i] == nil {
			encodedChildsString += "<nil>"
			continue
		}

		encodedChildsString += hex.EncodeToString(bn.EncodedChildren[i])
	}

	childsString := ""
	for i := 0; i < len(bn.children); i++ {
		if i > 0 {
			childsString += ", "
		}

		if bn.children[i] == nil {
			childsString += "<nil>"
			continue
		}

		childsString += fmt.Sprintf("%p", bn.children[i])
	}

	str := fmt.Sprintf(`extension node:
  		encoded child: %s
  		hash: %s
 		child: %s,
  		dirty: %v
`,
		encodedChildsString,
		hex.EncodeToString(bn.hash),
		childsString,
		bn.dirty)

	return str
}

func BenchmarkMarshallNodeJson(b *testing.B) {
	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	marsh := marshal.JsonMarshalizer{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = marsh.Marshal(bn)
	}
}

func TestBranchNode_newBranchNodeNilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	bn, err := newBranchNode(nil, mock.HasherMock{})
	assert.Nil(t, bn)
	assert.Equal(t, ErrNilMarshalizer, err)
}

func TestBranchNode_newBranchNodeNilHasherShouldErr(t *testing.T) {
	t.Parallel()

	bn, err := newBranchNode(&mock.MarshalizerMock{}, nil)
	assert.Nil(t, bn)
	assert.Equal(t, ErrNilHasher, err)
}

func TestBranchNode_newBranchNodeOkVals(t *testing.T) {
	t.Parallel()

	var children [nrOfChildren]node
	marsh, hasher := getTestMarshAndHasher()
	bn, err := newBranchNode(marsh, hasher)

	assert.Nil(t, err)
	assert.Equal(t, make([][]byte, nrOfChildren), bn.EncodedChildren)
	assert.Equal(t, children, bn.children)
	assert.Equal(t, marsh, bn.marsh)
	assert.Equal(t, hasher, bn.hasher)
	assert.True(t, bn.dirty)
}

func TestBranchNode_getMarshalizer(t *testing.T) {
	t.Parallel()

	expectedMarsh := &mock.MarshalizerMock{}
	bn := &branchNode{
		baseNode: &baseNode{
			marsh: expectedMarsh,
		},
	}

	marsh := bn.getMarshalizer()
	assert.Equal(t, expectedMarsh, marsh)
}

func TestBranchNode_setRootHashCollapsedChildren(t *testing.T) {
	t.Parallel()

	marsh, hasher := getTestMarshAndHasher()
	bn := &branchNode{
		baseNode: &baseNode{
			marsh:  marsh,
			hasher: hasher,
		},
	}

	_, collapsedBn := getBnAndCollapsedBn(marsh, hasher)
	_, collapsedEn := getEnAndCollapsedEn()
	collapsedLn := getLn(marsh, hasher)

	bn.children[0] = collapsedBn
	bn.children[1] = collapsedEn
	bn.children[2] = collapsedLn

	err := bn.setRootHash()
	assert.Nil(t, err)
}

func TestBranchNode_commitCollapsesTrieIfMaxTrieLevelInMemoryIsReached(t *testing.T) {
	t.Parallel()

	bn, collapsedBn := getBnAndCollapsedBn(getTestMarshAndHasher())
	_ = collapsedBn.setRootHash()

	err := bn.commit(true, 0, 1, mock.NewMemDbMock(), mock.NewMemDbMock())
	assert.Nil(t, err)

	assert.Equal(t, collapsedBn.EncodedChildren, bn.EncodedChildren)
	assert.Equal(t, collapsedBn.children, bn.children)
	assert.Equal(t, collapsedBn.hash, bn.hash)
}

func TestBranchNode_reduceNodeBnChild(t *testing.T) {
	t.Parallel()

	marsh, hasher := getTestMarshAndHasher()
	en, _ := getEnAndCollapsedEn()
	pos := 5
	expectedNode, _ := newExtensionNode([]byte{byte(pos)}, en.child, marsh, hasher)

	newNode, err := en.child.reduceNode(pos)
	assert.Nil(t, err)
	assert.Equal(t, expectedNode, newNode)
}

func TestBranchNode_printShouldNotPanicEvenIfNodeIsCollapsed(t *testing.T) {
	t.Parallel()

	bnWriter := bytes.NewBuffer(make([]byte, 0))
	collapsedBnWriter := bytes.NewBuffer(make([]byte, 0))

	db := mock.NewMemDbMock()
	bn, collapsedBn := getBnAndCollapsedBn(getTestMarshAndHasher())
	_ = bn.commit(true, 0, 5, db, db)
	_ = collapsedBn.commit(true, 0, 5, db, db)

	bn.print(bnWriter, 0, db)
	collapsedBn.print(collapsedBnWriter, 0, db)

	assert.Equal(t, bnWriter.Bytes(), collapsedBnWriter.Bytes())
}

func TestBranchNode_getDirtyHashesFromCleanNode(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	bn, _ := getBnAndCollapsedBn(getTestMarshAndHasher())
	_ = bn.commit(true, 0, 5, db, db)
	dirtyHashes := make(data.ModifiedHashes)

	err := bn.getDirtyHashes(dirtyHashes)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(dirtyHashes))
}
