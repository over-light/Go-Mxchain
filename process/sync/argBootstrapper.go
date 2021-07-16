package sync

import (
	"time"

	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go/state"
	"github.com/ElrondNetwork/elrond-go-core/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// ArgBaseBootstrapper holds all dependencies required by the bootstrap data factory in order to create
// new instances
type ArgBaseBootstrapper struct {
	PoolsHolder          dataRetriever.PoolsHolder
	Store                dataRetriever.StorageService
	ChainHandler         data.ChainHandler
	RoundHandler         consensus.RoundHandler
	BlockProcessor       process.BlockProcessor
	WaitTime             time.Duration
	Hasher               hashing.Hasher
	Marshalizer          marshal.Marshalizer
	ForkDetector         process.ForkDetector
	RequestHandler       process.RequestHandler
	ShardCoordinator     sharding.Coordinator
	Accounts             state.AccountsAdapter
	BlackListHandler     process.TimeCacher
	NetworkWatcher       process.NetworkConnectionWatcher
	BootStorer           process.BootStorer
	StorageBootstrapper  process.BootstrapperFromStorage
	EpochHandler         dataRetriever.EpochHandler
	MiniblocksProvider   process.MiniBlockProvider
	Uint64Converter      typeConverters.Uint64ByteSliceConverter
	AppStatusHandler     core.AppStatusHandler
	Indexer              process.Indexer
	AccountsDBSyncer     process.AccountsDBSyncer
	CurrentEpochProvider process.CurrentNetworkEpochProviderHandler
	IsInImportMode       bool
}

// ArgShardBootstrapper holds all dependencies required by the bootstrap data factory in order to create
// new instances of shard bootstrapper
type ArgShardBootstrapper struct {
	ArgBaseBootstrapper
}

// ArgMetaBootstrapper holds all dependencies required by the bootstrap data factory in order to create
// new instances of meta bootstrapper
type ArgMetaBootstrapper struct {
	ArgBaseBootstrapper
	EpochBootstrapper           process.EpochBootstrapper
	ValidatorStatisticsDBSyncer process.AccountsDBSyncer
	ValidatorAccountsDB         state.AccountsAdapter
}
