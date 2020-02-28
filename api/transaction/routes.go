package transaction

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"

	"github.com/ElrondNetwork/elrond-go/api/errors"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/gin-gonic/gin"
)

// TxService interface defines methods that can be used from `elrondFacade` context variable
type TxService interface {
	CreateTransaction(nonce uint64, value string, receiverHex string, senderHex string, gasPrice uint64, gasLimit uint64, data []byte, signatureHex string) (*transaction.Transaction, error)
	SendTransaction(nonce uint64, sender string, receiver string, value string, gasPrice uint64, gasLimit uint64, txData []byte, signature []byte) (string, error)
	SendBulkTransactions([]*transaction.Transaction) (uint64, error)
	GetTransaction(hash string) (*transaction.Transaction, error)
	ComputeTransactionCost(tx *transaction.Transaction) (*big.Int, error)
	IsInterfaceNil() bool
}

// TxRequest represents the structure on which user input for generating a new transaction will validate against
type TxRequest struct {
	Sender   string   `form:"sender" json:"sender"`
	Receiver string   `form:"receiver" json:"receiver"`
	Value    *big.Int `form:"value" json:"value"`
	Data     string   `form:"data" json:"data"`
}

// MultipleTxRequest represents the structure on which user input for generating a bulk of transactions will validate against
type MultipleTxRequest struct {
	Receiver string   `form:"receiver" json:"receiver"`
	Value    *big.Int `form:"value" json:"value"`
	TxCount  int      `form:"txCount" json:"txCount"`
}

// SendTxRequest represents the structure that maps and validates user input for publishing a new transaction
type SendTxRequest struct {
	Sender    string `form:"sender" json:"sender"`
	Receiver  string `form:"receiver" json:"receiver"`
	Value     string `form:"value" json:"value"`
	Data      []byte `form:"data" json:"data"`
	Nonce     uint64 `form:"nonce" json:"nonce"`
	GasPrice  uint64 `form:"gasPrice" json:"gasPrice"`
	GasLimit  uint64 `form:"gasLimit" json:"gasLimit"`
	Signature string `form:"signature" json:"signature"`
}

//TxResponse represents the structure on which the response will be validated against
type TxResponse struct {
	SendTxRequest
	ShardID     uint32 `json:"shardId"`
	Hash        string `json:"hash"`
	BlockNumber uint64 `json:"blockNumber"`
	BlockHash   string `json:"blockHash"`
	Timestamp   uint64 `json:"timestamp"`
}

// Routes defines transaction related routes
func Routes(router *gin.RouterGroup) {
	router.POST("/send", SendTransaction)
	router.POST("/cost", GetTransactionCost)
	router.POST("/send-multiple", SendMultipleTransactions)
	router.GET("/:txhash", GetTransaction)
}

// SendTransaction will receive a transaction from the client and propagate it for processing
func SendTransaction(c *gin.Context) {
	ef, ok := c.MustGet("elrondFacade").(TxService)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInvalidAppContext.Error()})
		return
	}

	var gtx = SendTxRequest{}
	err := c.ShouldBindJSON(&gtx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), err.Error())})
		return
	}

	signature, err := hex.DecodeString(gtx.Signature)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s: %s", errors.ErrInvalidSignatureHex.Error(), err.Error())})
		return
	}

	txHash, err := ef.SendTransaction(gtx.Nonce, gtx.Sender, gtx.Receiver, gtx.Value, gtx.GasPrice, gtx.GasLimit, gtx.Data, signature)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s: %s", errors.ErrTxGenerationFailed.Error(), err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"txHash": txHash})
}

// SendMultipleTransactions will receive a number of transactions and will propagate them for processing
func SendMultipleTransactions(c *gin.Context) {
	ef, ok := c.MustGet("elrondFacade").(TxService)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInvalidAppContext.Error()})
		return
	}

	var gtx []SendTxRequest
	err := c.ShouldBindJSON(&gtx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), err.Error())})
		return
	}

	var txs []*transaction.Transaction
	var tx *transaction.Transaction
	for _, receivedTx := range gtx {
		tx, err = ef.CreateTransaction(
			receivedTx.Nonce,
			receivedTx.Value,
			receivedTx.Receiver,
			receivedTx.Sender,
			receivedTx.GasPrice,
			receivedTx.GasLimit,
			receivedTx.Data,
			receivedTx.Signature,
		)
		if err != nil {
			continue
		}

		txs = append(txs, tx)
	}

	numOfSentTxs, err := ef.SendBulkTransactions(txs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"txsSent": numOfSentTxs})
}

// GetTransaction returns transaction details for a given txhash
func GetTransaction(c *gin.Context) {

	ef, ok := c.MustGet("elrondFacade").(TxService)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInvalidAppContext.Error()})
		return
	}

	txhash := c.Param("txhash")
	if txhash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrValidationEmptyTxHash.Error())})
		return
	}

	tx, err := ef.GetTransaction(txhash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrGetTransaction.Error()})
		return
	}

	if tx == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errors.ErrTxNotFound.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transaction": txResponseFromTransaction(tx)})
}

func txResponseFromTransaction(tx *transaction.Transaction) TxResponse {
	response := TxResponse{}
	response.Nonce = tx.Nonce
	response.Sender = hex.EncodeToString(tx.SndAddr)
	response.Receiver = hex.EncodeToString(tx.RcvAddr)
	response.Data = tx.Data
	response.Signature = hex.EncodeToString(tx.Signature)
	response.Value = tx.Value.String()
	response.GasLimit = tx.GasLimit
	response.GasPrice = tx.GasPrice

	return response
}

// GetTransactionCost returns how many gas a transaction wil consume
func GetTransactionCost(c *gin.Context) {
	ef, ok := c.MustGet("elrondFacade").(TxService)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInvalidAppContext.Error()})
		return
	}
	var gtx SendTxRequest
	err := c.ShouldBindJSON(&gtx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), err.Error())})
		return
	}

	tx, err := ef.CreateTransaction(
		gtx.Nonce,
		gtx.Value,
		gtx.Receiver,
		gtx.Sender,
		gtx.GasPrice,
		gtx.GasLimit,
		gtx.Data,
		gtx.Signature,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), err.Error())})
		return
	}

	cost, err := ef.ComputeTransactionCost(tx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"txGasUnits": cost.String()})
}
