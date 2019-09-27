package trie

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/data/mock"
	protobuf "github.com/ElrondNetwork/elrond-go/data/trie/proto"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/stretchr/testify/assert"
)

func TestNode_hashChildrenAndNodeBranchNode(t *testing.T) {
	t.Parallel()

	marsh, hasher := getTestMarshAndHasher()
	bn, collapsedBn := getBnAndCollapsedBn()
	expectedNodeHash, _ := encodeNodeAndGetHash(collapsedBn, marsh, hasher)

	hash, err := hashChildrenAndNode(bn, marsh, hasher)
	assert.Nil(t, err)
	assert.Equal(t, expectedNodeHash, hash)
}

func TestNode_hashChildrenAndNodeExtensionNode(t *testing.T) {
	t.Parallel()

	marsh, hasher := getTestMarshAndHasher()
	en, collapsedEn := getEnAndCollapsedEn()
	expectedNodeHash, _ := encodeNodeAndGetHash(collapsedEn, marsh, hasher)

	hash, err := hashChildrenAndNode(en, marsh, hasher)
	assert.Nil(t, err)
	assert.Equal(t, expectedNodeHash, hash)
}

func TestNode_hashChildrenAndNodeLeafNode(t *testing.T) {
	t.Parallel()

	marsh, hasher := getTestMarshAndHasher()
	ln := getLn()
	expectedNodeHash, _ := encodeNodeAndGetHash(ln, marsh, hasher)

	hash, err := hashChildrenAndNode(ln, marsh, hasher)
	assert.Nil(t, err)
	assert.Equal(t, expectedNodeHash, hash)
}

func TestNode_encodeNodeAndGetHashBranchNode(t *testing.T) {
	t.Parallel()

	marsh, hasher := getTestMarshAndHasher()

	encChildren := make([][]byte, nrOfChildren)
	encChildren[1] = []byte("dog")
	encChildren[10] = []byte("doge")
	bn := newBranchNode()
	bn.EncodedChildren = encChildren

	encNode, _ := marsh.Marshal(bn)
	encNode = append(encNode, branch)
	expextedHash := hasher.Compute(string(encNode))

	hash, err := encodeNodeAndGetHash(bn, marsh, hasher)
	assert.Nil(t, err)
	assert.Equal(t, expextedHash, hash)
}

func TestNode_encodeNodeAndGetHashExtensionNode(t *testing.T) {
	t.Parallel()

	marsh, hasher := getTestMarshAndHasher()
	en := &extensionNode{CollapsedEn: protobuf.CollapsedEn{Key: []byte{2}, EncodedChild: []byte("doge")}}

	encNode, _ := marsh.Marshal(en)
	encNode = append(encNode, extension)
	expextedHash := hasher.Compute(string(encNode))

	hash, err := encodeNodeAndGetHash(en, marsh, hasher)
	assert.Nil(t, err)
	assert.Equal(t, expextedHash, hash)
}

func TestNode_encodeNodeAndGetHashLeafNode(t *testing.T) {
	t.Parallel()

	marsh, hasher := getTestMarshAndHasher()
	ln := newLeafNode([]byte("dog"), []byte("dog"))

	encNode, _ := marsh.Marshal(ln)
	encNode = append(encNode, leaf)
	expextedHash := hasher.Compute(string(encNode))

	hash, err := encodeNodeAndGetHash(ln, marsh, hasher)
	assert.Nil(t, err)
	assert.Equal(t, expextedHash, hash)
}

func TestNode_encodeNodeAndCommitToDBBranchNode(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	marsh, hasher := getTestMarshAndHasher()
	_, collapsedBn := getBnAndCollapsedBn()
	encNode, _ := marsh.Marshal(collapsedBn)
	encNode = append(encNode, branch)
	nodeHash := hasher.Compute(string(encNode))

	err := encodeNodeAndCommitToDB(collapsedBn, db, marsh, hasher)
	assert.Nil(t, err)

	val, _ := db.Get(nodeHash)
	assert.Equal(t, encNode, val)
}

func TestNode_encodeNodeAndCommitToDBExtensionNode(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	marsh, hasher := getTestMarshAndHasher()
	_, collapsedEn := getEnAndCollapsedEn()
	encNode, _ := marsh.Marshal(collapsedEn)
	encNode = append(encNode, extension)
	nodeHash := hasher.Compute(string(encNode))

	err := encodeNodeAndCommitToDB(collapsedEn, db, marsh, hasher)
	assert.Nil(t, err)

	val, _ := db.Get(nodeHash)
	assert.Equal(t, encNode, val)
}

func TestNode_encodeNodeAndCommitToDBLeafNode(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	marsh, hasher := getTestMarshAndHasher()
	ln := getLn()
	encNode, _ := marsh.Marshal(ln)
	encNode = append(encNode, leaf)
	nodeHash := hasher.Compute(string(encNode))

	err := encodeNodeAndCommitToDB(ln, db, marsh, hasher)
	assert.Nil(t, err)

	val, _ := db.Get(nodeHash)
	assert.Equal(t, encNode, val)
}

func TestNode_getNodeFromDBAndDecodeBranchNode(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	marsh, hasher := getTestMarshAndHasher()
	bn, collapsedBn := getBnAndCollapsedBn()
	_ = bn.commit(0, db, marsh, hasher)

	encNode, _ := marsh.Marshal(collapsedBn)
	encNode = append(encNode, branch)
	nodeHash := hasher.Compute(string(encNode))

	node, err := getNodeFromDBAndDecode(nodeHash, db, marsh)
	assert.Nil(t, err)

	h1, _ := encodeNodeAndGetHash(collapsedBn, marsh, hasher)
	h2, _ := encodeNodeAndGetHash(node, marsh, hasher)
	assert.Equal(t, h1, h2)
}

func TestNode_getNodeFromDBAndDecodeExtensionNode(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	marsh, hasher := getTestMarshAndHasher()
	en, collapsedEn := getEnAndCollapsedEn()
	_ = en.commit(0, db, marsh, hasher)

	encNode, _ := marsh.Marshal(collapsedEn)
	encNode = append(encNode, extension)
	nodeHash := hasher.Compute(string(encNode))

	node, err := getNodeFromDBAndDecode(nodeHash, db, marsh)
	assert.Nil(t, err)

	h1, _ := encodeNodeAndGetHash(collapsedEn, marsh, hasher)
	h2, _ := encodeNodeAndGetHash(node, marsh, hasher)
	assert.Equal(t, h1, h2)
}

func TestNode_getNodeFromDBAndDecodeLeafNode(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	marsh, hasher := getTestMarshAndHasher()
	ln := getLn()
	_ = ln.commit(0, db, marsh, hasher)

	encNode, _ := marsh.Marshal(ln)
	encNode = append(encNode, leaf)
	nodeHash := hasher.Compute(string(encNode))

	node, err := getNodeFromDBAndDecode(nodeHash, db, marsh)
	assert.Nil(t, err)
	ln = getLn()
	ln.dirty = false
	assert.Equal(t, ln, node)
}

func TestNode_resolveIfCollapsedBranchNode(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	marsh, hasher := getTestMarshAndHasher()
	bn, collapsedBn := getBnAndCollapsedBn()
	childPos := byte(2)

	_ = bn.commit(0, db, marsh, hasher)

	err := resolveIfCollapsed(collapsedBn, childPos, db, marsh)
	assert.Nil(t, err)
	assert.False(t, collapsedBn.isCollapsed())
}

func TestNode_resolveIfCollapsedExtensionNode(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	marsh, hasher := getTestMarshAndHasher()
	en, collapsedEn := getEnAndCollapsedEn()

	_ = en.commit(0, db, marsh, hasher)

	err := resolveIfCollapsed(collapsedEn, 0, db, marsh)
	assert.Nil(t, err)
	assert.False(t, collapsedEn.isCollapsed())
}

func TestNode_resolveIfCollapsedLeafNode(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	marsh, hasher := getTestMarshAndHasher()
	ln := getLn()

	_ = ln.commit(0, db, marsh, hasher)

	err := resolveIfCollapsed(ln, 0, db, marsh)
	assert.Nil(t, err)
	assert.False(t, ln.isCollapsed())
}

func TestNode_resolveIfCollapsedNilNode(t *testing.T) {
	t.Parallel()

	db := mock.NewMemDbMock()
	marsh, _ := getTestMarshAndHasher()
	var node *extensionNode

	err := resolveIfCollapsed(node, 0, db, marsh)
	assert.Equal(t, ErrNilNode, err)
}

func TestNode_concat(t *testing.T) {
	t.Parallel()

	a := []byte{1, 2, 3}
	var b byte
	b = 4
	ab := []byte{1, 2, 3, 4}
	assert.Equal(t, ab, concat(a, b))
}

func TestNode_hasValidHash(t *testing.T) {
	t.Parallel()

	marsh, hasher := getTestMarshAndHasher()
	bn, _ := getBnAndCollapsedBn()
	ok, err := hasValidHash(bn)
	assert.Nil(t, err)
	assert.False(t, ok)

	_ = bn.setHash(marsh, hasher)
	bn.dirty = false

	ok, err = hasValidHash(bn)
	assert.Nil(t, err)
	assert.True(t, ok)
}

func TestNode_hasValidHashNilNode(t *testing.T) {
	t.Parallel()

	var node *branchNode
	ok, err := hasValidHash(node)
	assert.Equal(t, ErrNilNode, err)
	assert.False(t, ok)
}

func TestNode_decodeNodeBranchNode(t *testing.T) {
	t.Parallel()

	marsh, hasher := getTestMarshAndHasher()
	_, collapsedBn := getBnAndCollapsedBn()

	encNode, _ := marsh.Marshal(collapsedBn)
	encNode = append(encNode, branch)

	node, err := decodeNode(encNode, marsh)
	assert.Nil(t, err)

	h1, _ := encodeNodeAndGetHash(collapsedBn, marsh, hasher)
	h2, _ := encodeNodeAndGetHash(node, marsh, hasher)
	assert.Equal(t, h1, h2)
}

func TestNode_decodeNodeExtensionNode(t *testing.T) {
	t.Parallel()

	marsh, hasher := getTestMarshAndHasher()
	_, collapsedEn := getEnAndCollapsedEn()

	encNode, _ := marsh.Marshal(collapsedEn)
	encNode = append(encNode, extension)

	node, err := decodeNode(encNode, marsh)
	assert.Nil(t, err)

	h1, _ := encodeNodeAndGetHash(collapsedEn, marsh, hasher)
	h2, _ := encodeNodeAndGetHash(node, marsh, hasher)
	assert.Equal(t, h1, h2)
}

func TestNode_decodeNodeLeafNode(t *testing.T) {
	t.Parallel()

	marsh, hasher := getTestMarshAndHasher()
	ln := getLn()

	encNode, _ := marsh.Marshal(ln)
	encNode = append(encNode, leaf)

	node, err := decodeNode(encNode, marsh)
	assert.Nil(t, err)
	ln.dirty = false

	h1, _ := encodeNodeAndGetHash(ln, marsh, hasher)
	h2, _ := encodeNodeAndGetHash(node, marsh, hasher)
	assert.Equal(t, h1, h2)
}

func TestNode_decodeNodeInvalidNode(t *testing.T) {
	t.Parallel()

	marsh, _ := getTestMarshAndHasher()
	ln := getLn()
	invalidNode := byte(6)

	encNode, _ := marsh.Marshal(ln)
	encNode = append(encNode, invalidNode)

	node, err := decodeNode(encNode, marsh)
	assert.Nil(t, node)
	assert.Equal(t, ErrInvalidNode, err)
}

func TestNode_decodeNodeInvalidEncoding(t *testing.T) {
	t.Parallel()

	marsh, _ := getTestMarshAndHasher()

	var encNode []byte

	node, err := decodeNode(encNode, marsh)
	assert.Nil(t, node)
	assert.Equal(t, ErrInvalidEncoding, err)
}

func TestNode_getEmptyNodeOfTypeBranchNode(t *testing.T) {
	t.Parallel()

	bn, err := getEmptyNodeOfType(branch)
	assert.Nil(t, err)
	assert.IsType(t, &branchNode{}, bn)
}

func TestNode_getEmptyNodeOfTypeExtensionNode(t *testing.T) {
	t.Parallel()

	en, err := getEmptyNodeOfType(extension)
	assert.Nil(t, err)
	assert.IsType(t, &extensionNode{}, en)
}

func TestNode_getEmptyNodeOfTypeLeafNode(t *testing.T) {
	t.Parallel()

	ln, err := getEmptyNodeOfType(leaf)
	assert.Nil(t, err)
	assert.IsType(t, &leafNode{}, ln)
}

func TestNode_getEmptyNodeOfTypeWrongNode(t *testing.T) {
	t.Parallel()

	n, err := getEmptyNodeOfType(6)
	assert.Equal(t, ErrInvalidNode, err)
	assert.Nil(t, n)
}

func TestNode_childPosOutOfRange(t *testing.T) {
	t.Parallel()

	assert.True(t, childPosOutOfRange(17))
	assert.False(t, childPosOutOfRange(5))
}

func TestMarshalingAndUnmarshalingWithCapnp(t *testing.T) {
	_, collapsedBn := getBnAndCollapsedBn()
	collapsedBn.dirty = false
	marsh := marshal.CapnpMarshalizer{}
	bn := newBranchNode()

	encBn, err := marsh.Marshal(collapsedBn)
	assert.Nil(t, err)
	assert.NotNil(t, encBn)

	err = marsh.Unmarshal(bn, encBn)
	assert.Nil(t, err)
	assert.Equal(t, collapsedBn, bn)
}

func TestKeyBytesToHex(t *testing.T) {
	t.Parallel()

	var test = []struct {
		key, hex []byte
	}{
		{[]byte("doe"), []byte{6, 4, 6, 15, 6, 5, 16}},
		{[]byte("dog"), []byte{6, 4, 6, 15, 6, 7, 16}},
	}

	for i := range test {
		assert.Equal(t, test[i].hex, keyBytesToHex(test[i].key))
	}
}

func TestPrefixLen(t *testing.T) {
	t.Parallel()

	var test = []struct {
		a, b   []byte
		length int
	}{
		{[]byte("doe"), []byte("dog"), 2},
		{[]byte("dog"), []byte("dogglesworth"), 3},
		{[]byte("mouse"), []byte("mouse"), 5},
		{[]byte("caterpillar"), []byte("cats"), 3},
		{[]byte("caterpillar"), []byte(""), 0},
		{[]byte(""), []byte("caterpillar"), 0},
		{[]byte("a"), []byte("caterpillar"), 0},
	}

	for i := range test {
		assert.Equal(t, test[i].length, prefixLen(test[i].a, test[i].b))
	}
}

func TestGetOldHashesIfNodeIsCollapsed(t *testing.T) {
	t.Parallel()

	msh, hsh := getTestMarshAndHasher()
	evictionCacheSize := 100
	evictionWaitList, _ := mock.NewEvictionWaitingList(evictionCacheSize, mock.NewMemDbMock(), msh)

	tr := &patriciaMerkleTrie{
		db:                    mock.NewMemDbMock(),
		dbEvictionWaitingList: evictionWaitList,
		oldHashes:             make([][]byte, 0),
		oldRoot:               make([]byte, 0),
		marshalizer:           msh,
		hasher:                hsh,
	}

	_ = tr.Update([]byte("doe"), []byte("reindeer"))
	_ = tr.Update([]byte("dog"), []byte("puppy"))
	_ = tr.Update([]byte("dogglesworth"), []byte("cat"))

	rootHash, _ := tr.Root()
	rootKey := []byte{6, 4, 6, 15, 6}
	nextNode, _, _ := tr.root.getNext(rootKey, tr.db, tr.marshalizer)

	_ = tr.Commit()

	tr.root = &extensionNode{
		CollapsedEn: protobuf.CollapsedEn{
			Key:          rootKey,
			EncodedChild: nextNode.getHash(),
		},
		child: nil,
		hash:  rootHash,
		dirty: false,
	}
	_ = tr.Update([]byte("doeee"), []byte("value of doeee"))

	assert.Equal(t, 3, len(tr.oldHashes))
}

func TestClearOldHashesAndOldRootOnCommit(t *testing.T) {
	t.Parallel()

	msh, hsh := getTestMarshAndHasher()
	evictionCacheSize := 100
	evictionWaitList, _ := mock.NewEvictionWaitingList(evictionCacheSize, mock.NewMemDbMock(), msh)

	tr := &patriciaMerkleTrie{
		db:                    mock.NewMemDbMock(),
		dbEvictionWaitingList: evictionWaitList,
		oldHashes:             make([][]byte, 0),
		oldRoot:               make([]byte, 0),
		marshalizer:           msh,
		hasher:                hsh,
	}

	_ = tr.Update([]byte("doe"), []byte("reindeer"))
	_ = tr.Update([]byte("dog"), []byte("puppy"))
	_ = tr.Update([]byte("dogglesworth"), []byte("cat"))
	_ = tr.Commit()
	root, _ := tr.Root()

	_ = tr.Update([]byte("doeee"), []byte("value of doeee"))

	assert.Equal(t, 3, len(tr.oldHashes))
	assert.Equal(t, root, tr.oldRoot)

	_ = tr.Commit()

	assert.Equal(t, 0, len(tr.oldHashes))
	assert.Equal(t, 0, len(tr.oldRoot))
}

func TestTrieDatabasePruning(t *testing.T) {
	t.Parallel()

	msh, hsh := getTestMarshAndHasher()
	size := 5
	evictionWaitList, _ := mock.NewEvictionWaitingList(size, mock.NewMemDbMock(), msh)

	tr := &patriciaMerkleTrie{
		db:                    mock.NewMemDbMock(),
		dbEvictionWaitingList: evictionWaitList,
		oldHashes:             make([][]byte, 0),
		oldRoot:               make([]byte, 0),
		marshalizer:           msh,

		hasher: hsh,
	}

	_ = tr.Update([]byte("doe"), []byte("reindeer"))
	_ = tr.Update([]byte("dog"), []byte("puppy"))
	_ = tr.Update([]byte("dogglesworth"), []byte("cat"))
	_ = tr.Commit()

	key := []byte{6, 4, 6, 15, 6, 7, 16}
	oldHashes := make([][]byte, 0)
	n := tr.root
	rootHash, _ := tr.Root()
	oldHashes = append(oldHashes, rootHash)

	for i := 0; i < 3; i++ {
		n, key, _ = n.getNext(key, tr.db, tr.marshalizer)
		oldHashes = append(oldHashes, n.getHash())
	}

	_ = tr.Update([]byte("dog"), []byte("doee"))
	_ = tr.Commit()
	err := tr.Prune(rootHash)
	assert.Nil(t, err)

	for i := range oldHashes {
		encNode, err := tr.db.Get(oldHashes[i])
		assert.Nil(t, encNode)
		assert.NotNil(t, err)
	}
}
