package networksharding

import (
	"fmt"
	"math/big"

	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/p2p/libp2p/networksharding/sorting"
	"github.com/libp2p/go-libp2p-core/peer"
)

type oneListSharder struct {
	selfPeerId      peer.ID
	maxPeerCount    int
	computeDistance func(src peer.ID, dest peer.ID) *big.Int
}

// NewOneListSharder creates a new sharder instance that is shard agnostic and uses one list
func NewOneListSharder(
	selfPeerId peer.ID,
	maxPeerCount int,
) (*oneListSharder, error) {
	if maxPeerCount < minAllowedConnectedPeers {
		return nil, fmt.Errorf("%w, maxPeerCount should be at least %d", p2p.ErrInvalidValue, minAllowedConnectedPeers)
	}

	return &oneListSharder{
		selfPeerId:      selfPeerId,
		maxPeerCount:    maxPeerCount,
		computeDistance: computeDistanceByCountingBits,
	}, nil
}

// ComputeEvictionList returns the eviction list
func (ols *oneListSharder) ComputeEvictionList(pidList []peer.ID) []peer.ID {
	list := ols.convertList(pidList)
	_, evictionProposed := evict(list, ols.maxPeerCount)

	return evictionProposed
}

func (ols *oneListSharder) convertList(peers []peer.ID) sorting.PeerDistances {
	list := sorting.PeerDistances{}

	for _, p := range peers {
		pd := &sorting.PeerDistance{
			ID:       p,
			Distance: ols.computeDistance(p, ols.selfPeerId),
		}
		list = append(list, pd)
	}

	return list
}

// Has returns true if provided pid is among the provided list
func (ols *oneListSharder) Has(pid peer.ID, list []peer.ID) bool {
	return has(pid, list)
}

// SetPeerShardResolver sets the peer shard resolver for this sharder. Doesn't do anything in this implementation
func (ols *oneListSharder) SetPeerShardResolver(_ p2p.PeerShardResolver) error {
	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (ols *oneListSharder) IsInterfaceNil() bool {
	return ols == nil
}
