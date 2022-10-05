package state

import (
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go/common"
	"github.com/ElrondNetwork/elrond-go/trie/statistics"
)

const numTriesToPrint = 10

type snapshotStatistics struct {
	trieStatisticsCollector common.TriesStatisticsCollector

	startTime time.Time

	wgSnapshot *sync.WaitGroup
	wgSync     *sync.WaitGroup
	mutex      sync.RWMutex
}

func newSnapshotStatistics(snapshotDelta int, syncDelta int) *snapshotStatistics {
	wgSnapshot := &sync.WaitGroup{}
	wgSnapshot.Add(snapshotDelta)

	wgSync := &sync.WaitGroup{}
	wgSync.Add(syncDelta)
	return &snapshotStatistics{
		wgSnapshot:              wgSnapshot,
		wgSync:                  wgSync,
		startTime:               time.Now(),
		trieStatisticsCollector: statistics.NewTrieStatisticsCollector(numTriesToPrint),
	}
}

// SnapshotFinished marks the ending of a snapshot goroutine
func (ss *snapshotStatistics) SnapshotFinished() {
	ss.wgSnapshot.Done()
}

// NewSnapshotStarted marks the starting of a new snapshot goroutine
func (ss *snapshotStatistics) NewSnapshotStarted() {
	ss.wgSnapshot.Add(1)
}

// WaitForSnapshotsToFinish will wait until the waitGroup counter is zero
func (ss *snapshotStatistics) WaitForSnapshotsToFinish() {
	ss.wgSnapshot.Wait()
}

// AddTrieStats adds the given trie stats to the snapshot statistics
func (ss *snapshotStatistics) AddTrieStats(trieStats *statistics.TrieStatsDTO) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	ss.trieStatisticsCollector.Add(trieStats)
}

// WaitForSyncToFinish will wait until the waitGroup counter is zero
func (ss *snapshotStatistics) WaitForSyncToFinish() {
	ss.wgSync.Wait()
}

// SyncFinished marks the end of the sync process
func (ss *snapshotStatistics) SyncFinished() {
	ss.wgSync.Done()
}

// PrintStats will print the stats after the snapshot has finished
func (ss *snapshotStatistics) PrintStats(identifier string, rootHash []byte) {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	log.Debug("snapshot statistics",
		"type", identifier,
		"duration", time.Since(ss.startTime).Truncate(time.Second),
		"rootHash", rootHash,
	)
	ss.trieStatisticsCollector.Print()
}
