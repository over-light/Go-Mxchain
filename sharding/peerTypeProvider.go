package sharding

import (
	"bytes"
	"sync"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/epochStart/notifier"
)

// TODO: move this component somewhere else and write tests for it

type peerListAndShard struct {
	pType  core.PeerType
	pShard uint32
}

// PeerTypeProvider handles the computation of a peer type
type PeerTypeProvider struct {
	nodesCoordinator NodesCoordinator
	epochHandler     EpochHandler
	cache            map[string]*peerListAndShard
	mutCache         sync.RWMutex
}

type ArgPeerTypeProvider struct {
	NodesCoordinator        NodesCoordinator
	EpochHandler            EpochHandler
	EpochStartEventNotifier EpochStartEventNotifier
}

// NewPeerTypeProvider will return a new instance of PeerTypeProvider
func NewPeerTypeProvider(arg ArgPeerTypeProvider) (*PeerTypeProvider, error) {
	if check.IfNil(arg.NodesCoordinator) {
		return nil, ErrNilNodesCoordinator
	}
	if check.IfNil(arg.EpochHandler) {
		return nil, ErrNilEpochHandler
	}
	if arg.EpochStartEventNotifier == nil {
		return nil, ErrNilEpochStartNotifier
	}

	ptp := &PeerTypeProvider{
		nodesCoordinator: arg.NodesCoordinator,
		epochHandler:     arg.EpochHandler,
		cache:            make(map[string]*peerListAndShard),
		mutCache:         sync.RWMutex{},
	}

	err := ptp.populateCache(ptp.epochHandler.MetaEpoch())
	if err != nil {
		return nil, err
	}

	arg.EpochStartEventNotifier.RegisterHandler(ptp.epochStartEventHandler())

	return ptp, nil
}

func (ptp *PeerTypeProvider) populateCache(epoch uint32) error {
	ptp.mutCache.Lock()
	defer ptp.mutCache.Unlock()

	ptp.cache = make(map[string]*peerListAndShard)

	nodesMapEligible, err := ptp.nodesCoordinator.GetAllEligibleValidatorsPublicKeys(epoch)
	if err != nil {
		return err
	}
	ptp.setPeerTypeAndShardId(nodesMapEligible, core.EligibleList)

	nodesMapWaiting, err := ptp.nodesCoordinator.GetAllWaitingValidatorsPublicKeys(epoch)
	if err != nil {
		return err
	}
	ptp.setPeerTypeAndShardId(nodesMapWaiting, core.WaitingList)

	return nil
}

func (ptp *PeerTypeProvider) setPeerTypeAndShardId(nodesMapEligible map[uint32][][]byte, currentPeerType core.PeerType) {
	for shardID, eligibleValidatorsInShard := range nodesMapEligible {
		for _, val := range eligibleValidatorsInShard {
			ptp.cache[string(val)] = &peerListAndShard{
				pType:  currentPeerType,
				pShard: shardID,
			}
		}
	}
}

// ComputeForPubKey returns the peer type for a given public key and shard id
func (ptp *PeerTypeProvider) ComputeForPubKey(pubKey []byte) (core.PeerType, uint32, error) {
	ptp.mutCache.RLock()
	peerData, ok := ptp.cache[string(pubKey)]
	ptp.mutCache.RUnlock()
	if ok {
		return peerData.pType, peerData.pShard, nil
	}

	return ptp.computeFromMaps(pubKey)
}

func (ptp *PeerTypeProvider) computeFromMaps(pubKey []byte) (core.PeerType, uint32, error) {
	nodesMapEligible, err := ptp.nodesCoordinator.GetAllEligibleValidatorsPublicKeys(ptp.epochHandler.MetaEpoch())
	if err != nil {
		return "", 0, err
	}
	for shID, eligibleValidatorsInShard := range nodesMapEligible {
		for _, val := range eligibleValidatorsInShard {
			if bytes.Equal(val, pubKey) {
				peerType := core.EligibleList
				shardID := shID
				ptp.mutCache.Lock()
				ptp.cache[string(pubKey)] = &peerListAndShard{
					pType:  peerType,
					pShard: shardID,
				}
				ptp.mutCache.Unlock()
				return peerType, shardID, nil
			}
		}
	}

	nodesMapWaiting, err := ptp.nodesCoordinator.GetAllWaitingValidatorsPublicKeys(ptp.epochHandler.MetaEpoch())
	if err != nil {
		return "", 0, err
	}
	for shID, waitingValidatorsInShard := range nodesMapWaiting {
		for _, val := range waitingValidatorsInShard {
			if bytes.Equal(val, pubKey) {
				peerType := core.WaitingList
				shardID := shID
				ptp.mutCache.Lock()
				ptp.cache[string(pubKey)] = &peerListAndShard{
					pType:  peerType,
					pShard: shardID,
				}
				ptp.mutCache.Unlock()
				return peerType, shardID, nil
			}
		}
	}

	return core.ObserverList, uint32(0), nil
}

func (ptp *PeerTypeProvider) epochStartEventHandler() EpochStartActionHandler {
	subscribeHandler := notifier.NewHandlerForEpochStart(func(hdr data.HeaderHandler) {
		err := ptp.populateCache(hdr.GetEpoch())
		if err != nil {
			log.Warn("change epoch in peer type provider", "error", err.Error())
		}
	}, func(_ data.HeaderHandler) {}, core.NodesCoordinatorOrder)

	return subscribeHandler
}

// IsInterfaceNil returns true if there is no value under the interface
func (ptp *PeerTypeProvider) IsInterfaceNil() bool {
	return ptp == nil
}
