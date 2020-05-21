package factory

import (
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// interceptedDataCoreComponentsHolder holds the core components required by the intercepted data factory
type interceptedDataCoreComponentsHolder interface {
	InternalMarshalizer() marshal.Marshalizer
	TxMarshalizer() marshal.Marshalizer
	Hasher() hashing.Hasher
	Uint64ByteSliceConverter() typeConverters.Uint64ByteSliceConverter
	AddressPubKeyConverter() state.PubkeyConverter
	ChainID() string
	IsInterfaceNil() bool
}

// interceptedDataCryptoComponentsHolder holds the crypto components required by the intercepted data factory
type interceptedDataCryptoComponentsHolder interface {
	TxSignKeyGen() crypto.KeyGenerator
	BlockSignKeyGen() crypto.KeyGenerator
	TxSingleSigner() crypto.SingleSigner
	BlockSigner() crypto.SingleSigner
	MultiSigner() crypto.MultiSigner
	PublicKey() crypto.PublicKey
	IsInterfaceNil() bool
}

// ArgInterceptedDataFactory holds all dependencies required by the shard and meta intercepted data factory in order to create
// new instances
type ArgInterceptedDataFactory struct {
	CoreComponents         interceptedDataCoreComponentsHolder
	CryptoComponents       interceptedDataCryptoComponentsHolder
	ShardCoordinator       sharding.Coordinator
	NodesCoordinator       sharding.NodesCoordinator
	FeeHandler             process.FeeHandler
	WhiteListerVerifiedTxs process.WhiteListHandler
	HeaderSigVerifier      process.InterceptedHeaderSigVerifier
	ValidityAttester       process.ValidityAttester
	EpochStartTrigger      process.EpochStartTriggerHandler
}
