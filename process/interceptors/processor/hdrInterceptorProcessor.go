package processor

import (
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/process"
)

// HdrInterceptorProcessor is the processor used when intercepting headers
// (shard headers, meta headers) structs which satisfy HeaderHandler interface.
type HdrInterceptorProcessor struct {
	headers      dataRetriever.HeadersPool
	hdrValidator process.HeaderValidator
	blackList    process.BlackListHandler
}

// NewHdrInterceptorProcessor creates a new TxInterceptorProcessor instance
func NewHdrInterceptorProcessor(argument *ArgHdrInterceptorProcessor) (*HdrInterceptorProcessor, error) {
	if argument == nil {
		return nil, process.ErrNilArgumentStruct
	}
	if check.IfNil(argument.Headers) {
		return nil, process.ErrNilCacher
	}
	if check.IfNil(argument.HdrValidator) {
		return nil, process.ErrNilHdrValidator
	}
	if check.IfNil(argument.BlackList) {
		return nil, process.ErrNilBlackListHandler
	}

	return &HdrInterceptorProcessor{
		headers:      argument.Headers,
		hdrValidator: argument.HdrValidator,
		blackList:    argument.BlackList,
	}, nil
}

// Validate checks if the intercepted data can be processed
func (hip *HdrInterceptorProcessor) Validate(data process.InterceptedData) error {
	interceptedHdr, ok := data.(process.HdrValidatorHandler)
	if !ok {
		return process.ErrWrongTypeAssertion
	}

	hip.blackList.Sweep()
	isBlackListed := hip.blackList.Has(string(interceptedHdr.Hash()))
	if isBlackListed {
		return process.ErrHeaderIsBlackListed
	}

	return hip.hdrValidator.HeaderValidForProcessing(interceptedHdr)
}

// Save will save the received data into the headers cacher as hash<->[plain header structure]
// and in headersNonces as nonce<->hash
func (hip *HdrInterceptorProcessor) Save(data process.InterceptedData) error {
	interceptedHdr, ok := data.(process.HdrValidatorHandler)
	if !ok {
		return process.ErrWrongTypeAssertion
	}

	hip.headers.AddHeader(interceptedHdr.Hash(), interceptedHdr.HeaderHandler())

	return nil
}

// SignalEndOfProcessing signals the end of processing
func (hip *HdrInterceptorProcessor) SignalEndOfProcessing(data []process.InterceptedData) {
}

// IsInterfaceNil returns true if there is no value under the interface
func (hip *HdrInterceptorProcessor) IsInterfaceNil() bool {
	return hip == nil
}
