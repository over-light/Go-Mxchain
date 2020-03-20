package resolverscontainer

import (
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// FactoryArgs will hold the arguments for ResolversContainerFactory for both shard and meta
type FactoryArgs struct {
	ShardCoordinator           sharding.Coordinator
	Messenger                  dataRetriever.TopicMessageHandler
	Store                      dataRetriever.StorageService
	Marshalizer                marshal.Marshalizer
	DataPools                  dataRetriever.PoolsHolder
	Uint64ByteSliceConverter   typeConverters.Uint64ByteSliceConverter
	DataPacker                 dataRetriever.DataPacker
	TriesContainer             state.TriesHolder
	SizeCheckDelta             uint32
	InputAntifloodHandler      dataRetriever.P2PAntifloodHandler
	OutputAntifloodHandler     dataRetriever.P2PAntifloodHandler
	NumConcurrentResolvingJobs int32
}
