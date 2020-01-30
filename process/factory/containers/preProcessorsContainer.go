package containers

import (
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/cornelk/hashmap"
)

// preProcessorsContainer is an PreProcessors holder organized by type
type preProcessorsContainer struct {
	objects *hashmap.HashMap
}

// NewPreProcessorsContainer will create a new instance of a container
func NewPreProcessorsContainer() *preProcessorsContainer {
	return &preProcessorsContainer{
		objects: &hashmap.HashMap{},
	}
}

// Get returns the object stored at a certain key.
// Returns an error if the element does not exist
func (ppc *preProcessorsContainer) Get(key block.Type) (process.PreProcessor, error) {
	value, ok := ppc.objects.Get(uint8(key))
	if !ok {
		return nil, process.ErrInvalidContainerKey
	}

	preProcessor, ok := value.(process.PreProcessor)
	if !ok {
		return nil, process.ErrWrongTypeInContainer
	}

	return preProcessor, nil
}

// Add will add an object at a given key. Returns
// an error if the element already exists
func (ppc *preProcessorsContainer) Add(key block.Type, preProcessor process.PreProcessor) error {
	if check.IfNil(preProcessor) {
		return process.ErrNilContainerElement
	}

	ok := ppc.objects.Insert(uint8(key), preProcessor)
	if !ok {
		return process.ErrContainerKeyAlreadyExists
	}

	return nil
}

// AddMultiple will add objects with given keys. Returns
// an error if one element already exists, lengths mismatch or an interceptor is nil
func (ppc *preProcessorsContainer) AddMultiple(keys []block.Type, preProcessors []process.PreProcessor) error {
	if len(keys) != len(preProcessors) {
		return process.ErrLenMismatch
	}

	for idx, key := range keys {
		err := ppc.Add(key, preProcessors[idx])
		if err != nil {
			return err
		}
	}

	return nil
}

// Replace will add (or replace if it already exists) an object at a given key
func (ppc *preProcessorsContainer) Replace(key block.Type, preProcessor process.PreProcessor) error {
	if check.IfNil(preProcessor) {
		return process.ErrNilContainerElement
	}

	ppc.objects.Set(uint8(key), preProcessor)
	return nil
}

// Remove will remove an object at a given key
func (ppc *preProcessorsContainer) Remove(key block.Type) {
	ppc.objects.Del(uint8(key))
}

// Len returns the length of the added objects
func (ppc *preProcessorsContainer) Len() int {
	return ppc.objects.Len()
}

// Keys returns all the keys from the container
func (ppc *preProcessorsContainer) Keys() []block.Type {
	keys := make([]block.Type, 0)
	for key := range ppc.objects.Iter() {
		uint8key, ok := key.Key.(uint8)
		if !ok {
			continue
		}

		blockType := block.Type(uint8key)
		keys = append(keys, blockType)
	}
	return keys
}

// IsInterfaceNil returns true if there is no value under the interface
func (ppc *preProcessorsContainer) IsInterfaceNil() bool {
	return ppc == nil
}
