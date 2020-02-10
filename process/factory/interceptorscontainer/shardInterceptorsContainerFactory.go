package interceptorscontainer

import (
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/throttler"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/factory/containers"
	interceptorFactory "github.com/ElrondNetwork/elrond-go/process/interceptors/factory"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

var _ process.InterceptorsContainerFactory = (*shardInterceptorsContainerFactory)(nil)

// shardInterceptorsContainerFactory will handle the creation the interceptors container for shards
type shardInterceptorsContainerFactory struct {
	*baseInterceptorsContainerFactory
	keyGen        crypto.KeyGenerator
	singleSigner  crypto.SingleSigner
	addrConverter state.AddressConverter
}

// NewShardInterceptorsContainerFactory is responsible for creating a new interceptors factory object
func NewShardInterceptorsContainerFactory(
	accounts state.AccountsAdapter,
	shardCoordinator sharding.Coordinator,
	nodesCoordinator sharding.NodesCoordinator,
	messenger process.TopicHandler,
	store dataRetriever.StorageService,
	marshalizer marshal.Marshalizer,
	hasher hashing.Hasher,
	keyGen crypto.KeyGenerator,
	blockSignKeyGen crypto.KeyGenerator,
	singleSigner crypto.SingleSigner,
	blockSingleSigner crypto.SingleSigner,
	multiSigner crypto.MultiSigner,
	dataPool dataRetriever.PoolsHolder,
	addrConverter state.AddressConverter,
	maxTxNonceDeltaAllowed int,
	txFeeHandler process.FeeHandler,
	blackList process.BlackListHandler,
	headerSigVerifier process.InterceptedHeaderSigVerifier,
	chainID []byte,
	sizeCheckDelta uint32,
	validityAttester process.ValidityAttester,
	epochStartTrigger process.EpochStartTriggerHandler,
) (*shardInterceptorsContainerFactory, error) {
	if sizeCheckDelta > 0 {
		marshalizer = marshal.NewSizeCheckUnmarshalizer(marshalizer, sizeCheckDelta)
	}
	err := checkBaseParams(
		shardCoordinator,
		accounts,
		marshalizer,
		hasher,
		store,
		dataPool,
		messenger,
		multiSigner,
		nodesCoordinator,
		blackList,
	)
	if err != nil {
		return nil, err
	}

	if check.IfNil(keyGen) {
		return nil, process.ErrNilKeyGen
	}
	if check.IfNil(singleSigner) {
		return nil, process.ErrNilSingleSigner
	}
	if check.IfNil(addrConverter) {
		return nil, process.ErrNilAddressConverter
	}
	if check.IfNil(txFeeHandler) {
		return nil, process.ErrNilEconomicsFeeHandler
	}
	if check.IfNil(blockSignKeyGen) {
		return nil, process.ErrNilKeyGen
	}
	if check.IfNil(blockSingleSigner) {
		return nil, process.ErrNilSingleSigner
	}
	if check.IfNil(headerSigVerifier) {
		return nil, process.ErrNilHeaderSigVerifier
	}
	if len(chainID) == 0 {
		return nil, process.ErrInvalidChainID
	}
	if check.IfNil(validityAttester) {
		return nil, process.ErrNilValidityAttester
	}
	if check.IfNil(epochStartTrigger) {
		return nil, process.ErrNilEpochStartTrigger
	}

	argInterceptorFactory := &interceptorFactory.ArgInterceptedDataFactory{
		Marshalizer:       marshalizer,
		Hasher:            hasher,
		ShardCoordinator:  shardCoordinator,
		MultiSigVerifier:  multiSigner,
		NodesCoordinator:  nodesCoordinator,
		KeyGen:            keyGen,
		BlockKeyGen:       blockSignKeyGen,
		Signer:            singleSigner,
		BlockSigner:       blockSingleSigner,
		AddrConv:          addrConverter,
		FeeHandler:        txFeeHandler,
		HeaderSigVerifier: headerSigVerifier,
		ChainID:           chainID,
		ValidityAttester:  validityAttester,
		EpochStartTrigger: epochStartTrigger,
	}

	base := &baseInterceptorsContainerFactory{
		accounts:               accounts,
		shardCoordinator:       shardCoordinator,
		messenger:              messenger,
		store:                  store,
		marshalizer:            marshalizer,
		hasher:                 hasher,
		multiSigner:            multiSigner,
		dataPool:               dataPool,
		nodesCoordinator:       nodesCoordinator,
		argInterceptorFactory:  argInterceptorFactory,
		blackList:              blackList,
		maxTxNonceDeltaAllowed: maxTxNonceDeltaAllowed,
	}

	icf := &shardInterceptorsContainerFactory{
		baseInterceptorsContainerFactory: base,
		keyGen:                           keyGen,
		singleSigner:                     singleSigner,
		addrConverter:                    addrConverter,
	}

	icf.globalThrottler, err = throttler.NewNumGoRoutineThrottler(numGoRoutines)
	if err != nil {
		return nil, err
	}

	return icf, nil
}

// Create returns an interceptor container that will hold all interceptors in the system
func (sicf *shardInterceptorsContainerFactory) Create() (process.InterceptorsContainer, error) {
	container := containers.NewInterceptorsContainer()

	keys, interceptorSlice, err := sicf.generateTxInterceptors()
	if err != nil {
		return nil, err
	}

	err = container.AddMultiple(keys, interceptorSlice)
	if err != nil {
		return nil, err
	}

	keys, interceptorSlice, err = sicf.generateUnsignedTxsInterceptors()
	if err != nil {
		return nil, err
	}

	err = container.AddMultiple(keys, interceptorSlice)
	if err != nil {
		return nil, err
	}

	keys, interceptorSlice, err = sicf.generateRewardTxInterceptors()
	if err != nil {
		return nil, err
	}

	err = container.AddMultiple(keys, interceptorSlice)
	if err != nil {
		return nil, err
	}

	keys, interceptorSlice, err = sicf.generateHeaderInterceptors()
	if err != nil {
		return nil, err
	}

	err = container.AddMultiple(keys, interceptorSlice)
	if err != nil {
		return nil, err
	}

	keys, interceptorSlice, err = sicf.generateMiniBlocksInterceptors()
	if err != nil {
		return nil, err
	}

	err = container.AddMultiple(keys, interceptorSlice)
	if err != nil {
		return nil, err
	}

	keys, interceptorSlice, err = sicf.generateMetachainHeaderInterceptors()
	if err != nil {
		return nil, err
	}

	err = container.AddMultiple(keys, interceptorSlice)
	if err != nil {
		return nil, err
	}

	keys, interceptorSlice, err = sicf.generateTrieNodesInterceptors()
	if err != nil {
		return nil, err
	}

	err = container.AddMultiple(keys, interceptorSlice)
	if err != nil {
		return nil, err
	}

	return container, nil
}

//------- Unsigned transactions interceptors

func (sicf *shardInterceptorsContainerFactory) generateUnsignedTxsInterceptors() ([]string, []process.Interceptor, error) {
	shardC := sicf.shardCoordinator

	noOfShards := shardC.NumberOfShards()

	keys := make([]string, noOfShards)
	interceptorSlice := make([]process.Interceptor, noOfShards)

	for idx := uint32(0); idx < noOfShards; idx++ {
		identifierScr := factory.UnsignedTransactionTopic + shardC.CommunicationIdentifier(idx)

		interceptor, err := sicf.createOneUnsignedTxInterceptor(identifierScr)
		if err != nil {
			return nil, nil, err
		}

		keys[int(idx)] = identifierScr
		interceptorSlice[int(idx)] = interceptor
	}

	identifierTx := factory.UnsignedTransactionTopic + shardC.CommunicationIdentifier(sharding.MetachainShardId)

	interceptor, err := sicf.createOneUnsignedTxInterceptor(identifierTx)
	if err != nil {
		return nil, nil, err
	}

	keys = append(keys, identifierTx)
	interceptorSlice = append(interceptorSlice, interceptor)
	return keys, interceptorSlice, nil
}

func (sicf *shardInterceptorsContainerFactory) generateTrieNodesInterceptors() ([]string, []process.Interceptor, error) {
	shardC := sicf.shardCoordinator

	keys := make([]string, 0)
	interceptorSlice := make([]process.Interceptor, 0)

	identifierTrieNodes := factory.AccountTrieNodesTopic + shardC.CommunicationIdentifier(sharding.MetachainShardId)
	interceptor, err := sicf.createOneTrieNodesInterceptor(identifierTrieNodes)
	if err != nil {
		return nil, nil, err
	}

	keys = append(keys, identifierTrieNodes)
	interceptorSlice = append(interceptorSlice, interceptor)

	identifierTrieNodes = factory.ValidatorTrieNodesTopic + shardC.CommunicationIdentifier(sharding.MetachainShardId)
	interceptor, err = sicf.createOneTrieNodesInterceptor(identifierTrieNodes)
	if err != nil {
		return nil, nil, err
	}

	keys = append(keys, identifierTrieNodes)
	interceptorSlice = append(interceptorSlice, interceptor)

	return keys, interceptorSlice, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (sicf *shardInterceptorsContainerFactory) IsInterfaceNil() bool {
	return sicf == nil
}
