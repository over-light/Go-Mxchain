package mock

import (
	"github.com/multiversx/mx-chain-go/consensus"
	"github.com/multiversx/mx-chain-go/dataRetriever"
	"github.com/multiversx/mx-chain-go/dblookupext"
	"github.com/multiversx/mx-chain-go/epochStart"
	"github.com/multiversx/mx-chain-go/factory"
	"github.com/multiversx/mx-chain-go/genesis"
	"github.com/multiversx/mx-chain-go/process"
	"github.com/multiversx/mx-chain-go/sharding"
	"github.com/multiversx/mx-chain-go/sharding/nodesCoordinator"
	"github.com/multiversx/mx-chain-go/update"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

// ProcessComponentsStub -
type ProcessComponentsStub struct {
	NodesCoord                           nodesCoordinator.NodesCoordinator
	ShardCoord                           sharding.Coordinator
	IntContainer                         process.InterceptorsContainer
	ResFinder                            dataRetriever.ResolversFinder
	RoundHandlerField                    consensus.RoundHandler
	EpochTrigger                         epochStart.TriggerHandler
	EpochNotifier                        factory.EpochStartNotifier
	ForkDetect                           process.ForkDetector
	BlockProcess                         process.BlockProcessor
	BlackListHdl                         process.TimeCacher
	BootSore                             process.BootStorer
	HeaderSigVerif                       process.InterceptedHeaderSigVerifier
	HeaderIntegrVerif                    process.HeaderIntegrityVerifier
	ValidatorStatistics                  process.ValidatorStatisticsProcessor
	ValidatorProvider                    process.ValidatorsProvider
	BlockTrack                           process.BlockTracker
	PendingMiniBlocksHdl                 process.PendingMiniBlocksHandler
	ReqHandler                           process.RequestHandler
	TxLogsProcess                        process.TransactionLogProcessorDatabase
	HeaderConstructValidator             process.HeaderConstructionValidator
	PeerMapper                           process.NetworkShardingCollector
	TxSimulatorProcessor                 factory.TransactionSimulatorProcessor
	FallbackHdrValidator                 process.FallbackHeaderValidator
	WhiteListHandlerInternal             process.WhiteListHandler
	WhiteListerVerifiedTxsInternal       process.WhiteListHandler
	HistoryRepositoryInternal            dblookupext.HistoryRepository
	ImportStartHandlerInternal           update.ImportStartHandler
	RequestedItemsHandlerInternal        dataRetriever.RequestedItemsHandler
	NodeRedundancyHandlerInternal        consensus.NodeRedundancyHandler
	AccountsParserInternal               genesis.AccountsParser
	CurrentEpochProviderInternal         process.CurrentNetworkEpochProviderHandler
	ScheduledTxsExecutionHandlerInternal process.ScheduledTxsExecutionHandler
	TxsSenderHandlerField                process.TxsSenderHandler
	HardforkTriggerField                 factory.HardforkTrigger
	ProcessedMiniBlocksTrackerInternal   process.ProcessedMiniBlocksTracker
	ReceiptsRepositoryInternal           factory.ReceiptsRepository
	ESDTDataStorageHandlerForAPIInternal vmcommon.ESDTNFTStorageHandler
}

// Create -
func (pcs *ProcessComponentsStub) Create() error {
	return nil
}

// Close -
func (pcs *ProcessComponentsStub) Close() error {
	return nil
}

// CheckSubcomponents -
func (pcs *ProcessComponentsStub) CheckSubcomponents() error {
	return nil
}

// NodesCoordinator -
func (pcs *ProcessComponentsStub) NodesCoordinator() nodesCoordinator.NodesCoordinator {
	return pcs.NodesCoord
}

// ShardCoordinator -
func (pcs *ProcessComponentsStub) ShardCoordinator() sharding.Coordinator {
	return pcs.ShardCoord
}

// InterceptorsContainer -
func (pcs *ProcessComponentsStub) InterceptorsContainer() process.InterceptorsContainer {
	return pcs.IntContainer
}

// ResolversFinder -
func (pcs *ProcessComponentsStub) ResolversFinder() dataRetriever.ResolversFinder {
	return pcs.ResFinder
}

// RoundHandler -
func (pcs *ProcessComponentsStub) RoundHandler() consensus.RoundHandler {
	return pcs.RoundHandlerField
}

// EpochStartTrigger -
func (pcs *ProcessComponentsStub) EpochStartTrigger() epochStart.TriggerHandler {
	return pcs.EpochTrigger
}

// EpochStartNotifier -
func (pcs *ProcessComponentsStub) EpochStartNotifier() factory.EpochStartNotifier {
	return pcs.EpochNotifier
}

// ForkDetector -
func (pcs *ProcessComponentsStub) ForkDetector() process.ForkDetector {
	return pcs.ForkDetect
}

// BlockProcessor -
func (pcs *ProcessComponentsStub) BlockProcessor() process.BlockProcessor {
	return pcs.BlockProcess
}

// BlackListHandler -
func (pcs *ProcessComponentsStub) BlackListHandler() process.TimeCacher {
	return pcs.BlackListHdl
}

// BootStorer -
func (pcs *ProcessComponentsStub) BootStorer() process.BootStorer {
	return pcs.BootSore
}

// HeaderSigVerifier -
func (pcs *ProcessComponentsStub) HeaderSigVerifier() process.InterceptedHeaderSigVerifier {
	return pcs.HeaderSigVerif
}

// HeaderIntegrityVerifier -
func (pcs *ProcessComponentsStub) HeaderIntegrityVerifier() process.HeaderIntegrityVerifier {
	return pcs.HeaderIntegrVerif
}

// ValidatorsStatistics -
func (pcs *ProcessComponentsStub) ValidatorsStatistics() process.ValidatorStatisticsProcessor {
	return pcs.ValidatorStatistics
}

// ValidatorsProvider -
func (pcs *ProcessComponentsStub) ValidatorsProvider() process.ValidatorsProvider {
	return pcs.ValidatorProvider
}

// BlockTracker -
func (pcs *ProcessComponentsStub) BlockTracker() process.BlockTracker {
	return pcs.BlockTrack
}

// PendingMiniBlocksHandler -
func (pcs *ProcessComponentsStub) PendingMiniBlocksHandler() process.PendingMiniBlocksHandler {
	return pcs.PendingMiniBlocksHdl
}

// RequestHandler -
func (pcs *ProcessComponentsStub) RequestHandler() process.RequestHandler {
	return pcs.ReqHandler
}

// TxLogsProcessor -
func (pcs *ProcessComponentsStub) TxLogsProcessor() process.TransactionLogProcessorDatabase {
	return pcs.TxLogsProcess
}

// HeaderConstructionValidator -
func (pcs *ProcessComponentsStub) HeaderConstructionValidator() process.HeaderConstructionValidator {
	return pcs.HeaderConstructValidator
}

// PeerShardMapper -
func (pcs *ProcessComponentsStub) PeerShardMapper() process.NetworkShardingCollector {
	return pcs.PeerMapper
}

// FallbackHeaderValidator -
func (pcs *ProcessComponentsStub) FallbackHeaderValidator() process.FallbackHeaderValidator {
	return pcs.FallbackHdrValidator
}

// TransactionSimulatorProcessor -
func (pcs *ProcessComponentsStub) TransactionSimulatorProcessor() factory.TransactionSimulatorProcessor {
	return pcs.TxSimulatorProcessor
}

// WhiteListHandler -
func (pcs *ProcessComponentsStub) WhiteListHandler() process.WhiteListHandler {
	return pcs.WhiteListHandlerInternal
}

// WhiteListerVerifiedTxs -
func (pcs *ProcessComponentsStub) WhiteListerVerifiedTxs() process.WhiteListHandler {
	return pcs.WhiteListerVerifiedTxsInternal
}

// HistoryRepository -
func (pcs *ProcessComponentsStub) HistoryRepository() dblookupext.HistoryRepository {
	return pcs.HistoryRepositoryInternal
}

// ImportStartHandler -
func (pcs *ProcessComponentsStub) ImportStartHandler() update.ImportStartHandler {
	return pcs.ImportStartHandlerInternal
}

// RequestedItemsHandler -
func (pcs *ProcessComponentsStub) RequestedItemsHandler() dataRetriever.RequestedItemsHandler {
	return pcs.RequestedItemsHandlerInternal
}

// NodeRedundancyHandler -
func (pcs *ProcessComponentsStub) NodeRedundancyHandler() consensus.NodeRedundancyHandler {
	return pcs.NodeRedundancyHandlerInternal
}

// AccountsParser -
func (pcs *ProcessComponentsStub) AccountsParser() genesis.AccountsParser {
	return pcs.AccountsParserInternal
}

// CurrentEpochProvider -
func (pcs *ProcessComponentsStub) CurrentEpochProvider() process.CurrentNetworkEpochProviderHandler {
	return pcs.CurrentEpochProviderInternal
}

// String -
func (pcs *ProcessComponentsStub) String() string {
	return "ProcessComponentsStub"
}

// ScheduledTxsExecutionHandler -
func (pcs *ProcessComponentsStub) ScheduledTxsExecutionHandler() process.ScheduledTxsExecutionHandler {
	return pcs.ScheduledTxsExecutionHandlerInternal
}

// TxsSenderHandler -
func (pcs *ProcessComponentsStub) TxsSenderHandler() process.TxsSenderHandler {
	return pcs.TxsSenderHandlerField
}

// HardforkTrigger -
func (pcs *ProcessComponentsStub) HardforkTrigger() factory.HardforkTrigger {
	return pcs.HardforkTriggerField
}

// ProcessedMiniBlocksTracker -
func (pcs *ProcessComponentsStub) ProcessedMiniBlocksTracker() process.ProcessedMiniBlocksTracker {
	return pcs.ProcessedMiniBlocksTrackerInternal
}

// ReceiptsRepository -
func (pcs *ProcessComponentsStub) ReceiptsRepository() factory.ReceiptsRepository {
	return pcs.ReceiptsRepositoryInternal
}

// ESDTDataStorageHandlerForAPI -
func (pcs *ProcessComponentsStub) ESDTDataStorageHandlerForAPI() vmcommon.ESDTNFTStorageHandler {
	return pcs.ESDTDataStorageHandlerForAPIInternal
}

// IsInterfaceNil -
func (pcs *ProcessComponentsStub) IsInterfaceNil() bool {
	return pcs == nil
}
