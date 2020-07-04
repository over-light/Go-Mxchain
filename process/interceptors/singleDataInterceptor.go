package interceptors

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/debug/resolver"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
)

// SingleDataInterceptor is used for intercepting packed multi data
type SingleDataInterceptor struct {
	topic                      string
	factory                    process.InterceptedDataFactory
	processor                  process.InterceptorProcessor
	throttler                  process.InterceptorThrottler
	whiteListRequested         process.WhiteListHandler
	antifloodHandler           process.P2PAntifloodHandler
	mutInterceptedDebugHandler sync.RWMutex
	interceptedDebugHandler    process.InterceptedDebugger
}

// NewSingleDataInterceptor hooks a new interceptor for single data
func NewSingleDataInterceptor(
	topic string,
	factory process.InterceptedDataFactory,
	processor process.InterceptorProcessor,
	throttler process.InterceptorThrottler,
	antifloodHandler process.P2PAntifloodHandler,
	whiteListRequested process.WhiteListHandler,
) (*SingleDataInterceptor, error) {
	if len(topic) == 0 {
		return nil, process.ErrEmptyTopic
	}
	if check.IfNil(factory) {
		return nil, process.ErrNilInterceptedDataFactory
	}
	if check.IfNil(processor) {
		return nil, process.ErrNilInterceptedDataProcessor
	}
	if check.IfNil(throttler) {
		return nil, process.ErrNilInterceptorThrottler
	}
	if check.IfNil(antifloodHandler) {
		return nil, process.ErrNilAntifloodHandler
	}
	if check.IfNil(whiteListRequested) {
		return nil, process.ErrNilWhiteListHandler
	}

	singleDataIntercept := &SingleDataInterceptor{
		topic:              topic,
		factory:            factory,
		processor:          processor,
		throttler:          throttler,
		antifloodHandler:   antifloodHandler,
		whiteListRequested: whiteListRequested,
	}
	singleDataIntercept.interceptedDebugHandler = resolver.NewDisabledInterceptorResolver()

	return singleDataIntercept, nil
}

// ProcessReceivedMessage is the callback func from the p2p.Messenger and will be called each time a new message was received
// (for the topic this validator was registered to)
func (sdi *SingleDataInterceptor) ProcessReceivedMessage(message p2p.MessageP2P, fromConnectedPeer core.PeerID) error {
	sdi.mutInterceptedDebugHandler.RLock()
	defer sdi.mutInterceptedDebugHandler.RUnlock()

	err := preProcessMesage(sdi.throttler, sdi.antifloodHandler, message, fromConnectedPeer, sdi.topic)
	if err != nil {
		return err
	}

	interceptedData, err := sdi.factory.Create(message.Data())
	if err != nil {
		sdi.throttler.EndProcessing()

		//this situation is so severe that we need to black list the peers
		reason := "can not create object from received bytes, topic " + sdi.topic
		sdi.antifloodHandler.BlacklistPeer(message.Peer(), reason, core.InvalidMessageBlacklistDuration)
		sdi.antifloodHandler.BlacklistPeer(fromConnectedPeer, reason, core.InvalidMessageBlacklistDuration)

		return err
	}

	receivedDebugInterceptedData(sdi.interceptedDebugHandler, interceptedData, sdi.topic)

	err = interceptedData.CheckValidity()
	if err != nil {
		sdi.throttler.EndProcessing()
		processDebugInterceptedData(sdi.interceptedDebugHandler, interceptedData, sdi.topic, err)

		isWrongVersion := err == process.ErrInvalidTransactionVersion || err == process.ErrInvalidChainID
		if isWrongVersion {
			//this situation is so severe that we need to black list de peers
			reason := "wrong version of received intercepted data, topic " + sdi.topic + ", error " + err.Error()
			sdi.antifloodHandler.BlacklistPeer(message.Peer(), reason, core.InvalidMessageBlacklistDuration)
			sdi.antifloodHandler.BlacklistPeer(fromConnectedPeer, reason, core.InvalidMessageBlacklistDuration)
		}

		return err
	}

	errOriginator := sdi.antifloodHandler.IsOriginatorEligibleForTopic(message.Peer(), sdi.topic)
	isWhiteListed := sdi.whiteListRequested.IsWhiteListed(interceptedData)
	if !isWhiteListed && errOriginator != nil {
		log.Debug("got message from peer on topic only for validators",
			"originator", p2p.PeerIdToShortString(message.Peer()), "topic",
			sdi.topic, "err", errOriginator)
		sdi.throttler.EndProcessing()
		return errOriginator
	}

	isForCurrentShard := interceptedData.IsForCurrentShard()
	shouldProcess := isForCurrentShard || isWhiteListed
	if !shouldProcess {
		sdi.throttler.EndProcessing()
		log.Trace("intercepted data is for other shards",
			"pid", p2p.MessageOriginatorPid(message),
			"seq no", p2p.MessageOriginatorSeq(message),
			"topics", message.Topics(),
			"hash", interceptedData.Hash(),
			"is for current shard", isForCurrentShard,
			"is white listed", isWhiteListed,
		)

		return nil
	}

	go func() {
		processInterceptedData(
			sdi.processor,
			sdi.interceptedDebugHandler,
			interceptedData,
			sdi.topic,
			message,
		)
		sdi.throttler.EndProcessing()
	}()

	return nil
}

// SetInterceptedDebugHandler will set a new intercepted debug handler
func (sdi *SingleDataInterceptor) SetInterceptedDebugHandler(handler process.InterceptedDebugger) error {
	if check.IfNil(handler) {
		return process.ErrNilDebugger
	}

	sdi.mutInterceptedDebugHandler.Lock()
	sdi.interceptedDebugHandler = handler
	sdi.mutInterceptedDebugHandler.Unlock()

	return nil
}

// RegisterHandler registers a callback function to be notified on received data
func (sdi *SingleDataInterceptor) RegisterHandler(handler func(topic string, hash []byte, data interface{})) {
	sdi.processor.RegisterHandler(handler)
}

// IsInterfaceNil returns true if there is no value under the interface
func (sdi *SingleDataInterceptor) IsInterfaceNil() bool {
	return sdi == nil
}
