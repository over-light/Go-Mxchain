package processedMb_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
	"github.com/ElrondNetwork/elrond-go/process/block/processedMb"
	"github.com/stretchr/testify/assert"
)

func TestProcessedMiniBlocks_AddMiniBlockHashShouldWork(t *testing.T) {
	t.Parallel()

	pmb := processedMb.NewProcessedMiniBlocks()

	mbHash1 := "hash1"
	mbHash2 := "hash2"
	mtbHash1 := "meta1"
	mtbHash2 := "meta2"

	pmb.AddMiniBlockHash(mtbHash1, mbHash1, nil)
	assert.True(t, pmb.IsMiniBlockFullyProcessed(mtbHash1, mbHash1))

	pmb.AddMiniBlockHash(mtbHash2, mbHash1, nil)
	assert.True(t, pmb.IsMiniBlockFullyProcessed(mtbHash2, mbHash1))

	pmb.AddMiniBlockHash(mtbHash1, mbHash2, nil)
	assert.True(t, pmb.IsMiniBlockFullyProcessed(mtbHash1, mbHash2))

	pmb.RemoveMiniBlockHash(mbHash1)
	assert.False(t, pmb.IsMiniBlockFullyProcessed(mtbHash1, mbHash1))

	pmb.RemoveMiniBlockHash(mbHash1)
	assert.False(t, pmb.IsMiniBlockFullyProcessed(mtbHash1, mbHash1))

	pmb.RemoveMetaBlockHash(mtbHash2)
	assert.False(t, pmb.IsMiniBlockFullyProcessed(mtbHash2, mbHash1))
}

func TestProcessedMiniBlocks_GetProcessedMiniBlocksHashes(t *testing.T) {
	t.Parallel()

	pmb := processedMb.NewProcessedMiniBlocks()

	mbHash1 := "hash1"
	mbHash2 := "hash2"
	mtbHash1 := "meta1"
	mtbHash2 := "meta2"

	pmb.AddMiniBlockHash(mtbHash1, mbHash1, nil)
	pmb.AddMiniBlockHash(mtbHash1, mbHash2, nil)
	pmb.AddMiniBlockHash(mtbHash2, mbHash2, nil)

	mapData := pmb.GetProcessedMiniBlocksHashes(mtbHash1)
	assert.NotNil(t, mapData[mbHash1])
	assert.NotNil(t, mapData[mbHash2])

	mapData = pmb.GetProcessedMiniBlocksHashes(mtbHash2)
	assert.NotNil(t, mapData[mbHash1])
}

func TestProcessedMiniBlocks_ConvertSliceToProcessedMiniBlocksMap(t *testing.T) {
	t.Parallel()

	pmb := processedMb.NewProcessedMiniBlocks()

	mbHash1 := "hash1"
	mtbHash1 := "meta1"

	data1 := bootstrapStorage.MiniBlocksInMeta{
		MetaHash:         []byte(mtbHash1),
		MiniBlocksHashes: [][]byte{[]byte(mbHash1)},
	}

	miniBlocksInMeta := []bootstrapStorage.MiniBlocksInMeta{data1}
	pmb.ConvertSliceToProcessedMiniBlocksMap(miniBlocksInMeta)
	assert.True(t, pmb.IsMiniBlockFullyProcessed(mtbHash1, mbHash1))

	convertedData := pmb.ConvertProcessedMiniBlocksMapToSlice()
	assert.Equal(t, miniBlocksInMeta, convertedData)
}
