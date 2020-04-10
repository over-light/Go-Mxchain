package smartContract

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/builtInFunctions"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateEmptyByteSlice(size int) []byte {
	buff := make([]byte, size)

	return buff
}

func createAccounts(tx *transaction.Transaction) (state.UserAccountHandler, state.UserAccountHandler) {
	acntSrc, _ := state.NewUserAccount(mock.NewAddressMock(tx.SndAddr))
	acntSrc.Balance = acntSrc.Balance.Add(acntSrc.Balance, tx.Value)
	totalFee := big.NewInt(0)
	totalFee = totalFee.Mul(big.NewInt(int64(tx.GasLimit)), big.NewInt(int64(tx.GasPrice)))
	acntSrc.Balance.Set(acntSrc.Balance.Add(acntSrc.Balance, totalFee))

	acntDst, _ := state.NewUserAccount(mock.NewAddressMock(tx.RcvAddr))

	return acntSrc, acntDst
}

func createMockSmartContractProcessorArguments() ArgsNewSmartContractProcessor {
	return ArgsNewSmartContractProcessor{
		VmContainer:     &mock.VMContainerMock{},
		ArgsParser:      &mock.ArgumentParserMock{},
		Hasher:          &mock.HasherMock{},
		Marshalizer:     &mock.MarshalizerMock{},
		AccountsDB:      &mock.AccountsStub{},
		TempAccounts:    &mock.TemporaryAccountsHandlerMock{},
		AdrConv:         &mock.AddressConverterMock{},
		Coordinator:     mock.NewMultiShardsCoordinatorMock(5),
		ScrForwarder:    &mock.IntermediateTransactionHandlerMock{},
		TxFeeHandler:    &mock.FeeAccumulatorStub{},
		TxLogsProcessor: &mock.TxLogsProcessorStub{},
		EconomicsFee: &mock.FeeHandlerStub{
			DeveloperPercentageCalled: func() float64 {
				return 0.0
			},
		},
		TxTypeHandler: &mock.TxTypeHandlerMock{},
		GasHandler: &mock.GasHandlerMock{
			SetGasRefundedCalled: func(gasRefunded uint64, hash []byte) {},
		},
		BuiltInFunctions: builtInFunctions.NewBuiltInFunctionContainer(),
	}
}

func TestNewSmartContractProcessorNilVM(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = nil
	sc, err := NewSmartContractProcessor(arguments)

	require.Nil(t, sc)
	require.Equal(t, process.ErrNoVM, err)
}

func TestNewSmartContractProcessorNilArgsParser(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	arguments.ArgsParser = nil
	sc, err := NewSmartContractProcessor(arguments)

	require.Nil(t, sc)
	require.Equal(t, process.ErrNilArgumentParser, err)
}

func TestNewSmartContractProcessorNilHasher(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	arguments.Hasher = nil
	sc, err := NewSmartContractProcessor(arguments)

	require.Nil(t, sc)
	require.Equal(t, process.ErrNilHasher, err)
}

func TestNewSmartContractProcessorNilMarshalizer(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	arguments.Marshalizer = nil
	sc, err := NewSmartContractProcessor(arguments)

	require.Nil(t, sc)
	require.Equal(t, process.ErrNilMarshalizer, err)
}

func TestNewSmartContractProcessorNilAccountsDB(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	arguments.AccountsDB = nil
	sc, err := NewSmartContractProcessor(arguments)

	require.Nil(t, sc)
	require.Equal(t, process.ErrNilAccountsAdapter, err)
}

func TestNewSmartContractProcessorNilAdrConv(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	arguments.AdrConv = nil
	sc, err := NewSmartContractProcessor(arguments)

	require.Nil(t, sc)
	require.Equal(t, process.ErrNilAddressConverter, err)
}

func TestNewSmartContractProcessorNilShardCoordinator(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	arguments.Coordinator = nil
	sc, err := NewSmartContractProcessor(arguments)

	require.Nil(t, sc)
	require.Equal(t, process.ErrNilShardCoordinator, err)
}

func TestNewSmartContractProcessorNilFakeAccountsHandler(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	arguments.TempAccounts = nil
	sc, err := NewSmartContractProcessor(arguments)

	require.Nil(t, sc)
	require.Equal(t, process.ErrNilTemporaryAccountsHandler, err)
}

func TestNewSmartContractProcessor_NilIntermediateMock(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	arguments.ScrForwarder = nil
	sc, err := NewSmartContractProcessor(arguments)

	require.Nil(t, sc)
	require.Equal(t, process.ErrNilIntermediateTransactionHandler, err)
}

func TestNewSmartContractProcessor_ErrNilUnsignedTxHandlerMock(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	arguments.TxFeeHandler = nil
	sc, err := NewSmartContractProcessor(arguments)

	require.Nil(t, sc)
	require.Equal(t, process.ErrNilUnsignedTxHandler, err)
}

func TestNewSmartContractProcessor_ErrErrNilGasHandlerMock(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	arguments.GasHandler = nil
	sc, err := NewSmartContractProcessor(arguments)

	require.Nil(t, sc)
	require.Equal(t, process.ErrNilGasHandler, err)
}

func TestNewSmartContractProcessor(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	sc, err := NewSmartContractProcessor(arguments)

	require.NotNil(t, sc)
	require.Nil(t, err)
	require.False(t, sc.IsInterfaceNil())
}

func TestScProcessor_DeploySmartContractBadParse(t *testing.T) {
	t.Parallel()

	addrConverter := &mock.AddressConverterMock{}

	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.AdrConv = addrConverter
	arguments.VmContainer = &mock.VMContainerMock{}
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = generateEmptyByteSlice(addrConverter.AddressLen())
	tx.Data = []byte("data")
	tx.Value = big.NewInt(45)
	acntSrc, _ := createAccounts(tx)

	parseError := fmt.Errorf("fooError")
	argParser.ParseDataCalled = func(data string) error {
		return parseError
	}

	_ = sc.DeploySmartContract(tx, acntSrc)
	require.Equal(t, parseError, GetLatestTestError(sc))
}

func TestScProcessor_DeploySmartContractRunError(t *testing.T) {
	t.Parallel()

	addrConverter := &mock.AddressConverterMock{}
	vmContainer := &mock.VMContainerMock{}
	argParser := vmcommon.NewAtArgumentParser()
	arguments := createMockSmartContractProcessorArguments()
	arguments.AdrConv = addrConverter
	arguments.VmContainer = vmContainer
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = generateEmptyByteSlice(addrConverter.AddressLen())
	tx.Data = []byte("abba@0500@0000")
	tx.Value = big.NewInt(45)
	acntSrc, _ := createAccounts(tx)

	vm := &mock.VMExecutionHandlerStub{}

	createError := fmt.Errorf("fooError")
	vm.RunSmartContractCreateCalled = func(input *vmcommon.ContractCreateInput) (output *vmcommon.VMOutput, e error) {
		return nil, createError
	}

	vmContainer.GetCalled = func(key []byte) (handler vmcommon.VMExecutionHandler, e error) {
		return vm, nil
	}

	_ = sc.DeploySmartContract(tx, acntSrc)
	require.Equal(t, createError, GetLatestTestError(sc))
}

func TestScProcessor_DeploySmartContractWrongTx(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")
	tx.Data = []byte("data")
	tx.Value = big.NewInt(45)
	acntSrc, _ := createAccounts(tx)

	err = sc.DeploySmartContract(tx, acntSrc)
	require.Equal(t, process.ErrWrongTransaction, err)
}

func TestScProcessor_DeploySmartContract(t *testing.T) {
	t.Parallel()

	addrConverter := &mock.AddressConverterMock{}
	vm := &mock.VMContainerMock{}
	argParser := vmcommon.NewAtArgumentParser()
	accntState := &mock.AccountsStub{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.AdrConv = addrConverter
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	arguments.AccountsDB = accntState
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = generateEmptyByteSlice(addrConverter.AddressLen())
	tx.Data = []byte("abba@0500@0000")
	tx.Value = big.NewInt(0)
	acntSrc, _ := createAccounts(tx)

	accntState.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return acntSrc, nil
	}

	err = sc.DeploySmartContract(tx, acntSrc)
	require.Nil(t, err)
	require.Nil(t, GetLatestTestError(sc))
}

func TestScProcessor_ExecuteSmartContractTransactionNilTx(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")
	tx.Data = []byte("data")
	tx.Value = big.NewInt(45)
	acntSrc, acntDst := createAccounts(tx)

	err = sc.ExecuteSmartContractTransaction(nil, acntSrc, acntDst)
	require.Equal(t, process.ErrNilTransaction, err)
}

func TestScProcessor_ExecuteSmartContractTransactionNilAccount(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")
	tx.Data = []byte("data")
	tx.Value = big.NewInt(45)
	acntSrc, acntDst := createAccounts(tx)

	err = sc.ExecuteSmartContractTransaction(tx, acntSrc, nil)
	require.Equal(t, process.ErrNilSCDestAccount, err)

	acntDst.SetCode(nil)
	err = sc.ExecuteSmartContractTransaction(tx, acntSrc, acntDst)
	require.Nil(t, err)

	acntDst = nil
	err = sc.ExecuteSmartContractTransaction(tx, acntSrc, acntDst)
	require.Equal(t, process.ErrNilSCDestAccount, err)
}

func TestScProcessor_ExecuteSmartContractTransactionBadParser(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")
	tx.Data = []byte("data")
	tx.Value = big.NewInt(45)
	acntSrc, acntDst := createAccounts(tx)

	acntDst.SetCode([]byte("code"))
	tmpError := errors.New("error")
	called := false
	argParser.ParseDataCalled = func(data string) error {
		called = true
		return tmpError
	}
	err = sc.ExecuteSmartContractTransaction(tx, acntSrc, acntDst)
	require.True(t, called)
	require.Nil(t, err)
}

func TestScProcessor_ExecuteSmartContractTransactionVMRunError(t *testing.T) {
	t.Parallel()

	vmContainer := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vmContainer
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST0000000")
	tx.Data = []byte("data")
	tx.Value = big.NewInt(45)
	acntSrc, acntDst := createAccounts(tx)

	acntDst.SetCode([]byte("code"))
	tmpError := errors.New("error")
	vm := &mock.VMExecutionHandlerStub{}
	called := false
	vm.RunSmartContractCallCalled = func(input *vmcommon.ContractCallInput) (output *vmcommon.VMOutput, e error) {
		called = true
		return nil, tmpError
	}
	vmContainer.GetCalled = func(key []byte) (handler vmcommon.VMExecutionHandler, e error) {
		return vm, nil
	}

	err = sc.ExecuteSmartContractTransaction(tx, acntSrc, acntDst)
	require.True(t, called)
	require.Nil(t, err)
}

func TestScProcessor_ExecuteSmartContractTransaction(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	accntState := &mock.AccountsStub{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	arguments.AccountsDB = accntState
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST0000000")
	tx.Data = []byte("data")
	tx.Value = big.NewInt(0)
	acntSrc, acntDst := createAccounts(tx)

	accntState.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return acntSrc, nil
	}

	acntDst.SetCode([]byte("code"))
	err = sc.ExecuteSmartContractTransaction(tx, acntSrc, acntDst)
	require.Nil(t, err)
}

func TestScProcessor_ExecuteSmartContractTransactionSaveLogsIsCalled(t *testing.T) {
	t.Parallel()

	slCalled := false

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	accntState := &mock.AccountsStub{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	arguments.AccountsDB = accntState
	arguments.TxLogsProcessor = &mock.TxLogsProcessorStub{
		SaveLogCalled: func(txHash []byte, tx data.TransactionHandler, vmLogs []*vmcommon.LogEntry) error {
			slCalled = true
			return nil
		},
	}
	sc, err := NewSmartContractProcessor(arguments)
	assert.NotNil(t, sc)
	assert.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST0000000")
	tx.Data = []byte("data")
	tx.Value = big.NewInt(0)
	acntSrc, acntDst := createAccounts(tx)

	accntState.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return acntSrc, nil
	}

	acntDst.SetCode([]byte("code"))
	err = sc.ExecuteSmartContractTransaction(tx, acntSrc, acntDst)
	assert.True(t, slCalled)
}

func TestScProcessor_CreateVMCallInputWrongCode(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")
	tx.Data = []byte("data")
	tx.Value = big.NewInt(45)

	tmpError := errors.New("error")
	argParser.GetFunctionCalled = func() (s string, e error) {
		return "", tmpError
	}
	input, err := sc.createVMCallInput(tx)
	require.Nil(t, input)
	require.Equal(t, tmpError, err)
}

func TestScProcessor_CreateVMCallInput(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")
	tx.Data = []byte("data")
	tx.Value = big.NewInt(45)

	input, err := sc.createVMCallInput(tx)
	require.NotNil(t, input)
	require.Nil(t, err)
}

func TestScProcessor_CreateVMDeployBadCode(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Data = nil
	tx.Value = big.NewInt(0)

	badCodeError := errors.New("fooError")
	argParser.GetCodeDecodedCalled = func() (code []byte, e error) {
		return nil, badCodeError
	}

	input, vmType, err := sc.createVMDeployInput(tx)
	require.Nil(t, vmType)
	require.Nil(t, input)
	require.Equal(t, badCodeError, err)
}

func TestScProcessor_CreateVMDeployInput(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")
	tx.Data = []byte("foobar")
	tx.Value = big.NewInt(45)

	expectedVMType := []byte{5, 6}
	expectedCodeMetadata := vmcommon.CodeMetadata{Upgradeable: true}
	argParser.GetVMTypeCalled = func() ([]byte, error) {
		return expectedVMType, nil
	}
	argParser.GetCodeMetadataCalled = func() (vmcommon.CodeMetadata, error) {
		return expectedCodeMetadata, nil
	}

	input, vmType, err := sc.createVMDeployInput(tx)
	require.NotNil(t, input)
	require.Equal(t, vmcommon.DirectCall, input.CallType)
	require.True(t, bytes.Equal(expectedVMType, vmType))
	require.Equal(t, expectedCodeMetadata.ToBytes(), input.ContractCodeMetadata)
	require.Nil(t, err)
}

func TestScProcessor_CreateVMDeployInputNotEnoughArguments(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := vmcommon.NewAtArgumentParser()
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")
	tx.Data = []byte("data@0000")
	tx.Value = big.NewInt(45)

	input, vmType, err := sc.createVMDeployInput(tx)
	require.Nil(t, input)
	require.Nil(t, vmType)
	require.Equal(t, vmcommon.ErrInvalidDeployArguments, err)
}

func TestScProcessor_CreateVMDeployInputWrongArgument(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")
	tx.Data = []byte("data")
	tx.Value = big.NewInt(45)

	tmpError := errors.New("fooError")
	argParser.GetConstructorArgumentsCalled = func() (ints [][]byte, e error) {
		return nil, tmpError
	}
	input, vmType, err := sc.createVMDeployInput(tx)
	require.Nil(t, input)
	require.Nil(t, vmType)
	require.Equal(t, tmpError, err)
}

func TestScProcessor_InitializeVMInputFromTx_ShouldErrNotEnoughGas(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	arguments.EconomicsFee = &mock.FeeHandlerStub{
		ComputeGasLimitCalled: func(tx process.TransactionWithFeeHandler) uint64 {
			return 1000
		},
	}
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")
	tx.Data = []byte("data")
	tx.Value = big.NewInt(45)
	tx.GasLimit = 100

	vmInput := &vmcommon.VMInput{}
	err = sc.initializeVMInputFromTx(vmInput, tx)
	require.Equal(t, process.ErrNotEnoughGas, err)
}

func TestScProcessor_InitializeVMInputFromTx(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")
	tx.Data = []byte("data")
	tx.Value = big.NewInt(45)

	vmInput := &vmcommon.VMInput{}
	err = sc.initializeVMInputFromTx(vmInput, tx)
	require.Nil(t, err)
}

func createAccountsAndTransaction() (state.UserAccountHandler, state.UserAccountHandler, *transaction.Transaction) {
	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")
	tx.Data = []byte("data")
	tx.Value = big.NewInt(45)

	acntSrc, acntDst := createAccounts(tx)

	return acntSrc.(state.UserAccountHandler), acntDst.(state.UserAccountHandler), tx
}

func TestScProcessor_processVMOutputNilVMOutput(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	acntSrc, _, tx := createAccountsAndTransaction()

	txHash, _ := core.CalculateHash(arguments.Marshalizer, arguments.Hasher, tx)
	_, _, err = sc.processVMOutput(nil, txHash, tx, acntSrc, vmcommon.DirectCall)
	require.Equal(t, process.ErrNilVMOutput, err)
}

func TestScProcessor_processVMOutputNilTx(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	acntSrc, _, _ := createAccountsAndTransaction()

	vmOutput := &vmcommon.VMOutput{}
	_, _, err = sc.processVMOutput(vmOutput, nil, nil, acntSrc, vmcommon.DirectCall)
	require.Equal(t, process.ErrNilTransaction, err)
}

func TestScProcessor_processVMOutputNilSndAcc(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{Value: big.NewInt(0)}

	vmOutput := &vmcommon.VMOutput{
		GasRefund:    big.NewInt(0),
		GasRemaining: 0,
	}
	txHash, _ := core.CalculateHash(arguments.Marshalizer, arguments.Hasher, tx)
	_, _, err = sc.processVMOutput(vmOutput, txHash, tx, nil, vmcommon.DirectCall)
	require.Nil(t, err)
}

func TestScProcessor_processVMOutputNilDstAcc(t *testing.T) {
	t.Parallel()

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	accntState := &mock.AccountsStub{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	arguments.AccountsDB = accntState
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	acntSnd, _, tx := createAccountsAndTransaction()

	vmOutput := &vmcommon.VMOutput{
		GasRefund:    big.NewInt(0),
		GasRemaining: 0,
	}

	accntState.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return acntSnd, nil
	}

	tx.Value = big.NewInt(0)
	txHash, _ := core.CalculateHash(arguments.Marshalizer, arguments.Hasher, tx)
	_, _, err = sc.processVMOutput(vmOutput, txHash, tx, acntSnd, vmcommon.DirectCall)
	require.Nil(t, err)
}

func TestScProcessor_GetAccountFromAddressAccNotFound(t *testing.T) {
	t.Parallel()

	accountsDB := &mock.AccountsStub{}
	accountsDB.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return nil, state.ErrAccNotFound
	}

	addrConv := &mock.AddressConverterMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(2)
	shardCoordinator.ComputeIdCalled = func(address state.AddressContainer) uint32 {
		return shardCoordinator.SelfId()
	}

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	arguments.AccountsDB = accountsDB
	arguments.Coordinator = shardCoordinator
	arguments.AdrConv = addrConv
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	acc, err := sc.getAccountFromAddress([]byte("SRC"))
	require.Nil(t, acc)
	require.Equal(t, state.ErrAccNotFound, err)
}

func TestScProcessor_GetAccountFromAddrFaildAddressConv(t *testing.T) {
	t.Parallel()

	addrConv := &mock.AddressConverterMock{}
	addrConv.Fail = true

	accountsDB := &mock.AccountsStub{}
	getCalled := 0
	accountsDB.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		getCalled++
		return nil, state.ErrAccNotFound
	}

	shardCoordinator := mock.NewMultiShardsCoordinatorMock(2)
	shardCoordinator.ComputeIdCalled = func(address state.AddressContainer) uint32 {
		return shardCoordinator.SelfId()
	}

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	arguments.AccountsDB = accountsDB
	arguments.Coordinator = shardCoordinator
	arguments.AdrConv = addrConv
	arguments.TxLogsProcessor = &mock.TxLogsProcessorStub{}
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	acc, err := sc.getAccountFromAddress([]byte("DST"))
	require.Nil(t, acc)
	require.NotNil(t, err)
	require.Equal(t, 0, getCalled)
}

func TestScProcessor_GetAccountFromAddrFailedGetExistingAccount(t *testing.T) {
	t.Parallel()

	addrConv := &mock.AddressConverterMock{}

	accountsDB := &mock.AccountsStub{}
	getCalled := 0
	accountsDB.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		getCalled++
		return nil, state.ErrAccNotFound
	}

	shardCoordinator := mock.NewMultiShardsCoordinatorMock(2)
	shardCoordinator.ComputeIdCalled = func(address state.AddressContainer) uint32 {
		return shardCoordinator.SelfId()
	}

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	arguments.AccountsDB = accountsDB
	arguments.Coordinator = shardCoordinator
	arguments.AdrConv = addrConv
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	acc, err := sc.getAccountFromAddress([]byte("DST"))
	require.Nil(t, acc)
	require.Equal(t, state.ErrAccNotFound, err)
	require.Equal(t, 1, getCalled)
}

func TestScProcessor_GetAccountFromAddrAccNotInShard(t *testing.T) {
	t.Parallel()

	addrConv := &mock.AddressConverterMock{}

	accountsDB := &mock.AccountsStub{}
	getCalled := 0
	accountsDB.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		getCalled++
		return nil, state.ErrAccNotFound
	}

	shardCoordinator := mock.NewMultiShardsCoordinatorMock(2)
	shardCoordinator.ComputeIdCalled = func(address state.AddressContainer) uint32 {
		return shardCoordinator.SelfId() + 1
	}

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	arguments.AccountsDB = accountsDB
	arguments.Coordinator = shardCoordinator
	arguments.AdrConv = addrConv
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	acc, err := sc.getAccountFromAddress([]byte("DST"))
	require.Nil(t, acc)
	require.Nil(t, err)
	require.Equal(t, 0, getCalled)
}

func TestScProcessor_GetAccountFromAddr(t *testing.T) {
	t.Parallel()

	addrConv := &mock.AddressConverterMock{}

	accountsDB := &mock.AccountsStub{}
	getCalled := 0
	accountsDB.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		getCalled++
		acc, _ := state.NewUserAccount(addressContainer)
		return acc, nil
	}

	shardCoordinator := mock.NewMultiShardsCoordinatorMock(2)
	shardCoordinator.ComputeIdCalled = func(address state.AddressContainer) uint32 {
		return shardCoordinator.SelfId()
	}

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	arguments.AccountsDB = accountsDB
	arguments.Coordinator = shardCoordinator
	arguments.AdrConv = addrConv
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	acc, err := sc.getAccountFromAddress([]byte("DST"))
	require.NotNil(t, acc)
	require.Nil(t, err)
	require.Equal(t, 1, getCalled)
}

func TestScProcessor_DeleteAccountsFailedAtRemove(t *testing.T) {
	t.Parallel()
	addrConv := &mock.AddressConverterMock{}

	accountsDB := &mock.AccountsStub{}
	removeCalled := 0
	accountsDB.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return nil, state.ErrAccNotFound
	}
	accountsDB.RemoveAccountCalled = func(addressContainer state.AddressContainer) error {
		removeCalled++
		return state.ErrAccNotFound
	}

	shardCoordinator := mock.NewMultiShardsCoordinatorMock(2)
	shardCoordinator.ComputeIdCalled = func(address state.AddressContainer) uint32 {
		return shardCoordinator.SelfId()
	}

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	arguments.AccountsDB = accountsDB
	arguments.Coordinator = shardCoordinator
	arguments.AdrConv = addrConv
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	deletedAccounts := make([][]byte, 0)
	deletedAccounts = append(deletedAccounts, []byte("acc1"), []byte("acc2"), []byte("acc3"))
	err = sc.deleteAccounts(deletedAccounts)
	require.Equal(t, state.ErrAccNotFound, err)
	require.Equal(t, 0, removeCalled)
}

func TestScProcessor_DeleteAccountsNotInShard(t *testing.T) {
	t.Parallel()
	addrConv := &mock.AddressConverterMock{}

	accountsDB := &mock.AccountsStub{}
	removeCalled := 0
	accountsDB.RemoveAccountCalled = func(addressContainer state.AddressContainer) error {
		removeCalled++
		return state.ErrAccNotFound
	}

	shardCoordinator := mock.NewMultiShardsCoordinatorMock(2)
	computeIdCalled := 0
	shardCoordinator.ComputeIdCalled = func(address state.AddressContainer) uint32 {
		computeIdCalled++
		return shardCoordinator.SelfId() + 1
	}

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	arguments.AccountsDB = accountsDB
	arguments.Coordinator = shardCoordinator
	arguments.AdrConv = addrConv
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	deletedAccounts := make([][]byte, 0)
	deletedAccounts = append(deletedAccounts, []byte("acc1"), []byte("acc2"), []byte("acc3"))
	err = sc.deleteAccounts(deletedAccounts)
	require.Nil(t, err)
	require.Equal(t, 0, removeCalled)
	require.Equal(t, len(deletedAccounts), computeIdCalled)
}

func TestScProcessor_DeleteAccountsInShard(t *testing.T) {
	t.Parallel()

	addrConv := &mock.AddressConverterMock{}
	accountsDB := &mock.AccountsStub{}
	removeCalled := 0
	accountsDB.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		acc, _ := state.NewUserAccount(addressContainer)
		return acc, nil
	}
	accountsDB.RemoveAccountCalled = func(addressContainer state.AddressContainer) error {
		removeCalled++
		return nil
	}

	shardCoordinator := mock.NewMultiShardsCoordinatorMock(2)
	computeIdCalled := 0
	shardCoordinator.ComputeIdCalled = func(address state.AddressContainer) uint32 {
		computeIdCalled++
		return shardCoordinator.SelfId()
	}

	vm := &mock.VMContainerMock{}
	argParser := &mock.ArgumentParserMock{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.VmContainer = vm
	arguments.ArgsParser = argParser
	arguments.AccountsDB = accountsDB
	arguments.Coordinator = shardCoordinator
	arguments.AdrConv = addrConv
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	deletedAccounts := make([][]byte, 0)
	deletedAccounts = append(deletedAccounts, []byte("acc1"), []byte("acc2"), []byte("acc3"))
	err = sc.deleteAccounts(deletedAccounts)
	require.Nil(t, err)
	require.Equal(t, len(deletedAccounts), removeCalled)
	require.Equal(t, len(deletedAccounts), computeIdCalled)
}

func TestScProcessor_ProcessSCPaymentAccNotInShardShouldNotReturnError(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	sc, err := NewSmartContractProcessor(arguments)

	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 1
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")

	tx.Value = big.NewInt(45)
	tx.GasPrice = 10
	tx.GasLimit = 10

	err = sc.processSCPayment(tx, nil)
	require.Nil(t, err)
}

func TestScProcessor_ProcessSCPaymentNotEnoughBalance(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	sc, err := NewSmartContractProcessor(arguments)

	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 1
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")

	tx.Value = big.NewInt(45)
	tx.GasPrice = 10
	tx.GasLimit = 15

	acntSrc, _ := state.NewUserAccount(mock.NewAddressMock(tx.SndAddr))
	_ = acntSrc.AddToBalance(big.NewInt(45))

	currBalance := acntSrc.GetBalance().Uint64()

	err = sc.processSCPayment(tx, acntSrc)
	require.Equal(t, process.ErrInsufficientFunds, err)
	require.Equal(t, currBalance, acntSrc.GetBalance().Uint64())
}

func TestScProcessor_ProcessSCPayment(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	sc, err := NewSmartContractProcessor(arguments)

	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")

	tx.Value = big.NewInt(45)
	tx.GasPrice = 10
	tx.GasLimit = 10

	acntSrc, _ := createAccounts(tx)
	currBalance := acntSrc.(state.UserAccountHandler).GetBalance().Uint64()
	modifiedBalance := currBalance - tx.Value.Uint64() - tx.GasLimit*tx.GasLimit

	err = sc.processSCPayment(tx, acntSrc)
	require.Nil(t, err)
	require.Equal(t, modifiedBalance, acntSrc.(state.UserAccountHandler).GetBalance().Uint64())
}

func TestScProcessor_RefundGasToSenderNilAndZeroRefund(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	sc, err := NewSmartContractProcessor(arguments)

	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 1
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")

	tx.Value = big.NewInt(45)
	tx.GasPrice = 10
	tx.GasLimit = 10

	txHash := []byte("txHash")

	acntSrc, _ := createAccounts(tx)
	currBalance := acntSrc.(state.UserAccountHandler).GetBalance().Uint64()
	vmOutput := &vmcommon.VMOutput{GasRemaining: 0, GasRefund: big.NewInt(0)}
	_, _, err = sc.createSCRForSender(
		vmOutput.GasRefund,
		vmOutput.GasRemaining,
		vmOutput.ReturnCode,
		vmOutput.ReturnData,
		tx,
		txHash,
		acntSrc,
		vmcommon.DirectCall,
	)
	require.Nil(t, err)
	require.Equal(t, currBalance, acntSrc.(state.UserAccountHandler).GetBalance().Uint64())
}

func TestScProcessor_RefundGasToSenderAccNotInShard(t *testing.T) {
	t.Parallel()

	arguments := createMockSmartContractProcessorArguments()
	sc, err := NewSmartContractProcessor(arguments)

	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 1
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")

	tx.Value = big.NewInt(45)
	tx.GasPrice = 10
	tx.GasLimit = 10
	txHash := []byte("txHash")
	vmOutput := &vmcommon.VMOutput{GasRemaining: 0, GasRefund: big.NewInt(10)}
	sctx, consumed, err := sc.createSCRForSender(
		vmOutput.GasRefund,
		vmOutput.GasRemaining,
		vmOutput.ReturnCode,
		vmOutput.ReturnData,
		tx,
		txHash,
		nil,
		vmcommon.DirectCall,
	)
	require.Nil(t, err)
	require.NotNil(t, sctx)
	require.Equal(t, 0, consumed.Cmp(big.NewInt(0).SetUint64(tx.GasPrice*tx.GasLimit)))

	vmOutput = &vmcommon.VMOutput{GasRemaining: 0, GasRefund: big.NewInt(10)}
	sctx, consumed, err = sc.createSCRForSender(
		vmOutput.GasRefund,
		vmOutput.GasRemaining,
		vmOutput.ReturnCode,
		vmOutput.ReturnData,
		tx,
		txHash,
		nil,
		vmcommon.DirectCall,
	)
	require.Nil(t, err)
	require.NotNil(t, sctx)
	require.Equal(t, 0, consumed.Cmp(big.NewInt(0).SetUint64(tx.GasPrice*tx.GasLimit)))
}

func TestScProcessor_RefundGasToSender(t *testing.T) {
	t.Parallel()

	minGasPrice := uint64(10)
	arguments := createMockSmartContractProcessorArguments()
	arguments.EconomicsFee = &mock.FeeHandlerStub{MinGasPriceCalled: func() uint64 {
		return minGasPrice
	}}
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 1
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")

	tx.Value = big.NewInt(45)
	tx.GasPrice = 15
	tx.GasLimit = 15
	txHash := []byte("txHash")
	acntSrc, _ := createAccounts(tx)
	currBalance := acntSrc.(state.UserAccountHandler).GetBalance().Uint64()

	refundGas := big.NewInt(10)
	vmOutput := &vmcommon.VMOutput{GasRemaining: 0, GasRefund: refundGas}
	_, _, err = sc.createSCRForSender(
		vmOutput.GasRefund,
		vmOutput.GasRemaining,
		vmOutput.ReturnCode,
		vmOutput.ReturnData,
		tx,
		txHash,
		acntSrc,
		vmcommon.DirectCall,
	)
	require.Nil(t, err)

	totalRefund := refundGas.Uint64() * minGasPrice
	require.Equal(t, currBalance+totalRefund, acntSrc.(state.UserAccountHandler).GetBalance().Uint64())
}

func TestScProcessor_processVMOutputNilOutput(t *testing.T) {
	t.Parallel()

	acntSrc, _, tx := createAccountsAndTransaction()

	arguments := createMockSmartContractProcessorArguments()
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)
	txHash, _ := core.CalculateHash(arguments.Marshalizer, arguments.Hasher, tx)
	_, _, err = sc.processVMOutput(nil, txHash, tx, acntSrc, vmcommon.DirectCall)

	require.Equal(t, process.ErrNilVMOutput, err)
}

func TestScProcessor_processVMOutputNilTransaction(t *testing.T) {
	t.Parallel()

	acntSrc, _, _ := createAccountsAndTransaction()

	arguments := createMockSmartContractProcessorArguments()
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	vmOutput := &vmcommon.VMOutput{}
	_, _, err = sc.processVMOutput(vmOutput, nil, nil, acntSrc, vmcommon.DirectCall)

	require.Equal(t, process.ErrNilTransaction, err)
}

func TestScProcessor_processVMOutput(t *testing.T) {
	t.Parallel()

	acntSrc, _, tx := createAccountsAndTransaction()

	accntState := &mock.AccountsStub{}
	arguments := createMockSmartContractProcessorArguments()
	arguments.AccountsDB = accntState
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	vmOutput := &vmcommon.VMOutput{
		GasRefund:    big.NewInt(0),
		GasRemaining: 0,
	}

	accntState.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return acntSrc, nil
	}

	tx.Value = big.NewInt(0)
	txHash, _ := core.CalculateHash(arguments.Marshalizer, arguments.Hasher, tx)
	_, _, err = sc.processVMOutput(vmOutput, txHash, tx, acntSrc, vmcommon.DirectCall)
	require.Nil(t, err)
}

func TestScProcessor_processSCOutputAccounts(t *testing.T) {
	t.Parallel()

	accountsDB := &mock.AccountsStub{}
	fakeAccountsHandler := &mock.TemporaryAccountsHandlerMock{}

	arguments := createMockSmartContractProcessorArguments()
	arguments.AccountsDB = accountsDB
	arguments.TempAccounts = fakeAccountsHandler
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{Value: big.NewInt(0)}
	outputAccounts := make([]*vmcommon.OutputAccount, 0)
	_, err = sc.processSCOutputAccounts(outputAccounts, tx, []byte("hash"))
	require.Nil(t, err)

	outaddress := []byte("newsmartcontract")
	outacc1 := &vmcommon.OutputAccount{}
	outacc1.Address = outaddress
	outacc1.Code = []byte("contract-code")
	outacc1.Nonce = 5
	outacc1.BalanceDelta = big.NewInt(int64(5))
	outputAccounts = append(outputAccounts, outacc1)

	testAddr := mock.NewAddressMock(outaddress)
	testAcc, _ := state.NewUserAccount(testAddr)

	accountsDB.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		if bytes.Equal(addressContainer.Bytes(), testAddr.Bytes()) {
			return testAcc, nil
		}
		return nil, state.ErrAccNotFound
	}

	accountsDB.SaveAccountCalled = func(accountHandler state.AccountHandler) error {
		return nil
	}

	tx.Value = big.NewInt(int64(5))
	_, err = sc.processSCOutputAccounts(outputAccounts, tx, []byte("hash"))
	require.Nil(t, err)

	outacc1.BalanceDelta = nil
	outacc1.Nonce++
	tx.Value = big.NewInt(0)
	_, err = sc.processSCOutputAccounts(outputAccounts, tx, []byte("hash"))
	require.Nil(t, err)

	outacc1.Nonce++
	outacc1.BalanceDelta = big.NewInt(int64(10))
	tx.Value = big.NewInt(int64(10))
	fakeAccountsHandler.TempAccountCalled = func(address []byte) state.AccountHandler {
		fakeAcc, _ := state.NewUserAccount(mock.NewAddressMock(address))
		fakeAcc.Balance = big.NewInt(int64(5))
		return fakeAcc
	}

	currentBalance := testAcc.Balance.Uint64()
	vmOutBalance := outacc1.BalanceDelta.Uint64()
	_, err = sc.processSCOutputAccounts(outputAccounts, tx, []byte("hash"))
	require.Nil(t, err)
	require.Equal(t, currentBalance+vmOutBalance, testAcc.Balance.Uint64())
}

func TestScProcessor_processSCOutputAccountsNotInShard(t *testing.T) {
	t.Parallel()

	accountsDB := &mock.AccountsStub{}
	fakeAccountsHandler := &mock.TemporaryAccountsHandlerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(5)
	arguments := createMockSmartContractProcessorArguments()
	arguments.AccountsDB = accountsDB
	arguments.TempAccounts = fakeAccountsHandler
	arguments.Coordinator = shardCoordinator
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	tx := &transaction.Transaction{Value: big.NewInt(0)}
	outputAccounts := make([]*vmcommon.OutputAccount, 0)
	_, err = sc.processSCOutputAccounts(outputAccounts, tx, []byte("hash"))
	require.Nil(t, err)

	outaddress := []byte("newsmartcontract")
	outacc1 := &vmcommon.OutputAccount{}
	outacc1.Address = outaddress
	outacc1.Code = []byte("contract-code")
	outacc1.Nonce = 5
	outputAccounts = append(outputAccounts, outacc1)

	shardCoordinator.ComputeIdCalled = func(address state.AddressContainer) uint32 {
		return shardCoordinator.SelfId() + 1
	}

	_, err = sc.processSCOutputAccounts(outputAccounts, tx, []byte("hash"))
	require.Nil(t, err)
}

func TestScProcessor_CreateCrossShardTransactions(t *testing.T) {
	t.Parallel()

	testAccounts, _ := state.NewUserAccount(state.NewAddress([]byte("address")))
	accountsDB := &mock.AccountsStub{
		LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, err error) {
			return testAccounts, nil
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			return nil
		},
	}
	fakeAccountsHandler := &mock.TemporaryAccountsHandlerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(5)
	arguments := createMockSmartContractProcessorArguments()
	arguments.AccountsDB = accountsDB
	arguments.TempAccounts = fakeAccountsHandler
	arguments.Coordinator = shardCoordinator
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	outputAccounts := make([]*vmcommon.OutputAccount, 0)
	outaddress := []byte("newsmartcontract")
	outacc1 := &vmcommon.OutputAccount{}
	outacc1.Address = outaddress
	outacc1.Nonce = 0
	outacc1.Balance = big.NewInt(int64(5))
	outacc1.BalanceDelta = big.NewInt(int64(15))
	outputAccounts = append(outputAccounts, outacc1, outacc1, outacc1)

	tx := &transaction.Transaction{}
	tx.Nonce = 1
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")

	tx.Value = big.NewInt(45)
	tx.GasPrice = 10
	tx.GasLimit = 15
	txHash := []byte("txHash")

	scTxs, err := sc.processSCOutputAccounts(outputAccounts, tx, txHash)
	require.Nil(t, err)
	require.Equal(t, len(outputAccounts), len(scTxs))
}

func TestScProcessor_ProcessSmartContractResultNilScr(t *testing.T) {
	t.Parallel()

	accountsDB := &mock.AccountsStub{}
	fakeAccountsHandler := &mock.TemporaryAccountsHandlerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(5)
	arguments := createMockSmartContractProcessorArguments()
	arguments.AccountsDB = accountsDB
	arguments.TempAccounts = fakeAccountsHandler
	arguments.Coordinator = shardCoordinator
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	err = sc.ProcessSmartContractResult(nil)
	require.Equal(t, process.ErrNilSmartContractResult, err)
}

func TestScProcessor_ProcessSmartContractResultErrGetAccount(t *testing.T) {
	t.Parallel()

	accError := errors.New("account get error")
	called := false
	accountsDB := &mock.AccountsStub{LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		called = true
		return nil, accError
	}}
	fakeAccountsHandler := &mock.TemporaryAccountsHandlerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(5)
	arguments := createMockSmartContractProcessorArguments()
	arguments.AccountsDB = accountsDB
	arguments.TempAccounts = fakeAccountsHandler
	arguments.Coordinator = shardCoordinator
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	scr := smartContractResult.SmartContractResult{RcvAddr: []byte("recv address")}
	_ = sc.ProcessSmartContractResult(&scr)
	require.True(t, called)
}

func TestScProcessor_ProcessSmartContractResultAccNotInShard(t *testing.T) {
	t.Parallel()

	accountsDB := &mock.AccountsStub{}
	fakeAccountsHandler := &mock.TemporaryAccountsHandlerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(5)
	arguments := createMockSmartContractProcessorArguments()
	arguments.AccountsDB = accountsDB
	arguments.TempAccounts = fakeAccountsHandler
	arguments.Coordinator = shardCoordinator
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	shardCoordinator.ComputeIdCalled = func(address state.AddressContainer) uint32 {
		return shardCoordinator.CurrentShard + 1
	}
	scr := smartContractResult.SmartContractResult{RcvAddr: []byte("recv address")}
	err = sc.ProcessSmartContractResult(&scr)
	require.Nil(t, err)
}

func TestScProcessor_ProcessSmartContractResultBadAccType(t *testing.T) {
	t.Parallel()

	accountsDB := &mock.AccountsStub{LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return &mock.AccountWrapMock{}, nil
	}}
	fakeAccountsHandler := &mock.TemporaryAccountsHandlerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(5)
	arguments := createMockSmartContractProcessorArguments()
	arguments.AccountsDB = accountsDB
	arguments.TempAccounts = fakeAccountsHandler
	arguments.Coordinator = shardCoordinator
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	scr := smartContractResult.SmartContractResult{RcvAddr: []byte("recv address")}
	err = sc.ProcessSmartContractResult(&scr)
	require.Nil(t, err)
}

func TestScProcessor_ProcessSmartContractResultOutputBalanceNil(t *testing.T) {
	t.Parallel()

	accountsDB := &mock.AccountsStub{
		LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
			return state.NewUserAccount(addressContainer)
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			return nil
		},
	}
	fakeAccountsHandler := &mock.TemporaryAccountsHandlerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(5)
	arguments := createMockSmartContractProcessorArguments()
	arguments.AccountsDB = accountsDB
	arguments.TempAccounts = fakeAccountsHandler
	arguments.Coordinator = shardCoordinator
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	scr := smartContractResult.SmartContractResult{
		RcvAddr: []byte("recv address")}
	err = sc.ProcessSmartContractResult(&scr)
	require.Nil(t, err)
}

func TestScProcessor_ProcessSmartContractResultWithCode(t *testing.T) {
	t.Parallel()

	putCodeCalled := 0
	accountsDB := &mock.AccountsStub{
		LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
			return state.NewUserAccount(addressContainer)
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			putCodeCalled++
			return nil
		},
	}
	fakeAccountsHandler := &mock.TemporaryAccountsHandlerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(5)
	arguments := createMockSmartContractProcessorArguments()
	arguments.AccountsDB = accountsDB
	arguments.TempAccounts = fakeAccountsHandler
	arguments.Coordinator = shardCoordinator
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	scr := smartContractResult.SmartContractResult{
		RcvAddr: []byte("recv address"),
		Code:    []byte("code"),
		Value:   big.NewInt(15),
	}
	err = sc.ProcessSmartContractResult(&scr)
	require.Nil(t, err)
	require.Equal(t, 1, putCodeCalled)
}

func TestScProcessor_ProcessSmartContractResultWithData(t *testing.T) {
	t.Parallel()

	saveAccountCalled := 0
	accountsDB := &mock.AccountsStub{
		LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
			return state.NewUserAccount(addressContainer)
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}
	fakeAccountsHandler := &mock.TemporaryAccountsHandlerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(5)
	arguments := createMockSmartContractProcessorArguments()
	arguments.AccountsDB = accountsDB
	arguments.TempAccounts = fakeAccountsHandler
	arguments.Coordinator = shardCoordinator
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	test := "test"
	result := ""
	sep := "@"
	for i := 0; i < 6; i++ {
		result += test
		result += sep
	}

	scr := smartContractResult.SmartContractResult{
		RcvAddr: []byte("recv address"),
		Data:    []byte(result),
		Value:   big.NewInt(15),
	}
	err = sc.ProcessSmartContractResult(&scr)
	require.Nil(t, err)
	require.Equal(t, 1, saveAccountCalled)
}

func TestScProcessor_ProcessSmartContractResultDeploySCShouldError(t *testing.T) {
	t.Parallel()

	accountsDB := &mock.AccountsStub{
		LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
			return state.NewUserAccount(addressContainer)
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			return nil
		},
	}
	fakeAccountsHandler := &mock.TemporaryAccountsHandlerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(5)
	arguments := createMockSmartContractProcessorArguments()
	arguments.AccountsDB = accountsDB
	arguments.TempAccounts = fakeAccountsHandler
	arguments.Coordinator = shardCoordinator
	arguments.TxTypeHandler = &mock.TxTypeHandlerMock{
		ComputeTransactionTypeCalled: func(tx data.TransactionHandler) (transactionType process.TransactionType, e error) {
			return process.SCDeployment, nil
		},
	}
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	scr := smartContractResult.SmartContractResult{
		RcvAddr: []byte("recv address"),
		Data:    []byte("code@06"),
		Value:   big.NewInt(15),
	}
	err = sc.ProcessSmartContractResult(&scr)
	require.Nil(t, err)
}

func TestScProcessor_ProcessSmartContractResultExecuteSC(t *testing.T) {
	t.Parallel()

	scAddress := []byte("000000000001234567890123456789012")
	dstScAddress, _ := state.NewUserAccount(mock.NewAddressMock(scAddress))
	dstScAddress.SetCode([]byte("code"))
	accountsDB := &mock.AccountsStub{
		LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
			return dstScAddress, nil
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			return nil
		},
	}
	fakeAccountsHandler := &mock.TemporaryAccountsHandlerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(5)
	executeCalled := false
	arguments := createMockSmartContractProcessorArguments()
	arguments.AccountsDB = accountsDB
	arguments.TempAccounts = fakeAccountsHandler
	arguments.Coordinator = shardCoordinator
	arguments.VmContainer = &mock.VMContainerMock{
		GetCalled: func(key []byte) (handler vmcommon.VMExecutionHandler, e error) {
			return &mock.VMExecutionHandlerStub{
				RunSmartContractCallCalled: func(input *vmcommon.ContractCallInput) (output *vmcommon.VMOutput, e error) {
					executeCalled = true
					return nil, nil
				},
			}, nil
		},
	}
	arguments.TxTypeHandler = &mock.TxTypeHandlerMock{
		ComputeTransactionTypeCalled: func(tx data.TransactionHandler) (transactionType process.TransactionType, e error) {
			return process.SCInvoking, nil
		},
	}
	sc, err := NewSmartContractProcessor(arguments)
	require.NotNil(t, sc)
	require.Nil(t, err)

	scr := smartContractResult.SmartContractResult{
		RcvAddr: []byte("recv address"),
		Data:    []byte("code@06"),
		Value:   big.NewInt(15),
	}
	err = sc.ProcessSmartContractResult(&scr)
	require.Nil(t, err)
	require.True(t, executeCalled)
}
