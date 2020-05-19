package mock

import (
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
)

// CoreComponentsMock -
type CoreComponentsMock struct {
	IntMarsh            marshal.Marshalizer
	Hash                hashing.Hasher
	UInt64ByteSliceConv typeConverters.Uint64ByteSliceConverter
	AddrPubKeyConv      state.PubkeyConverter
	Chain               string
}

// InternalMarshalizer -
func (ccm *CoreComponentsMock) InternalMarshalizer() marshal.Marshalizer {
	return ccm.IntMarsh
}

// Hasher -
func (ccm *CoreComponentsMock) Hasher() hashing.Hasher {
	return ccm.Hash
}

// Uint64ByteSliceConverter -
func (ccm *CoreComponentsMock) Uint64ByteSliceConverter() typeConverters.Uint64ByteSliceConverter {
	return ccm.UInt64ByteSliceConv
}

// AddressPubKeyConverter -
func (ccm *CoreComponentsMock) AddressPubKeyConverter() state.PubkeyConverter {
	return ccm.AddrPubKeyConv
}

func (ccm *CoreComponentsMock) ChainID() string {
	return ccm.Chain
}

// IsInterfaceNil -
func (ccm *CoreComponentsMock) IsInterfaceNil() bool {
	return ccm == nil
}
