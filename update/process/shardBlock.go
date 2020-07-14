package process

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/update"
)

// ArgsNewShardBlockCreatorAfterHardFork defines the arguments structure to create a new shard block creator
type ArgsNewShardBlockCreatorAfterHardFork struct {
	ShardCoordinator   sharding.Coordinator
	TxCoordinator      process.TransactionCoordinator
	PendingTxProcessor update.PendingTransactionProcessor
	ImportHandler      update.ImportHandler
	Marshalizer        marshal.Marshalizer
	Hasher             hashing.Hasher
	SelfShardID        uint32
	DataPool           dataRetriever.PoolsHolder
	Storage            dataRetriever.StorageService
}

type shardBlockCreator struct {
	shardCoordinator   sharding.Coordinator
	txCoordinator      process.TransactionCoordinator
	pendingTxProcessor update.PendingTransactionProcessor
	importHandler      update.ImportHandler
	marshalizer        marshal.Marshalizer
	hasher             hashing.Hasher
	selfShardID        uint32
	dataPool           dataRetriever.PoolsHolder
	storage            dataRetriever.StorageService
}

// NewShardBlockCreatorAfterHardFork creates a shard block processor for the first block after hardfork
func NewShardBlockCreatorAfterHardFork(args ArgsNewShardBlockCreatorAfterHardFork) (*shardBlockCreator, error) {
	if check.IfNil(args.ImportHandler) {
		return nil, update.ErrNilImportHandler
	}
	if check.IfNil(args.Marshalizer) {
		return nil, update.ErrNilMarshalizer
	}
	if check.IfNil(args.Hasher) {
		return nil, update.ErrNilHasher
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, update.ErrNilShardCoordinator
	}
	if check.IfNil(args.TxCoordinator) {
		return nil, update.ErrNilTxCoordinator
	}
	if check.IfNil(args.PendingTxProcessor) {
		return nil, update.ErrNilPendingTxProcessor
	}
	if check.IfNil(args.DataPool) {
		return nil, update.ErrNilDataPoolHolder
	}
	if check.IfNil(args.Storage) {
		return nil, update.ErrNilStorage
	}

	return &shardBlockCreator{
		shardCoordinator:   args.ShardCoordinator,
		txCoordinator:      args.TxCoordinator,
		pendingTxProcessor: args.PendingTxProcessor,
		importHandler:      args.ImportHandler,
		marshalizer:        args.Marshalizer,
		hasher:             args.Hasher,
		selfShardID:        args.SelfShardID,
		dataPool:           args.DataPool,
		storage:            args.Storage,
	}, nil
}

// CreateNewBlock will create a new block after hardfork import
func (s *shardBlockCreator) CreateNewBlock(
	chainID string,
	round uint64,
	nonce uint64,
	epoch uint32,
) (data.HeaderHandler, data.BodyHandler, error) {
	if len(chainID) == 0 {
		return nil, nil, update.ErrEmptyChainID
	}

	blockBody, err := s.createBody()
	if err != nil {
		return nil, nil, err
	}

	rootHash, err := s.pendingTxProcessor.RootHash()
	if err != nil {
		return nil, nil, err
	}

	shardHeader := &block.Header{
		Nonce:           nonce,
		ShardID:         s.shardCoordinator.SelfId(),
		Round:           round,
		Epoch:           epoch,
		ChainID:         []byte(chainID),
		SoftwareVersion: []byte(""),
		RootHash:        rootHash,
		RandSeed:        rootHash,
		PrevHash:        rootHash,
		PrevRandSeed:    rootHash,
		AccumulatedFees: big.NewInt(0),
		PubKeysBitmap:   []byte{1},
	}

	shardHeader.ReceiptsHash, err = s.txCoordinator.CreateReceiptsHash()
	if err != nil {
		return nil, nil, err
	}

	totalTxCount, miniBlockHeaders, err := s.createMiniBlockHeaders(blockBody)
	if err != nil {
		return nil, nil, err
	}

	shardHeader.MiniBlockHeaders = miniBlockHeaders
	shardHeader.TxCount = uint32(totalTxCount)

	metaBlockHash, err := core.CalculateHash(s.marshalizer, s.hasher, s.importHandler.GetHardForkMetaBlock())
	if err != nil {
		return nil, nil, err
	}

	shardHeader.MetaBlockHashes = [][]byte{metaBlockHash}
	shardHeader.AccumulatedFees = big.NewInt(0)
	shardHeader.DeveloperFees = big.NewInt(0)

	s.saveAllCreatedDSTMeTransactionsToCache(shardHeader, blockBody)
	s.saveAllTransactionsToStorageIfSelfShard(shardHeader, blockBody)

	return shardHeader, blockBody, nil
}

func (s *shardBlockCreator) createBody() (*block.Body, error) {
	mapTxs := s.importHandler.GetTransactions()

	s.txCoordinator.CreateBlockStarted()

	dstMeMiniBlocks, err := s.pendingTxProcessor.ProcessTransactionsDstMe(mapTxs)
	if err != nil {
		return nil, err
	}

	postProcessMiniBlocks := s.txCoordinator.CreatePostProcessMiniBlocks()

	return &block.Body{
		MiniBlocks: append(dstMeMiniBlocks, postProcessMiniBlocks...),
	}, nil
}

func (s *shardBlockCreator) createMiniBlockHeaders(body *block.Body) (int, []block.MiniBlockHeader, error) {
	if len(body.MiniBlocks) == 0 {
		return 0, nil, nil
	}

	totalTxCount := 0
	miniBlockHeaders := make([]block.MiniBlockHeader, len(body.MiniBlocks))

	for i := 0; i < len(body.MiniBlocks); i++ {
		txCount := len(body.MiniBlocks[i].TxHashes)
		totalTxCount += txCount

		miniBlockHash, err := core.CalculateHash(s.marshalizer, s.hasher, body.MiniBlocks[i])
		if err != nil {
			return 0, nil, err
		}

		miniBlockHeaders[i] = block.MiniBlockHeader{
			Hash:            miniBlockHash,
			SenderShardID:   body.MiniBlocks[i].SenderShardID,
			ReceiverShardID: body.MiniBlocks[i].ReceiverShardID,
			TxCount:         uint32(txCount),
			Type:            body.MiniBlocks[i].Type,
		}
	}

	return totalTxCount, miniBlockHeaders, nil
}

func (s *shardBlockCreator) saveAllTransactionsToStorageIfSelfShard(
	shardHdr *block.Header,
	body *block.Body,
) {
	if shardHdr.GetShardID() != s.selfShardID {
		return
	}

	mapBlockTypesTxs := make(map[block.Type]map[string]data.TransactionHandler)
	for i := 0; i < len(body.MiniBlocks); i++ {
		miniBlock := body.MiniBlocks[i]
		if _, ok := mapBlockTypesTxs[miniBlock.Type]; !ok {
			mapBlockTypesTxs[miniBlock.Type] = s.txCoordinator.GetAllCurrentUsedTxs(miniBlock.Type)
		}

		marshalizedMiniBlock, errNotCritical := s.marshalizer.Marshal(miniBlock)
		if errNotCritical != nil {
			log.Warn("saveAllTransactionsToStorageIfSelfShard.Marshal", "error", errNotCritical.Error())
			continue
		}

		errNotCritical = s.storage.Put(dataRetriever.MiniBlockUnit, shardHdr.MiniBlockHeaders[i].Hash, marshalizedMiniBlock)
		if errNotCritical != nil {
			log.Warn("saveAllTransactionsToStorageIfSelfShard.Put -> MiniBlockUnit", "error", errNotCritical.Error())
		}
	}

	mapTxs := s.importHandler.GetTransactions()
	// save transactions from imported map
	for _, miniBlock := range body.MiniBlocks {
		if miniBlock.SenderShardID == s.selfShardID {
			continue
		}

		for _, txHash := range miniBlock.TxHashes {
			tx, ok := mapTxs[string(txHash)]
			if !ok {
				log.Warn("miniblock destination me in genesis block should contain only imported txs")
				continue
			}

			unitType := dataRetriever.TransactionUnit
			switch miniBlock.Type {
			case block.TxBlock:
				unitType = dataRetriever.TransactionUnit
			case block.RewardsBlock:
				unitType = dataRetriever.RewardTransactionUnit
			case block.SmartContractResultBlock:
				unitType = dataRetriever.UnsignedTransactionUnit
			}

			marshalledData, errNotCritical := s.marshalizer.Marshal(tx)
			if errNotCritical != nil {
				log.Warn("saveAllTransactionsToStorageIfSelfShard.Marshal", "error", errNotCritical.Error())
				continue
			}

			errNotCritical = s.storage.Put(unitType, txHash, marshalledData)
			if errNotCritical != nil {
				log.Warn("saveAllTransactionsToStorageIfSelfShard.Put -> Transaction", "error", errNotCritical.Error())
			}
		}
	}
}

func (s *shardBlockCreator) saveAllCreatedDSTMeTransactionsToCache(
	shardHdr *block.Header,
	body *block.Body,
) {
	// no need to save from me, only from other genesis blocks which has results towards me
	if shardHdr.GetShardID() == s.selfShardID {
		return
	}

	mapBlockTypesTxs := make(map[block.Type]map[string]data.TransactionHandler)
	crossMiniBlocksToMe := make([]*block.MiniBlock, 0)
	for i := 0; i < len(body.MiniBlocks); i++ {
		miniBlock := body.MiniBlocks[i]
		isCrossShardDstMe := miniBlock.SenderShardID == s.shardCoordinator.SelfId() &&
			miniBlock.ReceiverShardID == s.selfShardID
		if !isCrossShardDstMe {
			continue
		}

		_ = s.dataPool.MiniBlocks().Put(shardHdr.MiniBlockHeaders[i].Hash, miniBlock, miniBlock.Size())
		crossMiniBlocksToMe = append(crossMiniBlocksToMe, miniBlock)

		if _, ok := mapBlockTypesTxs[miniBlock.Type]; !ok {
			mapBlockTypesTxs[miniBlock.Type] = s.txCoordinator.GetAllCurrentUsedTxs(miniBlock.Type)
		}
	}

	for _, miniBlock := range crossMiniBlocksToMe {
		for _, txHash := range miniBlock.TxHashes {
			tx := mapBlockTypesTxs[miniBlock.Type][string(txHash)]
			s.saveTxToCache(tx, txHash, miniBlock)
		}
	}
}

// with the current design only smart contract results are possible for this scenario - but wanted to leave it open,
// to see if other scenarios are needed as well
func (s *shardBlockCreator) saveTxToCache(
	tx data.TransactionHandler,
	txHash []byte,
	miniBlock *block.MiniBlock,
) {
	if check.IfNil(tx) {
		log.Warn("missing transaction in saveTxToCache shard genesis block", "hash", txHash, "type", miniBlock.Type)
		return
	}

	var chosenCache dataRetriever.ShardedDataCacherNotifier
	strCache := process.ShardCacherIdentifier(miniBlock.SenderShardID, miniBlock.ReceiverShardID)
	switch miniBlock.Type {
	case block.TxBlock:
		chosenCache = s.dataPool.Transactions()
	case block.RewardsBlock:
		chosenCache = s.dataPool.RewardTransactions()
	case block.SmartContractResultBlock:
		chosenCache = s.dataPool.UnsignedTransactions()
	default:
		log.Warn("invalid miniblock type to save into cache", miniBlock.Type)
		return
	}

	chosenCache.AddData(txHash, tx, tx.Size(), strCache)
}

// IsInterfaceNil returns true if underlying object is nil
func (s *shardBlockCreator) IsInterfaceNil() bool {
	return s == nil
}
