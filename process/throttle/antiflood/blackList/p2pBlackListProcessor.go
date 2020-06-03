package blackList

import (
	"fmt"
	"time"

	"github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/storage"
)

var log = logger.GetOrCreate("process/throttle/antiflood/blacklist")

const minBanDuration = time.Second
const minFloodingRounds = 2
const sizeBlacklistInfo = 4

type p2pBlackListProcessor struct {
	thresholdNumReceivedFlood  uint32
	numFloodingRounds          uint32
	thresholdSizeReceivedFlood uint64
	cacher                     storage.Cacher
	peerBlacklistHandler       process.PeerBlackListHandler
	banDuration                time.Duration
}

// NewP2PBlackListProcessor creates a new instance of p2pQuotaBlacklistProcessor able to determine
// a flooding peer and mark it accordingly
func NewP2PBlackListProcessor(
	cacher storage.Cacher,
	peerBlacklistHandler process.PeerBlackListHandler,
	thresholdNumReceivedFlood uint32,
	thresholdSizeReceivedFlood uint64,
	numFloodingRounds uint32,
	banDuration time.Duration,
) (*p2pBlackListProcessor, error) {

	if check.IfNil(cacher) {
		return nil, fmt.Errorf("%w, NewP2PBlackListProcessor", process.ErrNilCacher)
	}
	if check.IfNil(peerBlacklistHandler) {
		return nil, fmt.Errorf("%w, NewP2PBlackListProcessor", process.ErrNilBlackListHandler)
	}
	if thresholdNumReceivedFlood == 0 {
		return nil, fmt.Errorf("%w, thresholdNumReceivedFlood == 0", process.ErrInvalidValue)
	}
	if thresholdSizeReceivedFlood == 0 {
		return nil, fmt.Errorf("%w, thresholdSizeReceivedFlood == 0", process.ErrInvalidValue)
	}
	if numFloodingRounds < minFloodingRounds {
		return nil, fmt.Errorf("%w, numFloodingRounds < %d", process.ErrInvalidValue, minFloodingRounds)
	}
	if banDuration < minBanDuration {
		return nil, fmt.Errorf("%w for ban duration in NewP2PBlackListProcessor", process.ErrInvalidValue)
	}

	return &p2pBlackListProcessor{
		cacher:                     cacher,
		peerBlacklistHandler:       peerBlacklistHandler,
		thresholdNumReceivedFlood:  thresholdNumReceivedFlood,
		thresholdSizeReceivedFlood: thresholdSizeReceivedFlood,
		numFloodingRounds:          numFloodingRounds,
		banDuration:                banDuration,
	}, nil
}

// ResetStatistics checks if an identifier reached its maximum flooding rounds. If it did, it will remove its
// cached information and adds it to the black list handler
func (pbp *p2pBlackListProcessor) ResetStatistics() {
	keys := pbp.cacher.Keys()
	for _, key := range keys {
		val, ok := pbp.getFloodingValue(key)
		if !ok {
			pbp.cacher.Remove(key)
			continue
		}

		if val >= pbp.numFloodingRounds-1 { //-1 because the reset function is called before the AddQuota
			pbp.cacher.Remove(key)
			pid := core.PeerID(key)
			log.Debug("added new peer to black list",
				"peer ID", pid.Pretty(),
				"ban period", pbp.banDuration,
			)
			_ = pbp.peerBlacklistHandler.AddWithSpan(pid, pbp.banDuration)
		}
	}
}

func (pbp *p2pBlackListProcessor) getFloodingValue(key []byte) (uint32, bool) {
	obj, ok := pbp.cacher.Peek(key)
	if !ok {
		return 0, false
	}

	val, ok := obj.(uint32)

	return val, ok
}

// AddQuota checks if the received quota for an identifier has exceeded the set thresholds
func (pbp *p2pBlackListProcessor) AddQuota(pid core.PeerID, numReceived uint32, sizeReceived uint64, _ uint32, _ uint64) {
	isFloodingPeer := numReceived >= pbp.thresholdNumReceivedFlood || sizeReceived >= pbp.thresholdSizeReceivedFlood
	if isFloodingPeer {
		pbp.incrementStatsFloodingPeer(pid)
	}
}

func (pbp *p2pBlackListProcessor) incrementStatsFloodingPeer(pid core.PeerID) {
	obj, ok := pbp.cacher.Get(pid.Bytes())
	if !ok {
		pbp.cacher.Put(pid.Bytes(), uint32(1), sizeBlacklistInfo)
		return
	}

	val, ok := obj.(uint32)
	if !ok {
		pbp.cacher.Put(pid.Bytes(), uint32(1), sizeBlacklistInfo)
		return
	}

	pbp.cacher.Put(pid.Bytes(), val+1, sizeBlacklistInfo)
}

// IsInterfaceNil returns true if there is no value under the interface
func (pbp *p2pBlackListProcessor) IsInterfaceNil() bool {
	return pbp == nil
}
