package factory

import (
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/unsigned"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type interceptedUnsignedTxDataFactory struct {
	protoMarshalizer marshal.Marshalizer
	hasher           hashing.Hasher
	pubkeyConverter  state.PubkeyConverter
	shardCoordinator sharding.Coordinator
}

// NewInterceptedUnsignedTxDataFactory creates an instance of interceptedUnsignedTxDataFactory
func NewInterceptedUnsignedTxDataFactory(argument *ArgInterceptedDataFactory) (*interceptedUnsignedTxDataFactory, error) {
	if argument == nil {
		return nil, process.ErrNilArgumentStruct
	}
	if check.IfNil(argument.ProtoMarshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if check.IfNil(argument.TxSignMarshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if check.IfNil(argument.Hasher) {
		return nil, process.ErrNilHasher
	}
	if check.IfNil(argument.AddressPubkeyConv) {
		return nil, process.ErrNilPubkeyConverter
	}
	if check.IfNil(argument.ShardCoordinator) {
		return nil, process.ErrNilShardCoordinator
	}

	return &interceptedUnsignedTxDataFactory{
		protoMarshalizer: argument.ProtoMarshalizer,
		hasher:           argument.Hasher,
		pubkeyConverter:  argument.AddressPubkeyConv,
		shardCoordinator: argument.ShardCoordinator,
	}, nil
}

// Create creates instances of InterceptedData by unmarshalling provided buffer
func (iutdf *interceptedUnsignedTxDataFactory) Create(buff []byte) (process.InterceptedData, error) {
	return unsigned.NewInterceptedUnsignedTransaction(
		buff,
		iutdf.protoMarshalizer,
		iutdf.hasher,
		iutdf.pubkeyConverter,
		iutdf.shardCoordinator,
	)
}

// IsInterfaceNil returns true if there is no value under the interface
func (iutdf *interceptedUnsignedTxDataFactory) IsInterfaceNil() bool {
	return iutdf == nil
}
