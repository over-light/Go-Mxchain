package trie

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go/data"
)

type snapshotDb struct {
	data.DBWriteCacher
	numReferences   uint32
	shouldBeRemoved bool
	path            string
	mutex           sync.RWMutex
}

// DecreaseNumReferences decreases the num references counter
func (s *snapshotDb) DecreaseNumReferences() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.numReferences--

	if s.numReferences == 0 && s.shouldBeRemoved {
		removeSnapshot(s.DBWriteCacher, s.path)
	}
}

// IncreaseNumReferences increases the num references counter
func (s *snapshotDb) IncreaseNumReferences() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.numReferences++
}

// MarkForRemoval marks the current db for removal. When the numReferences buffer reaches 0, the db will be removed
func (s *snapshotDb) MarkForRemoval() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.shouldBeRemoved = true
}

// SetPath sets the db path
func (s *snapshotDb) SetPath(path string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.path = path
}

// IsInUse returns true if the numReferences counter is greater than 0
func (s *snapshotDb) IsInUse() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.numReferences != 0 {
		return true
	}

	return false
}
