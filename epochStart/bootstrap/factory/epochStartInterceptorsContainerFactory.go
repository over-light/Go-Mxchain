package factory

import (
	"time"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/epochStart/bootstrap/disabled"
	disabledGenesis "github.com/ElrondNetwork/elrond-go/genesis/process/disabled"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/factory/interceptorscontainer"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage/timecache"
	"github.com/ElrondNetwork/elrond-go/update"
)

const timeSpanForBadHeaders = time.Minute

// ArgsEpochStartInterceptorContainer holds the arguments needed for creating a new epoch start interceptors
// container factory
type ArgsEpochStartInterceptorContainer struct {
	Config            config.Config
	ShardCoordinator  sharding.Coordinator
	TxSignMarshalizer marshal.Marshalizer
	ProtoMarshalizer  marshal.Marshalizer
	Hasher            hashing.Hasher
	Messenger         process.TopicHandler
	DataPool          dataRetriever.PoolsHolder
	SingleSigner      crypto.SingleSigner
	BlockSingleSigner crypto.SingleSigner
	KeyGen            crypto.KeyGenerator
	BlockKeyGen       crypto.KeyGenerator
	WhiteListHandler  update.WhiteListHandler
	AddressPubkeyConv state.PubkeyConverter
	ChainID           []byte
	NonceConverter    typeConverters.Uint64ByteSliceConverter
}

// NewEpochStartInterceptorsContainer will return a real interceptors container factory, but will many disabled
// components
func NewEpochStartInterceptorsContainer(args ArgsEpochStartInterceptorContainer) (process.InterceptorsContainer, error) {
	nodesCoordinator := disabled.NewNodesCoordinator()
	storer := disabled.NewChainStorer()
	antiFloodHandler := disabled.NewAntiFloodHandler()
	multiSigner := disabled.NewMultiSigner()
	accountsAdapter := disabled.NewAccountsAdapter()
	if check.IfNil(args.AddressPubkeyConv) {
		return nil, epochStart.ErrNilPubkeyConverter
	}
	blackListHandler := timecache.NewTimeCache(timeSpanForBadHeaders)
	feeHandler := &disabledGenesis.DisabledFeeHandler{}
	headerSigVerifier := disabled.NewHeaderSigVerifier()
	sizeCheckDelta := 0
	validityAttester := disabled.NewValidityAttester()
	epochStartTrigger := disabled.NewEpochStartTrigger()

	containerFactoryArgs := interceptorscontainer.MetaInterceptorsContainerFactoryArgs{
		ShardCoordinator:       args.ShardCoordinator,
		NodesCoordinator:       nodesCoordinator,
		Messenger:              args.Messenger,
		Store:                  storer,
		ProtoMarshalizer:       args.ProtoMarshalizer,
		TxSignMarshalizer:      args.TxSignMarshalizer,
		Hasher:                 args.Hasher,
		MultiSigner:            multiSigner,
		DataPool:               args.DataPool,
		Accounts:               accountsAdapter,
		AddressPubkeyConverter: args.AddressPubkeyConv,
		SingleSigner:           args.SingleSigner,
		BlockSingleSigner:      args.BlockSingleSigner,
		KeyGen:                 args.KeyGen,
		BlockKeyGen:            args.BlockKeyGen,
		MaxTxNonceDeltaAllowed: core.MaxTxNonceDeltaAllowed,
		TxFeeHandler:           feeHandler,
		BlackList:              blackListHandler,
		HeaderSigVerifier:      headerSigVerifier,
		ChainID:                args.ChainID,
		SizeCheckDelta:         uint32(sizeCheckDelta),
		ValidityAttester:       validityAttester,
		EpochStartTrigger:      epochStartTrigger,
		WhiteListHandler:       args.WhiteListHandler,
		AntifloodHandler:       antiFloodHandler,
		NonceConverter:         args.NonceConverter,
	}

	interceptorsContainerFactory, err := interceptorscontainer.NewMetaInterceptorsContainerFactory(containerFactoryArgs)
	if err != nil {
		return nil, err
	}

	container, err := interceptorsContainerFactory.Create()
	if err != nil {
		return nil, err
	}

	return container, nil
}
