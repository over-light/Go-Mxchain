package factory

import (
	"time"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data/state/addressConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/epochStart/bootstrap/disabled"
	"github.com/ElrondNetwork/elrond-go/epochStart/genesis"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/factory/interceptorscontainer"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/timecache"
	"github.com/ElrondNetwork/elrond-go/update"
)

// ArgsEpochStartInterceptorContainer holds the arguments needed for creating a new epoch start interceptors
// container factory
type ArgsEpochStartInterceptorContainer struct {
	Config            config.Config
	ShardCoordinator  sharding.Coordinator
	Marshalizer       marshal.Marshalizer
	Hasher            hashing.Hasher
	Messenger         process.TopicHandler
	DataPool          dataRetriever.PoolsHolder
	SingleSigner      crypto.SingleSigner
	BlockSingleSigner crypto.SingleSigner
	KeyGen            crypto.KeyGenerator
	BlockKeyGen       crypto.KeyGenerator
	WhiteListHandler  update.WhiteListHandler
}

// NewEpochStartInterceptorsContainer will return a real interceptors container factory, but will many disabled
// components
func NewEpochStartInterceptorsContainer(args ArgsEpochStartInterceptorContainer) (process.InterceptorsContainer, error) {
	nodesCoordinator := disabled.NewNodesCoordinator()
	storer := disabled.ChainStorer{GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
		return disabled.NewDisabledStorer()
	}}
	txSignMarshalizer := marshal.JsonMarshalizer{}
	antiFloodHandler := disabled.NewAntiFloodHandler()
	multiSigner := disabled.NewMultiSigner()
	accountsAdapter := disabled.NewAccountsAdapter()
	addressConverter, err := addressConverters.NewPlainAddressConverter(
		args.Config.Address.Length,
		args.Config.Address.Prefix,
	)
	if err != nil {
		return nil, err
	}
	blackListHandler := timecache.NewTimeCache(1 * time.Minute)
	feeHandler := genesis.NewGenesisFeeHandler()
	headerSigVerifier := disabled.NewHeaderSigVerifier()
	chainID := []byte("chain ID")
	sizeCheckDelta := 0
	validityAttester := disabled.NewValidityAttester()
	epochStartTrigger := disabled.NewEpochStartTrigger()

	containerFactoryArgs := interceptorscontainer.MetaInterceptorsContainerFactoryArgs{
		ShardCoordinator:       args.ShardCoordinator,
		NodesCoordinator:       nodesCoordinator,
		Messenger:              args.Messenger,
		Store:                  &storer,
		ProtoMarshalizer:       args.Marshalizer,
		TxSignMarshalizer:      &txSignMarshalizer,
		Hasher:                 args.Hasher,
		MultiSigner:            multiSigner,
		DataPool:               args.DataPool,
		Accounts:               accountsAdapter,
		AddrConverter:          addressConverter,
		SingleSigner:           args.SingleSigner,
		BlockSingleSigner:      args.BlockSingleSigner,
		KeyGen:                 args.KeyGen,
		BlockKeyGen:            args.BlockKeyGen,
		MaxTxNonceDeltaAllowed: core.MaxTxNonceDeltaAllowed,
		TxFeeHandler:           feeHandler,
		BlackList:              blackListHandler,
		HeaderSigVerifier:      headerSigVerifier,
		ChainID:                chainID,
		SizeCheckDelta:         uint32(sizeCheckDelta),
		ValidityAttester:       validityAttester,
		EpochStartTrigger:      epochStartTrigger,
		WhiteListHandler:       args.WhiteListHandler,
		AntifloodHandler:       antiFloodHandler,
	}

	interceptorsContainerFactory, err := interceptorscontainer.NewMetaInterceptorsContainerFactory(containerFactoryArgs)
	if err != nil {
		return nil, err
	}

	container, err := interceptorsContainerFactory.Create()
	if err != nil {
		return nil, err
	}

	err = interceptorscontainer.SetWhiteListHandlerToInterceptors(container, args.WhiteListHandler)
	if err != nil {
		return nil, err
	}

	return container, nil
}
