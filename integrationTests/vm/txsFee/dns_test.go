package txsFee

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core/vmcommon"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/integrationTests/vm"
	"github.com/ElrondNetwork/elrond-go/integrationTests/vm/txsFee/utils"
	"github.com/stretchr/testify/require"
)

func TestDeployDNSContract_TestRegisterAndResolveAndSendTxWithSndAndRcvUserName(t *testing.T) {
	testContext := vm.CreatePreparedTxProcessorWithVMs(t, vm.ArgEnableEpoch{})
	defer testContext.Close()

	scAddress, _ := utils.DoDeployDNS(t, &testContext, "../../multiShard/smartContract/dns/dns.wasm")
	fmt.Println(scAddress)
	utils.CleanAccumulatedIntermediateTransactions(t, &testContext)

	sndAddr := []byte("12345678901234567890123456789112")
	senderBalance := big.NewInt(10000000)
	gasPrice := uint64(10)
	gasLimit := uint64(500000)

	rcvAddr := []byte("12345678901234567890123456789113")

	_, _ = vm.CreateAccount(testContext.Accounts, sndAddr, 0, senderBalance)
	_, _ = vm.CreateAccount(testContext.Accounts, rcvAddr, 0, senderBalance)

	userName := utils.GenerateUserNameForMyDNSContract()
	txData := []byte("register@" + hex.EncodeToString(userName))
	// create user name for sender
	tx := vm.CreateTransaction(0, big.NewInt(0), sndAddr, scAddress, gasPrice, gasLimit, txData)
	retCode, err := testContext.TxProcessor.ProcessTransaction(tx)
	require.Equal(t, vmcommon.Ok, retCode)
	require.Nil(t, err)
	require.Nil(t, testContext.GetLatestError())

	vm.TestAccount(t, testContext.Accounts, sndAddr, 1, big.NewInt(9361230))
	// check accumulated fees
	accumulatedFees := testContext.TxFeeHandler.GetAccumulatedFees()
	require.Equal(t, big.NewInt(638770), accumulatedFees)

	developerFees := testContext.TxFeeHandler.GetDeveloperFees()
	require.Equal(t, big.NewInt(63847), developerFees)

	intermediateTxs := testContext.GetIntermediateTransactions(t)
	testIndexer := vm.CreateTestIndexer(t, testContext.ShardCoordinator, testContext.EconomicsData)
	testIndexer.SaveTransaction(tx, block.TxBlock, intermediateTxs)

	indexerTx := testIndexer.GetIndexerPreparedTransaction(t)
	require.Equal(t, uint64(63877), indexerTx.GasUsed)
	require.Equal(t, "638770", indexerTx.Fee)

	utils.CleanAccumulatedIntermediateTransactions(t, &testContext)

	// create user name for receiver
	rcvUserName := utils.GenerateUserNameForMyDNSContract()
	txData = []byte("register@" + hex.EncodeToString(rcvUserName))
	tx = vm.CreateTransaction(0, big.NewInt(0), rcvAddr, scAddress, gasPrice, gasLimit, txData)
	retCode, err = testContext.TxProcessor.ProcessTransaction(tx)
	require.Equal(t, vmcommon.Ok, retCode)
	require.Nil(t, err)
	require.Nil(t, testContext.GetLatestError())

	_, err = testContext.Accounts.Commit()
	require.Nil(t, err)

	vm.TestAccount(t, testContext.Accounts, rcvAddr, 1, big.NewInt(9361230))
	// check accumulated fees
	accumulatedFees = testContext.TxFeeHandler.GetAccumulatedFees()
	require.Equal(t, big.NewInt(1277540), accumulatedFees)

	developerFees = testContext.TxFeeHandler.GetDeveloperFees()
	require.Equal(t, big.NewInt(127694), developerFees)

	ret := vm.GetVmOutput(nil, testContext.Accounts, scAddress, "resolve", userName)
	dnsUserNameAddr := ret.ReturnData[0]
	require.Equal(t, sndAddr, dnsUserNameAddr)

	ret = vm.GetVmOutput(nil, testContext.Accounts, scAddress, "resolve", rcvUserName)
	dnsUserNameAddr = ret.ReturnData[0]
	require.Equal(t, rcvAddr, dnsUserNameAddr)

	acc, _ := testContext.Accounts.GetExistingAccount(sndAddr)
	account, _ := acc.(state.UserAccountHandler)
	un := account.GetUserName()
	require.Equal(t, userName, un)

	acc, _ = testContext.Accounts.GetExistingAccount(rcvAddr)
	account, _ = acc.(state.UserAccountHandler)
	un = account.GetUserName()
	require.Equal(t, rcvUserName, un)

	intermediateTxs = testContext.GetIntermediateTransactions(t)
	testIndexer = vm.CreateTestIndexer(t, testContext.ShardCoordinator, testContext.EconomicsData)
	testIndexer.SaveTransaction(tx, block.TxBlock, intermediateTxs)

	indexerTx = testIndexer.GetIndexerPreparedTransaction(t)
	require.Equal(t, uint64(63877), indexerTx.GasUsed)
	require.Equal(t, "638770", indexerTx.Fee)

	gasLimit = 10
	tx = vm.CreateTransaction(1, big.NewInt(0), sndAddr, rcvAddr, gasPrice, gasLimit, nil)
	tx.SndUserName = userName
	tx.RcvUserName = rcvUserName

	retCode, err = testContext.TxProcessor.ProcessTransaction(tx)
	require.Equal(t, vmcommon.Ok, retCode)
	require.Nil(t, err)
	require.Equal(t, nil, testContext.GetLatestError())

	testIndexer = vm.CreateTestIndexer(t, testContext.ShardCoordinator, testContext.EconomicsData)
	testIndexer.SaveTransaction(tx, block.TxBlock, nil)

	indexerTx = testIndexer.GetIndexerPreparedTransaction(t)
	require.Equal(t, uint64(1), indexerTx.GasUsed)
	require.Equal(t, "10", indexerTx.Fee)
	require.Equal(t, rcvUserName, indexerTx.ReceiverUserName)
	require.Equal(t, userName, indexerTx.SenderUserName)
}
