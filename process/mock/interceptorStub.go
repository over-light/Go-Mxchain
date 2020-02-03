package mock

import (
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
)

type InterceptorStub struct {
	ProcessReceivedMessageCalled           func(message p2p.MessageP2P, broadcastHandler func(buffToSend []byte)) error
	SetIsDataForCurrentShardVerifierCalled func(verifier process.InterceptedDataVerifier)
}

func (is *InterceptorStub) SetIsDataForCurrentShardVerifier(verifier process.InterceptedDataVerifier) {
	is.SetIsDataForCurrentShardVerifierCalled(verifier)
}

func (is *InterceptorStub) ProcessReceivedMessage(message p2p.MessageP2P, broadcastHandler func(buffToSend []byte)) error {
	return is.ProcessReceivedMessageCalled(message, broadcastHandler)
}

// IsInterfaceNil returns true if there is no value under the interface
func (is *InterceptorStub) IsInterfaceNil() bool {
	return is == nil
}
