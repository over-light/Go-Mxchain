package node

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/closing"
	"github.com/ElrondNetwork/elrond-go-core/core/throttler"
	"github.com/ElrondNetwork/elrond-go-core/data/endProcess"
	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/api/gin"
	"github.com/ElrondNetwork/elrond-go/api/shared"
	"github.com/ElrondNetwork/elrond-go/cmd/node/factory"
	"github.com/ElrondNetwork/elrond-go/common"
	"github.com/ElrondNetwork/elrond-go/common/forking"
	"github.com/ElrondNetwork/elrond-go/common/statistics"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/consensus/spos"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	dbLookupFactory "github.com/ElrondNetwork/elrond-go/dblookupext/factory"
	"github.com/ElrondNetwork/elrond-go/facade"
	"github.com/ElrondNetwork/elrond-go/facade/initial"
	mainFactory "github.com/ElrondNetwork/elrond-go/factory"
	apiComp "github.com/ElrondNetwork/elrond-go/factory/api"
	bootstrapComp "github.com/ElrondNetwork/elrond-go/factory/bootstrap"
	consensusComp "github.com/ElrondNetwork/elrond-go/factory/consensus"
	coreComp "github.com/ElrondNetwork/elrond-go/factory/core"
	cryptoComp "github.com/ElrondNetwork/elrond-go/factory/crypto"
	dataComp "github.com/ElrondNetwork/elrond-go/factory/data"
	heartbeatComp "github.com/ElrondNetwork/elrond-go/factory/heartbeat"
	networkComp "github.com/ElrondNetwork/elrond-go/factory/network"
	processComp "github.com/ElrondNetwork/elrond-go/factory/processing"
	stateComp "github.com/ElrondNetwork/elrond-go/factory/state"
	statusComp "github.com/ElrondNetwork/elrond-go/factory/status"
	"github.com/ElrondNetwork/elrond-go/factory/statusCore"
	"github.com/ElrondNetwork/elrond-go/genesis"
	"github.com/ElrondNetwork/elrond-go/genesis/parsing"
	"github.com/ElrondNetwork/elrond-go/health"
	"github.com/ElrondNetwork/elrond-go/node/metrics"
	"github.com/ElrondNetwork/elrond-go/outport"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/interceptors"
	"github.com/ElrondNetwork/elrond-go/sharding/nodesCoordinator"
	"github.com/ElrondNetwork/elrond-go/state/syncer"
	"github.com/ElrondNetwork/elrond-go/storage/cache"
	storageFactory "github.com/ElrondNetwork/elrond-go/storage/factory"
	"github.com/ElrondNetwork/elrond-go/storage/storageunit"
	trieFactory "github.com/ElrondNetwork/elrond-go/trie/factory"
	"github.com/ElrondNetwork/elrond-go/trie/storageMarker"
	"github.com/ElrondNetwork/elrond-go/update/trigger"
	"github.com/google/gops/agent"
)

const (
	// TODO: remove this after better handling VM versions switching
	// delayBeforeScQueriesStart represents the delay before the sc query processor should start to allow external queries
	delayBeforeScQueriesStart = 2 * time.Minute

	maxTimeToClose = 10 * time.Second
	// SoftRestartMessage is the custom message used when the node does a soft restart operation
	SoftRestartMessage = "Shuffled out - soft restart"
)

// nodeRunner holds the node runner configuration and controls running of a node
type nodeRunner struct {
	configs *config.Configs
}

// NewNodeRunner creates a nodeRunner instance
func NewNodeRunner(cfgs *config.Configs) (*nodeRunner, error) {
	if cfgs == nil {
		return nil, fmt.Errorf("nil configs provided")
	}

	return &nodeRunner{
		configs: cfgs,
	}, nil
}

// Start creates and starts the managed components
func (nr *nodeRunner) Start() error {
	configs := nr.configs
	flagsConfig := configs.FlagsConfig
	configurationPaths := configs.ConfigurationPathsHolder
	chanStopNodeProcess := make(chan endProcess.ArgEndProcess, 1)

	enableGopsIfNeeded(flagsConfig.EnableGops)

	var err error
	configurationPaths.Nodes, err = nr.getNodesFileName()
	if err != nil {
		return err
	}

	log.Debug("config", "file", configurationPaths.Nodes)
	log.Debug("config", "file", configurationPaths.Genesis)

	log.Info("starting node", "version", flagsConfig.Version, "pid", os.Getpid())

	err = cleanupStorageIfNecessary(flagsConfig.WorkingDir, flagsConfig.CleanupStorage)
	if err != nil {
		return err
	}

	printEnableEpochs(nr.configs)

	core.DumpGoRoutinesToLog(0, log)

	err = nr.startShufflingProcessLoop(chanStopNodeProcess)
	if err != nil {
		return err
	}

	return nil
}

func printEnableEpochs(configs *config.Configs) {
	var readEpochFor = func(flag string) string {
		return fmt.Sprintf("read enable epoch for %s", flag)
	}

	enableEpochs := configs.EpochConfig.EnableEpochs

	log.Debug(readEpochFor("sc deploy"), "epoch", enableEpochs.SCDeployEnableEpoch)
	log.Debug(readEpochFor("built in functions"), "epoch", enableEpochs.BuiltInFunctionsEnableEpoch)
	log.Debug(readEpochFor("relayed transactions"), "epoch", enableEpochs.RelayedTransactionsEnableEpoch)
	log.Debug(readEpochFor("penalized too much gas"), "epoch", enableEpochs.PenalizedTooMuchGasEnableEpoch)
	log.Debug(readEpochFor("switch jail waiting"), "epoch", enableEpochs.SwitchJailWaitingEnableEpoch)
	log.Debug(readEpochFor("switch hysteresis for min nodes"), "epoch", enableEpochs.SwitchHysteresisForMinNodesEnableEpoch)
	log.Debug(readEpochFor("below signed threshold"), "epoch", enableEpochs.BelowSignedThresholdEnableEpoch)
	log.Debug(readEpochFor("transaction signed with tx hash"), "epoch", enableEpochs.TransactionSignedWithTxHashEnableEpoch)
	log.Debug(readEpochFor("meta protection"), "epoch", enableEpochs.MetaProtectionEnableEpoch)
	log.Debug(readEpochFor("ahead of time gas usage"), "epoch", enableEpochs.AheadOfTimeGasUsageEnableEpoch)
	log.Debug(readEpochFor("gas price modifier"), "epoch", enableEpochs.GasPriceModifierEnableEpoch)
	log.Debug(readEpochFor("repair callback"), "epoch", enableEpochs.RepairCallbackEnableEpoch)
	log.Debug(readEpochFor("max nodes change"), "epoch", enableEpochs.MaxNodesChangeEnableEpoch)
	log.Debug(readEpochFor("block gas and fees re-check"), "epoch", enableEpochs.BlockGasAndFeesReCheckEnableEpoch)
	log.Debug(readEpochFor("staking v2 epoch"), "epoch", enableEpochs.StakingV2EnableEpoch)
	log.Debug(readEpochFor("stake"), "epoch", enableEpochs.StakeEnableEpoch)
	log.Debug(readEpochFor("double key protection"), "epoch", enableEpochs.DoubleKeyProtectionEnableEpoch)
	log.Debug(readEpochFor("esdt"), "epoch", enableEpochs.ESDTEnableEpoch)
	log.Debug(readEpochFor("governance"), "epoch", enableEpochs.GovernanceEnableEpoch)
	log.Debug(readEpochFor("delegation manager"), "epoch", enableEpochs.DelegationManagerEnableEpoch)
	log.Debug(readEpochFor("delegation smart contract"), "epoch", enableEpochs.DelegationSmartContractEnableEpoch)
	log.Debug(readEpochFor("correct last unjailed"), "epoch", enableEpochs.CorrectLastUnjailedEnableEpoch)
	log.Debug(readEpochFor("balance waiting lists"), "epoch", enableEpochs.BalanceWaitingListsEnableEpoch)
	log.Debug(readEpochFor("relayed transactions v2"), "epoch", enableEpochs.RelayedTransactionsV2EnableEpoch)
	log.Debug(readEpochFor("unbond tokens v2"), "epoch", enableEpochs.UnbondTokensV2EnableEpoch)
	log.Debug(readEpochFor("save jailed always"), "epoch", enableEpochs.SaveJailedAlwaysEnableEpoch)
	log.Debug(readEpochFor("validator to delegation"), "epoch", enableEpochs.ValidatorToDelegationEnableEpoch)
	log.Debug(readEpochFor("re-delegate below minimum check"), "epoch", enableEpochs.ReDelegateBelowMinCheckEnableEpoch)
	log.Debug(readEpochFor("waiting waiting list"), "epoch", enableEpochs.WaitingListFixEnableEpoch)
	log.Debug(readEpochFor("increment SCR nonce in multi transfer"), "epoch", enableEpochs.IncrementSCRNonceInMultiTransferEnableEpoch)
	log.Debug(readEpochFor("esdt and NFT multi transfer"), "epoch", enableEpochs.ESDTMultiTransferEnableEpoch)
	log.Debug(readEpochFor("contract global mint and burn"), "epoch", enableEpochs.GlobalMintBurnDisableEpoch)
	log.Debug(readEpochFor("contract transfer role"), "epoch", enableEpochs.ESDTTransferRoleEnableEpoch)
	log.Debug(readEpochFor("built in functions on metachain"), "epoch", enableEpochs.BuiltInFunctionOnMetaEnableEpoch)
	log.Debug(readEpochFor("compute rewards checkpoint on delegation"), "epoch", enableEpochs.ComputeRewardCheckpointEnableEpoch)
	log.Debug(readEpochFor("esdt NFT create on multiple shards"), "epoch", enableEpochs.ESDTNFTCreateOnMultiShardEnableEpoch)
	log.Debug(readEpochFor("SCR size invariant check"), "epoch", enableEpochs.SCRSizeInvariantCheckEnableEpoch)
	log.Debug(readEpochFor("backward compatibility flag for save key value"), "epoch", enableEpochs.BackwardCompSaveKeyValueEnableEpoch)
	log.Debug(readEpochFor("meta ESDT, financial SFT"), "epoch", enableEpochs.MetaESDTSetEnableEpoch)
	log.Debug(readEpochFor("add tokens to delegation"), "epoch", enableEpochs.AddTokensToDelegationEnableEpoch)
	log.Debug(readEpochFor("multi ESDT transfer on callback"), "epoch", enableEpochs.MultiESDTTransferFixOnCallBackOnEnableEpoch)
	log.Debug(readEpochFor("optimize gas used in cross mini blocks"), "epoch", enableEpochs.OptimizeGasUsedInCrossMiniBlocksEnableEpoch)
	log.Debug(readEpochFor("correct first queued"), "epoch", enableEpochs.CorrectFirstQueuedEpoch)
	log.Debug(readEpochFor("fix out of gas return code"), "epoch", enableEpochs.FixOOGReturnCodeEnableEpoch)
	log.Debug(readEpochFor("remove non updated storage"), "epoch", enableEpochs.RemoveNonUpdatedStorageEnableEpoch)
	log.Debug(readEpochFor("delete delegator data after claim rewards"), "epoch", enableEpochs.DeleteDelegatorAfterClaimRewardsEnableEpoch)
	log.Debug(readEpochFor("optimize nft metadata store"), "epoch", enableEpochs.OptimizeNFTStoreEnableEpoch)
	log.Debug(readEpochFor("create nft through execute on destination by caller"), "epoch", enableEpochs.CreateNFTThroughExecByCallerEnableEpoch)
	log.Debug(readEpochFor("payable by smart contract"), "epoch", enableEpochs.IsPayableBySCEnableEpoch)
	log.Debug(readEpochFor("cleanup informative only SCRs"), "epoch", enableEpochs.CleanUpInformativeSCRsEnableEpoch)
	log.Debug(readEpochFor("storage API cost optimization"), "epoch", enableEpochs.StorageAPICostOptimizationEnableEpoch)
	log.Debug(readEpochFor("transform to multi shard create on esdt"), "epoch", enableEpochs.TransformToMultiShardCreateEnableEpoch)
	log.Debug(readEpochFor("esdt: enable epoch for esdt register and set all roles function"), "epoch", enableEpochs.ESDTRegisterAndSetAllRolesEnableEpoch)
	log.Debug(readEpochFor("scheduled mini blocks"), "epoch", enableEpochs.ScheduledMiniBlocksEnableEpoch)
	log.Debug(readEpochFor("correct jailed not unstaked if empty queue"), "epoch", enableEpochs.CorrectJailedNotUnstakedEmptyQueueEpoch)
	log.Debug(readEpochFor("do not return old block in blockchain hook"), "epoch", enableEpochs.DoNotReturnOldBlockInBlockchainHookEnableEpoch)
	log.Debug(readEpochFor("scr size invariant check on built in"), "epoch", enableEpochs.SCRSizeInvariantOnBuiltInResultEnableEpoch)
	log.Debug(readEpochFor("correct check on tokenID for transfer role"), "epoch", enableEpochs.CheckCorrectTokenIDForTransferRoleEnableEpoch)
	log.Debug(readEpochFor("disable check value on exec by caller"), "epoch", enableEpochs.DisableExecByCallerEnableEpoch)
	log.Debug(readEpochFor("fail execution on every wrong API call"), "epoch", enableEpochs.FailExecutionOnEveryAPIErrorEnableEpoch)
	log.Debug(readEpochFor("managed crypto API in wasm vm"), "epoch", enableEpochs.ManagedCryptoAPIsEnableEpoch)
	log.Debug(readEpochFor("refactor contexts"), "epoch", enableEpochs.RefactorContextEnableEpoch)
	log.Debug(readEpochFor("disable heartbeat v1"), "epoch", enableEpochs.HeartbeatDisableEpoch)
	log.Debug(readEpochFor("mini block partial execution"), "epoch", enableEpochs.MiniBlockPartialExecutionEnableEpoch)
	log.Debug(readEpochFor("fix async callback arguments list"), "epoch", enableEpochs.FixAsyncCallBackArgsListEnableEpoch)
	log.Debug(readEpochFor("set sender in eei output transfer"), "epoch", enableEpochs.SetSenderInEeiOutputTransferEnableEpoch)
	log.Debug(readEpochFor("refactor peers mini blocks"), "epoch", enableEpochs.RefactorPeersMiniBlocksEnableEpoch)
	gasSchedule := configs.EpochConfig.GasSchedule

	log.Debug(readEpochFor("gas schedule directories paths"), "epoch", gasSchedule.GasScheduleByEpochs)
}

func (nr *nodeRunner) startShufflingProcessLoop(
	chanStopNodeProcess chan endProcess.ArgEndProcess,
) error {
	for {
		log.Debug("\n\n====================Starting managedComponents creation================================")

		shouldStop, err := nr.executeOneComponentCreationCycle(chanStopNodeProcess)
		if shouldStop {
			return err
		}

		nr.shuffleOutStatsAndGC()
	}
}

func (nr *nodeRunner) shuffleOutStatsAndGC() {
	debugConfig := nr.configs.GeneralConfig.Debug.ShuffleOut

	extraMessage := ""
	if debugConfig.CallGCWhenShuffleOut {
		extraMessage = " before running GC"
	}
	if debugConfig.ExtraPrintsOnShuffleOut {
		log.Debug("node statistics"+extraMessage, statistics.GetRuntimeStatistics()...)
	}
	if debugConfig.CallGCWhenShuffleOut {
		log.Debug("running runtime.GC()")
		runtime.GC()
	}
	shouldPrintAnotherNodeStatistics := debugConfig.CallGCWhenShuffleOut && debugConfig.ExtraPrintsOnShuffleOut
	if shouldPrintAnotherNodeStatistics {
		log.Debug("node statistics after running GC", statistics.GetRuntimeStatistics()...)
	}

	nr.doProfileOnShuffleOut()
}

func (nr *nodeRunner) doProfileOnShuffleOut() {
	debugConfig := nr.configs.GeneralConfig.Debug.ShuffleOut
	shouldDoProfile := debugConfig.DoProfileOnShuffleOut && nr.configs.FlagsConfig.UseHealthService
	if !shouldDoProfile {
		return
	}

	log.Debug("running profile job")
	parentPath := filepath.Join(nr.configs.FlagsConfig.WorkingDir, nr.configs.GeneralConfig.Health.FolderPath)
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	err := health.WriteMemoryUseInfo(stats, time.Now(), parentPath, "softrestart")
	log.LogIfError(err)
}

func (nr *nodeRunner) executeOneComponentCreationCycle(
	chanStopNodeProcess chan endProcess.ArgEndProcess,
) (bool, error) {
	goRoutinesNumberStart := runtime.NumGoroutine()
	configs := nr.configs
	flagsConfig := configs.FlagsConfig
	configurationPaths := configs.ConfigurationPathsHolder

	log.Debug("creating healthService")
	healthService := nr.createHealthService(flagsConfig)

	log.Debug("creating status core components")
	managedStatusCoreComponents, err := nr.CreateManagedStatusCoreComponents()
	if err != nil {
		return true, err
	}

	log.Debug("creating core components")
	managedCoreComponents, err := nr.CreateManagedCoreComponents(
		chanStopNodeProcess,
	)
	if err != nil {
		return true, err
	}

	log.Debug("creating crypto components")
	managedCryptoComponents, err := nr.CreateManagedCryptoComponents(managedCoreComponents)
	if err != nil {
		return true, err
	}

	log.Debug("creating network components")
	managedNetworkComponents, err := nr.CreateManagedNetworkComponents(managedCoreComponents)
	if err != nil {
		return true, err
	}

	log.Debug("creating disabled API services")
	webServerHandler, err := nr.createHttpServer()
	if err != nil {
		return true, err
	}

	log.Debug("creating bootstrap components")
	managedBootstrapComponents, err := nr.CreateManagedBootstrapComponents(managedCoreComponents, managedCryptoComponents, managedNetworkComponents)
	if err != nil {
		return true, err
	}

	nr.logInformation(managedCoreComponents, managedCryptoComponents, managedBootstrapComponents)

	log.Debug("creating data components")
	managedDataComponents, err := nr.CreateManagedDataComponents(managedCoreComponents, managedBootstrapComponents)
	if err != nil {
		return true, err
	}

	log.Debug("creating state components")
	managedStateComponents, err := nr.CreateManagedStateComponents(
		managedCoreComponents,
		managedBootstrapComponents,
		managedDataComponents,
	)
	if err != nil {
		return true, err
	}

	log.Debug("creating metrics")
	// this should be called before setting the storer (done in the managedDataComponents creation)
	err = nr.createMetrics(managedCoreComponents, managedCryptoComponents, managedBootstrapComponents)
	if err != nil {
		return true, err
	}

	log.Debug("registering components in healthService")
	nr.registerDataComponentsInHealthService(healthService, managedDataComponents)

	nodesShufflerOut, err := bootstrapComp.CreateNodesShuffleOut(
		managedCoreComponents.GenesisNodesSetup(),
		configs.GeneralConfig.EpochStartConfig,
		managedCoreComponents.ChanStopNodeProcess(),
	)
	if err != nil {
		return true, err
	}

	bootstrapStorer, err := managedDataComponents.StorageService().GetStorer(dataRetriever.BootstrapUnit)
	if err != nil {
		return true, err
	}

	log.Debug("creating nodes coordinator")
	nodesCoordinatorInstance, err := bootstrapComp.CreateNodesCoordinator(
		nodesShufflerOut,
		managedCoreComponents.GenesisNodesSetup(),
		configs.PreferencesConfig.Preferences,
		managedCoreComponents.EpochStartNotifierWithConfirm(),
		managedCryptoComponents.PublicKey(),
		managedCoreComponents.InternalMarshalizer(),
		managedCoreComponents.Hasher(),
		managedCoreComponents.Rater(),
		bootstrapStorer,
		managedCoreComponents.NodesShuffler(),
		managedBootstrapComponents.ShardCoordinator().SelfId(),
		managedBootstrapComponents.EpochBootstrapParams(),
		managedBootstrapComponents.EpochBootstrapParams().Epoch(),
		managedCoreComponents.ChanStopNodeProcess(),
		managedCoreComponents.NodeTypeProvider(),
		managedCoreComponents.EnableEpochsHandler(),
		managedDataComponents.Datapool().CurrentEpochValidatorInfo(),
	)
	if err != nil {
		return true, err
	}

	log.Debug("starting status pooling components")
	managedStatusComponents, err := nr.CreateManagedStatusComponents(
		managedStatusCoreComponents,
		managedCoreComponents,
		managedNetworkComponents,
		managedBootstrapComponents,
		managedDataComponents,
		managedStateComponents,
		nodesCoordinatorInstance,
		configs.ImportDbConfig.IsImportDBMode,
	)
	if err != nil {
		return true, err
	}

	argsGasScheduleNotifier := forking.ArgsNewGasScheduleNotifier{
		GasScheduleConfig: configs.EpochConfig.GasSchedule,
		ConfigDir:         configurationPaths.GasScheduleDirectoryName,
		EpochNotifier:     managedCoreComponents.EpochNotifier(),
		ArwenChangeLocker: managedCoreComponents.ArwenChangeLocker(),
	}
	gasScheduleNotifier, err := forking.NewGasScheduleNotifier(argsGasScheduleNotifier)
	if err != nil {
		return true, err
	}

	log.Debug("creating process components")
	managedProcessComponents, err := nr.CreateManagedProcessComponents(
		managedCoreComponents,
		managedCryptoComponents,
		managedNetworkComponents,
		managedBootstrapComponents,
		managedStateComponents,
		managedDataComponents,
		managedStatusComponents,
		gasScheduleNotifier,
		nodesCoordinatorInstance,
	)
	if err != nil {
		return true, err
	}

	err = addSyncersToAccountsDB(
		configs.GeneralConfig,
		managedCoreComponents,
		managedDataComponents,
		managedStateComponents,
		managedBootstrapComponents,
		managedProcessComponents,
	)
	if err != nil {
		return true, err
	}

	hardforkTrigger := managedProcessComponents.HardforkTrigger()
	err = hardforkTrigger.AddCloser(nodesShufflerOut)
	if err != nil {
		return true, fmt.Errorf("%w when adding nodeShufflerOut in hardForkTrigger", err)
	}

	managedStatusComponents.SetForkDetector(managedProcessComponents.ForkDetector())
	err = managedStatusComponents.StartPolling()
	if err != nil {
		return true, err
	}

	log.Debug("starting node... executeOneComponentCreationCycle")

	managedConsensusComponents, err := nr.CreateManagedConsensusComponents(
		managedCoreComponents,
		managedNetworkComponents,
		managedCryptoComponents,
		managedDataComponents,
		managedStateComponents,
		managedStatusComponents,
		managedProcessComponents,
	)
	if err != nil {
		return true, err
	}

	managedHeartbeatComponents, err := nr.CreateManagedHeartbeatComponents(
		managedCoreComponents,
		managedNetworkComponents,
		managedCryptoComponents,
		managedDataComponents,
		managedProcessComponents,
		managedProcessComponents.NodeRedundancyHandler(),
	)

	if err != nil {
		return true, err
	}

	managedHeartbeatV2Components, err := nr.CreateManagedHeartbeatV2Components(
		managedBootstrapComponents,
		managedCoreComponents,
		managedNetworkComponents,
		managedCryptoComponents,
		managedDataComponents,
		managedProcessComponents,
	)

	if err != nil {
		return true, err
	}

	log.Debug("creating node structure")
	currentNode, err := CreateNode(
		configs.GeneralConfig,
		managedStatusCoreComponents,
		managedBootstrapComponents,
		managedCoreComponents,
		managedCryptoComponents,
		managedDataComponents,
		managedNetworkComponents,
		managedProcessComponents,
		managedStateComponents,
		managedStatusComponents,
		managedHeartbeatComponents,
		managedHeartbeatV2Components,
		managedConsensusComponents,
		flagsConfig.BootstrapRoundIndex,
		configs.ImportDbConfig.IsImportDBMode,
	)
	if err != nil {
		return true, err
	}

	if managedBootstrapComponents.ShardCoordinator().SelfId() == core.MetachainShardId {
		log.Debug("activating nodesCoordinator's validators indexing")
		indexValidatorsListIfNeeded(
			managedStatusComponents.OutportHandler(),
			nodesCoordinatorInstance,
			managedProcessComponents.EpochStartTrigger().Epoch(),
		)
	}

	// this channel will trigger the moment when the sc query service should be able to process VM Query requests
	allowExternalVMQueriesChan := make(chan struct{})

	log.Debug("updating the API service after creating the node facade")
	ef, err := nr.createApiFacade(currentNode, webServerHandler, gasScheduleNotifier, allowExternalVMQueriesChan)
	if err != nil {
		return true, err
	}

	log.Info("application is now running")

	// TODO: remove this and treat better the VM versions switching
	go func(statusHandler core.AppStatusHandler) {
		time.Sleep(delayBeforeScQueriesStart)
		close(allowExternalVMQueriesChan)
		statusHandler.SetStringValue(common.MetricAreVMQueriesReady, strconv.FormatBool(true))
	}(managedCoreComponents.StatusHandler())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	err = waitForSignal(
		sigs,
		managedCoreComponents.ChanStopNodeProcess(),
		healthService,
		ef,
		webServerHandler,
		currentNode,
		goRoutinesNumberStart,
	)
	if err != nil {
		return true, nil
	}

	return false, nil
}

func addSyncersToAccountsDB(
	config *config.Config,
	coreComponents mainFactory.CoreComponentsHolder,
	dataComponents mainFactory.DataComponentsHolder,
	stateComponents mainFactory.StateComponentsHolder,
	bootstrapComponents mainFactory.BootstrapComponentsHolder,
	processComponents mainFactory.ProcessComponentsHolder,
) error {
	selfId := bootstrapComponents.ShardCoordinator().SelfId()
	if selfId == core.MetachainShardId {
		stateSyncer, err := getValidatorAccountSyncer(
			config,
			coreComponents,
			dataComponents,
			stateComponents,
			processComponents,
		)
		if err != nil {
			return err
		}

		err = stateComponents.PeerAccounts().SetSyncer(stateSyncer)
		if err != nil {
			return err
		}

		err = stateComponents.PeerAccounts().StartSnapshotIfNeeded()
		if err != nil {
			return err
		}
	}

	stateSyncer, err := getUserAccountSyncer(
		config,
		coreComponents,
		dataComponents,
		stateComponents,
		bootstrapComponents,
		processComponents,
	)
	if err != nil {
		return err
	}
	err = stateComponents.AccountsAdapter().SetSyncer(stateSyncer)
	if err != nil {
		return err
	}

	return stateComponents.AccountsAdapter().StartSnapshotIfNeeded()
}

func getUserAccountSyncer(
	config *config.Config,
	coreComponents mainFactory.CoreComponentsHolder,
	dataComponents mainFactory.DataComponentsHolder,
	stateComponents mainFactory.StateComponentsHolder,
	bootstrapComponents mainFactory.BootstrapComponentsHolder,
	processComponents mainFactory.ProcessComponentsHolder,
) (process.AccountsDBSyncer, error) {
	maxTrieLevelInMemory := config.StateTriesConfig.MaxStateTrieLevelInMemory
	userTrie := stateComponents.TriesContainer().Get([]byte(trieFactory.UserAccountTrie))
	storageManager := userTrie.GetStorageManager()

	thr, err := throttler.NewNumGoRoutinesThrottler(int32(config.TrieSync.NumConcurrentTrieSyncers))
	if err != nil {
		return nil, err
	}

	args := syncer.ArgsNewUserAccountsSyncer{
		ArgsNewBaseAccountsSyncer: getBaseAccountSyncerArgs(
			config,
			coreComponents,
			dataComponents,
			processComponents,
			storageManager,
			maxTrieLevelInMemory,
		),
		ShardId:                bootstrapComponents.ShardCoordinator().SelfId(),
		Throttler:              thr,
		AddressPubKeyConverter: coreComponents.AddressPubKeyConverter(),
	}

	return syncer.NewUserAccountsSyncer(args)
}

func getValidatorAccountSyncer(
	config *config.Config,
	coreComponents mainFactory.CoreComponentsHolder,
	dataComponents mainFactory.DataComponentsHolder,
	stateComponents mainFactory.StateComponentsHolder,
	processComponents mainFactory.ProcessComponentsHolder,
) (process.AccountsDBSyncer, error) {
	maxTrieLevelInMemory := config.StateTriesConfig.MaxPeerTrieLevelInMemory
	peerTrie := stateComponents.TriesContainer().Get([]byte(trieFactory.PeerAccountTrie))
	storageManager := peerTrie.GetStorageManager()

	args := syncer.ArgsNewValidatorAccountsSyncer{
		ArgsNewBaseAccountsSyncer: getBaseAccountSyncerArgs(
			config,
			coreComponents,
			dataComponents,
			processComponents,
			storageManager,
			maxTrieLevelInMemory,
		),
	}

	return syncer.NewValidatorAccountsSyncer(args)
}

func getBaseAccountSyncerArgs(
	config *config.Config,
	coreComponents mainFactory.CoreComponentsHolder,
	dataComponents mainFactory.DataComponentsHolder,
	processComponents mainFactory.ProcessComponentsHolder,
	storageManager common.StorageManager,
	maxTrieLevelInMemory uint,
) syncer.ArgsNewBaseAccountsSyncer {
	return syncer.ArgsNewBaseAccountsSyncer{
		Hasher:                    coreComponents.Hasher(),
		Marshalizer:               coreComponents.InternalMarshalizer(),
		TrieStorageManager:        storageManager,
		RequestHandler:            processComponents.RequestHandler(),
		Timeout:                   common.TimeoutGettingTrieNodes,
		Cacher:                    dataComponents.Datapool().TrieNodes(),
		MaxTrieLevelInMemory:      maxTrieLevelInMemory,
		MaxHardCapForMissingNodes: config.TrieSync.MaxHardCapForMissingNodes,
		TrieSyncerVersion:         config.TrieSync.TrieSyncerVersion,
		StorageMarker:             storageMarker.NewDisabledStorageMarker(),
		CheckNodesOnDisk:          true,
	}
}

func (nr *nodeRunner) createApiFacade(
	currentNode *Node,
	upgradableHttpServer shared.UpgradeableHttpServerHandler,
	gasScheduleNotifier common.GasScheduleNotifierAPI,
	allowVMQueriesChan chan struct{},
) (closing.Closer, error) {
	configs := nr.configs

	log.Debug("creating api resolver structure")

	apiResolverArgs := &apiComp.ApiResolverArgs{
		Configs:             configs,
		CoreComponents:      currentNode.coreComponents,
		DataComponents:      currentNode.dataComponents,
		StateComponents:     currentNode.stateComponents,
		BootstrapComponents: currentNode.bootstrapComponents,
		CryptoComponents:    currentNode.cryptoComponents,
		ProcessComponents:   currentNode.processComponents,
		GasScheduleNotifier: gasScheduleNotifier,
		Bootstrapper:        currentNode.consensusComponents.Bootstrapper(),
		AllowVMQueriesChan:  allowVMQueriesChan,
	}

	apiResolver, err := apiComp.CreateApiResolver(apiResolverArgs)
	if err != nil {
		return nil, err
	}

	log.Debug("creating elrond node facade")

	flagsConfig := configs.FlagsConfig

	argNodeFacade := facade.ArgNodeFacade{
		Node:                   currentNode,
		ApiResolver:            apiResolver,
		TxSimulatorProcessor:   currentNode.processComponents.TransactionSimulatorProcessor(),
		RestAPIServerDebugMode: flagsConfig.EnableRestAPIServerDebugMode,
		WsAntifloodConfig:      configs.GeneralConfig.Antiflood.WebServer,
		FacadeConfig: config.FacadeConfig{
			RestApiInterface: flagsConfig.RestApiInterface,
			PprofEnabled:     flagsConfig.EnablePprof,
		},
		ApiRoutesConfig: *configs.ApiRoutesConfig,
		AccountsState:   currentNode.stateComponents.AccountsAdapter(),
		PeerState:       currentNode.stateComponents.PeerAccounts(),
		Blockchain:      currentNode.dataComponents.Blockchain(),
	}

	ef, err := facade.NewNodeFacade(argNodeFacade)
	if err != nil {
		return nil, fmt.Errorf("%w while creating NodeFacade", err)
	}

	ef.SetSyncer(currentNode.coreComponents.SyncTimer())

	err = upgradableHttpServer.UpdateFacade(ef)
	if err != nil {
		return nil, err
	}

	log.Debug("updated node facade")

	log.Trace("starting background services")

	return ef, nil
}

func (nr *nodeRunner) createHttpServer() (shared.UpgradeableHttpServerHandler, error) {
	httpServerArgs := gin.ArgsNewWebServer{
		Facade:          initial.NewInitialNodeFacade(nr.configs.FlagsConfig.RestApiInterface, nr.configs.FlagsConfig.EnablePprof),
		ApiConfig:       *nr.configs.ApiRoutesConfig,
		AntiFloodConfig: nr.configs.GeneralConfig.Antiflood.WebServer,
	}

	httpServerWrapper, err := gin.NewGinWebServerHandler(httpServerArgs)
	if err != nil {
		return nil, err
	}

	err = httpServerWrapper.StartHttpServer()
	if err != nil {
		return nil, err
	}

	return httpServerWrapper, nil
}

func (nr *nodeRunner) createMetrics(
	coreComponents mainFactory.CoreComponentsHolder,
	cryptoComponents mainFactory.CryptoComponentsHolder,
	bootstrapComponents mainFactory.BootstrapComponentsHolder,
) error {
	err := metrics.InitMetrics(
		coreComponents.StatusHandlerUtils(),
		cryptoComponents.PublicKeyString(),
		bootstrapComponents.NodeType(),
		bootstrapComponents.ShardCoordinator(),
		coreComponents.GenesisNodesSetup(),
		nr.configs.FlagsConfig.Version,
		nr.configs.EconomicsConfig,
		nr.configs.GeneralConfig.EpochStartConfig.RoundsPerEpoch,
		coreComponents.MinTransactionVersion(),
	)

	if err != nil {
		return err
	}

	metrics.SaveStringMetric(coreComponents.StatusHandler(), common.MetricNodeDisplayName, nr.configs.PreferencesConfig.Preferences.NodeDisplayName)
	metrics.SaveStringMetric(coreComponents.StatusHandler(), common.MetricRedundancyLevel, fmt.Sprintf("%d", nr.configs.PreferencesConfig.Preferences.RedundancyLevel))
	metrics.SaveStringMetric(coreComponents.StatusHandler(), common.MetricRedundancyIsMainActive, common.MetricValueNA)
	metrics.SaveStringMetric(coreComponents.StatusHandler(), common.MetricChainId, coreComponents.ChainID())
	metrics.SaveUint64Metric(coreComponents.StatusHandler(), common.MetricGasPerDataByte, coreComponents.EconomicsData().GasPerDataByte())
	metrics.SaveUint64Metric(coreComponents.StatusHandler(), common.MetricMinGasPrice, coreComponents.EconomicsData().MinGasPrice())
	metrics.SaveUint64Metric(coreComponents.StatusHandler(), common.MetricMinGasLimit, coreComponents.EconomicsData().MinGasLimit())
	metrics.SaveStringMetric(coreComponents.StatusHandler(), common.MetricRewardsTopUpGradientPoint, coreComponents.EconomicsData().RewardsTopUpGradientPoint().String())
	metrics.SaveStringMetric(coreComponents.StatusHandler(), common.MetricTopUpFactor, fmt.Sprintf("%g", coreComponents.EconomicsData().RewardsTopUpFactor()))
	metrics.SaveStringMetric(coreComponents.StatusHandler(), common.MetricGasPriceModifier, fmt.Sprintf("%g", coreComponents.EconomicsData().GasPriceModifier()))
	metrics.SaveUint64Metric(coreComponents.StatusHandler(), common.MetricMaxGasPerTransaction, coreComponents.EconomicsData().MaxGasLimitPerTx())
	return nil
}

func (nr *nodeRunner) createHealthService(flagsConfig *config.ContextFlagsConfig) HealthService {
	healthService := health.NewHealthService(nr.configs.GeneralConfig.Health, flagsConfig.WorkingDir)
	if flagsConfig.UseHealthService {
		healthService.Start()
	}

	return healthService
}

func (nr *nodeRunner) registerDataComponentsInHealthService(healthService HealthService, dataComponents mainFactory.DataComponentsHolder) {
	healthService.RegisterComponent(dataComponents.Datapool().Transactions())
	healthService.RegisterComponent(dataComponents.Datapool().UnsignedTransactions())
	healthService.RegisterComponent(dataComponents.Datapool().RewardTransactions())
}

// CreateManagedConsensusComponents is the managed consensus components factory
func (nr *nodeRunner) CreateManagedConsensusComponents(
	coreComponents mainFactory.CoreComponentsHolder,
	networkComponents mainFactory.NetworkComponentsHolder,
	cryptoComponents mainFactory.CryptoComponentsHolder,
	dataComponents mainFactory.DataComponentsHolder,
	stateComponents mainFactory.StateComponentsHolder,
	statusComponents mainFactory.StatusComponentsHolder,
	processComponents mainFactory.ProcessComponentsHolder,
) (mainFactory.ConsensusComponentsHandler, error) {
	scheduledProcessorArgs := spos.ScheduledProcessorWrapperArgs{
		SyncTimer:                coreComponents.SyncTimer(),
		Processor:                processComponents.BlockProcessor(),
		RoundTimeDurationHandler: coreComponents.RoundHandler(),
	}

	scheduledProcessor, err := spos.NewScheduledProcessorWrapper(scheduledProcessorArgs)
	if err != nil {
		return nil, err
	}

	consensusArgs := consensusComp.ConsensusComponentsFactoryArgs{
		Config:                *nr.configs.GeneralConfig,
		BootstrapRoundIndex:   nr.configs.FlagsConfig.BootstrapRoundIndex,
		CoreComponents:        coreComponents,
		NetworkComponents:     networkComponents,
		CryptoComponents:      cryptoComponents,
		DataComponents:        dataComponents,
		ProcessComponents:     processComponents,
		StateComponents:       stateComponents,
		StatusComponents:      statusComponents,
		ScheduledProcessor:    scheduledProcessor,
		IsInImportMode:        nr.configs.ImportDbConfig.IsImportDBMode,
		ShouldDisableWatchdog: nr.configs.FlagsConfig.DisableConsensusWatchdog,
	}

	consensusFactory, err := consensusComp.NewConsensusComponentsFactory(consensusArgs)
	if err != nil {
		return nil, fmt.Errorf("NewConsensusComponentsFactory failed: %w", err)
	}

	managedConsensusComponents, err := consensusComp.NewManagedConsensusComponents(consensusFactory)
	if err != nil {
		return nil, err
	}

	err = managedConsensusComponents.Create()
	if err != nil {
		return nil, err
	}
	return managedConsensusComponents, nil
}

// CreateManagedHeartbeatComponents is the managed heartbeat components factory
func (nr *nodeRunner) CreateManagedHeartbeatComponents(
	coreComponents mainFactory.CoreComponentsHolder,
	networkComponents mainFactory.NetworkComponentsHolder,
	cryptoComponents mainFactory.CryptoComponentsHolder,
	dataComponents mainFactory.DataComponentsHolder,
	processComponents mainFactory.ProcessComponentsHolder,
	redundancyHandler consensus.NodeRedundancyHandler,
) (mainFactory.HeartbeatComponentsHandler, error) {
	genesisTime := time.Unix(coreComponents.GenesisNodesSetup().GetStartTime(), 0)

	heartbeatArgs := heartbeatComp.HeartbeatComponentsFactoryArgs{
		Config:            *nr.configs.GeneralConfig,
		Prefs:             *nr.configs.PreferencesConfig,
		AppVersion:        nr.configs.FlagsConfig.Version,
		GenesisTime:       genesisTime,
		RedundancyHandler: redundancyHandler,
		CoreComponents:    coreComponents,
		DataComponents:    dataComponents,
		NetworkComponents: networkComponents,
		CryptoComponents:  cryptoComponents,
		ProcessComponents: processComponents,
	}

	heartbeatComponentsFactory, err := heartbeatComp.NewHeartbeatComponentsFactory(heartbeatArgs)
	if err != nil {
		return nil, fmt.Errorf("NewHeartbeatComponentsFactory failed: %w", err)
	}

	managedHeartbeatComponents, err := heartbeatComp.NewManagedHeartbeatComponents(heartbeatComponentsFactory)
	if err != nil {
		return nil, err
	}

	err = managedHeartbeatComponents.Create()
	if err != nil {
		return nil, err
	}
	return managedHeartbeatComponents, nil
}

// CreateManagedHeartbeatV2Components is the managed heartbeatV2 components factory
func (nr *nodeRunner) CreateManagedHeartbeatV2Components(
	bootstrapComponents mainFactory.BootstrapComponentsHolder,
	coreComponents mainFactory.CoreComponentsHolder,
	networkComponents mainFactory.NetworkComponentsHolder,
	cryptoComponents mainFactory.CryptoComponentsHolder,
	dataComponents mainFactory.DataComponentsHolder,
	processComponents mainFactory.ProcessComponentsHolder,
) (mainFactory.HeartbeatV2ComponentsHandler, error) {
	heartbeatV2Args := heartbeatComp.ArgHeartbeatV2ComponentsFactory{
		Config:             *nr.configs.GeneralConfig,
		Prefs:              *nr.configs.PreferencesConfig,
		AppVersion:         nr.configs.FlagsConfig.Version,
		BoostrapComponents: bootstrapComponents,
		CoreComponents:     coreComponents,
		DataComponents:     dataComponents,
		NetworkComponents:  networkComponents,
		CryptoComponents:   cryptoComponents,
		ProcessComponents:  processComponents,
	}

	heartbeatV2ComponentsFactory, err := heartbeatComp.NewHeartbeatV2ComponentsFactory(heartbeatV2Args)
	if err != nil {
		return nil, fmt.Errorf("NewHeartbeatV2ComponentsFactory failed: %w", err)
	}

	managedHeartbeatV2Components, err := heartbeatComp.NewManagedHeartbeatV2Components(heartbeatV2ComponentsFactory)
	if err != nil {
		return nil, err
	}

	err = managedHeartbeatV2Components.Create()
	if err != nil {
		return nil, err
	}
	return managedHeartbeatV2Components, nil
}

func waitForSignal(
	sigs chan os.Signal,
	chanStopNodeProcess chan endProcess.ArgEndProcess,
	healthService closing.Closer,
	ef closing.Closer,
	httpServer shared.UpgradeableHttpServerHandler,
	currentNode *Node,
	goRoutinesNumberStart int,
) error {
	var sig endProcess.ArgEndProcess
	reshuffled := false
	wrongConfig := false
	wrongConfigDescription := ""

	select {
	case <-sigs:
		log.Info("terminating at user's signal...")
	case sig = <-chanStopNodeProcess:
		log.Info("terminating at internal stop signal", "reason", sig.Reason, "description", sig.Description)
		if sig.Reason == common.ShuffledOut {
			reshuffled = true
		}
		if sig.Reason == common.WrongConfiguration {
			wrongConfig = true
			wrongConfigDescription = sig.Description
		}
	}

	chanCloseComponents := make(chan struct{})
	go func() {
		closeAllComponents(healthService, ef, httpServer, currentNode, chanCloseComponents)
	}()

	select {
	case <-chanCloseComponents:
		log.Debug("Closed all components gracefully")
	case <-time.After(maxTimeToClose):
		log.Warn("force closing the node", "error", "closeAllComponents did not finish on time")
		return fmt.Errorf("did NOT close all components gracefully")
	}

	if wrongConfig {
		// hang the node's process because it cannot continue with the current configuration and a restart doesn't
		// change this behaviour
		for {
			log.Error("wrong configuration. stopped processing", "description", wrongConfigDescription)
			time.Sleep(1 * time.Minute)
		}
	}

	if reshuffled {
		log.Info("=============================" + SoftRestartMessage + "==================================")
		core.DumpGoRoutinesToLog(goRoutinesNumberStart, log)

		return nil
	}

	return fmt.Errorf("not reshuffled, closing")
}

func (nr *nodeRunner) logInformation(
	coreComponents mainFactory.CoreComponentsHolder,
	cryptoComponents mainFactory.CryptoComponentsHolder,
	bootstrapComponents mainFactory.BootstrapComponentsHolder,
) {
	log.Info("Bootstrap", "epoch", bootstrapComponents.EpochBootstrapParams().Epoch())
	if bootstrapComponents.EpochBootstrapParams().NodesConfig() != nil {
		log.Info("the epoch from nodesConfig is",
			"epoch", bootstrapComponents.EpochBootstrapParams().NodesConfig().CurrentEpoch)
	}

	var shardIdString = core.GetShardIDString(bootstrapComponents.ShardCoordinator().SelfId())
	logger.SetCorrelationShard(shardIdString)

	sessionInfoFileOutput := fmt.Sprintf("%s:%s\n%s:%s\n%s:%v\n%s:%s\n%s:%v\n",
		"PkBlockSign", cryptoComponents.PublicKeyString(),
		"ShardId", shardIdString,
		"TotalShards", bootstrapComponents.ShardCoordinator().NumberOfShards(),
		"AppVersion", nr.configs.FlagsConfig.Version,
		"GenesisTimeStamp", coreComponents.GenesisTime().Unix(),
	)

	sessionInfoFileOutput += "\nStarted with parameters:\n"
	sessionInfoFileOutput += nr.configs.FlagsConfig.SessionInfoFileOutput

	nr.logSessionInformation(nr.configs.FlagsConfig.WorkingDir, sessionInfoFileOutput, coreComponents)
}

func (nr *nodeRunner) getNodesFileName() (string, error) {
	flagsConfig := nr.configs.FlagsConfig
	configurationPaths := nr.configs.ConfigurationPathsHolder
	nodesFileName := configurationPaths.Nodes

	exportFolder := filepath.Join(flagsConfig.WorkingDir, nr.configs.GeneralConfig.Hardfork.ImportFolder)
	if nr.configs.GeneralConfig.Hardfork.AfterHardFork {
		exportFolderNodesSetupPath := filepath.Join(exportFolder, common.NodesSetupJsonFileName)
		if !core.FileExists(exportFolderNodesSetupPath) {
			return "", fmt.Errorf("cannot find %s in the export folder", common.NodesSetupJsonFileName)
		}

		nodesFileName = exportFolderNodesSetupPath
	}
	return nodesFileName, nil
}

// CreateManagedStatusComponents is the managed status components factory
func (nr *nodeRunner) CreateManagedStatusComponents(
	managedStatusCoreComponents mainFactory.StatusCoreComponentsHolder,
	managedCoreComponents mainFactory.CoreComponentsHolder,
	managedNetworkComponents mainFactory.NetworkComponentsHolder,
	managedBootstrapComponents mainFactory.BootstrapComponentsHolder,
	managedDataComponents mainFactory.DataComponentsHolder,
	managedStateComponents mainFactory.StateComponentsHolder,
	nodesCoordinator nodesCoordinator.NodesCoordinator,
	isInImportMode bool,
) (mainFactory.StatusComponentsHandler, error) {
	statArgs := statusComp.StatusComponentsFactoryArgs{
		Config:               *nr.configs.GeneralConfig,
		ExternalConfig:       *nr.configs.ExternalConfig,
		EconomicsConfig:      *nr.configs.EconomicsConfig,
		ShardCoordinator:     managedBootstrapComponents.ShardCoordinator(),
		NodesCoordinator:     nodesCoordinator,
		EpochStartNotifier:   managedCoreComponents.EpochStartNotifierWithConfirm(),
		CoreComponents:       managedCoreComponents,
		DataComponents:       managedDataComponents,
		NetworkComponents:    managedNetworkComponents,
		StateComponents:      managedStateComponents,
		IsInImportMode:       isInImportMode,
		StatusCoreComponents: managedStatusCoreComponents,
	}

	statusComponentsFactory, err := statusComp.NewStatusComponentsFactory(statArgs)
	if err != nil {
		return nil, fmt.Errorf("NewStatusComponentsFactory failed: %w", err)
	}

	managedStatusComponents, err := statusComp.NewManagedStatusComponents(statusComponentsFactory)
	if err != nil {
		return nil, err
	}
	err = managedStatusComponents.Create()
	if err != nil {
		return nil, err
	}
	return managedStatusComponents, nil
}

func (nr *nodeRunner) logSessionInformation(
	workingDir string,
	sessionInfoFileOutput string,
	coreComponents mainFactory.CoreComponentsHolder,
) {
	statsFolder := filepath.Join(workingDir, common.DefaultStatsPath)
	configurationPaths := nr.configs.ConfigurationPathsHolder
	copyConfigToStatsFolder(
		statsFolder,
		configurationPaths.GasScheduleDirectoryName,
		[]string{
			configurationPaths.MainConfig,
			configurationPaths.Economics,
			configurationPaths.Ratings,
			configurationPaths.Preferences,
			configurationPaths.P2p,
			configurationPaths.Genesis,
			configurationPaths.Nodes,
			configurationPaths.ApiRoutes,
			configurationPaths.External,
			configurationPaths.SystemSC,
			configurationPaths.RoundActivation,
			configurationPaths.Epoch,
		})

	statsFile := filepath.Join(statsFolder, "session.info")
	err := ioutil.WriteFile(statsFile, []byte(sessionInfoFileOutput), core.FileModeReadWrite)
	log.LogIfError(err)

	computedRatingsDataStr := createStringFromRatingsData(coreComponents.RatingsData())
	log.Debug("rating data", "rating", computedRatingsDataStr)
}

// CreateManagedProcessComponents is the managed process components factory
func (nr *nodeRunner) CreateManagedProcessComponents(
	coreComponents mainFactory.CoreComponentsHolder,
	cryptoComponents mainFactory.CryptoComponentsHolder,
	networkComponents mainFactory.NetworkComponentsHolder,
	bootstrapComponents mainFactory.BootstrapComponentsHolder,
	stateComponents mainFactory.StateComponentsHolder,
	dataComponents mainFactory.DataComponentsHolder,
	statusComponents mainFactory.StatusComponentsHolder,
	gasScheduleNotifier core.GasScheduleNotifier,
	nodesCoordinator nodesCoordinator.NodesCoordinator,
) (mainFactory.ProcessComponentsHandler, error) {
	configs := nr.configs
	configurationPaths := nr.configs.ConfigurationPathsHolder
	importStartHandler, err := trigger.NewImportStartHandler(filepath.Join(configs.FlagsConfig.WorkingDir, common.DefaultDBPath), configs.FlagsConfig.Version)
	if err != nil {
		return nil, err
	}

	totalSupply, ok := big.NewInt(0).SetString(configs.EconomicsConfig.GlobalSettings.GenesisTotalSupply, 10)
	if !ok {
		return nil, fmt.Errorf("can not parse total suply from economics.toml, %s is not a valid value",
			configs.EconomicsConfig.GlobalSettings.GenesisTotalSupply)
	}

	mintingSenderAddress := configs.EconomicsConfig.GlobalSettings.GenesisMintingSenderAddress

	args := genesis.AccountsParserArgs{
		GenesisFilePath: configurationPaths.Genesis,
		EntireSupply:    totalSupply,
		MinterAddress:   mintingSenderAddress,
		PubkeyConverter: coreComponents.AddressPubKeyConverter(),
		KeyGenerator:    cryptoComponents.TxSignKeyGen(),
		Hasher:          coreComponents.Hasher(),
		Marshalizer:     coreComponents.InternalMarshalizer(),
	}

	accountsParser, err := parsing.NewAccountsParser(args)
	if err != nil {
		return nil, err
	}

	smartContractParser, err := parsing.NewSmartContractsParser(
		configurationPaths.SmartContracts,
		coreComponents.AddressPubKeyConverter(),
		cryptoComponents.TxSignKeyGen(),
	)
	if err != nil {
		return nil, err
	}

	historyRepoFactoryArgs := &dbLookupFactory.ArgsHistoryRepositoryFactory{
		SelfShardID:              bootstrapComponents.ShardCoordinator().SelfId(),
		Config:                   configs.GeneralConfig.DbLookupExtensions,
		Hasher:                   coreComponents.Hasher(),
		Marshalizer:              coreComponents.InternalMarshalizer(),
		Store:                    dataComponents.StorageService(),
		Uint64ByteSliceConverter: coreComponents.Uint64ByteSliceConverter(),
	}
	historyRepositoryFactory, err := dbLookupFactory.NewHistoryRepositoryFactory(historyRepoFactoryArgs)
	if err != nil {
		return nil, err
	}

	whiteListCache, err := storageunit.NewCache(storageFactory.GetCacherFromConfig(configs.GeneralConfig.WhiteListPool))
	if err != nil {
		return nil, err
	}
	whiteListRequest, err := interceptors.NewWhiteListDataVerifier(whiteListCache)
	if err != nil {
		return nil, err
	}

	whiteListerVerifiedTxs, err := createWhiteListerVerifiedTxs(configs.GeneralConfig)
	if err != nil {
		return nil, err
	}

	historyRepository, err := historyRepositoryFactory.Create()
	if err != nil {
		return nil, err
	}

	log.Trace("creating time cache for requested items components")
	// TODO consider lowering this (perhaps to 1 second) and use a common const
	requestedItemsHandler := cache.NewTimeCache(
		time.Duration(uint64(time.Millisecond) * coreComponents.GenesisNodesSetup().GetRoundDuration()))

	processArgs := processComp.ProcessComponentsFactoryArgs{
		Config:                 *configs.GeneralConfig,
		EpochConfig:            *configs.EpochConfig,
		PrefConfigs:            configs.PreferencesConfig.Preferences,
		ImportDBConfig:         *configs.ImportDbConfig,
		AccountsParser:         accountsParser,
		SmartContractParser:    smartContractParser,
		GasSchedule:            gasScheduleNotifier,
		NodesCoordinator:       nodesCoordinator,
		Data:                   dataComponents,
		CoreData:               coreComponents,
		Crypto:                 cryptoComponents,
		State:                  stateComponents,
		Network:                networkComponents,
		BootstrapComponents:    bootstrapComponents,
		StatusComponents:       statusComponents,
		RequestedItemsHandler:  requestedItemsHandler,
		WhiteListHandler:       whiteListRequest,
		WhiteListerVerifiedTxs: whiteListerVerifiedTxs,
		MaxRating:              configs.RatingsConfig.General.MaxRating,
		SystemSCConfig:         configs.SystemSCConfig,
		Version:                configs.FlagsConfig.Version,
		ImportStartHandler:     importStartHandler,
		WorkingDir:             configs.FlagsConfig.WorkingDir,
		HistoryRepo:            historyRepository,
	}
	processComponentsFactory, err := processComp.NewProcessComponentsFactory(processArgs)
	if err != nil {
		return nil, fmt.Errorf("NewProcessComponentsFactory failed: %w", err)
	}

	managedProcessComponents, err := processComp.NewManagedProcessComponents(processComponentsFactory)
	if err != nil {
		return nil, err
	}

	err = managedProcessComponents.Create()
	if err != nil {
		return nil, err
	}

	return managedProcessComponents, nil
}

// CreateManagedDataComponents is the managed data components factory
func (nr *nodeRunner) CreateManagedDataComponents(
	coreComponents mainFactory.CoreComponentsHolder,
	bootstrapComponents mainFactory.BootstrapComponentsHolder,
) (mainFactory.DataComponentsHandler, error) {
	configs := nr.configs
	storerEpoch := bootstrapComponents.EpochBootstrapParams().Epoch()
	if !configs.GeneralConfig.StoragePruning.Enabled {
		// TODO: refactor this as when the pruning storer is disabled, the default directory path is Epoch_0
		// and it should be Epoch_ALL or something similar
		storerEpoch = 0
	}

	dataArgs := dataComp.DataComponentsFactoryArgs{
		Config:                        *configs.GeneralConfig,
		PrefsConfig:                   configs.PreferencesConfig.Preferences,
		ShardCoordinator:              bootstrapComponents.ShardCoordinator(),
		Core:                          coreComponents,
		EpochStartNotifier:            coreComponents.EpochStartNotifierWithConfirm(),
		CurrentEpoch:                  storerEpoch,
		CreateTrieEpochRootHashStorer: configs.ImportDbConfig.ImportDbSaveTrieEpochRootHash,
	}

	dataComponentsFactory, err := dataComp.NewDataComponentsFactory(dataArgs)
	if err != nil {
		return nil, fmt.Errorf("NewDataComponentsFactory failed: %w", err)
	}
	managedDataComponents, err := dataComp.NewManagedDataComponents(dataComponentsFactory)
	if err != nil {
		return nil, err
	}
	err = managedDataComponents.Create()
	if err != nil {
		return nil, err
	}

	statusMetricsStorer, err := managedDataComponents.StorageService().GetStorer(dataRetriever.StatusMetricsUnit)
	if err != nil {
		return nil, err
	}

	err = coreComponents.StatusHandlerUtils().UpdateStorerAndMetricsForPersistentHandler(statusMetricsStorer)

	if err != nil {
		return nil, err
	}

	return managedDataComponents, nil
}

// CreateManagedStateComponents is the managed state components factory
func (nr *nodeRunner) CreateManagedStateComponents(
	coreComponents mainFactory.CoreComponentsHolder,
	bootstrapComponents mainFactory.BootstrapComponentsHolder,
	dataComponents mainFactory.DataComponentsHandler,
) (mainFactory.StateComponentsHandler, error) {
	processingMode := common.Normal
	if nr.configs.ImportDbConfig.IsImportDBMode {
		processingMode = common.ImportDb
	}
	stateArgs := stateComp.StateComponentsFactoryArgs{
		Config:                   *nr.configs.GeneralConfig,
		ShardCoordinator:         bootstrapComponents.ShardCoordinator(),
		Core:                     coreComponents,
		StorageService:           dataComponents.StorageService(),
		ProcessingMode:           processingMode,
		ShouldSerializeSnapshots: nr.configs.FlagsConfig.SerializeSnapshots,
		ChainHandler:             dataComponents.Blockchain(),
	}

	stateComponentsFactory, err := stateComp.NewStateComponentsFactory(stateArgs)
	if err != nil {
		return nil, fmt.Errorf("NewStateComponentsFactory failed: %w", err)
	}

	managedStateComponents, err := stateComp.NewManagedStateComponents(stateComponentsFactory)
	if err != nil {
		return nil, err
	}

	err = managedStateComponents.Create()
	if err != nil {
		return nil, err
	}
	return managedStateComponents, nil
}

// CreateManagedBootstrapComponents is the managed bootstrap components factory
func (nr *nodeRunner) CreateManagedBootstrapComponents(
	coreComponents mainFactory.CoreComponentsHolder,
	cryptoComponents mainFactory.CryptoComponentsHolder,
	networkComponents mainFactory.NetworkComponentsHolder,
) (mainFactory.BootstrapComponentsHandler, error) {

	bootstrapComponentsFactoryArgs := bootstrapComp.BootstrapComponentsFactoryArgs{
		Config:            *nr.configs.GeneralConfig,
		PrefConfig:        *nr.configs.PreferencesConfig,
		ImportDbConfig:    *nr.configs.ImportDbConfig,
		FlagsConfig:       *nr.configs.FlagsConfig,
		WorkingDir:        nr.configs.FlagsConfig.WorkingDir,
		CoreComponents:    coreComponents,
		CryptoComponents:  cryptoComponents,
		NetworkComponents: networkComponents,
	}

	bootstrapComponentsFactory, err := bootstrapComp.NewBootstrapComponentsFactory(bootstrapComponentsFactoryArgs)
	if err != nil {
		return nil, fmt.Errorf("NewBootstrapComponentsFactory failed: %w", err)
	}

	managedBootstrapComponents, err := bootstrapComp.NewManagedBootstrapComponents(bootstrapComponentsFactory)
	if err != nil {
		return nil, err
	}

	err = managedBootstrapComponents.Create()
	if err != nil {
		return nil, err
	}

	return managedBootstrapComponents, nil
}

// CreateManagedNetworkComponents is the managed network components factory
func (nr *nodeRunner) CreateManagedNetworkComponents(
	coreComponents mainFactory.CoreComponentsHolder,
) (mainFactory.NetworkComponentsHandler, error) {
	decodedPreferredPeers, err := decodePreferredPeers(*nr.configs.PreferencesConfig, coreComponents.ValidatorPubKeyConverter())
	if err != nil {
		return nil, err
	}

	networkComponentsFactoryArgs := networkComp.NetworkComponentsFactoryArgs{
		P2pConfig:             *nr.configs.P2pConfig,
		MainConfig:            *nr.configs.GeneralConfig,
		RatingsConfig:         *nr.configs.RatingsConfig,
		StatusHandler:         coreComponents.StatusHandler(),
		Marshalizer:           coreComponents.InternalMarshalizer(),
		Syncer:                coreComponents.SyncTimer(),
		PreferredPeersSlices:  decodedPreferredPeers,
		BootstrapWaitTime:     common.TimeToWaitForP2PBootstrap,
		NodeOperationMode:     p2p.NormalOperation,
		ConnectionWatcherType: nr.configs.PreferencesConfig.Preferences.ConnectionWatcherType,
		P2pKeyPemFileName:     nr.configs.ConfigurationPathsHolder.P2pKey,
	}
	if nr.configs.ImportDbConfig.IsImportDBMode {
		networkComponentsFactoryArgs.BootstrapWaitTime = 0
	}
	if nr.configs.PreferencesConfig.Preferences.FullArchive {
		networkComponentsFactoryArgs.NodeOperationMode = p2p.FullArchiveMode
	}

	networkComponentsFactory, err := networkComp.NewNetworkComponentsFactory(networkComponentsFactoryArgs)
	if err != nil {
		return nil, fmt.Errorf("NewNetworkComponentsFactory failed: %w", err)
	}

	managedNetworkComponents, err := networkComp.NewManagedNetworkComponents(networkComponentsFactory)
	if err != nil {
		return nil, err
	}
	err = managedNetworkComponents.Create()
	if err != nil {
		return nil, err
	}
	return managedNetworkComponents, nil
}

// CreateManagedCoreComponents is the managed core components factory
func (nr *nodeRunner) CreateManagedCoreComponents(
	chanStopNodeProcess chan endProcess.ArgEndProcess,
) (mainFactory.CoreComponentsHandler, error) {
	statusHandlersFactory, err := factory.NewStatusHandlersFactory()
	if err != nil {
		return nil, err
	}

	coreArgs := coreComp.CoreComponentsFactoryArgs{
		Config:                *nr.configs.GeneralConfig,
		ConfigPathsHolder:     *nr.configs.ConfigurationPathsHolder,
		EpochConfig:           *nr.configs.EpochConfig,
		RoundConfig:           *nr.configs.RoundConfig,
		ImportDbConfig:        *nr.configs.ImportDbConfig,
		RatingsConfig:         *nr.configs.RatingsConfig,
		EconomicsConfig:       *nr.configs.EconomicsConfig,
		NodesFilename:         nr.configs.ConfigurationPathsHolder.Nodes,
		WorkingDirectory:      nr.configs.FlagsConfig.WorkingDir,
		ChanStopNodeProcess:   chanStopNodeProcess,
		StatusHandlersFactory: statusHandlersFactory,
	}

	coreComponentsFactory, err := coreComp.NewCoreComponentsFactory(coreArgs)
	if err != nil {
		return nil, fmt.Errorf("NewCoreComponentsFactory failed: %w", err)
	}

	managedCoreComponents, err := coreComp.NewManagedCoreComponents(coreComponentsFactory)
	if err != nil {
		return nil, err
	}

	err = managedCoreComponents.Create()
	if err != nil {
		return nil, err
	}

	return managedCoreComponents, nil
}

// CreateManagedStatusCoreComponents is the managed status core components factory
func (nr *nodeRunner) CreateManagedStatusCoreComponents() (mainFactory.StatusCoreComponentsHandler, error) {
	args := statusCore.StatusCoreComponentsFactoryArgs{
		Config: *nr.configs.GeneralConfig,
	}

	statusCoreComponentsFactory := statusCore.NewStatusCoreComponentsFactory(args)
	managedStatusCoreComponents, err := statusCore.NewManagedStatusCoreComponents(statusCoreComponentsFactory)
	if err != nil {
		return nil, err
	}

	err = managedStatusCoreComponents.Create()
	if err != nil {
		return nil, err
	}

	return managedStatusCoreComponents, nil
}

// CreateManagedCryptoComponents is the managed crypto components factory
func (nr *nodeRunner) CreateManagedCryptoComponents(
	coreComponents mainFactory.CoreComponentsHolder,
) (mainFactory.CryptoComponentsHandler, error) {
	configs := nr.configs
	validatorKeyPemFileName := configs.ConfigurationPathsHolder.ValidatorKey
	cryptoComponentsHandlerArgs := cryptoComp.CryptoComponentsFactoryArgs{
		ValidatorKeyPemFileName:              validatorKeyPemFileName,
		SkIndex:                              configs.FlagsConfig.ValidatorKeyIndex,
		Config:                               *configs.GeneralConfig,
		CoreComponentsHolder:                 coreComponents,
		ActivateBLSPubKeyMessageVerification: configs.SystemSCConfig.StakingSystemSCConfig.ActivateBLSPubKeyMessageVerification,
		KeyLoader:                            &core.KeyLoader{},
		ImportModeNoSigCheck:                 configs.ImportDbConfig.ImportDbNoSigCheckFlag,
		IsInImportMode:                       configs.ImportDbConfig.IsImportDBMode,
		EnableEpochs:                         configs.EpochConfig.EnableEpochs,
		NoKeyProvided:                        configs.FlagsConfig.NoKeyProvided,
	}

	cryptoComponentsFactory, err := cryptoComp.NewCryptoComponentsFactory(cryptoComponentsHandlerArgs)
	if err != nil {
		return nil, fmt.Errorf("NewCryptoComponentsFactory failed: %w", err)
	}

	managedCryptoComponents, err := cryptoComp.NewManagedCryptoComponents(cryptoComponentsFactory)
	if err != nil {
		return nil, err
	}

	err = managedCryptoComponents.Create()
	if err != nil {
		return nil, err
	}

	return managedCryptoComponents, nil
}

func closeAllComponents(
	healthService io.Closer,
	facade mainFactory.Closer,
	httpServer shared.UpgradeableHttpServerHandler,
	node *Node,
	chanCloseComponents chan struct{},
) {
	log.Debug("closing health service...")
	err := healthService.Close()
	log.LogIfError(err)

	log.Debug("closing http server")
	log.LogIfError(httpServer.Close())

	log.Debug("closing facade")
	log.LogIfError(facade.Close())

	log.Debug("closing node")
	log.LogIfError(node.Close())

	chanCloseComponents <- struct{}{}
}

func createStringFromRatingsData(ratingsData process.RatingsInfoHandler) string {
	metaChainStepHandler := ratingsData.MetaChainRatingsStepHandler()
	shardChainHandler := ratingsData.ShardChainRatingsStepHandler()
	computedRatingsDataStr := fmt.Sprintf(
		"meta:\n"+
			"ProposerIncrease=%v\n"+
			"ProposerDecrease=%v\n"+
			"ValidatorIncrease=%v\n"+
			"ValidatorDecrease=%v\n\n"+
			"shard:\n"+
			"ProposerIncrease=%v\n"+
			"ProposerDecrease=%v\n"+
			"ValidatorIncrease=%v\n"+
			"ValidatorDecrease=%v",
		metaChainStepHandler.ProposerIncreaseRatingStep(),
		metaChainStepHandler.ProposerDecreaseRatingStep(),
		metaChainStepHandler.ValidatorIncreaseRatingStep(),
		metaChainStepHandler.ValidatorDecreaseRatingStep(),
		shardChainHandler.ProposerIncreaseRatingStep(),
		shardChainHandler.ProposerDecreaseRatingStep(),
		shardChainHandler.ValidatorIncreaseRatingStep(),
		shardChainHandler.ValidatorDecreaseRatingStep(),
	)
	return computedRatingsDataStr
}

func cleanupStorageIfNecessary(workingDir string, cleanupStorage bool) error {
	if !cleanupStorage {
		return nil
	}

	dbPath := filepath.Join(
		workingDir,
		common.DefaultDBPath)
	log.Trace("cleaning storage", "path", dbPath)

	return os.RemoveAll(dbPath)
}

func copyConfigToStatsFolder(statsFolder string, gasScheduleFolder string, configs []string) {
	err := os.MkdirAll(statsFolder, os.ModePerm)
	log.LogIfError(err)

	err = copyDirectory(gasScheduleFolder, statsFolder)
	log.LogIfError(err)

	for _, configFile := range configs {
		copySingleFile(statsFolder, configFile)
	}
}

// TODO: add some unit tests
func copyDirectory(source string, destination string) error {
	fileDescriptors, err := ioutil.ReadDir(source)
	if err != nil {
		return err
	}

	sourceInfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	err = os.MkdirAll(destination, sourceInfo.Mode())
	if err != nil {
		return err
	}

	for _, fd := range fileDescriptors {
		srcFilePath := path.Join(source, fd.Name())
		dstFilePath := path.Join(destination, fd.Name())
		if fd.IsDir() {
			err = copyDirectory(srcFilePath, dstFilePath)
			log.LogIfError(err)
		} else {
			copySingleFile(dstFilePath, srcFilePath)
		}
	}
	return nil
}

func copySingleFile(folder string, configFile string) {
	fileName := filepath.Base(configFile)

	source, err := core.OpenFile(configFile)
	if err != nil {
		return
	}
	defer func() {
		err = source.Close()
		if err != nil {
			log.Warn("copySingleFile", "Could not close file", source.Name(), "error", err.Error())
		}
	}()

	destPath := filepath.Join(folder, fileName)
	destination, err := os.Create(destPath)
	if err != nil {
		return
	}
	defer func() {
		err = destination.Close()
		if err != nil {
			log.Warn("copySingleFile", "Could not close file", source.Name(), "error", err.Error())
		}
	}()

	_, err = io.Copy(destination, source)
	if err != nil {
		log.Warn("copySingleFile", "Could not copy file", source.Name(), "error", err.Error())
	}
}

func indexValidatorsListIfNeeded(
	outportHandler outport.OutportHandler,
	coordinator nodesCoordinator.NodesCoordinator,
	epoch uint32,
) {
	if !outportHandler.HasDrivers() {
		return
	}

	validatorsPubKeys, err := coordinator.GetAllEligibleValidatorsPublicKeys(epoch)
	if err != nil {
		log.Warn("GetAllEligibleValidatorPublicKeys for epoch 0 failed", "error", err)
	}

	if len(validatorsPubKeys) > 0 {
		outportHandler.SaveValidatorsPubKeys(validatorsPubKeys, epoch)
	}
}

func enableGopsIfNeeded(gopsEnabled bool) {
	if gopsEnabled {
		if err := agent.Listen(agent.Options{}); err != nil {
			log.Error("failure to init gops", "error", err.Error())
		}
	}

	log.Trace("gops", "enabled", gopsEnabled)
}

func decodePreferredPeers(prefConfig config.Preferences, validatorPubKeyConverter core.PubkeyConverter) ([]string, error) {
	decodedPeers := make([]string, 0)
	for _, connectionSlice := range prefConfig.Preferences.PreferredConnections {
		peerBytes, err := validatorPubKeyConverter.Decode(connectionSlice)
		if err != nil {
			return nil, fmt.Errorf("cannot decode preferred peer(%s) : %w", connectionSlice, err)
		}

		decodedPeers = append(decodedPeers, string(peerBytes))
	}

	return decodedPeers, nil
}

func createWhiteListerVerifiedTxs(generalConfig *config.Config) (process.WhiteListHandler, error) {
	whiteListCacheVerified, err := storageunit.NewCache(storageFactory.GetCacherFromConfig(generalConfig.WhiteListerVerifiedTxs))
	if err != nil {
		return nil, err
	}
	return interceptors.NewWhiteListDataVerifier(whiteListCacheVerified)
}
