package interceptors

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
)

func preProcessMesage(
	throttler process.InterceptorThrottler,
	antifloodHandler process.P2PAntifloodHandler,
	message p2p.MessageP2P,
	fromConnectedPeer p2p.PeerID,
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
	err = antifloodHandler.CanProcessMessageOnTopic(fromConnectedPeer, topic)
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
	whiteListhandler process.WhiteListHandler,
	data process.InterceptedData,
	wgProcess *sync.WaitGroup,
	msg p2p.MessageP2P,
) {
	hash := data.Hash()
	err := processor.Validate(data)
	if err != nil {
		log.Trace("intercepted data is not valid",
			"hash", data.Hash(),
			"type", data.Type(),
			"pid", p2p.MessageOriginatorPid(msg),
			"seq no", p2p.MessageOriginatorSeq(msg),
			"data", data.String(),
			"error", err.Error(),
		)
		whiteListhandler.Remove([][]byte{hash})
		wgProcess.Done()

		return
	}

	err = processor.Save(data)
	if err != nil {
		log.Trace("intercepted data can not be processed",
			"hash", data.Hash(),
			"type", data.Type(),
			"pid", p2p.MessageOriginatorPid(msg),
			"seq no", p2p.MessageOriginatorSeq(msg),
			"data", data.String(),
			"error", err.Error(),
		)
		whiteListhandler.Remove([][]byte{hash})
		wgProcess.Done()

		return
	}

	log.Trace("intercepted data is processed",
		"hash", data.Hash(),
		"type", data.Type(),
		"pid", p2p.MessageOriginatorPid(msg),
		"seq no", p2p.MessageOriginatorSeq(msg),
		"data", data.String(),
	)
	whiteListhandler.Remove([][]byte{hash})
	wgProcess.Done()
}
