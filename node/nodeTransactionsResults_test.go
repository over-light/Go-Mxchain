package node_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core/dblookupext"
	"github.com/ElrondNetwork/elrond-go/data/receipt"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/node"
	"github.com/ElrondNetwork/elrond-go/node/mock"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	dblookupext2 "github.com/ElrondNetwork/elrond-go/testscommon/dblookupext"
	"github.com/stretchr/testify/require"
)

func TestPutEventsInTransactionReceipt(t *testing.T) {
	t.Parallel()

	txHash := []byte("txHash")
	receiptHash := []byte("hash")
	rec := &receipt.Receipt{
		TxHash:  txHash,
		Data:    []byte("invalid tx"),
		Value:   big.NewInt(1000),
		SndAddr: []byte("sndAddr"),
	}

	marshalizerdMock := &mock.MarshalizerFake{}
	dataStore := &mock.ChainStorerMock{
		GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
			return &testscommon.StorerStub{
				GetFromEpochCalled: func(key []byte, epoch uint32) ([]byte, error) {
					recBytes, _ := json.Marshal(rec)
					return recBytes, nil
				},
			}
		},
	}
	historyRepo := &dblookupext2.HistoryRepositoryStub{
		GetEventsHashesByTxHashCalled: func(hash []byte, epoch uint32) (*dblookupext.ResultsHashesByTxHash, error) {
			return &dblookupext.ResultsHashesByTxHash{
				ReceiptsHash: receiptHash,
			}, nil
		},
	}

	coreComponents := getDefaultCoreComponents()
	coreComponents.IntMarsh = marshalizerdMock
	coreComponents.AddrPubKeyConv = &mock.PubkeyConverterMock{}

	dataComponents := getDefaultDataComponents()
	dataComponents.Store = dataStore

	processComponents := getDefaultProcessComponents()
	processComponents.HistoryRepositoryInternal = historyRepo

	n, _ := node.NewNode(
		node.WithCoreComponents(coreComponents),
		node.WithDataComponents(dataComponents),
		node.WithProcessComponents(processComponents),
	)

	epoch := uint32(0)

	tx := &transaction.ApiTransactionResult{}

	expectedRecAPI := &transaction.ReceiptApi{
		Value:   rec.Value,
		Data:    string(rec.Data),
		TxHash:  hex.EncodeToString(txHash),
		SndAddr: n.GetCoreComponents().AddressPubKeyConverter().Encode(rec.SndAddr),
	}

	n.PutResultsInTransaction(txHash, tx, epoch)
	require.Equal(t, expectedRecAPI, tx.Receipt)
}

func TestPutEventsInTransactionSmartContractResults(t *testing.T) {
	t.Parallel()

	epoch := uint32(0)
	txHash := []byte("txHash")
	scrHash1 := []byte("scrHash1")
	scrHash2 := []byte("scrHash2")

	scr1 := &smartContractResult.SmartContractResult{
		OriginalTxHash: txHash,
		RelayerAddr:    []byte("relayer"),
		OriginalSender: []byte("originalSender"),
		PrevTxHash:     []byte("prevTxHash"),
		SndAddr:        []byte("sender"),
		RcvAddr:        []byte("receiver"),
		Nonce:          1,
		Value:          big.NewInt(1000),
		GasLimit:       1,
		GasPrice:       5,
		Code:           []byte("code"),
		Data:           []byte("data"),
	}
	scr2 := &smartContractResult.SmartContractResult{
		OriginalTxHash: txHash,
	}

	marshalizerdMock := &mock.MarshalizerFake{}
	dataStore := &mock.ChainStorerMock{
		GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
			return &testscommon.StorerStub{
				GetFromEpochCalled: func(key []byte, epoch uint32) ([]byte, error) {
					switch {
					case bytes.Equal(key, scrHash1):
						return marshalizerdMock.Marshal(scr1)
					case bytes.Equal(key, scrHash2):
						return marshalizerdMock.Marshal(scr2)
					default:
						return nil, nil
					}
				},
			}
		},
	}
	historyRepo := &dblookupext2.HistoryRepositoryStub{
		GetEventsHashesByTxHashCalled: func(hash []byte, e uint32) (*dblookupext.ResultsHashesByTxHash, error) {
			return &dblookupext.ResultsHashesByTxHash{
				ReceiptsHash: nil,
				ScResultsHashesAndEpoch: []*dblookupext.ScResultsHashesAndEpoch{
					{
						Epoch:           epoch,
						ScResultsHashes: [][]byte{scrHash1, scrHash2},
					},
				},
			}, nil
		},
	}

	coreComponents := getDefaultCoreComponents()
	coreComponents.IntMarsh = marshalizerdMock
	coreComponents.AddrPubKeyConv = &mock.PubkeyConverterMock{}

	dataComponents := getDefaultDataComponents()
	dataComponents.Store = dataStore

	processComponents := getDefaultProcessComponents()
	processComponents.HistoryRepositoryInternal = historyRepo

	n, _ := node.NewNode(
		node.WithCoreComponents(coreComponents),
		node.WithDataComponents(dataComponents),
		node.WithProcessComponents(processComponents),
	)

	addressPubKeyConverter := n.GetCoreComponents().AddressPubKeyConverter()
	expectedSCRS := []*transaction.ApiSmartContractResult{
		{
			Hash:           hex.EncodeToString(scrHash1),
			Nonce:          scr1.Nonce,
			Value:          scr1.Value,
			RelayedValue:   scr1.RelayedValue,
			Code:           string(scr1.Code),
			Data:           string(scr1.Data),
			PrevTxHash:     hex.EncodeToString(scr1.PrevTxHash),
			OriginalTxHash: hex.EncodeToString(scr1.OriginalTxHash),
			GasLimit:       scr1.GasLimit,
			GasPrice:       scr1.GasPrice,
			CallType:       scr1.CallType,
			CodeMetadata:   string(scr1.CodeMetadata),
			ReturnMessage:  string(scr1.ReturnMessage),
			SndAddr:        addressPubKeyConverter.Encode(scr1.SndAddr),
			RcvAddr:        addressPubKeyConverter.Encode(scr1.RcvAddr),
			RelayerAddr:    addressPubKeyConverter.Encode(scr1.RelayerAddr),
			OriginalSender: addressPubKeyConverter.Encode(scr1.OriginalSender),
		},
		{
			Hash:           hex.EncodeToString(scrHash2),
			OriginalTxHash: hex.EncodeToString(scr1.OriginalTxHash),
		},
	}

	tx := &transaction.ApiTransactionResult{}
	n.PutResultsInTransaction(txHash, tx, epoch)
	require.Equal(t, expectedSCRS, tx.SmartContractResults)
}
