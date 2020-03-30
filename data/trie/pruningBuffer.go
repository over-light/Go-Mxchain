package trie

import (
	"bytes"
	"sync"
)

type pruningBuffer struct {
	mutOp  sync.RWMutex
	buffer [][]byte
}

func newPruningBuffer() *pruningBuffer {
	return &pruningBuffer{
		buffer: make([][]byte, 0),
	}
}

func (sb *pruningBuffer) add(rootHash []byte) {
	sb.mutOp.Lock()
	defer sb.mutOp.Unlock()

	if len(sb.buffer) == pruningBufferLen {
		log.Trace("pruning buffer is full", "rootHash", rootHash)
		return
	}

	sb.buffer = append(sb.buffer, rootHash)
	log.Trace("pruning buffer add", "rootHash", rootHash)
}

func (sb *pruningBuffer) remove(rootHash []byte) {
	sb.mutOp.Lock()
	defer sb.mutOp.Unlock()

	for i := 0; i < len(sb.buffer); i++ {
		if bytes.Equal(sb.buffer[i], rootHash) {
			sb.buffer = append(sb.buffer[:i], sb.buffer[i+1:]...)
		}
	}
}

func (sb *pruningBuffer) removeAll() [][]byte {
	sb.mutOp.Lock()
	defer sb.mutOp.Unlock()

	log.Trace("pruning buffer", "len", len(sb.buffer))

	buffer := sb.buffer
	sb.buffer = make([][]byte, 0)

	return buffer
}
