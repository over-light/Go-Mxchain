package coordinator

import (
	"sort"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/logger"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
)

var log = logger.GetOrCreate("process/coordinator")

type transactionCoordinator struct {
	shardCoordinator sharding.Coordinator
	accounts         state.AccountsAdapter
	miniBlockPool    storage.Cacher

	mutPreProcessor sync.RWMutex
	txPreProcessors map[block.Type]process.PreProcessor
	keysTxPreProcs  []block.Type

	mutInterimProcessors sync.RWMutex
	interimProcessors    map[block.Type]process.IntermediateTransactionHandler
	keysInterimProcs     []block.Type

	mutRequestedTxs sync.RWMutex
	requestedTxs    map[block.Type]int

	onRequestMiniBlock    func(shardId uint32, mbHash []byte)
	requestedItemsHandler process.RequestedItemsHandler
}

// NewTransactionCoordinator creates a transaction coordinator to run and coordinate preprocessors and processors
func NewTransactionCoordinator(
	shardCoordinator sharding.Coordinator,
	accounts state.AccountsAdapter,
	miniBlockPool storage.Cacher,
	requestHandler process.RequestHandler,
	preProcessors process.PreProcessorsContainer,
	interProcessors process.IntermediateProcessorContainer,
	requestedItemsHandler process.RequestedItemsHandler,
) (*transactionCoordinator, error) {

	if check.IfNil(shardCoordinator) {
		return nil, process.ErrNilShardCoordinator
	}
	if check.IfNil(accounts) {
		return nil, process.ErrNilAccountsAdapter
	}
	if check.IfNil(miniBlockPool) {
		return nil, process.ErrNilMiniBlockPool
	}
	if check.IfNil(requestHandler) {
		return nil, process.ErrNilRequestHandler
	}
	if check.IfNil(interProcessors) {
		return nil, process.ErrNilIntermediateProcessorContainer
	}
	if check.IfNil(preProcessors) {
		return nil, process.ErrNilPreProcessorsContainer
	}
	if check.IfNil(requestedItemsHandler) {
		return nil, process.ErrNilRequestedItemsHandler
	}

	tc := &transactionCoordinator{
		shardCoordinator: shardCoordinator,
		accounts:         accounts,
	}

	tc.miniBlockPool = miniBlockPool
	tc.miniBlockPool.RegisterHandler(tc.receivedMiniBlock)

	tc.onRequestMiniBlock = requestHandler.RequestMiniBlock
	tc.requestedItemsHandler = requestedItemsHandler
	tc.requestedTxs = make(map[block.Type]int)
	tc.txPreProcessors = make(map[block.Type]process.PreProcessor)
	tc.interimProcessors = make(map[block.Type]process.IntermediateTransactionHandler)

	tc.keysTxPreProcs = preProcessors.Keys()
	sort.Slice(tc.keysTxPreProcs, func(i, j int) bool {
		return tc.keysTxPreProcs[i] < tc.keysTxPreProcs[j]
	})
	for _, value := range tc.keysTxPreProcs {
		preProc, err := preProcessors.Get(value)
		if err != nil {
			return nil, err
		}
		tc.txPreProcessors[value] = preProc
	}

	tc.keysInterimProcs = interProcessors.Keys()
	sort.Slice(tc.keysInterimProcs, func(i, j int) bool {
		return tc.keysInterimProcs[i] < tc.keysInterimProcs[j]
	})
	for _, value := range tc.keysInterimProcs {
		interProc, err := interProcessors.Get(value)
		if err != nil {
			return nil, err
		}
		tc.interimProcessors[value] = interProc
	}

	return tc, nil
}

// separateBodyByType creates a map of bodies according to type
func (tc *transactionCoordinator) separateBodyByType(body block.Body) map[block.Type]block.Body {
	separatedBodies := make(map[block.Type]block.Body)

	for i := 0; i < len(body); i++ {
		mb := body[i]

		if separatedBodies[mb.Type] == nil {
			separatedBodies[mb.Type] = block.Body{}
		}

		separatedBodies[mb.Type] = append(separatedBodies[mb.Type], mb)
	}

	return separatedBodies
}

// initRequestedTxs init the requested txs number
func (tc *transactionCoordinator) initRequestedTxs() {
	tc.mutRequestedTxs.Lock()
	tc.requestedTxs = make(map[block.Type]int)
	tc.mutRequestedTxs.Unlock()
}

// RequestBlockTransactions verifies missing transaction and requests them
func (tc *transactionCoordinator) RequestBlockTransactions(body block.Body) {
	separatedBodies := tc.separateBodyByType(body)

	tc.initRequestedTxs()

	wg := sync.WaitGroup{}
	wg.Add(len(separatedBodies))

	for key, value := range separatedBodies {
		go func(blockType block.Type, blockBody block.Body) {
			preproc := tc.getPreProcessor(blockType)
			if preproc == nil {
				wg.Done()
				return
			}
			requestedTxs := preproc.RequestBlockTransactions(blockBody)

			tc.mutRequestedTxs.Lock()
			tc.requestedTxs[blockType] = requestedTxs
			tc.mutRequestedTxs.Unlock()

			wg.Done()
		}(key, value)
	}

	wg.Wait()
}

// IsDataPreparedForProcessing verifies if all the needed data is prepared
func (tc *transactionCoordinator) IsDataPreparedForProcessing(haveTime func() time.Duration) error {
	var errFound error
	errMutex := sync.Mutex{}

	wg := sync.WaitGroup{}

	tc.mutRequestedTxs.RLock()
	wg.Add(len(tc.requestedTxs))

	for key, value := range tc.requestedTxs {
		go func(blockType block.Type, requestedTxs int) {
			preproc := tc.getPreProcessor(blockType)
			if preproc == nil {
				wg.Done()

				return
			}

			err := preproc.IsDataPrepared(requestedTxs, haveTime)
			if err != nil {
				log.Trace("IsDataPrepared", "error", err.Error())

				errMutex.Lock()
				errFound = err
				errMutex.Unlock()
			}
			wg.Done()
		}(key, value)
	}

	tc.mutRequestedTxs.RUnlock()
	wg.Wait()

	return errFound
}

// SaveBlockDataToStorage saves the data from block body into storage units
func (tc *transactionCoordinator) SaveBlockDataToStorage(body block.Body) error {
	separatedBodies := tc.separateBodyByType(body)

	var errFound error
	errMutex := sync.Mutex{}

	wg := sync.WaitGroup{}
	wg.Add(len(separatedBodies) + len(tc.keysInterimProcs))

	for key, value := range separatedBodies {
		go func(blockType block.Type, blockBody block.Body) {
			preproc := tc.getPreProcessor(blockType)
			if preproc == nil {
				wg.Done()
				return
			}

			err := preproc.SaveTxBlockToStorage(blockBody)
			if err != nil {
				log.Trace("SaveTxBlockToStorage", "error", err.Error())

				errMutex.Lock()
				errFound = err
				errMutex.Unlock()
			}

			wg.Done()
		}(key, value)
	}

	for _, blockType := range tc.keysInterimProcs {
		go func(blockType block.Type) {
			intermediateProc := tc.getInterimProcessor(blockType)
			if intermediateProc == nil {
				wg.Done()
				return
			}

			err := intermediateProc.SaveCurrentIntermediateTxToStorage()
			if err != nil {
				log.Trace("SaveCurrentIntermediateTxToStorage", "error", err.Error())

				errMutex.Lock()
				errFound = err
				errMutex.Unlock()
			}

			wg.Done()
		}(blockType)
	}

	wg.Wait()

	return errFound
}

// RestoreBlockDataFromStorage restores block data from storage to pool
func (tc *transactionCoordinator) RestoreBlockDataFromStorage(body block.Body) (int, error) {
	separatedBodies := tc.separateBodyByType(body)

	var errFound error
	localMutex := sync.Mutex{}
	totalRestoredTx := 0

	wg := sync.WaitGroup{}
	wg.Add(len(separatedBodies))

	for key, value := range separatedBodies {
		go func(blockType block.Type, blockBody block.Body) {
			preproc := tc.getPreProcessor(blockType)
			if preproc == nil {
				wg.Done()
				return
			}

			restoredTxs, err := preproc.RestoreTxBlockIntoPools(blockBody, tc.miniBlockPool)
			if err != nil {
				log.Trace("RestoreTxBlockIntoPools", "error", err.Error())

				localMutex.Lock()
				errFound = err
				localMutex.Unlock()
			}

			localMutex.Lock()
			totalRestoredTx += restoredTxs

			localMutex.Unlock()

			wg.Done()
		}(key, value)
	}

	wg.Wait()

	return totalRestoredTx, errFound
}

// RemoveBlockDataFromPool deletes block data from pools
func (tc *transactionCoordinator) RemoveBlockDataFromPool(body block.Body) error {
	separatedBodies := tc.separateBodyByType(body)

	var errFound error
	errMutex := sync.Mutex{}

	wg := sync.WaitGroup{}
	wg.Add(len(separatedBodies))

	for key, value := range separatedBodies {
		go func(blockType block.Type, blockBody block.Body) {
			preproc := tc.getPreProcessor(blockType)
			if preproc == nil || preproc.IsInterfaceNil() {
				wg.Done()
				return
			}

			err := preproc.RemoveTxBlockFromPools(blockBody, tc.miniBlockPool)
			if err != nil {
				log.Trace("RemoveTxBlockFromPools", "error", err.Error())

				errMutex.Lock()
				errFound = err
				errMutex.Unlock()
			}
			wg.Done()
		}(key, value)
	}

	wg.Wait()

	return errFound
}

// ProcessBlockTransaction processes transactions and updates state tries
func (tc *transactionCoordinator) ProcessBlockTransaction(
	body block.Body,
	round uint64,
	timeRemaining func() time.Duration,
) error {

	haveTime := func() bool {
		return timeRemaining() >= 0
	}

	separatedBodies := tc.separateBodyByType(body)
	// processing has to be done in order, as the order of different type of transactions over the same account is strict
	for _, blockType := range tc.keysTxPreProcs {
		if separatedBodies[blockType] == nil {
			continue
		}

		preproc := tc.getPreProcessor(blockType)
		if preproc == nil || preproc.IsInterfaceNil() {
			return process.ErrMissingPreProcessor
		}

		err := preproc.ProcessBlockTransactions(separatedBodies[blockType], round, haveTime)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateMbsAndProcessCrossShardTransactionsDstMe creates miniblocks and processes cross shard transaction
// with destination of current shard
func (tc *transactionCoordinator) CreateMbsAndProcessCrossShardTransactionsDstMe(
	hdr data.HeaderHandler,
	processedMiniBlocksHashes map[string]struct{},
	maxTxRemaining uint32,
	maxMbRemaining uint32,
	round uint64,
	haveTime func() bool,
) (block.MiniBlockSlice, uint32, bool) {
	miniBlocks := make(block.MiniBlockSlice, 0)
	nrTxAdded := uint32(0)
	nrMiniBlocksProcessed := 0

	if hdr == nil || hdr.IsInterfaceNil() {
		return miniBlocks, nrTxAdded, true
	}

	crossMiniBlockHashes := hdr.GetMiniBlockHeadersWithDst(tc.shardCoordinator.SelfId())
	for key, senderShardId := range crossMiniBlockHashes {
		if !haveTime() {
			break
		}

		_, ok := processedMiniBlocksHashes[key]
		if ok {
			nrMiniBlocksProcessed++
			continue
		}

		miniVal, _ := tc.miniBlockPool.Peek([]byte(key))
		if miniVal == nil {
			if !tc.requestedItemsHandler.Has(key) {
				go tc.onRequestMiniBlock(senderShardId, []byte(key))
				errNotCritical := tc.requestedItemsHandler.Add(key)
				if errNotCritical != nil {
					log.Trace("add requested item with error", errNotCritical.Error())
				}
			}

			continue
		}

		miniBlock, ok := miniVal.(*block.MiniBlock)
		if !ok {
			continue
		}

		preproc := tc.getPreProcessor(miniBlock.Type)
		if preproc == nil || preproc.IsInterfaceNil() {
			continue
		}

		// overflow would happen if processing would continue
		txOverFlow := nrTxAdded+uint32(len(miniBlock.TxHashes)) > maxTxRemaining
		if txOverFlow {
			return miniBlocks, nrTxAdded, false
		}

		requestedTxs := preproc.RequestTransactionsForMiniBlock(miniBlock)
		if requestedTxs > 0 {
			continue
		}

		err := tc.processCompleteMiniBlock(preproc, miniBlock, round, haveTime)
		if err != nil {
			continue
		}

		// all txs processed, add to processed miniblocks
		miniBlocks = append(miniBlocks, miniBlock)
		nrTxAdded = nrTxAdded + uint32(len(miniBlock.TxHashes))
		nrMiniBlocksProcessed++

		mbOverFlow := uint32(len(miniBlocks)) >= maxMbRemaining
		if mbOverFlow {
			return miniBlocks, nrTxAdded, false
		}
	}

	allMBsProcessed := nrMiniBlocksProcessed == len(crossMiniBlockHashes)
	return miniBlocks, nrTxAdded, allMBsProcessed
}

// CreateMbsAndProcessTransactionsFromMe creates miniblocks and processes transactions from pool
func (tc *transactionCoordinator) CreateMbsAndProcessTransactionsFromMe(
	maxTxSpaceRemained uint32,
	maxMbSpaceRemained uint32,
	round uint64,
	haveTime func() bool,
) block.MiniBlockSlice {

	miniBlocks := make(block.MiniBlockSlice, 0)
	for _, blockType := range tc.keysTxPreProcs {
		txPreProc := tc.getPreProcessor(blockType)
		if txPreProc == nil || txPreProc.IsInterfaceNil() {
			return nil
		}

		mbs, err := txPreProc.CreateAndProcessMiniBlocks(
			maxTxSpaceRemained,
			maxMbSpaceRemained,
			round,
			haveTime,
		)
		if err != nil {
			log.Debug("CreateAndProcessMiniBlocks", "error", err.Error())
		}

		if len(mbs) > 0 {
			miniBlocks = append(miniBlocks, mbs...)
		}
	}

	interMBs := tc.processAddedInterimTransactions()
	if len(interMBs) > 0 {
		miniBlocks = append(miniBlocks, interMBs...)
	}

	return miniBlocks
}

func (tc *transactionCoordinator) processAddedInterimTransactions() block.MiniBlockSlice {
	miniBlocks := make(block.MiniBlockSlice, 0)

	// processing has to be done in order, as the order of different type of transactions over the same account is strict
	for _, blockType := range tc.keysInterimProcs {
		if blockType == block.RewardsBlock {
			// this has to be processed last
			continue
		}

		interimProc := tc.getInterimProcessor(blockType)
		if interimProc == nil {
			// this will never be reached as keysInterimProcs are the actual keys from the interimMap
			continue
		}

		currMbs := interimProc.CreateAllInterMiniBlocks()
		for _, value := range currMbs {
			miniBlocks = append(miniBlocks, value)
		}
	}

	return miniBlocks
}

// CreateBlockStarted initializes necessary data for preprocessors at block create or block process
func (tc *transactionCoordinator) CreateBlockStarted() {
	tc.mutPreProcessor.RLock()
	for _, value := range tc.txPreProcessors {
		value.CreateBlockStarted()
	}
	tc.mutPreProcessor.RUnlock()

	tc.mutInterimProcessors.RLock()
	for _, value := range tc.interimProcessors {
		value.CreateBlockStarted()
	}
	tc.mutInterimProcessors.RUnlock()
}

func (tc *transactionCoordinator) getPreProcessor(blockType block.Type) process.PreProcessor {
	tc.mutPreProcessor.RLock()
	preprocessor, exists := tc.txPreProcessors[blockType]
	tc.mutPreProcessor.RUnlock()

	if !exists {
		return nil
	}

	return preprocessor
}

func (tc *transactionCoordinator) getInterimProcessor(blockType block.Type) process.IntermediateTransactionHandler {
	tc.mutInterimProcessors.RLock()
	interProcessor, exists := tc.interimProcessors[blockType]
	tc.mutInterimProcessors.RUnlock()

	if !exists {
		return nil
	}

	return interProcessor
}

func createBroadcastTopic(shardC sharding.Coordinator, destShId uint32, mbType block.Type) (string, error) {
	var baseTopic string

	switch mbType {
	case block.TxBlock:
		baseTopic = factory.TransactionTopic
	case block.PeerBlock:
		baseTopic = factory.PeerChBodyTopic
	case block.SmartContractResultBlock:
		baseTopic = factory.UnsignedTransactionTopic
	case block.RewardsBlock:
		baseTopic = factory.RewardsTransactionTopic
	default:
		return "", process.ErrUnknownBlockType
	}

	transactionTopic := baseTopic +
		shardC.CommunicationIdentifier(destShId)

	return transactionTopic, nil
}

// CreateMarshalizedData creates marshalized data for broadcasting
func (tc *transactionCoordinator) CreateMarshalizedData(body block.Body) (map[uint32]block.MiniBlockSlice, map[string][][]byte) {
	mrsTxs := make(map[string][][]byte)
	bodies := make(map[uint32]block.MiniBlockSlice)

	for i := 0; i < len(body); i++ {
		miniblock := body[i]
		receiverShardId := miniblock.ReceiverShardID
		if receiverShardId == tc.shardCoordinator.SelfId() { // not taking into account miniblocks for current shard
			continue
		}

		broadcastTopic, err := createBroadcastTopic(tc.shardCoordinator, receiverShardId, miniblock.Type)
		if err != nil {
			log.Trace("createBroadcastTopic", "error", err.Error())
			continue
		}

		appended := false
		preproc := tc.getPreProcessor(miniblock.Type)
		if preproc != nil && !preproc.IsInterfaceNil() {
			bodies[receiverShardId] = append(bodies[receiverShardId], miniblock)
			appended = true

			currMrsTxs, err := preproc.CreateMarshalizedData(miniblock.TxHashes)
			if err != nil {
				log.Trace("CreateMarshalizedData", "error", err.Error())
				continue
			}

			if len(currMrsTxs) > 0 {
				mrsTxs[broadcastTopic] = append(mrsTxs[broadcastTopic], currMrsTxs...)
			}
		}

		interimProc := tc.getInterimProcessor(miniblock.Type)
		if interimProc != nil && !interimProc.IsInterfaceNil() {
			if !appended {
				bodies[receiverShardId] = append(bodies[receiverShardId], miniblock)
			}

			currMrsInterTxs, err := interimProc.CreateMarshalizedData(miniblock.TxHashes)
			if err != nil {
				log.Trace("CreateMarshalizedData", "error", err.Error())
				continue
			}

			if len(currMrsInterTxs) > 0 {
				mrsTxs[broadcastTopic] = append(mrsTxs[broadcastTopic], currMrsInterTxs...)
			}
		}
	}

	return bodies, mrsTxs
}

// GetAllCurrentUsedTxs returns the cached transaction data for current round
func (tc *transactionCoordinator) GetAllCurrentUsedTxs(blockType block.Type) map[string]data.TransactionHandler {
	txPool := make(map[string]data.TransactionHandler, 0)
	interTxPool := make(map[string]data.TransactionHandler, 0)

	preProc := tc.getPreProcessor(blockType)
	if preProc != nil {
		txPool = preProc.GetAllCurrentUsedTxs()
	}

	interProc := tc.getInterimProcessor(blockType)
	if interProc != nil {
		interTxPool = interProc.GetAllCurrentFinishedTxs()
	}

	for hash, tx := range interTxPool {
		txPool[hash] = tx
	}

	return txPool
}

// RequestMiniBlocks request miniblocks if missing
func (tc *transactionCoordinator) RequestMiniBlocks(header data.HeaderHandler) {
	if header == nil || header.IsInterfaceNil() {
		return
	}

	crossMiniBlockHashes := header.GetMiniBlockHeadersWithDst(tc.shardCoordinator.SelfId())
	for key, senderShardId := range crossMiniBlockHashes {
		obj, _ := tc.miniBlockPool.Peek([]byte(key))
		if obj == nil {
			if !tc.requestedItemsHandler.Has(key) {
				go tc.onRequestMiniBlock(senderShardId, []byte(key))
				errNotCritical := tc.requestedItemsHandler.Add(key)
				if errNotCritical != nil {
					log.Trace("add requested item with error", errNotCritical.Error())
				}
			}
		}
	}
}

// receivedMiniBlock is a callback function when a new miniblock was received
// it will further ask for missing transactions
func (tc *transactionCoordinator) receivedMiniBlock(miniBlockHash []byte) {
	val, ok := tc.miniBlockPool.Peek(miniBlockHash)
	if !ok {
		return
	}

	miniBlock, ok := val.(*block.MiniBlock)
	if !ok {
		return
	}

	preproc := tc.getPreProcessor(miniBlock.Type)
	if preproc == nil || preproc.IsInterfaceNil() {
		return
	}

	_ = preproc.RequestTransactionsForMiniBlock(miniBlock)
}

// processMiniBlockComplete - all transactions must be processed together, otherwise error
func (tc *transactionCoordinator) processCompleteMiniBlock(
	preproc process.PreProcessor,
	miniBlock *block.MiniBlock,
	round uint64,
	haveTime func() bool,
) error {

	snapshot := tc.accounts.JournalLen()
	err := preproc.ProcessMiniBlock(miniBlock, haveTime, round)
	if err != nil {
		log.Debug("ProcessMiniBlock", "error", err.Error())

		errAccountState := tc.accounts.RevertToSnapshot(snapshot)
		if errAccountState != nil {
			// TODO: evaluate if reloading the trie from disk will might solve the problem
			log.Debug("RevertToSnapshot", "error", errAccountState.Error())
		}

		return err
	}

	return nil
}

// VerifyCreatedBlockTransactions checks whether the created transactions are the same as the one proposed
func (tc *transactionCoordinator) VerifyCreatedBlockTransactions(body block.Body) error {
	tc.mutInterimProcessors.RLock()
	defer tc.mutInterimProcessors.RUnlock()
	errMutex := sync.Mutex{}
	var errFound error
	// TODO: think if it is good in parallel or it is needed in sequences
	wg := sync.WaitGroup{}
	wg.Add(len(tc.interimProcessors))

	for key, interimProc := range tc.interimProcessors {
		if key == block.RewardsBlock {
			// this has to be processed last
			wg.Done()
			continue
		}

		go func(intermediateProcessor process.IntermediateTransactionHandler) {
			err := intermediateProcessor.VerifyInterMiniBlocks(body)
			if err != nil {
				errMutex.Lock()
				errFound = err
				errMutex.Unlock()
			}
			wg.Done()
		}(interimProc)
	}

	wg.Wait()

	if errFound != nil {
		return errFound
	}

	interimProc := tc.getInterimProcessor(block.RewardsBlock)
	if interimProc == nil {
		return nil
	}

	return interimProc.VerifyInterMiniBlocks(body)
}

// IsInterfaceNil returns true if there is no value under the interface
func (tc *transactionCoordinator) IsInterfaceNil() bool {
	if tc == nil {
		return true
	}
	return false
}
