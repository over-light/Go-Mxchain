package preprocess

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/ElrondNetwork/elrond-go-core/data/scheduled"
	"github.com/ElrondNetwork/elrond-go-core/data/transaction"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
)

type intermediateTxInfo struct {
	txHash    []byte
	txHandler data.TransactionHandler
}

type scheduledTxsExecution struct {
	txProcessor                 process.TransactionProcessor
	txCoordinator               process.TransactionCoordinator
	mapScheduledTxs             map[string]data.TransactionHandler
	mapScheduledIntermediateTxs map[block.Type][]data.TransactionHandler
	scheduledTxs                []data.TransactionHandler
	scheduledMBs                block.MiniBlockSlice
	scheduledRootHash           []byte
	gasAndFees                  scheduled.GasAndFees
	storer                      storage.Storer
	marshaller                  marshal.Marshalizer
	mutScheduledTxs             sync.RWMutex
	shardCoordinator            sharding.Coordinator
}

// NewScheduledTxsExecution creates a new object which handles the execution of scheduled transactions
func NewScheduledTxsExecution(
	txProcessor process.TransactionProcessor,
	txCoordinator process.TransactionCoordinator,
	storer storage.Storer,
	marshaller marshal.Marshalizer,
	shardCoordinator sharding.Coordinator,
) (*scheduledTxsExecution, error) {

	if check.IfNil(txProcessor) {
		return nil, process.ErrNilTxProcessor
	}
	if check.IfNil(txCoordinator) {
		return nil, process.ErrNilTransactionCoordinator
	}
	if check.IfNil(storer) {
		return nil, process.ErrNilStorage
	}
	if check.IfNil(marshaller) {
		return nil, process.ErrNilMarshalizer
	}
	if check.IfNil(shardCoordinator) {
		return nil, process.ErrNilShardCoordinator
	}

	ste := &scheduledTxsExecution{
		txProcessor:                 txProcessor,
		txCoordinator:               txCoordinator,
		mapScheduledTxs:             make(map[string]data.TransactionHandler),
		mapScheduledIntermediateTxs: make(map[block.Type][]data.TransactionHandler),
		scheduledTxs:                make([]data.TransactionHandler, 0),
		scheduledMBs:                make(block.MiniBlockSlice, 0),
		gasAndFees:                  process.GetZeroGasAndFees(),
		storer:                      storer,
		marshaller:                  marshaller,
		scheduledRootHash:           nil,
		shardCoordinator:            shardCoordinator,
	}

	return ste, nil
}

// Init method removes all the scheduled transactions
func (ste *scheduledTxsExecution) Init() {
	ste.mutScheduledTxs.Lock()
	log.Debug("scheduledTxsExecution.Init", "num of last scheduled txs", len(ste.scheduledTxs))
	ste.mapScheduledTxs = make(map[string]data.TransactionHandler)
	ste.scheduledTxs = make([]data.TransactionHandler, 0)
	ste.mutScheduledTxs.Unlock()
}

// AddScheduledTx method adds a scheduled transaction to be executed
func (ste *scheduledTxsExecution) AddScheduledTx(txHash []byte, tx data.TransactionHandler) bool {
	ste.mutScheduledTxs.Lock()
	defer ste.mutScheduledTxs.Unlock()

	_, exist := ste.mapScheduledTxs[string(txHash)]
	if exist {
		return false
	}

	ste.mapScheduledTxs[string(txHash)] = tx
	ste.scheduledTxs = append(ste.scheduledTxs, tx)

	log.Trace("scheduledTxsExecution.Add", "tx hash", txHash, "num of scheduled txs", len(ste.scheduledTxs))
	return true
}

// AddScheduledMiniBlocks method adds all the scheduled mini blocks to be executed
func (ste *scheduledTxsExecution) AddScheduledMiniBlocks(miniBlocks block.MiniBlockSlice) {
	ste.mutScheduledTxs.Lock()
	defer ste.mutScheduledTxs.Unlock()

	ste.scheduledMBs = make(block.MiniBlockSlice, len(miniBlocks))
	for index, miniBlock := range miniBlocks {
		ste.scheduledMBs[index] = miniBlock.Clone()
		//TODO: Remove both if when method clone will be fixed
		if miniBlock.Reserved == nil {
			ste.scheduledMBs[index].Reserved = nil
		}
		if miniBlock.TxHashes == nil {
			ste.scheduledMBs[index].TxHashes = nil
		}
	}

	log.Debug("scheduledTxsExecution.AddMiniBlocks", "num of scheduled mbs", len(ste.scheduledMBs))
}

// Execute method executes the given scheduled transaction
func (ste *scheduledTxsExecution) Execute(txHash []byte) error {
	ste.mutScheduledTxs.RLock()
	defer ste.mutScheduledTxs.RUnlock()

	txHandler, exist := ste.mapScheduledTxs[string(txHash)]
	if !exist {
		return fmt.Errorf("%w: in scheduledTxsExecution.Execute", process.ErrMissingTransaction)
	}

	err := ste.execute(txHandler)
	if err != nil && !errors.Is(err, process.ErrFailedTransaction) {
		return err
	}

	return nil
}

// ExecuteAll method executes all the scheduled transactions
func (ste *scheduledTxsExecution) ExecuteAll(haveTime func() time.Duration) error {
	ste.mutScheduledTxs.Lock()
	defer ste.mutScheduledTxs.Unlock()

	if haveTime == nil {
		return process.ErrNilHaveTimeHandler
	}

	log.Debug("scheduledTxsExecution.ExecuteAll", "num of scheduled txs to be executed", len(ste.scheduledTxs))

	mapAllIntermediateTxsBeforeScheduledExecution := ste.txCoordinator.GetAllIntermediateTxs()

	for _, txHandler := range ste.scheduledTxs {
		if haveTime() <= 0 {
			return process.ErrTimeIsOut
		}

		err := ste.execute(txHandler)
		if err != nil {
			log.Debug("scheduledTxsExecution.ExecuteAll: execute(txHandler)",
				"nonce", txHandler.GetNonce(),
				"value", txHandler.GetValue(),
				"gas limit", txHandler.GetGasLimit(),
				"gas price", txHandler.GetGasPrice(),
				"sender address", string(txHandler.GetSndAddr()),
				"receiver address", string(txHandler.GetRcvAddr()),
				"data", string(txHandler.GetData()),
				"error", err.Error())
			if !errors.Is(err, process.ErrFailedTransaction) {
				return err
			}
		}
	}

	mapAllIntermediateTxsAfterScheduledExecution := ste.txCoordinator.GetAllIntermediateTxs()
	ste.computeScheduledIntermediateTxs(mapAllIntermediateTxsBeforeScheduledExecution, mapAllIntermediateTxsAfterScheduledExecution)

	return nil
}

func (ste *scheduledTxsExecution) execute(txHandler data.TransactionHandler) error {
	tx, ok := txHandler.(*transaction.Transaction)
	if !ok {
		return fmt.Errorf("%w: in scheduledTxsExecution.execute", process.ErrWrongTypeAssertion)
	}

	_, err := ste.txProcessor.ProcessTransaction(tx)
	return err
}

func (ste *scheduledTxsExecution) computeScheduledIntermediateTxs(
	mapAllIntermediateTxsBeforeScheduledExecution map[block.Type]map[string]data.TransactionHandler,
	mapAllIntermediateTxsAfterScheduledExecution map[block.Type]map[string]data.TransactionHandler,
) {
	numScheduledIntermediateTxs := 0
	ste.mapScheduledIntermediateTxs = make(map[block.Type][]data.TransactionHandler)
	for blockType, allIntermediateTxsAfterScheduledExecution := range mapAllIntermediateTxsAfterScheduledExecution {
		intermediateTxsInfo := ste.getAllIntermediateTxsAfterScheduledExecution(
			mapAllIntermediateTxsBeforeScheduledExecution[blockType],
			allIntermediateTxsAfterScheduledExecution,
			blockType,
		)
		if len(intermediateTxsInfo) == 0 {
			continue
		}

		sort.Slice(intermediateTxsInfo, func(a, b int) bool {
			return bytes.Compare(intermediateTxsInfo[a].txHash, intermediateTxsInfo[b].txHash) < 0
		})

		if blockType == block.InvalidBlock {
			ste.removeInvalidTxsFromScheduledMiniBlocks(intermediateTxsInfo)
		}

		ste.mapScheduledIntermediateTxs[blockType] = make([]data.TransactionHandler, len(intermediateTxsInfo))
		for index, interTxInfo := range intermediateTxsInfo {
			ste.mapScheduledIntermediateTxs[blockType][index] = interTxInfo.txHandler
			log.Trace("scheduledTxsExecution.computeScheduledIntermediateTxs", "blockType", blockType, "sender", ste.mapScheduledIntermediateTxs[blockType][index].GetSndAddr(), "receiver", ste.mapScheduledIntermediateTxs[blockType][index].GetRcvAddr())
		}

		numScheduledIntermediateTxs += len(intermediateTxsInfo)
	}

	log.Debug("scheduledTxsExecution.computeScheduledIntermediateTxs", "num of scheduled intermediate txs created", numScheduledIntermediateTxs)
}

func (ste *scheduledTxsExecution) removeInvalidTxsFromScheduledMiniBlocks(intermediateTxsInfo []*intermediateTxInfo) {
	log.Debug("scheduledTxsExecution.removeInvalidTxsFromScheduledMiniBlocks", "num of invalid txs", len(intermediateTxsInfo))

	numInvalidTxsRemoved := 0
	for _, interTxInfo := range intermediateTxsInfo {
		for index, miniBlock := range ste.scheduledMBs {
			indexOfTxHashInMiniBlock := getIndexOfTxHashInMiniBlock(interTxInfo.txHash, miniBlock)
			if indexOfTxHashInMiniBlock >= 0 {
				log.Trace("scheduledTxsExecution.removeInvalidTxsFromScheduledMiniBlocks", "tx hash", interTxInfo.txHash)
				ste.scheduledMBs[index].TxHashes = append(miniBlock.TxHashes[:indexOfTxHashInMiniBlock], miniBlock.TxHashes[indexOfTxHashInMiniBlock+1:]...)
				numInvalidTxsRemoved++
				break
			}
		}
	}

	log.Debug("scheduledTxsExecution.removeInvalidTxsFromScheduledMiniBlocks", "num of invalid txs removed", numInvalidTxsRemoved)
}

func getIndexOfTxHashInMiniBlock(txHash []byte, miniBlock *block.MiniBlock) int {
	indexOfTxHashInMiniBlock := -1
	for index, hash := range miniBlock.TxHashes {
		if bytes.Equal(txHash, hash) {
			indexOfTxHashInMiniBlock = index
			break
		}
	}

	return indexOfTxHashInMiniBlock
}

func (ste *scheduledTxsExecution) getAllIntermediateTxsAfterScheduledExecution(
	allIntermediateTxsBeforeScheduledExecution map[string]data.TransactionHandler,
	allIntermediateTxsAfterScheduledExecution map[string]data.TransactionHandler,
	blockType block.Type,
) []*intermediateTxInfo {
	intermediateTxsInfo := make([]*intermediateTxInfo, 0)
	for txHash, txHandler := range allIntermediateTxsAfterScheduledExecution {
		_, txExists := allIntermediateTxsBeforeScheduledExecution[txHash]
		if txExists {
			continue
		}

		isInShardUnsignedTx := ste.shardCoordinator.SameShard(txHandler.GetSndAddr(), txHandler.GetRcvAddr()) &&
			(blockType == block.ReceiptBlock || blockType == block.SmartContractResultBlock)
		if isInShardUnsignedTx {
			log.Trace("scheduledTxsExecution.getAllIntermediateTxsAfterScheduledExecution: intra shard unsigned tx skipped", "hash", []byte(txHash))
			continue
		}

		intermediateTxsInfo = append(intermediateTxsInfo, &intermediateTxInfo{
			txHash:    []byte(txHash),
			txHandler: txHandler,
		})
	}

	return intermediateTxsInfo
}

// GetScheduledTxs gets all the scheduled txs to be executed
func (ste *scheduledTxsExecution) GetScheduledTxs() []data.TransactionHandler {
	ste.mutScheduledTxs.RLock()
	defer ste.mutScheduledTxs.RUnlock()

	scheduledTxs := make([]data.TransactionHandler, len(ste.scheduledTxs))
	for index, scheduledTx := range ste.scheduledTxs {
		scheduledTxs[index] = scheduledTx
		log.Trace("scheduledTxsExecution.GetScheduledTxs", "sender", scheduledTxs[index].GetSndAddr(), "receiver", scheduledTxs[index].GetRcvAddr())
	}

	log.Debug("scheduledTxsExecution.GetScheduledTxs", "num of scheduled txs", len(scheduledTxs))

	return scheduledTxs
}

// GetScheduledIntermediateTxs gets the resulted intermediate txs after the execution of scheduled transactions
func (ste *scheduledTxsExecution) GetScheduledIntermediateTxs() map[block.Type][]data.TransactionHandler {
	ste.mutScheduledTxs.RLock()
	defer ste.mutScheduledTxs.RUnlock()

	numScheduledIntermediateTxs := 0
	mapScheduledIntermediateTxs := make(map[block.Type][]data.TransactionHandler)
	for blockType, scheduledIntermediateTxs := range ste.mapScheduledIntermediateTxs {
		if len(scheduledIntermediateTxs) == 0 {
			continue
		}

		mapScheduledIntermediateTxs[blockType] = make([]data.TransactionHandler, len(scheduledIntermediateTxs))
		for index, scheduledIntermediateTx := range scheduledIntermediateTxs {
			mapScheduledIntermediateTxs[blockType][index] = scheduledIntermediateTx
			log.Trace("scheduledTxsExecution.GetScheduledIntermediateTxs", "blockType", blockType, "sender", mapScheduledIntermediateTxs[blockType][index].GetSndAddr(), "receiver", mapScheduledIntermediateTxs[blockType][index].GetRcvAddr())
		}
		numScheduledIntermediateTxs += len(scheduledIntermediateTxs)
	}

	log.Debug("scheduledTxsExecution.GetScheduledIntermediateTxs", "num of scheduled intermediate txs", numScheduledIntermediateTxs)

	return mapScheduledIntermediateTxs
}

// GetScheduledMBs gets the resulted mini blocks after the execution of scheduled transactions
func (ste *scheduledTxsExecution) GetScheduledMBs() block.MiniBlockSlice {
	ste.mutScheduledTxs.RLock()
	defer ste.mutScheduledTxs.RUnlock()

	if len(ste.scheduledMBs) == 0 {
		return nil
	}

	miniBlocks := make(block.MiniBlockSlice, len(ste.scheduledMBs))
	for index, scheduledMb := range ste.scheduledMBs {
		miniBlock := scheduledMb.Clone()
		//TODO: Remove both if when method clone will be fixed
		if scheduledMb.Reserved == nil {
			miniBlock.Reserved = nil
		}
		if scheduledMb.TxHashes == nil {
			miniBlock.TxHashes = nil
		}
		miniBlocks[index] = miniBlock
	}

	log.Debug("scheduledTxsExecution.GetScheduledMBs", "num of scheduled mbs", len(miniBlocks))

	return miniBlocks
}

// SetScheduledInfo sets the resulted scheduled mini blocks, root hash, intermediate txs, gas and fees after the execution of scheduled transactions
func (ste *scheduledTxsExecution) SetScheduledInfo(scheduledInfo *process.ScheduledInfo) {
	ste.mutScheduledTxs.Lock()
	defer ste.mutScheduledTxs.Unlock()

	ste.scheduledRootHash = scheduledInfo.RootHash

	numScheduledIntermediateTxs := 0
	ste.mapScheduledIntermediateTxs = make(map[block.Type][]data.TransactionHandler)
	for blockType, intermediateTxs := range scheduledInfo.IntermediateTxs {
		if len(intermediateTxs) == 0 {
			continue
		}

		ste.mapScheduledIntermediateTxs[blockType] = make([]data.TransactionHandler, len(intermediateTxs))
		for index, intermediateTx := range intermediateTxs {
			ste.mapScheduledIntermediateTxs[blockType][index] = intermediateTx
			log.Trace("scheduledTxsExecution.SetScheduledInfo", "blockType", blockType, "sender", ste.mapScheduledIntermediateTxs[blockType][index].GetSndAddr(), "receiver", ste.mapScheduledIntermediateTxs[blockType][index].GetRcvAddr())
		}

		numScheduledIntermediateTxs += len(intermediateTxs)
	}

	ste.gasAndFees = scheduledInfo.GasAndFees

	ste.scheduledMBs = make(block.MiniBlockSlice, len(scheduledInfo.MiniBlocks))
	for index, scheduledMiniBlock := range scheduledInfo.MiniBlocks {
		miniBlock := scheduledMiniBlock.Clone()
		//TODO: Remove both if when method clone will be fixed
		if scheduledMiniBlock.Reserved == nil {
			miniBlock.Reserved = nil
		}
		if scheduledMiniBlock.TxHashes == nil {
			miniBlock.TxHashes = nil
		}

		ste.scheduledMBs[index] = miniBlock
	}

	log.Debug("scheduledTxsExecution.SetScheduledInfo",
		"scheduled root hash", ste.scheduledRootHash,
		"num of scheduled mbs", len(ste.scheduledMBs),
		"num of scheduled intermediate txs", numScheduledIntermediateTxs,
		"accumulatedFees", ste.gasAndFees.AccumulatedFees.String(),
		"developerFees", ste.gasAndFees.DeveloperFees.String(),
		"gasProvided", ste.gasAndFees.GasProvided,
		"gasPenalized", ste.gasAndFees.GasPenalized,
		"gasRefunded", ste.gasAndFees.GasRefunded)
}

// GetScheduledRootHash gets the resulted root hash after the execution of scheduled transactions
func (ste *scheduledTxsExecution) GetScheduledRootHash() []byte {
	ste.mutScheduledTxs.RLock()
	rootHash := ste.scheduledRootHash
	ste.mutScheduledTxs.RUnlock()

	log.Debug("scheduledTxsExecution.GetScheduledRootHash", "scheduled root hash", rootHash)

	return rootHash
}

// GetScheduledGasAndFees returns the gas and fees for the scheduled transactions in last processed block
// if there are no scheduled transactions in the last processed block, the returned struct has zero values
func (ste *scheduledTxsExecution) GetScheduledGasAndFees() scheduled.GasAndFees {
	ste.mutScheduledTxs.RLock()
	gasAndFees := ste.gasAndFees
	ste.mutScheduledTxs.RUnlock()

	log.Debug("scheduledTxsExecution.GetScheduledGasAndFees",
		"accumulatedFees", gasAndFees.AccumulatedFees.String(),
		"developerFees", gasAndFees.DeveloperFees.String(),
		"gasProvided", gasAndFees.GasProvided,
		"gasPenalized", gasAndFees.GasPenalized,
		"gasRefunded", gasAndFees.GasRefunded)

	return gasAndFees
}

// SetScheduledRootHash sets the resulted root hash after the execution of scheduled transactions
func (ste *scheduledTxsExecution) SetScheduledRootHash(rootHash []byte) {
	ste.mutScheduledTxs.Lock()
	defer ste.mutScheduledTxs.Unlock()

	ste.scheduledRootHash = rootHash
	log.Debug("scheduledTxsExecution.SetScheduledRootHash", "scheduled root hash", ste.scheduledRootHash)
}

// SetScheduledGasAndFees sets the gas and fees for the scheduled transactions
func (ste *scheduledTxsExecution) SetScheduledGasAndFees(gasAndFees scheduled.GasAndFees) {
	ste.mutScheduledTxs.Lock()
	defer ste.mutScheduledTxs.Unlock()

	ste.gasAndFees = gasAndFees
	log.Debug("scheduledTxsExecution.SetScheduledGasAndFees",
		"accumulatedFees", ste.gasAndFees.AccumulatedFees.String(),
		"developerFees", ste.gasAndFees.DeveloperFees.String(),
		"gasProvided", ste.gasAndFees.GasProvided,
		"gasPenalized", ste.gasAndFees.GasPenalized,
		"gasRefunded", ste.gasAndFees.GasRefunded)
}

// SetTransactionProcessor sets the transaction processor needed by scheduled txs execution component
func (ste *scheduledTxsExecution) SetTransactionProcessor(txProcessor process.TransactionProcessor) {
	ste.txProcessor = txProcessor
}

// SetTransactionCoordinator sets the transaction coordinator needed by scheduled txs execution component
func (ste *scheduledTxsExecution) SetTransactionCoordinator(txCoordinator process.TransactionCoordinator) {
	ste.txCoordinator = txCoordinator
}

// GetScheduledRootHashForHeader gets scheduled root hash of the given header from storage
func (ste *scheduledTxsExecution) GetScheduledRootHashForHeader(
	headerHash []byte,
) ([]byte, error) {
	scheduledInfo, err := ste.getScheduledInfoForHeader(headerHash)
	if err != nil {
		return nil, err
	}

	log.Debug("scheduledTxsExecution.GetScheduledRootHashForHeader", "header hash", headerHash, "scheduled root hash", scheduledInfo.RootHash)

	return scheduledInfo.RootHash, nil
}

// RollBackToBlock rolls back the scheduled txs execution handler to the given header
func (ste *scheduledTxsExecution) RollBackToBlock(headerHash []byte) error {
	scheduledInfo, err := ste.getScheduledInfoForHeader(headerHash)
	if err != nil {
		return err
	}

	log.Debug("scheduledTxsExecution.RollBackToBlock",
		"header hash", headerHash,
		"scheduled root hash", scheduledInfo.RootHash,
		"num of scheduled mbs", len(scheduledInfo.MiniBlocks),
		"num of scheduled intermediate txs", getNumScheduledIntermediateTxs(scheduledInfo.IntermediateTxs),
		"accumulatedFees", scheduledInfo.GasAndFees.AccumulatedFees.String(),
		"developerFees", scheduledInfo.GasAndFees.DeveloperFees.String(),
		"gasProvided", scheduledInfo.GasAndFees.GasProvided,
		"gasPenalized", scheduledInfo.GasAndFees.GasPenalized,
		"gasRefunded", scheduledInfo.GasAndFees.GasRefunded)

	ste.SetScheduledInfo(scheduledInfo)

	return nil
}

// SaveStateIfNeeded saves the scheduled SC execution state for the given header hash, if there are scheduled txs
func (ste *scheduledTxsExecution) SaveStateIfNeeded(headerHash []byte) {
	scheduledInfo := &process.ScheduledInfo{
		RootHash:        ste.GetScheduledRootHash(),
		IntermediateTxs: ste.GetScheduledIntermediateTxs(),
		GasAndFees:      ste.GetScheduledGasAndFees(),
		MiniBlocks:      ste.GetScheduledMBs(),
	}

	ste.mutScheduledTxs.RLock()
	numScheduledTxs := len(ste.scheduledTxs)
	ste.mutScheduledTxs.RUnlock()

	log.Debug("scheduledTxsExecution.SaveStateIfNeeded",
		"header hash", headerHash,
		"scheduled root hash", scheduledInfo.RootHash,
		"num of scheduled txs", numScheduledTxs,
		"num of scheduled intermediate txs", getNumScheduledIntermediateTxs(scheduledInfo.IntermediateTxs),
		"accumulatedFees", scheduledInfo.GasAndFees.AccumulatedFees.String(),
		"developerFees", scheduledInfo.GasAndFees.DeveloperFees.String(),
		"gasProvided", scheduledInfo.GasAndFees.GasProvided,
		"gasPenalized", scheduledInfo.GasAndFees.GasPenalized,
		"gasRefunded", scheduledInfo.GasAndFees.GasRefunded)

	if numScheduledTxs > 0 {
		ste.SaveState(headerHash, scheduledInfo)
	}
}

// SaveState saves the scheduled SC execution state
func (ste *scheduledTxsExecution) SaveState(headerHash []byte, scheduledInfo *process.ScheduledInfo) {
	marshalledScheduledInfo, err := ste.getMarshalledScheduledInfo(scheduledInfo)
	if err != nil {
		log.Warn("scheduledTxsExecution.SaveState: getMarshalledScheduledInfo", "error", err.Error())
		return
	}

	log.Debug("scheduledTxsExecution.SaveState: Put",
		"header hash", headerHash,
		"scheduled root hash", scheduledInfo.RootHash,
		"num of scheduled intermediate txs", getNumScheduledIntermediateTxs(scheduledInfo.IntermediateTxs),
		"gasAndFees.AccumulatedFees", scheduledInfo.GasAndFees.AccumulatedFees.String(),
		"gasAndFees.DeveloperFees", scheduledInfo.GasAndFees.DeveloperFees.String(),
		"gasAndFees.GasProvided", scheduledInfo.GasAndFees.GasProvided,
		"gasAndFees.GasPenalized", scheduledInfo.GasAndFees.GasPenalized,
		"gasAndFees.GasRefunded", scheduledInfo.GasAndFees.GasRefunded,
		"length of marshalized scheduled info", len(marshalledScheduledInfo))
	err = ste.storer.Put(headerHash, marshalledScheduledInfo)
	if err != nil {
		log.Warn("scheduledTxsExecution.SaveState Put -> ScheduledIntermediateTxsUnit", "error", err.Error())
	}
}

// getScheduledInfoForHeader gets scheduled mini blocks, root hash, intermediate txs, gas and fees of the given header from storage
func (ste *scheduledTxsExecution) getScheduledInfoForHeader(headerHash []byte) (*process.ScheduledInfo, error) {
	var err error
	defer func() {
		if err != nil {
			log.Trace("getScheduledInfoForHeader: given header does not have scheduled txs",
				"header hash", headerHash,
			)
		}
	}()

	marshalledSCRsSavedData, err := ste.storer.Get(headerHash)
	if err != nil {
		return nil, err
	}

	scheduledSCRs := &scheduled.ScheduledSCRs{}
	err = ste.marshaller.Unmarshal(scheduledSCRs, marshalledSCRsSavedData)
	if err != nil {
		return nil, err
	}

	scheduledInfo := &process.ScheduledInfo{
		RootHash:        scheduledSCRs.RootHash,
		IntermediateTxs: scheduledSCRs.GetTransactionHandlersMap(),
		GasAndFees:      *scheduledSCRs.GasAndFees,
		//TODO: Replace the next line with the real scheduledMiniBlock taken from scheduledSCRs DTO
		MiniBlocks: make(block.MiniBlockSlice, 0),
	}

	return scheduledInfo, nil
}

func (ste *scheduledTxsExecution) getMarshalledScheduledInfo(
	scheduledInfo *process.ScheduledInfo,
) ([]byte, error) {
	scheduledSCRs := &scheduled.ScheduledSCRs{
		RootHash:   scheduledInfo.RootHash,
		GasAndFees: &scheduledInfo.GasAndFees,
	}

	//TODO: Set separately SCRs and invalid txs depending of the block type
	err := scheduledSCRs.SetTransactionHandlersMap(scheduledInfo.IntermediateTxs)
	if err != nil {
		return nil, err
	}

	return ste.marshaller.Marshal(scheduledSCRs)
}

// IsScheduledTx returns true if the given txHash was scheduled for execution for the current block
func (ste *scheduledTxsExecution) IsScheduledTx(txHash []byte) bool {
	ste.mutScheduledTxs.RLock()
	_, ok := ste.mapScheduledTxs[string(txHash)]
	ste.mutScheduledTxs.RUnlock()

	return ok
}

// SetScheduledMiniBlocksAsExecuted sets all the scheduled mini blocks as executed
func (ste *scheduledTxsExecution) SetScheduledMiniBlocksAsExecuted() {
	ste.mutScheduledTxs.Lock()
	defer ste.mutScheduledTxs.Unlock()

	for index := range ste.scheduledMBs {
		//TODO: Here should be set each mini block as executed with the real value when elrond-go-core PR will be done
		ste.scheduledMBs[index].Reserved = nil
	}
}

func getNumScheduledIntermediateTxs(mapScheduledIntermediateTxs map[block.Type][]data.TransactionHandler) int {
	numScheduledIntermediateTxs := 0
	for _, scheduledIntermediateTxs := range mapScheduledIntermediateTxs {
		numScheduledIntermediateTxs += len(scheduledIntermediateTxs)
	}

	return numScheduledIntermediateTxs
}

// IsInterfaceNil returns true if there is no value under the interface
func (ste *scheduledTxsExecution) IsInterfaceNil() bool {
	return ste == nil
}
