package preprocess

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionPreprocessor_SplitMiniBlockBasedOnTxTypeIfNeededShouldWork(t *testing.T) {
	t.Parallel()

	mb := block.MiniBlock{
		TxHashes: make([][]byte, 0),
	}

	mapSCTxs := make(map[string]struct{})
	txHash1 := []byte("txHash1")
	txHash2 := []byte("txHash2")
	mb.TxHashes = append(mb.TxHashes, txHash1)
	mb.TxHashes = append(mb.TxHashes, txHash2)
	mapSCTxs[string(txHash1)] = struct{}{}

	mbs := splitMiniBlockBasedOnTxTypeIfNeeded(&mb, mapSCTxs)
	require.Equal(t, 2, len(mbs))
	require.Equal(t, 1, len(mbs[0].TxHashes))
	require.Equal(t, 1, len(mbs[1].TxHashes))
	assert.Equal(t, txHash2, mbs[0].TxHashes[0])
	assert.Equal(t, txHash1, mbs[1].TxHashes[0])
}

//TODO: Add more unit tests for transactionV2.go