package interceptors

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
)

func preProcessMesage(
	throttler process.InterceptorThrottler,
	antifloodHandler process.P2PAntifloodHandler,
	message p2p.MessageP2P,
	fromConnectedPeer core.PeerID,
	topic string,
) error {

	if message == nil {
		return process.ErrNilMessage
	}
	if message.Data() == nil {
		return process.ErrNilDataToProcess
	}
	err := antifloodHandler.CanProcessMessage(message, fromConnectedPeer)
	if err != nil {
		return err
	}
	err = antifloodHandler.CanProcessMessagesOnTopic(fromConnectedPeer, topic, 1, uint64(len(message.Data())), message.SeqNo())
	if err != nil {
		return err
	}

	if !throttler.CanProcess() {
		return process.ErrSystemBusy
	}

	throttler.StartProcessing()
	return nil
}

func processInterceptedData(
	processor process.InterceptorProcessor,
	handler process.InterceptedDebugger,
	data process.InterceptedData,
	topic string,
	wgProcess *sync.WaitGroup,
	msg p2p.MessageP2P,
) {
	err := processor.Validate(data, msg.Peer())

	defer func() {
		wgProcess.Done()
	}()
	if err != nil {
		log.Trace("intercepted data is not valid",
			"hash", data.Hash(),
			"type", data.Type(),
			"pid", p2p.MessageOriginatorPid(msg),
			"seq no", p2p.MessageOriginatorSeq(msg),
			"data", data.String(),
			"error", err.Error(),
		)
		processDebugInterceptedData(handler, data, topic, err)

		return
	}

	err = processor.Save(data, msg.Peer(), topic)
	if err != nil {
		log.Trace("intercepted data can not be processed",
			"hash", data.Hash(),
			"type", data.Type(),
			"pid", p2p.MessageOriginatorPid(msg),
			"seq no", p2p.MessageOriginatorSeq(msg),
			"data", data.String(),
			"error", err.Error(),
		)
		processDebugInterceptedData(handler, data, topic, err)

		return
	}

	log.Trace("intercepted data is processed",
		"hash", data.Hash(),
		"type", data.Type(),
		"pid", p2p.MessageOriginatorPid(msg),
		"seq no", p2p.MessageOriginatorSeq(msg),
		"data", data.String(),
	)
	processDebugInterceptedData(handler, data, topic, err)
}

func processDebugInterceptedData(
	debugHandler process.InterceptedDebugger,
	interceptedData process.InterceptedData,
	topic string,
	err error,
) {
	identifiers := interceptedData.Identifiers()
	debugHandler.LogProcessedHashes(topic, identifiers, err)
}

func receivedDebugInterceptedData(
	debugHandler process.InterceptedDebugger,
	interceptedData process.InterceptedData,
	topic string,
) {
	identifiers := interceptedData.Identifiers()
	debugHandler.LogReceivedHashes(topic, identifiers)
}
