package mock

import (
	"github.com/ElrondNetwork/elrond-go/p2p"
)

// ResolverStub -
type ResolverStub struct {
	RequestDataFromHashCalled    func(hash []byte, epoch uint32) error
	ProcessReceivedMessageCalled func(message p2p.MessageP2P) error
}

// SetIntraAndCrossShardNumPeersToQuery -
func (rs *ResolverStub) SetIntraAndCrossShardNumPeersToQuery(intra int, cross int) {
}

// GetIntraAndCrossShardNumPeersToQuery -
func (rs *ResolverStub) GetIntraAndCrossShardNumPeersToQuery() (int, int) {
	return 2, 2
}

// RequestDataFromHash -
func (rs *ResolverStub) RequestDataFromHash(hash []byte, epoch uint32) error {
	return rs.RequestDataFromHashCalled(hash, epoch)
}

// ProcessReceivedMessage -
func (rs *ResolverStub) ProcessReceivedMessage(message p2p.MessageP2P, _ p2p.PeerID) error {
	return rs.ProcessReceivedMessageCalled(message)
}

// IsInterfaceNil returns true if there is no value under the interface
func (rs *ResolverStub) IsInterfaceNil() bool {
	return rs == nil
}
