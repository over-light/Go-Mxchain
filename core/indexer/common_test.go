package indexer

import (
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core/mock"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/receipt"
	"github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTransactionByType_SC(t *testing.T) {
	t.Parallel()

	cp := commonProcessor{
		pubkeyConverter: mock.NewPubkeyConverterMock(32),
	}

	nonce := uint64(10)
	smartContract := &smartContractResult.SmartContractResult{Nonce: nonce}
	txHash := []byte("txHash")
	mbHash := []byte("mbHash")
	blockHash := []byte("blockHash")
	mb := &block.MiniBlock{TxHashes: [][]byte{txHash}}
	header := &block.Header{Nonce: 2}

	resultTx, err := cp.getTransactionByType(smartContract, txHash, mbHash, blockHash, mb, header, "")
	expectedTx := &Transaction{
		Hash:      hex.EncodeToString(txHash),
		MBHash:    hex.EncodeToString(mbHash),
		BlockHash: hex.EncodeToString(blockHash),
		Nonce:     nonce,
		Value:     "<nil>",
		Status:    "Success",
	}

	require.Equal(t, expectedTx, resultTx)
	assert.Nil(t, err)
}

func TestGetTransactionByType_RewardTx(t *testing.T) {
	t.Parallel()

	cp := commonProcessor{
		pubkeyConverter: mock.NewPubkeyConverterMock(32),
	}

	round := uint64(10)
	rcvAddr := []byte("receiver")
	rwdTx := &rewardTx.RewardTx{Round: round, RcvAddr: rcvAddr}
	txHash := []byte("txHash")
	mbHash := []byte("mbHash")
	blockHash := []byte("blockHash")
	mb := &block.MiniBlock{TxHashes: [][]byte{txHash}}
	header := &block.Header{Nonce: 2}

	resultTx, err := cp.getTransactionByType(rwdTx, txHash, mbHash, blockHash, mb, header, "")
	expectedTx := &Transaction{
		Hash:      hex.EncodeToString(txHash),
		MBHash:    hex.EncodeToString(mbHash),
		BlockHash: hex.EncodeToString(blockHash),
		Round:     round,
		Receiver:  hex.EncodeToString(rcvAddr),
		Status:    "Success",
		Value:     "<nil>",
		Sender:    metachainTpsDocID,
		Data:      []byte(""),
	}

	require.Equal(t, expectedTx, resultTx)
	assert.Nil(t, err)
}

func TestGetTransactionByType_Receipt(t *testing.T) {
	t.Parallel()

	cp := commonProcessor{
		pubkeyConverter: mock.NewPubkeyConverterMock(32),
	}

	receiptTest := &receipt.Receipt{Value: big.NewInt(100)}
	txHash := []byte("txHash")
	mbHash := []byte("mbHash")
	blockHash := []byte("blockHash")
	mb := &block.MiniBlock{TxHashes: [][]byte{txHash}}
	header := &block.Header{Nonce: 2}

	resultTx, err := cp.getTransactionByType(receiptTest, txHash, mbHash, blockHash, mb, header, "")
	expectedTx := &Transaction{
		Hash:      hex.EncodeToString(txHash),
		MBHash:    hex.EncodeToString(mbHash),
		BlockHash: hex.EncodeToString(blockHash),
		Value:     receiptTest.Value.String(),
		Status:    "Success",
	}

	require.Equal(t, expectedTx, resultTx)
	assert.Nil(t, err)
}

func TestGetTransactionByType_Nil(t *testing.T) {
	t.Parallel()

	cp := commonProcessor{
		pubkeyConverter: mock.NewPubkeyConverterMock(32),
	}

	txHash := []byte("txHash")
	mbHash := []byte("mbHash")
	blockHash := []byte("blockHash")
	mb := &block.MiniBlock{TxHashes: [][]byte{txHash}}
	header := &block.Header{Nonce: 2}

	resultTx, err := cp.getTransactionByType(nil, txHash, mbHash, blockHash, mb, header, "")

	require.Nil(t, resultTx)
	assert.True(t, errors.Is(err, ErrUnknownTransactionHandler))
}
