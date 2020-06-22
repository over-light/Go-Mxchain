package blackList

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go-logger/check"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/process"
)

type peerDenialEvaluator struct {
	peerIdsCache    process.PeerBlackListCacher
	publicKeysCache process.TimeCacher
	peerShardMapper process.PeerShardMapper
}

// NewPeerDenialEvaluator will create a new instance of a peer deny cache evaluator
func NewPeerDenialEvaluator(
	pids process.PeerBlackListCacher,
	pks process.TimeCacher,
	psm process.PeerShardMapper,
) (*peerDenialEvaluator, error) {

	if check.IfNil(pids) {
		return nil, fmt.Errorf("%w for peer IDs cacher", process.ErrNilBlackListCacher)
	}
	if check.IfNil(pks) {
		return nil, fmt.Errorf("%w for public keys cacher", process.ErrNilBlackListCacher)
	}
	if check.IfNil(psm) {
		return nil, process.ErrNilPeerShardMapper
	}

	return &peerDenialEvaluator{
		peerIdsCache:    pids,
		publicKeysCache: pks,
		peerShardMapper: psm,
	}, nil
}

// IsDenied returns true if the provided peer id is denied to access the network
// It also checks if the provided peer id has a backing public key, checking also that the public key is not denied
func (pde *peerDenialEvaluator) IsDenied(pid core.PeerID) bool {
	if pde.peerIdsCache.Has(pid) {
		return true
	}

	peerInfo := pde.peerShardMapper.GetPeerInfo(pid)
	pkBytes := peerInfo.PkBytes
	if len(pkBytes) == 0 {
		return false //no need to further search in the next cache, this is an unknown peer
	}

	return pde.publicKeysCache.Has(string(pkBytes))
}

// IsInterfaceNil returns true if there is no value under the interface
func (pde *peerDenialEvaluator) IsInterfaceNil() bool {
	return pde == nil
}
