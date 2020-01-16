package containers

import (
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// resolversFinder is an implementation of process.ResolverContainer meant to be used
// wherever a resolver fetch is required
type resolversFinder struct {
	dataRetriever.ResolversContainer
	coordinator sharding.Coordinator
}

// NewResolversFinder creates a new resolversFinder object
func NewResolversFinder(container dataRetriever.ResolversContainer, coordinator sharding.Coordinator) (*resolversFinder, error) {
	if container == nil || container.IsInterfaceNil() {
		return nil, dataRetriever.ErrNilResolverContainer
	}

	if coordinator == nil || coordinator.IsInterfaceNil() {
		return nil, dataRetriever.ErrNilShardCoordinator
	}

	return &resolversFinder{
		ResolversContainer: container,
		coordinator:        coordinator,
	}, nil
}

// IntraShardResolver fetches the intrashard Resolver starting from a baseTopic
// baseTopic will be one of the constants defined in factory.go: TransactionTopic, HeadersTopic and so on
func (rf *resolversFinder) IntraShardResolver(baseTopic string) (dataRetriever.Resolver, error) {
	topic := baseTopic + rf.coordinator.CommunicationIdentifier(rf.coordinator.SelfId())
	return rf.Get(topic)
}

// MetaChainResolver fetches the metachain Resolver starting from a baseTopic
// baseTopic will be one of the constants defined in factory.go: metaHeaderTopic, MetaPeerChangeTopic and so on
func (rf *resolversFinder) MetaChainResolver(baseTopic string) (dataRetriever.Resolver, error) {
	return rf.Get(baseTopic)
}

// CrossShardResolver fetches the cross shard Resolver starting from a baseTopic and a cross shard id
// baseTopic will be one of the constants defined in factory.go: TransactionTopic, HeadersTopic and so on
func (rf *resolversFinder) CrossShardResolver(baseTopic string, crossShard uint32) (dataRetriever.Resolver, error) {
	topic := baseTopic + rf.coordinator.CommunicationIdentifier(crossShard)
	return rf.Get(topic)
}

// IsInterfaceNil returns true if underlying struct is nil
func (rf *resolversFinder) IsInterfaceNil() bool {
	return rf == nil || check.IfNil(rf.ResolversContainer)
}
