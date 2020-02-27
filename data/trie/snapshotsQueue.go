package trie

import (
	"sync"
)

type snapshotsQueue struct {
	queue []*snapshotsQueueEntry
	mut   sync.RWMutex
}

type snapshotsQueueEntry struct {
	rootHash []byte
	newDb    bool
}

func newSnapshotsQueue() *snapshotsQueue {
	return &snapshotsQueue{
		queue: make([]*snapshotsQueueEntry, 0),
	}
}

func (sq *snapshotsQueue) add(rootHash []byte, newDb bool) {
	sq.mut.Lock()
	newSnapshot := &snapshotsQueueEntry{
		rootHash: rootHash,
		newDb:    newDb,
	}
	sq.queue = append(sq.queue, newSnapshot)
	for i := range sq.queue {
		log.Trace("snapshots queue add", "rootHash", sq.queue[i].rootHash)
	}

	sq.mut.Unlock()
}

func (sq *snapshotsQueue) len() int {
	sq.mut.Lock()
	defer sq.mut.Unlock()
	for i := range sq.queue {
		log.Trace("snapshots queue len", "rootHash", sq.queue[i].rootHash)
	}
	return len(sq.queue)
}

func (sq *snapshotsQueue) clone() snapshotsBuffer {
	sq.mut.Lock()

	newQueue := make([]*snapshotsQueueEntry, len(sq.queue))
	for i := range newQueue {
		newQueue[i] = &snapshotsQueueEntry{
			rootHash: sq.queue[i].rootHash,
			newDb:    sq.queue[i].newDb,
		}
	}

	sq.mut.Unlock()

	return &snapshotsQueue{queue: newQueue}
}

func (sq *snapshotsQueue) getFirst() *snapshotsQueueEntry {
	sq.mut.Lock()
	defer sq.mut.Unlock()
	for i := range sq.queue {
		log.Trace("snapshots queue getFirst", "rootHash", sq.queue[i].rootHash)
	}
	return sq.queue[0]
}

func (sq *snapshotsQueue) removeFirst() {
	sq.mut.Lock()

	if len(sq.queue) != 0 {
		log.Trace("snapshots queue removeFirst", "firstRootHash", sq.queue[0].rootHash)
		sq.queue = sq.queue[1:]
	}

	for i := range sq.queue {
		log.Trace("snapshots queue removeFirst", "rootHash", sq.queue[i].rootHash)
	}

	sq.mut.Unlock()
}
