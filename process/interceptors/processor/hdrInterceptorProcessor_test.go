package processor_test

import (
	"errors"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/interceptors/processor"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/stretchr/testify/assert"
)

func createMockHdrArgument() *processor.ArgHdrInterceptorProcessor {
	arg := &processor.ArgHdrInterceptorProcessor{
		Headers:       &mock.CacherStub{},
		HeadersNonces: &mock.Uint64SyncMapCacherStub{},
		HdrValidator:  &mock.HeaderValidatorStub{},
		BlackList:     &mock.BlackListHandlerStub{},
	}

	return arg
}

//------- NewHdrInterceptorProcessor

func TestNewHdrInterceptorProcessor_NilArgumentShouldErr(t *testing.T) {
	t.Parallel()

	hip, err := processor.NewHdrInterceptorProcessor(nil)

	assert.Nil(t, hip)
	assert.Equal(t, process.ErrNilArgumentStruct, err)
}

func TestNewHdrInterceptorProcessor_NilHeadersShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockHdrArgument()
	arg.Headers = nil
	hip, err := processor.NewHdrInterceptorProcessor(arg)

	assert.Nil(t, hip)
	assert.Equal(t, process.ErrNilCacher, err)
}

func TestNewHdrInterceptorProcessor_NilHeadersNoncesShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockHdrArgument()
	arg.HeadersNonces = nil
	hip, err := processor.NewHdrInterceptorProcessor(arg)

	assert.Nil(t, hip)
	assert.Equal(t, process.ErrNilUint64SyncMapCacher, err)
}

func TestNewHdrInterceptorProcessor_NilValidatorShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockHdrArgument()
	arg.HdrValidator = nil
	hip, err := processor.NewHdrInterceptorProcessor(arg)

	assert.Nil(t, hip)
	assert.Equal(t, process.ErrNilHdrValidator, err)
}

func TestNewHdrInterceptorProcessor_NilBlackListHandlerShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockHdrArgument()
	arg.BlackList = nil
	hip, err := processor.NewHdrInterceptorProcessor(arg)

	assert.Nil(t, hip)
	assert.Equal(t, process.ErrNilBlackListHandler, err)
}

func TestNewHdrInterceptorProcessor_ShouldWork(t *testing.T) {
	t.Parallel()

	arg := createMockHdrArgument()
	hip, err := processor.NewHdrInterceptorProcessor(arg)

	assert.False(t, check.IfNil(hip))
	assert.Nil(t, err)
}

//------- Validate

func TestHdrInterceptorProcessor_ValidateNilHdrShouldErr(t *testing.T) {
	t.Parallel()

	hip, _ := processor.NewHdrInterceptorProcessor(createMockHdrArgument())

	err := hip.Validate(nil)

	assert.Equal(t, process.ErrWrongTypeAssertion, err)
}

func TestHdrInterceptorProcessor_ValidateHeaderIsBlackListedShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockHdrArgument()
	arg.HdrValidator = &mock.HeaderValidatorStub{
		HeaderValidForProcessingCalled: func(hdrValidatorHandler process.HdrValidatorHandler) error {
			return nil
		},
	}
	arg.BlackList = &mock.BlackListHandlerStub{
		HasCalled: func(key string) bool {
			return true
		},
	}
	hip, _ := processor.NewHdrInterceptorProcessor(arg)

	hdrInterceptedData := &struct {
		mock.InterceptedDataStub
		mock.GetHdrHandlerStub
	}{
		InterceptedDataStub: mock.InterceptedDataStub{
			HashCalled: func() []byte {
				return make([]byte, 0)
			},
		},
	}
	err := hip.Validate(hdrInterceptedData)

	assert.Equal(t, process.ErrHeaderIsBlackListed, err)
}

func TestHdrInterceptorProcessor_ValidateReturnsErrFromIsValid(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("expected error")
	arg := createMockHdrArgument()
	arg.HdrValidator = &mock.HeaderValidatorStub{
		HeaderValidForProcessingCalled: func(hdrValidatorHandler process.HdrValidatorHandler) error {
			return expectedErr
		},
	}
	arg.BlackList = &mock.BlackListHandlerStub{}
	hip, _ := processor.NewHdrInterceptorProcessor(arg)

	hdrInterceptedData := &struct {
		mock.InterceptedDataStub
		mock.GetHdrHandlerStub
	}{
		InterceptedDataStub: mock.InterceptedDataStub{
			HashCalled: func() []byte {
				return make([]byte, 0)
			},
		},
	}
	err := hip.Validate(hdrInterceptedData)

	assert.Equal(t, expectedErr, err)
}

//------- Save

func TestHdrInterceptorProcessor_SaveNilDataShouldErr(t *testing.T) {
	t.Parallel()

	hip, _ := processor.NewHdrInterceptorProcessor(createMockHdrArgument())

	err := hip.Save(nil)

	assert.Equal(t, process.ErrWrongTypeAssertion, err)
}

func TestHdrInterceptorProcessor_SaveShouldWork(t *testing.T) {
	t.Parallel()

	hdrInterceptedData := &struct {
		mock.InterceptedDataStub
		mock.GetHdrHandlerStub
	}{
		InterceptedDataStub: mock.InterceptedDataStub{
			HashCalled: func() []byte {
				return []byte("hash")
			},
		},
		GetHdrHandlerStub: mock.GetHdrHandlerStub{
			HeaderHandlerCalled: func() data.HeaderHandler {
				return &mock.HeaderHandlerStub{}
			},
		},
	}

	wasAddedHeaders := false
	wasMergedHeadersNonces := false

	arg := createMockHdrArgument()
	arg.Headers = &mock.CacherStub{
		HasOrAddCalled: func(key []byte, value interface{}) (ok, evicted bool) {
			wasAddedHeaders = true

			return true, true
		},
	}
	arg.HeadersNonces = &mock.Uint64SyncMapCacherStub{
		MergeCalled: func(nonce uint64, src dataRetriever.ShardIdHashMap) {
			wasMergedHeadersNonces = true
		},
	}

	hip, _ := processor.NewHdrInterceptorProcessor(arg)

	err := hip.Save(hdrInterceptedData)

	assert.Nil(t, err)
	assert.True(t, wasAddedHeaders && wasMergedHeadersNonces)
}

//------- IsInterfaceNil

func TestHdrInterceptorProcessor_IsInterfaceNil(t *testing.T) {
	t.Parallel()

	var hip *processor.HdrInterceptorProcessor

	assert.True(t, check.IfNil(hip))
}
