package factory

import (
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block/interceptedBlocks"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type metaInterceptedDataFactory struct {
	marshalizer         marshal.Marshalizer
	hasher              hashing.Hasher
	keyGen              crypto.KeyGenerator
	singleSigner        crypto.SingleSigner
	addrConverter       state.AddressConverter
	shardCoordinator    sharding.Coordinator
	interceptedDataType InterceptedDataType
	multiSigVerifier    crypto.MultiSigVerifier
	chronologyValidator process.ChronologyValidator
}

// NewMetaInterceptedDataFactory creates an instance of interceptedDataFactory that can create
// instances of process.InterceptedData and is used on meta nodes
func NewMetaInterceptedDataFactory(
	argument *ArgMetaInterceptedDataFactory,
	dataType InterceptedDataType,
) (*metaInterceptedDataFactory, error) {

	if argument == nil {
		return nil, process.ErrNilArguments
	}
	if check.IfNil(argument.Marshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if check.IfNil(argument.Hasher) {
		return nil, process.ErrNilHasher
	}
	if check.IfNil(argument.ShardCoordinator) {
		return nil, process.ErrNilShardCoordinator
	}
	if check.IfNil(argument.MultiSigVerifier) {
		return nil, process.ErrNilMultiSigVerifier
	}
	if check.IfNil(argument.ChronologyValidator) {
		return nil, process.ErrNilChronologyValidator
	}

	return &metaInterceptedDataFactory{
		marshalizer:         argument.Marshalizer,
		hasher:              argument.Hasher,
		shardCoordinator:    argument.ShardCoordinator,
		interceptedDataType: dataType,
		multiSigVerifier:    argument.MultiSigVerifier,
		chronologyValidator: argument.ChronologyValidator,
	}, nil
}

// Create creates instances of InterceptedData by unmarshalling provided buffer
// The type of the output instance is provided in the constructor
func (midf *metaInterceptedDataFactory) Create(buff []byte) (process.InterceptedData, error) {
	switch midf.interceptedDataType {
	case InterceptedShardHeader:
		return midf.createInterceptedShardHeader(buff)
	default:
		return nil, process.ErrInterceptedDataTypeNotDefined
	}
}

func (midf *metaInterceptedDataFactory) createInterceptedShardHeader(buff []byte) (process.InterceptedData, error) {
	arg := &interceptedBlocks.ArgInterceptedBlockHeader{
		HdrBuff:             buff,
		Marshalizer:         midf.marshalizer,
		Hasher:              midf.hasher,
		MultiSigVerifier:    midf.multiSigVerifier,
		ChronologyValidator: midf.chronologyValidator,
		ShardCoordinator:    midf.shardCoordinator,
	}

	return interceptedBlocks.NewInterceptedHeader(arg)
}

// IsInterfaceNil returns true if there is no value under the interface
func (midf *metaInterceptedDataFactory) IsInterfaceNil() bool {
	if midf == nil {
		return true
	}
	return false
}
