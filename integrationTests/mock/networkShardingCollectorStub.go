package mock

import "github.com/ElrondNetwork/elrond-go/p2p"

// NetworkShardingCollectorStub -
type NetworkShardingCollectorStub struct {
	UpdatePeerIdPublicKeyCalled  func(pid p2p.PeerID, pk []byte)
	UpdatePublicKeyShardIdCalled func(pk []byte, shardId uint32)
	UpdatePeerIdShardIdCalled    func(pid p2p.PeerID, shardId uint32)
}

// UpdatePeerIdPublicKey -
func (nscs *NetworkShardingCollectorStub) UpdatePeerIdPublicKey(pid p2p.PeerID, pk []byte) {
	nscs.UpdatePeerIdPublicKeyCalled(pid, pk)
}

// UpdatePublicKeyShardId -
func (nscs *NetworkShardingCollectorStub) UpdatePublicKeyShardId(pk []byte, shardId uint32) {
	nscs.UpdatePublicKeyShardIdCalled(pk, shardId)
}

// UpdatePeerIdShardId -
func (nscs *NetworkShardingCollectorStub) UpdatePeerIdShardId(pid p2p.PeerID, shardId uint32) {
	nscs.UpdatePeerIdShardIdCalled(pid, shardId)
}

// IsInterfaceNil -
func (nscs *NetworkShardingCollectorStub) IsInterfaceNil() bool {
	return nscs == nil
}
