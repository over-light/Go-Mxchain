package mock

import (
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/p2p"
)

// HeaderResolverMock -
type HeaderResolverMock struct {
	RequestDataFromHashCalled    func(hash []byte, epoch uint32) error
	ProcessReceivedMessageCalled func(message p2p.MessageP2P) error
	RequestDataFromNonceCalled   func(nonce uint64, epoch uint32) error
	RequestDataFromEpochCalled   func(identifier []byte) error
	SetEpochHandlerCalled        func(epochHandler dataRetriever.EpochHandler) error
	SetNumPeersToQueryCalled     func(intra int, cross int)
	GetNumPeersToQueryCalled     func() (int, int)
}

// SetNumPeersToQuery -
func (hrm *HeaderResolverMock) SetNumPeersToQuery(intra int, cross int) {
	if hrm.SetNumPeersToQueryCalled != nil {
		hrm.SetNumPeersToQueryCalled(intra, cross)
	}
}

// GetNumPeersToQuery -
func (hrm *HeaderResolverMock) GetNumPeersToQuery() (int, int) {
	if hrm.GetNumPeersToQueryCalled != nil {
		return hrm.GetNumPeersToQueryCalled()
	}

	return 2, 2
}

// RequestDataFromEpoch -
func (hrm *HeaderResolverMock) RequestDataFromEpoch(identifier []byte) error {
	if hrm.RequestDataFromEpochCalled != nil {
		return hrm.RequestDataFromEpochCalled(identifier)
	}
	return nil
}

// SetEpochHandler -
func (hrm *HeaderResolverMock) SetEpochHandler(epochHandler dataRetriever.EpochHandler) error {
	if hrm.SetEpochHandlerCalled != nil {
		return hrm.SetEpochHandlerCalled(epochHandler)
	}
	return nil
}

// RequestDataFromHash -
func (hrm *HeaderResolverMock) RequestDataFromHash(hash []byte, epoch uint32) error {
	if hrm.RequestDataFromHashCalled == nil {
		return nil
	}
	return hrm.RequestDataFromHashCalled(hash, epoch)
}

// ProcessReceivedMessage -
func (hrm *HeaderResolverMock) ProcessReceivedMessage(message p2p.MessageP2P, _ p2p.PeerID) error {
	if hrm.ProcessReceivedMessageCalled == nil {
		return nil
	}
	return hrm.ProcessReceivedMessageCalled(message)
}

// RequestDataFromNonce -
func (hrm *HeaderResolverMock) RequestDataFromNonce(nonce uint64, epoch uint32) error {
	if hrm.RequestDataFromNonceCalled == nil {
		return nil
	}
	return hrm.RequestDataFromNonceCalled(nonce, epoch)
}

// IsInterfaceNil returns true if there is no value under the interface
func (hrm *HeaderResolverMock) IsInterfaceNil() bool {
	return hrm == nil
}
