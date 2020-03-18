package arwen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"path"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/state/addressConverters"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/hashing/sha256"
	"github.com/ElrondNetwork/elrond-go/integrationTests/mock"
	"github.com/ElrondNetwork/elrond-go/integrationTests/vm"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/coordinator"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	transaction2 "github.com/ElrondNetwork/elrond-go/process/transaction"
	"github.com/stretchr/testify/assert"
)

func TestVmDeployWithTransferAndGasShouldDeploySCCode(t *testing.T) {
	senderAddressBytes := []byte("12345678901234567890123456789012")
	senderNonce := uint64(0)
	senderBalance := big.NewInt(100000000)
	gasPrice := uint64(1)
	gasLimit := uint64(100000)
	transferOnCalls := big.NewInt(50)

	scCode, err := getBytecode("misc/fib_arwen.wasm")
	assert.Nil(t, err)

	scCodeString := hex.EncodeToString(scCode)

	tx := vm.CreateTx(
		t,
		senderAddressBytes,
		vm.CreateEmptyAddress().Bytes(),
		senderNonce,
		transferOnCalls,
		gasPrice,
		gasLimit,
		scCodeString+"@"+hex.EncodeToString(factory.ArwenVirtualMachine),
	)

	testContext := vm.CreatePreparedTxProcessorAndAccountsWithVMs(senderNonce, senderAddressBytes, senderBalance)
	defer testContext.Close()

	err = testContext.TxProcessor.ProcessTransaction(tx)
	assert.Nil(t, err)

	_, err = testContext.Accounts.Commit()
	assert.Nil(t, err)

	expectedBalance := big.NewInt(99999699)
	fmt.Printf("%s \n", hex.EncodeToString(expectedBalance.Bytes()))

	vm.TestAccount(
		t,
		testContext.Accounts,
		senderAddressBytes,
		senderNonce+1,
		expectedBalance)
}

func TestSCMoveBalanceBeforeSCDeploy(t *testing.T) {
	ownerAddressBytes := []byte("12345678901234567890123456789012")
	ownerNonce := uint64(0)
	ownerBalance := big.NewInt(100000000)
	gasPrice := uint64(0)
	gasLimit := uint64(100000)
	transferOnCalls := big.NewInt(50)

	scCode, err := getBytecode("misc/fib_arwen.wasm")
	assert.Nil(t, err)
	scCodeString := hex.EncodeToString(scCode)

	testContext := vm.CreatePreparedTxProcessorAndAccountsWithVMs(ownerNonce, ownerAddressBytes, ownerBalance)
	defer testContext.Close()

	scAddressBytes, _ := testContext.BlockchainHook.NewAddress(ownerAddressBytes, ownerNonce+1, factory.ArwenVirtualMachine)
	fmt.Println(hex.EncodeToString(scAddressBytes))

	tx := vm.CreateTx(t,
		ownerAddressBytes,
		scAddressBytes,
		ownerNonce,
		transferOnCalls,
		gasPrice,
		gasLimit,
		"")

	err = testContext.TxProcessor.ProcessTransaction(tx)
	assert.Nil(t, err)

	_, err = testContext.Accounts.Commit()
	assert.Nil(t, err)

	ownerNonce++
	tx = vm.CreateTx(
		t,
		ownerAddressBytes,
		vm.CreateEmptyAddress().Bytes(),
		ownerNonce,
		transferOnCalls,
		gasPrice,
		gasLimit,
		scCodeString+"@"+hex.EncodeToString(factory.ArwenVirtualMachine),
	)

	err = testContext.TxProcessor.ProcessTransaction(tx)
	assert.Nil(t, err)

	_, err = testContext.Accounts.Commit()
	assert.Nil(t, err)

	expectedBalance := ownerBalance.Uint64() - 2*transferOnCalls.Uint64()
	vm.TestAccount(
		t,
		testContext.Accounts,
		ownerAddressBytes,
		ownerNonce+1,
		big.NewInt(0).SetUint64(expectedBalance))

	expectedBalance = 2 * transferOnCalls.Uint64()
	vm.TestAccount(
		t,
		testContext.Accounts,
		scAddressBytes,
		0,
		big.NewInt(0).SetUint64(expectedBalance))
}

func Benchmark_VmDeployWithFibbonacciAndExecute(b *testing.B) {
	runWASMVMBenchmark(b, "misc/fib_arwen.wasm", b.N, 32, nil)
}

func Benchmark_VmDeployWithCPUCalculateAndExecute(b *testing.B) {
	runWASMVMBenchmark(b, "misc/cpucalculate_arwen.wasm", b.N, 8000, nil)
}

func Benchmark_VmDeployWithStringConcatAndExecute(b *testing.B) {
	runWASMVMBenchmark(b, "misc/stringconcat_arwen.wasm", b.N, 10000, nil)
}

func runWASMVMBenchmark(
	tb testing.TB,
	fileSC string,
	numRun int,
	testingValue uint64,
	gasSchedule map[string]map[string]uint64,
) {
	ownerAddressBytes := []byte("12345678901234567890123456789012")
	ownerNonce := uint64(11)
	ownerBalance := big.NewInt(0xfffffffffffffff)
	ownerBalance.Mul(ownerBalance, big.NewInt(0xffffffff))
	gasPrice := uint64(1)
	gasLimit := uint64(0xffffffffffffffff)
	transferOnCalls := big.NewInt(1)

	scCode, err := getBytecode(fileSC)
	assert.Nil(tb, err)

	scCodeString := hex.EncodeToString(scCode)

	tx := &transaction.Transaction{
		Nonce:     ownerNonce,
		Value:     new(big.Int).Set(transferOnCalls),
		RcvAddr:   vm.CreateEmptyAddress().Bytes(),
		SndAddr:   ownerAddressBytes,
		GasPrice:  gasPrice,
		GasLimit:  gasLimit,
		Data:      []byte(scCodeString + "@" + hex.EncodeToString(factory.ArwenVirtualMachine)),
		Signature: nil,
	}

	testContext := vm.CreateTxProcessorArwenVMWithGasSchedule(ownerNonce, ownerAddressBytes, ownerBalance, gasSchedule)
	defer testContext.Close()

	scAddress, _ := testContext.BlockchainHook.NewAddress(ownerAddressBytes, ownerNonce, factory.ArwenVirtualMachine)

	err = testContext.TxProcessor.ProcessTransaction(tx)
	assert.Nil(tb, err)

	_, err = testContext.Accounts.Commit()
	assert.Nil(tb, err)

	alice := []byte("12345678901234567890123456789111")
	aliceNonce := uint64(0)
	_, _ = vm.CreateAccount(testContext.Accounts, alice, aliceNonce, big.NewInt(10000000000))

	tx = &transaction.Transaction{
		Nonce:     aliceNonce,
		Value:     new(big.Int).Set(big.NewInt(0).SetUint64(testingValue)),
		RcvAddr:   scAddress,
		SndAddr:   alice,
		GasPrice:  0,
		GasLimit:  gasLimit,
		Data:      []byte("_main"),
		Signature: nil,
	}

	for i := 0; i < numRun; i++ {
		tx.Nonce = aliceNonce

		_ = testContext.TxProcessor.ProcessTransaction(tx)

		aliceNonce++
	}
}

func TestGasModel(t *testing.T) {
	gasSchedule, _ := core.LoadGasScheduleConfig("./gasSchedule.toml")

	totalOp := uint64(0)
	for _, opCodeClass := range gasSchedule {
		for _, opCode := range opCodeClass {
			totalOp += opCode
		}
	}
	fmt.Println("gasSchedule: " + big.NewInt(int64(totalOp)).String())
	fmt.Println("FIBONNACI 32 ")
	runWASMVMBenchmark(t, "misc/fib_arwen.wasm", 1, 32, gasSchedule)
	fmt.Println("CPUCALCULATE 8000 ")
	runWASMVMBenchmark(t, "misc/cpucalculate_arwen.wasm", 1, 8000, gasSchedule)
	fmt.Println("STRINGCONCAT 1000 ")
	runWASMVMBenchmark(t, "misc/stringconcat_arwen.wasm", 1, 10000, gasSchedule)
	fmt.Println("ERC20 ")
	deployWithTransferAndExecuteERC20(t, 2, gasSchedule)
	fmt.Println("ERC20 BIGINT")
	deployAndExecuteERC20WithBigInt(t, 2, gasSchedule)
}

func TestMultipleTimesERC20InBatches(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	for i := 0; i < 10; i++ {
		deployWithTransferAndExecuteERC20(t, 1000, nil)
	}
}

func deployWithTransferAndExecuteERC20(t *testing.T, numRun int, gasSchedule map[string]map[string]uint64) {
	ownerAddressBytes := []byte("12345678901234567890123456789011")
	ownerNonce := uint64(11)
	ownerBalance := big.NewInt(10000000000000)
	gasPrice := uint64(1)
	gasLimit := uint64(10000000000)
	transferOnCalls := big.NewInt(5)

	scCode, err := getBytecode("erc20/wrc20_arwen_03.wasm")
	assert.Nil(t, err)

	scCodeString := hex.EncodeToString(scCode)
	testContext := vm.CreateTxProcessorArwenVMWithGasSchedule(ownerNonce, ownerAddressBytes, ownerBalance, gasSchedule)
	defer testContext.Close()

	scAddress, _ := testContext.BlockchainHook.NewAddress(ownerAddressBytes, ownerNonce, factory.ArwenVirtualMachine)

	initialSupply := hex.EncodeToString(big.NewInt(100000000000).Bytes())
	tx := vm.CreateDeployTx(
		ownerAddressBytes,
		ownerNonce,
		transferOnCalls,
		gasPrice,
		gasLimit,
		[]byte(scCodeString+"@"+hex.EncodeToString(factory.ArwenVirtualMachine)+"@"+initialSupply),
	)

	err = testContext.TxProcessor.ProcessTransaction(tx)
	assert.Nil(t, err)

	ownerNonce++

	alice := []byte("12345678901234567890123456789111")
	aliceNonce := uint64(0)
	_, _ = vm.CreateAccount(testContext.Accounts, alice, aliceNonce, big.NewInt(1000000))

	bob := []byte("12345678901234567890123456789222")
	_, _ = vm.CreateAccount(testContext.Accounts, bob, 0, big.NewInt(1000000))

	initAlice := big.NewInt(100000)
	tx = vm.CreateTransferTx(ownerNonce, initAlice, scAddress, ownerAddressBytes, alice)

	err = testContext.TxProcessor.ProcessTransaction(tx)
	assert.Nil(t, err)

	start := time.Now()

	for i := 0; i < numRun; i++ {
		tx = vm.CreateTransferTx(aliceNonce, transferOnCalls, scAddress, alice, bob)

		err = testContext.TxProcessor.ProcessTransaction(tx)
		if err != nil {
			assert.Nil(t, err)
		}
		assert.Nil(t, err)

		aliceNonce++
	}

	elapsedTime := time.Since(start)
	fmt.Printf("time elapsed to process %d ERC20 transfers %s \n", numRun, elapsedTime.String())

	_, err = testContext.Accounts.Commit()
	assert.Nil(t, err)

	finalAlice := big.NewInt(0).Sub(initAlice, big.NewInt(int64(numRun)*transferOnCalls.Int64()))
	assert.Equal(t, finalAlice.Uint64(), vm.GetIntValueFromSC(gasSchedule, testContext.Accounts, scAddress, "balanceOf", alice).Uint64())
	finalBob := big.NewInt(int64(numRun) * transferOnCalls.Int64())
	assert.Equal(t, finalBob.Uint64(), vm.GetIntValueFromSC(gasSchedule, testContext.Accounts, scAddress, "balanceOf", bob).Uint64())
}

func TestWASMNamespacing(t *testing.T) {
	ownerAddressBytes := []byte("12345678901234567890123456789012")
	ownerNonce := uint64(11)
	ownerBalance := big.NewInt(0xfffffffffffffff)
	ownerBalance.Mul(ownerBalance, big.NewInt(0xffffffff))
	gasPrice := uint64(1)
	gasLimit := uint64(0xffffffffffffffff)
	transferOnCalls := big.NewInt(1)

	// This SmartContract had its imports modified after compilation, replacing
	// the namespace 'env' to 'ethereum'. If WASM namespacing is done correctly
	// by Arwen, then this SC should have no problem to call imported functions
	// (as if it were run by Ethereuem).
	scCode, err := getBytecode("misc/fib_ewasmified.wasm")
	assert.Nil(t, err)

	scCodeString := hex.EncodeToString(scCode)

	tx := &transaction.Transaction{
		Nonce:     ownerNonce,
		Value:     new(big.Int).Set(transferOnCalls),
		RcvAddr:   vm.CreateEmptyAddress().Bytes(),
		SndAddr:   ownerAddressBytes,
		GasPrice:  gasPrice,
		GasLimit:  gasLimit,
		Data:      []byte(scCodeString + "@" + hex.EncodeToString(factory.ArwenVirtualMachine)),
		Signature: nil,
	}

	testContext := vm.CreatePreparedTxProcessorAndAccountsWithVMs(ownerNonce, ownerAddressBytes, ownerBalance)
	defer testContext.Close()

	scAddress, _ := testContext.BlockchainHook.NewAddress(ownerAddressBytes, ownerNonce, factory.ArwenVirtualMachine)

	err = testContext.TxProcessor.ProcessTransaction(tx)
	assert.Nil(t, err)

	_, err = testContext.Accounts.Commit()
	assert.Nil(t, err)

	alice := []byte("12345678901234567890123456789111")
	aliceNonce := uint64(0)
	aliceInitialBalance := uint64(3000)
	_, _ = vm.CreateAccount(testContext.Accounts, alice, aliceNonce, big.NewInt(0).SetUint64(aliceInitialBalance))

	testingValue := uint64(15)

	gasLimit = uint64(2000)

	tx = &transaction.Transaction{
		Nonce:     aliceNonce,
		Value:     new(big.Int).Set(big.NewInt(0).SetUint64(testingValue)),
		RcvAddr:   scAddress,
		SndAddr:   alice,
		GasPrice:  gasPrice,
		GasLimit:  gasLimit,
		Data:      []byte("main"),
		Signature: nil,
	}

	err = testContext.TxProcessor.ProcessTransaction(tx)
	assert.Nil(t, err)
}

func TestWASMMetering(t *testing.T) {
	ownerAddressBytes := []byte("12345678901234567890123456789012")
	ownerNonce := uint64(11)
	ownerBalance := big.NewInt(0xfffffffffffffff)
	ownerBalance.Mul(ownerBalance, big.NewInt(0xffffffff))
	gasPrice := uint64(1)
	gasLimit := uint64(0xffffffffffffffff)
	transferOnCalls := big.NewInt(1)

	scCode, err := getBytecode("misc/cpucalculate_arwen.wasm")
	assert.Nil(t, err)

	scCodeString := hex.EncodeToString(scCode)

	tx := &transaction.Transaction{
		Nonce:     ownerNonce,
		Value:     new(big.Int).Set(transferOnCalls),
		RcvAddr:   vm.CreateEmptyAddress().Bytes(),
		SndAddr:   ownerAddressBytes,
		GasPrice:  gasPrice,
		GasLimit:  gasLimit,
		Data:      []byte(scCodeString + "@" + hex.EncodeToString(factory.ArwenVirtualMachine)),
		Signature: nil,
	}

	testContext := vm.CreatePreparedTxProcessorAndAccountsWithVMs(ownerNonce, ownerAddressBytes, ownerBalance)
	defer testContext.Close()

	scAddress, _ := testContext.BlockchainHook.NewAddress(ownerAddressBytes, ownerNonce, factory.ArwenVirtualMachine)

	err = testContext.TxProcessor.ProcessTransaction(tx)
	assert.Nil(t, err)

	_, err = testContext.Accounts.Commit()
	assert.Nil(t, err)

	alice := []byte("12345678901234567890123456789111")
	aliceNonce := uint64(0)
	aliceInitialBalance := uint64(3000)
	_, _ = vm.CreateAccount(testContext.Accounts, alice, aliceNonce, big.NewInt(0).SetUint64(aliceInitialBalance))

	testingValue := uint64(15)

	gasLimit = uint64(2000)

	tx = &transaction.Transaction{
		Nonce:     aliceNonce,
		Value:     new(big.Int).Set(big.NewInt(0).SetUint64(testingValue)),
		RcvAddr:   scAddress,
		SndAddr:   alice,
		GasPrice:  gasPrice,
		GasLimit:  gasLimit,
		Data:      []byte("_main"),
		Signature: nil,
	}

	err = testContext.TxProcessor.ProcessTransaction(tx)
	assert.Nil(t, err)

	expectedBalance := big.NewInt(2615)
	expectedNonce := uint64(1)

	actualBalanceBigInt := vm.TestAccount(
		t,
		testContext.Accounts,
		alice,
		expectedNonce,
		expectedBalance)

	actualBalance := actualBalanceBigInt.Uint64()

	consumedGasValue := aliceInitialBalance - actualBalance - testingValue

	assert.Equal(t, 370, int(consumedGasValue))
}

func TestMultipleTimesERC20BigIntInBatches(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	for i := 0; i < 10; i++ {
		deployAndExecuteERC20WithBigInt(t, 1000, nil)
	}
}

func deployAndExecuteERC20WithBigInt(t *testing.T, numRun int, gasSchedule map[string]map[string]uint64) {
	ownerAddressBytes := []byte("12345678901234567890123456789011")
	ownerNonce := uint64(11)
	ownerBalance := big.NewInt(10000000000000)
	gasPrice := uint64(1)
	gasLimit := uint64(10000000000)
	transferOnCalls := big.NewInt(5)

	scCode, err := getBytecode("erc20/wrc20_arwen_03.wasm")
	assert.Nil(t, err)

	scCodeString := hex.EncodeToString(scCode)
	testContext := vm.CreateTxProcessorArwenVMWithGasSchedule(ownerNonce, ownerAddressBytes, ownerBalance, gasSchedule)
	defer testContext.Close()

	scAddress, _ := testContext.BlockchainHook.NewAddress(ownerAddressBytes, ownerNonce, factory.ArwenVirtualMachine)

	tx := vm.CreateDeployTx(
		ownerAddressBytes,
		ownerNonce,
		transferOnCalls,
		gasPrice,
		gasLimit,
		[]byte(scCodeString+"@"+hex.EncodeToString(factory.ArwenVirtualMachine)+"@"+hex.EncodeToString(ownerBalance.Bytes())),
	)

	err = testContext.TxProcessor.ProcessTransaction(tx)
	assert.Nil(t, err)
	ownerNonce++

	alice := []byte("12345678901234567890123456789111")
	aliceNonce := uint64(0)
	_, _ = vm.CreateAccount(testContext.Accounts, alice, aliceNonce, big.NewInt(1000000))

	bob := []byte("12345678901234567890123456789222")
	_, _ = vm.CreateAccount(testContext.Accounts, bob, 0, big.NewInt(1000000))

	initAlice := big.NewInt(100000)
	tx = vm.CreateTransferTokenTx(ownerNonce, initAlice, scAddress, ownerAddressBytes, alice)

	err = testContext.TxProcessor.ProcessTransaction(tx)
	assert.Nil(t, err)

	start := time.Now()

	for i := 0; i < numRun; i++ {
		tx = vm.CreateTransferTokenTx(aliceNonce, transferOnCalls, scAddress, alice, bob)

		err = testContext.TxProcessor.ProcessTransaction(tx)
		if err != nil {
			assert.Nil(t, err)
		}
		assert.Nil(t, err)

		aliceNonce++
	}

	elapsedTime := time.Since(start)
	fmt.Printf("time elapsed to process %d ERC20 transfers %s \n", numRun, elapsedTime.String())

	_, err = testContext.Accounts.Commit()
	assert.Nil(t, err)

	finalAlice := big.NewInt(0).Sub(initAlice, big.NewInt(int64(numRun)*transferOnCalls.Int64()))
	assert.Equal(t, finalAlice.Uint64(), vm.GetIntValueFromSC(gasSchedule, testContext.Accounts, scAddress, "balanceOf", alice).Uint64())
	finalBob := big.NewInt(int64(numRun) * transferOnCalls.Int64())
	assert.Equal(t, finalBob.Uint64(), vm.GetIntValueFromSC(gasSchedule, testContext.Accounts, scAddress, "balanceOf", bob).Uint64())
}

func generateRandomByteArray(size int) []byte {
	r := make([]byte, size)
	_, _ = rand.Read(r)
	return r
}

func createTestAddresses(numAddresses uint64) [][]byte {
	testAccounts := make([][]byte, numAddresses)

	for i := uint64(0); i < numAddresses; i++ {
		acc := generateRandomByteArray(32)
		testAccounts[i] = append(testAccounts[i], acc...)
	}

	return testAccounts
}

func TestJournalizingAndTimeToProcessChange(t *testing.T) {
	// Only a test to benchmark jurnalizing and getting data from trie
	t.Skip()

	numRun := 1000
	ownerAddressBytes := []byte("12345678901234567890123456789011")
	ownerNonce := uint64(11)
	ownerBalance := big.NewInt(10000000000000)
	gasPrice := uint64(1)
	gasLimit := uint64(10000000000)
	transferOnCalls := big.NewInt(5)

	scCode, err := getBytecode("erc20/wrc20_arwen_03.wasm")
	assert.Nil(t, err)

	scCodeString := hex.EncodeToString(scCode)
	testContext := vm.CreateTxProcessorArwenVMWithGasSchedule(ownerNonce, ownerAddressBytes, ownerBalance, nil)
	defer testContext.Close()

	scAddress, _ := testContext.BlockchainHook.NewAddress(ownerAddressBytes, ownerNonce, factory.ArwenVirtualMachine)

	tx := vm.CreateDeployTx(
		ownerAddressBytes,
		ownerNonce,
		transferOnCalls,
		gasPrice,
		gasLimit,
		[]byte(scCodeString+"@"+hex.EncodeToString(factory.ArwenVirtualMachine)+"@"+hex.EncodeToString(ownerBalance.Bytes())),
	)

	err = testContext.TxProcessor.ProcessTransaction(tx)
	assert.Nil(t, err)
	ownerNonce++

	alice := []byte("12345678901234567890123456789111")
	aliceNonce := uint64(0)
	_, _ = vm.CreateAccount(testContext.Accounts, alice, aliceNonce, big.NewInt(1000000))

	bob := []byte("12345678901234567890123456789222")
	_, _ = vm.CreateAccount(testContext.Accounts, bob, 0, big.NewInt(1000000))

	testAddresses := createTestAddresses(2000000)
	fmt.Println("done")

	initAlice := big.NewInt(100000)
	tx = vm.CreateTransferTokenTx(ownerNonce, initAlice, scAddress, ownerAddressBytes, alice)

	err = testContext.TxProcessor.ProcessTransaction(tx)
	assert.Nil(t, err)

	for j := 0; j < 2000; j++ {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			tx = vm.CreateTransferTokenTx(aliceNonce, transferOnCalls, scAddress, alice, testAddresses[j*1000+i])

			err = testContext.TxProcessor.ProcessTransaction(tx)
			if err != nil {
				assert.Nil(t, err)
			}
			assert.Nil(t, err)

			aliceNonce++
		}

		elapsedTime := time.Since(start)
		fmt.Printf("time elapsed to process 1000 ERC20 transfers %s \n", elapsedTime.String())

		_, err = testContext.Accounts.Commit()
		assert.Nil(t, err)
	}

	_, err = testContext.Accounts.Commit()
	assert.Nil(t, err)

	start := time.Now()

	for i := 0; i < numRun; i++ {
		tx = vm.CreateTransferTokenTx(aliceNonce, transferOnCalls, scAddress, alice, testAddresses[i])

		err = testContext.TxProcessor.ProcessTransaction(tx)
		if err != nil {
			assert.Nil(t, err)
		}
		assert.Nil(t, err)

		aliceNonce++
	}

	elapsedTime := time.Since(start)
	fmt.Printf("time elapsed to process %d ERC20 transfers %s \n", numRun, elapsedTime.String())

	_, err = testContext.Accounts.Commit()
	assert.Nil(t, err)
}

func getBytecode(relativePath string) ([]byte, error) {
	return ioutil.ReadFile(path.Join(".", "testdata", relativePath))
}

func TestExecuteTransactionAndTimeToProcessChange(t *testing.T) {
	// Only a test to benchmark transaction processing
	t.Skip()

	testMarshalizer := &marshal.JsonMarshalizer{}
	testHasher := sha256.Sha256{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(2)
	addrConv, _ := addressConverters.NewPlainAddressConverter(32, "0x")
	accnts := vm.CreateInMemoryShardAccountsDB()
	txTypeHandler, _ := coordinator.NewTxTypeHandler(
		addrConv,
		shardCoordinator,
		accnts)
	feeHandler := &mock.FeeHandlerStub{
		ComputeFeeCalled: func(tx process.TransactionWithFeeHandler) *big.Int {
			return big.NewInt(10)
		},
	}
	numRun := 20000
	ownerAddressBytes := []byte("12345678901234567890123456789011")
	ownerNonce := uint64(11)
	ownerBalance := big.NewInt(10000000000000)
	transferOnCalls := big.NewInt(5)

	_, _ = vm.CreateAccount(accnts, ownerAddressBytes, ownerNonce, ownerBalance)
	txProc, _ := transaction2.NewTxProcessor(
		accnts,
		testHasher,
		addrConv,
		testMarshalizer,
		shardCoordinator,
		&mock.SCProcessorMock{},
		&mock.UnsignedTxHandlerMock{},
		txTypeHandler,
		feeHandler,
		&mock.IntermediateTransactionHandlerMock{},
		&mock.IntermediateTransactionHandlerMock{},
	)

	alice := []byte("12345678901234567890123456789111")
	aliceNonce := uint64(0)
	_, _ = vm.CreateAccount(accnts, alice, aliceNonce, big.NewInt(1000000))

	bob := []byte("12345678901234567890123456789222")
	_, _ = vm.CreateAccount(accnts, bob, 0, big.NewInt(1000000))

	testAddresses := createTestAddresses(uint64(numRun))
	fmt.Println("done")

	gasLimit := feeHandler.ComputeFeeCalled(&transaction.Transaction{}).Uint64()
	initAlice := big.NewInt(100000)
	tx := vm.CreateMoveBalanceTx(ownerNonce, initAlice, ownerAddressBytes, alice, gasLimit)

	err := txProc.ProcessTransaction(tx)
	assert.Nil(t, err)

	for j := 0; j < 20; j++ {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			tx = vm.CreateMoveBalanceTx(aliceNonce, transferOnCalls, alice, testAddresses[j*1000+i], gasLimit)

			err = txProc.ProcessTransaction(tx)
			if err != nil {
				assert.Nil(t, err)
			}
			assert.Nil(t, err)

			aliceNonce++
		}

		elapsedTime := time.Since(start)
		fmt.Printf("time elapsed to process 1000 move balances %s \n", elapsedTime.String())

		_, err = accnts.Commit()
		assert.Nil(t, err)
	}

	_, err = accnts.Commit()
	assert.Nil(t, err)

	start := time.Now()

	for i := 0; i < numRun; i++ {
		tx = vm.CreateMoveBalanceTx(aliceNonce, transferOnCalls, alice, testAddresses[i], gasLimit)

		err = txProc.ProcessTransaction(tx)
		if err != nil {
			assert.Nil(t, err)
		}
		assert.Nil(t, err)

		aliceNonce++
	}

	elapsedTime := time.Since(start)
	fmt.Printf("time elapsed to process %d move balances %s \n", numRun, elapsedTime.String())

	_, err = accnts.Commit()
	assert.Nil(t, err)
}
