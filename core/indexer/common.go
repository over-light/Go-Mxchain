package indexer

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/statistics"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/receipt"
	"github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/marshal"
)

func checkElasticSearchParams(arguments ElasticIndexerArgs) error {
	if arguments.Url == "" {
		return core.ErrNilUrl
	}
	if arguments.UserName == "" {
		return ErrEmptyUserName
	}
	if arguments.Password == "" {
		return ErrEmptyPassword
	}
	if check.IfNil(arguments.Marshalizer) {
		return core.ErrNilMarshalizer
	}
	if check.IfNil(arguments.Hasher) {
		return core.ErrNilHasher
	}
	if check.IfNil(arguments.NodesCoordinator) {
		return core.ErrNilNodesCoordinator
	}
	if arguments.EpochStartNotifier == nil {
		return core.ErrNilEpochStartNotifier
	}

	return nil
}

func timestampMapping() io.Reader {
	return strings.NewReader(
		`{
				"settings": {"index": {"sort.field": "timestamp", "sort.order": "desc"}},
				"mappings": {"_doc": {"properties": {"timestamp": {"type": "date"}}}}
			}`,
	)
}

func prepareGeneralInfo(tpsBenchmark statistics.TPSBenchmark) bytes.Buffer {
	var buff bytes.Buffer

	meta := []byte(fmt.Sprintf(`{ "index" : { "_id" : "%s", "_type" : "%s" } }%s`, metachainTpsDocID, tpsIndex, "\n"))
	generalInfo := TPS{
		LiveTPS:               tpsBenchmark.LiveTPS(),
		PeakTPS:               tpsBenchmark.PeakTPS(),
		NrOfShards:            tpsBenchmark.NrOfShards(),
		BlockNumber:           tpsBenchmark.BlockNumber(),
		RoundNumber:           tpsBenchmark.RoundNumber(),
		RoundTime:             tpsBenchmark.RoundTime(),
		AverageBlockTxCount:   tpsBenchmark.AverageBlockTxCount(),
		LastBlockTxCount:      tpsBenchmark.LastBlockTxCount(),
		TotalProcessedTxCount: tpsBenchmark.TotalProcessedTxCount(),
	}

	serializedInfo, err := json.Marshal(generalInfo)
	if err != nil {
		log.Debug("indexer: could not serialize tps info, will skip indexing tps this round")
		return buff
	}
	// append a newline foreach element in the bulk we create
	serializedInfo = append(serializedInfo, "\n"...)

	buff.Grow(len(meta) + len(serializedInfo))
	_, err = buff.Write(meta)
	if err != nil {
		log.Warn("elastic search: update TPS write meta", "error", err.Error())
	}
	_, err = buff.Write(serializedInfo)
	if err != nil {
		log.Warn("elastic search: update TPS write serialized info", "error", err.Error())
	}

	return buff
}

func serializeShardInfo(shardInfo statistics.ShardStatistic) ([]byte, []byte) {
	meta := []byte(fmt.Sprintf(`{ "index" : { "_id" : "%s%d", "_type" : "%s" } }%s`,
		shardTpsDocIDPrefix, shardInfo.ShardID(), tpsIndex, "\n"))

	bigTxCount := big.NewInt(int64(shardInfo.AverageBlockTxCount()))
	shardTPS := TPS{
		ShardID:               shardInfo.ShardID(),
		LiveTPS:               shardInfo.LiveTPS(),
		PeakTPS:               shardInfo.PeakTPS(),
		AverageTPS:            shardInfo.AverageTPS(),
		AverageBlockTxCount:   bigTxCount,
		CurrentBlockNonce:     shardInfo.CurrentBlockNonce(),
		LastBlockTxCount:      shardInfo.LastBlockTxCount(),
		TotalProcessedTxCount: shardInfo.TotalProcessedTxCount(),
	}

	serializedInfo, err := json.Marshal(shardTPS)
	if err != nil {
		log.Debug("indexer: could not serialize tps info, will skip indexing tps this shard")
		return nil, nil
	}
	// append a newline foreach element in the bulk we create
	serializedInfo = append(serializedInfo, "\n"...)

	return serializedInfo, meta
}

func getTransactionByType(
	tx data.TransactionHandler,
	txHash []byte,
	mbHash []byte,
	blockHash []byte,
	mb *block.MiniBlock,
	header data.HeaderHandler,
	txStatus string,
) *Transaction {
	switch currentType := tx.(type) {
	case *transaction.Transaction:
		return buildTransaction(currentType, txHash, mbHash, blockHash, mb, header, txStatus)
	case *smartContractResult.SmartContractResult:
		return buildSmartContractResult(currentType, txHash, mbHash, blockHash, mb, header)
	case *rewardTx.RewardTx:
		return buildRewardTransaction(currentType, txHash, mbHash, blockHash, mb, header, txStatus)
	case *receipt.Receipt:
		return buildReceiptTransaction(currentType, txHash, mbHash, blockHash, mb, header)
	default:
		return nil
	}
}

func buildTransaction(
	tx *transaction.Transaction,
	txHash []byte,
	mbHash []byte,
	blockHash []byte,
	mb *block.MiniBlock,
	header data.HeaderHandler,
	txStatus string,
) *Transaction {
	return &Transaction{
		Hash:          hex.EncodeToString(txHash),
		MBHash:        hex.EncodeToString(mbHash),
		BlockHash:     hex.EncodeToString(blockHash),
		Nonce:         tx.Nonce,
		Round:         header.GetRound(),
		Value:         tx.Value.String(),
		Receiver:      hex.EncodeToString(tx.RcvAddr),
		Sender:        hex.EncodeToString(tx.SndAddr),
		ReceiverShard: mb.ReceiverShardID,
		SenderShard:   mb.SenderShardID,
		GasPrice:      tx.GasPrice,
		GasLimit:      tx.GasLimit,
		Data:          string(tx.Data),
		Signature:     hex.EncodeToString(tx.Signature),
		Timestamp:     time.Duration(header.GetTimeStamp()),
		Status:        txStatus,
	}
}

func buildSmartContractResult(
	scr *smartContractResult.SmartContractResult,
	txHash []byte,
	mbHash []byte,
	blockHash []byte,
	mb *block.MiniBlock,
	header data.HeaderHandler,
) *Transaction {
	return &Transaction{
		Hash:          hex.EncodeToString(txHash),
		MBHash:        hex.EncodeToString(mbHash),
		BlockHash:     hex.EncodeToString(blockHash),
		Nonce:         scr.Nonce,
		Round:         header.GetRound(),
		Value:         scr.Value.String(),
		Receiver:      hex.EncodeToString(scr.RcvAddr),
		Sender:        hex.EncodeToString(scr.SndAddr),
		ReceiverShard: mb.ReceiverShardID,
		SenderShard:   mb.SenderShardID,
		GasPrice:      scr.GasPrice,
		GasLimit:      scr.GasPrice,
		Data:          string(scr.Data),
		Signature:     "",
		Timestamp:     time.Duration(header.GetTimeStamp()),
		Status:        "Success",
	}
}

func buildRewardTransaction(
	rTx *rewardTx.RewardTx,
	txHash []byte,
	mbHash []byte,
	blockHash []byte,
	mb *block.MiniBlock,
	header data.HeaderHandler,
	txStatus string,
) *Transaction {
	return &Transaction{
		Hash:          hex.EncodeToString(txHash),
		MBHash:        hex.EncodeToString(mbHash),
		BlockHash:     hex.EncodeToString(blockHash),
		Nonce:         0,
		Round:         rTx.Round,
		Value:         rTx.Value.String(),
		Receiver:      hex.EncodeToString(rTx.RcvAddr),
		Sender:        fmt.Sprintf("%d", core.MetachainShardId),
		ReceiverShard: mb.ReceiverShardID,
		SenderShard:   mb.SenderShardID,
		GasPrice:      0,
		GasLimit:      0,
		Data:          "",
		Signature:     "",
		Timestamp:     time.Duration(header.GetTimeStamp()),
		Status:        txStatus,
	}
}

func buildReceiptTransaction(
	rpt *receipt.Receipt,
	txHash []byte,
	mbHash []byte,
	blockHash []byte,
	mb *block.MiniBlock,
	header data.HeaderHandler,
) *Transaction {
	return &Transaction{
		Hash:          hex.EncodeToString(txHash),
		MBHash:        hex.EncodeToString(mbHash),
		BlockHash:     hex.EncodeToString(blockHash),
		Nonce:         rpt.GetNonce(),
		Round:         header.GetRound(),
		Value:         rpt.Value.String(),
		Receiver:      hex.EncodeToString(rpt.GetRcvAddr()),
		Sender:        hex.EncodeToString(rpt.GetSndAddr()),
		ReceiverShard: mb.ReceiverShardID,
		SenderShard:   mb.SenderShardID,
		GasPrice:      0,
		GasLimit:      0,
		Data:          string(rpt.Data),
		Signature:     "",
		Timestamp:     time.Duration(header.GetTimeStamp()),
		Status:        "Success",
	}
}

func serializeBulkMiniBlocks(hdrShardID uint32, bulkMbs []*Miniblock) (insert, update bytes.Buffer) {
	for _, mb := range bulkMbs {
		if hdrShardID == mb.SenderShardID {
			//insert miniblock
			meta := []byte(fmt.Sprintf(`{ "index" : { "_id" : "%s", "_type" : "%s" } }%s`, mb.Hash, "_doc", "\n"))
			serializedData, err := json.Marshal(mb)
			if err != nil {
				log.Debug("indexer: marshal",
					"error", "could not serialize miniblock, will skip indexing",
					"mb hash", mb.Hash)
				continue
			}

			insert = prepareBufferMiniblocks(insert, meta, serializedData)

		} else {
			// update miniblock
			meta := []byte(fmt.Sprintf(`{ "update" : { "_id" : "%s", "_type" : "%s"  } }%s`, mb.Hash, "_doc", "\n"))
			serializedData := []byte(fmt.Sprintf(`{ "doc" : { "receiverBlockHash" : "%s" } }`, mb.ReceiverBlockHash))

			update = prepareBufferMiniblocks(update, meta, serializedData)
		}
	}

	return
}

func prepareBufferMiniblocks(buff bytes.Buffer, meta, serializedData []byte) bytes.Buffer {
	// append a newline for each element
	serializedData = append(serializedData, "\n"...)
	buff.Grow(len(meta) + len(serializedData))
	_, err := buff.Write(meta)
	if err != nil {
		log.Warn("elastic search: serialize bulk miniblocks, write meta", "error", err.Error())
	}
	_, err = buff.Write(serializedData)
	if err != nil {
		log.Warn("elastic search: serialize bulk miniblocks, write serialized miniblock", "error", err.Error())
	}

	return buff
}

func serializeBulkTxs(bulk []*Transaction) bytes.Buffer {
	var buff bytes.Buffer
	for _, tx := range bulk {
		meta := []byte(fmt.Sprintf(`{ "index" : { "_id" : "%s", "_type" : "%s" } }%s`, tx.Hash, "_doc", "\n"))
		serializedTx, err := json.Marshal(tx)
		if err != nil {
			log.Debug("indexer: marshal",
				"error", "could not serialize transaction, will skip indexing",
				"tx hash", tx.Hash)
			continue
		}
		// append a newline for each element
		serializedTx = append(serializedTx, "\n"...)

		buff.Grow(len(meta) + len(serializedTx))
		_, err = buff.Write(meta)
		if err != nil {
			log.Warn("elastic search: serialize bulk tx, write meta", "error", err.Error())
		}
		_, err = buff.Write(serializedTx)
		if err != nil {
			log.Warn("elastic search: serialize bulk tx, write serialized tx", "error", err.Error())
		}
	}

	return buff
}

func computeSizeOfTxs(marshalizer marshal.Marshalizer, txs map[string]data.TransactionHandler) int {
	if len(txs) == 0 {
		return 0
	}

	txsSize := 0
	for _, tx := range txs {
		txBytes, err := marshalizer.Marshal(tx)
		if err != nil {
			log.Debug("indexer: marshal transaction", "error", err)
			continue
		}

		txsSize += len(txBytes)
	}

	return txsSize
}
