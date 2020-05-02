// +build cgo

package vm

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"testing"

	arwenConfig "github.com/ElrondNetwork/arwen-wasm-vm/config"
	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/state/pubkeyConverter"
	dataTransaction "github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/data/trie"
	"github.com/ElrondNetwork/elrond-go/data/trie/evictionWaitingList"
	"github.com/ElrondNetwork/elrond-go/hashing/sha256"
	"github.com/ElrondNetwork/elrond-go/integrationTests/mock"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/coordinator"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/factory/shard"
	"github.com/ElrondNetwork/elrond-go/process/smartContract"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/builtInFunctions"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/hooks"
	"github.com/ElrondNetwork/elrond-go/process/transaction"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/memorydb"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm/iele/elrond/node/endpoint"
	"github.com/stretchr/testify/assert"
)

// TODO: Merge test utilities from this file with the ones from "arwen/utils.go"

var testMarshalizer = &marshal.GogoProtoMarshalizer{}
var testHasher = sha256.Sha256{}
var oneShardCoordinator = mock.NewMultiShardsCoordinatorMock(2)
var pubkeyConv, _ = pubkeyConverter.NewHexPubkeyConverter(32)

var log = logger.GetOrCreate("integrationtests")

// VMTestContext -
type VMTestContext struct {
	TxProcessor    process.TransactionProcessor
	ScProcessor    process.SmartContractProcessor
	Accounts       state.AccountsAdapter
	BlockchainHook vmcommon.BlockchainHook
	VMContainer    process.VirtualMachinesContainer
}

// Close -
func (vmTestContext *VMTestContext) Close() {
	_ = vmTestContext.VMContainer.Close()
}

// GetLatestError -
func (vmTestContext *VMTestContext) GetLatestError() error {
	return smartContract.GetLatestTestError(vmTestContext.ScProcessor)
}

type accountFactory struct {
}

// CreateAccount -
func (af *accountFactory) CreateAccount(address []byte) (state.AccountHandler, error) {
	return state.NewUserAccount(address)
}

// IsInterfaceNil returns true if there is no value under the interface
func (af *accountFactory) IsInterfaceNil() bool {
	return af == nil
}

// CreateEmptyAddress -
func CreateEmptyAddress() []byte {
	buff := make([]byte, testHasher.Size())

	return buff
}

// CreateMemUnit -
func CreateMemUnit() storage.Storer {
	cache, _ := storageUnit.NewCache(storageUnit.LRUCache, 10, 1)

	unit, _ := storageUnit.NewStorageUnit(cache, memorydb.New())
	return unit
}

// CreateInMemoryShardAccountsDB -
func CreateInMemoryShardAccountsDB() *state.AccountsDB {
	marsh := &marshal.GogoProtoMarshalizer{}
	store := CreateMemUnit()
	ewl, _ := evictionWaitingList.NewEvictionWaitingList(100, memorydb.New(), marsh)
	generalCfg := config.TrieStorageManagerConfig{
		PruningBufferLen:   1000,
		SnapshotsBufferLen: 10,
		MaxSnapshots:       2,
	}
	trieStorage, _ := trie.NewTrieStorageManager(
		store,
		marsh,
		testHasher,
		config.DBConfig{
			FilePath:          "TrieStorage",
			Type:              "MemoryDB",
			BatchDelaySeconds: 30,
			MaxBatchSize:      6,
			MaxOpenFiles:      10,
		},
		ewl,
		generalCfg,
	)

	tr, _ := trie.NewTrie(trieStorage, marsh, testHasher)
	adb, _ := state.NewAccountsDB(tr, testHasher, marsh, &accountFactory{})

	return adb
}

// CreateAccount -
func CreateAccount(accnts state.AccountsAdapter, pubKey []byte, nonce uint64, balance *big.Int) ([]byte, error) {
	account, err := accnts.LoadAccount(pubKey)
	if err != nil {
		return nil, err
	}

	account.(state.UserAccountHandler).IncreaseNonce(nonce)
	_ = account.(state.UserAccountHandler).AddToBalance(balance)

	err = accnts.SaveAccount(account)
	if err != nil {
		return nil, err
	}

	hashCreated, err := accnts.Commit()
	if err != nil {
		return nil, err
	}

	return hashCreated, nil
}

// CreateTxProcessorWithOneSCExecutorMockVM -
func CreateTxProcessorWithOneSCExecutorMockVM(accnts state.AccountsAdapter, opGas uint64) process.TransactionProcessor {
	builtInFuncs := builtInFunctions.NewBuiltInFunctionContainer()
	args := hooks.ArgBlockChainHook{
		Accounts:         accnts,
		PubkeyConv:       pubkeyConv,
		StorageService:   &mock.ChainStorerMock{},
		BlockChain:       &mock.BlockChainMock{},
		ShardCoordinator: oneShardCoordinator,
		Marshalizer:      testMarshalizer,
		Uint64Converter:  &mock.Uint64ByteSliceConverterMock{},
		BuiltInFunctions: builtInFuncs,
	}

	blockChainHook, _ := hooks.NewBlockChainHookImpl(args)
	vm, _ := mock.NewOneSCExecutorMockVM(blockChainHook, testHasher)
	vm.GasForOperation = opGas

	vmContainer := &mock.VMContainerMock{
		GetCalled: func(key []byte) (handler vmcommon.VMExecutionHandler, e error) {
			return vm, nil
		}}
	argsParser := vmcommon.NewAtArgumentParser()
	argsTxTypeHandler := coordinator.ArgNewTxTypeHandler{
		PubkeyConverter:  pubkeyConv,
		ShardCoordinator: oneShardCoordinator,
		BuiltInFuncNames: builtInFuncs.Keys(),
		ArgumentParser:   argsParser,
	}
	txTypeHandler, _ := coordinator.NewTxTypeHandler(argsTxTypeHandler)
	gasSchedule := make(map[string]map[string]uint64)
	FillGasMapInternal(gasSchedule, 1)

	argsNewSCProcessor := smartContract.ArgsNewSmartContractProcessor{
		VmContainer:  vmContainer,
		ArgsParser:   argsParser,
		Hasher:       testHasher,
		Marshalizer:  testMarshalizer,
		AccountsDB:   accnts,
		TempAccounts: blockChainHook,
		PubkeyConv:   pubkeyConv,
		Coordinator:  oneShardCoordinator,
		ScrForwarder: &mock.IntermediateTransactionHandlerMock{},
		TxFeeHandler: &mock.UnsignedTxHandlerMock{},
		EconomicsFee: &mock.FeeHandlerStub{
			DeveloperPercentageCalled: func() float64 {
				return 0.0
			},
		},
		TxTypeHandler: txTypeHandler,
		GasHandler: &mock.GasHandlerMock{
			SetGasRefundedCalled: func(gasRefunded uint64, hash []byte) {},
		},
		BuiltInFunctions: blockChainHook.GetBuiltInFunctions(),
		TxLogsProcessor:  &mock.TxLogsProcessorStub{},
	}
	scProcessor, _ := smartContract.NewSmartContractProcessor(argsNewSCProcessor)

	txProcessor, _ := transaction.NewTxProcessor(
		accnts,
		testHasher,
		pubkeyConv,
		testMarshalizer,
		oneShardCoordinator,
		scProcessor,
		&mock.UnsignedTxHandlerMock{},
		txTypeHandler,
		&mock.FeeHandlerStub{},
		&mock.IntermediateTransactionHandlerMock{},
		&mock.IntermediateTransactionHandlerMock{},
	)

	return txProcessor
}

// CreateOneSCExecutorMockVM -
func CreateOneSCExecutorMockVM(accnts state.AccountsAdapter) vmcommon.VMExecutionHandler {
	args := hooks.ArgBlockChainHook{
		Accounts:         accnts,
		PubkeyConv:       pubkeyConv,
		StorageService:   &mock.ChainStorerMock{},
		BlockChain:       &mock.BlockChainMock{},
		ShardCoordinator: oneShardCoordinator,
		Marshalizer:      testMarshalizer,
		Uint64Converter:  &mock.Uint64ByteSliceConverterMock{},
		BuiltInFunctions: builtInFunctions.NewBuiltInFunctionContainer(),
	}
	blockChainHook, _ := hooks.NewBlockChainHookImpl(args)
	vm, _ := mock.NewOneSCExecutorMockVM(blockChainHook, testHasher)

	return vm
}

func createAndAddIeleVM(
	vmContainer process.VirtualMachinesContainer,
	blockChainHook vmcommon.BlockchainHook,
) {
	cryptoHook := hooks.NewVMCryptoHook()
	ieleVM := endpoint.NewElrondIeleVM(factory.IELEVirtualMachine, endpoint.ElrondTestnet, blockChainHook, cryptoHook)
	_ = vmContainer.Add(factory.IELEVirtualMachine, ieleVM)
}

// CreateVMAndBlockchainHook -
func CreateVMAndBlockchainHook(
	accnts state.AccountsAdapter,
	gasSchedule map[string]map[string]uint64,
) (process.VirtualMachinesContainer, *hooks.BlockChainHookImpl) {
	actualGasSchedule := gasSchedule
	if gasSchedule == nil {
		actualGasSchedule = arwenConfig.MakeGasMap(1)
		FillGasMapInternal(actualGasSchedule, 1)
	}

	argsBuiltIn := builtInFunctions.ArgsCreateBuiltInFunctionContainer{
		GasMap:          actualGasSchedule,
		MapDNSAddresses: make(map[string]struct{}),
		Marshalizer:     testMarshalizer,
	}
	builtInFuncs, _ := builtInFunctions.CreateBuiltInFunctionContainer(argsBuiltIn)

	args := hooks.ArgBlockChainHook{
		Accounts:         accnts,
		PubkeyConv:       pubkeyConv,
		StorageService:   &mock.ChainStorerMock{},
		BlockChain:       &mock.BlockChainMock{},
		ShardCoordinator: oneShardCoordinator,
		Marshalizer:      testMarshalizer,
		Uint64Converter:  &mock.Uint64ByteSliceConverterMock{},
		BuiltInFunctions: builtInFuncs,
	}

	//Uncomment this to enable trace printing of the vm
	//vm.SetTracePretty()

	maxGasLimitPerBlock := uint64(0xFFFFFFFFFFFFFFFF)
	vmFactory, err := shard.NewVMContainerFactory(
		config.VirtualMachineConfig{
			OutOfProcessEnabled: true,
			OutOfProcessConfig:  config.VirtualMachineOutOfProcessConfig{MaxLoopTime: 1000},
		},
		maxGasLimitPerBlock,
		actualGasSchedule,
		args,
	)
	if err != nil {
		log.LogIfError(err)
	}

	vmContainer, err := vmFactory.Create()
	if err != nil {
		panic(err)
	}

	blockChainHook, _ := vmFactory.BlockChainHookImpl().(*hooks.BlockChainHookImpl)
	createAndAddIeleVM(vmContainer, blockChainHook)

	return vmContainer, blockChainHook
}

// CreateTxProcessorWithOneSCExecutorWithVMs -
func CreateTxProcessorWithOneSCExecutorWithVMs(
	accnts state.AccountsAdapter,
	vmContainer process.VirtualMachinesContainer,
	blockChainHook *hooks.BlockChainHookImpl,
) (process.TransactionProcessor, process.SmartContractProcessor) {
	argsParser := vmcommon.NewAtArgumentParser()
	argsTxTypeHandler := coordinator.ArgNewTxTypeHandler{
		PubkeyConverter:  pubkeyConv,
		ShardCoordinator: oneShardCoordinator,
		BuiltInFuncNames: blockChainHook.GetBuiltInFunctions().Keys(),
		ArgumentParser:   argsParser,
	}
	txTypeHandler, _ := coordinator.NewTxTypeHandler(argsTxTypeHandler)

	gasSchedule := make(map[string]map[string]uint64)
	FillGasMapInternal(gasSchedule, 1)
	argsNewSCProcessor := smartContract.ArgsNewSmartContractProcessor{
		VmContainer:  vmContainer,
		ArgsParser:   argsParser,
		Hasher:       testHasher,
		Marshalizer:  testMarshalizer,
		AccountsDB:   accnts,
		TempAccounts: blockChainHook,
		PubkeyConv:   pubkeyConv,
		Coordinator:  oneShardCoordinator,
		ScrForwarder: &mock.IntermediateTransactionHandlerMock{},
		TxFeeHandler: &mock.UnsignedTxHandlerMock{},
		EconomicsFee: &mock.FeeHandlerStub{
			DeveloperPercentageCalled: func() float64 {
				return 0.0
			},
		},
		TxTypeHandler: txTypeHandler,
		GasHandler: &mock.GasHandlerMock{
			SetGasRefundedCalled: func(gasRefunded uint64, hash []byte) {},
		},
		BuiltInFunctions: blockChainHook.GetBuiltInFunctions(),
		TxLogsProcessor:  &mock.TxLogsProcessorStub{},
	}

	scProcessor, _ := smartContract.NewSmartContractProcessor(argsNewSCProcessor)

	txProcessor, _ := transaction.NewTxProcessor(
		accnts,
		testHasher,
		pubkeyConv,
		testMarshalizer,
		oneShardCoordinator,
		scProcessor,
		&mock.UnsignedTxHandlerMock{},
		txTypeHandler,
		&mock.FeeHandlerStub{},
		&mock.IntermediateTransactionHandlerMock{},
		&mock.IntermediateTransactionHandlerMock{},
	)

	return txProcessor, scProcessor
}

// TestDeployedContractContents -
func TestDeployedContractContents(
	t *testing.T,
	destinationAddressBytes []byte,
	accnts state.AccountsAdapter,
	requiredBalance *big.Int,
	scCode string,
	dataValues map[string]*big.Int,
) {

	scCodeBytes, _ := hex.DecodeString(scCode)
	destinationRecovAccount, _ := accnts.GetExistingAccount(destinationAddressBytes)
	destinationRecovShardAccount, ok := destinationRecovAccount.(state.UserAccountHandler)

	assert.True(t, ok)
	assert.NotNil(t, destinationRecovShardAccount)
	assert.Equal(t, uint64(0), destinationRecovShardAccount.GetNonce())
	assert.Equal(t, requiredBalance, destinationRecovShardAccount.GetBalance())
	//test codehash
	assert.Equal(t, testHasher.Compute(string(scCodeBytes)), destinationRecovShardAccount.GetCodeHash())
	//test code
	assert.Equal(t, scCodeBytes, destinationRecovShardAccount.GetCode())
	//in this test we know we have a as a variable inside the contract, we can ask directly its value
	// using trackableDataTrie functionality
	assert.NotNil(t, destinationRecovShardAccount.GetRootHash())

	for variable, requiredVal := range dataValues {
		contractVariableData, err := destinationRecovShardAccount.DataTrieTracker().RetrieveValue([]byte(variable))
		assert.Nil(t, err)
		assert.NotNil(t, contractVariableData)

		contractVariableValue := big.NewInt(0).SetBytes(contractVariableData)
		assert.Equal(t, requiredVal, contractVariableValue)
	}
}

// AccountExists -
func AccountExists(accnts state.AccountsAdapter, addressBytes []byte) bool {
	accnt, _ := accnts.GetExistingAccount(addressBytes)

	return accnt != nil
}

// CreatePreparedTxProcessorAndAccountsWithVMs -
func CreatePreparedTxProcessorAndAccountsWithVMs(
	senderNonce uint64,
	senderAddressBytes []byte,
	senderBalance *big.Int,
) VMTestContext {
	accounts := CreateInMemoryShardAccountsDB()
	_, _ = CreateAccount(accounts, senderAddressBytes, senderNonce, senderBalance)
	vmContainer, blockchainHook := CreateVMAndBlockchainHook(accounts, nil)
	txProcessor, scProcessor := CreateTxProcessorWithOneSCExecutorWithVMs(accounts, vmContainer, blockchainHook)

	return VMTestContext{TxProcessor: txProcessor, ScProcessor: scProcessor, Accounts: accounts, BlockchainHook: blockchainHook, VMContainer: vmContainer}
}

// CreateTxProcessorArwenVMWithGasSchedule -
func CreateTxProcessorArwenVMWithGasSchedule(
	senderNonce uint64,
	senderAddressBytes []byte,
	senderBalance *big.Int,
	gasSchedule map[string]map[string]uint64,
) VMTestContext {
	accounts := CreateInMemoryShardAccountsDB()
	_, _ = CreateAccount(accounts, senderAddressBytes, senderNonce, senderBalance)
	vmContainer, blockchainHook := CreateVMAndBlockchainHook(accounts, gasSchedule)
	txProcessor, scProcessor := CreateTxProcessorWithOneSCExecutorWithVMs(accounts, vmContainer, blockchainHook)

	return VMTestContext{TxProcessor: txProcessor, ScProcessor: scProcessor, Accounts: accounts, BlockchainHook: blockchainHook, VMContainer: vmContainer}
}

// CreatePreparedTxProcessorAndAccountsWithMockedVM -
func CreatePreparedTxProcessorAndAccountsWithMockedVM(
	t *testing.T,
	vmOpGas uint64,
	senderNonce uint64,
	senderAddressBytes []byte,
	senderBalance *big.Int,
) (process.TransactionProcessor, state.AccountsAdapter) {

	accnts := CreateInMemoryShardAccountsDB()
	_, _ = CreateAccount(accnts, senderAddressBytes, senderNonce, senderBalance)

	txProcessor := CreateTxProcessorWithOneSCExecutorMockVM(accnts, vmOpGas)
	assert.NotNil(t, txProcessor)

	return txProcessor, accnts
}

// CreateTx -
func CreateTx(
	tb testing.TB,
	senderAddressBytes []byte,
	receiverAddressBytes []byte,
	senderNonce uint64,
	value *big.Int,
	gasPrice uint64,
	gasLimit uint64,
	scCodeOrFunc string,
) *dataTransaction.Transaction {

	txData := scCodeOrFunc
	tx := &dataTransaction.Transaction{
		Nonce:    senderNonce,
		Value:    new(big.Int).Set(value),
		SndAddr:  senderAddressBytes,
		RcvAddr:  receiverAddressBytes,
		Data:     []byte(txData),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
	}
	assert.NotNil(tb, tx)

	return tx
}

// CreateDeployTx -
func CreateDeployTx(
	senderAddressBytes []byte,
	senderNonce uint64,
	value *big.Int,
	gasPrice uint64,
	gasLimit uint64,
	scCodeAndVMType string,
) *dataTransaction.Transaction {

	return &dataTransaction.Transaction{
		Nonce:    senderNonce,
		Value:    new(big.Int).Set(value),
		SndAddr:  senderAddressBytes,
		RcvAddr:  CreateEmptyAddress(),
		Data:     []byte(scCodeAndVMType),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
	}
}

// TestAccount -
func TestAccount(
	t *testing.T,
	accnts state.AccountsAdapter,
	senderAddressBytes []byte,
	expectedNonce uint64,
	expectedBalance *big.Int,
) *big.Int {

	senderRecovAccount, _ := accnts.GetExistingAccount(senderAddressBytes)
	senderRecovShardAccount := senderRecovAccount.(state.UserAccountHandler)

	assert.Equal(t, expectedNonce, senderRecovShardAccount.GetNonce())
	assert.Equal(t, expectedBalance, senderRecovShardAccount.GetBalance())
	return senderRecovShardAccount.GetBalance()
}

// ComputeExpectedBalance -
func ComputeExpectedBalance(
	existing *big.Int,
	transferred *big.Int,
	gasLimit uint64,
	gasPrice uint64,
) *big.Int {

	expectedSenderBalance := big.NewInt(0).Sub(existing, transferred)
	gasFunds := big.NewInt(0).Mul(big.NewInt(0).SetUint64(gasLimit), big.NewInt(0).SetUint64(gasPrice))
	expectedSenderBalance.Sub(expectedSenderBalance, gasFunds)

	return expectedSenderBalance
}

// GetAccountsBalance -
func GetAccountsBalance(addrBytes []byte, accnts state.AccountsAdapter) *big.Int {
	accnt, _ := accnts.GetExistingAccount(addrBytes)
	shardAccnt, _ := accnt.(state.UserAccountHandler)

	return shardAccnt.GetBalance()
}

// GetIntValueFromSC -
func GetIntValueFromSC(gasSchedule map[string]map[string]uint64, accnts state.AccountsAdapter, scAddressBytes []byte, funcName string, args ...[]byte) *big.Int {
	vmContainer, _ := CreateVMAndBlockchainHook(accnts, gasSchedule)
	defer func() {
		_ = vmContainer.Close()
	}()

	feeHandler := &mock.FeeHandlerStub{
		MaxGasLimitPerBlockCalled: func() uint64 {
			return uint64(math.MaxUint64)
		},
	}

	scQueryService, _ := smartContract.NewSCQueryService(vmContainer, feeHandler)

	vmOutput, err := scQueryService.ExecuteQuery(&process.SCQuery{
		ScAddress: scAddressBytes,
		FuncName:  funcName,
		Arguments: args,
	})

	if err != nil {
		fmt.Println("ERROR at GetIntValueFromSC()", err)
		return big.NewInt(-1)
	}

	return big.NewInt(0).SetBytes(vmOutput.ReturnData[0])
}

// CreateTransferTokenTx -
func CreateTransferTokenTx(
	nonce uint64,
	value *big.Int,
	scAddrress []byte,
	sndAddress []byte,
	rcvAddress []byte,
) *dataTransaction.Transaction {
	return &dataTransaction.Transaction{
		Nonce:    nonce,
		Value:    big.NewInt(0),
		RcvAddr:  scAddrress,
		SndAddr:  sndAddress,
		GasPrice: 0,
		GasLimit: 5000000,
		Data:     []byte("transferToken@" + hex.EncodeToString(rcvAddress) + "@00" + hex.EncodeToString(value.Bytes())),
	}
}

// CreateMoveBalanceTx -
func CreateMoveBalanceTx(
	nonce uint64,
	value *big.Int,
	sndAddress []byte,
	rcvAddress []byte,
	gasLimit uint64,
) *dataTransaction.Transaction {
	return &dataTransaction.Transaction{
		Nonce:    nonce,
		Value:    big.NewInt(0).Set(value),
		RcvAddr:  rcvAddress,
		SndAddr:  sndAddress,
		GasPrice: 1,
		GasLimit: gasLimit,
	}
}

// FillGasMapInternal -
func FillGasMapInternal(gasMap map[string]map[string]uint64, value uint64) map[string]map[string]uint64 {
	gasMap[core.BaseOperationCost] = FillGasMapBaseOperationCosts(value)
	gasMap[core.BuiltInCost] = FillGasMapBuiltInCosts(value)
	gasMap[core.MetaChainSystemSCsCost] = FillGasMapMetaChainSystemSCsCosts(value)

	return gasMap
}

// FillGasMapBaseOperationCosts -
func FillGasMapBaseOperationCosts(value uint64) map[string]uint64 {
	gasMap := make(map[string]uint64)
	gasMap["StorePerByte"] = value
	gasMap["DataCopyPerByte"] = value
	gasMap["ReleasePerByte"] = value
	gasMap["PersistPerByte"] = value
	gasMap["CompilePerByte"] = value

	return gasMap
}

// FillGasMapBuiltInCosts -
func FillGasMapBuiltInCosts(value uint64) map[string]uint64 {
	gasMap := make(map[string]uint64)
	gasMap["ClaimDeveloperRewards"] = value
	gasMap["ChangeOwnerAddress"] = value
	gasMap["SaveUserName"] = value
	gasMap["SaveKeyValue"] = value
	gasMap["ESDTTransfer"] = value

	return gasMap
}

// FillGasMapMetaChainSystemSCsCosts -
func FillGasMapMetaChainSystemSCsCosts(value uint64) map[string]uint64 {
	gasMap := make(map[string]uint64)
	gasMap["Stake"] = value
	gasMap["UnStake"] = value
	gasMap["UnBond"] = value
	gasMap["Claim"] = value
	gasMap["Get"] = value
	gasMap["ChangeRewardAddress"] = value
	gasMap["ChangeValidatorKeys"] = value
	gasMap["UnJail"] = value
	gasMap["ESDTIssue"] = value
	gasMap["ESDTOperations"] = value

	return gasMap
}
