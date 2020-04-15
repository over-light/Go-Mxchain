package factory

import (
	"math"
	"path"
	"time"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	epochStartGenesis "github.com/ElrondNetwork/elrond-go/epochStart/genesis"
	"github.com/ElrondNetwork/elrond-go/epochStart/notifier"
	"github.com/ElrondNetwork/elrond-go/epochStart/shardchain"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	storageFactory "github.com/ElrondNetwork/elrond-go/storage/factory"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/ElrondNetwork/elrond-go/storage/timecache"
	"github.com/ElrondNetwork/elrond-go/update"
	"github.com/ElrondNetwork/elrond-go/update/files"
	"github.com/ElrondNetwork/elrond-go/update/genesis"
	"github.com/ElrondNetwork/elrond-go/update/sync"
)

// ArgsExporter is the argument structure to create a new exporter
type ArgsExporter struct {
	TxSignMarshalizer        marshal.Marshalizer
	Marshalizer              marshal.Marshalizer
	Hasher                   hashing.Hasher
	HeaderValidator          epochStart.HeaderValidator
	Uint64Converter          typeConverters.Uint64ByteSliceConverter
	DataPool                 dataRetriever.PoolsHolder
	StorageService           dataRetriever.StorageService
	RequestHandler           process.RequestHandler
	ShardCoordinator         sharding.Coordinator
	Messenger                p2p.Messenger
	ActiveAccountsDBs        map[state.AccountsDbIdentifier]state.AccountsAdapter
	ExistingResolvers        dataRetriever.ResolversContainer
	ExportFolder             string
	ExportTriesStorageConfig config.StorageConfig
	ExportStateStorageConfig config.StorageConfig
	WhiteListHandler         process.WhiteListHandler
	InterceptorsContainer    process.InterceptorsContainer
	MultiSigner              crypto.MultiSigner
	NodesCoordinator         sharding.NodesCoordinator
	SingleSigner             crypto.SingleSigner
	AddressPubkeyConverter   state.PubkeyConverter
	BlockKeyGen              crypto.KeyGenerator
	KeyGen                   crypto.KeyGenerator
	BlockSigner              crypto.SingleSigner
	HeaderSigVerifier        process.InterceptedHeaderSigVerifier
	ChainID                  []byte
	ValidityAttester         process.ValidityAttester
	InputAntifloodHandler    dataRetriever.P2PAntifloodHandler
	OutputAntifloodHandler   dataRetriever.P2PAntifloodHandler
}

type exportHandlerFactory struct {
	txSignMarshalizer        marshal.Marshalizer
	marshalizer              marshal.Marshalizer
	hasher                   hashing.Hasher
	headerValidator          epochStart.HeaderValidator
	uint64Converter          typeConverters.Uint64ByteSliceConverter
	dataPool                 dataRetriever.PoolsHolder
	storageService           dataRetriever.StorageService
	requestHandler           process.RequestHandler
	shardCoordinator         sharding.Coordinator
	messenger                p2p.Messenger
	activeAccountsDBs        map[state.AccountsDbIdentifier]state.AccountsAdapter
	exportFolder             string
	exportTriesStorageConfig config.StorageConfig
	exportStateStorageConfig config.StorageConfig
	whiteListHandler         process.WhiteListHandler
	interceptorsContainer    process.InterceptorsContainer
	existingResolvers        dataRetriever.ResolversContainer
	epochStartTrigger        epochStart.TriggerHandler
	accounts                 state.AccountsAdapter
	multiSigner              crypto.MultiSigner
	nodesCoordinator         sharding.NodesCoordinator
	singleSigner             crypto.SingleSigner
	blockKeyGen              crypto.KeyGenerator
	keyGen                   crypto.KeyGenerator
	blockSigner              crypto.SingleSigner
	addressPubkeyConverter   state.PubkeyConverter
	headerSigVerifier        process.InterceptedHeaderSigVerifier
	chainID                  []byte
	validityAttester         process.ValidityAttester
	resolverContainer        dataRetriever.ResolversContainer
	inputAntifloodHandler    dataRetriever.P2PAntifloodHandler
	outputAntifloodHandler   dataRetriever.P2PAntifloodHandler
}

// NewExportHandlerFactory creates an exporter factory
func NewExportHandlerFactory(args ArgsExporter) (*exportHandlerFactory, error) {
	if check.IfNil(args.ShardCoordinator) {
		return nil, update.ErrNilShardCoordinator
	}
	if check.IfNil(args.Hasher) {
		return nil, update.ErrNilHasher
	}
	if check.IfNil(args.Marshalizer) {
		return nil, update.ErrNilMarshalizer
	}
	if check.IfNil(args.HeaderValidator) {
		return nil, update.ErrNilHeaderValidator
	}
	if check.IfNil(args.Uint64Converter) {
		return nil, update.ErrNilUint64Converter
	}
	if check.IfNil(args.DataPool) {
		return nil, update.ErrNilDataPoolHolder
	}
	if check.IfNil(args.StorageService) {
		return nil, update.ErrNilStorage
	}
	if check.IfNil(args.RequestHandler) {
		return nil, update.ErrNilRequestHandler
	}
	if check.IfNil(args.Messenger) {
		return nil, update.ErrNilMessenger
	}
	if args.ActiveAccountsDBs == nil {
		return nil, update.ErrNilAccounts
	}
	if check.IfNil(args.WhiteListHandler) {
		return nil, update.ErrNilWhiteListHandler
	}
	if check.IfNil(args.InterceptorsContainer) {
		return nil, update.ErrNilInterceptorsContainer
	}
	if check.IfNil(args.ExistingResolvers) {
		return nil, update.ErrNilResolverContainer
	}
	if check.IfNil(args.MultiSigner) {
		return nil, update.ErrNilMultiSigner
	}
	if check.IfNil(args.NodesCoordinator) {
		return nil, update.ErrNilNodesCoordinator
	}
	if check.IfNil(args.SingleSigner) {
		return nil, update.ErrNilSingleSigner
	}
	if check.IfNil(args.AddressPubkeyConverter) {
		return nil, update.ErrNilPubkeyConverter
	}
	if check.IfNil(args.BlockKeyGen) {
		return nil, update.ErrNilBlockKeyGen
	}
	if check.IfNil(args.KeyGen) {
		return nil, update.ErrNilKeyGenerator
	}
	if check.IfNil(args.BlockSigner) {
		return nil, update.ErrNilBlockSigner
	}
	if check.IfNil(args.HeaderSigVerifier) {
		return nil, update.ErrNilHeaderSigVerifier
	}
	if check.IfNil(args.ValidityAttester) {
		return nil, update.ErrNilValidityAttester
	}
	if check.IfNil(args.TxSignMarshalizer) {
		return nil, update.ErrNilMarshalizer
	}
	if check.IfNil(args.InputAntifloodHandler) {
		return nil, update.ErrNilAntiFloodHandler
	}
	if check.IfNil(args.OutputAntifloodHandler) {
		return nil, update.ErrNilAntiFloodHandler
	}

	e := &exportHandlerFactory{
		txSignMarshalizer:        args.TxSignMarshalizer,
		marshalizer:              args.Marshalizer,
		hasher:                   args.Hasher,
		headerValidator:          args.HeaderValidator,
		uint64Converter:          args.Uint64Converter,
		dataPool:                 args.DataPool,
		storageService:           args.StorageService,
		requestHandler:           args.RequestHandler,
		shardCoordinator:         args.ShardCoordinator,
		messenger:                args.Messenger,
		activeAccountsDBs:        args.ActiveAccountsDBs,
		exportFolder:             args.ExportFolder,
		exportTriesStorageConfig: args.ExportTriesStorageConfig,
		exportStateStorageConfig: args.ExportStateStorageConfig,
		interceptorsContainer:    args.InterceptorsContainer,
		whiteListHandler:         args.WhiteListHandler,
		existingResolvers:        args.ExistingResolvers,
		accounts:                 args.ActiveAccountsDBs[state.UserAccountsState],
		multiSigner:              args.MultiSigner,
		nodesCoordinator:         args.NodesCoordinator,
		singleSigner:             args.SingleSigner,
		addressPubkeyConverter:   args.AddressPubkeyConverter,
		blockKeyGen:              args.BlockKeyGen,
		keyGen:                   args.KeyGen,
		blockSigner:              args.BlockSigner,
		headerSigVerifier:        args.HeaderSigVerifier,
		validityAttester:         args.ValidityAttester,
		chainID:                  args.ChainID,
		inputAntifloodHandler:    args.InputAntifloodHandler,
		outputAntifloodHandler:   args.OutputAntifloodHandler,
	}

	return e, nil
}

// Create makes a new export handler
func (e *exportHandlerFactory) Create() (update.ExportHandler, error) {
	argsPeerMiniBlocksSyncer := shardchain.ArgPeerMiniBlockSyncer{
		MiniBlocksPool: e.dataPool.MiniBlocks(),
		Requesthandler: e.requestHandler,
	}
	peerMiniBlocksSyncer, err := shardchain.NewPeerMiniBlockSyncer(argsPeerMiniBlocksSyncer)
	if err != nil {
		return nil, err
	}
	argsEpochTrigger := shardchain.ArgsShardEpochStartTrigger{
		Marshalizer:          e.marshalizer,
		Hasher:               e.hasher,
		HeaderValidator:      e.headerValidator,
		Uint64Converter:      e.uint64Converter,
		DataPool:             e.dataPool,
		Storage:              e.storageService,
		RequestHandler:       e.requestHandler,
		EpochStartNotifier:   notifier.NewEpochStartSubscriptionHandler(),
		Epoch:                0,
		Validity:             process.MetaBlockValidity,
		Finality:             process.BlockFinality,
		PeerMiniBlocksSyncer: peerMiniBlocksSyncer,
	}
	epochHandler, err := shardchain.NewEpochStartTrigger(&argsEpochTrigger)
	if err != nil {
		return nil, err
	}

	argsDataTrieFactory := ArgsNewDataTrieFactory{
		StorageConfig:    e.exportTriesStorageConfig,
		SyncFolder:       e.exportFolder,
		Marshalizer:      e.marshalizer,
		Hasher:           e.hasher,
		ShardCoordinator: e.shardCoordinator,
	}
	dataTriesContainerFactory, err := NewDataTrieFactory(argsDataTrieFactory)
	if err != nil {
		return nil, err
	}
	dataTries, err := dataTriesContainerFactory.Create()
	if err != nil {
		return nil, err
	}

	argsResolvers := ArgsNewResolversContainerFactory{
		ShardCoordinator:           e.shardCoordinator,
		Messenger:                  e.messenger,
		Marshalizer:                e.marshalizer,
		DataTrieContainer:          dataTries,
		ExistingResolvers:          e.existingResolvers,
		NumConcurrentResolvingJobs: 100,
		InputAntifloodHandler:      e.inputAntifloodHandler,
		OutputAntifloodHandler:     e.outputAntifloodHandler,
	}
	resolversFactory, err := NewResolversContainerFactory(argsResolvers)
	if err != nil {
		return nil, err
	}
	e.resolverContainer, err = resolversFactory.Create()
	if err != nil {
		return nil, err
	}

	argsAccountsSyncers := ArgsNewAccountsDBSyncersContainerFactory{
		TrieCacher:         e.dataPool.TrieNodes(),
		RequestHandler:     e.requestHandler,
		ShardCoordinator:   e.shardCoordinator,
		Hasher:             e.hasher,
		Marshalizer:        e.marshalizer,
		TrieStorageManager: dataTriesContainerFactory.TrieStorageManager(),
		WaitTime:           time.Minute,
	}
	accountsDBSyncerFactory, err := NewAccountsDBSContainerFactory(argsAccountsSyncers)
	if err != nil {
		return nil, err
	}
	accountsDBSyncerContainer, err := accountsDBSyncerFactory.Create()
	if err != nil {
		return nil, err
	}

	argsNewHeadersSync := sync.ArgsNewHeadersSyncHandler{
		StorageService:  e.storageService,
		Cache:           e.dataPool.Headers(),
		Marshalizer:     e.marshalizer,
		EpochHandler:    epochHandler,
		RequestHandler:  e.requestHandler,
		Uint64Converter: e.uint64Converter,
	}
	epochStartHeadersSyncer, err := sync.NewHeadersSyncHandler(argsNewHeadersSync)
	if err != nil {
		return nil, err
	}

	argsNewSyncAccountsDBsHandler := sync.ArgsNewSyncAccountsDBsHandler{
		AccountsDBsSyncers: accountsDBSyncerContainer,
		ActiveAccountsDBs:  e.activeAccountsDBs,
	}
	epochStartTrieSyncer, err := sync.NewSyncAccountsDBsHandler(argsNewSyncAccountsDBsHandler)
	if err != nil {
		return nil, err
	}

	argsMiniBlockSyncer := sync.ArgsNewPendingMiniBlocksSyncer{
		Storage:        e.storageService.GetStorer(dataRetriever.MiniBlockUnit),
		Cache:          e.dataPool.MiniBlocks(),
		Marshalizer:    e.marshalizer,
		RequestHandler: e.requestHandler,
	}
	epochStartMiniBlocksSyncer, err := sync.NewPendingMiniBlocksSyncer(argsMiniBlockSyncer)
	if err != nil {
		return nil, err
	}

	argsPendingTransactions := sync.ArgsNewPendingTransactionsSyncer{
		DataPools:      e.dataPool,
		Storages:       e.storageService,
		Marshalizer:    e.marshalizer,
		RequestHandler: e.requestHandler,
	}
	epochStartTransactionsSyncer, err := sync.NewPendingTransactionsSyncer(argsPendingTransactions)
	if err != nil {
		return nil, err
	}

	argsSyncState := sync.ArgsNewSyncState{
		Headers:      epochStartHeadersSyncer,
		Tries:        epochStartTrieSyncer,
		MiniBlocks:   epochStartMiniBlocksSyncer,
		Transactions: epochStartTransactionsSyncer,
	}
	stateSyncer, err := sync.NewSyncState(argsSyncState)
	if err != nil {
		return nil, err
	}

	exportStore, err := createFinalExportStorage(e.exportStateStorageConfig, e.exportFolder)
	if err != nil {
		return nil, err
	}

	argsWriter := files.ArgsNewMultiFileWriter{
		ExportFolder: e.exportFolder,
		ExportStore:  exportStore,
	}
	writer, err := files.NewMultiFileWriter(argsWriter)
	if err != nil {
		return nil, err
	}

	argsExporter := genesis.ArgsNewStateExporter{
		ShardCoordinator: e.shardCoordinator,
		StateSyncer:      stateSyncer,
		Marshalizer:      e.marshalizer,
		Writer:           writer,
		Hasher:           e.hasher,
	}
	exportHandler, err := genesis.NewStateExporter(argsExporter)
	if err != nil {
		return nil, err
	}

	e.epochStartTrigger = epochHandler
	err = e.createInterceptors()
	if err != nil {
		return nil, err
	}

	return exportHandler, nil
}

func (e *exportHandlerFactory) createInterceptors() error {
	argsInterceptors := ArgsNewFullSyncInterceptorsContainerFactory{
		Accounts:               e.accounts,
		ShardCoordinator:       e.shardCoordinator,
		NodesCoordinator:       e.nodesCoordinator,
		Messenger:              e.messenger,
		Store:                  e.storageService,
		Marshalizer:            e.marshalizer,
		TxSignMarshalizer:      e.txSignMarshalizer,
		Hasher:                 e.hasher,
		KeyGen:                 e.keyGen,
		BlockSignKeyGen:        e.blockKeyGen,
		SingleSigner:           e.singleSigner,
		BlockSingleSigner:      e.blockSigner,
		MultiSigner:            e.multiSigner,
		DataPool:               e.dataPool,
		AddressPubkeyConverter: e.addressPubkeyConverter,
		MaxTxNonceDeltaAllowed: math.MaxInt32,
		TxFeeHandler:           epochStartGenesis.NewGenesisFeeHandler(),
		BlackList:              timecache.NewTimeCache(time.Second),
		HeaderSigVerifier:      e.headerSigVerifier,
		ChainID:                e.chainID,
		SizeCheckDelta:         math.MaxUint32,
		ValidityAttester:       e.validityAttester,
		EpochStartTrigger:      e.epochStartTrigger,
		WhiteListHandler:       e.whiteListHandler,
		InterceptorsContainer:  e.interceptorsContainer,
		AntifloodHandler:       e.inputAntifloodHandler,
	}
	fullSyncInterceptors, err := NewFullSyncInterceptorsContainerFactory(argsInterceptors)
	if err != nil {
		return err
	}

	interceptorsContainer, err := fullSyncInterceptors.Create()
	if err != nil {
		return err
	}

	e.interceptorsContainer = interceptorsContainer
	return nil
}

func createFinalExportStorage(storageConfig config.StorageConfig, folder string) (storage.Storer, error) {
	dbConfig := storageFactory.GetDBFromConfig(storageConfig.DB)
	dbConfig.FilePath = path.Join(folder, storageConfig.DB.FilePath)
	accountsTrieStorage, err := storageUnit.NewStorageUnitFromConf(
		storageFactory.GetCacherFromConfig(storageConfig.Cache),
		dbConfig,
		storageFactory.GetBloomFromConfig(storageConfig.Bloom),
	)
	if err != nil {
		return nil, err
	}

	return accountsTrieStorage, nil
}

// IsInterfaceNil returns true if underlying object is nil
func (e *exportHandlerFactory) IsInterfaceNil() bool {
	return e == nil
}
