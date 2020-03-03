package mockVM

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/integrationTests/vm"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/stretchr/testify/assert"
)

func TestVMInvalidSmartContractCodeShouldNotGenerateAccount(t *testing.T) {
	scCode := []byte("wrong smart contract code")

	senderAddressBytes := []byte("12345678901234567890123456789012")
	senderNonce := uint64(11)
	senderBalance := big.NewInt(100000000)
	gasPrice := uint64(1)
	gasLimit := uint64(1000000)

	testContext := vm.CreatePreparedTxProcessorAndAccountsWithVMs(senderNonce, senderAddressBytes, senderBalance)
	defer testContext.Close()

	assert.Equal(t, 0, testContext.Accounts.JournalLen())

	tx := &transaction.Transaction{
		Nonce:    senderNonce,
		Value:    big.NewInt(0),
		SndAddr:  senderAddressBytes,
		RcvAddr:  vm.CreateEmptyAddress().Bytes(),
		Data:     []byte(string(scCode) + "@" + hex.EncodeToString(factory.IELEVirtualMachine)),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
	}

	// tx is not processed due to the invalid sc code
	_ = testContext.TxProcessor.ProcessTransaction(tx)

	scAddressBytes, _ := testContext.BlockchainHook.NewAddress(senderAddressBytes, senderNonce, factory.IELEVirtualMachine)

	scAccount, err := testContext.Accounts.GetExistingAccount(state.NewAddress(scAddressBytes))
	assert.Nil(t, scAccount)
	assert.Equal(t, state.ErrAccNotFound, err)
}

func TestVmDeployWithTransferAndGasShouldDeploySCCode(t *testing.T) {
	senderAddressBytes := []byte("12345678901234567890123456789012")
	senderNonce := uint64(11)
	senderBalance := big.NewInt(100000000)
	gasPrice := uint64(1)
	gasLimit := uint64(100000)
	transferOnCalls := big.NewInt(50)

	initialValueForInternalVariable := uint64(45)
	scCode := fmt.Sprintf("0000003B6302690003616464690004676574416700000001616101550468000100016161015406010A6161015506F6000068000200006161005401F6000101@%s@%X",
		hex.EncodeToString(factory.IELEVirtualMachine), initialValueForInternalVariable)

	tx := vm.CreateTx(
		t,
		senderAddressBytes,
		vm.CreateEmptyAddress().Bytes(),
		senderNonce,
		transferOnCalls,
		gasPrice,
		gasLimit,
		scCode,
	)

	testContext := vm.CreatePreparedTxProcessorAndAccountsWithVMs(senderNonce, senderAddressBytes, senderBalance)
	defer testContext.Close()

	err := testContext.TxProcessor.ProcessTransaction(tx)
	assert.Nil(t, err)

	_, err = testContext.Accounts.Commit()
	assert.Nil(t, err)

	expectedBalance := big.NewInt(99999811)
	vm.TestAccount(
		t,
		testContext.Accounts,
		senderAddressBytes,
		senderNonce+1,
		expectedBalance)
	destinationAddressBytes, _ := testContext.BlockchainHook.NewAddress(senderAddressBytes, senderNonce, factory.IELEVirtualMachine)

	vm.TestDeployedContractContents(
		t,
		destinationAddressBytes,
		testContext.Accounts,
		transferOnCalls,
		scCode,
		map[string]*big.Int{"a": big.NewInt(0).SetUint64(initialValueForInternalVariable)})
}

func TestVMDeployWithTransferWithInsufficientGasShouldReturnErr(t *testing.T) {
	senderAddressBytes := []byte("12345678901234567890123456789012")
	senderNonce := uint64(11)
	senderBalance := big.NewInt(100000000)
	gasPrice := uint64(1)
	//less than requirement
	gasLimit := uint64(100)
	transferOnCalls := big.NewInt(50)

	initialValueForInternalVariable := uint64(45)
	scCode := fmt.Sprintf("0000003B6302690003616464690004676574416700000001616101550468000100016161015406010A6161015506F6000068000200006161005401F6000101@%s@%X",
		hex.EncodeToString(factory.IELEVirtualMachine), initialValueForInternalVariable)

	tx := vm.CreateTx(
		t,
		senderAddressBytes,
		vm.CreateEmptyAddress().Bytes(),
		senderNonce,
		transferOnCalls,
		gasPrice,
		gasLimit,
		scCode,
	)

	testContext := vm.CreatePreparedTxProcessorAndAccountsWithVMs(senderNonce, senderAddressBytes, senderBalance)
	defer testContext.Close()

	err := testContext.TxProcessor.ProcessTransaction(tx)
	assert.Nil(t, err)

	_, err = testContext.Accounts.Commit()
	assert.Nil(t, err)

	expectedBalance := big.NewInt(99999900)
	vm.TestAccount(
		t,
		testContext.Accounts,
		senderAddressBytes,
		senderNonce+1,
		//the transfer should get back to the sender as the tx failed
		expectedBalance)
	destinationAddressBytes, _ := testContext.BlockchainHook.NewAddress(senderAddressBytes, senderNonce, factory.IELEVirtualMachine)

	assert.False(t, vm.AccountExists(testContext.Accounts, destinationAddressBytes))
}
