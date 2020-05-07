package indexer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/mock"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/marshal"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/require"
)

func createCommonProcessor() commonProcessor {
	return commonProcessor{
		addressPubkeyConverter:   mock.NewPubkeyConverterMock(32),
		validatorPubkeyConverter: mock.NewPubkeyConverterMock(32),
	}
}

func TestGetTransactionByType_SC(t *testing.T) {
	t.Parallel()

	cp := createCommonProcessor()

	nonce := uint64(10)
	txHash := []byte("txHash")
	code := []byte("code")
	sndAddr, rcvAddr := []byte("snd"), []byte("rec")
	smartContractRes := &smartContractResult.SmartContractResult{
		Nonce:      nonce,
		PrevTxHash: txHash,
		Code:       code,
		Data:       []byte(""),
		SndAddr:    sndAddr,
		RcvAddr:    rcvAddr,
		CallType:   vmcommon.CallType(0),
	}

	scRes := cp.convertScResultInDatabaseScr(smartContractRes)
	expectedTx := ScResult{
		Nonce:     nonce,
		PreTxHash: hex.EncodeToString(txHash),
		Code:      string(code),
		Data:      "",
		Sender:    cp.addressPubkeyConverter.Encode(sndAddr),
		Receiver:  cp.addressPubkeyConverter.Encode(rcvAddr),
		Value:     "<nil>",
		CallType:  "\x00",
	}

	require.Equal(t, expectedTx, scRes)
}

func TestGetTransactionByType_RewardTx(t *testing.T) {
	t.Parallel()

	cp := createCommonProcessor()

	round := uint64(10)
	rcvAddr := []byte("receiver")
	rwdTx := &rewardTx.RewardTx{Round: round, RcvAddr: rcvAddr}
	txHash := []byte("txHash")
	mbHash := []byte("mbHash")
	mb := &block.MiniBlock{TxHashes: [][]byte{txHash}}
	header := &block.Header{Nonce: 2}
	status := "Success"

	resultTx := cp.buildRewardTransaction(rwdTx, txHash, mbHash, mb, header, status)
	expectedTx := &Transaction{
		Hash:     hex.EncodeToString(txHash),
		MBHash:   hex.EncodeToString(mbHash),
		Round:    round,
		Receiver: hex.EncodeToString(rcvAddr),
		Status:   status,
		Value:    "<nil>",
		Sender:   fmt.Sprintf("%d", core.MetachainShardId),
		Data:     "",
	}

	require.Equal(t, expectedTx, resultTx)
}

func TestPrepareBufferMiniblocks(t *testing.T) {
	var buff bytes.Buffer

	meta := []byte("test1")
	serializedData := []byte("test2")

	buff = prepareBufferMiniblocks(buff, meta, serializedData)

	var expectedBuff bytes.Buffer
	serializedData = append(serializedData, "\n"...)
	expectedBuff.Grow(len(meta) + len(serializedData))
	_, _ = expectedBuff.Write(meta)
	_, _ = expectedBuff.Write(serializedData)

	require.Equal(t, expectedBuff, buff)
}

func generateTxs(numTxs int) map[string]data.TransactionHandler {
	txs := make(map[string]data.TransactionHandler, numTxs)
	for i := 0; i < numTxs; i++ {
		tx := &transaction.Transaction{
			Nonce:     uint64(i),
			Value:     big.NewInt(int64(i)),
			RcvAddr:   []byte("443e79a8d99ba093262c1db48c58ab3d59bcfeb313ca5cddf2a9d1d06f9894ec"),
			SndAddr:   []byte("443e79a8d99ba093262c1db48c58ab3d59bcfeb313ca5cddf2a9d1d06f9894ec"),
			GasPrice:  10000000,
			GasLimit:  1000,
			Data:      []byte("dasjdksakjdksajdjksajkdjkasjdksajkdasjdksakjdksajdjksajkdjkasjdksajkdasjdksakjdksajdjksajkdjkasjdksajk"),
			Signature: []byte("randomSignatureasdasldkasdsahjgdlhjaskldsjkaldjklasjkdjskladjkl;sajkl"),
		}
		txs[fmt.Sprintf("%d", i)] = tx
	}

	return txs
}

func TestComputeSizeOfTxsDuration(t *testing.T) {
	res := testing.Benchmark(benchmarkComputeSizeOfTxsDuration)

	fmt.Println("Time to calculate size of txs :", time.Duration(res.NsPerOp()))
}

func benchmarkComputeSizeOfTxsDuration(b *testing.B) {
	numTxs := 20000
	txs := generateTxs(numTxs)
	gogoMarsh := &marshal.GogoProtoMarshalizer{}

	for i := 0; i < b.N; i++ {
		computeSizeOfTxs(gogoMarsh, txs)
	}
}

func TestComputeSizeOfTxs(t *testing.T) {
	const kb = 1024
	numTxs := 20000

	txs := generateTxs(numTxs)
	gogoMarsh := &marshal.GogoProtoMarshalizer{}
	lenTxs := computeSizeOfTxs(gogoMarsh, txs)

	keys := reflect.ValueOf(txs).MapKeys()
	oneTxBytes, _ := gogoMarsh.Marshal(txs[keys[0].String()])
	oneTxSize := len(oneTxBytes)
	expectedSize := numTxs * oneTxSize
	expectedSizeDeltaPlus := expectedSize + int(0.01*float64(expectedSize))
	expectedSizeDeltaMinus := expectedSize - int(0.01*float64(expectedSize))

	require.Greater(t, lenTxs, expectedSizeDeltaMinus)
	require.Less(t, lenTxs, expectedSizeDeltaPlus)
	fmt.Printf("Size of %d transactions : %d Kbs \n", numTxs, lenTxs/kb)
}

func TestInterpretAsString(t *testing.T) {
	t.Parallel()

	data1 := []byte("@75736572206572726f72@b099086f9bddfcb0a4f45bada01b528f0d1981d7e20344523a7e41a7d8e9c7a6")
	expectedData1 := "@user error@b099086f9bddfcb0a4f45bada01b528f0d1981d7e20344523a7e41a7d8e9c7a6"

	decodedData := decodeScResultData(data1)
	require.Equal(t, expectedData1, decodedData)

	data2 := append([]byte("@75736572206572726f72@"), 150, 160)
	expectedData2 := "@user error"

	decodedData = decodeScResultData(data2)
	require.Equal(t, expectedData2, decodedData)
}
