package interceptors

import (
	"sync"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/batch"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
)

var log = logger.GetOrCreate("process/interceptors")

// MultiDataInterceptor is used for intercepting packed multi data
type MultiDataInterceptor struct {
	topic            string
	marshalizer      marshal.Marshalizer
	factory          process.InterceptedDataFactory
	processor        process.InterceptorProcessor
	throttler        process.InterceptorThrottler
	whiteListHandler process.WhiteListHandler
	antifloodHandler process.P2PAntifloodHandler
}

// NewMultiDataInterceptor hooks a new interceptor for packed multi data
func NewMultiDataInterceptor(
	topic string,
	marshalizer marshal.Marshalizer,
	factory process.InterceptedDataFactory,
	processor process.InterceptorProcessor,
	throttler process.InterceptorThrottler,
	antifloodHandler process.P2PAntifloodHandler,
	whiteListHandler process.WhiteListHandler,
) (*MultiDataInterceptor, error) {
	if len(topic) == 0 {
		return nil, process.ErrEmptyTopic
	}
	if check.IfNil(marshalizer) {
		return nil, process.ErrNilMarshalizer
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
	if check.IfNil(whiteListHandler) {
		return nil, process.ErrNilWhiteListHandler
	}

	multiDataIntercept := &MultiDataInterceptor{
		topic:            topic,
		marshalizer:      marshalizer,
		factory:          factory,
		processor:        processor,
		throttler:        throttler,
		whiteListHandler: whiteListHandler,
		antifloodHandler: antifloodHandler,
	}

	return multiDataIntercept, nil
}

// ProcessReceivedMessage is the callback func from the p2p.Messenger and will be called each time a new message was received
// (for the topic this validator was registered to)
func (mdi *MultiDataInterceptor) ProcessReceivedMessage(message p2p.MessageP2P, fromConnectedPeer p2p.PeerID) error {
	err := preProcessMesage(mdi.throttler, mdi.antifloodHandler, message, fromConnectedPeer, mdi.topic)
	if err != nil {
		return err
	}

	b := batch.Batch{}
	err = mdi.marshalizer.Unmarshal(&b, message.Data())
	if err != nil {
		mdi.throttler.EndProcessing()
		return err
	}
	multiDataBuff := b.Data
	if len(multiDataBuff) == 0 {
		mdi.throttler.EndProcessing()
		return process.ErrNoDataInMessage
	}

	interceptedMultiData := make([]process.InterceptedData, 0)
	lastErrEncountered := error(nil)
	wgProcess := &sync.WaitGroup{}
	wgProcess.Add(len(multiDataBuff))

	go func() {
		wgProcess.Wait()
		mdi.processor.SignalEndOfProcessing(interceptedMultiData)
		mdi.throttler.EndProcessing()
	}()

	for _, dataBuff := range multiDataBuff {
		var interceptedData process.InterceptedData
		interceptedData, err = mdi.interceptedData(dataBuff)
		if err != nil {
			lastErrEncountered = err
			wgProcess.Done()
			continue
		}

		interceptedMultiData = append(interceptedMultiData, interceptedData)

		isForCurrentShard := interceptedData.IsForCurrentShard()
		isWhiteListed := mdi.whiteListHandler.IsWhiteListed(interceptedData)
		shouldProcess := isForCurrentShard || isWhiteListed
		if !shouldProcess {
			log.Trace("intercepted data should not be processed",
				"pid", p2p.MessageOriginatorPid(message),
				"seq no", p2p.MessageOriginatorSeq(message),
				"topics", message.Topics(),
				"hash", interceptedData.Hash(),
				"is for this shard", isForCurrentShard,
				"is white listed", isWhiteListed,
			)
			wgProcess.Done()
			continue
		}

		go processInterceptedData(mdi.processor, interceptedData, wgProcess, message)
	}

	return lastErrEncountered
}

func (mdi *MultiDataInterceptor) interceptedData(dataBuff []byte) (process.InterceptedData, error) {
	interceptedData, err := mdi.factory.Create(dataBuff)
	if err != nil {
		return nil, err
	}

	err = interceptedData.CheckValidity()
	if err != nil {
		return nil, err
	}

	return interceptedData, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (mdi *MultiDataInterceptor) IsInterfaceNil() bool {
	return mdi == nil
}
