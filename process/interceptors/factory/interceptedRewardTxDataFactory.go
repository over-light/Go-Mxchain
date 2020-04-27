package factory

import (
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/rewardTransaction"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type interceptedRewardTxDataFactory struct {
	protoMarshalizer marshal.Marshalizer
	hasher           hashing.Hasher
	pubkeyConverter  state.PubkeyConverter
	shardCoordinator sharding.Coordinator
}

// NewInterceptedRewardTxDataFactory creates an instance of interceptedRewardTxDataFactory
func NewInterceptedRewardTxDataFactory(argument *ArgInterceptedDataFactory) (*interceptedRewardTxDataFactory, error) {
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

	return &interceptedRewardTxDataFactory{
		protoMarshalizer: argument.ProtoMarshalizer,
		hasher:           argument.Hasher,
		pubkeyConverter:  argument.AddressPubkeyConv,
		shardCoordinator: argument.ShardCoordinator,
	}, nil
}

// Create creates instances of InterceptedData by unmarshalling provided buffer
func (irtdf *interceptedRewardTxDataFactory) Create(buff []byte) (process.InterceptedData, error) {
	return rewardTransaction.NewInterceptedRewardTransaction(
		buff,
		irtdf.protoMarshalizer,
		irtdf.hasher,
		irtdf.pubkeyConverter,
		irtdf.shardCoordinator,
	)
}

// IsInterfaceNil returns true if there is no value under the interface
func (irtdf *interceptedRewardTxDataFactory) IsInterfaceNil() bool {
	return irtdf == nil
}
