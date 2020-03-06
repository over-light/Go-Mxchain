package factory

import (
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/trie"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
)

type interceptedTrieNodeDataFactory struct {
	marshalizer marshal.Marshalizer
	hasher      hashing.Hasher
}

// NewInterceptedTrieNodeDataFactory creates an instance of interceptedTrieNodeDataFactory
func NewInterceptedTrieNodeDataFactory(
	argument *ArgInterceptedDataFactory,
) (*interceptedTrieNodeDataFactory, error) {

	if argument == nil {
		return nil, process.ErrNilArgumentStruct
	}
	if check.IfNil(argument.ProtoMarshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if check.IfNil(argument.Hasher) {
		return nil, process.ErrNilHasher
	}

	return &interceptedTrieNodeDataFactory{
		marshalizer: argument.ProtoMarshalizer,
		hasher:      argument.Hasher,
	}, nil
}

// Create creates instances of InterceptedData by unmarshalling provided buffer
func (sidf *interceptedTrieNodeDataFactory) Create(buff []byte) (process.InterceptedData, error) {
	return trie.NewInterceptedTrieNode(buff, sidf.marshalizer, sidf.hasher)
}

// IsInterfaceNil returns true if there is no value under the interface
func (sidf *interceptedTrieNodeDataFactory) IsInterfaceNil() bool {
	return sidf == nil
}
