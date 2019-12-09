package interceptors

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/logger"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
)

var log = logger.GetOrCreate("process/interceptors")

// MultiDataInterceptor is used for intercepting packed multi data
type MultiDataInterceptor struct {
	marshalizer marshal.Marshalizer
	factory     process.InterceptedDataFactory
	processor   process.InterceptorProcessor
	throttler   process.InterceptorThrottler
}

// NewMultiDataInterceptor hooks a new interceptor for packed multi data
func NewMultiDataInterceptor(
	marshalizer marshal.Marshalizer,
	factory process.InterceptedDataFactory,
	processor process.InterceptorProcessor,
	throttler process.InterceptorThrottler,
) (*MultiDataInterceptor, error) {

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

	multiDataIntercept := &MultiDataInterceptor{
		marshalizer: marshalizer,
		factory:     factory,
		processor:   processor,
		throttler:   throttler,
	}

	return multiDataIntercept, nil
}

// ProcessReceivedMessage is the callback func from the p2p.Messenger and will be called each time a new message was received
// (for the topic this validator was registered to)
func (mdi *MultiDataInterceptor) ProcessReceivedMessage(message p2p.MessageP2P, fromConnectedPeer p2p.PeerID) error {
	err := preProcessMesage(mdi.throttler, message)
	if err != nil {
		return err
	}

	multiDataBuff := make([][]byte, 0)
	err = mdi.marshalizer.Unmarshal(&multiDataBuff, message.Data())
	if err != nil {
		mdi.throttler.EndProcessing()
		return err
	}
	if len(multiDataBuff) == 0 {
		mdi.throttler.EndProcessing()
		return process.ErrNoDataInMessage
	}

	lastErrEncountered := error(nil)
	wgProcess := &sync.WaitGroup{}
	wgProcess.Add(len(multiDataBuff))
	go func() {
		wgProcess.Wait()
		mdi.throttler.EndProcessing()
	}()

	for _, dataBuff := range multiDataBuff {
		interceptedData, err := mdi.factory.Create(dataBuff)
		if err != nil {
			lastErrEncountered = err
			wgProcess.Done()
			continue
		}

		err = interceptedData.CheckValidity()
		if err != nil {
			lastErrEncountered = err
			wgProcess.Done()
			continue
		}

		if !interceptedData.IsForCurrentShard() {
			log.Trace("intercepted data is for other shards")
			wgProcess.Done()
			continue
		}

		go processInterceptedData(mdi.processor, interceptedData, wgProcess)
	}

	return lastErrEncountered
}

// IsInterfaceNil returns true if there is no value under the interface
func (mdi *MultiDataInterceptor) IsInterfaceNil() bool {
	if mdi == nil {
		return true
	}
	return false
}
