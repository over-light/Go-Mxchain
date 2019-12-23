package track

import (
	"github.com/ElrondNetwork/elrond-go/data"
)

// RegisterSelfNotarizedHeadersHandler registers a new handler to be called when self notarized header is changed
func (bbt *baseBlockTrack) RegisterSelfNotarizedHeadersHandler(handler func(shardID uint32, headers []data.HeaderHandler, headersHashes [][]byte)) {
	if handler == nil {
		log.Debug("attempt to register a nil handler to a tracker object")
		return
	}

	bbt.mutSelfNotarizedHeadersHandlers.Lock()
	bbt.selfNotarizedHeadersHandlers = append(bbt.selfNotarizedHeadersHandlers, handler)
	bbt.mutSelfNotarizedHeadersHandlers.Unlock()
}

func (bbt *baseBlockTrack) callSelfNotarizedHeadersHandlers(shardID uint32, headers []data.HeaderHandler, headersHashes [][]byte) {
	bbt.mutSelfNotarizedHeadersHandlers.RLock()
	for _, handler := range bbt.selfNotarizedHeadersHandlers {
		go handler(shardID, headers, headersHashes)
	}
	bbt.mutSelfNotarizedHeadersHandlers.RUnlock()
}

// RegisterCrossNotarizedHeadersHandler registers a new handler to be called when cross notarized header is changed
func (bbt *baseBlockTrack) RegisterCrossNotarizedHeadersHandler(handler func(shardID uint32, headers []data.HeaderHandler, headersHashes [][]byte)) {
	if handler == nil {
		log.Debug("attempt to register a nil handler to a tracker object")
		return
	}

	bbt.mutCrossNotarizedHeadersHandlers.Lock()
	bbt.crossNotarizedHeadersHandlers = append(bbt.crossNotarizedHeadersHandlers, handler)
	bbt.mutCrossNotarizedHeadersHandlers.Unlock()
}

func (bbt *baseBlockTrack) callCrossNotarizedHeadersHandlers(shardID uint32, headers []data.HeaderHandler, headersHashes [][]byte) {
	bbt.mutCrossNotarizedHeadersHandlers.RLock()
	for _, handler := range bbt.crossNotarizedHeadersHandlers {
		go handler(shardID, headers, headersHashes)
	}
	bbt.mutCrossNotarizedHeadersHandlers.RUnlock()
}
