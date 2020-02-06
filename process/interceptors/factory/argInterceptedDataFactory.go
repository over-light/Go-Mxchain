package factory

import (
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// ArgInterceptedDataFactory holds all dependencies required by the shard and meta intercepted data factory in order to create
// new instances
type ArgInterceptedDataFactory struct {
	ProtoMarshalizer  marshal.Marshalizer
	SignMarshalizer   marshal.Marshalizer
	Hasher            hashing.Hasher
	ShardCoordinator  sharding.Coordinator
	MultiSigVerifier  crypto.MultiSigVerifier
	NodesCoordinator  sharding.NodesCoordinator
	KeyGen            crypto.KeyGenerator
	BlockKeyGen       crypto.KeyGenerator
	Signer            crypto.SingleSigner
	BlockSigner       crypto.SingleSigner
	AddrConv          state.AddressConverter
	FeeHandler        process.FeeHandler
	HeaderSigVerifier process.InterceptedHeaderSigVerifier
}
