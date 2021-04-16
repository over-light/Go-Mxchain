package process

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/vmcommon"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/esdt"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	testVm "github.com/ElrondNetwork/elrond-go/integrationTests/vm"
	"github.com/ElrondNetwork/elrond-go/integrationTests/vm/arwen"
	esdtCommon "github.com/ElrondNetwork/elrond-go/integrationTests/vm/esdt"
	"github.com/ElrondNetwork/elrond-go/process"
	vmFactory "github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/builtInFunctions"
	"github.com/ElrondNetwork/elrond-go/testscommon/txDataBuilder"
	"github.com/ElrondNetwork/elrond-go/vm"
	"github.com/ElrondNetwork/elrond-go/vm/systemSmartContracts"
	"github.com/stretchr/testify/require"
)

func TestESDTIssueAndTransactionsOnMultiShardEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 2
	nodesPerShard := 2
	numMetachainNodes := 2

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := int64(10000000000)
	integrationTests.MintAllNodes(nodes, big.NewInt(initialVal))

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// send token issue

	initialSupply := int64(10000000000)
	ticker := "TCK"
	esdtCommon.IssueTestToken(nodes, initialSupply, ticker)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 12
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(integrationTests.GetTokenIdentifier(nodes, []byte(ticker)))

	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply)

	txData := txDataBuilder.NewBuilder()

	// send tx to other nodes
	valueToSend := int64(100)
	for _, node := range nodes[1:] {
		txData = txData.Clear().TransferESDT(tokenIdentifier, valueToSend)
		integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), node.OwnAccount.Address, txData.ToString(), integrationTests.AdditionalGasLimit)
	}

	mintValue := int64(10000)
	txData = txData.Clear().Func("mint").Str(tokenIdentifier).Int64(mintValue)
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vm.ESDTSCAddress, txData.ToString(), core.MinMetaTxExtraGasCost)

	txData.Clear().Func("freeze").Str(tokenIdentifier).Bytes(nodes[2].OwnAccount.Address)
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vm.ESDTSCAddress, txData.ToString(), core.MinMetaTxExtraGasCost)

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	finalSupply := initialSupply + mintValue
	for _, node := range nodes[1:] {
		esdtCommon.CheckAddressHasESDTTokens(t, node.OwnAccount.Address, nodes, tokenIdentifier, valueToSend)
		finalSupply = finalSupply - valueToSend
	}

	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, finalSupply)

	txData.Clear().BurnESDT(tokenIdentifier, mintValue)
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vm.ESDTSCAddress, txData.ToString(), core.MinMetaTxExtraGasCost)

	txData.Clear().Func("freeze").Str(tokenIdentifier).Bytes(nodes[1].OwnAccount.Address)
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vm.ESDTSCAddress, txData.ToString(), core.MinMetaTxExtraGasCost)

	txData.Clear().Func("wipe").Str(tokenIdentifier).Bytes(nodes[2].OwnAccount.Address)
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vm.ESDTSCAddress, txData.ToString(), core.MinMetaTxExtraGasCost)

	txData.Clear().Func("pause").Str(tokenIdentifier)
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vm.ESDTSCAddress, txData.ToString(), core.MinMetaTxExtraGasCost)

	time.Sleep(time.Second)

	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	esdtFrozenData := esdtCommon.GetESDTTokenData(t, nodes[1].OwnAccount.Address, nodes, tokenIdentifier)
	esdtUserMetaData := builtInFunctions.ESDTUserMetadataFromBytes(esdtFrozenData.Properties)
	require.True(t, esdtUserMetaData.Frozen)

	wipedAcc := esdtCommon.GetUserAccountWithAddress(t, nodes[2].OwnAccount.Address, nodes)
	tokenKey := []byte(core.ElrondProtectedKeyPrefix + "esdt" + tokenIdentifier)
	retrievedData, _ := wipedAcc.DataTrieTracker().RetrieveValue(tokenKey)
	require.Equal(t, 0, len(retrievedData))

	systemSCAcc := esdtCommon.GetUserAccountWithAddress(t, core.SystemAccountAddress, nodes)
	retrievedData, _ = systemSCAcc.DataTrieTracker().RetrieveValue(tokenKey)
	esdtGlobalMetaData := builtInFunctions.ESDTGlobalMetadataFromBytes(retrievedData)
	require.True(t, esdtGlobalMetaData.Paused)

	finalSupply = finalSupply - mintValue
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, finalSupply)

	esdtSCAcc := esdtCommon.GetUserAccountWithAddress(t, vm.ESDTSCAddress, nodes)
	retrievedData, _ = esdtSCAcc.DataTrieTracker().RetrieveValue([]byte(tokenIdentifier))
	tokenInSystemSC := &systemSmartContracts.ESDTData{}
	_ = integrationTests.TestMarshalizer.Unmarshal(tokenInSystemSC, retrievedData)
	require.Zero(t, tokenInSystemSC.MintedValue.Cmp(big.NewInt(initialSupply+mintValue)))
	require.Zero(t, tokenInSystemSC.BurntValue.Cmp(big.NewInt(mintValue)))
	require.True(t, tokenInSystemSC.IsPaused)
}

func TestESDTCallBurnOnANonBurnableToken(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 2
	nodesPerShard := 2
	numMetachainNodes := 2

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := big.NewInt(10000000000)
	integrationTests.MintAllNodes(nodes, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// send token issue
	ticker := "ALC"
	issuePrice := big.NewInt(1000)
	initialSupply := int64(10000000000)
	tokenIssuer := nodes[0]
	txData := txDataBuilder.NewBuilder()

	txData.Clear().IssueESDT("aliceToken", ticker, initialSupply, 6)
	txData.CanFreeze(true).CanWipe(true).CanPause(true).CanMint(true).CanBurn(false)
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, issuePrice, vm.ESDTSCAddress, txData.ToString(), core.MinMetaTxExtraGasCost)

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 12
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(integrationTests.GetTokenIdentifier(nodes, []byte(ticker)))

	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply)

	// send tx to other nodes
	valueToSend := int64(100)
	for _, node := range nodes[1:] {
		txData.Clear().TransferESDT(tokenIdentifier, valueToSend)
		integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), node.OwnAccount.Address, txData.ToString(), integrationTests.AdditionalGasLimit)
	}

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	finalSupply := initialSupply
	for _, node := range nodes[1:] {
		esdtCommon.CheckAddressHasESDTTokens(t, node.OwnAccount.Address, nodes, tokenIdentifier, valueToSend)
		finalSupply = finalSupply - valueToSend
	}

	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, finalSupply)

	burnValue := int64(77)
	txData.Clear().BurnESDT(tokenIdentifier, burnValue)
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vm.ESDTSCAddress, txData.ToString(), core.MinMetaTxExtraGasCost)

	time.Sleep(time.Second)

	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	esdtSCAcc := esdtCommon.GetUserAccountWithAddress(t, vm.ESDTSCAddress, nodes)
	retrievedData, _ := esdtSCAcc.DataTrieTracker().RetrieveValue([]byte(tokenIdentifier))
	tokenInSystemSC := &systemSmartContracts.ESDTData{}
	_ = integrationTests.TestMarshalizer.Unmarshal(tokenInSystemSC, retrievedData)
	require.Equal(t, initialSupply, tokenInSystemSC.MintedValue.Int64())
	require.Zero(t, tokenInSystemSC.BurntValue.Int64())

	// if everything is ok, the caller should have received the amount of burnt tokens back because canBurn = false
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, finalSupply)
}

func TestESDTIssueFromASmartContractSimulated(t *testing.T) {
	metaNode := integrationTests.NewTestProcessorNode(1, core.MetachainShardId, 0)
	defer func() {
		_ = metaNode.Messenger.Close()
	}()

	txData := txDataBuilder.NewBuilder()

	ticker := "RBT"
	issuePrice := big.NewInt(1000)
	initialSupply := big.NewInt(10000000000)
	numDecimals := byte(6)

	txData.Clear().IssueESDT("robertWhyNot", ticker, initialSupply.Int64(), numDecimals)
	txData.CanFreeze(true).CanWipe(true).CanPause(true).CanMint(true).CanBurn(true).Int(1000)
	scr := &smartContractResult.SmartContractResult{
		Nonce:          0,
		Value:          issuePrice,
		RcvAddr:        vm.ESDTSCAddress,
		SndAddr:        metaNode.OwnAccount.Address,
		Data:           txData.ToBytes(),
		PrevTxHash:     []byte("hash"),
		OriginalTxHash: []byte("hash"),
		GasLimit:       10000000,
		GasPrice:       1,
		CallType:       vmcommon.AsynchronousCall,
		OriginalSender: metaNode.OwnAccount.Address,
	}

	returnCode, err := metaNode.ScProcessor.ProcessSmartContractResult(scr)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, returnCode)

	interimProc, _ := metaNode.InterimProcContainer.Get(block.SmartContractResultBlock)
	mapCreatedSCRs := interimProc.GetAllCurrentFinishedTxs()

	require.Equal(t, len(mapCreatedSCRs), 1)
	for _, addedSCR := range mapCreatedSCRs {
		strings.Contains(string(addedSCR.GetData()), core.BuiltInFunctionESDTTransfer)
	}
}

func TestScSendsEsdtToUserWithMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 2
	nodesPerShard := 2
	numMetachainNodes := 2

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := big.NewInt(10000000000)
	integrationTests.MintAllNodes(nodes, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// send token issue
	initialSupply := int64(10000000000)
	ticker := "TCK"
	esdtCommon.IssueTestToken(nodes, initialSupply, ticker)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 12
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(integrationTests.GetTokenIdentifier(nodes, []byte(ticker)))
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply)

	// deploy the smart contract

	vaultScCode := arwen.GetSCCode("../testdata/vault.wasm")
	vaultScAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(vaultScCode),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err := nodes[0].AccntState.GetExistingAccount(vaultScAddress)
	require.Nil(t, err)

	txData := txDataBuilder.NewBuilder()

	// feed funds to the vault
	valueToSendToSc := int64(1000)
	txData.Clear().TransferESDT(tokenIdentifier, valueToSendToSc)
	txData.Str("accept_funds")
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vaultScAddress, txData.ToString(), integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply-valueToSendToSc)
	esdtCommon.CheckAddressHasESDTTokens(t, vaultScAddress, nodes, tokenIdentifier, valueToSendToSc)

	// take them back, with a message
	valueToRequest := valueToSendToSc / 4
	txData.Clear().Func("retrieve_funds").Str(tokenIdentifier).Int64(valueToRequest).Str("ESDT transfer message")
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vaultScAddress, txData.ToString(), integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply-valueToSendToSc+valueToRequest)
	esdtCommon.CheckAddressHasESDTTokens(t, vaultScAddress, nodes, tokenIdentifier, valueToSendToSc-valueToRequest)
}

func TestESDTcallsSC(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 2
	nodesPerShard := 2
	numMetachainNodes := 2

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := big.NewInt(10000000000)
	integrationTests.MintAllNodes(nodes, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// send token issue

	initialSupply := int64(10000000000)
	ticker := "TCK"
	esdtCommon.IssueTestToken(nodes, initialSupply, ticker)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 12
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(integrationTests.GetTokenIdentifier(nodes, []byte(ticker)))
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply)

	// send tx to other nodes
	txData := txDataBuilder.NewBuilder()
	valueToSend := int64(100)
	for _, node := range nodes[1:] {
		txData.Clear().TransferESDT(tokenIdentifier, valueToSend)
		integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), node.OwnAccount.Address, txData.ToString(), integrationTests.AdditionalGasLimit)
	}

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	numNodesWithoutIssuer := int64(len(nodes) - 1)
	issuerBalance := initialSupply - valueToSend*numNodesWithoutIssuer
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, issuerBalance)
	for i := 1; i < len(nodes); i++ {
		esdtCommon.CheckAddressHasESDTTokens(t, nodes[i].OwnAccount.Address, nodes, tokenIdentifier, valueToSend)
	}

	// deploy the smart contract
	scCode := arwen.GetSCCode("../testdata/crowdfunding-esdt.wasm")
	scAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(scCode)+"@"+
			hex.EncodeToString(big.NewInt(1000).Bytes())+"@"+
			hex.EncodeToString(big.NewInt(1000).Bytes())+"@"+
			hex.EncodeToString([]byte(tokenIdentifier)),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err := nodes[0].AccntState.GetExistingAccount(scAddress)
	require.Nil(t, err)

	// call sc with esdt
	valueToSendToSc := int64(10)
	for _, node := range nodes {
		txData.Clear().TransferESDT(tokenIdentifier, valueToSendToSc).Str("fund")
		integrationTests.CreateAndSendTransaction(node, nodes, big.NewInt(0), scAddress, txData.ToString(), integrationTests.AdditionalGasLimit)
	}

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	scQuery1 := &process.SCQuery{
		ScAddress: scAddress,
		FuncName:  "currentFunds",
		Arguments: [][]byte{},
	}
	vmOutput1, _ := nodes[0].SCQueryService.ExecuteQuery(scQuery1)
	require.Equal(t, big.NewInt(60).Bytes(), vmOutput1.ReturnData[0])

	nodesBalance := valueToSend - valueToSendToSc
	issuerBalance = issuerBalance - valueToSendToSc
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, issuerBalance)
	for i := 1; i < len(nodes); i++ {
		esdtCommon.CheckAddressHasESDTTokens(t, nodes[i].OwnAccount.Address, nodes, tokenIdentifier, nodesBalance)
	}
}

func TestScCallsScWithEsdtIntraShard(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 1
	nodesPerShard := 1
	numMetachainNodes := 1

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := big.NewInt(10000000000)
	integrationTests.MintAllNodes(nodes, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// send token issue
	initialSupply := int64(10000000000)
	ticker := "TCK"
	esdtCommon.IssueTestToken(nodes, initialSupply, ticker)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 12
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(integrationTests.GetTokenIdentifier(nodes, []byte(ticker)))
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply)

	// deploy the smart contracts

	vaultCode := arwen.GetSCCode("../testdata/vault.wasm")
	vault, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(vaultCode),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err := nodes[0].AccntState.GetExistingAccount(vault)
	require.Nil(t, err)

	forwarderCode := arwen.GetSCCode("../testdata/forwarder-raw.wasm")
	forwarder, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(forwarderCode),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err = nodes[0].AccntState.GetExistingAccount(forwarder)
	require.Nil(t, err)

	txData := txDataBuilder.NewBuilder()

	// call forwarder with esdt, and forwarder automatically calls second sc
	valueToSendToSc := int64(1000)
	txData.TransferESDT(tokenIdentifier, valueToSendToSc)
	txData.Str("forward_async_call_half_payment").Bytes(vault).Str("accept_funds")
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), forwarder, txData.ToString(), integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIssuerBalance := initialSupply - valueToSendToSc
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, tokenIssuerBalance)
	esdtCommon.CheckAddressHasESDTTokens(t, forwarder, nodes, tokenIdentifier, valueToSendToSc/2)
	esdtCommon.CheckAddressHasESDTTokens(t, vault, nodes, tokenIdentifier, valueToSendToSc/2)

	esdtCommon.CheckNumCallBacks(t, forwarder, nodes, 1)
	esdtCommon.CheckSavedCallBackData(t, forwarder, nodes, 1, "EGLD", big.NewInt(0), vmcommon.Ok, [][]byte{})

	// call forwarder to ask the second one to send it back some esdt
	valueToRequest := valueToSendToSc / 4
	txData.Clear().Func("forward_async_call").Bytes(vault).Str("retrieve_funds").Str(tokenIdentifier).Int64(valueToRequest)

	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), forwarder, txData.ToString(), integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	time.Sleep(time.Second)

	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, tokenIssuerBalance)
	esdtCommon.CheckAddressHasESDTTokens(t, forwarder, nodes, tokenIdentifier, valueToSendToSc*3/4)
	esdtCommon.CheckAddressHasESDTTokens(t, vault, nodes, tokenIdentifier, valueToSendToSc/4)

	esdtCommon.CheckNumCallBacks(t, forwarder, nodes, 2)
	esdtCommon.CheckSavedCallBackData(t, forwarder, nodes, 2, tokenIdentifier, big.NewInt(valueToRequest), vmcommon.Ok, [][]byte{})

	// call forwarder to ask the second one to execute a method
	valueToTransferWithExecSc := valueToSendToSc / 4
	txData.Clear().TransferESDT(tokenIdentifier, valueToTransferWithExecSc)
	txData.Str("forward_transf_exec").Bytes(vault).Str("accept_funds")
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), forwarder, txData.ToString(), integrationTests.AdditionalGasLimit)

	time.Sleep(5 * time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	time.Sleep(5 * time.Second)

	tokenIssuerBalance -= valueToTransferWithExecSc
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, tokenIssuerBalance)
	esdtCommon.CheckAddressHasESDTTokens(t, forwarder, nodes, tokenIdentifier, valueToSendToSc*3/4)
	esdtCommon.CheckAddressHasESDTTokens(t, vault, nodes, tokenIdentifier, valueToSendToSc/2)

	// call forwarder to ask the second one to execute a method that transfers ESDT twice, with execution
	valueToTransferWithExecSc = valueToSendToSc / 10
	txData.Clear().TransferESDT(tokenIdentifier, valueToTransferWithExecSc)
	txData.Str("forward_transf_exec_twice").Bytes(vault).Str("accept_funds")
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), forwarder, txData.ToString(), integrationTests.AdditionalGasLimit)

	time.Sleep(5 * time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	time.Sleep(5 * time.Second)

	tokenIssuerBalance -= valueToTransferWithExecSc
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, tokenIssuerBalance)
	esdtCommon.CheckAddressHasESDTTokens(t, forwarder, nodes, tokenIdentifier, valueToSendToSc*3/4)
	esdtCommon.CheckAddressHasESDTTokens(t, vault, nodes, tokenIdentifier, valueToSendToSc/2+valueToTransferWithExecSc)
}

func TestCallbackPaymentEgld(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 1
	nodesPerShard := 1
	numMetachainNodes := 1

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := big.NewInt(10000000000)
	integrationTests.MintAllNodes(nodes, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// send token issue
	initialSupply := int64(10000000000)
	ticker := "TCK"
	esdtCommon.IssueTestToken(nodes, initialSupply, ticker)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 12
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(integrationTests.GetTokenIdentifier(nodes, []byte(ticker)))
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply)

	// deploy the smart contracts

	vaultCode := arwen.GetSCCode("../testdata/vault.wasm")
	secondScAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(vaultCode),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err := nodes[0].AccntState.GetExistingAccount(secondScAddress)
	require.Nil(t, err)

	forwarderCode := arwen.GetSCCode("../testdata/forwarder-raw.wasm")
	forwarder, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(forwarderCode),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err = nodes[0].AccntState.GetExistingAccount(forwarder)
	require.Nil(t, err)

	txData := txDataBuilder.NewBuilder()
	// call first sc with esdt, and first sc automatically calls second sc
	valueToSendToSc := int64(1000)
	txData.Clear().Func("forward_async_call_half_payment").Bytes(secondScAddress).Str("accept_funds")
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(valueToSendToSc), forwarder, txData.ToString(), integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 1, nonce, round, idxProposers)
	time.Sleep(time.Second)

	esdtCommon.CheckNumCallBacks(t, forwarder, nodes, 1)
	esdtCommon.CheckSavedCallBackData(t, forwarder, nodes, 1, "EGLD", big.NewInt(0), vmcommon.Ok, [][]byte{})

	// call first sc to ask the second one to send it back some esdt
	valueToRequest := valueToSendToSc / 4
	txData.Clear().Func("forward_async_call").Bytes(secondScAddress).Str("retrieve_funds").Str("EGLD").Int64(valueToRequest)
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), forwarder, txData.ToString(), integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 1, nonce, round, idxProposers)
	time.Sleep(time.Second)

	esdtCommon.CheckNumCallBacks(t, forwarder, nodes, 2)
	esdtCommon.CheckSavedCallBackData(t, forwarder, nodes, 2, "EGLD", big.NewInt(valueToRequest), vmcommon.Ok, [][]byte{})
}

func TestScCallsScWithEsdtCrossShard(t *testing.T) {
	t.Skip("test is not ready yet")

	numOfShards := 2
	nodesPerShard := 2
	numMetachainNodes := 2

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := big.NewInt(10000000000)
	integrationTests.MintAllNodes(nodes, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// send token issue

	initialSupply := int64(10000000000)
	ticker := "TCK"
	esdtCommon.IssueTestToken(nodes, initialSupply, ticker)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 12
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(integrationTests.GetTokenIdentifier(nodes, []byte(ticker)))
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply)

	// deploy the smart contracts

	vaultCode := arwen.GetSCCode("../testdata/vault.wasm")
	secondScAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(vaultCode),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err := nodes[0].AccntState.GetExistingAccount(secondScAddress)
	require.Nil(t, err)

	forwarderCode := arwen.GetSCCode("../testdata/forwarder-raw.wasm")
	forwarder, _ := nodes[2].BlockchainHook.NewAddress(nodes[2].OwnAccount.Address, nodes[2].OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)
	integrationTests.CreateAndSendTransaction(
		nodes[2],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(forwarderCode),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err = nodes[2].AccntState.GetExistingAccount(forwarder)
	require.Nil(t, err)

	txData := txDataBuilder.NewBuilder()

	// call forwarder with esdt, and the forwarder automatically calls second sc
	valueToSendToSc := int64(1000)
	txData.Clear().TransferESDT(tokenIdentifier, valueToSendToSc)
	txData.Str("forward_async_call_half_payment").Bytes(secondScAddress).Str("accept_funds")
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), forwarder, txData.ToString(), integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply-valueToSendToSc)
	esdtCommon.CheckAddressHasESDTTokens(t, forwarder, nodes, tokenIdentifier, valueToSendToSc/2)
	esdtCommon.CheckAddressHasESDTTokens(t, secondScAddress, nodes, tokenIdentifier, valueToSendToSc/2)

	esdtCommon.CheckNumCallBacks(t, forwarder, nodes, 1)
	esdtCommon.CheckSavedCallBackData(t, forwarder, nodes, 1, "EGLD", big.NewInt(0), vmcommon.Ok, [][]byte{})

	// call forwarder to ask the second one to send it back some esdt
	valueToRequest := valueToSendToSc / 4
	txData.Clear().Func("forward_async_call").Bytes(secondScAddress)
	txData.Str("retrieve_funds").Str(tokenIdentifier).Int64(valueToRequest)
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), forwarder, txData.ToString(), integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	esdtCommon.CheckAddressHasESDTTokens(t, forwarder, nodes, tokenIdentifier, valueToSendToSc*3/4)
	esdtCommon.CheckAddressHasESDTTokens(t, secondScAddress, nodes, tokenIdentifier, valueToSendToSc/4)

	esdtCommon.CheckNumCallBacks(t, forwarder, nodes, 2)
	esdtCommon.CheckSavedCallBackData(t, forwarder, nodes, 1, tokenIdentifier, big.NewInt(valueToSendToSc), vmcommon.Ok, [][]byte{})
}

func TestScCallsScWithEsdtIntraShard_SecondScRefusesPayment(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 1
	nodesPerShard := 1
	numMetachainNodes := 1

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := big.NewInt(10000000000)
	integrationTests.MintAllNodes(nodes, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// send token issue
	initialSupply := int64(10000000000)
	ticker := "TCK"
	esdtCommon.IssueTestToken(nodes, initialSupply, ticker)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 12
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(integrationTests.GetTokenIdentifier(nodes, []byte(ticker)))
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply)

	// deploy the smart contracts

	secondScCode := arwen.GetSCCode("../testdata/second-contract.wasm")
	secondScAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxDataNonPayable(secondScCode)+"@"+
			hex.EncodeToString([]byte(tokenIdentifier)),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 2, nonce, round, idxProposers)
	_, err := nodes[0].AccntState.GetExistingAccount(secondScAddress)
	require.Nil(t, err)

	firstScCode := arwen.GetSCCode("../testdata/first-contract.wasm")
	firstScAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxDataNonPayable(firstScCode)+"@"+
			hex.EncodeToString([]byte(tokenIdentifier))+"@"+
			hex.EncodeToString(secondScAddress),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 2, nonce, round, idxProposers)
	_, err = nodes[0].AccntState.GetExistingAccount(firstScAddress)
	require.Nil(t, err)

	nonce, round = transferRejectedBySecondContract(t, nonce, round, nodes, tokenIssuer, idxProposers, initialSupply, tokenIdentifier, firstScAddress, secondScAddress, "transferToSecondContractRejected", 2)
	_, _ = transferRejectedBySecondContract(t, nonce, round, nodes, tokenIssuer, idxProposers, initialSupply, tokenIdentifier, firstScAddress, secondScAddress, "transferToSecondContractRejectedWithTransferAndExecute", 2)
}

func TestScACallsScBWithExecOnDestESDT_TxPending(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 1
	nodesPerShard := 1
	numMetachainNodes := 1

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := big.NewInt(10000000000)
	integrationTests.MintAllNodes(nodes, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// send token issue
	initialSupply := int64(10000000000)
	ticker := "TCK"
	esdtCommon.IssueTestToken(nodes, initialSupply, ticker)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 15
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(integrationTests.GetTokenIdentifier(nodes, []byte(ticker)))
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply)

	// deploy smart contracts

	callerScCode := arwen.GetSCCode("../testdata/exec-on-dest-caller.wasm")
	callerScAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxDataNonPayable(callerScCode),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 2, nonce, round, idxProposers)
	_, err := nodes[0].AccntState.GetExistingAccount(callerScAddress)
	require.Nil(t, err)

	receiverScCode := arwen.GetSCCode("../testdata/exec-on-dest-receiver.wasm")
	receiverScAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxDataNonPayable(receiverScCode)+"@"+
			hex.EncodeToString([]byte(tokenIdentifier)),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 2, nonce, round, idxProposers)
	_, err = nodes[0].AccntState.GetExistingAccount(receiverScAddress)
	require.Nil(t, err)

	// set receiver address in caller contract | map[ticker] -> receiverAddress

	txData := txDataBuilder.NewBuilder()
	txData.Clear().
		Func("setPoolAddress").
		Str(tokenIdentifier).
		Str(string(receiverScAddress))

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		callerScAddress,
		txData.ToString(),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 2, nonce, round, idxProposers)
	_, err = nodes[0].AccntState.GetExistingAccount(callerScAddress)
	require.Nil(t, err)

	// issue 1:1 esdt:interestEsdt in receiver contract

	txData = txDataBuilder.NewBuilder()
	issueTokenSupply := big.NewInt(100000000) // 100 tokens
	issueTokenDecimals := 6
	issuePrice := big.NewInt(1000)
	txData.Clear().
		Func("issue").
		Str(tokenIdentifier).
		Str("token-name").
		Str("L").
		BigInt(issueTokenSupply).
		Int(issueTokenDecimals)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		issuePrice,
		receiverScAddress,
		txData.ToString(),
		integrationTests.AdditionalGasLimit,
	)

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	// call caller sc with ESDTTransfer which will call the second sc with execute_on_dest_context
	txData = txDataBuilder.NewBuilder()
	valueToTransfer := int64(1000)
	txData.Clear().
		TransferESDT(tokenIdentifier, valueToTransfer).
		Str("deposit").
		Str(string(callerScAddress))

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		callerScAddress,
		txData.ToString(),
		integrationTests.AdditionalGasLimit,
	)

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply-valueToTransfer)

	// should be int64(1000)
	esdtData := esdtCommon.GetESDTTokenData(t, receiverScAddress, nodes, tokenIdentifier)
	require.EqualValues(t, &esdt.ESDigitalToken{Value: big.NewInt(valueToTransfer)}, esdtData)

	// no tokens in caller contract
	esdtData = esdtCommon.GetESDTTokenData(t, callerScAddress, nodes, tokenIdentifier)
	require.EqualValues(t, &esdt.ESDigitalToken{}, esdtData)
}

func TestScACallsScBWithExecOnDestScAPerformsAsyncCall_NoCallbackInScB(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 1
	nodesPerShard := 1
	numMetachainNodes := 1

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	for _, n := range nodes {
		n.EconomicsData.SetMaxGasLimitPerBlock(1500000000)
	}

	initialVal := big.NewInt(10000000000)
	integrationTests.MintAllNodes(nodes, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	tokenIssuer := nodes[0]

	// deploy parent contract
	callerScCode := arwen.GetSCCode("../testdata/parent.wasm")
	callerScAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxDataNonPayable(callerScCode),
		integrationTests.AdditionalGasLimit,
	)

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 2, nonce, round, idxProposers)
	_, err := nodes[0].AccntState.GetExistingAccount(callerScAddress)
	require.Nil(t, err)

	// deploy child contract by calling deployChildContract endpoint
	receiverScCode := arwen.GetSCCode("../testdata/child.wasm")
	txDeployData := txDataBuilder.NewBuilder()
	txDeployData.Func("deployChildContract").Str(receiverScCode)

	indirectDeploy := "deployChildContract@" + receiverScCode
	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		callerScAddress,
		indirectDeploy,
		integrationTests.AdditionalGasLimit,
	)

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 2, nonce, round, idxProposers)
	time.Sleep(time.Second)

	// issue ESDT by calling exec on dest context on child contract
	ticker := "DSN"
	name := "DisplayName"
	issueCost := big.NewInt(1000)
	txIssueData := txDataBuilder.NewBuilder()
	txIssueData.Func("executeOnDestIssueToken").
		Str(name).
		Str(ticker).
		BigInt(big.NewInt(500000))

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		issueCost,
		callerScAddress,
		txIssueData.ToString(),
		1000000000,
	)

	nrRoundsToPropagateMultiShard := 12
	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenID := integrationTests.GetTokenIdentifier(nodes, []byte(ticker))

	scQuery := nodes[0].SCQueryService
	childScAddressQuery := &process.SCQuery{
		ScAddress:  callerScAddress,
		FuncName:   "getChildContractAddress",
		CallerAddr: nil,
		CallValue:  big.NewInt(0),
		Arguments:  [][]byte{},
	}

	res, err := scQuery.ExecuteQuery(childScAddressQuery)
	require.Nil(t, err)

	receiverScAddress := res.ReturnData[0]

	tokenIdQuery := &process.SCQuery{
		ScAddress:  receiverScAddress,
		FuncName:   "getWrappedEgldTokenIdentifier",
		CallerAddr: nil,
		CallValue:  big.NewInt(0),
		Arguments:  [][]byte{},
	}

	res, err = scQuery.ExecuteQuery(tokenIdQuery)
	require.Nil(t, err)
	require.True(t, strings.Contains(string(res.ReturnData[0]), ticker))

	esdtCommon.CheckAddressHasESDTTokens(t, receiverScAddress, nodes, string(tokenID), 500000)
}

func TestIssueESDT_FromSCWithNotEnoughGasOnCallBack(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 3
	nodesPerShard := 1
	numMetachainNodes := 1

	gasSchedule, _ := core.LoadGasScheduleConfig("../../../../cmd/node/config/gasSchedules/gasScheduleV3.toml")
	nodes := integrationTests.CreateNodesWithGasSchedule(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		gasSchedule,
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	for _, n := range nodes {
		n.EconomicsData.SetMaxGasLimitPerBlock(1500000000)
		if check.IfNil(n.SystemSCFactory) {
			continue
		}
		gasScheduleHandler := n.SystemSCFactory.(core.GasScheduleSubscribeHandler)
		gasScheduleHandler.GasScheduleChange(gasSchedule)
	}

	initialVal := big.NewInt(10000000000)
	integrationTests.MintAllNodes(nodes, big.NewInt(0).Mul(initialVal, initialVal))

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	scCode := arwen.GetSCCode("../testdata/local-esdt-and-nft.wasm")
	scAddress, _ := nodes[0].BlockchainHook.NewAddress(nodes[0].OwnAccount.Address, nodes[0].OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxDataNonPayable(scCode),
		500000000,
	)
	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 2, nonce, round, idxProposers)
	_, err := nodes[0].AccntState.GetExistingAccount(scAddress)
	require.Nil(t, err)

	alice := nodes[1]
	issuePrice := big.NewInt(1000)
	txData := []byte("issueFungibleToken" + "@" + hex.EncodeToString([]byte("TOKEN")) +
		"@" + hex.EncodeToString([]byte("TKR")) + "@" + hex.EncodeToString(big.NewInt(1).Bytes()))
	integrationTests.CreateAndSendTransaction(
		alice,
		nodes,
		issuePrice,
		scAddress,
		string(txData),
		40000000,
	)

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 2, nonce, round, idxProposers)
	time.Sleep(time.Second)

	userAccount := esdtCommon.GetUserAccountWithAddress(t, alice.OwnAccount.Address, nodes)
	balanceAfterTransfer := userAccount.GetBalance()

	nrRoundsToPropagateMultiShard := 25
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)
	userAccount = esdtCommon.GetUserAccountWithAddress(t, alice.OwnAccount.Address, nodes)
	require.Equal(t, userAccount.GetBalance(), balanceAfterTransfer)

	scAccount := esdtCommon.GetUserAccountWithAddress(t, scAddress, nodes)
	require.Equal(t, scAccount.GetBalance(), issuePrice)
}

func TestIssueESDT_FromSCWithNotEnoughGasSendMoneyToUserOnCallBack(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 3
	nodesPerShard := 1
	numMetachainNodes := 1

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	gasSchedule, _ := core.LoadGasScheduleConfig("../../../../cmd/node/config/gasSchedules/gasScheduleV3.toml")
	for _, n := range nodes {
		n.EconomicsData.SetMaxGasLimitPerBlock(1500000000)
		if check.IfNil(n.SystemSCFactory) {
			continue
		}
		gasScheduleHandler := n.SystemSCFactory.(core.GasScheduleSubscribeHandler)
		gasScheduleHandler.GasScheduleChange(gasSchedule)
	}

	initialVal := big.NewInt(10000000000)
	integrationTests.MintAllNodes(nodes, big.NewInt(0).Mul(initialVal, initialVal))

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	scCode := arwen.GetSCCode("../testdata/local-esdt-and-nft.wasm")
	scAddress, _ := nodes[0].BlockchainHook.NewAddress(nodes[0].OwnAccount.Address, nodes[0].OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxDataNonPayable(scCode),
		500000000,
	)
	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 2, nonce, round, idxProposers)
	_, err := nodes[0].AccntState.GetExistingAccount(scAddress)
	require.Nil(t, err)

	alice := nodes[1]
	issuePrice := big.NewInt(1000)
	txData := []byte("issueFungibleToken" + "@" + hex.EncodeToString([]byte("TOKEN")) +
		"@" + hex.EncodeToString([]byte("TKR")) + "@" + hex.EncodeToString(big.NewInt(1).Bytes()))
	integrationTests.CreateAndSendTransaction(
		alice,
		nodes,
		issuePrice,
		scAddress,
		string(txData),
		3000000,
	)

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 2, nonce, round, idxProposers)
	time.Sleep(time.Second)

	userAccount := esdtCommon.GetUserAccountWithAddress(t, alice.OwnAccount.Address, nodes)
	balanceAfterTransfer := userAccount.GetBalance()

	nrRoundsToPropagateMultiShard := 25
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)
	userAccount = esdtCommon.GetUserAccountWithAddress(t, alice.OwnAccount.Address, nodes)
	require.Equal(t, userAccount.GetBalance(), big.NewInt(0).Add(balanceAfterTransfer, issuePrice))
}

func TestIssueAndBurnESDT_MaxGasPerBlockExceeded(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numIssues := 22
	numBurns := 300

	numOfShards := 1
	nodesPerShard := 2
	numMetachainNodes := 1

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	gasSchedule, _ := core.LoadGasScheduleConfig("../../../../cmd/node/config/gasSchedules/gasScheduleV3.toml")
	for _, n := range nodes {
		n.EconomicsData.SetMaxGasLimitPerBlock(1500000000)
		if check.IfNil(n.SystemSCFactory) {
			continue
		}
		gasScheduleHandler := n.SystemSCFactory.(core.GasScheduleSubscribeHandler)
		gasScheduleHandler.GasScheduleChange(gasSchedule)
	}

	initialVal := big.NewInt(10000000000)
	integrationTests.MintAllNodes(nodes, big.NewInt(0).Mul(initialVal, initialVal))

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// send token issue

	initialSupply := int64(10000000000)
	ticker := "TCK"
	esdtCommon.IssueTestTokenWithCustomGas(nodes, initialSupply, ticker, 60000000)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 12
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(integrationTests.GetTokenIdentifier(nodes, []byte(ticker)))

	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply)

	tokenName := "token"
	issuePrice := big.NewInt(1000)

	txData := txDataBuilder.NewBuilder()
	txData.Clear().IssueESDT(tokenName, ticker, initialSupply, 6)
	txData.CanFreeze(true).CanWipe(true).CanPause(true).CanMint(true).CanBurn(true)
	for i := 0; i < numIssues; i++ {
		integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, issuePrice, vm.ESDTSCAddress, txData.ToString(), 60000000)
	}

	txDataBuilderObj := txDataBuilder.NewBuilder()
	txDataBuilderObj.Clear().Func("ESDTBurn").Str(tokenIdentifier).Int(1)

	burnTxData := txDataBuilderObj.ToString()
	for i := 0; i < numBurns; i++ {
		integrationTests.CreateAndSendTransaction(
			nodes[0],
			nodes,
			big.NewInt(0),
			vm.ESDTSCAddress,
			burnTxData,
			60000000,
		)
	}

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 15, nonce, round, idxProposers)
	time.Sleep(time.Second)

	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply-int64(numBurns))
}

func TestScCallsScWithEsdtCrossShard_SecondScRefusesPayment(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 2
	nodesPerShard := 2
	numMetachainNodes := 2

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := big.NewInt(10000000000)
	integrationTests.MintAllNodes(nodes, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// send token issue

	initialSupply := int64(10000000000)
	ticker := "TCK"
	esdtCommon.IssueTestToken(nodes, initialSupply, ticker)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 12
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(integrationTests.GetTokenIdentifier(nodes, []byte(ticker)))
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply)

	// deploy the smart contracts

	secondScCode := arwen.GetSCCode("../testdata/second-contract.wasm")
	secondScAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxDataNonPayable(secondScCode)+"@"+
			hex.EncodeToString([]byte(tokenIdentifier)),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err := nodes[0].AccntState.GetExistingAccount(secondScAddress)
	require.Nil(t, err)

	firstScCode := arwen.GetSCCode("../testdata/first-contract.wasm")
	firstScAddress, _ := nodes[2].BlockchainHook.NewAddress(nodes[2].OwnAccount.Address, nodes[2].OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)
	integrationTests.CreateAndSendTransaction(
		nodes[2],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxDataNonPayable(firstScCode)+"@"+
			hex.EncodeToString([]byte(tokenIdentifier))+"@"+
			hex.EncodeToString(secondScAddress),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err = nodes[2].AccntState.GetExistingAccount(firstScAddress)
	require.Nil(t, err)

	nonce, round = transferRejectedBySecondContract(t, nonce, round, nodes, tokenIssuer, idxProposers, initialSupply, tokenIdentifier, firstScAddress, secondScAddress, "transferToSecondContractRejected", 20)
	_, _ = transferRejectedBySecondContract(t, nonce, round, nodes, tokenIssuer, idxProposers, initialSupply, tokenIdentifier, firstScAddress, secondScAddress, "transferToSecondContractRejectedWithTransferAndExecute", 20)
}

func transferRejectedBySecondContract(
	t *testing.T,
	nonce, round uint64,
	nodes []*integrationTests.TestProcessorNode,
	tokenIssuer *integrationTests.TestProcessorNode,
	idxProposers []int,
	initialSupply int64,
	tokenIdentifier string,
	firstScAddress []byte,
	secondScAddress []byte,
	functionToCall string,
	nrRoundToPropagate int,
) (uint64, uint64) {
	// call first sc with esdt, and first sc automatically calls second sc which returns error
	valueToSendToSc := int64(1000)
	txData := txDataBuilder.NewBuilder()
	txData.Clear().TransferESDT(tokenIdentifier, valueToSendToSc)
	txData.Str(functionToCall)
	integrationTests.CreateAndSendTransaction(
		tokenIssuer,
		nodes,
		big.NewInt(0),
		firstScAddress,
		txData.ToString(),
		integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundToPropagate, nonce, round, idxProposers)
	time.Sleep(time.Second)

	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply-valueToSendToSc)

	esdtData := esdtCommon.GetESDTTokenData(t, firstScAddress, nodes, tokenIdentifier)
	require.Equal(t, &esdt.ESDigitalToken{Value: big.NewInt(valueToSendToSc)}, esdtData)

	esdtData = esdtCommon.GetESDTTokenData(t, secondScAddress, nodes, tokenIdentifier)
	require.Equal(t, &esdt.ESDigitalToken{}, esdtData)

	return nonce, round
}

func TestESDTMultiTransferFromSC(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 1
	nodesPerShard := 1
	numMetachainNodes := 1

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := big.NewInt(10000000000)
	integrationTests.MintAllNodes(nodes, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// send token issue

	initialSupply := int64(10000000000)
	ticker := "TCK"
	esdtCommon.IssueTestToken(nodes, initialSupply, ticker)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 12
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(integrationTests.GetTokenIdentifier(nodes, []byte(ticker)))
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply)

	txData := txDataBuilder.NewBuilder()

	// send tx to other nodes
	valueToSend := int64(100)
	for _, node := range nodes[1:] {
		txData.Clear().TransferESDT(tokenIdentifier, valueToSend)
		integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), node.OwnAccount.Address, txData.ToString(), integrationTests.AdditionalGasLimit)
	}

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	numNodesWithoutIssuer := int64(len(nodes) - 1)
	issuerBalance := initialSupply - valueToSend*numNodesWithoutIssuer
	esdtCommon.CheckAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, issuerBalance)
	for i := 1; i < len(nodes); i++ {
		esdtCommon.CheckAddressHasESDTTokens(t, nodes[i].OwnAccount.Address, nodes, tokenIdentifier, valueToSend)
	}

	// deploy the smart contract
	scCode := arwen.GetSCCode("../testdata/transfer-esdt-hook.wasm")
	scAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(scCode),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err := nodes[0].AccntState.GetExistingAccount(scAddress)
	require.Nil(t, err)

	// call sc with esdt
	valueToSendToSc := int64(100)
	for _, node := range nodes {
		txData.Clear().TransferESDT(tokenIdentifier, valueToSendToSc).Str("acceptEsdt")
		integrationTests.CreateAndSendTransaction(node, nodes, big.NewInt(0), scAddress, txData.ToString(), integrationTests.AdditionalGasLimit)
	}

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	time.Sleep(time.Second)

	// transferOne
	for _, node := range nodes {
		txData.Clear().Func("transferEsdtOnce").Bytes(node.OwnAccount.Address).Str(tokenIdentifier).Int(1)
		integrationTests.CreateAndSendTransaction(node, nodes, big.NewInt(0), scAddress, txData.ToString(), integrationTests.AdditionalGasLimit)
	}

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	time.Sleep(time.Second)

	// transferMultiple
	for _, node := range nodes {
		txData.Clear().Func("transferEsdtMultiple").Bytes(node.OwnAccount.Address).Str(tokenIdentifier).Int(1).Int(3)
		integrationTests.CreateAndSendTransaction(node, nodes, big.NewInt(0), scAddress, txData.ToString(), integrationTests.AdditionalGasLimit)
	}

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	time.Sleep(time.Second)
}
