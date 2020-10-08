package indexer

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strconv"
	"strings"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/indexer/disabled"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/receipt"
	"github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
)

const (
	txStatusSuccess     = "Success"
	txStatusPending     = "Pending"
	txStatusInvalid     = "Invalid"
	txStatusNotExecuted = "Not Executed"
	// A smart contract action (deploy, call, ...) should have minimum 2 smart contract results
	// exception to this rule are smart contract calls to ESDT contract
	minimumNumberOfSmartContractResults = 2
)

type txDatabaseProcessor struct {
	*commonProcessor
	txLogsProcessor process.TransactionLogProcessorDatabase
	hasher          hashing.Hasher
	marshalizer     marshal.Marshalizer
}

func newTxDatabaseProcessor(
	hasher hashing.Hasher,
	marshalizer marshal.Marshalizer,
	addressPubkeyConverter core.PubkeyConverter,
	validatorPubkeyConverter core.PubkeyConverter,
	feeConfig *config.FeeSettings,
) *txDatabaseProcessor {
	// this should never return error because is tested when economics file is created
	minGasLimit, _ := strconv.ParseUint(feeConfig.MinGasLimit, 10, 64)
	gasPerDataByte, _ := strconv.ParseUint(feeConfig.GasPerDataByte, 10, 64)

	return &txDatabaseProcessor{
		hasher:      hasher,
		marshalizer: marshalizer,
		commonProcessor: &commonProcessor{
			addressPubkeyConverter:   addressPubkeyConverter,
			validatorPubkeyConverter: validatorPubkeyConverter,
			minGasLimit:              minGasLimit,
			gasPerDataByte:           gasPerDataByte,
		},
		txLogsProcessor: disabled.NewNilTxLogsProcessor(),
	}
}

func (tdp *txDatabaseProcessor) prepareTransactionsForDatabase(
	body *block.Body,
	header data.HeaderHandler,
	txPool map[string]data.TransactionHandler,
	selfShardID uint32,
) ([]*Transaction, map[string]struct{}) {
	transactions, rewardsTxs, alteredAddresses := tdp.groupNormalTxsAndRewards(body, txPool, header, selfShardID)
	receipts := groupReceipts(txPool)
	scResults := groupSmartContractResults(txPool)

	transactions = tdp.setTransactionSearchOrder(transactions)
	for _, rec := range receipts {
		tx, ok := transactions[string(rec.TxHash)]
		if !ok {
			continue
		}

		gasUsed := big.NewInt(0).SetUint64(tx.GasPrice)
		gasUsed.Mul(gasUsed, big.NewInt(0).SetUint64(tx.GasLimit))
		gasUsed.Sub(gasUsed, rec.Value)
		gasUsed.Div(gasUsed, big.NewInt(0).SetUint64(tx.GasPrice))

		tx.GasUsed = gasUsed.Uint64()
	}

	countScResults := make(map[string]int)
	for scHash, scResult := range scResults {
		tx, ok := transactions[string(scResult.OriginalTxHash)]
		if !ok {
			continue
		}

		tx = tdp.addScResultInfoInTx(scHash, scResult, tx)
		countScResults[string(scResult.OriginalTxHash)]++
		delete(scResults, scHash)

		// append child smart contract results
		scrs := findAllChildScrResults(scHash, scResults)
		for scHash, sc := range scrs {
			tx = tdp.addScResultInfoInTx(scHash, sc, tx)
			countScResults[string(scResult.OriginalTxHash)]++
		}
	}

	for hash, nrScResult := range countScResults {
		if nrScResult < minimumNumberOfSmartContractResults {
			if len(transactions[hash].SmartContractResults) > 0 {
				scResultData := transactions[hash].SmartContractResults[0].Data
				if bytes.Contains(scResultData, []byte("@ok")) {
					// ESDT contract calls generate just one smart contract result
					continue
				}
			}

			if strings.Contains(string(transactions[hash].Data), "relayedTx") {
				continue
			}

			transactions[hash].Status = txStatusNotExecuted
		}
	}

	// TODO for the moment do not save logs in database
	// uncomment this when transaction logs need to be saved in database
	//for hash, tx := range transactions {
	//	txLog, ok := tdp.txLogsProcessor.GetLogFromCache([]byte(hash))
	//	if !ok {
	//		continue
	//	}
	//
	//	tx.Log = tdp.prepareTxLog(txLog)
	//}

	tdp.txLogsProcessor.Clean()

	return append(convertMapTxsToSlice(transactions), rewardsTxs...), alteredAddresses
}

func findAllChildScrResults(hash string, scrs map[string]*smartContractResult.SmartContractResult) map[string]*smartContractResult.SmartContractResult {
	scrResults := make(map[string]*smartContractResult.SmartContractResult, 0)
	for scrHash, scr := range scrs {
		if string(scr.OriginalTxHash) == hash {
			scrResults[scrHash] = scr
			delete(scrs, scrHash)
		}
	}

	return scrResults
}

func (tdp *txDatabaseProcessor) addScResultInfoInTx(scHash string, scr *smartContractResult.SmartContractResult, tx *Transaction) *Transaction {
	dbScResult := tdp.commonProcessor.convertScResultInDatabaseScr(scHash, scr)
	tx.SmartContractResults = append(tx.SmartContractResults, dbScResult)

	if isSCRForSenderWithGasUsed(dbScResult, tx) {
		gasUsed := tx.GasLimit - scr.GasLimit
		tx.GasUsed = gasUsed
	}

	return tx
}

func isSCRForSenderWithGasUsed(dbScResult ScResult, tx *Transaction) bool {
	isForSender := dbScResult.Receiver == tx.Sender
	isWithGasLimit := dbScResult.GasLimit != 0
	isFromCurrentTx := dbScResult.PreTxHash == tx.Hash

	return isFromCurrentTx && isForSender && isWithGasLimit
}

func (tdp *txDatabaseProcessor) prepareTxLog(log data.LogHandler) TxLog {
	scAddr := tdp.addressPubkeyConverter.Encode(log.GetAddress())
	events := log.GetLogEvents()

	txLogEvents := make([]Event, len(events))
	for i, event := range events {
		txLogEvents[i].Address = hex.EncodeToString(event.GetAddress())
		txLogEvents[i].Data = hex.EncodeToString(event.GetData())
		txLogEvents[i].Identifier = hex.EncodeToString(event.GetIdentifier())

		topics := event.GetTopics()
		txLogEvents[i].Topics = make([]string, len(topics))
		for j, topic := range topics {
			txLogEvents[i].Topics[j] = hex.EncodeToString(topic)
		}
	}

	return TxLog{
		Address: scAddr,
		Events:  txLogEvents,
	}
}

func convertMapTxsToSlice(txs map[string]*Transaction) []*Transaction {
	transactions := make([]*Transaction, len(txs))
	i := 0
	for _, tx := range txs {
		transactions[i] = tx
		i++
	}
	return transactions
}

func (tdp *txDatabaseProcessor) groupNormalTxsAndRewards(
	body *block.Body,
	txPool map[string]data.TransactionHandler,
	header data.HeaderHandler,
	selfShardID uint32,
) (
	map[string]*Transaction,
	[]*Transaction,
	map[string]struct{},
) {
	alteredAddresses := make(map[string]struct{})
	transactions := make(map[string]*Transaction)
	rewardsTxs := make([]*Transaction, 0)

	for _, mb := range body.MiniBlocks {
		mbHash, err := core.CalculateHash(tdp.marshalizer, tdp.hasher, mb)
		if err != nil {
			continue
		}

		mbTxStatus := txStatusPending
		if selfShardID == mb.ReceiverShardID {
			mbTxStatus = txStatusSuccess
		}

		switch mb.Type {
		case block.TxBlock:
			txs := getTransactions(txPool, mb.TxHashes)
			for hash, tx := range txs {
				dbTx := tdp.commonProcessor.buildTransaction(tx, []byte(hash), mbHash, mb, header, mbTxStatus)
				addToAlteredAddresses(dbTx, alteredAddresses, mb, selfShardID, false)
				transactions[hash] = dbTx
				delete(txPool, hash)
			}
		case block.InvalidBlock:
			txs := getTransactions(txPool, mb.TxHashes)
			for hash, tx := range txs {
				dbTx := tdp.commonProcessor.buildTransaction(tx, []byte(hash), mbHash, mb, header, txStatusInvalid)
				addToAlteredAddresses(dbTx, alteredAddresses, mb, selfShardID, false)
				transactions[hash] = dbTx
				delete(txPool, hash)
			}
		case block.RewardsBlock:
			rTxs := getRewardsTransaction(txPool, mb.TxHashes)
			for hash, rtx := range rTxs {
				dbTx := tdp.commonProcessor.buildRewardTransaction(rtx, []byte(hash), mbHash, mb, header, mbTxStatus)
				addToAlteredAddresses(dbTx, alteredAddresses, mb, selfShardID, true)
				alteredAddresses[dbTx.Receiver] = struct{}{}
				rewardsTxs = append(rewardsTxs, dbTx)
				delete(txPool, hash)
			}
		default:
			continue
		}
	}

	return transactions, rewardsTxs, alteredAddresses
}

func (tdp *txDatabaseProcessor) setTransactionSearchOrder(transactions map[string]*Transaction) map[string]*Transaction {
	currentOrder := uint32(0)
	for _, tx := range transactions {
		tx.SearchOrder = currentOrder
		currentOrder++
	}

	return transactions
}

func addToAlteredAddresses(
	tx *Transaction,
	alteredAddresses map[string]struct{},
	miniBlock *block.MiniBlock,
	selfShardID uint32,
	isRewardTx bool,
) {
	if selfShardID == miniBlock.SenderShardID && !isRewardTx {
		alteredAddresses[tx.Sender] = struct{}{}
	}
	if selfShardID == miniBlock.ReceiverShardID || miniBlock.ReceiverShardID == core.AllShardId {
		alteredAddresses[tx.Receiver] = struct{}{}
	}
}

func groupSmartContractResults(txPool map[string]data.TransactionHandler) map[string]*smartContractResult.SmartContractResult {
	scResults := make(map[string]*smartContractResult.SmartContractResult, 0)
	for hash, tx := range txPool {
		scResult, ok := tx.(*smartContractResult.SmartContractResult)
		if !ok {
			continue
		}
		scResults[hash] = scResult
	}

	return scResults
}

func groupReceipts(txPool map[string]data.TransactionHandler) []*receipt.Receipt {
	receipts := make([]*receipt.Receipt, 0)
	for hash, tx := range txPool {
		rec, ok := tx.(*receipt.Receipt)
		if !ok {
			continue
		}

		receipts = append(receipts, rec)
		delete(txPool, hash)
	}

	return receipts
}

func getTransactions(txPool map[string]data.TransactionHandler,
	txHashes [][]byte,
) map[string]*transaction.Transaction {
	transactions := make(map[string]*transaction.Transaction)
	for _, txHash := range txHashes {
		txHandler, ok := txPool[string(txHash)]
		if !ok {
			continue
		}

		tx, ok := txHandler.(*transaction.Transaction)
		if !ok {
			continue
		}
		transactions[string(txHash)] = tx
	}
	return transactions
}

func getRewardsTransaction(txPool map[string]data.TransactionHandler,
	txHashes [][]byte,
) map[string]*rewardTx.RewardTx {
	rewardsTxs := make(map[string]*rewardTx.RewardTx)
	for _, txHash := range txHashes {
		txHandler, ok := txPool[string(txHash)]
		if !ok {
			continue
		}

		reward, ok := txHandler.(*rewardTx.RewardTx)
		if !ok {
			continue
		}
		rewardsTxs[string(txHash)] = reward
	}
	return rewardsTxs
}
