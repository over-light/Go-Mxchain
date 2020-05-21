package interceptorscontainer

import (
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// ShardInterceptorsContainerFactoryArgs holds the arguments needed for ShardInterceptorsContainerFactory
type ShardInterceptorsContainerFactoryArgs struct {
	CoreComponents         process.CoreComponentsHolder
	CryptoComponents       process.CryptoComponentsHolder
	Accounts               state.AccountsAdapter
	ShardCoordinator       sharding.Coordinator
	NodesCoordinator       sharding.NodesCoordinator
	Messenger              process.TopicHandler
	Store                  dataRetriever.StorageService
	DataPool               dataRetriever.PoolsHolder
	MaxTxNonceDeltaAllowed int
	TxFeeHandler           process.FeeHandler
	BlackList              process.BlackListHandler
	HeaderSigVerifier      process.InterceptedHeaderSigVerifier
	HeaderIntegrityVerifier process.InterceptedHeaderIntegrityVerifier
	SizeCheckDelta         uint32
	ValidityAttester       process.ValidityAttester
	EpochStartTrigger      process.EpochStartTriggerHandler
	WhiteListHandler       process.WhiteListHandler
	WhiteListerVerifiedTxs process.WhiteListHandler
	AntifloodHandler       process.P2PAntifloodHandler
}

// MetaInterceptorsContainerFactoryArgs holds the arguments needed for MetaInterceptorsContainerFactory
type MetaInterceptorsContainerFactoryArgs struct {
	CoreComponents         process.CoreComponentsHolder
	CryptoComponents       process.CryptoComponentsHolder
	ShardCoordinator       sharding.Coordinator
	NodesCoordinator       sharding.NodesCoordinator
	Messenger              process.TopicHandler
	Store                  dataRetriever.StorageService
	DataPool               dataRetriever.PoolsHolder
	Accounts               state.AccountsAdapter
	MaxTxNonceDeltaAllowed int
	TxFeeHandler           process.FeeHandler
	BlackList              process.BlackListHandler
	HeaderSigVerifier      process.InterceptedHeaderSigVerifier
	HeaderIntegrityVerifier process.InterceptedHeaderIntegrityVerifier
	SizeCheckDelta         uint32
	ValidityAttester       process.ValidityAttester
	EpochStartTrigger      process.EpochStartTriggerHandler
	WhiteListHandler       process.WhiteListHandler
	WhiteListerVerifiedTxs process.WhiteListHandler
	AntifloodHandler       process.P2PAntifloodHandler
}
