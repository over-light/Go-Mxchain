package state

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go/data"
)

type dataTriesHolder struct {
	tries map[string]data.Trie
	mutex sync.RWMutex
}

// NewDataTriesHolder creates a new instance of dataTriesHolder
func NewDataTriesHolder() *dataTriesHolder {
	return &dataTriesHolder{
		tries: make(map[string]data.Trie),
	}
}

// Put adds a trie pointer to the tries map
func (dth *dataTriesHolder) Put(key []byte, tr data.Trie) {
	dth.mutex.Lock()
	dth.tries[string(key)] = tr
	dth.mutex.Unlock()
}

// Get returns the trie pointer that is stored in the map at the given key
func (dth *dataTriesHolder) Get(key []byte) data.Trie {
	dth.mutex.Lock()
	defer dth.mutex.Unlock()

	return dth.tries[string(key)]
}

// GetAll returns all trie pointers from the map
func (dth *dataTriesHolder) GetAll() []data.Trie {
	dth.mutex.Lock()
	defer dth.mutex.Unlock()

	tries := make([]data.Trie, 0)
	for key := range dth.tries {
		tries = append(tries, dth.tries[key])
	}

	return tries
}

// Reset clears the tries map
func (dth *dataTriesHolder) Reset() {
	dth.mutex.Lock()
	dth.tries = make(map[string]data.Trie)
	dth.mutex.Unlock()
}

// IsInterfaceNil returns true if underlying object is nil
func (dth *dataTriesHolder) IsInterfaceNil() bool {
	return dth == nil
}
