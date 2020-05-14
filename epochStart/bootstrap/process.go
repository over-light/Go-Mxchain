package bootstrap

import (
	"context"
	"fmt"
	"time"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/partitioning"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/syncer"
	"github.com/ElrondNetwork/elrond-go/data/trie/factory"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters/uint64ByteSlice"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	factoryDataPool "github.com/ElrondNetwork/elrond-go/dataRetriever/factory"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/factory/containers"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/factory/resolverscontainer"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/requestHandlers"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/epochStart/bootstrap/disabled"
	factoryInterceptors "github.com/ElrondNetwork/elrond-go/epochStart/bootstrap/factory"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	"github.com/ElrondNetwork/elrond-go/process/interceptors"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/ElrondNetwork/elrond-go/storage/timecache"
	"github.com/ElrondNetwork/elrond-go/update"
	"github.com/ElrondNetwork/elrond-go/update/sync"
)

var log = logger.GetOrCreate("epochStart/bootstrap")

const timeToWait = time.Minute
const trieSyncWaitTime = 10 * time.Minute
const timeBetweenRequests = 100 * time.Millisecond
const maxToRequest = 100
const gracePeriodInPercentage = float64(0.25)
const roundGracePeriod = 25

// Parameters defines the DTO for the result produced by the bootstrap component
type Parameters struct {
	Epoch       uint32
	SelfShardId uint32
	NumOfShards uint32
	NodesConfig *sharding.NodesCoordinatorRegistry
}

// ComponentsNeededForBootstrap holds the components which need to be initialized from network
type ComponentsNeededForBootstrap struct {
	EpochStartMetaBlock *block.MetaBlock
	PreviousEpochStart  *block.MetaBlock
	ShardHeader         *block.Header
	NodesConfig         *sharding.NodesCoordinatorRegistry
	Headers             map[string]data.HeaderHandler
	ShardCoordinator    sharding.Coordinator
	UserAccountTries    map[string]data.Trie
	PeerAccountTries    map[string]data.Trie
	PendingMiniBlocks   map[string]*block.MiniBlock
}

// epochStartBootstrap will handle requesting the needed data to start when joining late the network
type epochStartBootstrap struct {
	// should come via arguments
	coreComponentsHolder       BootstrapCoreComponentsHolder
	cryptoComponentsHolder     BootstrapCryptoComponentsHolder
	messenger                  Messenger
	generalConfig              config.Config
	economicsData              *economics.EconomicsData
	shardCoordinator           sharding.Coordinator
	genesisNodesConfig         sharding.GenesisNodesSetupHandler
	genesisShardCoordinator    sharding.Coordinator
	destinationShardAsObserver uint32
	rater                      sharding.ChanceComputer
	trieContainer              state.TriesHolder
	trieStorageManagers        map[string]data.StorageManager
	nodeShuffler               sharding.NodesShuffler
	rounder                    epochStart.Rounder

	// created components
	requestHandler            process.RequestHandler
	interceptorContainer      process.InterceptorsContainer
	dataPool                  dataRetriever.PoolsHolder
	miniBlocksSyncer          epochStart.PendingMiniBlocksSyncHandler
	headersSyncer             epochStart.HeadersByHashSyncer
	epochStartMetaBlockSyncer epochStart.StartOfEpochMetaSyncer
	nodesConfigHandler        StartOfEpochNodesConfigHandler
	whiteListHandler          update.WhiteListHandler
	whiteListerVerifiedTxs    update.WhiteListHandler
	storageOpenerHandler      storage.UnitOpenerHandler
	latestStorageDataProvider storage.LatestStorageDataProviderHandler

	// gathered data
	epochStartMeta     *block.MetaBlock
	prevEpochStartMeta *block.MetaBlock
	syncedHeaders      map[string]data.HeaderHandler
	nodesConfig        *sharding.NodesCoordinatorRegistry
	userAccountTries   map[string]data.Trie
	peerAccountTries   map[string]data.Trie
	baseData           baseDataInStorage
	shuffledOut        bool
}

type baseDataInStorage struct {
	shardId         uint32
	numberOfShards  uint32
	lastRound       int64
	lastEpoch       uint32
	epochStartRound uint64
	storageExists   bool
}

// ArgsEpochStartBootstrap holds the arguments needed for creating an epoch start data provider component
type ArgsEpochStartBootstrap struct {
	CoreComponentsHolder       BootstrapCoreComponentsHolder
	CryptoComponentsHolder     BootstrapCryptoComponentsHolder
	DestinationShardAsObserver uint32
	TrieStorageManagers        map[string]data.StorageManager
	Messenger                  Messenger
	GeneralConfig              config.Config
	EconomicsData              *economics.EconomicsData
	GenesisNodesConfig         sharding.GenesisNodesSetupHandler
	GenesisShardCoordinator    sharding.Coordinator
	StorageUnitOpener          storage.UnitOpenerHandler
	LatestStorageDataProvider  storage.LatestStorageDataProviderHandler
	Rater                      sharding.ChanceComputer
	TrieContainer              state.TriesHolder
	NodeShuffler               sharding.NodesShuffler
	Rounder                    epochStart.Rounder
}

// NewEpochStartBootstrap will return a new instance of epochStartBootstrap
func NewEpochStartBootstrap(args ArgsEpochStartBootstrap) (*epochStartBootstrap, error) {
	err := checkArguments(args)
	if err != nil {
		return nil, err
	}

	epochStartProvider := &epochStartBootstrap{
		coreComponentsHolder:       args.CoreComponentsHolder,
		cryptoComponentsHolder:     args.CryptoComponentsHolder,
		messenger:                  args.Messenger,
		generalConfig:              args.GeneralConfig,
		economicsData:              args.EconomicsData,
		genesisNodesConfig:         args.GenesisNodesConfig,
		genesisShardCoordinator:    args.GenesisShardCoordinator,
		rater:                      args.Rater,
		destinationShardAsObserver: args.DestinationShardAsObserver,
		trieContainer:              args.TrieContainer,
		trieStorageManagers:        args.TrieStorageManagers,
		nodeShuffler:               args.NodeShuffler,
		rounder:                    args.Rounder,
		storageOpenerHandler:       args.StorageUnitOpener,
		latestStorageDataProvider:  args.LatestStorageDataProvider,
		shuffledOut:                false,
	}

	whiteListCache, err := storageUnit.NewCache(
		storageUnit.CacheType(epochStartProvider.generalConfig.WhiteListPool.Type),
		epochStartProvider.generalConfig.WhiteListPool.Size,
		epochStartProvider.generalConfig.WhiteListPool.Shards,
	)
	if err != nil {
		return nil, err
	}

	epochStartProvider.whiteListHandler, err = interceptors.NewWhiteListDataVerifier(whiteListCache)
	if err != nil {
		return nil, err
	}

	epochStartProvider.whiteListerVerifiedTxs, err = interceptors.NewDisabledWhiteListDataVerifier()
	if err != nil {
		return nil, err
	}

	return epochStartProvider, nil
}

func (e *epochStartBootstrap) isStartInEpochZero() bool {
	startTime := time.Unix(e.genesisNodesConfig.GetStartTime(), 0)
	isCurrentTimeBeforeGenesis := time.Since(startTime) < 0
	if isCurrentTimeBeforeGenesis {
		return true
	}

	currentRound := e.rounder.Index()
	epochEndPlusGracePeriod := float64(e.generalConfig.EpochStartConfig.RoundsPerEpoch) * (gracePeriodInPercentage + 1.0)
	log.Debug("IsStartInEpochZero", "currentRound", currentRound, "epochEndRound", epochEndPlusGracePeriod)
	return float64(currentRound) < epochEndPlusGracePeriod
}

func (e *epochStartBootstrap) prepareEpochZero() (Parameters, error) {
	parameters := Parameters{
		Epoch:       0,
		SelfShardId: e.genesisShardCoordinator.SelfId(),
		NumOfShards: e.genesisShardCoordinator.NumberOfShards(),
	}
	return parameters, nil
}

// Bootstrap runs the fast bootstrap method from the network or local storage
func (e *epochStartBootstrap) Bootstrap() (Parameters, error) {
	if !e.generalConfig.GeneralSettings.StartInEpochEnabled {
		log.Warn("fast bootstrap is disabled")

		e.initializeFromLocalStorage()

		return Parameters{
			Epoch:       e.baseData.lastEpoch,
			SelfShardId: e.genesisShardCoordinator.SelfId(),
			NumOfShards: e.genesisShardCoordinator.NumberOfShards(),
		}, nil
	}

	defer func() {
		log.Debug("unregistering all message processor")
		errMessenger := e.messenger.UnregisterAllMessageProcessors()
		log.LogIfError(errMessenger)
	}()

	var err error
	e.shardCoordinator, err = sharding.NewMultiShardCoordinator(e.genesisShardCoordinator.NumberOfShards(), core.MetachainShardId)
	if err != nil {
		return Parameters{}, err
	}

	e.dataPool, err = factoryDataPool.NewDataPoolFromConfig(
		factoryDataPool.ArgsDataPool{
			Config:           &e.generalConfig,
			EconomicsData:    e.economicsData,
			ShardCoordinator: e.shardCoordinator,
		},
	)
	if err != nil {
		return Parameters{}, err
	}

	isCurrentEpochSaved := e.computeIfCurrentEpochIsSaved()
	if isCurrentEpochSaved || e.isStartInEpochZero() {
		if e.baseData.lastEpoch == 0 {
			return e.prepareEpochZero()
		}

		parameters, errPrepare := e.prepareEpochFromStorage()
		if errPrepare == nil {
			return parameters, nil
		}

		if e.shuffledOut {
			// sync was already tried - not need to continue from here
			return Parameters{}, err
		}

		log.Debug("could not start from storage - will try sync for start in epoch", "error", errPrepare)
	}

	err = e.prepareComponentsToSyncFromNetwork()
	if err != nil {
		return Parameters{}, err
	}

	e.epochStartMeta, err = e.epochStartMetaBlockSyncer.SyncEpochStartMeta(timeToWait)
	if err != nil {
		return Parameters{}, err
	}
	log.Debug("start in epoch bootstrap: got epoch start meta header", "epoch", e.epochStartMeta.Epoch, "nonce", e.epochStartMeta.Nonce)

	err = e.createSyncers()
	if err != nil {
		return Parameters{}, err
	}

	params, err := e.requestAndProcessing()
	if err != nil {
		return Parameters{}, err
	}

	return params, nil
}

func (e *epochStartBootstrap) computeIfCurrentEpochIsSaved() bool {
	e.initializeFromLocalStorage()
	if !e.baseData.storageExists {
		return false
	}

	computedRound := e.rounder.Index()
	log.Debug("computed round", "round", computedRound, "lastRound", e.baseData.lastRound)
	if computedRound-e.baseData.lastRound < roundGracePeriod {
		return true
	}

	roundsSinceEpochStart := computedRound - int64(e.baseData.epochStartRound)
	log.Debug("epoch start round", "round", e.baseData.epochStartRound, "roundsSinceEpochStart", roundsSinceEpochStart)
	epochEndPlusGracePeriod := float64(e.generalConfig.EpochStartConfig.RoundsPerEpoch) * (gracePeriodInPercentage + 1.0)
	return float64(roundsSinceEpochStart) < epochEndPlusGracePeriod
}

func (e *epochStartBootstrap) prepareComponentsToSyncFromNetwork() error {
	err := e.createRequestHandler()
	if err != nil {
		return err
	}

	argsEpochStartSyncer := ArgsNewEpochStartMetaSyncer{
		CoreComponentsHolder:   e.coreComponentsHolder,
		CryptoComponentsHolder: e.cryptoComponentsHolder,
		RequestHandler:         e.requestHandler,
		Messenger:              e.messenger,
		ShardCoordinator:       e.shardCoordinator,
		EconomicsData:          e.economicsData,
		WhitelistHandler:       e.whiteListHandler,
	}
	e.epochStartMetaBlockSyncer, err = NewEpochStartMetaSyncer(argsEpochStartSyncer)
	if err != nil {
		return err
	}

	return nil
}

func (e *epochStartBootstrap) createSyncers() error {
	var err error

	args := factoryInterceptors.ArgsEpochStartInterceptorContainer{
		Config:                 e.generalConfig,
		ShardCoordinator:       e.shardCoordinator,
		ProtoMarshalizer:       e.coreComponentsHolder.InternalMarshalizer(),
		TxSignMarshalizer:      e.coreComponentsHolder.TxMarshalizer(),
		Hasher:                 e.coreComponentsHolder.Hasher(),
		Messenger:              e.messenger,
		DataPool:               e.dataPool,
		SingleSigner:           e.cryptoComponentsHolder.TxSingleSigner(),
		BlockSingleSigner:      e.cryptoComponentsHolder.BlockSigner(),
		KeyGen:                 e.cryptoComponentsHolder.TxSignKeyGen(),
		BlockKeyGen:            e.cryptoComponentsHolder.BlockSignKeyGen(),
		WhiteListHandler:       e.whiteListHandler,
		WhiteListerVerifiedTxs: e.whiteListerVerifiedTxs,
		ChainID:                []byte(e.coreComponentsHolder.ChainID()),
		AddressPubkeyConv:      e.coreComponentsHolder.AddressPubKeyConverter(),
		NonceConverter:         e.coreComponentsHolder.Uint64ByteSliceConverter(),
	}

	e.interceptorContainer, err = factoryInterceptors.NewEpochStartInterceptorsContainer(args)
	if err != nil {
		return err
	}

	syncMiniBlocksArgs := sync.ArgsNewPendingMiniBlocksSyncer{
		Storage:        disabled.CreateMemUnit(),
		Cache:          e.dataPool.MiniBlocks(),
		Marshalizer:    e.coreComponentsHolder.InternalMarshalizer(),
		RequestHandler: e.requestHandler,
	}
	e.miniBlocksSyncer, err = sync.NewPendingMiniBlocksSyncer(syncMiniBlocksArgs)
	if err != nil {
		return err
	}

	syncMissingHeadersArgs := sync.ArgsNewMissingHeadersByHashSyncer{
		Storage:        disabled.CreateMemUnit(),
		Cache:          e.dataPool.Headers(),
		Marshalizer:    e.coreComponentsHolder.InternalMarshalizer(),
		RequestHandler: e.requestHandler,
	}
	e.headersSyncer, err = sync.NewMissingheadersByHashSyncer(syncMissingHeadersArgs)
	if err != nil {
		return err
	}

	return nil
}

func (e *epochStartBootstrap) syncHeadersFrom(meta *block.MetaBlock) (map[string]data.HeaderHandler, error) {
	hashesToRequest := make([][]byte, 0, len(meta.EpochStart.LastFinalizedHeaders)+1)
	shardIds := make([]uint32, 0, len(meta.EpochStart.LastFinalizedHeaders)+1)

	for _, epochStartData := range meta.EpochStart.LastFinalizedHeaders {
		hashesToRequest = append(hashesToRequest, epochStartData.HeaderHash)
		shardIds = append(shardIds, epochStartData.ShardID)
	}

	if meta.Epoch > 1 { // no need to request genesis block
		hashesToRequest = append(hashesToRequest, meta.EpochStart.Economics.PrevEpochStartHash)
		shardIds = append(shardIds, core.MetachainShardId)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeToWait)
	err := e.headersSyncer.SyncMissingHeadersByHash(shardIds, hashesToRequest, ctx)
	cancel()
	if err != nil {
		return nil, err
	}

	syncedHeaders, err := e.headersSyncer.GetHeaders()
	if err != nil {
		return nil, err
	}

	if meta.Epoch == 1 {
		syncedHeaders[string(meta.EpochStart.Economics.PrevEpochStartHash)] = &block.MetaBlock{}
	}

	return syncedHeaders, nil
}

// Bootstrap will handle requesting and receiving the needed information the node will bootstrap from
func (e *epochStartBootstrap) requestAndProcessing() (Parameters, error) {
	var err error
	e.baseData.numberOfShards = uint32(len(e.epochStartMeta.EpochStart.LastFinalizedHeaders))
	e.baseData.lastEpoch = e.epochStartMeta.Epoch

	e.syncedHeaders, err = e.syncHeadersFrom(e.epochStartMeta)
	if err != nil {
		return Parameters{}, err
	}
	log.Debug("start in epoch bootstrap: got shard headers and previous epoch start meta block")

	prevEpochStartMetaHash := e.epochStartMeta.EpochStart.Economics.PrevEpochStartHash
	prevEpochStartMeta, ok := e.syncedHeaders[string(prevEpochStartMetaHash)].(*block.MetaBlock)
	if !ok {
		return Parameters{}, epochStart.ErrWrongTypeAssertion
	}
	e.prevEpochStartMeta = prevEpochStartMeta

	pubKeyBytes, err := e.cryptoComponentsHolder.PublicKey().ToByteArray()
	if err != nil {
		return Parameters{}, err
	}

	err = e.processNodesConfig(pubKeyBytes)
	if err != nil {
		return Parameters{}, err
	}
	log.Debug("start in epoch bootstrap: processNodesConfig")

	e.saveSelfShardId()
	e.shardCoordinator, err = sharding.NewMultiShardCoordinator(e.baseData.numberOfShards, e.baseData.shardId)
	if err != nil {
		return Parameters{}, fmt.Errorf("%w numberOfShards=%v shardId=%v", err, e.baseData.numberOfShards, e.baseData.shardId)
	}
	log.Debug("start in epoch bootstrap: shardCoordinator", "numOfShards", e.baseData.numberOfShards, "shardId", e.baseData.shardId)

	if e.shardCoordinator.SelfId() != e.genesisShardCoordinator.SelfId() {
		err = e.createTriesForNewShardId(e.shardCoordinator.SelfId())
		if err != nil {
			return Parameters{}, err
		}
	}

	err = e.messenger.CreateTopic(core.ConsensusTopic+e.shardCoordinator.CommunicationIdentifier(e.shardCoordinator.SelfId()), true)
	if err != nil {
		return Parameters{}, err
	}

	if e.shardCoordinator.SelfId() == core.MetachainShardId {
		err = e.requestAndProcessForMeta()
		if err != nil {
			return Parameters{}, err
		}
	} else {
		err = e.requestAndProcessForShard()
		if err != nil {
			return Parameters{}, err
		}
	}

	parameters := Parameters{
		Epoch:       e.baseData.lastEpoch,
		SelfShardId: e.baseData.shardId,
		NumOfShards: e.baseData.numberOfShards,
		NodesConfig: e.nodesConfig,
	}

	return parameters, nil
}

func (e *epochStartBootstrap) saveSelfShardId() {
	if e.baseData.shardId != core.AllShardId {
		return
	}

	e.baseData.shardId = e.destinationShardAsObserver

	if e.baseData.shardId > e.baseData.numberOfShards &&
		e.baseData.shardId != core.MetachainShardId {
		e.baseData.shardId = e.genesisShardCoordinator.SelfId()
	}
}

func (e *epochStartBootstrap) processNodesConfig(pubKey []byte) error {
	var err error
	argsNewValidatorStatusSyncers := ArgsNewSyncValidatorStatus{
		DataPool:           e.dataPool,
		Marshalizer:        e.coreComponentsHolder.InternalMarshalizer(),
		RequestHandler:     e.requestHandler,
		ChanceComputer:     e.rater,
		GenesisNodesConfig: e.genesisNodesConfig,
		NodeShuffler:       e.nodeShuffler,
		Hasher:             e.coreComponentsHolder.Hasher(),
		PubKey:             pubKey,
		ShardIdAsObserver:  e.destinationShardAsObserver,
	}
	e.nodesConfigHandler, err = NewSyncValidatorStatus(argsNewValidatorStatusSyncers)
	if err != nil {
		return err
	}

	e.nodesConfig, e.baseData.shardId, err = e.nodesConfigHandler.NodesConfigFromMetaBlock(e.epochStartMeta, e.prevEpochStartMeta)
	return err
}

func (e *epochStartBootstrap) requestAndProcessForMeta() error {
	var err error

	log.Debug("start in epoch bootstrap: started syncPeerAccountsState")
	err = e.syncPeerAccountsState(e.epochStartMeta.ValidatorStatsRootHash)
	if err != nil {
		return err
	}
	log.Debug("start in epoch bootstrap: syncPeerAccountsState", "peer account tries map length", len(e.peerAccountTries))

	err = e.syncUserAccountsState(e.epochStartMeta.RootHash)
	if err != nil {
		return err
	}

	components := &ComponentsNeededForBootstrap{
		EpochStartMetaBlock: e.epochStartMeta,
		PreviousEpochStart:  e.prevEpochStartMeta,
		NodesConfig:         e.nodesConfig,
		Headers:             e.syncedHeaders,
		ShardCoordinator:    e.shardCoordinator,
		UserAccountTries:    e.userAccountTries,
		PeerAccountTries:    e.peerAccountTries,
	}

	storageHandlerComponent, err := NewMetaStorageHandler(
		e.generalConfig,
		e.shardCoordinator,
		e.coreComponentsHolder.PathHandler(),
		e.coreComponentsHolder.InternalMarshalizer(),
		e.coreComponentsHolder.Hasher(),
		e.epochStartMeta.Epoch,
		e.coreComponentsHolder.Uint64ByteSliceConverter(),
	)
	if err != nil {
		return err
	}

	errSavingToStorage := storageHandlerComponent.SaveDataToStorage(components)
	if errSavingToStorage != nil {
		return errSavingToStorage
	}

	return nil
}

func (e *epochStartBootstrap) findSelfShardEpochStartData() (block.EpochStartShardData, error) {
	var epochStartData block.EpochStartShardData
	for _, shardData := range e.epochStartMeta.EpochStart.LastFinalizedHeaders {
		if shardData.ShardID == e.shardCoordinator.SelfId() {
			return shardData, nil
		}
	}
	return epochStartData, epochStart.ErrEpochStartDataForShardNotFound
}

func (e *epochStartBootstrap) requestAndProcessForShard() error {
	epochStartData, err := e.findSelfShardEpochStartData()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeToWait)
	err = e.miniBlocksSyncer.SyncPendingMiniBlocks(epochStartData.PendingMiniBlockHeaders, ctx)
	cancel()
	if err != nil {
		return err
	}

	pendingMiniBlocks, err := e.miniBlocksSyncer.GetMiniBlocks()
	if err != nil {
		return err
	}
	log.Debug("start in epoch bootstrap: GetMiniBlocks")

	shardIds := []uint32{
		core.MetachainShardId,
		core.MetachainShardId,
	}
	hashesToRequest := [][]byte{
		epochStartData.LastFinishedMetaBlock,
		epochStartData.FirstPendingMetaBlock,
	}

	e.headersSyncer.ClearFields()
	ctx, cancel = context.WithTimeout(context.Background(), timeToWait)
	err = e.headersSyncer.SyncMissingHeadersByHash(shardIds, hashesToRequest, ctx)
	cancel()
	if err != nil {
		return err
	}

	neededHeaders, err := e.headersSyncer.GetHeaders()
	if err != nil {
		return err
	}
	log.Debug("start in epoch bootstrap: SyncMissingHeadersByHash")

	for hash, hdr := range neededHeaders {
		e.syncedHeaders[hash] = hdr
	}

	ownShardHdr, ok := e.syncedHeaders[string(epochStartData.HeaderHash)].(*block.Header)
	if !ok {
		return epochStart.ErrWrongTypeAssertion
	}

	log.Debug("start in epoch bootstrap: started syncUserAccountsState")
	err = e.syncUserAccountsState(ownShardHdr.RootHash)
	if err != nil {
		return err
	}
	log.Debug("start in epoch bootstrap: syncUserAccountsState")

	components := &ComponentsNeededForBootstrap{
		EpochStartMetaBlock: e.epochStartMeta,
		PreviousEpochStart:  e.prevEpochStartMeta,
		ShardHeader:         ownShardHdr,
		NodesConfig:         e.nodesConfig,
		Headers:             e.syncedHeaders,
		ShardCoordinator:    e.shardCoordinator,
		UserAccountTries:    e.userAccountTries,
		PeerAccountTries:    e.peerAccountTries,
		PendingMiniBlocks:   pendingMiniBlocks,
	}

	storageHandlerComponent, err := NewShardStorageHandler(
		e.generalConfig,
		e.shardCoordinator,
		e.coreComponentsHolder.PathHandler(),
		e.coreComponentsHolder.InternalMarshalizer(),
		e.coreComponentsHolder.Hasher(),
		e.baseData.lastEpoch,
		e.coreComponentsHolder.Uint64ByteSliceConverter(),
	)
	if err != nil {
		return err
	}

	errSavingToStorage := storageHandlerComponent.SaveDataToStorage(components)
	if errSavingToStorage != nil {
		return errSavingToStorage
	}

	return nil
}

func (e *epochStartBootstrap) syncUserAccountsState(rootHash []byte) error {
	argsUserAccountsSyncer := syncer.ArgsNewUserAccountsSyncer{
		ArgsNewBaseAccountsSyncer: syncer.ArgsNewBaseAccountsSyncer{
			Hasher:             e.coreComponentsHolder.Hasher(),
			Marshalizer:        e.coreComponentsHolder.InternalMarshalizer(),
			TrieStorageManager: e.trieStorageManagers[factory.UserAccountTrie],
			RequestHandler:     e.requestHandler,
			WaitTime:           trieSyncWaitTime,
			Cacher:             e.dataPool.TrieNodes(),
		},
		ShardId: e.shardCoordinator.SelfId(),
	}
	accountsDBSyncer, err := syncer.NewUserAccountsSyncer(argsUserAccountsSyncer)
	if err != nil {
		return err
	}

	err = accountsDBSyncer.SyncAccounts(rootHash)
	if err != nil {
		return err
	}

	e.userAccountTries = accountsDBSyncer.GetSyncedTries()
	return nil
}

func (e *epochStartBootstrap) createTriesForNewShardId(shardId uint32) error {
	trieFactoryArgs := factory.TrieFactoryArgs{
		EvictionWaitingListCfg:   e.generalConfig.EvictionWaitingList,
		SnapshotDbCfg:            e.generalConfig.TrieSnapshotDB,
		Marshalizer:              e.coreComponentsHolder.InternalMarshalizer(),
		Hasher:                   e.coreComponentsHolder.Hasher(),
		PathManager:              e.coreComponentsHolder.PathHandler(),
		ShardId:                  core.GetShardIdString(shardId),
		TrieStorageManagerConfig: e.generalConfig.TrieStorageManagerConfig,
	}
	trieFactory, err := factory.NewTrieFactory(trieFactoryArgs)
	if err != nil {
		return err
	}

	userStorageManager, userAccountTrie, err := trieFactory.Create(e.generalConfig.AccountsTrieStorage, e.generalConfig.StateTriesConfig.AccountsStatePruningEnabled)
	if err != nil {
		return err
	}

	e.trieContainer.Replace([]byte(factory.UserAccountTrie), userAccountTrie)
	e.trieStorageManagers[factory.UserAccountTrie] = userStorageManager

	peerStorageManager, peerAccountsTrie, err := trieFactory.Create(e.generalConfig.PeerAccountsTrieStorage, e.generalConfig.StateTriesConfig.PeerStatePruningEnabled)
	if err != nil {
		return err
	}

	e.trieContainer.Replace([]byte(factory.PeerAccountTrie), peerAccountsTrie)
	e.trieStorageManagers[factory.PeerAccountTrie] = peerStorageManager

	return nil
}

func (e *epochStartBootstrap) syncPeerAccountsState(rootHash []byte) error {
	argsValidatorAccountsSyncer := syncer.ArgsNewValidatorAccountsSyncer{
		ArgsNewBaseAccountsSyncer: syncer.ArgsNewBaseAccountsSyncer{
			Hasher:             e.coreComponentsHolder.Hasher(),
			Marshalizer:        e.coreComponentsHolder.InternalMarshalizer(),
			TrieStorageManager: e.trieStorageManagers[factory.PeerAccountTrie],
			RequestHandler:     e.requestHandler,
			WaitTime:           trieSyncWaitTime,
			Cacher:             e.dataPool.TrieNodes(),
		},
	}
	accountsDBSyncer, err := syncer.NewValidatorAccountsSyncer(argsValidatorAccountsSyncer)
	if err != nil {
		return err
	}

	err = accountsDBSyncer.SyncAccounts(rootHash)
	if err != nil {
		return err
	}

	e.peerAccountTries = accountsDBSyncer.GetSyncedTries()
	return nil
}

func (e *epochStartBootstrap) createRequestHandler() error {
	dataPacker, err := partitioning.NewSimpleDataPacker(e.coreComponentsHolder.InternalMarshalizer())
	if err != nil {
		return err
	}

	storageService := disabled.NewChainStorer()

	resolversContainerArgs := resolverscontainer.FactoryArgs{
		ShardCoordinator:           e.shardCoordinator,
		Messenger:                  e.messenger,
		Store:                      storageService,
		Marshalizer:                e.coreComponentsHolder.InternalMarshalizer(),
		DataPools:                  e.dataPool,
		Uint64ByteSliceConverter:   uint64ByteSlice.NewBigEndianConverter(),
		NumConcurrentResolvingJobs: 10,
		DataPacker:                 dataPacker,
		TriesContainer:             e.trieContainer,
		SizeCheckDelta:             0,
		InputAntifloodHandler:      disabled.NewAntiFloodHandler(),
		OutputAntifloodHandler:     disabled.NewAntiFloodHandler(),
	}
	resolverFactory, err := resolverscontainer.NewMetaResolversContainerFactory(resolversContainerArgs)
	if err != nil {
		return err
	}

	container, err := resolverFactory.Create()
	if err != nil {
		return err
	}

	finder, err := containers.NewResolversFinder(container, e.shardCoordinator)
	if err != nil {
		return err
	}

	requestedItemsHandler := timecache.NewTimeCache(timeBetweenRequests)
	e.requestHandler, err = requestHandlers.NewResolverRequestHandler(
		finder,
		requestedItemsHandler,
		e.whiteListHandler,
		maxToRequest,
		core.MetachainShardId,
		timeBetweenRequests,
	)
	return err
}

// IsInterfaceNil returns true if there is no value under the interface
func (e *epochStartBootstrap) IsInterfaceNil() bool {
	return e == nil
}
