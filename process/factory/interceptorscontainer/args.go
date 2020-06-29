package interceptorscontainer

import (
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// ShardInterceptorsContainerFactoryArgs holds the arguments needed for ShardInterceptorsContainerFactory
type ShardInterceptorsContainerFactoryArgs struct {
	Accounts                state.AccountsAdapter
	ShardCoordinator        sharding.Coordinator
	NodesCoordinator        sharding.NodesCoordinator
	Messenger               process.TopicHandler
	Store                   dataRetriever.StorageService
	ProtoMarshalizer        marshal.Marshalizer
	TxSignMarshalizer       marshal.Marshalizer
	Hasher                  hashing.Hasher
	KeyGen                  crypto.KeyGenerator
	BlockSignKeyGen         crypto.KeyGenerator
	SingleSigner            crypto.SingleSigner
	BlockSingleSigner       crypto.SingleSigner
	MultiSigner             crypto.MultiSigner
	DataPool                dataRetriever.PoolsHolder
	AddressPubkeyConverter  core.PubkeyConverter
	MaxTxNonceDeltaAllowed  int
	TxFeeHandler            process.FeeHandler
	BlockBlackList          process.TimeCacher
	HeaderSigVerifier       process.InterceptedHeaderSigVerifier
	HeaderIntegrityVerifier process.InterceptedHeaderIntegrityVerifier
	SizeCheckDelta          uint32
	ValidityAttester        process.ValidityAttester
	EpochStartTrigger       process.EpochStartTriggerHandler
	WhiteListHandler        process.WhiteListHandler
	WhiteListerVerifiedTxs  process.WhiteListHandler
	AntifloodHandler        process.P2PAntifloodHandler
	ArgumentsParser         process.ArgumentsParser
	ChainID                 []byte
}

// MetaInterceptorsContainerFactoryArgs holds the arguments needed for MetaInterceptorsContainerFactory
type MetaInterceptorsContainerFactoryArgs struct {
	ShardCoordinator        sharding.Coordinator
	NodesCoordinator        sharding.NodesCoordinator
	Messenger               process.TopicHandler
	Store                   dataRetriever.StorageService
	ProtoMarshalizer        marshal.Marshalizer
	TxSignMarshalizer       marshal.Marshalizer
	Hasher                  hashing.Hasher
	MultiSigner             crypto.MultiSigner
	DataPool                dataRetriever.PoolsHolder
	Accounts                state.AccountsAdapter
	AddressPubkeyConverter  core.PubkeyConverter
	SingleSigner            crypto.SingleSigner
	BlockSingleSigner       crypto.SingleSigner
	KeyGen                  crypto.KeyGenerator
	BlockKeyGen             crypto.KeyGenerator
	MaxTxNonceDeltaAllowed  int
	TxFeeHandler            process.FeeHandler
	BlackList               process.TimeCacher
	HeaderSigVerifier       process.InterceptedHeaderSigVerifier
	HeaderIntegrityVerifier process.InterceptedHeaderIntegrityVerifier
	SizeCheckDelta          uint32
	ValidityAttester        process.ValidityAttester
	EpochStartTrigger       process.EpochStartTriggerHandler
	WhiteListHandler        process.WhiteListHandler
	WhiteListerVerifiedTxs  process.WhiteListHandler
	AntifloodHandler        process.P2PAntifloodHandler
	ArgumentsParser         process.ArgumentsParser
	ChainID                 []byte
}
