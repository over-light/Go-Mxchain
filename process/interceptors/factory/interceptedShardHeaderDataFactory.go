package factory

import (
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/crypto"
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
	singleSigVerifier crypto.SingleSigner
	multiSigVerifier  crypto.MultiSigVerifier
	nodesCoordinator  sharding.NodesCoordinator
	keyGen            crypto.KeyGenerator
	chainID           []byte
}

// NewInterceptedShardHeaderDataFactory creates an instance of interceptedShardHeaderDataFactory
func NewInterceptedShardHeaderDataFactory(argument *ArgInterceptedDataFactory) (*interceptedShardHeaderDataFactory, error) {
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
	if check.IfNil(argument.NodesCoordinator) {
		return nil, process.ErrNilNodesCoordinator
	}
	if check.IfNil(argument.BlockSigner) {
		return nil, process.ErrNilSingleSigner
	}
	if check.IfNil(argument.BlockKeyGen) {
		return nil, process.ErrNilKeyGen
	}
	if len(argument.ChainID) == 0 {
		return nil, process.ErrInvalidChainID
	}

	return &interceptedShardHeaderDataFactory{
		marshalizer:       argument.Marshalizer,
		hasher:            argument.Hasher,
		shardCoordinator:  argument.ShardCoordinator,
		multiSigVerifier:  argument.MultiSigVerifier,
		nodesCoordinator:  argument.NodesCoordinator,
		singleSigVerifier: argument.BlockSigner,
		keyGen:            argument.BlockKeyGen,
		chainID:           argument.ChainID,
	}, nil
}

// Create creates instances of InterceptedData by unmarshalling provided buffer
func (ishdf *interceptedShardHeaderDataFactory) Create(buff []byte) (process.InterceptedData, error) {
	arg := &interceptedBlocks.ArgInterceptedBlockHeader{
		HdrBuff:           buff,
		Marshalizer:       ishdf.marshalizer,
		Hasher:            ishdf.hasher,
		SingleSigVerifier: ishdf.singleSigVerifier,
		MultiSigVerifier:  ishdf.multiSigVerifier,
		NodesCoordinator:  ishdf.nodesCoordinator,
		ShardCoordinator:  ishdf.shardCoordinator,
		KeyGen:            ishdf.keyGen,
		ChainID:           ishdf.chainID,
	}

	return interceptedBlocks.NewInterceptedHeader(arg)
}

// IsInterfaceNil returns true if there is no value under the interface
func (ishdf *interceptedShardHeaderDataFactory) IsInterfaceNil() bool {
	if ishdf == nil {
		return true
	}
	return false
}
