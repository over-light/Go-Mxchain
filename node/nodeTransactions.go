package node

import (
	"encoding/hex"
	"fmt"

	"github.com/ElrondNetwork/elrond-go/core"
	rewardTxData "github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
)

type transactionType string

const (
	normalTx   transactionType = "normal"
	unsignedTx transactionType = "unsignedTx"
	rewardTx   transactionType = "rewardTx"
	invalidTx  transactionType = "invalidTx"
)

// GetTransaction gets the transaction based on the given hash. It will search in the cache and the storage and
// will return the transaction in a format which can be respected by all types of transactions (normal, reward or unsigned)
func (n *Node) GetTransaction(txHash string) (*transaction.ApiTransactionResult, error) {
	if !n.apiTransactionByHashThrottler.CanProcess() {
		return nil, ErrSystemBusyTxHash
	}

	n.apiTransactionByHashThrottler.StartProcessing()
	defer n.apiTransactionByHashThrottler.EndProcessing()

	hash, err := hex.DecodeString(txHash)
	if err != nil {
		return nil, err
	}

	txObj, txType, found := n.getTxObjFromDataPool(hash)
	if found {
		return n.castObjToTransaction(txObj, txType)
	}

	txBytes, txType, found := n.getTxBytesFromStorage(hash)
	if found {
		return n.unmarshalTransaction(txBytes, txType)
	}

	return nil, fmt.Errorf("transaction not found")
}

// GetTransactionStatus gets the transaction status
func (n *Node) GetTransactionStatus(txHash string) (string, error) {
	if !n.apiTransactionByHashThrottler.CanProcess() {
		return "", ErrSystemBusyTxHash
	}

	n.apiTransactionByHashThrottler.StartProcessing()
	defer n.apiTransactionByHashThrottler.EndProcessing()

	hash, err := hex.DecodeString(txHash)
	if err != nil {
		return "", err
	}

	_, _, foundInDataPool := n.getTxObjFromDataPool(hash)
	if foundInDataPool {
		return string(core.TxStatusReceived), nil
	}

	foundInStorage := n.isTxInStorage(hash)
	if foundInStorage {
		return string(core.TxStatusExecuted), nil
	}

	return string(core.TxStatusUnknown), nil
}

func (n *Node) getTxObjFromDataPool(hash []byte) (interface{}, transactionType, bool) {
	txsPool := n.dataPool.Transactions()
	txObj, found := txsPool.SearchFirstData(hash)
	if found && txObj != nil {
		return txObj, normalTx, true
	}

	rewardTxsPool := n.dataPool.RewardTransactions()
	txObj, found = rewardTxsPool.SearchFirstData(hash)
	if found && txObj != nil {
		return txObj, rewardTx, true
	}

	unsignedTxsPool := n.dataPool.UnsignedTransactions()
	txObj, found = unsignedTxsPool.SearchFirstData(hash)
	if found && txObj != nil {
		return txObj, unsignedTx, true
	}

	return nil, invalidTx, false
}

func (n *Node) isTxInStorage(hash []byte) bool {
	txsStorer := n.store.GetStorer(dataRetriever.TransactionUnit)
	err := txsStorer.Has(hash)
	if err == nil {
		return true
	}

	rewardTxsStorer := n.store.GetStorer(dataRetriever.RewardTransactionUnit)
	err = rewardTxsStorer.Has(hash)
	if err == nil {
		return true
	}

	unsignedTxsStorer := n.store.GetStorer(dataRetriever.UnsignedTransactionUnit)
	err = unsignedTxsStorer.Has(hash)
	return err == nil
}

func (n *Node) getTxBytesFromStorage(hash []byte) ([]byte, transactionType, bool) {
	txsStorer := n.store.GetStorer(dataRetriever.TransactionUnit)
	txBytes, err := txsStorer.SearchFirst(hash)
	if err == nil {
		return txBytes, normalTx, true
	}

	rewardTxsStorer := n.store.GetStorer(dataRetriever.RewardTransactionUnit)
	txBytes, err = rewardTxsStorer.SearchFirst(hash)
	if err == nil {
		return txBytes, rewardTx, true
	}

	unsignedTxsStorer := n.store.GetStorer(dataRetriever.UnsignedTransactionUnit)
	txBytes, err = unsignedTxsStorer.SearchFirst(hash)
	if err == nil {
		return txBytes, unsignedTx, true
	}

	return nil, invalidTx, false
}

func (n *Node) castObjToTransaction(txObj interface{}, txType transactionType) (*transaction.ApiTransactionResult, error) {
	switch txType {
	case normalTx:
		if tx, ok := txObj.(*transaction.Transaction); ok {
			return n.prepareNormalTx(tx)
		}
	case rewardTx:
		if tx, ok := txObj.(*rewardTxData.RewardTx); ok {
			return n.prepareRewardTx(tx)
		}
	case unsignedTx:
		if tx, ok := txObj.(*smartContractResult.SmartContractResult); ok {
			return n.prepareUnsignedTx(tx)
		}
	}

	return &transaction.ApiTransactionResult{Type: string(invalidTx)}, nil // this shouldn't happen
}

func (n *Node) unmarshalTransaction(txBytes []byte, txType transactionType) (*transaction.ApiTransactionResult, error) {
	switch txType {
	case normalTx:
		var tx transaction.Transaction
		err := n.internalMarshalizer.Unmarshal(&tx, txBytes)
		if err != nil {
			return nil, err
		}
		return n.prepareNormalTx(&tx)
	case rewardTx:
		var tx rewardTxData.RewardTx
		err := n.internalMarshalizer.Unmarshal(&tx, txBytes)
		if err != nil {
			return nil, err
		}
		return n.prepareRewardTx(&tx)

	case unsignedTx:
		var tx smartContractResult.SmartContractResult
		err := n.internalMarshalizer.Unmarshal(&tx, txBytes)
		if err != nil {
			return nil, err
		}
		return n.prepareUnsignedTx(&tx)
	default:
		return &transaction.ApiTransactionResult{Type: string(invalidTx)}, nil // this shouldn't happen
	}
}

func (n *Node) prepareNormalTx(tx *transaction.Transaction) (*transaction.ApiTransactionResult, error) {
	return &transaction.ApiTransactionResult{
		Type:      string(normalTx),
		Nonce:     tx.Nonce,
		Value:     tx.Value.String(),
		Receiver:  n.addressPubkeyConverter.Encode(tx.RcvAddr),
		Sender:    n.addressPubkeyConverter.Encode(tx.SndAddr),
		GasPrice:  tx.GasPrice,
		GasLimit:  tx.GasLimit,
		Data:      string(tx.Data),
		Signature: hex.EncodeToString(tx.Signature),
	}, nil
}

func (n *Node) prepareRewardTx(tx *rewardTxData.RewardTx) (*transaction.ApiTransactionResult, error) {
	return &transaction.ApiTransactionResult{
		Type:     string(rewardTx),
		Round:    tx.GetRound(),
		Epoch:    tx.GetEpoch(),
		Value:    tx.GetValue().String(),
		Receiver: n.addressPubkeyConverter.Encode(tx.GetRcvAddr()),
	}, nil
}

func (n *Node) prepareUnsignedTx(tx *smartContractResult.SmartContractResult) (*transaction.ApiTransactionResult, error) {
	return &transaction.ApiTransactionResult{
		Type:      string(unsignedTx),
		Nonce:     tx.GetNonce(),
		Value:     tx.GetValue().String(),
		Receiver:  n.addressPubkeyConverter.Encode(tx.GetRcvAddr()),
		Sender:    n.addressPubkeyConverter.Encode(tx.GetSndAddr()),
		GasPrice:  tx.GetGasPrice(),
		GasLimit:  tx.GetGasLimit(),
		Data:      string(tx.GetData()),
		Code:      string(tx.GetCode()),
		Signature: "",
	}, nil
}
