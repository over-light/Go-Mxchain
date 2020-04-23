package interceptorscontainer

import (
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// ShardInterceptorsContainerFactoryArgs holds the arguments needed for ShardInterceptorsContainerFactory
type ShardInterceptorsContainerFactoryArgs struct {
	Accounts               state.AccountsAdapter
	ShardCoordinator       sharding.Coordinator
	NodesCoordinator       sharding.NodesCoordinator
	Messenger              process.TopicHandler
	Store                  dataRetriever.StorageService
	ProtoMarshalizer       marshal.Marshalizer
	TxSignMarshalizer      marshal.Marshalizer
	Hasher                 hashing.Hasher
	KeyGen                 crypto.KeyGenerator
	BlockSignKeyGen        crypto.KeyGenerator
	SingleSigner           crypto.SingleSigner
	BlockSingleSigner      crypto.SingleSigner
	MultiSigner            crypto.MultiSigner
	DataPool               dataRetriever.PoolsHolder
	AddressPubkeyConverter state.PubkeyConverter
	MaxTxNonceDeltaAllowed int
	TxFeeHandler           process.FeeHandler
	BlackList              process.BlackListHandler
	HeaderSigVerifier      process.InterceptedHeaderSigVerifier
	ChainID                []byte
	SizeCheckDelta         uint32
	ValidityAttester       process.ValidityAttester
	EpochStartTrigger      process.EpochStartTriggerHandler
	WhiteListHandler       process.WhiteListHandler
	AntifloodHandler       process.P2PAntifloodHandler
	NonceConverter         typeConverters.Uint64ByteSliceConverter
}

// MetaInterceptorsContainerFactoryArgs holds the arguments needed for MetaInterceptorsContainerFactory
type MetaInterceptorsContainerFactoryArgs struct {
	ShardCoordinator       sharding.Coordinator
	NodesCoordinator       sharding.NodesCoordinator
	Messenger              process.TopicHandler
	Store                  dataRetriever.StorageService
	ProtoMarshalizer       marshal.Marshalizer
	TxSignMarshalizer      marshal.Marshalizer
	Hasher                 hashing.Hasher
	MultiSigner            crypto.MultiSigner
	DataPool               dataRetriever.PoolsHolder
	Accounts               state.AccountsAdapter
	AddressPubkeyConverter state.PubkeyConverter
	SingleSigner           crypto.SingleSigner
	BlockSingleSigner      crypto.SingleSigner
	KeyGen                 crypto.KeyGenerator
	BlockKeyGen            crypto.KeyGenerator
	MaxTxNonceDeltaAllowed int
	TxFeeHandler           process.FeeHandler
	BlackList              process.BlackListHandler
	HeaderSigVerifier      process.InterceptedHeaderSigVerifier
	ChainID                []byte
	SizeCheckDelta         uint32
	ValidityAttester       process.ValidityAttester
	EpochStartTrigger      process.EpochStartTriggerHandler
	WhiteListHandler       process.WhiteListHandler
	AntifloodHandler       process.P2PAntifloodHandler
	NonceConverter         typeConverters.Uint64ByteSliceConverter
}
