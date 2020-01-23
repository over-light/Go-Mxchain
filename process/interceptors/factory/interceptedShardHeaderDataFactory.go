package factory

import (
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block/interceptedBlocks"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type interceptedShardHeaderDataFactory struct {
	marshalizer       marshal.Marshalizer
	hasher            hashing.Hasher
	shardCoordinator  sharding.Coordinator
	headerSigVerifier process.InterceptedHeaderSigVerifier
	chainID           []byte
	finalityAttester  process.FinalityAttester
}

// NewInterceptedShardHeaderDataFactory creates an instance of interceptedShardHeaderDataFactory
func NewInterceptedShardHeaderDataFactory(argument *ArgInterceptedDataFactory) (*interceptedShardHeaderDataFactory, error) {
	if argument == nil {
		return nil, process.ErrNilArgumentStruct
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
	if check.IfNil(argument.HeaderSigVerifier) {
		return nil, process.ErrNilHeaderSigVerifier
	}
	if len(argument.ChainID) == 0 {
		return nil, process.ErrInvalidChainID
	}

	return &interceptedShardHeaderDataFactory{
		marshalizer:       argument.Marshalizer,
		hasher:            argument.Hasher,
		shardCoordinator:  argument.ShardCoordinator,
		headerSigVerifier: argument.HeaderSigVerifier,
		chainID:           argument.ChainID,
		finalityAttester:  &nilFinalityAttester{},
	}, nil
}

// Create creates instances of InterceptedData by unmarshalling provided buffer
func (ishdf *interceptedShardHeaderDataFactory) Create(buff []byte) (process.InterceptedData, error) {
	arg := &interceptedBlocks.ArgInterceptedBlockHeader{
		HdrBuff:           buff,
		Marshalizer:       ishdf.marshalizer,
		Hasher:            ishdf.hasher,
		ShardCoordinator:  ishdf.shardCoordinator,
		HeaderSigVerifier: ishdf.headerSigVerifier,
		ChainID:           ishdf.chainID,
		FinalityAttester:  ishdf.finalityAttester,
	}

	return interceptedBlocks.NewInterceptedHeader(arg)
}

// SetFinalityAttester sets the finality attester
func (ishdf *interceptedShardHeaderDataFactory) SetFinalityAttester(attester process.FinalityAttester) error {
	if check.IfNil(attester) {
		return process.ErrNilFinalityAttester
	}

	ishdf.finalityAttester = attester
	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (ishdf *interceptedShardHeaderDataFactory) IsInterfaceNil() bool {
	return ishdf == nil
}
