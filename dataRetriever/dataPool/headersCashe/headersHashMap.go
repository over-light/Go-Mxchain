package headersCashe

import "sync"

type headersHashMap struct {
	hdrsHash    map[string]headerInfo
	mutHdrsHash sync.RWMutex
}

func newHeadersHashMap() *headersHashMap {
	return &headersHashMap{
		hdrsHash:    make(map[string]headerInfo),
		mutHdrsHash: sync.RWMutex{},
	}
}

func (hh *headersHashMap) addElement(hash []byte, info headerInfo) {
	hh.mutHdrsHash.Lock()
	defer hh.mutHdrsHash.Unlock()

	hh.hdrsHash[string(hash)] = info
}

func (hh *headersHashMap) deleteElement(hash []byte) {
	hh.mutHdrsHash.Lock()
	defer hh.mutHdrsHash.Unlock()

	delete(hh.hdrsHash, string(hash))
}

func (hh *headersHashMap) getElement(hash []byte) (headerInfo, bool) {
	hh.mutHdrsHash.RLock()
	defer hh.mutHdrsHash.RUnlock()

	if _, ok := hh.hdrsHash[string(hash)]; !ok {
		return headerInfo{}, false
	}

	return hh.hdrsHash[string(hash)], true
}
