package trie

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/storage"
)

type trieSyncer struct {
	trie      *patriciaMerkleTrie
	rootFound bool
	rootHash  []byte

	requestHandler          RequestHandler
	interceptedNodes        storage.Cacher
	shardId                 uint32
	topic                   string
	waitTimeBetweenRequests time.Duration

	nodeHashes      map[string]struct{}
	nodeHashesMutex sync.Mutex

	receivedNodes      map[string]node
	receivedNodesMutex sync.Mutex

	chanReceivedNew chan bool
}

// NewTrieSyncer creates a new instance of trieSyncer
func NewTrieSyncer(
	requestHandler RequestHandler,
	interceptedNodes storage.Cacher,
	trie data.Trie,
	shardId uint32,
	topic string,
) (*trieSyncer, error) {
	if check.IfNil(requestHandler) {
		return nil, ErrNilRequestHandler
	}
	if check.IfNil(interceptedNodes) {
		return nil, data.ErrNilCacher
	}
	if check.IfNil(trie) {
		return nil, ErrNilTrie
	}
	if len(topic) == 0 {
		return nil, ErrInvalidTrieTopic
	}

	pmt, ok := trie.(*patriciaMerkleTrie)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	ts := &trieSyncer{
		requestHandler:          requestHandler,
		interceptedNodes:        interceptedNodes,
		trie:                    pmt,
		nodeHashes:              make(map[string]struct{}),
		receivedNodes:           make(map[string]node),
		topic:                   topic,
		shardId:                 shardId,
		waitTimeBetweenRequests: requestHandler.RequestInterval(),
		chanReceivedNew:         make(chan bool),
	}
	ts.interceptedNodes.RegisterHandler(ts.trieNodeIntercepted)

	return ts, nil
}

// StartSyncing completes the trie, asking for missing trie nodes on the network
func (ts *trieSyncer) StartSyncing(rootHash []byte, ctx context.Context) error {
	if len(rootHash) == 0 {
		return ErrInvalidHash
	}
	if ctx == nil {
		return ErrNilContext
	}

	ts.nodeHashesMutex.Lock()
	ts.nodeHashes = make(map[string]struct{})
	ts.nodeHashes[string(rootHash)] = struct{}{}
	ts.nodeHashesMutex.Unlock()

	ts.rootFound = false
	ts.rootHash = rootHash

	for {
		err := ts.getNextNodes()
		if err != nil {
			return err
		}

		_ = core.EmptyChannel(ts.chanReceivedNew)

		numRequested := ts.requestNodes()
		if numRequested == 0 {
			err := ts.trie.Commit()
			if err != nil {
				return err
			}

			return nil
		}

		select {
		case <-ts.chanReceivedNew:
			continue
		case <-time.After(ts.waitTimeBetweenRequests):
			continue
		case <-ctx.Done():
			return ErrTimeIsOut
		}
	}
}

func (ts *trieSyncer) getNextNodes() error {
	var currentNode node
	var err error
	nextNodes := make([]node, 0)
	missingNodes := make([][]byte, 0)
	currentMissingNodes := make([][]byte, 0)

	newElement := true

	for newElement {
		newElement = false

		ts.nodeHashesMutex.Lock()
		for nodeHash := range ts.nodeHashes {
			currentMissingNodes = currentMissingNodes[:0]

			currentNode, err = ts.getNode([]byte(nodeHash))
			if err != nil {
				continue
			}

			if !ts.rootFound && bytes.Equal([]byte(nodeHash), ts.rootHash) {
				ts.trie.root = currentNode
			}

			currentMissingNodes, err = currentNode.loadChildren(ts.getNode)
			if err != nil {
				ts.nodeHashesMutex.Unlock()
				return err
			}

			if len(currentMissingNodes) > 0 {
				missingNodes = append(missingNodes, currentMissingNodes...)
				continue
			}

			delete(ts.nodeHashes, nodeHash)
			ts.deleteFromReceived(nodeHash)

			nextNodes, err = currentNode.getChildren(ts.trie.Database())
			if err != nil {
				ts.nodeHashesMutex.Unlock()
				return err
			}

			tmpNewElement := ts.addNew(nextNodes)
			newElement = newElement || tmpNewElement
		}
		ts.nodeHashesMutex.Unlock()
	}

	ts.nodeHashesMutex.Lock()
	for _, missingNode := range missingNodes {
		ts.nodeHashes[string(missingNode)] = struct{}{}
	}
	ts.nodeHashesMutex.Unlock()

	return nil
}

func (ts *trieSyncer) deleteFromReceived(nodeHash string) {
	ts.receivedNodesMutex.Lock()
	delete(ts.receivedNodes, nodeHash)
	ts.receivedNodesMutex.Unlock()
}

// adds new elements to needed hash map, lock ts.nodeHashesMutex before calling
func (ts *trieSyncer) addNew(nextNodes []node) bool {
	newElement := false
	for _, nextNode := range nextNodes {
		nextHash := string(nextNode.getHash())
		if _, ok := ts.nodeHashes[nextHash]; !ok {
			ts.nodeHashes[nextHash] = struct{}{}
			newElement = true
		}
	}

	return newElement
}

// Trie returns the synced trie
func (ts *trieSyncer) Trie() data.Trie {
	return ts.trie
}

func (ts *trieSyncer) getNode(hash []byte) (node, error) {
	n, ok := ts.interceptedNodes.Get(hash)
	if ok {
		return trieNode(n)
	}

	ts.receivedNodesMutex.Lock()
	node, ok := ts.receivedNodes[string(hash)]
	ts.receivedNodesMutex.Unlock()

	if ok {
		return node, nil
	}

	return nil, ErrNodeNotFound
}

func trieNode(data interface{}) (node, error) {
	n, ok := data.(*InterceptedTrieNode)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return n.node, nil
}

func (ts *trieSyncer) requestNodes() uint32 {
	ts.nodeHashesMutex.Lock()
	numRequested := uint32(len(ts.nodeHashes))
	for hash := range ts.nodeHashes {
		ts.requestHandler.RequestTrieNodes(ts.shardId, []byte(hash), ts.topic)
	}
	ts.nodeHashesMutex.Unlock()

	return numRequested
}

func (ts *trieSyncer) trieNodeIntercepted(hash []byte, val interface{}) {
	ts.nodeHashesMutex.Lock()
	_, ok := ts.nodeHashes[string(hash)]
	ts.nodeHashesMutex.Unlock()
	if !ok {
		return
	}

	node, err := trieNode(val)
	if err != nil {
		return
	}

	ts.receivedNodesMutex.Lock()
	ts.receivedNodes[string(hash)] = node
	ts.receivedNodesMutex.Unlock()

	ts.chanReceivedNew <- true
}

// IsInterfaceNil returns true if there is no value under the interface
func (ts *trieSyncer) IsInterfaceNil() bool {
	return ts == nil
}
