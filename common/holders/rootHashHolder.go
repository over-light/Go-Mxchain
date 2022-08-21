package holders

import "github.com/ElrondNetwork/elrond-go-core/core"

type rootHashHolder struct {
	rootHash []byte
	epoch    core.OptionalUint32
}

// NewRootHashHolder creates a rootHashHolder
func NewRootHashHolder(rootHash []byte) *rootHashHolder {
	return &rootHashHolder{
		rootHash: rootHash,
		epoch:    core.OptionalUint32{},
	}
}

// NewRootHashHolderWithEpoch creates a rootHashHolder
func NewRootHashHolderWithEpoch(rootHash []byte, epoch uint32) *rootHashHolder {
	return &rootHashHolder{
		rootHash: rootHash,
		epoch:    core.OptionalUint32{Value: epoch, HasValue: true},
	}
}

// NewRootHashHolderAsEmpty creates an empty rootHashHolder
func NewRootHashHolderAsEmpty() *rootHashHolder {
	return &rootHashHolder{
		rootHash: nil,
		epoch:    core.OptionalUint32{},
	}
}

// GetRootHash returns the contained rootHash
func (holder *rootHashHolder) GetRootHash() []byte {
	return holder.rootHash
}

// GetEpoch returns the epoch of the contained rootHash
func (holder *rootHashHolder) GetEpoch() core.OptionalUint32 {
	return holder.epoch
}

// IsInterfaceNil returns true if there is no value under the interface
func (holder *rootHashHolder) IsInterfaceNil() bool {
	return holder == nil
}
