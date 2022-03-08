package processedMb

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
)

var log = logger.GetOrCreate("process/processedMb")

// ProcessedMiniBlockInfo will keep the info about processed mini blocks
type ProcessedMiniBlockInfo struct {
	IsFullyProcessed       bool
	IndexOfLastTxProcessed int32
}

// MiniBlockHashes will keep a list of miniblock hashes as keys in a map for easy access
type MiniBlockHashes map[string]*ProcessedMiniBlockInfo

// ProcessedMiniBlockTracker is used to store all processed mini blocks hashes grouped by a metahash
type ProcessedMiniBlockTracker struct {
	processedMiniBlocks    map[string]MiniBlockHashes
	mutProcessedMiniBlocks sync.RWMutex
}

// NewProcessedMiniBlocks will create a complex type of processedMb
func NewProcessedMiniBlocks() *ProcessedMiniBlockTracker {
	return &ProcessedMiniBlockTracker{
		processedMiniBlocks: make(map[string]MiniBlockHashes),
	}
}

// AddMiniBlockHash will add a miniblock hash
func (pmb *ProcessedMiniBlockTracker) AddMiniBlockHash(metaBlockHash string, miniBlockHash string, processedMbInfo *ProcessedMiniBlockInfo) {
	pmb.mutProcessedMiniBlocks.Lock()
	defer pmb.mutProcessedMiniBlocks.Unlock()

	miniBlocksProcessed, ok := pmb.processedMiniBlocks[metaBlockHash]
	if !ok {
		miniBlocksProcessed = make(MiniBlockHashes)
		miniBlocksProcessed[miniBlockHash] = processedMbInfo
		pmb.processedMiniBlocks[metaBlockHash] = miniBlocksProcessed

		return
	}

	miniBlocksProcessed[miniBlockHash] = processedMbInfo
}

// RemoveMetaBlockHash will remove a meta block hash
func (pmb *ProcessedMiniBlockTracker) RemoveMetaBlockHash(metaBlockHash string) {
	pmb.mutProcessedMiniBlocks.Lock()
	delete(pmb.processedMiniBlocks, metaBlockHash)
	pmb.mutProcessedMiniBlocks.Unlock()
}

// RemoveMiniBlockHash will remove a mini block hash
func (pmb *ProcessedMiniBlockTracker) RemoveMiniBlockHash(miniBlockHash string) {
	pmb.mutProcessedMiniBlocks.Lock()
	for metaHash, miniBlocksProcessed := range pmb.processedMiniBlocks {
		delete(miniBlocksProcessed, miniBlockHash)

		if len(miniBlocksProcessed) == 0 {
			delete(pmb.processedMiniBlocks, metaHash)
		}
	}
	pmb.mutProcessedMiniBlocks.Unlock()
}

// GetProcessedMiniBlocksInfo will return all processed miniblocks info for a metablock
func (pmb *ProcessedMiniBlockTracker) GetProcessedMiniBlocksInfo(metaBlockHash string) map[string]*ProcessedMiniBlockInfo {
	pmb.mutProcessedMiniBlocks.RLock()
	processedMiniBlocksInfo := make(map[string]*ProcessedMiniBlockInfo)
	for hash, value := range pmb.processedMiniBlocks[metaBlockHash] {
		processedMiniBlocksInfo[hash] = value
	}
	pmb.mutProcessedMiniBlocks.RUnlock()

	return processedMiniBlocksInfo
}

// IsMiniBlockFullyProcessed will return true if a mini block is fully processed
func (pmb *ProcessedMiniBlockTracker) IsMiniBlockFullyProcessed(metaBlockHash string, miniBlockHash string) bool {
	pmb.mutProcessedMiniBlocks.RLock()
	defer pmb.mutProcessedMiniBlocks.RUnlock()

	miniBlocksProcessed, ok := pmb.processedMiniBlocks[metaBlockHash]
	if !ok {
		return false
	}

	processedMbInfo, hashExists := miniBlocksProcessed[miniBlockHash]
	if !hashExists {
		return false
	}

	return processedMbInfo.IsFullyProcessed
}

// ConvertProcessedMiniBlocksMapToSlice will convert a map[string]map[string]struct{} in a slice of MiniBlocksInMeta
func (pmb *ProcessedMiniBlockTracker) ConvertProcessedMiniBlocksMapToSlice() []bootstrapStorage.MiniBlocksInMeta {
	pmb.mutProcessedMiniBlocks.RLock()
	defer pmb.mutProcessedMiniBlocks.RUnlock()

	if len(pmb.processedMiniBlocks) == 0 {
		return nil
	}

	miniBlocksInMetaBlocks := make([]bootstrapStorage.MiniBlocksInMeta, 0, len(pmb.processedMiniBlocks))

	for metaHash, miniBlocksHashes := range pmb.processedMiniBlocks {
		miniBlocksInMeta := bootstrapStorage.MiniBlocksInMeta{
			MetaHash:         []byte(metaHash),
			MiniBlocksHashes: make([][]byte, 0, len(miniBlocksHashes)),
		}

		for miniBlockHash := range miniBlocksHashes {
			miniBlocksInMeta.MiniBlocksHashes = append(miniBlocksInMeta.MiniBlocksHashes, []byte(miniBlockHash))
		}

		miniBlocksInMetaBlocks = append(miniBlocksInMetaBlocks, miniBlocksInMeta)
	}

	return miniBlocksInMetaBlocks
}

// ConvertSliceToProcessedMiniBlocksMap will convert a slice of MiniBlocksInMeta in an map[string]MiniBlockHashes
func (pmb *ProcessedMiniBlockTracker) ConvertSliceToProcessedMiniBlocksMap(miniBlocksInMetaBlocks []bootstrapStorage.MiniBlocksInMeta) {
	pmb.mutProcessedMiniBlocks.Lock()
	defer pmb.mutProcessedMiniBlocks.Unlock()

	for _, miniBlocksInMeta := range miniBlocksInMetaBlocks {
		miniBlocksHashes := make(MiniBlockHashes)
		for index, miniBlockHash := range miniBlocksInMeta.MiniBlocksHashes {
			miniBlocksHashes[string(miniBlockHash)] = &ProcessedMiniBlockInfo{
				IsFullyProcessed:       miniBlocksInMeta.IsFullyProcessed[index],
				IndexOfLastTxProcessed: miniBlocksInMeta.IndexOfLastTxProcessed[index],
			}
		}
		pmb.processedMiniBlocks[string(miniBlocksInMeta.MetaHash)] = miniBlocksHashes
	}
}

// DisplayProcessedMiniBlocks will display all miniblocks hashes and meta block hash from the map
func (pmb *ProcessedMiniBlockTracker) DisplayProcessedMiniBlocks() {
	log.Debug("processed mini blocks applied")

	pmb.mutProcessedMiniBlocks.RLock()
	for metaBlockHash, miniBlocksHashes := range pmb.processedMiniBlocks {
		log.Debug("processed",
			"meta hash", []byte(metaBlockHash))
		for miniBlockHash := range miniBlocksHashes {
			log.Debug("processed",
				"mini block hash", []byte(miniBlockHash))
		}
	}
	pmb.mutProcessedMiniBlocks.RUnlock()
}
