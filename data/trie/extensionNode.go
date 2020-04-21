package trie

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"sync"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
)

var _ = node(&extensionNode{})

// Save saves the serialized data of an extension node into a stream through protobuf
func (en *extensionNode) Save(w io.Writer) error {
	b, err := en.Marshal()
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

// Load loads the data from the stream into an extension node object through protobuf
func (en *extensionNode) Load(r io.Reader) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return en.Unmarshal(b)
}

func newExtensionNode(key []byte, child node, marshalizer marshal.Marshalizer, hasher hashing.Hasher) (*extensionNode, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(hasher) {
		return nil, ErrNilHasher
	}

	return &extensionNode{
		CollapsedEn: CollapsedEn{
			Key:          key,
			EncodedChild: nil,
		},
		child: child,
		baseNode: &baseNode{
			dirty:  true,
			marsh:  marshalizer,
			hasher: hasher,
		},
	}, nil
}

func (en *extensionNode) getHash() []byte {
	return en.hash
}

func (en *extensionNode) setGivenHash(hash []byte) {
	en.hash = hash
}

func (en *extensionNode) isDirty() bool {
	return en.dirty
}

func (en *extensionNode) getMarshalizer() marshal.Marshalizer {
	return en.marsh
}

func (en *extensionNode) setMarshalizer(marshalizer marshal.Marshalizer) {
	en.marsh = marshalizer
}

func (en *extensionNode) getHasher() hashing.Hasher {
	return en.hasher
}

func (en *extensionNode) setHasher(hasher hashing.Hasher) {
	en.hasher = hasher
}

func (en *extensionNode) getCollapsed() (node, error) {
	err := en.isEmptyOrNil()
	if err != nil {
		return nil, err
	}
	if en.isCollapsed() {
		return en, nil
	}
	collapsed := en.clone()
	ok, err := hasValidHash(en.child)
	if err != nil {
		return nil, err
	}
	if !ok {
		err = en.child.setHash()
		if err != nil {
			return nil, err
		}
	}
	collapsed.EncodedChild = en.child.getHash()
	collapsed.child = nil
	return collapsed, nil
}

func (en *extensionNode) setHash() error {
	err := en.isEmptyOrNil()
	if err != nil {
		return err
	}
	if en.getHash() != nil {
		return nil
	}
	if en.isCollapsed() {
		var hash []byte
		hash, err = encodeNodeAndGetHash(en)
		if err != nil {
			return err
		}
		en.hash = hash
		return nil
	}
	hash, err := hashChildrenAndNode(en)
	if err != nil {
		return err
	}
	en.hash = hash
	return nil
}

func (en *extensionNode) setHashConcurrent(wg *sync.WaitGroup, c chan error) {
	err := en.setHash()
	if err != nil {
		c <- err
	}
	wg.Done()
}
func (en *extensionNode) setRootHash() error {
	return en.setHash()
}

func (en *extensionNode) hashChildren() error {
	err := en.isEmptyOrNil()
	if err != nil {
		return err
	}
	if en.child != nil {
		err = en.child.setHash()
		if err != nil {
			return err
		}
	}
	return nil
}

func (en *extensionNode) hashNode() ([]byte, error) {
	err := en.isEmptyOrNil()
	if err != nil {
		return nil, err
	}
	if en.child != nil {
		var encChild []byte
		encChild, err = encodeNodeAndGetHash(en.child)
		if err != nil {
			return nil, err
		}
		en.EncodedChild = encChild
	}
	return encodeNodeAndGetHash(en)
}

func (en *extensionNode) commit(force bool, level byte, originDb data.DBWriteCacher, targetDb data.DBWriteCacher) error {
	level++
	err := en.isEmptyOrNil()
	if err != nil {
		return err
	}

	shouldNotCommit := !en.dirty && !force
	if shouldNotCommit {
		return nil
	}

	if force {
		err = resolveIfCollapsed(en, 0, originDb)
		if err != nil {
			return err
		}
	}

	if en.child != nil {
		err = en.child.commit(force, level, originDb, targetDb)
		if err != nil {
			return err
		}
	}

	en.dirty = false
	err = encodeNodeAndCommitToDB(en, targetDb)
	if err != nil {
		return err
	}
	if level == maxTrieLevelAfterCommit {
		var collapsed node
		collapsed, err = en.getCollapsed()
		if err != nil {
			return err
		}
		if n, ok := collapsed.(*extensionNode); ok {
			*en = *n
		}
	}
	return nil
}

func (en *extensionNode) getEncodedNode() ([]byte, error) {
	err := en.isEmptyOrNil()
	if err != nil {
		return nil, err
	}
	marshaledNode, err := en.marsh.Marshal(en)
	if err != nil {
		return nil, err
	}
	marshaledNode = append(marshaledNode, extension)
	return marshaledNode, nil
}

func (en *extensionNode) resolveCollapsed(_ byte, db data.DBWriteCacher) error {
	err := en.isEmptyOrNil()
	if err != nil {
		return err
	}
	child, err := getNodeFromDBAndDecode(en.EncodedChild, db, en.marsh, en.hasher)
	if err != nil {
		return err
	}
	child.setGivenHash(en.EncodedChild)
	en.child = child
	return nil
}

func (en *extensionNode) isCollapsed() bool {
	return en.child == nil && len(en.EncodedChild) != 0
}

func (en *extensionNode) isPosCollapsed(_ int) bool {
	return en.isCollapsed()
}

func (en *extensionNode) tryGet(key []byte, db data.DBWriteCacher) (value []byte, err error) {
	err = en.isEmptyOrNil()
	if err != nil {
		return nil, err
	}
	keyTooShort := len(key) < len(en.Key)
	if keyTooShort {
		return nil, nil
	}
	keysDontMatch := !bytes.Equal(en.Key, key[:len(en.Key)])
	if keysDontMatch {
		return nil, nil
	}
	key = key[len(en.Key):]
	err = resolveIfCollapsed(en, 0, db)
	if err != nil {
		return nil, err
	}

	return en.child.tryGet(key, db)
}

func (en *extensionNode) getNext(key []byte, db data.DBWriteCacher) (node, []byte, error) {
	err := en.isEmptyOrNil()
	if err != nil {
		return nil, nil, err
	}
	keyTooShort := len(key) < len(en.Key)
	if keyTooShort {
		return nil, nil, ErrNodeNotFound
	}
	keysDontMatch := !bytes.Equal(en.Key, key[:len(en.Key)])
	if keysDontMatch {
		return nil, nil, ErrNodeNotFound
	}
	err = resolveIfCollapsed(en, 0, db)
	if err != nil {
		return nil, nil, err
	}

	key = key[len(en.Key):]
	return en.child, key, nil
}

func (en *extensionNode) insert(n *leafNode, db data.DBWriteCacher) (bool, node, [][]byte, error) {
	emptyHashes := make([][]byte, 0)
	err := en.isEmptyOrNil()
	if err != nil {
		return false, nil, emptyHashes, err
	}
	err = resolveIfCollapsed(en, 0, db)
	if err != nil {
		return false, nil, emptyHashes, err
	}
	keyMatchLen := prefixLen(n.Key, en.Key)

	// If the whole key matches, keep this extension node as is
	// and only update the value.
	if keyMatchLen == len(en.Key) {
		var dirty bool
		var newNode, newEn node
		var oldHashes [][]byte

		n.Key = n.Key[keyMatchLen:]
		dirty, newNode, oldHashes, err = en.child.insert(n, db)
		if !dirty || err != nil {
			return false, en, emptyHashes, err
		}

		if !en.dirty {
			oldHashes = append(oldHashes, en.hash)
		}

		newEn, err = newExtensionNode(en.Key, newNode, en.marsh, en.hasher)
		if err != nil {
			return false, nil, emptyHashes, err
		}

		return true, newEn, oldHashes, nil
	}

	oldHash := make([][]byte, 0)
	if !en.dirty {
		oldHash = append(oldHash, en.hash)
	}

	// Otherwise branch out at the index where they differ.
	bn, err := newBranchNode(en.marsh, en.hasher)
	if err != nil {
		return false, nil, emptyHashes, err
	}

	oldChildPos := en.Key[keyMatchLen]
	newChildPos := n.Key[keyMatchLen]
	if childPosOutOfRange(oldChildPos) || childPosOutOfRange(newChildPos) {
		return false, nil, emptyHashes, ErrChildPosOutOfRange
	}

	followingExtensionNode, err := newExtensionNode(en.Key[keyMatchLen+1:], en.child, en.marsh, en.hasher)
	if err != nil {
		return false, nil, emptyHashes, err
	}

	if len(followingExtensionNode.Key) < 1 {
		bn.children[oldChildPos] = en.child
	} else {
		bn.children[oldChildPos] = followingExtensionNode
	}
	n.Key = n.Key[keyMatchLen+1:]
	bn.children[newChildPos] = n

	if keyMatchLen == 0 {
		return true, bn, oldHash, nil
	}

	newEn, err := newExtensionNode(en.Key[:keyMatchLen], bn, en.marsh, en.hasher)
	if err != nil {
		return false, nil, emptyHashes, err
	}

	return true, newEn, oldHash, nil
}

func (en *extensionNode) delete(key []byte, db data.DBWriteCacher) (bool, node, [][]byte, error) {
	emptyHashes := make([][]byte, 0)
	err := en.isEmptyOrNil()
	if err != nil {
		return false, nil, emptyHashes, err
	}
	if len(key) == 0 {
		return false, nil, emptyHashes, ErrValueTooShort
	}
	keyMatchLen := prefixLen(key, en.Key)
	if keyMatchLen < len(en.Key) {
		return false, en, emptyHashes, nil
	}
	err = resolveIfCollapsed(en, 0, db)
	if err != nil {
		return false, nil, emptyHashes, err
	}

	dirty, newNode, oldHashes, err := en.child.delete(key[len(en.Key):], db)
	if !dirty || err != nil {
		return false, en, emptyHashes, err
	}

	if !en.dirty {
		oldHashes = append(oldHashes, en.hash)
	}

	var n node
	switch newNode := newNode.(type) {
	case *leafNode:
		n, err = newLeafNode(concat(en.Key, newNode.Key...), newNode.Value, en.marsh, en.hasher)
		if err != nil {
			return false, nil, emptyHashes, err
		}

		return true, n, oldHashes, nil
	case *extensionNode:
		n, err = newExtensionNode(concat(en.Key, newNode.Key...), newNode.child, en.marsh, en.hasher)
		if err != nil {
			return false, nil, emptyHashes, err
		}

		return true, n, oldHashes, nil
	default:
		n, err = newExtensionNode(en.Key, newNode, en.marsh, en.hasher)
		if err != nil {
			return false, nil, emptyHashes, err
		}

		return true, n, oldHashes, nil
	}
}

func (en *extensionNode) reduceNode(pos int) (node, error) {
	k := append([]byte{byte(pos)}, en.Key...)
	en.Key = k

	return en, nil
}

func (en *extensionNode) clone() *extensionNode {
	nodeClone := *en
	return &nodeClone
}

func (en *extensionNode) isEmptyOrNil() error {
	if en == nil {
		return ErrNilNode
	}
	if en.child == nil && len(en.EncodedChild) == 0 {
		return ErrEmptyNode
	}
	return nil
}

func (en *extensionNode) print(writer io.Writer, index int, db data.DBWriteCacher) {
	if en == nil {
		return
	}

	err := resolveIfCollapsed(en, 0, db)
	if err != nil {
		log.Debug("extension node: print trie err", "error", err, "hash", en.EncodedChild)
	}

	key := ""
	for _, k := range en.Key {
		key += fmt.Sprintf("%d", k)
	}

	str := fmt.Sprintf("E:(%v) - %v", hex.EncodeToString(en.hash), en.dirty)
	_, _ = fmt.Fprint(writer, str)

	if en.child == nil {
		return
	}
	en.child.print(writer, index+len(str), db)
}

func (en *extensionNode) deepClone() node {
	if en == nil {
		return nil
	}

	clonedNode := &extensionNode{baseNode: &baseNode{}}

	if en.Key != nil {
		clonedNode.Key = make([]byte, len(en.Key))
		copy(clonedNode.Key, en.Key)
	}

	if en.EncodedChild != nil {
		clonedNode.EncodedChild = make([]byte, len(en.EncodedChild))
		copy(clonedNode.EncodedChild, en.EncodedChild)
	}

	if en.hash != nil {
		clonedNode.hash = make([]byte, len(en.hash))
		copy(clonedNode.hash, en.hash)
	}

	clonedNode.dirty = en.dirty

	if en.child != nil {
		clonedNode.child = en.child.deepClone()
	}

	clonedNode.marsh = en.marsh
	clonedNode.hasher = en.hasher

	return clonedNode
}

func (en *extensionNode) getDirtyHashes(hashes data.ModifiedHashes) error {
	err := en.isEmptyOrNil()
	if err != nil {
		return err
	}

	if !en.isDirty() {
		return nil
	}

	if en.child == nil {
		return nil
	}

	err = en.child.getDirtyHashes(hashes)
	if err != nil {
		return err
	}
	hashes[hex.EncodeToString(en.getHash())] = struct{}{}

	return nil
}

func (en *extensionNode) getChildren(db data.DBWriteCacher) ([]node, error) {
	err := en.isEmptyOrNil()
	if err != nil {
		return nil, err
	}

	nextNodes := make([]node, 0)

	err = resolveIfCollapsed(en, 0, db)
	if err != nil {
		return nil, err
	}

	nextNodes = append(nextNodes, en.child)

	return nextNodes, nil
}

func (en *extensionNode) isValid() bool {
	if len(en.EncodedChild) == 0 && en.child == nil {
		return false
	}

	if len(en.Key) == 0 {
		return false
	}

	return true
}

func (en *extensionNode) setDirty(dirty bool) {
	en.dirty = dirty
}

func (en *extensionNode) loadChildren(getNode func([]byte) (node, error)) ([][]byte, []node, error) {
	err := en.isEmptyOrNil()
	if err != nil {
		return nil, nil, err
	}

	if en.EncodedChild == nil {
		return nil, nil, ErrNilNode
	}

	child, err := getNode(en.EncodedChild)
	if err != nil {
		return [][]byte{en.EncodedChild}, nil, nil
	}
	en.child = child

	return nil, []node{child}, nil
}

func (en *extensionNode) getAllLeaves(leaves map[string][]byte, key []byte, db data.DBWriteCacher, marshalizer marshal.Marshalizer) error {
	err := en.isEmptyOrNil()
	if err != nil {
		return err
	}

	err = resolveIfCollapsed(en, 0, db)
	if err != nil {
		return err
	}

	childKey := append(key, en.Key...)
	err = en.child.getAllLeaves(leaves, childKey, db, marshalizer)
	if err != nil {
		return err
	}

	return nil
}
