package mock

import (
	"bytes"
	"sort"
	"sync"

	"github.com/ElrondNetwork/elrond-go/core"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type headerInfo struct {
	hash   []byte
	header data.HeaderHandler
}

// BlockTrackerMock -
type BlockTrackerMock struct {
	AddTrackedHeaderCalled                            func(header data.HeaderHandler, hash []byte)
	AddCrossNotarizedHeaderCalled                     func(shardID uint32, crossNotarizedHeader data.HeaderHandler, crossNotarizedHeaderHash []byte)
	AddSelfNotarizedHeaderCalled                      func(shardID uint32, selfNotarizedHeader data.HeaderHandler, selfNotarizedHeaderHash []byte)
	CheckBlockAgainstFinalCalled                      func(headerHandler data.HeaderHandler) error
	CheckBlockAgainstRounderCalled                    func(headerHandler data.HeaderHandler) error
	CleanupHeadersBehindNonceCalled                   func(shardID uint32, selfNotarizedNonce uint64, crossNotarizedNonce uint64)
	ComputeLongestChainCalled                         func(shardID uint32, header data.HeaderHandler) ([]data.HeaderHandler, [][]byte)
	ComputeLongestMetaChainFromLastNotarizedCalled    func() ([]data.HeaderHandler, [][]byte, error)
	ComputeLongestShardsChainsFromLastNotarizedCalled func() ([]data.HeaderHandler, [][]byte, map[uint32][]data.HeaderHandler, error)
	DisplayTrackedHeadersCalled                       func()
	GetCrossNotarizedHeaderCalled                     func(shardID uint32, offset uint64) (data.HeaderHandler, []byte, error)
	GetFinalHeaderCalled                              func(shardID uint32) (data.HeaderHandler, []byte, error)
	GetLastCrossNotarizedHeaderCalled                 func(shardID uint32) (data.HeaderHandler, []byte, error)
	GetLastCrossNotarizedHeadersForAllShardsCalled    func() (map[uint32]data.HeaderHandler, error)
	GetTrackedHeadersCalled                           func(shardID uint32) ([]data.HeaderHandler, [][]byte)
	GetTrackedHeadersForAllShardsCalled               func() map[uint32][]data.HeaderHandler
	GetTrackedHeadersWithNonceCalled                  func(shardID uint32, nonce uint64) ([]data.HeaderHandler, [][]byte)
	IsShardStuckCalled                                func(shardId uint32) bool
	RegisterCrossNotarizedHeadersHandlerCalled        func(handler func(shardID uint32, headers []data.HeaderHandler, headersHashes [][]byte))
	RegisterSelfNotarizedHeadersHandlerCalled         func(handler func(shardID uint32, headers []data.HeaderHandler, headersHashes [][]byte))
	RemoveLastNotarizedHeadersCalled                  func()
	RestoreToGenesisCalled                            func()

	shardCoordinator         sharding.Coordinator
	mutCrossNotarizedHeaders sync.RWMutex
	crossNotarizedHeaders    map[uint32][]*headerInfo

	mutHeaders sync.RWMutex
	headers    map[uint32][]*headerInfo
}

// NewBlockTrackerMock -
func NewBlockTrackerMock(shardCoordinator sharding.Coordinator, startHeaders map[uint32]data.HeaderHandler) *BlockTrackerMock {
	bts := BlockTrackerMock{
		shardCoordinator: shardCoordinator,
	}
	bts.headers = make(map[uint32][]*headerInfo)
	_ = bts.InitCrossNotarizedHeaders(startHeaders)
	return &bts
}

// AddTrackedHeader -
func (btm *BlockTrackerMock) AddTrackedHeader(header data.HeaderHandler, hash []byte) {
	if btm.AddTrackedHeaderCalled != nil {
		btm.AddTrackedHeaderCalled(header, hash)
	}

	if check.IfNil(header) {
		return
	}

	shardID := header.GetShardID()

	btm.mutHeaders.Lock()
	defer btm.mutHeaders.Unlock()

	headersForShard, ok := btm.headers[shardID]
	if !ok {
		headersForShard = make([]*headerInfo, 0)
	}

	for _, headerInfo := range headersForShard {
		if bytes.Equal(headerInfo.hash, hash) {
			return
		}
	}

	headersForShard = append(headersForShard, &headerInfo{hash: hash, header: header})
	btm.headers[shardID] = headersForShard
}

// InitCrossNotarizedHeaders -
func (btm *BlockTrackerMock) InitCrossNotarizedHeaders(startHeaders map[uint32]data.HeaderHandler) error {
	btm.mutCrossNotarizedHeaders.Lock()
	defer btm.mutCrossNotarizedHeaders.Unlock()

	if startHeaders == nil {
		return process.ErrNotarizedHeadersSliceIsNil
	}

	btm.crossNotarizedHeaders = make(map[uint32][]*headerInfo)

	for _, startHeader := range startHeaders {
		shardID := startHeader.GetShardID()
		btm.crossNotarizedHeaders[shardID] = append(btm.crossNotarizedHeaders[shardID], &headerInfo{header: startHeader, hash: nil})
	}

	return nil
}

// AddCrossNotarizedHeader -
func (btm *BlockTrackerMock) AddCrossNotarizedHeader(shardID uint32, crossNotarizedHeader data.HeaderHandler, crossNotarizedHeaderHash []byte) {
	if btm.AddCrossNotarizedHeaderCalled != nil {
		btm.AddCrossNotarizedHeaderCalled(shardID, crossNotarizedHeader, crossNotarizedHeaderHash)
		return
	}

	if check.IfNil(crossNotarizedHeader) {
		return
	}

	btm.mutCrossNotarizedHeaders.Lock()
	btm.crossNotarizedHeaders[shardID] = append(btm.crossNotarizedHeaders[shardID], &headerInfo{header: crossNotarizedHeader, hash: crossNotarizedHeaderHash})
	if len(btm.crossNotarizedHeaders[shardID]) > 1 {
		sort.Slice(btm.crossNotarizedHeaders[shardID], func(i, j int) bool {
			return btm.crossNotarizedHeaders[shardID][i].header.GetNonce() < btm.crossNotarizedHeaders[shardID][j].header.GetNonce()
		})
	}
	btm.mutCrossNotarizedHeaders.Unlock()
}

// AddSelfNotarizedHeader -
func (btm *BlockTrackerMock) AddSelfNotarizedHeader(shardID uint32, selfNotarizedHeader data.HeaderHandler, selfNotarizedHeaderHash []byte) {
	if btm.AddSelfNotarizedHeaderCalled != nil {
		btm.AddSelfNotarizedHeaderCalled(shardID, selfNotarizedHeader, selfNotarizedHeaderHash)
	}
}

// CheckBlockAgainstRounder -
func (btm *BlockTrackerMock) CheckBlockAgainstRounder(headerHandler data.HeaderHandler) error {
	if btm.CheckBlockAgainstRounderCalled != nil {
		return btm.CheckBlockAgainstRounderCalled(headerHandler)
	}

	return nil
}

// CheckBlockAgainstFinal -
func (btm *BlockTrackerMock) CheckBlockAgainstFinal(headerHandler data.HeaderHandler) error {
	if btm.CheckBlockAgainstFinalCalled != nil {
		return btm.CheckBlockAgainstFinalCalled(headerHandler)
	}

	return nil
}

// CleanupHeadersBehindNonce -
func (btm *BlockTrackerMock) CleanupHeadersBehindNonce(shardID uint32, selfNotarizedNonce uint64, crossNotarizedNonce uint64) {
	if btm.CleanupHeadersBehindNonceCalled != nil {
		btm.CleanupHeadersBehindNonceCalled(shardID, selfNotarizedNonce, crossNotarizedNonce)
	}
}

func (btm *BlockTrackerMock) CleanupInvalidHeaders(header data.HeaderHandler) {

}

// ComputeLongestChain -
func (btm *BlockTrackerMock) ComputeLongestChain(shardID uint32, header data.HeaderHandler) ([]data.HeaderHandler, [][]byte) {
	if btm.ComputeLongestChainCalled != nil {
		return btm.ComputeLongestChainCalled(shardID, header)
	}

	headersInfo, ok := btm.headers[shardID]
	if !ok {
		return nil, nil
	}

	headers := make([]data.HeaderHandler, 0)
	hashes := make([][]byte, 0)

	for _, headerInfo := range headersInfo {
		headers = append(headers, headerInfo.header)
		hashes = append(hashes, headerInfo.hash)
	}

	return headers, hashes
}

// ComputeLongestMetaChainFromLastNotarized -
func (btm *BlockTrackerMock) ComputeLongestMetaChainFromLastNotarized() ([]data.HeaderHandler, [][]byte, error) {
	lastCrossNotarizedHeader, _, err := btm.GetLastCrossNotarizedHeader(core.MetachainShardId)
	if err != nil {
		return nil, nil, err
	}

	hdrsForShard, hdrsHashesForShard := btm.ComputeLongestChain(core.MetachainShardId, lastCrossNotarizedHeader)

	return hdrsForShard, hdrsHashesForShard, nil
}

// ComputeLongestShardsChainsFromLastNotarized -
func (btm *BlockTrackerMock) ComputeLongestShardsChainsFromLastNotarized() ([]data.HeaderHandler, [][]byte, map[uint32][]data.HeaderHandler, error) {
	hdrsMap := make(map[uint32][]data.HeaderHandler)
	hdrsHashesMap := make(map[uint32][][]byte)

	lastCrossNotarizedHeaders, err := btm.GetLastCrossNotarizedHeadersForAllShards()
	if err != nil {
		return nil, nil, nil, err
	}

	maxHdrLen := 0
	for shardID := uint32(0); shardID < btm.shardCoordinator.NumberOfShards(); shardID++ {
		hdrsForShard, hdrsHashesForShard := btm.ComputeLongestChain(shardID, lastCrossNotarizedHeaders[shardID])

		hdrsMap[shardID] = append(hdrsMap[shardID], hdrsForShard...)
		hdrsHashesMap[shardID] = append(hdrsHashesMap[shardID], hdrsHashesForShard...)

		tmpHdrLen := len(hdrsForShard)
		if maxHdrLen < tmpHdrLen {
			maxHdrLen = tmpHdrLen
		}
	}

	orderedHeaders := make([]data.HeaderHandler, 0)
	orderedHeadersHashes := make([][]byte, 0)

	// copy from map to lists - equality between number of headers per shard
	for i := 0; i < maxHdrLen; i++ {
		for shardID := uint32(0); shardID < btm.shardCoordinator.NumberOfShards(); shardID++ {
			hdrsForShard := hdrsMap[shardID]
			hdrsHashesForShard := hdrsHashesMap[shardID]
			if i >= len(hdrsForShard) {
				continue
			}

			orderedHeaders = append(orderedHeaders, hdrsForShard[i])
			orderedHeadersHashes = append(orderedHeadersHashes, hdrsHashesForShard[i])
		}
	}

	return orderedHeaders, orderedHeadersHashes, hdrsMap, nil
}

// DisplayTrackedHeaders -
func (btm *BlockTrackerMock) DisplayTrackedHeaders() {
	if btm.DisplayTrackedHeadersCalled != nil {
		btm.DisplayTrackedHeadersCalled()
	}
}

// GetCrossNotarizedHeader -
func (btm *BlockTrackerMock) GetCrossNotarizedHeader(shardID uint32, offset uint64) (data.HeaderHandler, []byte, error) {
	if btm.GetCrossNotarizedHeaderCalled != nil {
		return btm.GetCrossNotarizedHeaderCalled(shardID, offset)
	}

	return nil, nil, nil
}

// GetFinalHeader -
func (btm *BlockTrackerMock) GetFinalHeader(shardID uint32) (data.HeaderHandler, []byte, error) {
	if btm.GetFinalHeaderCalled != nil {
		return btm.GetFinalHeaderCalled(shardID)
	}

	return nil, nil, nil
}

// GetLastCrossNotarizedHeader -
func (btm *BlockTrackerMock) GetLastCrossNotarizedHeader(shardID uint32) (data.HeaderHandler, []byte, error) {
	if btm.GetLastCrossNotarizedHeaderCalled != nil {
		return btm.GetLastCrossNotarizedHeaderCalled(shardID)
	}

	btm.mutCrossNotarizedHeaders.RLock()
	defer btm.mutCrossNotarizedHeaders.RUnlock()

	if btm.crossNotarizedHeaders == nil {
		return nil, nil, process.ErrNotarizedHeadersSliceIsNil
	}

	headerInfo := btm.lastCrossNotarizedHdrForShard(shardID)
	if headerInfo == nil {
		return nil, nil, process.ErrNotarizedHeadersSliceForShardIsNil
	}

	return headerInfo.header, headerInfo.hash, nil
}

// GetLastCrossNotarizedHeadersForAllShards -
func (btm *BlockTrackerMock) GetLastCrossNotarizedHeadersForAllShards() (map[uint32]data.HeaderHandler, error) {
	lastCrossNotarizedHeaders := make(map[uint32]data.HeaderHandler, btm.shardCoordinator.NumberOfShards())

	// save last committed header for verification
	for shardID := uint32(0); shardID < btm.shardCoordinator.NumberOfShards(); shardID++ {
		lastCrossNotarizedHeader, _, err := btm.GetLastCrossNotarizedHeader(shardID)
		if err != nil {
			return nil, err
		}

		lastCrossNotarizedHeaders[shardID] = lastCrossNotarizedHeader
	}

	return lastCrossNotarizedHeaders, nil
}

func (btm *BlockTrackerMock) lastCrossNotarizedHdrForShard(shardID uint32) *headerInfo {
	crossNotarizedHeadersCount := len(btm.crossNotarizedHeaders[shardID])
	if crossNotarizedHeadersCount > 0 {
		return btm.crossNotarizedHeaders[shardID][crossNotarizedHeadersCount-1]
	}

	return nil
}

// GetTrackedHeaders -
func (btm *BlockTrackerMock) GetTrackedHeaders(shardID uint32) ([]data.HeaderHandler, [][]byte) {
	if btm.GetTrackedHeadersCalled != nil {
		return btm.GetTrackedHeadersCalled(shardID)
	}

	return nil, nil
}

// GetTrackedHeadersForAllShards -
func (btm *BlockTrackerMock) GetTrackedHeadersForAllShards() map[uint32][]data.HeaderHandler {
	trackedHeaders := make(map[uint32][]data.HeaderHandler)

	for shardID := uint32(0); shardID < btm.shardCoordinator.NumberOfShards(); shardID++ {
		trackedHeadersForShard, _ := btm.GetTrackedHeaders(shardID)
		trackedHeaders[shardID] = append(trackedHeaders[shardID], trackedHeadersForShard...)
	}

	return trackedHeaders
}

// GetTrackedHeadersWithNonce -
func (btm *BlockTrackerMock) GetTrackedHeadersWithNonce(shardID uint32, nonce uint64) ([]data.HeaderHandler, [][]byte) {
	if btm.GetTrackedHeadersWithNonceCalled != nil {
		return btm.GetTrackedHeadersWithNonceCalled(shardID, nonce)
	}

	return nil, nil
}

// IsShardStuck -
func (btm *BlockTrackerMock) IsShardStuck(shardId uint32) bool {
	if btm.IsShardStuckCalled != nil {
		return btm.IsShardStuckCalled(shardId)
	}

	return false
}

// RegisterCrossNotarizedHeadersHandler -
func (btm *BlockTrackerMock) RegisterCrossNotarizedHeadersHandler(handler func(shardID uint32, headers []data.HeaderHandler, headersHashes [][]byte)) {
	if btm.RegisterCrossNotarizedHeadersHandlerCalled != nil {
		btm.RegisterCrossNotarizedHeadersHandlerCalled(handler)
	}
}

// RegisterSelfNotarizedHeadersHandler -
func (btm *BlockTrackerMock) RegisterSelfNotarizedHeadersHandler(handler func(shardID uint32, headers []data.HeaderHandler, headersHashes [][]byte)) {
	if btm.RegisterSelfNotarizedHeadersHandlerCalled != nil {
		btm.RegisterSelfNotarizedHeadersHandlerCalled(handler)
	}
}

// RemoveLastNotarizedHeaders -
func (btm *BlockTrackerMock) RemoveLastNotarizedHeaders() {
	if btm.RemoveLastNotarizedHeadersCalled != nil {
		btm.RemoveLastNotarizedHeadersCalled()
	}
}

// RestoreToGenesis -
func (btm *BlockTrackerMock) RestoreToGenesis() {
	if btm.RestoreToGenesisCalled != nil {
		btm.RestoreToGenesisCalled()
	}
}

// IsInterfaceNil -
func (btm *BlockTrackerMock) IsInterfaceNil() bool {
	return btm == nil
}
