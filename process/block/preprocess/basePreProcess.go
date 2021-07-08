package preprocess

import (
	"math/big"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/storage"
)

const additionalTimeForCreatingScheduledMiniBlocks = 150 * time.Millisecond

type txType int32

const (
	nonScTx txType = 0
	scTx    txType = 1
)

type gasConsumedInfo struct {
	gasConsumedByMiniBlocksInSenderShard  uint64
	gasConsumedByMiniBlockInReceiverShard uint64
	totalGasConsumedInSelfShard           uint64
}

type txAndMbInfo struct {
	numNewTxs                      int
	numNewMiniBlocks               int
	isReceiverSmartContractAddress bool
	isCrossShardScCallOrSpecialTx  bool
	txType                         txType
}

type scheduledTxAndMbInfo struct {
	numNewTxs            int
	numNewMiniBlocks     int
	isCrossShardScCallTx bool
}

type processedTxsInfo struct {
	numTxsAdded                        int
	numBadTxs                          int
	numTxsSkipped                      int
	numTxsFailed                       int
	numTxsWithInitialBalanceConsumed   int
	numCrossShardScCallsOrSpecialTxs   int
	totalTimeUsedForProcess            time.Duration
	totalTimeUsedForComputeGasConsumed time.Duration
}

type createAndProcessMiniBlocksInfo struct {
	mapSCTxs                                 map[string]struct{}
	mapTxsForShard                           map[uint32]int
	mapScsForShard                           map[uint32]int
	mapCrossShardScCallsOrSpecialTxs         map[uint32]int
	mapGasConsumedByMiniBlockInReceiverShard map[uint32]map[txType]uint64
	mapMiniBlocks                            map[uint32]*block.MiniBlock
	senderAddressToSkip                      []byte
	maxCrossShardScCallsOrSpecialTxsPerShard int
	firstInvalidTxFound                      bool
	firstCrossShardScCallOrSpecialTxFound    bool
	processingInfo                           processedTxsInfo
	gasInfo                                  gasConsumedInfo
}

type scheduledTxsInfo struct {
	numScheduledTxsAdded                        int
	numScheduledBadTxs                          int
	numScheduledTxsSkipped                      int
	numScheduledTxsWithInitialBalanceConsumed   int
	numScheduledCrossShardScCalls               int
	totalTimeUsedForScheduledVerify             time.Duration
	totalTimeUsedForScheduledComputeGasConsumed time.Duration
}

type createScheduledMiniBlocksInfo struct {
	mapMiniBlocks                            map[uint32]*block.MiniBlock
	mapCrossShardScCallTxs                   map[uint32]int
	maxCrossShardScCallTxsPerShard           int
	schedulingInfo                           scheduledTxsInfo
	firstCrossShardScCallTxFound             bool
	mapGasConsumedByMiniBlockInReceiverShard map[uint32]map[txType]uint64
	gasInfo                                  gasConsumedInfo
	senderAddressToSkip                      []byte
}

type txShardInfo struct {
	senderShardID   uint32
	receiverShardID uint32
}

type txInfo struct {
	tx data.TransactionHandler
	*txShardInfo
}

type txsForBlock struct {
	missingTxs     int
	mutTxsForBlock sync.RWMutex
	txHashAndInfo  map[string]*txInfo
}

type basePreProcess struct {
	gasTracker
	hasher               hashing.Hasher
	marshalizer          marshal.Marshalizer
	blockSizeComputation BlockSizeComputationHandler
	balanceComputation   BalanceComputationHandler
	accounts             state.AccountsAdapter
	pubkeyConverter      core.PubkeyConverter
}

func (bpp *basePreProcess) removeBlockDataFromPools(
	body *block.Body,
	miniBlockPool storage.Cacher,
	txPool dataRetriever.ShardedDataCacherNotifier,
	isMiniBlockCorrect func(block.Type) bool,
) error {
	err := bpp.removeTxsFromPools(body, txPool, isMiniBlockCorrect)
	if err != nil {
		return err
	}

	err = bpp.removeMiniBlocksFromPools(body, miniBlockPool, isMiniBlockCorrect)
	if err != nil {
		return err
	}

	return nil
}

func (bpp *basePreProcess) removeTxsFromPools(
	body *block.Body,
	txPool dataRetriever.ShardedDataCacherNotifier,
	isMiniBlockCorrect func(block.Type) bool,
) error {
	if check.IfNil(body) {
		return process.ErrNilTxBlockBody
	}
	if check.IfNil(txPool) {
		return process.ErrNilTransactionPool
	}

	for i := 0; i < len(body.MiniBlocks); i++ {
		currentMiniBlock := body.MiniBlocks[i]
		if !isMiniBlockCorrect(currentMiniBlock.Type) {
			log.Trace("removeTxsFromPools.isMiniBlockCorrect: false",
				"miniblock type", currentMiniBlock.Type)
			continue
		}

		//TODO: Remove this if when processing of scheduled mini blocks will be done in the source shard
		if currentMiniBlock.IsScheduledMiniBlock() {
			log.Warn("removeTxsFromPools: deletion of scheduled transactions should be skipped for now")
			continue
		}

		strCache := process.ShardCacherIdentifier(currentMiniBlock.SenderShardID, currentMiniBlock.ReceiverShardID)
		txPool.RemoveSetOfDataFromPool(currentMiniBlock.TxHashes, strCache)
	}

	return nil
}

func (bpp *basePreProcess) removeMiniBlocksFromPools(
	body *block.Body,
	miniBlockPool storage.Cacher,
	isMiniBlockCorrect func(block.Type) bool,
) error {
	if check.IfNil(body) {
		return process.ErrNilTxBlockBody
	}
	if check.IfNil(miniBlockPool) {
		return process.ErrNilMiniBlockPool
	}

	for i := 0; i < len(body.MiniBlocks); i++ {
		currentMiniBlock := body.MiniBlocks[i]
		if !isMiniBlockCorrect(currentMiniBlock.Type) {
			log.Trace("removeMiniBlocksFromPools.isMiniBlockCorrect: false",
				"miniblock type", currentMiniBlock.Type)
			continue
		}

		miniBlockHash, err := core.CalculateHash(bpp.marshalizer, bpp.hasher, currentMiniBlock)
		if err != nil {
			return err
		}

		miniBlockPool.Remove(miniBlockHash)
	}

	return nil
}

func (bpp *basePreProcess) createMarshalizedData(txHashes [][]byte, forBlock *txsForBlock) ([][]byte, error) {
	mrsTxs := make([][]byte, 0, len(txHashes))
	for _, txHash := range txHashes {
		forBlock.mutTxsForBlock.RLock()
		txInfoFromMap := forBlock.txHashAndInfo[string(txHash)]
		forBlock.mutTxsForBlock.RUnlock()

		if txInfoFromMap == nil || check.IfNil(txInfoFromMap.tx) {
			log.Warn("basePreProcess.createMarshalizedData: tx not found", "hash", txHash)
			continue
		}

		txMrs, err := bpp.marshalizer.Marshal(txInfoFromMap.tx)
		if err != nil {
			return nil, process.ErrMarshalWithoutSuccess
		}
		mrsTxs = append(mrsTxs, txMrs)
	}

	log.Trace("basePreProcess.createMarshalizedData",
		"num txs", len(mrsTxs),
	)

	return mrsTxs, nil
}

func (bpp *basePreProcess) saveTxsToStorage(
	txHashes [][]byte,
	forBlock *txsForBlock,
	store dataRetriever.StorageService,
	dataUnit dataRetriever.UnitType,
) error {

	for i := 0; i < len(txHashes); i++ {
		txHash := txHashes[i]

		forBlock.mutTxsForBlock.RLock()
		txInfoFromMap := forBlock.txHashAndInfo[string(txHash)]
		forBlock.mutTxsForBlock.RUnlock()

		if txInfoFromMap == nil || txInfoFromMap.tx == nil {
			log.Debug("missing transaction in saveTxsToStorage ", "type", dataUnit, "txHash", txHash)
			return process.ErrMissingTransaction
		}

		buff, err := bpp.marshalizer.Marshal(txInfoFromMap.tx)
		if err != nil {
			return err
		}

		errNotCritical := store.Put(dataUnit, txHash, buff)
		if errNotCritical != nil {
			log.Debug("store.Put",
				"error", errNotCritical.Error(),
				"dataUnit", dataUnit,
			)
		}
	}

	return nil
}

func (bpp *basePreProcess) baseReceivedTransaction(
	txHash []byte,
	tx data.TransactionHandler,
	forBlock *txsForBlock,
) bool {

	forBlock.mutTxsForBlock.Lock()
	defer forBlock.mutTxsForBlock.Unlock()

	if forBlock.missingTxs > 0 {
		txInfoForHash := forBlock.txHashAndInfo[string(txHash)]
		if txInfoForHash != nil && txInfoForHash.txShardInfo != nil &&
			(txInfoForHash.tx == nil || txInfoForHash.tx.IsInterfaceNil()) {
			forBlock.txHashAndInfo[string(txHash)].tx = tx
			forBlock.missingTxs--
		}

		return forBlock.missingTxs == 0
	}

	return false
}

func (bpp *basePreProcess) computeExistingAndRequestMissing(
	body *block.Body,
	forBlock *txsForBlock,
	_ chan bool,
	isMiniBlockCorrect func(block.Type) bool,
	txPool dataRetriever.ShardedDataCacherNotifier,
	onRequestTxs func(shardID uint32, txHashes [][]byte),
) int {

	if check.IfNil(body) {
		return 0
	}

	forBlock.mutTxsForBlock.Lock()
	defer forBlock.mutTxsForBlock.Unlock()

	missingTxsForShard := make(map[uint32][][]byte, bpp.shardCoordinator.NumberOfShards())
	txHashes := make([][]byte, 0)
	for i := 0; i < len(body.MiniBlocks); i++ {
		miniBlock := body.MiniBlocks[i]
		if !isMiniBlockCorrect(miniBlock.Type) {
			continue
		}

		txShardInfoObject := &txShardInfo{senderShardID: miniBlock.SenderShardID, receiverShardID: miniBlock.ReceiverShardID}
		searchFirst := miniBlock.Type == block.InvalidBlock
		for j := 0; j < len(miniBlock.TxHashes); j++ {
			txHash := miniBlock.TxHashes[j]
			tx, err := process.GetTransactionHandlerFromPool(
				miniBlock.SenderShardID,
				miniBlock.ReceiverShardID,
				txHash,
				txPool,
				searchFirst)

			if err != nil {
				txHashes = append(txHashes, txHash)
				forBlock.missingTxs++
				log.Trace("missing tx",
					"miniblock type", miniBlock.Type,
					"sender", miniBlock.SenderShardID,
					"receiver", miniBlock.ReceiverShardID,
					"hash", txHash,
				)
				continue
			}

			forBlock.txHashAndInfo[string(txHash)] = &txInfo{tx: tx, txShardInfo: txShardInfoObject}
		}

		if len(txHashes) > 0 {
			bpp.setMissingTxsForShard(miniBlock.SenderShardID, miniBlock.ReceiverShardID, txHashes, forBlock)
			missingTxsForShard[miniBlock.SenderShardID] = append(missingTxsForShard[miniBlock.SenderShardID], txHashes...)
		}

		txHashes = txHashes[:0]
	}

	return bpp.requestMissingTxsForShard(missingTxsForShard, onRequestTxs)
}

// this method should be called only under the mutex protection: forBlock.mutTxsForBlock
func (bpp *basePreProcess) setMissingTxsForShard(
	senderShardID uint32,
	receiverShardID uint32,
	txHashes [][]byte,
	forBlock *txsForBlock,
) {
	txShardInfoToSet := &txShardInfo{
		senderShardID:   senderShardID,
		receiverShardID: receiverShardID,
	}

	for _, txHash := range txHashes {
		forBlock.txHashAndInfo[string(txHash)] = &txInfo{
			tx:          nil,
			txShardInfo: txShardInfoToSet,
		}
	}
}

// this method should be called only under the mutex protection: forBlock.mutTxsForBlock
func (bpp *basePreProcess) requestMissingTxsForShard(
	missingTxsForShard map[uint32][][]byte,
	onRequestTxs func(shardID uint32, txHashes [][]byte),
) int {
	requestedTxs := 0
	for shardID, txHashes := range missingTxsForShard {
		requestedTxs += len(txHashes)
		go onRequestTxs(shardID, txHashes)
	}

	return requestedTxs
}

func (bpp *basePreProcess) saveAccountBalanceForAddress(address []byte) {
	if bpp.balanceComputation.IsAddressSet(address) {
		return
	}

	balance, err := bpp.getBalanceForAddress(address)
	if err != nil {
		balance = big.NewInt(0)
	}

	bpp.balanceComputation.SetBalanceToAddress(address, balance)
}

func (bpp *basePreProcess) getBalanceForAddress(address []byte) (*big.Int, error) {
	accountHandler, err := bpp.accounts.GetExistingAccount(address)
	if err != nil {
		return nil, err
	}

	account, ok := accountHandler.(state.UserAccountHandler)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	return account.GetBalance(), nil
}

func (bpp *basePreProcess) getTxMaxTotalCost(txHandler data.TransactionHandler) *big.Int {
	cost := big.NewInt(0)
	cost.Mul(big.NewInt(0).SetUint64(txHandler.GetGasPrice()), big.NewInt(0).SetUint64(txHandler.GetGasLimit()))

	if txHandler.GetValue() != nil {
		cost.Add(cost, txHandler.GetValue())
	}

	return cost
}
