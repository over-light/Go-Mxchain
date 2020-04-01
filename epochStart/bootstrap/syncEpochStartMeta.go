package bootstrap

import (
	"context"
	"time"

	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state/addressConverters"
	"github.com/ElrondNetwork/elrond-go/epochStart/bootstrap/disabled"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/interceptors"
	interceptorsFactory "github.com/ElrondNetwork/elrond-go/process/interceptors/factory"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type epochStartMetaSyncer struct {
	requestHandler                 process.RequestHandler
	messenger                      p2p.Messenger
	epochStartMetaBlockInterceptor EpochStartInterceptor
	marshalizer                    marshal.Marshalizer
	hasher                         hashing.Hasher
	singleDataInterceptor          process.Interceptor
	metaBlockProcessor             *epochStartMetaBlockProcessor
}

// ArgsNewEpochStartMetaSyncer -
type ArgsNewEpochStartMetaSyncer struct {
	RequestHandler    process.RequestHandler
	Messenger         p2p.Messenger
	Marshalizer       marshal.Marshalizer
	TxSignMarshalizer marshal.Marshalizer
	ShardCoordinator  sharding.Coordinator
	KeyGen            crypto.KeyGenerator
	BlockKeyGen       crypto.KeyGenerator
	Hasher            hashing.Hasher
	Signer            crypto.SingleSigner
	BlockSigner       crypto.SingleSigner
	ChainID           []byte
	EconomicsData     *economics.EconomicsData
}

const thresholdForConsideringMetaBlockCorrect = 0.2

// NewEpochStartMetaSyncer will return a new instance of epochStartMetaSyncer
func NewEpochStartMetaSyncer(args ArgsNewEpochStartMetaSyncer) (*epochStartMetaSyncer, error) {
	e := &epochStartMetaSyncer{
		requestHandler: args.RequestHandler,
		messenger:      args.Messenger,
		marshalizer:    args.Marshalizer,
		hasher:         args.Hasher,
	}

	processor, err := NewEpochStartMetaBlockProcessor(
		args.Messenger,
		args.RequestHandler,
		args.Marshalizer,
		args.Hasher,
		thresholdForConsideringMetaBlockCorrect,
	)
	if err != nil {
		return nil, err
	}
	e.metaBlockProcessor = processor.(*epochStartMetaBlockProcessor)

	addrConv, err := addressConverters.NewPlainAddressConverter(32, "")
	if err != nil {
		return nil, err
	}

	argsInterceptedDataFactory := interceptorsFactory.ArgInterceptedDataFactory{
		ProtoMarshalizer:  args.Marshalizer,
		TxSignMarshalizer: args.TxSignMarshalizer,
		Hasher:            args.Hasher,
		ShardCoordinator:  args.ShardCoordinator,
		MultiSigVerifier:  disabled.NewMultiSigVerifier(),
		NodesCoordinator:  disabled.NewNodesCoordinator(),
		KeyGen:            args.KeyGen,
		BlockKeyGen:       args.BlockKeyGen,
		Signer:            args.Signer,
		BlockSigner:       args.BlockSigner,
		AddrConv:          addrConv,
		FeeHandler:        args.EconomicsData,
		HeaderSigVerifier: disabled.NewHeaderSigVerifier(),
		ChainID:           args.ChainID,
		ValidityAttester:  disabled.NewValidityAttester(),
		EpochStartTrigger: disabled.NewEpochStartTrigger(),
	}

	interceptedMetaHdrDataFactory, err := interceptorsFactory.NewInterceptedMetaHeaderDataFactory(&argsInterceptedDataFactory)
	if err != nil {
		return nil, err
	}

	e.singleDataInterceptor, err = interceptors.NewSingleDataInterceptor(
		factory.MetachainBlocksTopic,
		interceptedMetaHdrDataFactory,
		processor,
		disabled.NewThrottler(),
		disabled.NewAntiFloodHandler(),
	)
	if err != nil {
		return nil, err
	}

	return e, nil
}

// SyncEpochStartMeta syncs the latest epoch start metablock
func (e *epochStartMetaSyncer) SyncEpochStartMeta(_ time.Duration) (*block.MetaBlock, error) {
	err := e.initTopicForEpochStartMetaBlockInterceptor()
	if err != nil {
		return nil, err
	}
	defer func() {
		e.resetTopicsAndInterceptors()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	mb, errConsensusNotReached := e.metaBlockProcessor.GetEpochStartMetaBlock(ctx)
	cancel()

	if errConsensusNotReached != nil {
		return nil, errConsensusNotReached
	}

	return mb, nil
}

func (e *epochStartMetaSyncer) resetTopicsAndInterceptors() {
	err := e.messenger.UnregisterMessageProcessor(factory.MetachainBlocksTopic)
	if err != nil {
		log.Info("error unregistering message processors", "error", err)
	}
}

func (e *epochStartMetaSyncer) initTopicForEpochStartMetaBlockInterceptor() error {
	err := e.messenger.CreateTopic(factory.MetachainBlocksTopic, true)
	if err != nil {
		log.Info("error registering message processor", "error", err)
		return err
	}

	err = e.messenger.RegisterMessageProcessor(factory.MetachainBlocksTopic, e.singleDataInterceptor)
	if err != nil {
		return err
	}

	return nil
}

// IsInterfaceNil returns true if underlying object is nil
func (e *epochStartMetaSyncer) IsInterfaceNil() bool {
	return e == nil
}
