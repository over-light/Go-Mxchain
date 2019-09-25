package smartContract_test

import (
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/process/smartContract"
	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/assert"
)

func TestNewSCDataGetter_NilVmShouldErr(t *testing.T) {
	t.Parallel()

	scdg, err := smartContract.NewSCDataGetter(
		nil,
	)

	assert.Nil(t, scdg)
	assert.Equal(t, process.ErrNoVM, err)
}

func TestNewSCDataGetter_ShouldWork(t *testing.T) {
	t.Parallel()

	scdg, err := smartContract.NewSCDataGetter(
		&mock.VMContainerMock{},
	)

	assert.NotNil(t, scdg)
	assert.Nil(t, err)
}

//------- Get

func TestScDataGetter_GetNilAddressShouldErr(t *testing.T) {
	t.Parallel()

	scdg, _ := smartContract.NewSCDataGetter(
		&mock.VMContainerMock{},
	)

	output, err := scdg.Get(nil, "function")

	assert.Nil(t, output)
	assert.Equal(t, process.ErrNilScAddress, err)
}

func TestScDataGetter_GetEmptyFunctionShouldErr(t *testing.T) {
	t.Parallel()

	scdg, _ := smartContract.NewSCDataGetter(
		&mock.VMContainerMock{},
	)

	output, err := scdg.Get([]byte("sc address"), "")

	assert.Nil(t, output)
	assert.Equal(t, process.ErrEmptyFunctionName, err)
}

func TestScDataGetter_GetShouldReceiveAddrFuncAndArgs(t *testing.T) {
	t.Parallel()

	args := [][]byte{[]byte("arg1"), []byte("arg2")}
	funcName := "called function"
	addressBytes := []byte("address bytes")

	wasCalled := false

	mockVM := &mock.VMExecutionHandlerStub{
		RunSmartContractCallCalled: func(input *vmcommon.ContractCallInput) (output *vmcommon.VMOutput, e error) {
			wasCalled = true
			assert.Equal(t, 2, len(input.Arguments))
			for idx, arg := range args {
				assert.Equal(t, arg, input.Arguments[idx].Bytes())
			}
			assert.Equal(t, addressBytes, input.CallerAddr)
			assert.Equal(t, funcName, input.Function)

			return &vmcommon.VMOutput{
				ReturnCode: vmcommon.Ok,
			}, nil
		},
	}
	scdg, _ := smartContract.NewSCDataGetter(
		&mock.VMContainerMock{
			GetCalled: func(key []byte) (handler vmcommon.VMExecutionHandler, e error) {
				return mockVM, nil
			},
		},
	)

	_, _ = scdg.Get(addressBytes, funcName, args...)
	assert.True(t, wasCalled)
}

func TestScDataGetter_GetReturnsDataShouldRet(t *testing.T) {
	t.Parallel()

	data := []*big.Int{big.NewInt(90), big.NewInt(91)}
	mockVM := &mock.VMExecutionHandlerStub{
		RunSmartContractCallCalled: func(input *vmcommon.ContractCallInput) (output *vmcommon.VMOutput, e error) {
			return &vmcommon.VMOutput{
				ReturnCode: vmcommon.Ok,
				ReturnData: data,
			}, nil
		},
	}

	scdg, _ := smartContract.NewSCDataGetter(
		&mock.VMContainerMock{
			GetCalled: func(key []byte) (handler vmcommon.VMExecutionHandler, e error) {
				return mockVM, nil
			},
		},
	)

	returnedData, err := scdg.Get([]byte("sc address"), "function")

	assert.Nil(t, err)
	assert.Equal(t, data[0].Bytes(), returnedData)
}

func TestScDataGetter_GetReturnsNotOkCodeShouldErr(t *testing.T) {
	t.Parallel()

	mockVM := &mock.VMExecutionHandlerStub{
		RunSmartContractCallCalled: func(input *vmcommon.ContractCallInput) (output *vmcommon.VMOutput, e error) {
			return &vmcommon.VMOutput{
				ReturnCode: vmcommon.OutOfGas,
			}, nil
		},
	}
	scdg, _ := smartContract.NewSCDataGetter(
		&mock.VMContainerMock{
			GetCalled: func(key []byte) (handler vmcommon.VMExecutionHandler, e error) {
				return mockVM, nil
			},
		},
	)

	returnedData, err := scdg.Get([]byte("sc address"), "function")

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "error running vm func")
	assert.Nil(t, returnedData)
}

func TestScDataGetter_GetShouldCallRunScSequentially(t *testing.T) {
	t.Parallel()

	running := int32(0)
	mockVM := &mock.VMExecutionHandlerStub{
		RunSmartContractCallCalled: func(input *vmcommon.ContractCallInput) (output *vmcommon.VMOutput, e error) {
			atomic.AddInt32(&running, 1)
			time.Sleep(time.Millisecond)

			val := atomic.LoadInt32(&running)
			assert.Equal(t, int32(1), val)

			atomic.AddInt32(&running, -1)

			return &vmcommon.VMOutput{
				ReturnCode: vmcommon.Ok,
			}, nil
		},
	}
	scdg, _ := smartContract.NewSCDataGetter(
		&mock.VMContainerMock{
			GetCalled: func(key []byte) (handler vmcommon.VMExecutionHandler, e error) {
				return mockVM, nil
			},
		},
	)

	noOfGoRoutines := 1000
	wg := sync.WaitGroup{}
	wg.Add(noOfGoRoutines)
	for i := 0; i < noOfGoRoutines; i++ {
		go func() {
			_, _ = scdg.Get([]byte("addressaddressaddress"), "function")
			wg.Done()
		}()
	}

	wg.Wait()
}
