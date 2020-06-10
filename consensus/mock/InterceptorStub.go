package mock

import (
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
)

// InterceptorStub -
type InterceptorStub struct {
	ProcessReceivedMessageCalled func(message p2p.MessageP2P) error
	RegisterHandlerCalled        func(handler func(topic string, data []byte, _ interface{}))
}

// ProcessReceivedMessage -
func (is *InterceptorStub) ProcessReceivedMessage(message p2p.MessageP2P, _ p2p.PeerID) error {
	return is.ProcessReceivedMessageCalled(message)
}

// SetInterceptedDebugHandler -
func (is *InterceptorStub) SetInterceptedDebugHandler(_ process.InterceptedDebugHandler) error {
	return nil
}

// RegisterHandler -
func (is *InterceptorStub) RegisterHandler(handler func(topic string, hash []byte, data interface{})) {
	if is.RegisterHandlerCalled != nil {
		is.RegisterHandlerCalled(handler)
	}
}

// IsInterfaceNil returns true if there is no value under the interface
func (is *InterceptorStub) IsInterfaceNil() bool {
	return is == nil
}
