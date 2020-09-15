package indexer

import (
	"bytes"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

//ArgDataIndexer is struct that is used to store all components that are needed to create a indexer
type ArgDataIndexer struct {
	TxIndexingEnabled  bool
	ShardID            uint32
	Marshalizer        marshal.Marshalizer
	EpochStartNotifier sharding.EpochStartEventNotifier
	NodesCoordinator   sharding.NodesCoordinator
	Options            *Options
	DataDispatcher     DispatcherHandler
	ElasticProcessor   ElasticProcessor
}

//ArgElasticProcessor is struct that is used to store all components that are needed to an elastic indexer
type ArgElasticProcessor struct {
	IndexTemplates           map[string]*bytes.Buffer
	IndexPolicies            map[string]*bytes.Buffer
	Marshalizer              marshal.Marshalizer
	Hasher                   hashing.Hasher
	AddressPubkeyConverter   core.PubkeyConverter
	ValidatorPubkeyConverter core.PubkeyConverter
	Options                  *Options
	DBClient                 DatabaseClientHandler
}
