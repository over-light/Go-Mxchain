package interceptors

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
)

func preProcessMesage(throttler process.InterceptorThrottler, message p2p.MessageP2P) error {
	if message == nil {
		return process.ErrNilMessage
	}
	if message.Data() == nil {
		return process.ErrNilDataToProcess
	}

	if !throttler.CanProcess() {
		return process.ErrSystemBusy
	}

	throttler.StartProcessing()
	return nil
}

func processInterceptedData(
	processor process.InterceptorProcessor,
	data process.InterceptedData,
	wgProcess *sync.WaitGroup,
	msg p2p.MessageP2P,
) {
	err := processor.Validate(data)
	if err != nil {
		seqNo := msg.SeqNo()
		var strSeq string
		if len(seqNo) >= 8 {
			strSeq = fmt.Sprintf("%d", binary.BigEndian.Uint64(seqNo))
		} else {
			strSeq = hex.EncodeToString(seqNo)
		}

		log.Trace("intercepted data is not valid",
			"hash", data.Hash(),
			"type", data.Type(),
			"pid", msg.Peer().Pretty(),
			"seq no", strSeq,
			"error", err.Error(),
		)
		wgProcess.Done()
		return
	}

	err = processor.Save(data)
	if err != nil {
		log.Trace("intercepted data can not be processed",
			"hash", data.Hash(),
			"type", data.Type(),
			"error", err.Error(),
		)
	}

	wgProcess.Done()
}
