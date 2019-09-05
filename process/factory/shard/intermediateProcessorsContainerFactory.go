package shard

import (
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block/preprocess"
	"github.com/ElrondNetwork/elrond-go/process/factory/containers"
	"github.com/ElrondNetwork/elrond-go/process/unsigned"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type intermediateProcessorsContainerFactory struct {
	shardCoordinator      sharding.Coordinator
	marshalizer           marshal.Marshalizer
	hasher                hashing.Hasher
	addrConverter         state.AddressConverter
	specialAddressHandler process.SpecialAddressHandler
	store            dataRetriever.StorageService
}

// NewIntermediateProcessorsContainerFactory is responsible for creating a new intermediate processors factory object
func NewIntermediateProcessorsContainerFactory(
	shardCoordinator sharding.Coordinator,
	marshalizer marshal.Marshalizer,
	hasher hashing.Hasher,
	addrConverter state.AddressConverter,
	specialAddressHandler process.SpecialAddressHandler,
	store dataRetriever.StorageService,
) (*intermediateProcessorsContainerFactory, error) {

	if shardCoordinator == nil || shardCoordinator.IsInterfaceNil() {
		return nil, process.ErrNilShardCoordinator
	}
	if marshalizer == nil || marshalizer.IsInterfaceNil() {
		return nil, process.ErrNilMarshalizer
	}
	if hasher == nil || hasher.IsInterfaceNil() {
		return nil, process.ErrNilHasher
	}
	if addrConverter == nil || addrConverter.IsInterfaceNil() {
		return nil, process.ErrNilAddressConverter
	}
	if specialAddressHandler == nil || specialAddressHandler.IsInterfaceNil(){
		return nil, process.ErrNilSpecialAddressHandler
	}
	if store == nil || store.IsInterfaceNil(){
		return nil, process.ErrNilStorage
	}

	return &intermediateProcessorsContainerFactory{
		shardCoordinator:      shardCoordinator,
		marshalizer:           marshalizer,
		hasher:                hasher,
		addrConverter:         addrConverter,
		specialAddressHandler: specialAddressHandler,
		store:            store,
	}, nil
}

// Create returns a preprocessor container that will hold all preprocessors in the system
func (ppcm *intermediateProcessorsContainerFactory) Create() (process.IntermediateProcessorContainer, error) {
	container := containers.NewIntermediateTransactionHandlersContainer()

	interproc, err := ppcm.createSmartContractResultsIntermediateProcessor()
	if err != nil {
		return nil, err
	}

	err = container.Add(block.SmartContractResultBlock, interproc)
	if err != nil {
		return nil, err
	}

	interproc, err = ppcm.createTxFeeIntermediateProcessor()
	if err != nil {
		return nil, err
	}

	err = container.Add(block.TxFeeBlock, interproc)
	if err != nil {
		return nil, err
	}

	return container, nil
}

func (ppcm *intermediateProcessorsContainerFactory) createSmartContractResultsIntermediateProcessor() (process.IntermediateTransactionHandler, error) {
	irp, err := preprocess.NewIntermediateResultsProcessor(
		ppcm.hasher,
		ppcm.marshalizer,
		ppcm.shardCoordinator,
		ppcm.addrConverter,
		ppcm.store,
		block.SmartContractResultBlock,
	)

	return irp, err
}

func (ppcm *intermediateProcessorsContainerFactory) createTxFeeIntermediateProcessor() (process.IntermediateTransactionHandler, error) {
	irp, err := unsigned.NewFeeTxHandler(
		ppcm.specialAddressHandler,
		ppcm.shardCoordinator,
		ppcm.hasher,
		ppcm.marshalizer,
	)

	return irp, err
}

// IsInterfaceNil returns true if there is no value under the interface
func (ppcm *intermediateProcessorsContainerFactory) IsInterfaceNil() bool {
	if ppcm == nil {
		return true
	}
	return false
}