package storageUnit

import (
	"github.com/ElrondNetwork/elrond-go/core"
)

type nilStorer struct {
}

// NewNilStorer will return a nil storer
func NewNilStorer() *nilStorer {
	return new(nilStorer)
}

// GetFromEpoch will do nothing
func (ns *nilStorer) GetFromEpoch(_ []byte, _ uint32) ([]byte, error) {
	return nil, nil
}

// HasInEpoch will do nothing
func (ns *nilStorer) HasInEpoch(_ []byte, _ uint32) error {
	return nil
}

// SearchFirst will do nothing
func (ns *nilStorer) SearchFirst(_ []byte) ([]byte, error) {
	return nil, nil
}

// Put will do nothing
func (ns *nilStorer) Put(_, _ []byte) error {
	return nil
}

// Close will do nothing
func (ns *nilStorer) Close() error {
	return nil
}

// Get will do nothing
func (ns *nilStorer) Get(_ []byte) ([]byte, error) {
	return nil, nil
}

// Has will do nothing
func (ns *nilStorer) Has(_ []byte) error {
	return nil
}

// Remove will do nothing
func (ns *nilStorer) Remove(_ []byte) error {
	return nil
}

// ClearCache will do nothing
func (ns *nilStorer) ClearCache() {
}

// DestroyUnit will do nothing
func (ns *nilStorer) DestroyUnit() error {
	return nil
}

// Iterate will return a closed channel
func (ns *nilStorer) Iterate() chan core.KeyValHolder {
	ch := make(chan core.KeyValHolder)
	close(ch)

	return ch
}

// IsInterfaceNil returns true if there is no value under the interface
func (ns *nilStorer) IsInterfaceNil() bool {
	return ns == nil
}
