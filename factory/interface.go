package factory

import (
	"time"

	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/indexer"
	"github.com/ElrondNetwork/elrond-go/core/statistics"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/epochStart/bootstrap"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/heartbeat"
	heartbeatData "github.com/ElrondNetwork/elrond-go/heartbeat/data"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/ntp"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/update"
	"github.com/ElrondNetwork/elrond-go/vm"
)

// EpochStartNotifier defines which actions should be done for handling new epoch's events
type EpochStartNotifier interface {
	RegisterHandler(handler epochStart.ActionHandler)
	UnregisterHandler(handler epochStart.ActionHandler)
	NotifyAll(hdr data.HeaderHandler)
	NotifyAllPrepare(metaHdr data.HeaderHandler, body data.BodyHandler)
	NotifyEpochChangeConfirmed(epoch uint32)
	IsInterfaceNil() bool
}

// NodesSetupHandler defines which actions should be done for handling initial nodes setup
type NodesSetupHandler interface {
	InitialNodesPubKeys() map[uint32][]string
	InitialEligibleNodesPubKeysForShard(shardId uint32) ([]string, error)
	IsInterfaceNil() bool
}

// P2PAntifloodHandler defines the behavior of a component able to signal that the system is too busy (or flooded) processing
// p2p messages
type P2PAntifloodHandler interface {
	CanProcessMessage(message p2p.MessageP2P, fromConnectedPeer core.PeerID) error
	CanProcessMessagesOnTopic(peer core.PeerID, topic string, numMessages uint32, totalSize uint64, sequence []byte) error
	ResetForTopic(topic string)
	SetMaxMessagesForTopic(topic string, maxNum uint32)
	SetDebugger(debugger process.AntifloodDebugger) error
	SetPeerValidatorMapper(validatorMapper process.PeerValidatorMapper) error
	SetTopicsForAll(topics ...string)
	ApplyConsensusSize(size int)
	BlacklistPeer(peer core.PeerID, reason string, duration time.Duration)
	IsOriginatorEligibleForTopic(pid core.PeerID, topic string) error
	IsInterfaceNil() bool
}

// HeaderIntegrityVerifierHandler is the interface needed to check that a header's integrity is correct
type HeaderIntegrityVerifierHandler interface {
	Verify(header data.HeaderHandler) error
	IsInterfaceNil() bool
}

// Closer defines the Close behavior
type Closer interface {
	Close() error
}

// ComponentHandler defines the actions common to all component handlers
type ComponentHandler interface {
	Create() error
	Close() error
	CheckSubcomponents() error
}

// CoreComponentsHolder holds the core components
type CoreComponentsHolder interface {
	InternalMarshalizer() marshal.Marshalizer
	SetInternalMarshalizer(marshalizer marshal.Marshalizer) error
	TxMarshalizer() marshal.Marshalizer
	VmMarshalizer() marshal.Marshalizer
	Hasher() hashing.Hasher
	Uint64ByteSliceConverter() typeConverters.Uint64ByteSliceConverter
	AddressPubKeyConverter() core.PubkeyConverter
	ValidatorPubKeyConverter() core.PubkeyConverter
	StatusHandler() core.AppStatusHandler
	SetStatusHandler(statusHandler core.AppStatusHandler) error
	PathHandler() storage.PathManagerHandler
	Watchdog() core.WatchdogTimer
	AlarmScheduler() core.TimersScheduler
	SyncTimer() ntp.SyncTimer
	Rounder() consensus.Rounder
	EconomicsData() process.EconomicsHandler
	RatingsData() process.RatingsInfoHandler
	Rater() sharding.PeerAccountListAndRatingHandler
	GenesisNodesSetup() NodesSetupHandler
	GenesisTime() time.Time
	ChainID() string
	MinTransactionVersion() uint32
	IsInterfaceNil() bool
}

// CoreComponentsHandler defines the core components handler actions
type CoreComponentsHandler interface {
	ComponentHandler
	CoreComponentsHolder
}

// CryptoParamsHolder permits access to crypto parameters such as the private and public keys
type CryptoParamsHolder interface {
	PublicKey() crypto.PublicKey
	PrivateKey() crypto.PrivateKey
	PublicKeyString() string
	PublicKeyBytes() []byte
	PrivateKeyBytes() []byte
}

// CryptoComponentsHolder holds the crypto components
type CryptoComponentsHolder interface {
	CryptoParamsHolder
	TxSingleSigner() crypto.SingleSigner
	BlockSigner() crypto.SingleSigner
	MultiSigner() crypto.MultiSigner
	PeerSignatureHandler() crypto.PeerSignatureHandler
	SetMultiSigner(ms crypto.MultiSigner) error
	BlockSignKeyGen() crypto.KeyGenerator
	TxSignKeyGen() crypto.KeyGenerator
	MessageSignVerifier() vm.MessageSignVerifier
	IsInterfaceNil() bool
}

// KeyLoaderHandler defines the loading of a key from a pem file and index
type KeyLoaderHandler interface {
	LoadKey(string, int) ([]byte, string, error)
}

// CryptoComponentsHandler defines the crypto components handler actions
type CryptoComponentsHandler interface {
	ComponentHandler
	CryptoComponentsHolder
}

// MiniBlockProvider defines what a miniblock data provider should do
type MiniBlockProvider interface {
	GetMiniBlocks(hashes [][]byte) ([]*block.MiniblockAndHash, [][]byte)
	GetMiniBlocksFromPool(hashes [][]byte) ([]*block.MiniblockAndHash, [][]byte)
	IsInterfaceNil() bool
}

// DataComponentsHolder holds the data components
type DataComponentsHolder interface {
	Blockchain() data.ChainHandler
	SetBlockchain(chain data.ChainHandler)
	StorageService() dataRetriever.StorageService
	Datapool() dataRetriever.PoolsHolder
	MiniBlocksProvider() MiniBlockProvider
	Clone() interface{}
	IsInterfaceNil() bool
}

// DataComponentsHandler defines the data components handler actions
type DataComponentsHandler interface {
	ComponentHandler
	DataComponentsHolder
}

// PeerHonestyHandler defines the behaivour of a component able to handle/monitor the peer honesty of nodes which are
// participating in consensus
type PeerHonestyHandler interface {
	ChangeScore(pk string, topic string, units int)
	IsInterfaceNil() bool
}

// NetworkComponentsHolder holds the network components
type NetworkComponentsHolder interface {
	NetworkMessenger() p2p.Messenger
	InputAntiFloodHandler() P2PAntifloodHandler
	OutputAntiFloodHandler() P2PAntifloodHandler
	PubKeyCacher() process.TimeCacher
	PeerBlackListHandler() process.PeerBlackListCacher
	PeerHonestyHandler() PeerHonestyHandler
	IsInterfaceNil() bool
}

// NetworkComponentsHandler defines the network components handler actions
type NetworkComponentsHandler interface {
	ComponentHandler
	NetworkComponentsHolder
}

// ProcessComponentsHolder holds the process components
type ProcessComponentsHolder interface {
	NodesCoordinator() sharding.NodesCoordinator
	ShardCoordinator() sharding.Coordinator
	InterceptorsContainer() process.InterceptorsContainer
	ResolversFinder() dataRetriever.ResolversFinder
	Rounder() consensus.Rounder
	EpochStartTrigger() epochStart.TriggerHandler
	EpochStartNotifier() EpochStartNotifier
	ForkDetector() process.ForkDetector
	BlockProcessor() process.BlockProcessor
	BlackListHandler() process.TimeCacher
	BootStorer() process.BootStorer
	HeaderSigVerifier() process.InterceptedHeaderSigVerifier
	HeaderIntegrityVerifier() process.HeaderIntegrityVerifier
	ValidatorsStatistics() process.ValidatorStatisticsProcessor
	ValidatorsProvider() process.ValidatorsProvider
	BlockTracker() process.BlockTracker
	PendingMiniBlocksHandler() process.PendingMiniBlocksHandler
	RequestHandler() process.RequestHandler
	TxLogsProcessor() process.TransactionLogProcessorDatabase
	HeaderConstructionValidator() process.HeaderConstructionValidator
	PeerShardMapper() process.NetworkShardingCollector
	IsInterfaceNil() bool
}

// ProcessComponentsHandler defines the process components handler actions
type ProcessComponentsHandler interface {
	ComponentHandler
	ProcessComponentsHolder
}

// StateComponentsHandler
type StateComponentsHandler interface {
	ComponentHandler
	StateComponentsHolder
}

// StateComponentsHolder holds the
type StateComponentsHolder interface {
	PeerAccounts() state.AccountsAdapter
	AccountsAdapter() state.AccountsAdapter
	TriesContainer() state.TriesHolder
	TrieStorageManagers() map[string]data.StorageManager
	IsInterfaceNil() bool
}

// StatusHandlersUtils provides some functionality for statusHandlers
type StatusHandlersUtils interface {
	UpdateStorerAndMetricsForPersistentHandler(store storage.Storer) error
	LoadTpsBenchmarkFromStorage(store storage.Storer, marshalizer marshal.Marshalizer) *statistics.TpsPersistentData
	IsInterfaceNil() bool
}

// StatusComponentsHolder holds the status components
type StatusComponentsHolder interface {
	TpsBenchmark() statistics.TPSBenchmark
	ElasticIndexer() indexer.Indexer
	SoftwareVersionChecker() statistics.SoftwareVersionChecker
	StatusHandler() core.AppStatusHandler
	IsInterfaceNil() bool
}

// StatusComponentsHandler defines the status components handler actions
type StatusComponentsHandler interface {
	ComponentHandler
	StatusComponentsHolder
	// SetForkDetector should be set before starting Polling for updates
	SetForkDetector(forkDetector process.ForkDetector)
	StartPolling() error
}

// HeartbeatSender sends heartbeat messages
type HeartbeatSender interface {
	SendHeartbeat() error
	IsInterfaceNil() bool
}

// HeartbeatMonitor monitors the received heartbeat messages
type HeartbeatMonitor interface {
	SetAppStatusHandler(ash core.AppStatusHandler) error
	ProcessReceivedMessage(message p2p.MessageP2P, fromConnectedPeer core.PeerID) error
	GetHeartbeats() []heartbeatData.PubKeyHeartbeat
	IsInterfaceNil() bool
}

// HeartbeatStorer provides storage functionality for the heartbeat component
type HeartbeatStorer interface {
	UpdateGenesisTime(genesisTime time.Time) error
	LoadGenesisTime() (time.Time, error)
	SaveKeys(peersSlice [][]byte) error
	LoadKeys() ([][]byte, error)
	IsInterfaceNil() bool
}

// HeartbeatComponentsHolder holds the heartbeat components
type HeartbeatComponentsHolder interface {
	MessageHandler() heartbeat.MessageHandler
	Monitor() HeartbeatMonitor
	Sender() HeartbeatSender
	Storer() HeartbeatStorer
	IsInterfaceNil() bool
}

// HeartbeatComponentsHandler defines the heartbeat components handler actions
type HeartbeatComponentsHandler interface {
	ComponentHandler
	HeartbeatComponentsHolder
}

// ConsensusWorker is the consensus worker handle for the exported functionality
type ConsensusWorker interface {
	Close() error
	StartWorking()
	//AddReceivedMessageCall adds a new handler function for a received message type
	AddReceivedMessageCall(messageType consensus.MessageType, receivedMessageCall func(cnsDta *consensus.Message) bool)
	//AddReceivedHeaderHandler adds a new handler function for a received header
	AddReceivedHeaderHandler(handler func(data.HeaderHandler))
	//RemoveAllReceivedMessagesCalls removes all the functions handlers
	RemoveAllReceivedMessagesCalls()
	//ProcessReceivedMessage method redirects the received message to the channel which should handle it
	ProcessReceivedMessage(message p2p.MessageP2P, fromConnectedPeer core.PeerID) error
	//Extend does an extension for the subround with subroundId
	Extend(subroundId int)
	//GetConsensusStateChangedChannel gets the channel for the consensusStateChanged
	GetConsensusStateChangedChannel() chan bool
	//ExecuteStoredMessages tries to execute all the messages received which are valid for execution
	ExecuteStoredMessages()
	//DisplayStatistics method displays statistics of worker at the end of the round
	DisplayStatistics()
	//ReceivedHeader method is a wired method through which worker will receive headers from network
	ReceivedHeader(headerHandler data.HeaderHandler, headerHash []byte)
	//SetAppStatusHandler sets the status handler object used to collect useful metrics about consensus state machine
	SetAppStatusHandler(ash core.AppStatusHandler) error
	// IsInterfaceNil returns true if there is no value under the interface
	IsInterfaceNil() bool
}

type HardforkTrigger interface {
	TriggerReceived(payload []byte, data []byte, pkBytes []byte) (bool, error)
	RecordedTriggerMessage() ([]byte, bool)
	Trigger(epoch uint32) error
	CreateData() []byte
	AddCloser(closer update.Closer) error
	NotifyTriggerReceived() <-chan struct{}
	IsSelfTrigger() bool
	IsInterfaceNil() bool
}

// ConsensusComponentsHolder holds the consensus components
type ConsensusComponentsHolder interface {
	Chronology() consensus.ChronologyHandler
	ConsensusWorker() ConsensusWorker
	BroadcastMessenger() consensus.BroadcastMessenger
	IsInterfaceNil() bool
}

// ConsensusComponentsHandler defines the consensus components handler actions
type ConsensusComponentsHandler interface {
	ComponentHandler
	ConsensusComponentsHolder
}

// BootstrapParamsHandler gives read access to parameters after bootstrap
type BootstrapParamsHandler interface {
	Epoch() uint32
	SelfShardID() uint32
	NumOfShards() uint32
	NodesConfig() *sharding.NodesCoordinatorRegistry
	IsInterfaceNil() bool
}

type EpochStartBootstrapper interface {
	GetTriesComponents() (state.TriesHolder, map[string]data.StorageManager)
	Bootstrap() (bootstrap.Parameters, error)
	IsInterfaceNil() bool
}

// BootstrapComponentsHolder holds the bootstrap components
type BootstrapComponentsHolder interface {
	EpochStartBootstrapper() EpochStartBootstrapper
	EpochBootstrapParams() BootstrapParamsHandler
	IsInterfaceNil() bool
}

// BootstrapComponentsHandler defines the bootstrap components handler actions
type BootstrapComponentsHandler interface {
	ComponentHandler
	BootstrapComponentsHolder
}
