package preprocess

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/sliceUtil"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type miniBlocksCompaction struct {
	economicsFee     process.FeeHandler
	shardCoordinator sharding.Coordinator
	gasHandler       process.GasHandler

	mapHashToTx             map[string]data.TransactionHandler
	mapMinSenderNonce       map[string]uint64
	mapUnallocatedTxsHashes map[string]struct{}
	gasSpentSender          map[int]uint64
	gasSpentReceiver        map[int]uint64

	mutMiniBlocksCompaction sync.RWMutex
}

// NewMiniBlocksCompaction creates a new mini blocks compaction object
func NewMiniBlocksCompaction(
	economicsFee process.FeeHandler,
	shardCoordinator sharding.Coordinator,
	gasHandler process.GasHandler,
) (*miniBlocksCompaction, error) {

	if check.IfNil(economicsFee) {
		return nil, process.ErrNilEconomicsFeeHandler
	}
	if check.IfNil(shardCoordinator) {
		return nil, process.ErrNilShardCoordinator
	}
	if check.IfNil(gasHandler) {
		return nil, process.ErrNilGasHandler
	}

	mbc := miniBlocksCompaction{
		economicsFee:     economicsFee,
		shardCoordinator: shardCoordinator,
		gasHandler:       gasHandler,
	}

	mbc.mapHashToTx = make(map[string]data.TransactionHandler)
	mbc.mapMinSenderNonce = make(map[string]uint64)
	mbc.mapUnallocatedTxsHashes = make(map[string]struct{})
	mbc.gasSpentSender = make(map[int]uint64)
	mbc.gasSpentReceiver = make(map[int]uint64)

	return &mbc, nil
}

// Compact method tries to compact the given mini blocks to have only one mini block per sender/received pair
func (mbc *miniBlocksCompaction) Compact(
	miniBlocks block.MiniBlockSlice,
	mapHashToTx map[string]data.TransactionHandler,
) block.MiniBlockSlice {

	mbc.mutMiniBlocksCompaction.Lock()
	defer mbc.mutMiniBlocksCompaction.Unlock()

	if len(miniBlocks) <= 1 {
		return miniBlocks
	}

	var err error

	mbc.mapHashToTx = mapHashToTx
	mbc.gasSpentSender = make(map[int]uint64)
	mbc.gasSpentReceiver = make(map[int]uint64)

	for index, miniBlock := range miniBlocks {
		mbc.gasSpentSender[index], mbc.gasSpentReceiver[index], err = mbc.gasHandler.ComputeGasConsumedByMiniBlock(
			miniBlock,
			mapHashToTx)

		if err != nil {
			log.Debug("computeGasConsumedByMiniBlock", "error", err.Error())
			return miniBlocks
		}
	}

	compactedMiniBlocks := make(block.MiniBlockSlice, 0)
	compactedMiniBlocks = append(compactedMiniBlocks, miniBlocks[0])

	for index, miniBlock := range miniBlocks {
		if index == 0 {
			continue
		}

		compactedMiniBlocks = mbc.merge(compactedMiniBlocks, miniBlock, index)
	}

	if len(miniBlocks) > len(compactedMiniBlocks) {
		log.Debug("compacted miniblocks",
			"from", len(miniBlocks),
			"to", len(compactedMiniBlocks),
		)
	}

	return compactedMiniBlocks
}

func (mbc *miniBlocksCompaction) merge(
	mergedMiniBlocks block.MiniBlockSlice,
	miniBlock *block.MiniBlock,
	srcIndex int,
) block.MiniBlockSlice {

	for dstIndex, mergedMiniBlock := range mergedMiniBlocks {
		sameType := miniBlock.Type == mergedMiniBlock.Type
		sameSenderShard := miniBlock.SenderShardID == mergedMiniBlock.SenderShardID
		sameReceiverShard := miniBlock.ReceiverShardID == mergedMiniBlock.ReceiverShardID

		canMerge := sameSenderShard && sameReceiverShard && sameType
		if canMerge {
			isFullyMerged := mbc.computeMerge(mergedMiniBlock, dstIndex, miniBlock, srcIndex)
			if isFullyMerged {
				return mergedMiniBlocks
			}
		}
	}

	mergedMiniBlocks = append(mergedMiniBlocks, miniBlock)

	return mergedMiniBlocks
}

// Expand method tries to expand the given mini blocks to their initial state before compaction
func (mbc *miniBlocksCompaction) Expand(
	miniBlocks block.MiniBlockSlice,
	mapHashToTx map[string]data.TransactionHandler,
) (block.MiniBlockSlice, error) {

	mbc.mutMiniBlocksCompaction.Lock()
	defer mbc.mutMiniBlocksCompaction.Unlock()

	mbc.mapHashToTx = mapHashToTx
	mbc.mapMinSenderNonce = make(map[string]uint64)
	mbc.mapUnallocatedTxsHashes = make(map[string]struct{})

	expandedMiniBlocks := make(block.MiniBlockSlice, 0)
	miniBlocksToExpand := make(block.MiniBlockSlice, 0)

	for _, miniBlock := range miniBlocks {
		if miniBlock.SenderShardID == mbc.shardCoordinator.SelfId() {
			miniBlocksToExpand = append(miniBlocksToExpand, miniBlock)
			continue
		}

		expandedMiniBlocks = append(expandedMiniBlocks, miniBlock)
	}

	if len(miniBlocksToExpand) > 0 {
		expandedMiniBlocksFromMe, err := mbc.expandMiniBlocks(miniBlocksToExpand)
		if err != nil {
			return nil, err
		}

		expandedMiniBlocks = append(expandedMiniBlocks, expandedMiniBlocksFromMe...)
	}

	if len(miniBlocks) < len(expandedMiniBlocks) {
		log.Debug("expanded miniblocks",
			"from", len(miniBlocks),
			"to", len(expandedMiniBlocks),
		)
	}

	return expandedMiniBlocks, nil
}

func (mbc *miniBlocksCompaction) expandMiniBlocks(miniBlocks block.MiniBlockSlice) (block.MiniBlockSlice, error) {
	for _, miniBlock := range miniBlocks {
		for _, txHash := range miniBlock.TxHashes {
			tx, ok := mbc.mapHashToTx[string(txHash)]
			if !ok {
				log.Debug("missing transaction in expandMiniBlocks ", "type", miniBlock.Type, "txHash", txHash)
				return nil, process.ErrMissingTransaction
			}

			nonce, ok := mbc.mapMinSenderNonce[string(tx.GetSndAddress())]
			if !ok || nonce > tx.GetNonce() {
				mbc.mapMinSenderNonce[string(tx.GetSndAddress())] = tx.GetNonce()
			}

			mbc.mapUnallocatedTxsHashes[string(txHash)] = struct{}{}
		}
	}

	expandedMiniBlocks := make(block.MiniBlockSlice, 0)

	for len(mbc.mapUnallocatedTxsHashes) > 0 {
		createdMiniBlocks, err := mbc.createExpandedMiniBlocks(miniBlocks)
		if err != nil {
			return nil, err
		}

		if len(createdMiniBlocks) == 0 {
			break
		}

		expandedMiniBlocks = append(expandedMiniBlocks, createdMiniBlocks...)
	}

	return expandedMiniBlocks, nil
}

func (mbc *miniBlocksCompaction) createExpandedMiniBlocks(
	miniBlocks block.MiniBlockSlice,
) (block.MiniBlockSlice, error) {

	expandedMiniBlocks := make(block.MiniBlockSlice, 0)

	for _, miniBlock := range miniBlocks {
		if len(mbc.mapUnallocatedTxsHashes) == 0 {
			break
		}

		miniBlockForShard, err := mbc.createMiniBlockForShard(miniBlock)
		if err != nil {
			return nil, err
		}

		if len(miniBlockForShard.TxHashes) > 0 {
			expandedMiniBlocks = append(expandedMiniBlocks, miniBlockForShard)
		}
	}

	return expandedMiniBlocks, nil
}

func (mbc *miniBlocksCompaction) createMiniBlockForShard(miniBlock *block.MiniBlock) (*block.MiniBlock, error) {
	miniBlockForShard := &block.MiniBlock{}
	miniBlockForShard.TxHashes = make([][]byte, 0)
	miniBlockForShard.ReceiverShardID = miniBlock.ReceiverShardID
	miniBlockForShard.SenderShardID = miniBlock.SenderShardID
	miniBlockForShard.Type = miniBlock.Type

	for _, txHash := range miniBlock.TxHashes {
		if len(mbc.mapUnallocatedTxsHashes) == 0 {
			break
		}

		_, ok := mbc.mapUnallocatedTxsHashes[string(txHash)]
		if !ok {
			continue
		}

		tx, ok := mbc.mapHashToTx[string(txHash)]
		if !ok {
			log.Debug("missing transaction in createMiniBlockForShard ", "type", miniBlock.Type, "txHash", txHash)
			return nil, process.ErrMissingTransaction
		}

		nonce := mbc.mapMinSenderNonce[string(tx.GetSndAddress())]
		if tx.GetNonce() == nonce {
			mbc.mapMinSenderNonce[string(tx.GetSndAddress())] = nonce + 1
			miniBlockForShard.TxHashes = append(miniBlockForShard.TxHashes, txHash)
			delete(mbc.mapUnallocatedTxsHashes, string(txHash))
		}
	}

	return miniBlockForShard, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (mbc *miniBlocksCompaction) IsInterfaceNil() bool {
	if mbc == nil {
		return true
	}
	return false
}

func (mbc *miniBlocksCompaction) computeMerge(
	dstMiniBlock *block.MiniBlock,
	dstIndex int,
	srcMiniBlock *block.MiniBlock,
	srcIndex int,
) bool {

	newGasSpentSender := mbc.gasSpentSender[dstIndex] + mbc.gasSpentSender[srcIndex]
	newGasSpentReceiver := mbc.gasSpentReceiver[dstIndex] + mbc.gasSpentReceiver[srcIndex]
	senderGasLimitOK := newGasSpentSender <= mbc.economicsFee.MaxGasLimitPerBlock()
	receiverGasLimitOK := newGasSpentReceiver <= mbc.economicsFee.MaxGasLimitPerBlock()

	canBeFullyMerged := senderGasLimitOK && receiverGasLimitOK
	if canBeFullyMerged {
		dstMiniBlock.TxHashes = append(dstMiniBlock.TxHashes, srcMiniBlock.TxHashes...)
		mbc.gasSpentSender[dstIndex] += mbc.gasSpentSender[srcIndex]
		mbc.gasSpentReceiver[dstIndex] += mbc.gasSpentReceiver[srcIndex]
		return true
	}

	mbGasSpentSender := mbc.gasSpentSender[dstIndex]
	mbGasSpentReceiver := mbc.gasSpentReceiver[dstIndex]

	txHashes := mbc.getMaxTxsForMerge(&mbGasSpentSender, &mbGasSpentReceiver, srcMiniBlock)
	if len(txHashes) > 0 {
		dstMiniBlock.TxHashes = append(dstMiniBlock.TxHashes, txHashes...)
		srcMiniBlock.TxHashes = mbc.removeTxHashesFromMiniBlock(srcMiniBlock, txHashes)

		gasTransferred := mbGasSpentSender - mbc.gasSpentSender[dstIndex]
		mbc.gasSpentSender[dstIndex] += gasTransferred
		mbc.gasSpentSender[srcIndex] -= gasTransferred

		gasTransferred = mbGasSpentReceiver - mbc.gasSpentReceiver[dstIndex]
		mbc.gasSpentReceiver[dstIndex] += gasTransferred
		mbc.gasSpentReceiver[srcIndex] -= gasTransferred
	}

	return false
}

func (mbc *miniBlocksCompaction) getMaxTxsForMerge(
	gasSpentSender *uint64,
	gasSpentReceiver *uint64,
	miniBlock *block.MiniBlock,
) [][]byte {

	txHashes := make([][]byte, 0, len(miniBlock.TxHashes))
	for _, txHash := range miniBlock.TxHashes {
		txHandler, ok := mbc.mapHashToTx[string(txHash)]
		if !ok {
			break
		}

		txGasSpentSender, txGasSpentReceiver, err := mbc.gasHandler.ComputeGasConsumedByTx(
			miniBlock.SenderShardID,
			miniBlock.ReceiverShardID,
			txHandler,
		)
		if err != nil {
			break
		}

		newGasSpentSender := *gasSpentSender + txGasSpentSender
		newGasSpentReceiver := *gasSpentReceiver + txGasSpentReceiver
		if newGasSpentSender > mbc.economicsFee.MaxGasLimitPerBlock() ||
			newGasSpentReceiver > mbc.economicsFee.MaxGasLimitPerBlock() {
			break
		}

		txHashes = append(txHashes, txHash)
		*gasSpentSender += txGasSpentSender
		*gasSpentReceiver += txGasSpentReceiver
	}

	return sliceUtil.TrimSliceSliceByte(txHashes)
}

func (mbc *miniBlocksCompaction) removeTxHashesFromMiniBlock(miniBlock *block.MiniBlock, txHashes [][]byte) [][]byte {
	mapTxHashesToBeRemoved := make(map[string]struct{}, len(txHashes))
	for _, txHash := range txHashes {
		mapTxHashesToBeRemoved[string(txHash)] = struct{}{}
	}

	preservedTxHashes := make([][]byte, 0, len(miniBlock.TxHashes))
	for _, txHash := range miniBlock.TxHashes {
		_, ok := mapTxHashesToBeRemoved[string(txHash)]
		if ok {
			continue
		}

		preservedTxHashes = append(preservedTxHashes, txHash)
	}

	return sliceUtil.TrimSliceSliceByte(preservedTxHashes)
}
