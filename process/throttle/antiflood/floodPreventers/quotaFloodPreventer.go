package floodPreventers

import (
	"fmt"
	"sync"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/storage"
)

const minMessages = 1
const minTotalSize = 1 //1Byte
const initNumMessages = 1

type quota struct {
	numReceivedMessages   uint32
	sizeReceivedMessages  uint64
	numProcessedMessages  uint32
	sizeProcessedMessages uint64
}

// quotaFloodPreventer represents a cache of quotas per peer used in antiflooding mechanism
type quotaFloodPreventer struct {
	mutOperation       *sync.RWMutex
	cacher             storage.Cacher
	statusHandlers     []QuotaStatusHandler
	maxMessagesPerPeer uint32
	maxSizePerPeer     uint64
	maxMessages        uint32
	maxSize            uint64
	globalQuota        *quota
}

// NewQuotaFloodPreventer creates a new flood preventer based on quota / peer
func NewQuotaFloodPreventer(
	cacher storage.Cacher,
	statusHandlers []QuotaStatusHandler,
	maxMessagesPerPeer uint32,
	maxTotalSizePerPeer uint64,
	maxMessages uint32,
	maxTotalSize uint64,
) (*quotaFloodPreventer, error) {

	if check.IfNil(cacher) {
		return nil, process.ErrNilCacher
	}
	for _, statusHandler := range statusHandlers {
		if check.IfNil(statusHandler) {
			return nil, process.ErrNilQuotaStatusHandler
		}
	}
	if maxMessagesPerPeer < minMessages {
		return nil, fmt.Errorf("%w raised in NewCountersMap, maxMessagesPerPeer: provided %d, minimum %d",
			process.ErrInvalidValue,
			maxMessagesPerPeer,
			minMessages,
		)
	}
	if maxTotalSizePerPeer < minTotalSize {
		return nil, fmt.Errorf("%w raised in NewCountersMap, maxTotalSizePerPeer: provided %d, minimum %d",
			process.ErrInvalidValue,
			maxTotalSize,
			minTotalSize,
		)
	}
	if maxMessages < minMessages {
		return nil, fmt.Errorf("%w raised in NewCountersMap, maxMessages: provided %d, minimum %d",
			process.ErrInvalidValue,
			maxMessagesPerPeer,
			minMessages,
		)
	}
	if maxTotalSize < minTotalSize {
		return nil, fmt.Errorf("%w raised in NewCountersMap, maxTotalSize: provided %d, minimum %d",
			process.ErrInvalidValue,
			maxTotalSize,
			minTotalSize,
		)
	}

	return &quotaFloodPreventer{
		mutOperation:       &sync.RWMutex{},
		cacher:             cacher,
		statusHandlers:     statusHandlers,
		maxMessagesPerPeer: maxMessagesPerPeer,
		maxSizePerPeer:     maxTotalSizePerPeer,
		maxMessages:        maxMessages,
		maxSize:            maxTotalSize,
		globalQuota:        &quota{},
	}, nil
}

// IncreaseLoadGlobal tries to increment the counter values held at "identifier" position
// It returns true if it had succeeded incrementing (existing counter value is lower or equal with provided maxOperations)
// We need the mutOperation here as the get and put should be done atomically.
// Otherwise we might yield a slightly higher number of false valid increments
// This method also checks the global sum quota and increment its values
func (qfp *quotaFloodPreventer) IncreaseLoadGlobal(identifier string, size uint64) error {
	qfp.mutOperation.Lock()

	qfp.globalQuota.numReceivedMessages++
	qfp.globalQuota.sizeReceivedMessages += size

	err := qfp.increaseLoad(identifier, size)
	if err == nil {
		qfp.globalQuota.numProcessedMessages++
		qfp.globalQuota.sizeProcessedMessages += size
	}
	qfp.mutOperation.Unlock()

	return err
}

// IncreaseLoad tries to increment the counter values held at "identifier" position
// It returns true if it had succeeded incrementing (existing counter value is lower or equal with provided maxOperations)
// We need the mutOperation here as the get and put should be done atomically.
// Otherwise we might yield a slightly higher number of false valid increments
// This method also checks the global sum quota but does not increment its values
func (qfp *quotaFloodPreventer) IncreaseLoad(identifier string, size uint64) error {
	qfp.mutOperation.Lock()
	defer qfp.mutOperation.Unlock()

	return qfp.increaseLoad(identifier, size)
}

func (qfp *quotaFloodPreventer) increaseLoad(identifier string, size uint64) error {
	isGlobalQuotaReached := qfp.globalQuota.numReceivedMessages > qfp.maxMessages ||
		qfp.globalQuota.sizeReceivedMessages > qfp.maxSize
	if isGlobalQuotaReached {
		return process.ErrSystemBusy
	}

	valueQuota, ok := qfp.cacher.Get([]byte(identifier))
	if !ok {
		qfp.putDefaultQuota(identifier, size)

		return nil
	}

	q, isQuota := valueQuota.(*quota)
	if !isQuota {
		qfp.putDefaultQuota(identifier, size)

		return nil
	}

	q.numReceivedMessages++
	q.sizeReceivedMessages += size

	isPeerQuotaReached := q.numReceivedMessages > qfp.maxMessagesPerPeer ||
		q.sizeReceivedMessages > qfp.maxSizePerPeer
	if isPeerQuotaReached {
		return process.ErrSystemBusy
	}

	q.numProcessedMessages++
	q.sizeProcessedMessages += size
	qfp.cacher.Put([]byte(identifier), q)

	return nil
}

func (qfp *quotaFloodPreventer) putDefaultQuota(identifier string, size uint64) {
	q := &quota{
		numReceivedMessages:   initNumMessages,
		sizeReceivedMessages:  size,
		numProcessedMessages:  initNumMessages,
		sizeProcessedMessages: size,
	}
	qfp.cacher.Put([]byte(identifier), q)
}

// Reset clears all map values
func (qfp *quotaFloodPreventer) Reset() {
	qfp.mutOperation.Lock()
	defer qfp.mutOperation.Unlock()

	qfp.resetStatusHandlers()
	qfp.createStatistics()

	//TODO change this if cacher.Clear() is time consuming
	qfp.cacher.Clear()
	qfp.globalQuota = &quota{}
}

func (qfp *quotaFloodPreventer) resetStatusHandlers() {
	for _, statusHandler := range qfp.statusHandlers {
		statusHandler.ResetStatistics()
	}
}

// createStatistics is useful to benchmark the system when running
func (qfp quotaFloodPreventer) createStatistics() {
	keys := qfp.cacher.Keys()
	for _, k := range keys {
		val, ok := qfp.cacher.Get(k)
		if !ok {
			continue
		}

		q, isQuota := val.(*quota)
		if !isQuota {
			continue
		}

		qfp.addQuota(
			string(k),
			q.numReceivedMessages,
			q.sizeReceivedMessages,
			q.numProcessedMessages,
			q.sizeProcessedMessages,
		)
	}

	qfp.setGlobalQuota(
		qfp.globalQuota.numReceivedMessages,
		qfp.globalQuota.sizeReceivedMessages,
		qfp.globalQuota.numProcessedMessages,
		qfp.globalQuota.sizeProcessedMessages,
	)
}

func (qfp *quotaFloodPreventer) addQuota(
	identifier string,
	numReceived uint32,
	sizeReceived uint64,
	numProcessed uint32,
	sizeProcessed uint64,
) {
	for _, statusHandler := range qfp.statusHandlers {
		statusHandler.AddQuota(identifier, numReceived, sizeReceived, numProcessed, sizeProcessed)
	}
}

func (qfp *quotaFloodPreventer) setGlobalQuota(
	numReceived uint32,
	sizeReceived uint64,
	numProcessed uint32,
	sizeProcessed uint64,
) {
	for _, statusHandler := range qfp.statusHandlers {
		statusHandler.SetGlobalQuota(numReceived, sizeReceived, numProcessed, sizeProcessed)
	}
}

// IsInterfaceNil returns true if there is no value under the interface
func (qfp *quotaFloodPreventer) IsInterfaceNil() bool {
	return qfp == nil
}
