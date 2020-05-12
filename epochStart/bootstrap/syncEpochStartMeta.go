package bootstrap

import (
	"context"
	"time"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/epochStart/bootstrap/disabled"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/interceptors"
	interceptorsFactory "github.com/ElrondNetwork/elrond-go/process/interceptors/factory"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

var _ epochStart.StartOfEpochMetaSyncer = (*epochStartMetaSyncer)(nil)

type epochStartMetaSyncer struct {
	requestHandler        RequestHandler
	messenger             Messenger
	marshalizer           marshal.Marshalizer
	hasher                hashing.Hasher
	singleDataInterceptor process.Interceptor
	metaBlockProcessor    EpochStartMetaBlockInterceptorProcessor
}

// ArgsNewEpochStartMetaSyncer -
type ArgsNewEpochStartMetaSyncer struct {
	RequestHandler    RequestHandler
	Messenger         Messenger
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
	WhitelistHandler  process.WhiteListHandler
	AddressPubkeyConv state.PubkeyConverter
	NonceConverter    typeConverters.Uint64ByteSliceConverter
}

// thresholdForConsideringMetaBlockCorrect represents the percentage (between 0 and 100) of connected peers to send
// the same meta block in order to consider it correct
const thresholdForConsideringMetaBlockCorrect = 67

// NewEpochStartMetaSyncer will return a new instance of epochStartMetaSyncer
func NewEpochStartMetaSyncer(args ArgsNewEpochStartMetaSyncer) (*epochStartMetaSyncer, error) {
	if check.IfNil(args.AddressPubkeyConv) {
		return nil, epochStart.ErrNilPubkeyConverter
	}

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
	e.metaBlockProcessor = processor

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
		AddressPubkeyConv: args.AddressPubkeyConv,
		FeeHandler:        args.EconomicsData,
		HeaderSigVerifier: disabled.NewHeaderSigVerifier(),
		ChainID:           args.ChainID,
		ValidityAttester:  disabled.NewValidityAttester(),
		EpochStartTrigger: disabled.NewEpochStartTrigger(),
		NonceConverter:    args.NonceConverter,
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
		args.WhitelistHandler,
	)
	if err != nil {
		return nil, err
	}

	return e, nil
}

// SyncEpochStartMeta syncs the latest epoch start metablock
func (e *epochStartMetaSyncer) SyncEpochStartMeta(timeToWait time.Duration) (*block.MetaBlock, error) {
	err := e.initTopicForEpochStartMetaBlockInterceptor()
	if err != nil {
		return nil, err
	}
	defer func() {
		e.resetTopicsAndInterceptors()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeToWait)
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
		log.Trace("error unregistering message processors", "error", err)
	}
}

func (e *epochStartMetaSyncer) initTopicForEpochStartMetaBlockInterceptor() error {
	err := e.messenger.CreateTopic(factory.MetachainBlocksTopic, true)
	if err != nil {
		log.Warn("error messenger create topic", "error", err)
		return err
	}

	e.resetTopicsAndInterceptors()
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
