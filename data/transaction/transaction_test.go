package transaction_test

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/stretchr/testify/assert"
)

func TestTransaction_SaveLoad(t *testing.T) {
	tx := transaction.Transaction{
		Nonce:     uint64(1),
		Value:     big.NewInt(1),
		RcvAddr:   []byte("receiver_address"),
		SndAddr:   []byte("sender_address"),
		GasPrice:  uint64(10000),
		GasLimit:  uint64(1000),
		Data:      "tx_data",
		Signature: []byte("signature"),
	}

	var b bytes.Buffer
	_ = tx.Save(&b)

	loadTx := transaction.Transaction{}
	_ = loadTx.Load(&b)

	assert.Equal(t, loadTx, tx)
}

func TestTransaction_GetData(t *testing.T) {
	t.Parallel()

	data := "data"
	tx := &transaction.Transaction{Data: data}

	assert.Equal(t, data, tx.Data)
}

func TestTransaction_GetRecvAddr(t *testing.T) {
	t.Parallel()

	data := []byte("data")
	tx := &transaction.Transaction{RcvAddr: data}

	assert.Equal(t, data, tx.RcvAddr)
}

func TestTransaction_GetSndAddr(t *testing.T) {
	t.Parallel()

	data := []byte("data")
	tx := &transaction.Transaction{SndAddr: data}

	assert.Equal(t, data, tx.SndAddr)
}

func TestTransaction_GetValue(t *testing.T) {
	t.Parallel()

	value := big.NewInt(10)
	tx := &transaction.Transaction{Value: value}

	assert.Equal(t, value, tx.Value)
}

func TestTransaction_SetData(t *testing.T) {
	t.Parallel()

	data := "data"
	tx := &transaction.Transaction{}
	tx.SetData(data)

	assert.Equal(t, data, tx.Data)
}

func TestTransaction_SetRecvAddr(t *testing.T) {
	t.Parallel()

	txData := []byte("data")
	tx := &transaction.Transaction{}
	tx.SetRcvAddr(txData)

	assert.Equal(t, txData, tx.RcvAddr)
}

func TestTransaction_SetSndAddr(t *testing.T) {
	t.Parallel()

	txData := []byte("data")
	tx := &transaction.Transaction{}
	tx.SetSndAddr(txData)

	assert.Equal(t, txData, tx.SndAddr)
}

func TestTransaction_SetValue(t *testing.T) {
	t.Parallel()

	value := big.NewInt(10)
	tx := &transaction.Transaction{}
	tx.SetValue(value)

	assert.Equal(t, value, tx.Value)
}

func TestTransaction_MarshalUnmarshalJsonShouldWork(t *testing.T) {
	t.Parallel()

	value := big.NewInt(445566)
	tx := &transaction.Transaction{
		Nonce:     112233,
		Value:     new(big.Int).Set(value),
		RcvAddr:   []byte("receiver"),
		SndAddr:   []byte("sender"),
		GasPrice:  1234,
		GasLimit:  5678,
		Data:      "data",
		Signature: []byte("signature"),
	}

	buff, err := json.Marshal(tx)
	assert.Nil(t, err)
	txRecovered := &transaction.Transaction{}
	err = json.Unmarshal(buff, txRecovered)
	assert.Nil(t, err)
	assert.Equal(t, tx, txRecovered)

	buffAsString := string(buff)
	assert.Contains(t, buffAsString, "\""+value.String()+"\"")
}
