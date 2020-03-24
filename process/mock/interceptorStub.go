package mock

import (
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
)

// InterceptorStub -
type InterceptorStub struct {
	ProcessReceivedMessageCalled func(message p2p.MessageP2P) error
	SetIsDataForCurrentShardVerifierCalled func(verifier process.InterceptedDataVerifier) error
}

func (is *InterceptorStub) SetIsDataForCurrentShardVerifier(verifier process.InterceptedDataVerifier) error {
	return is.SetIsDataForCurrentShardVerifierCalled(verifier)
}

// ProcessReceivedMessage -
func (is *InterceptorStub) ProcessReceivedMessage(message p2p.MessageP2P, _ p2p.PeerID) error {
	return is.ProcessReceivedMessageCalled(message)
}

// IsInterfaceNil returns true if there is no value under the interface
func (is *InterceptorStub) IsInterfaceNil() bool {
	return is == nil
}
