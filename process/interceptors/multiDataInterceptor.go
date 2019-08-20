package interceptors

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go/core/logger"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
)

var log = logger.DefaultLogger()

// MultiDataInterceptor is used for intercepting packed multi data
type MultiDataInterceptor struct {
	marshalizer              marshal.Marshalizer
	factory                  process.InterceptedDataFactory
	processor                process.InterceptorProcessor
	throttler                process.InterceptorThrottler
	broadcastCallbackHandler func(buffToSend []byte)
}

// NewMultiDataInterceptor hooks a new interceptor for packed multi data
func NewMultiDataInterceptor(
	marshalizer marshal.Marshalizer,
	factory process.InterceptedDataFactory,
	processor process.InterceptorProcessor,
	throttler process.InterceptorThrottler,
) (*MultiDataInterceptor, error) {

	if marshalizer == nil {
		return nil, process.ErrNilMarshalizer
	}
	if factory == nil {
		return nil, process.ErrNilInterceptedDataFactory
	}
	if processor == nil {
		return nil, process.ErrNilInterceptedDataProcessor
	}
	if throttler == nil {
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
func (mdi *MultiDataInterceptor) ProcessReceivedMessage(message p2p.MessageP2P) error {
	err := preProcessMesage(mdi.throttler, message)
	if err != nil {
		return err
	}

	multiDataBuff := make([][]byte, 0)
	err = mdi.marshalizer.Unmarshal(&multiDataBuff, message.Data())
	if err != nil {
		mdi.throttler.EndProcess()
		return err
	}
	if len(multiDataBuff) == 0 {
		mdi.throttler.EndProcess()
		return process.ErrNoDataInMessage
	}

	filteredMultiDataBuff := make([][]byte, 0)
	lastErrEncountered := error(nil)
	wgProcess := &sync.WaitGroup{}
	wgProcess.Add(len(multiDataBuff))
	go func() {
		wgProcess.Wait()
		mdi.throttler.EndProcess()
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

		//data is validated, add it to filtered out buff
		filteredMultiDataBuff = append(filteredMultiDataBuff, dataBuff)
		if !interceptedData.IsForMyShard() {
			log.Debug("intercepted data is for other shards")
			wgProcess.Done()
			continue
		}

		go processInterceptedData(mdi.processor, interceptedData, wgProcess)
	}

	var buffToSend []byte
	haveDataForBroadcast := len(filteredMultiDataBuff) > 0 && lastErrEncountered != nil
	if haveDataForBroadcast {
		buffToSend, err = mdi.marshalizer.Marshal(filteredMultiDataBuff)
		if err != nil {
			return err
		}

		if mdi.broadcastCallbackHandler != nil {
			mdi.broadcastCallbackHandler(buffToSend)
		}
	}

	return lastErrEncountered
}

// SetBroadcastCallback sets the callback method to broadcast validated data
func (mdi *MultiDataInterceptor) SetBroadcastCallback(callback func(buffToSend []byte)) {
	mdi.broadcastCallbackHandler = callback
}
