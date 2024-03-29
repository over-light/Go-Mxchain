package factory

import (
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/p2p"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type interceptedPeerShardFactory struct {
	marshaller       marshal.Marshalizer
	shardCoordinator sharding.Coordinator
}

// NewInterceptedPeerShardFactory creates an instance of interceptedPeerShardFactory
func NewInterceptedPeerShardFactory(args ArgInterceptedDataFactory) (*interceptedPeerShardFactory, error) {
	err := checkArgs(args)
	if err != nil {
		return nil, err
	}

	return &interceptedPeerShardFactory{
		marshaller:       args.CoreComponents.InternalMarshalizer(),
		shardCoordinator: args.ShardCoordinator,
	}, nil
}

func checkArgs(args ArgInterceptedDataFactory) error {
	if check.IfNil(args.CoreComponents) {
		return process.ErrNilCoreComponentsHolder
	}
	if check.IfNil(args.CoreComponents.InternalMarshalizer()) {
		return process.ErrNilMarshalizer
	}
	if check.IfNil(args.ShardCoordinator) {
		return process.ErrNilShardCoordinator
	}

	return nil
}

// Create creates instances of InterceptedData by unmarshalling provided buffer
func (ipsf *interceptedPeerShardFactory) Create(buff []byte) (process.InterceptedData, error) {
	args := p2p.ArgInterceptedPeerShard{
		Marshaller:  ipsf.marshaller,
		DataBuff:    buff,
		NumOfShards: ipsf.shardCoordinator.NumberOfShards(),
	}

	return p2p.NewInterceptedPeerShard(args)
}

// IsInterfaceNil returns true if there is no value under the interface
func (ipsf *interceptedPeerShardFactory) IsInterfaceNil() bool {
	return ipsf == nil
}
