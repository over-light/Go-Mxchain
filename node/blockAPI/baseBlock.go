package blockAPI

import (
	"encoding/hex"
	"fmt"
	"time"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core/fullHistory"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/marshal"
)

type baseAPIBockProcessor struct {
	isFullHistoryNode        bool
	selfShardID              uint32
	store                    dataRetriever.StorageService
	marshalizer              marshal.Marshalizer
	uint64ByteSliceConverter typeConverters.Uint64ByteSliceConverter
	historyRepo              fullHistory.HistoryRepository
	unmarshalTx              func(txBytes []byte, txType transaction.TxType) (*transaction.ApiTransactionResult, error)
}

var log = logger.GetOrCreate("node/blockAPI")

func (bap *baseAPIBockProcessor) getTxsByMb(mbHeader *block.MiniBlockHeader, epoch uint32) []*transaction.ApiTransactionResult {
	miniblockHash := mbHeader.Hash
	mbBytes, err := bap.getFromStorerWithEpoch(dataRetriever.MiniBlockUnit, miniblockHash, epoch)
	if err != nil {
		log.Warn("cannot get miniblock from storage",
			"hash", hex.EncodeToString(miniblockHash),
			"error", err.Error())
		return nil
	}

	miniBlock := &block.MiniBlock{}
	err = bap.marshalizer.Unmarshal(miniBlock, mbBytes)
	if err != nil {
		log.Warn("cannot unmarshal miniblock",
			"hash", hex.EncodeToString(miniblockHash),
			"error", err.Error())
		return nil
	}

	switch miniBlock.Type {
	case block.TxBlock:
		return bap.getTxsFromMiniblock(miniBlock, miniblockHash, epoch, transaction.TxTypeNormal, dataRetriever.TransactionUnit)
	case block.RewardsBlock:
		return bap.getTxsFromMiniblock(miniBlock, miniblockHash, epoch, transaction.TxTypeReward, dataRetriever.RewardTransactionUnit)
	case block.SmartContractResultBlock:
		return bap.getTxsFromMiniblock(miniBlock, miniblockHash, epoch, transaction.TxTypeUnsigned, dataRetriever.UnsignedTransactionUnit)
	case block.InvalidBlock:
		return bap.getTxsFromMiniblock(miniBlock, miniblockHash, epoch, transaction.TxTypeInvalid, dataRetriever.TransactionUnit)
	default:
		return nil
	}
}

func (bap *baseAPIBockProcessor) getTxsFromMiniblock(
	miniblock *block.MiniBlock,
	miniblockHash []byte,
	epoch uint32,
	txType transaction.TxType,
	unit dataRetriever.UnitType,
) []*transaction.ApiTransactionResult {
	storer := bap.store.GetStorer(unit)
	start := time.Now()
	marshalizedTxs, err := storer.GetBulkFromEpoch(miniblock.TxHashes, epoch)
	if err != nil {
		log.Warn("cannot get from storage transactions",
			"error", err.Error())
		return []*transaction.ApiTransactionResult{}
	}
	log.Debug(fmt.Sprintf("GetBulkFromEpoch took %s", time.Since(start)))

	start = time.Now()
	txs := make([]*transaction.ApiTransactionResult, 0)
	for txHash, txBytes := range marshalizedTxs {
		tx, err := bap.unmarshalTx(txBytes, txType)
		if err != nil {
			log.Warn("cannot unmarshal transaction",
				"hash", hex.EncodeToString([]byte(txHash)),
				"error", err.Error())
			continue
		}
		tx.Hash = hex.EncodeToString([]byte(txHash))
		tx.MiniBlockType = miniblock.Type.String()
		tx.MiniBlockHash = hex.EncodeToString([]byte(miniblockHash))
		tx.SourceShard = miniblock.SenderShardID
		tx.DestinationShard = miniblock.ReceiverShardID

		tx.Status = (&transaction.StatusComputer{
			MiniblockType:    miniblock.Type,
			SourceShard:      tx.SourceShard,
			DestinationShard: tx.DestinationShard,
			Receiver:         tx.Tx.GetRcvAddr(),
			TransactionData:  tx.Data,
			SelfShard:        bap.selfShardID,
		}).ComputeStatusWhenInStorageKnowingMiniblock()

		txs = append(txs, tx)
	}
	log.Debug(fmt.Sprintf("UnmarshalTransactions took %s", time.Since(start)))

	return txs
}

func (bap *baseAPIBockProcessor) getFromStorer(unit dataRetriever.UnitType, key []byte) ([]byte, error) {
	if !bap.isFullHistoryNode {
		return bap.store.Get(unit, key)
	}

	epoch, err := bap.historyRepo.GetEpochByHash(key)
	if err != nil {
		return nil, err
	}

	storer := bap.store.GetStorer(unit)
	return storer.GetFromEpoch(key, epoch)
}

func (bap *baseAPIBockProcessor) getFromStorerWithEpoch(unit dataRetriever.UnitType, key []byte, epoch uint32) ([]byte, error) {
	storer := bap.store.GetStorer(unit)
	return storer.GetFromEpoch(key, epoch)
}
