package trie

import (
	"encoding/hex"
	"fmt"
	"io"
	"sync"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
)

var _ = node(&branchNode{})

func newBranchNode(marshalizer marshal.Marshalizer, hasher hashing.Hasher) (*branchNode, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(hasher) {
		return nil, ErrNilHasher
	}

	var children [nrOfChildren]node
	encChildren := make([][]byte, nrOfChildren)

	return &branchNode{
		CollapsedBn: CollapsedBn{
			EncodedChildren: encChildren,
		},
		children: children,
		baseNode: &baseNode{
			dirty:  true,
			marsh:  marshalizer,
			hasher: hasher,
		},
	}, nil
}

func emptyDirtyBranchNode() *branchNode {
	var children [nrOfChildren]node
	encChildren := make([][]byte, nrOfChildren)

	return &branchNode{
		CollapsedBn: CollapsedBn{
			EncodedChildren: encChildren,
		},
		children: children,
		baseNode: &baseNode{
			dirty: true,
		},
	}
}

func (bn *branchNode) getHash() []byte {
	return bn.hash
}

func (bn *branchNode) setGivenHash(hash []byte) {
	bn.hash = hash
}

func (bn *branchNode) isDirty() bool {
	return bn.dirty
}

func (bn *branchNode) getMarshalizer() marshal.Marshalizer {
	return bn.marsh
}

func (bn *branchNode) setMarshalizer(marshalizer marshal.Marshalizer) {
	bn.marsh = marshalizer
}

func (bn *branchNode) getHasher() hashing.Hasher {
	return bn.hasher
}

func (bn *branchNode) setHasher(hasher hashing.Hasher) {
	bn.hasher = hasher
}

func (bn *branchNode) getCollapsed() (node, error) {
	err := bn.isEmptyOrNil()
	if err != nil {
		return nil, err
	}
	if bn.isCollapsed() {
		return bn, nil
	}
	collapsed := bn.clone()
	for i := range bn.children {
		if bn.children[i] != nil {
			var ok bool
			ok, err = hasValidHash(bn.children[i])
			if err != nil {
				return nil, err
			}
			if !ok {
				err = bn.children[i].setHash()
				if err != nil {
					return nil, err
				}
			}
			collapsed.EncodedChildren[i] = bn.children[i].getHash()
			collapsed.children[i] = nil
		}
	}
	return collapsed, nil
}

func (bn *branchNode) setHash() error {
	err := bn.isEmptyOrNil()
	if err != nil {
		return err
	}
	if bn.getHash() != nil {
		return nil
	}
	if bn.isCollapsed() {
		var hash []byte
		hash, err = encodeNodeAndGetHash(bn)
		if err != nil {
			return err
		}
		bn.hash = hash
		return nil
	}
	hash, err := hashChildrenAndNode(bn)
	if err != nil {
		return err
	}
	bn.hash = hash
	return nil
}

func (bn *branchNode) setRootHash() error {
	err := bn.isEmptyOrNil()
	if err != nil {
		return err
	}
	if bn.getHash() != nil {
		return nil
	}
	if bn.isCollapsed() {
		var hash []byte
		hash, err = encodeNodeAndGetHash(bn)
		if err != nil {
			return err
		}
		bn.hash = hash
		return nil
	}

	var wg sync.WaitGroup
	errc := make(chan error, nrOfChildren)

	for i := 0; i < nrOfChildren; i++ {
		if bn.children[i] != nil {
			wg.Add(1)
			go bn.children[i].setHashConcurrent(&wg, errc)
		}
	}
	wg.Wait()
	if len(errc) != 0 {
		for err = range errc {
			return err
		}
	}

	hashed, err := bn.hashNode()
	if err != nil {
		return err
	}

	bn.hash = hashed
	return nil
}

func (bn *branchNode) setHashConcurrent(wg *sync.WaitGroup, c chan error) {
	defer wg.Done()
	err := bn.isEmptyOrNil()
	if err != nil {
		c <- err
		return
	}
	if bn.getHash() != nil {
		return
	}
	if bn.isCollapsed() {
		var hash []byte
		hash, err = encodeNodeAndGetHash(bn)
		if err != nil {
			c <- err
			return
		}
		bn.hash = hash
		return
	}
	hash, err := hashChildrenAndNode(bn)
	if err != nil {
		c <- err
		return
	}
	bn.hash = hash
}

func (bn *branchNode) hashChildren() error {
	err := bn.isEmptyOrNil()
	if err != nil {
		return err
	}
	for i := 0; i < nrOfChildren; i++ {
		if bn.children[i] != nil {
			err = bn.children[i].setHash()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (bn *branchNode) hashNode() ([]byte, error) {
	err := bn.isEmptyOrNil()
	if err != nil {
		return nil, err
	}
	for i := range bn.EncodedChildren {
		if bn.children[i] != nil {
			var encChild []byte
			encChild, err = encodeNodeAndGetHash(bn.children[i])
			if err != nil {
				return nil, err
			}
			bn.EncodedChildren[i] = encChild
		}
	}
	return encodeNodeAndGetHash(bn)
}

func (bn *branchNode) commit(force bool, level byte, originDb data.DBWriteCacher, targetDb data.DBWriteCacher) error {
	level++
	err := bn.isEmptyOrNil()
	if err != nil {
		return err
	}

	shouldNotCommit := !bn.dirty && !force
	if shouldNotCommit {
		return nil
	}

	for i := range bn.children {
		if force {
			err = resolveIfCollapsed(bn, byte(i), originDb)
			if err != nil {
				return err
			}
		}

		if bn.children[i] == nil {
			continue
		}

		err = bn.children[i].commit(force, level, originDb, targetDb)
		if err != nil {
			return err
		}
	}
	bn.dirty = false
	err = encodeNodeAndCommitToDB(bn, targetDb)
	if err != nil {
		return err
	}
	if level == maxTrieLevelAfterCommit {
		var collapsed node
		collapsed, err = bn.getCollapsed()
		if err != nil {
			return err
		}
		if n, ok := collapsed.(*branchNode); ok {
			*bn = *n
		}
	}
	return nil
}

func (bn *branchNode) getEncodedNode() ([]byte, error) {
	err := bn.isEmptyOrNil()
	if err != nil {
		return nil, err
	}
	marshaledNode, err := bn.marsh.Marshal(bn)
	if err != nil {
		return nil, err
	}
	marshaledNode = append(marshaledNode, branch)
	return marshaledNode, nil
}

func (bn *branchNode) resolveCollapsed(pos byte, db data.DBWriteCacher) error {
	err := bn.isEmptyOrNil()
	if err != nil {
		return err
	}
	if childPosOutOfRange(pos) {
		return ErrChildPosOutOfRange
	}
	if len(bn.EncodedChildren[pos]) != 0 {
		var child node
		child, err = getNodeFromDBAndDecode(bn.EncodedChildren[pos], db, bn.marsh, bn.hasher)
		if err != nil {
			return err
		}
		child.setGivenHash(bn.EncodedChildren[pos])
		bn.children[pos] = child
	}
	return nil
}

func (bn *branchNode) isCollapsed() bool {
	for i := range bn.children {
		if bn.children[i] != nil {
			return false
		}
	}
	return true
}

func (bn *branchNode) isPosCollapsed(pos int) bool {
	return bn.children[pos] == nil && len(bn.EncodedChildren[pos]) != 0
}

func (bn *branchNode) tryGet(key []byte, db data.DBWriteCacher) (value []byte, err error) {
	err = bn.isEmptyOrNil()
	if err != nil {
		return nil, err
	}
	if len(key) == 0 {
		return nil, nil
	}
	childPos := key[firstByte]
	if childPosOutOfRange(childPos) {
		return nil, ErrChildPosOutOfRange
	}
	key = key[1:]
	err = resolveIfCollapsed(bn, childPos, db)
	if err != nil {
		return nil, err
	}
	if bn.children[childPos] == nil {
		return nil, nil
	}

	return bn.children[childPos].tryGet(key, db)
}

func (bn *branchNode) getNext(key []byte, db data.DBWriteCacher) (node, []byte, error) {
	err := bn.isEmptyOrNil()
	if err != nil {
		return nil, nil, err
	}
	if len(key) == 0 {
		return nil, nil, ErrValueTooShort
	}
	childPos := key[firstByte]
	if childPosOutOfRange(childPos) {
		return nil, nil, ErrChildPosOutOfRange
	}
	key = key[1:]
	err = resolveIfCollapsed(bn, childPos, db)
	if err != nil {
		return nil, nil, err
	}

	if bn.children[childPos] == nil {
		return nil, nil, ErrNodeNotFound
	}
	return bn.children[childPos], key, nil
}

func (bn *branchNode) insert(n *leafNode, db data.DBWriteCacher) (bool, node, [][]byte, error) {
	emptyHashes := make([][]byte, 0)
	err := bn.isEmptyOrNil()
	if err != nil {
		return false, nil, emptyHashes, err
	}
	if len(n.Key) == 0 {
		return false, nil, emptyHashes, ErrValueTooShort
	}
	childPos := n.Key[firstByte]
	if childPosOutOfRange(childPos) {
		return false, nil, emptyHashes, ErrChildPosOutOfRange
	}
	n.Key = n.Key[1:]
	err = resolveIfCollapsed(bn, childPos, db)
	if err != nil {
		return false, nil, emptyHashes, err
	}

	if bn.children[childPos] != nil {
		var dirty bool
		var newNode node
		var oldHashes [][]byte

		dirty, newNode, oldHashes, err = bn.children[childPos].insert(n, db)
		if !dirty || err != nil {
			return false, bn, emptyHashes, err
		}

		if !bn.dirty {
			oldHashes = append(oldHashes, bn.hash)
		}

		bn.children[childPos] = newNode
		bn.dirty = dirty
		if dirty {
			bn.hash = nil
		}
		return true, bn, oldHashes, nil
	}

	newLn, err := newLeafNode(n.Key, n.Value, bn.marsh, bn.hasher)
	if err != nil {
		return false, nil, emptyHashes, err
	}
	bn.children[childPos] = newLn

	oldHash := make([][]byte, 0)
	if !bn.dirty {
		oldHash = append(oldHash, bn.hash)
	}

	bn.dirty = true
	bn.hash = nil
	return true, bn, oldHash, nil
}

func (bn *branchNode) delete(key []byte, db data.DBWriteCacher) (bool, node, [][]byte, error) {
	emptyHashes := make([][]byte, 0)
	err := bn.isEmptyOrNil()
	if err != nil {
		return false, nil, emptyHashes, err
	}
	if len(key) == 0 {
		return false, nil, emptyHashes, ErrValueTooShort
	}
	childPos := key[firstByte]
	if childPosOutOfRange(childPos) {
		return false, nil, emptyHashes, ErrChildPosOutOfRange
	}
	key = key[1:]
	err = resolveIfCollapsed(bn, childPos, db)
	if err != nil {
		return false, nil, emptyHashes, err
	}

	if bn.children[childPos] == nil {
		return false, bn, emptyHashes, nil
	}

	dirty, newNode, oldHashes, err := bn.children[childPos].delete(key, db)
	if !dirty || err != nil {
		return false, bn, emptyHashes, err
	}

	if !bn.dirty {
		oldHashes = append(oldHashes, bn.hash)
	}

	bn.hash = nil
	bn.children[childPos] = newNode
	if newNode == nil {
		bn.EncodedChildren[childPos] = nil
	}

	numChildren, pos := getChildPosition(bn)

	if numChildren == 1 {
		err = resolveIfCollapsed(bn, byte(pos), db)
		if err != nil {
			return false, nil, emptyHashes, err
		}

		newNode, err = bn.children[pos].reduceNode(pos)
		if err != nil {
			return false, nil, emptyHashes, err
		}

		return true, newNode, oldHashes, nil
	}

	bn.dirty = dirty

	return true, bn, oldHashes, nil
}

func (bn *branchNode) reduceNode(pos int) (node, error) {
	newEn, err := newExtensionNode([]byte{byte(pos)}, bn, bn.marsh, bn.hasher)
	if err != nil {
		return nil, err
	}

	return newEn, nil
}

func getChildPosition(n *branchNode) (nrOfChildren int, childPos int) {
	for i := range n.children {
		if n.children[i] != nil || len(n.EncodedChildren[i]) != 0 {
			nrOfChildren++
			childPos = i
		}
	}
	return
}

func (bn *branchNode) clone() *branchNode {
	nodeClone := *bn
	return &nodeClone
}

func (bn *branchNode) isEmptyOrNil() error {
	if bn == nil {
		return ErrNilNode
	}
	for i := range bn.children {
		if bn.children[i] != nil || len(bn.EncodedChildren[i]) != 0 {
			return nil
		}
	}
	return ErrEmptyNode
}

func (bn *branchNode) print(writer io.Writer, index int, db data.DBWriteCacher) {
	if bn == nil {
		return
	}

	str := fmt.Sprintf("B: %v - %v", hex.EncodeToString(bn.hash), bn.dirty)
	_, _ = fmt.Fprintln(writer, str)
	for i := 0; i < len(bn.children); i++ {
		err := resolveIfCollapsed(bn, byte(i), db)
		if err != nil {
			log.Debug("print trie err", "error", err, "hash", bn.EncodedChildren[i])
		}

		if bn.children[i] == nil {
			continue
		}

		child := bn.children[i]
		for j := 0; j < index+len(str)-1; j++ {
			_, _ = fmt.Fprint(writer, " ")
		}
		str2 := fmt.Sprintf("+ %d: ", i)
		_, _ = fmt.Fprint(writer, str2)
		childIndex := index + len(str) - 1 + len(str2)
		child.print(writer, childIndex, db)
	}
}

func (bn *branchNode) deepClone() node {
	if bn == nil {
		return nil
	}

	clonedNode := &branchNode{baseNode: &baseNode{}}

	if bn.hash != nil {
		clonedNode.hash = make([]byte, len(bn.hash))
		copy(clonedNode.hash, bn.hash)
	}

	clonedNode.EncodedChildren = make([][]byte, len(bn.EncodedChildren))
	for idx, encChild := range bn.EncodedChildren {
		if encChild == nil {
			continue
		}

		clonedEncChild := make([]byte, len(encChild))
		copy(clonedEncChild, encChild)

		clonedNode.EncodedChildren[idx] = clonedEncChild
	}

	for idx, child := range bn.children {
		if child == nil {
			continue
		}

		clonedNode.children[idx] = child.deepClone()
	}

	clonedNode.dirty = bn.dirty
	clonedNode.marsh = bn.marsh
	clonedNode.hasher = bn.hasher

	return clonedNode
}

func (bn *branchNode) getDirtyHashes(hashes data.ModifiedHashes) error {
	err := bn.isEmptyOrNil()
	if err != nil {
		return err
	}

	if !bn.isDirty() {
		return nil
	}

	for i := range bn.children {
		if bn.children[i] == nil {
			continue
		}

		err = bn.children[i].getDirtyHashes(hashes)
		if err != nil {
			return err
		}
	}

	hashes[hex.EncodeToString(bn.getHash())] = struct{}{}
	return nil
}

func (bn *branchNode) getChildren(db data.DBWriteCacher) ([]node, error) {
	err := bn.isEmptyOrNil()
	if err != nil {
		return nil, err
	}

	nextNodes := make([]node, 0)

	for i := range bn.children {
		err = resolveIfCollapsed(bn, byte(i), db)
		if err != nil {
			return nil, err
		}

		if bn.children[i] == nil {
			continue
		}

		nextNodes = append(nextNodes, bn.children[i])
	}

	return nextNodes, nil
}

func (bn *branchNode) isValid() bool {
	nrChildren := 0
	for i := range bn.EncodedChildren {
		if len(bn.EncodedChildren[i]) != 0 || bn.children[i] != nil {
			nrChildren++
		}
	}

	return nrChildren >= 2
}

func (bn *branchNode) setDirty(dirty bool) {
	bn.dirty = dirty
}

func (bn *branchNode) loadChildren(getNode func([]byte) (node, error)) ([][]byte, []node, error) {
	err := bn.isEmptyOrNil()
	if err != nil {
		return nil, nil, err
	}

	existingChildren := make([]node, 0)
	missingChildren := make([][]byte, 0)
	for i := range bn.EncodedChildren {
		if len(bn.EncodedChildren[i]) == 0 {
			continue
		}

		var child node
		child, err = getNode(bn.EncodedChildren[i])
		if err != nil {
			missingChildren = append(missingChildren, bn.EncodedChildren[i])
			continue
		}

		existingChildren = append(existingChildren, child)
		bn.children[i] = child
	}

	return missingChildren, existingChildren, nil
}

func (bn *branchNode) getAllLeaves(leaves map[string][]byte, key []byte, db data.DBWriteCacher, marshalizer marshal.Marshalizer) error {
	err := bn.isEmptyOrNil()
	if err != nil {
		return err
	}

	for i := range bn.children {
		err = resolveIfCollapsed(bn, byte(i), db)
		if err != nil {
			return err
		}

		if bn.children[i] == nil {
			continue
		}

		childKey := append(key, byte(i))
		err = bn.children[i].getAllLeaves(leaves, childKey, db, marshalizer)
		if err != nil {
			return err
		}
	}

	return nil
}
