package esdt

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/vmcommon"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/esdt"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	testVm "github.com/ElrondNetwork/elrond-go/integrationTests/vm"
	"github.com/ElrondNetwork/elrond-go/integrationTests/vm/arwen"
	"github.com/ElrondNetwork/elrond-go/process"
	vmFactory "github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/builtInFunctions"
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

	advertiser := integrationTests.CreateMessengerWithKadDht("")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
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

	///////////------- send token issue

	initialSupply := int64(10000000000)
	issueTestToken(nodes, initialSupply)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 10
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(getTokenIdentifier(nodes))

	checkAddressHasESDTTokensInt64(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply)

	/////////------ send tx to other nodes
	valueToSend := int64(100)
	for _, node := range nodes[1:] {
		txData := testVm.NewTxDataBuilder().TransferESDT(tokenIdentifier, valueToSend)
		integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), node.OwnAccount.Address, txData.String(), integrationTests.AdditionalGasLimit)
	}

	mintValue := int64(10000)
	txData := testVm.NewTxDataBuilder()
	txData = txData.Func("mint").Str(tokenIdentifier).Int64(mintValue)
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vm.ESDTSCAddress, txData.String(), core.MinMetaTxExtraGasCost)

	txData.New().Func("freeze").Str(tokenIdentifier).Bytes(nodes[2].OwnAccount.Address)
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vm.ESDTSCAddress, txData.String(), core.MinMetaTxExtraGasCost)

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	finalSupply := initialSupply + mintValue
	for _, node := range nodes[1:] {
		checkAddressHasESDTTokensInt64(t, node.OwnAccount.Address, nodes, tokenIdentifier, valueToSend)
		finalSupply = finalSupply - valueToSend
	}

	checkAddressHasESDTTokensInt64(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, finalSupply)

	txData.New().Func("ESDTBurn").Str(tokenIdentifier).Int64(mintValue)
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vm.ESDTSCAddress, txData.String(), core.MinMetaTxExtraGasCost)

	txData.New().Func("freeze").Str(tokenIdentifier).Bytes(nodes[1].OwnAccount.Address)
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vm.ESDTSCAddress, txData.String(), core.MinMetaTxExtraGasCost)

	txData.New().Func("wipe").Str(tokenIdentifier).Bytes(nodes[2].OwnAccount.Address)
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vm.ESDTSCAddress, txData.String(), core.MinMetaTxExtraGasCost)

	txData.New().Func("pause").Str(tokenIdentifier)
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vm.ESDTSCAddress, txData.String(), core.MinMetaTxExtraGasCost)

	time.Sleep(time.Second)

	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	esdtFrozenData := getESDTTokenData(t, nodes[1].OwnAccount.Address, nodes, tokenIdentifier)
	esdtUserMetaData := builtInFunctions.ESDTUserMetadataFromBytes(esdtFrozenData.Properties)
	require.True(t, esdtUserMetaData.Frozen)

	wipedAcc := getUserAccountWithAddress(t, nodes[2].OwnAccount.Address, nodes)
	tokenKey := []byte(core.ElrondProtectedKeyPrefix + "esdt" + tokenIdentifier)
	retrievedData, _ := wipedAcc.DataTrieTracker().RetrieveValue(tokenKey)
	require.Equal(t, 0, len(retrievedData))

	systemSCAcc := getUserAccountWithAddress(t, core.SystemAccountAddress, nodes)
	retrievedData, _ = systemSCAcc.DataTrieTracker().RetrieveValue(tokenKey)
	esdtGlobalMetaData := builtInFunctions.ESDTGlobalMetadataFromBytes(retrievedData)
	require.True(t, esdtGlobalMetaData.Paused)

	finalSupply = finalSupply - mintValue
	checkAddressHasESDTTokensInt64(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, finalSupply)

	esdtSCAcc := getUserAccountWithAddress(t, vm.ESDTSCAddress, nodes)
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

	advertiser := integrationTests.CreateMessengerWithKadDht("")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
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

	///////////------- send token issue
	ticker := "ALC"
	issuePrice := big.NewInt(1000)
	initialSupply := big.NewInt(10000000000)
	tokenIssuer := nodes[0]
	hexEncodedTrue := hex.EncodeToString([]byte("true"))
	hexEncodedFalse := hex.EncodeToString([]byte("false"))
	txData := "issue" +
		"@" + hex.EncodeToString([]byte("aliceToken")) +
		"@" + hex.EncodeToString([]byte(ticker)) +
		"@" + hex.EncodeToString(initialSupply.Bytes()) +
		"@" + hex.EncodeToString([]byte{6})
	properties := "@" + hex.EncodeToString([]byte("canFreeze")) + "@" + hexEncodedTrue +
		"@" + hex.EncodeToString([]byte("canWipe")) + "@" + hexEncodedTrue +
		"@" + hex.EncodeToString([]byte("canPause")) + "@" + hexEncodedTrue +
		"@" + hex.EncodeToString([]byte("canMint")) + "@" + hexEncodedTrue +
		"@" + hex.EncodeToString([]byte("canBurn")) + "@" + hexEncodedFalse
	txData += properties
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, issuePrice, vm.ESDTSCAddress, txData, core.MinMetaTxExtraGasCost)

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 10
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(getTokenIdentifier(nodes))

	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply)

	/////////------ send tx to other nodes
	valueToSend := big.NewInt(100)
	for _, node := range nodes[1:] {
		txData = core.BuiltInFunctionESDTTransfer + "@" + hex.EncodeToString([]byte(tokenIdentifier)) + "@" + hex.EncodeToString(valueToSend.Bytes())
		integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), node.OwnAccount.Address, txData, integrationTests.AdditionalGasLimit)
	}

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	finalSupply := big.NewInt(0).Set(initialSupply)
	for _, node := range nodes[1:] {
		checkAddressHasESDTTokens(t, node.OwnAccount.Address, nodes, tokenIdentifier, valueToSend)
		finalSupply.Sub(finalSupply, valueToSend)
	}

	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, finalSupply)

	burnValue := big.NewInt(77)
	txData = core.BuiltInFunctionESDTBurn + "@" + hex.EncodeToString([]byte(tokenIdentifier)) + "@" + hex.EncodeToString(burnValue.Bytes())
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vm.ESDTSCAddress, txData, core.MinMetaTxExtraGasCost)

	time.Sleep(time.Second)

	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	esdtSCAcc := getUserAccountWithAddress(t, vm.ESDTSCAddress, nodes)
	retrievedData, _ := esdtSCAcc.DataTrieTracker().RetrieveValue([]byte(tokenIdentifier))
	tokenInSystemSC := &systemSmartContracts.ESDTData{}
	_ = integrationTests.TestMarshalizer.Unmarshal(tokenInSystemSC, retrievedData)
	require.True(t, tokenInSystemSC.MintedValue.Cmp(initialSupply) == 0)
	require.True(t, tokenInSystemSC.BurntValue.Cmp(big.NewInt(0)) == 0)

	// if everything is ok, the caller should have received the amount of burnt tokens back because canBurn = false
	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, finalSupply)
}

func TestESDTIssueFromASmartContractSimulated(t *testing.T) {
	advertiser := integrationTests.CreateMessengerWithKadDht("")
	_ = advertiser.Bootstrap()
	metaNode := integrationTests.NewTestProcessorNode(1, core.MetachainShardId, 0, integrationTests.GetConnectableAddress(advertiser))
	defer func() {
		_ = advertiser.Close()
		_ = metaNode.Messenger.Close()
	}()

	ticker := "RBT"
	issuePrice := big.NewInt(1000)
	initialSupply := big.NewInt(10000000000)
	numDecimals := []byte{6}
	hexEncodedTrue := hex.EncodeToString([]byte("true"))
	txData := "issue" +
		"@" + hex.EncodeToString([]byte("robertWhyNot")) +
		"@" + hex.EncodeToString([]byte(ticker)) +
		"@" + hex.EncodeToString(initialSupply.Bytes()) +
		"@" + hex.EncodeToString(numDecimals)
	properties := "@" + hex.EncodeToString([]byte("canFreeze")) + "@" + hexEncodedTrue +
		"@" + hex.EncodeToString([]byte("canWipe")) + "@" + hexEncodedTrue +
		"@" + hex.EncodeToString([]byte("canPause")) + "@" + hexEncodedTrue +
		"@" + hex.EncodeToString([]byte("canMint")) + "@" + hexEncodedTrue +
		"@" + hex.EncodeToString([]byte("canBurn")) + "@" + hexEncodedTrue +
		"@" + hex.EncodeToString(big.NewInt(0).SetUint64(1000).Bytes())
	txData += properties

	scr := &smartContractResult.SmartContractResult{
		Nonce:          0,
		Value:          issuePrice,
		RcvAddr:        vm.ESDTSCAddress,
		SndAddr:        metaNode.OwnAccount.Address,
		Data:           []byte(txData),
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

	advertiser := integrationTests.CreateMessengerWithKadDht("")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
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

	//----------------- send token issue
	initialSupply := int64(10000000000)
	issueTestToken(nodes, initialSupply)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 10
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(getTokenIdentifier(nodes))
	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(initialSupply))

	//------------- deploy the smart contract

	vaultScCode := arwen.GetSCCode("./testdata/vault.wasm")
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

	//// feed funds to the vault
	valueToSendToSc := int64(1000)
	txData := core.BuiltInFunctionESDTTransfer + "@" +
		hex.EncodeToString([]byte(tokenIdentifier)) + "@" +
		hex.EncodeToString(big.NewInt(valueToSendToSc).Bytes()) + "@" +
		hex.EncodeToString([]byte("accept_funds"))
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vaultScAddress, txData, integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(initialSupply-valueToSendToSc))
	checkAddressHasESDTTokens(t, vaultScAddress, nodes, tokenIdentifier, big.NewInt(valueToSendToSc))

	//// take them back, with a message
	valueToRequest := valueToSendToSc / 4
	txData = "retrieve_funds@" +
		hex.EncodeToString([]byte(tokenIdentifier)) + "@" +
		hex.EncodeToString(big.NewInt(valueToRequest).Bytes()) + "@" +
		hex.EncodeToString([]byte("ESDT transfer message"))
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), vaultScAddress, txData, integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(initialSupply-valueToSendToSc+valueToRequest))
	checkAddressHasESDTTokens(t, vaultScAddress, nodes, tokenIdentifier, big.NewInt(valueToSendToSc-valueToRequest))
}

func TestESDTcallsSC(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 2
	nodesPerShard := 2
	numMetachainNodes := 2

	advertiser := integrationTests.CreateMessengerWithKadDht("")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
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

	///////////------- send token issue

	initialSupply := int64(10000000000)
	issueTestToken(nodes, initialSupply)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 10
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(getTokenIdentifier(nodes))
	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(initialSupply))

	/////////------ send tx to other nodes
	valueToSend := int64(100)
	for _, node := range nodes[1:] {
		txData := core.BuiltInFunctionESDTTransfer + "@" + hex.EncodeToString([]byte(tokenIdentifier)) + "@" + hex.EncodeToString(big.NewInt(valueToSend).Bytes())
		integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), node.OwnAccount.Address, txData, integrationTests.AdditionalGasLimit)
	}

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	numNodesWithoutIssuer := int64(len(nodes) - 1)
	issuerBalance := initialSupply - valueToSend*numNodesWithoutIssuer
	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(issuerBalance))
	for i := 1; i < len(nodes); i++ {
		checkAddressHasESDTTokens(t, nodes[i].OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(valueToSend))
	}

	// deploy the smart contract
	scCode := arwen.GetSCCode("./testdata/crowdfunding-esdt.wasm")
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
		txData := core.BuiltInFunctionESDTTransfer + "@" +
			hex.EncodeToString([]byte(tokenIdentifier)) + "@" +
			hex.EncodeToString(big.NewInt(valueToSendToSc).Bytes()) + "@" +
			hex.EncodeToString([]byte("fund"))
		integrationTests.CreateAndSendTransaction(node, nodes, big.NewInt(0), scAddress, txData, integrationTests.AdditionalGasLimit)
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
	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(issuerBalance))
	for i := 1; i < len(nodes); i++ {
		checkAddressHasESDTTokens(t, nodes[i].OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(nodesBalance))
	}
}

func TestScCallsScWithEsdtIntraShard(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	logger.SetLogLevel("*:NONE")

	numOfShards := 1
	nodesPerShard := 1
	numMetachainNodes := 1

	advertiser := integrationTests.CreateMessengerWithKadDht("")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
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

	//----------------- send token issue
	initialSupply := int64(10000000000)
	issueTestToken(nodes, initialSupply)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 10
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(getTokenIdentifier(nodes))
	checkAddressHasESDTTokensInt64(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply)

	//------------- deploy the smart contracts

	vaultCode := arwen.GetSCCode("./testdata/vault.wasm")
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

	forwarderCode := arwen.GetSCCode("./testdata/forwarder-raw.wasm")
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

	txData := testVm.NewTxDataBuilder()

	//// call first sc with esdt, and first sc automatically calls second sc
	valueToSendToSc := int64(1000)
	txData.TransferESDT(tokenIdentifier, valueToSendToSc)
	txData.Str("forward_async_call_half_payment").Bytes(vault).Str("accept_funds")

	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), forwarder, txData.String(), integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	checkAddressHasESDTTokensInt64(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply-valueToSendToSc)
	checkAddressHasESDTTokensInt64(t, forwarder, nodes, tokenIdentifier, valueToSendToSc/2)
	checkAddressHasESDTTokensInt64(t, vault, nodes, tokenIdentifier, valueToSendToSc/2)

	checkNumCallBacks(t, forwarder, nodes, 1)
	checkSavedCallBackData(t, forwarder, nodes, 1, "EGLD", big.NewInt(0), vmcommon.Ok, [][]byte{})

	// //// call first sc to ask the second one to send it back some esdt
	valueToRequest := valueToSendToSc / 4
	txData.New().Func("forward_async_call").Bytes(vault).Str("retrieve_funds").Str(tokenIdentifier).Int64(valueToRequest)

	fmt.Println(txData.String())
	fmt.Println("Before transfer ---------------------------")
	tokenIssuerBalance := getESDTTokenData(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier).Value
	forwarderBalance := getESDTTokenData(t, forwarder, nodes, tokenIdentifier).Value
	vaultBalance := getESDTTokenData(t, vault, nodes, tokenIdentifier).Value

	fmt.Println("tokenIssuerBalance", tokenIssuerBalance)
	fmt.Println("forwarderBalance", forwarderBalance)
	fmt.Println("vault", vaultBalance)
	fmt.Println("-------------------------------------------")

	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), forwarder, txData.String(), integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	time.Sleep(time.Second)

	checkAddressHasESDTTokensInt64(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply-valueToSendToSc)
	checkAddressHasESDTTokensInt64(t, forwarder, nodes, tokenIdentifier, valueToSendToSc*3/4)
	checkAddressHasESDTTokensInt64(t, vault, nodes, tokenIdentifier, valueToSendToSc/4)

	checkNumCallBacks(t, forwarder, nodes, 2)
	checkSavedCallBackData(t, forwarder, nodes, 2, tokenIdentifier, big.NewInt(valueToRequest), vmcommon.Ok, [][]byte{})

	fmt.Println("After transfer ---------------------------")
	tokenIssuerBalance = getESDTTokenData(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier).Value
	forwarderBalance = getESDTTokenData(t, forwarder, nodes, tokenIdentifier).Value
	vaultBalance = getESDTTokenData(t, vault, nodes, tokenIdentifier).Value

	fmt.Println("tokenIssuerBalance", tokenIssuerBalance)
	fmt.Println("forwarderBalance", forwarderBalance)
	fmt.Println("vault", vaultBalance)
	fmt.Println("-------------------------------------------")

	//// call first sc to ask the second one to execute a method
	valueToTransferWithExecSc := valueToSendToSc / 4
	txData.New().TransferESDT(tokenIdentifier, valueToTransferWithExecSc)
	txData.Str("forward_async_call").Bytes(vault).Str("accept_funds")

	fmt.Println(txData.String())
	fmt.Println("Before transfer ---------------------------")
	tokenIssuerBalance = getESDTTokenData(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier).Value
	forwarderBalance = getESDTTokenData(t, forwarder, nodes, tokenIdentifier).Value
	vaultBalance = getESDTTokenData(t, vault, nodes, tokenIdentifier).Value

	fmt.Println("tokenIssuerBalance", tokenIssuerBalance)
	fmt.Println("forwarderBalance", forwarderBalance)
	fmt.Println("vault", vaultBalance)
	fmt.Println("-------------------------------------------")

	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), forwarder, txData.String(), integrationTests.AdditionalGasLimit)
	time.Sleep(5 * time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	time.Sleep(5 * time.Second)

	fmt.Println("After transfer ---------------------------")
	tokenIssuerBalance = getESDTTokenData(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier).Value
	forwarderBalance = getESDTTokenData(t, forwarder, nodes, tokenIdentifier).Value
	vaultBalance = getESDTTokenData(t, vault, nodes, tokenIdentifier).Value

	fmt.Println("tokenIssuerBalance", tokenIssuerBalance)
	fmt.Println("forwarderBalance", forwarderBalance)
	fmt.Println("vault", vaultBalance)
	fmt.Println("-------------------------------------------")

	checkAddressHasESDTTokensInt64(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, initialSupply-valueToSendToSc-250)
	checkAddressHasESDTTokensInt64(t, forwarder, nodes, tokenIdentifier, 750)
	checkAddressHasESDTTokensInt64(t, vault, nodes, tokenIdentifier, 500)
}

func TestCallbackPaymentEgld(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 1
	nodesPerShard := 1
	numMetachainNodes := 1

	advertiser := integrationTests.CreateMessengerWithKadDht("")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
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

	//----------------- send token issue
	initialSupply := int64(10000000000)
	issueTestToken(nodes, initialSupply)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 10
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(getTokenIdentifier(nodes))
	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(initialSupply))

	//------------- deploy the smart contracts

	secondScCode := arwen.GetSCCode("./testdata/vault.wasm")
	secondScAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(secondScCode),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err := nodes[0].AccntState.GetExistingAccount(secondScAddress)
	require.Nil(t, err)

	firstScCode := arwen.GetSCCode("./testdata/forwarder-raw.wasm")
	firstScAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(firstScCode),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err = nodes[0].AccntState.GetExistingAccount(firstScAddress)
	require.Nil(t, err)

	//// call first sc with esdt, and first sc automatically calls second sc
	valueToSendToSc := int64(1000)
	txData := "forward_async_call_half_payment@" +
		hex.EncodeToString(secondScAddress) + "@" +
		hex.EncodeToString([]byte("accept_funds"))
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(valueToSendToSc), firstScAddress, txData, integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 1, nonce, round, idxProposers)
	time.Sleep(time.Second)

	checkNumCallBacks(t, firstScAddress, nodes, 1)
	checkSavedCallBackData(t, firstScAddress, nodes, 1, "EGLD", big.NewInt(0), vmcommon.Ok, [][]byte{})

	//// call first sc to ask the second one to send it back some esdt
	valueToRequest := valueToSendToSc / 4
	txData = "forward_async_call@" +
		hex.EncodeToString(secondScAddress) + "@" +
		hex.EncodeToString([]byte("retrieve_funds")) + "@" +
		hex.EncodeToString([]byte("EGLD")) + "@" +
		hex.EncodeToString(big.NewInt(valueToRequest).Bytes())
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), firstScAddress, txData, integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 1, nonce, round, idxProposers)
	time.Sleep(time.Second)

	checkNumCallBacks(t, firstScAddress, nodes, 2)
	checkSavedCallBackData(t, firstScAddress, nodes, 2, "EGLD", big.NewInt(valueToRequest), vmcommon.Ok, [][]byte{})
}

func TestScCallsScWithEsdtCrossShard(t *testing.T) {
	t.Skip("test is not ready yet")

	numOfShards := 2
	nodesPerShard := 2
	numMetachainNodes := 2

	advertiser := integrationTests.CreateMessengerWithKadDht("")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
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

	//----------------- send token issue

	initialSupply := int64(10000000000)
	issueTestToken(nodes, initialSupply)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 10
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(getTokenIdentifier(nodes))
	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(initialSupply))

	//------------- deploy the smart contracts

	secondScCode := arwen.GetSCCode("./testdata/vault.wasm")
	secondScAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(secondScCode),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err := nodes[0].AccntState.GetExistingAccount(secondScAddress)
	require.Nil(t, err)

	firstScCode := arwen.GetSCCode("./testdata/forwarder-raw.wasm")
	firstScAddress, _ := nodes[2].BlockchainHook.NewAddress(nodes[2].OwnAccount.Address, nodes[2].OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)
	integrationTests.CreateAndSendTransaction(
		nodes[2],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(firstScCode),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err = nodes[2].AccntState.GetExistingAccount(firstScAddress)
	require.Nil(t, err)

	//// call first sc with esdt, and first sc automatically calls second sc
	valueToSendToSc := int64(1000)
	txData := core.BuiltInFunctionESDTTransfer + "@" +
		hex.EncodeToString([]byte(tokenIdentifier)) + "@" +
		hex.EncodeToString(big.NewInt(valueToSendToSc).Bytes()) + "@" +
		hex.EncodeToString([]byte("forward_async_call_half_payment")) + "@" +
		hex.EncodeToString(secondScAddress) + "@" +
		hex.EncodeToString([]byte("accept_funds"))
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), firstScAddress, txData, integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(initialSupply-valueToSendToSc))
	checkAddressHasESDTTokens(t, firstScAddress, nodes, tokenIdentifier, big.NewInt(valueToSendToSc/2))
	checkAddressHasESDTTokens(t, secondScAddress, nodes, tokenIdentifier, big.NewInt(valueToSendToSc/2))

	checkNumCallBacks(t, firstScAddress, nodes, 1)
	checkSavedCallBackData(t, firstScAddress, nodes, 1, "EGLD", big.NewInt(0), vmcommon.Ok, [][]byte{})

	//// call first sc to ask the second one to send it back some esdt
	valueToRequest := valueToSendToSc / 4
	txData = "forward_async_call@" +
		hex.EncodeToString(secondScAddress) + "@" +
		hex.EncodeToString([]byte("retrieve_funds")) + "@" +
		hex.EncodeToString([]byte(tokenIdentifier)) + "@" +
		hex.EncodeToString(big.NewInt(valueToRequest).Bytes())
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), firstScAddress, txData, integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	checkAddressHasESDTTokens(t, firstScAddress, nodes, tokenIdentifier, big.NewInt(valueToSendToSc*3/4))
	checkAddressHasESDTTokens(t, secondScAddress, nodes, tokenIdentifier, big.NewInt(valueToSendToSc/4))

	checkNumCallBacks(t, firstScAddress, nodes, 2)
	checkSavedCallBackData(t, firstScAddress, nodes, 1, tokenIdentifier, big.NewInt(valueToSendToSc), vmcommon.Ok, [][]byte{})
}

func TestScCallsScWithEsdtIntraShard_SecondScRefusesPayment(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 1
	nodesPerShard := 1
	numMetachainNodes := 1

	advertiser := integrationTests.CreateMessengerWithKadDht("")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
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

	//----------------- send token issue
	initialSupply := int64(10000000000)
	issueTestToken(nodes, initialSupply)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 10
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(getTokenIdentifier(nodes))
	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(initialSupply))

	//------------- deploy the smart contracts

	secondScCode := arwen.GetSCCode("./testdata/second-contract.wasm")
	secondScAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(secondScCode)+"@"+
			hex.EncodeToString([]byte(tokenIdentifier)),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 2, nonce, round, idxProposers)
	_, err := nodes[0].AccntState.GetExistingAccount(secondScAddress)
	require.Nil(t, err)

	firstScCode := arwen.GetSCCode("./testdata/first-contract.wasm")
	firstScAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(firstScCode)+"@"+
			hex.EncodeToString([]byte(tokenIdentifier))+"@"+
			hex.EncodeToString(secondScAddress),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 2, nonce, round, idxProposers)
	_, err = nodes[0].AccntState.GetExistingAccount(firstScAddress)
	require.Nil(t, err)

	//// call first sc with esdt, and first sc automatically calls second sc which returns error
	valueToSendToSc := int64(1000)
	txData := core.BuiltInFunctionESDTTransfer + "@" +
		hex.EncodeToString([]byte(tokenIdentifier)) + "@" +
		hex.EncodeToString(big.NewInt(valueToSendToSc).Bytes()) + "@" +
		hex.EncodeToString([]byte("transfer_to_second_contract_rejected"))
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), firstScAddress, txData, integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(initialSupply-valueToSendToSc))

	esdtData := getESDTTokenData(t, firstScAddress, nodes, tokenIdentifier)
	require.Equal(t, &esdt.ESDigitalToken{Value: big.NewInt(valueToSendToSc)}, esdtData)

	esdtData = getESDTTokenData(t, secondScAddress, nodes, tokenIdentifier)
	require.Equal(t, &esdt.ESDigitalToken{Value: big.NewInt(0)}, esdtData)
}

func TestScCallsScWithEsdtCrossShard_SecondScRefusesPayment(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 2
	nodesPerShard := 2
	numMetachainNodes := 2

	advertiser := integrationTests.CreateMessengerWithKadDht("")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
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

	//----------------- send token issue

	initialSupply := int64(10000000000)
	issueTestToken(nodes, initialSupply)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 10
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(getTokenIdentifier(nodes))
	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(initialSupply))

	//------------- deploy the smart contracts

	secondScCode := arwen.GetSCCode("./testdata/second-contract.wasm")
	secondScAddress, _ := tokenIssuer.BlockchainHook.NewAddress(tokenIssuer.OwnAccount.Address, tokenIssuer.OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(secondScCode)+"@"+
			hex.EncodeToString([]byte(tokenIdentifier)),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err := nodes[0].AccntState.GetExistingAccount(secondScAddress)
	require.Nil(t, err)

	firstScCode := arwen.GetSCCode("./testdata/first-contract.wasm")
	firstScAddress, _ := nodes[2].BlockchainHook.NewAddress(nodes[2].OwnAccount.Address, nodes[2].OwnAccount.Nonce, vmFactory.ArwenVirtualMachine)
	integrationTests.CreateAndSendTransaction(
		nodes[2],
		nodes,
		big.NewInt(0),
		testVm.CreateEmptyAddress(),
		arwen.CreateDeployTxData(firstScCode)+"@"+
			hex.EncodeToString([]byte(tokenIdentifier))+"@"+
			hex.EncodeToString(secondScAddress),
		integrationTests.AdditionalGasLimit,
	)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	_, err = nodes[2].AccntState.GetExistingAccount(firstScAddress)
	require.Nil(t, err)

	//// call first sc with esdt, and first sc automatically calls second sc which returns error
	valueToSendToSc := int64(1000)
	txData := core.BuiltInFunctionESDTTransfer + "@" +
		hex.EncodeToString([]byte(tokenIdentifier)) + "@" +
		hex.EncodeToString(big.NewInt(valueToSendToSc).Bytes()) + "@" +
		hex.EncodeToString([]byte("transfer_to_second_contract_rejected"))
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), firstScAddress, txData, integrationTests.AdditionalGasLimit)

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard*2, nonce, round, idxProposers)
	time.Sleep(time.Second)

	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(initialSupply-valueToSendToSc))

	esdtData := getESDTTokenData(t, firstScAddress, nodes, tokenIdentifier)
	require.Equal(t, &esdt.ESDigitalToken{Value: big.NewInt(valueToSendToSc)}, esdtData)

	esdtData = getESDTTokenData(t, secondScAddress, nodes, tokenIdentifier)
	require.Equal(t, &esdt.ESDigitalToken{}, esdtData)
}

func TestESDTMultiTransferFromSC(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 1
	nodesPerShard := 1
	numMetachainNodes := 1

	advertiser := integrationTests.CreateMessengerWithKadDht("")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
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

	///////////------- send token issue

	initialSupply := int64(10000000000)
	issueTestToken(nodes, initialSupply)
	tokenIssuer := nodes[0]

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 10
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	tokenIdentifier := string(getTokenIdentifier(nodes))
	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(initialSupply))

	/////////------ send tx to other nodes
	valueToSend := int64(100)
	for _, node := range nodes[1:] {
		txData := core.BuiltInFunctionESDTTransfer + "@" + hex.EncodeToString([]byte(tokenIdentifier)) + "@" + hex.EncodeToString(big.NewInt(valueToSend).Bytes())
		integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, big.NewInt(0), node.OwnAccount.Address, txData, integrationTests.AdditionalGasLimit)
	}

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	numNodesWithoutIssuer := int64(len(nodes) - 1)
	issuerBalance := initialSupply - valueToSend*numNodesWithoutIssuer
	checkAddressHasESDTTokens(t, tokenIssuer.OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(issuerBalance))
	for i := 1; i < len(nodes); i++ {
		checkAddressHasESDTTokens(t, nodes[i].OwnAccount.Address, nodes, tokenIdentifier, big.NewInt(valueToSend))
	}

	// deploy the smart contract
	scCode := arwen.GetSCCode("./testdata/transfer-esdt-hook.wasm")
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
		txData := core.BuiltInFunctionESDTTransfer + "@" +
			hex.EncodeToString([]byte(tokenIdentifier)) + "@" +
			hex.EncodeToString(big.NewInt(valueToSendToSc).Bytes()) + "@" +
			hex.EncodeToString([]byte("acceptEsdt"))
		integrationTests.CreateAndSendTransaction(node, nodes, big.NewInt(0), scAddress, txData, integrationTests.AdditionalGasLimit)
	}

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	time.Sleep(time.Second)

	// transferOne
	for _, node := range nodes {
		txData := "transferEsdtOnce" + "@" +
			hex.EncodeToString(node.OwnAccount.Address) + "@" +
			hex.EncodeToString([]byte(tokenIdentifier)) + "@" +
			hex.EncodeToString(big.NewInt(1).Bytes())
		integrationTests.CreateAndSendTransaction(node, nodes, big.NewInt(0), scAddress, txData, integrationTests.AdditionalGasLimit)
	}

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	time.Sleep(time.Second)

	// transferMultiple
	for _, node := range nodes {
		txData := "transferEsdtMultiple" + "@" +
			hex.EncodeToString(node.OwnAccount.Address) + "@" +
			hex.EncodeToString([]byte(tokenIdentifier)) + "@" +
			hex.EncodeToString(big.NewInt(1).Bytes()) + "@" +
			hex.EncodeToString(big.NewInt(3).Bytes())
		integrationTests.CreateAndSendTransaction(node, nodes, big.NewInt(0), scAddress, txData, integrationTests.AdditionalGasLimit)
	}

	time.Sleep(time.Second)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 4, nonce, round, idxProposers)
	time.Sleep(time.Second)
}

func issueTestToken(nodes []*integrationTests.TestProcessorNode, initialSupply int64) {
	ticker := "TKN"
	tokenName := "token"
	issuePrice := big.NewInt(1000)

	tokenIssuer := nodes[0]
	hexEncodedTrue := hex.EncodeToString([]byte("true"))

	txData := "issue" +
		"@" + hex.EncodeToString([]byte(tokenName)) +
		"@" + hex.EncodeToString([]byte(ticker)) +
		"@" + hex.EncodeToString(big.NewInt(initialSupply).Bytes()) +
		"@" + hex.EncodeToString([]byte{6})
	properties := "@" + hex.EncodeToString([]byte("canFreeze")) + "@" + hexEncodedTrue +
		"@" + hex.EncodeToString([]byte("canWipe")) + "@" + hexEncodedTrue +
		"@" + hex.EncodeToString([]byte("canPause")) + "@" + hexEncodedTrue +
		"@" + hex.EncodeToString([]byte("canMint")) + "@" + hexEncodedTrue +
		"@" + hex.EncodeToString([]byte("canBurn")) + "@" + hexEncodedTrue
	txData += properties
	integrationTests.CreateAndSendTransaction(tokenIssuer, nodes, issuePrice, vm.ESDTSCAddress, txData, core.MinMetaTxExtraGasCost)
}

func getTokenIdentifier(nodes []*integrationTests.TestProcessorNode) []byte {
	for _, node := range nodes {
		if node.ShardCoordinator.SelfId() != core.MetachainShardId {
			continue
		}

		scQuery := &process.SCQuery{
			ScAddress:  vm.ESDTSCAddress,
			FuncName:   "getAllESDTTokens",
			CallerAddr: vm.ESDTSCAddress,
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{},
		}
		vmOutput, err := node.SCQueryService.ExecuteQuery(scQuery)
		if err != nil || vmOutput == nil || vmOutput.ReturnCode != vmcommon.Ok {
			return nil
		}
		if len(vmOutput.ReturnData) == 0 {
			return nil
		}

		return vmOutput.ReturnData[0]
	}

	return nil
}

func getESDTTokenData(
	t *testing.T,
	address []byte,
	nodes []*integrationTests.TestProcessorNode,
	tokenName string,
) *esdt.ESDigitalToken {
	userAcc := getUserAccountWithAddress(t, address, nodes)
	require.False(t, check.IfNil(userAcc))

	tokenKey := []byte(core.ElrondProtectedKeyPrefix + "esdt" + tokenName)
	esdtData, err := getESDTDataFromKey(userAcc, tokenKey)
	require.Nil(t, err)

	return esdtData
}

func checkAddressHasESDTTokensInt64(
	t *testing.T,
	address []byte,
	nodes []*integrationTests.TestProcessorNode,
	tokenName string,
	value int64,
) {
	esdtData := getESDTTokenData(t, address, nodes, tokenName)
	bigValue := big.NewInt(value)
	if esdtData.Value.Cmp(bigValue) != 0 {
		require.Fail(t, fmt.Sprintf("esdt balance difference. expected %s, but got %s", bigValue.String(), esdtData.Value.String()))
	}
}

func checkAddressHasESDTTokens(
	t *testing.T,
	address []byte,
	nodes []*integrationTests.TestProcessorNode,
	tokenName string,
	value *big.Int,
) {
	esdtData := getESDTTokenData(t, address, nodes, tokenName)
	if esdtData.Value.Cmp(value) != 0 {
		require.Fail(t, fmt.Sprintf("esdt balance difference. expected %s, but got %s", value.String(), esdtData.Value.String()))
	}
}

func checkNumCallBacks(
	t *testing.T,
	address []byte,
	nodes []*integrationTests.TestProcessorNode,
	expectedNumCallbacks int) {

	contractID := nodes[0].ShardCoordinator.ComputeId(address)
	for _, node := range nodes {
		if node.ShardCoordinator.SelfId() != contractID {
			continue
		}

		scQuery := &process.SCQuery{
			ScAddress:  address,
			FuncName:   "callback_data",
			CallerAddr: address,
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{},
		}
		vmOutput, err := node.SCQueryService.ExecuteQuery(scQuery)
		require.Nil(t, err)
		require.NotNil(t, vmOutput)
		require.Equal(t, vmOutput.ReturnCode, vmcommon.Ok)
		require.Equal(t, expectedNumCallbacks, len(vmOutput.ReturnData))
	}
}

func checkSavedCallBackData(
	t *testing.T,
	address []byte,
	nodes []*integrationTests.TestProcessorNode,
	callbackIndex int,
	expectedTokenId string,
	expectedPayment *big.Int,
	expectedResultCode vmcommon.ReturnCode,
	expectedArguments [][]byte) {

	contractID := nodes[0].ShardCoordinator.ComputeId(address)
	for _, node := range nodes {
		if node.ShardCoordinator.SelfId() != contractID {
			continue
		}

		scQuery := &process.SCQuery{
			ScAddress:  address,
			FuncName:   "callback_data_at_index",
			CallerAddr: address,
			CallValue:  big.NewInt(0),
			Arguments: [][]byte{
				{byte(callbackIndex)},
			},
		}
		vmOutput, err := node.SCQueryService.ExecuteQuery(scQuery)
		require.Nil(t, err)
		require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
		require.True(t, len(vmOutput.ReturnData) >= 3)
		require.Equal(t, []byte(expectedTokenId), vmOutput.ReturnData[0])
		require.Equal(t, expectedPayment.Bytes(), vmOutput.ReturnData[1])
		if expectedResultCode == vmcommon.Ok {
			require.Equal(t, []byte{}, vmOutput.ReturnData[2])
		} else {
			require.Equal(t, []byte{byte(expectedResultCode)}, vmOutput.ReturnData[2])
		}
		require.Equal(t, expectedArguments, vmOutput.ReturnData[3:])
	}
}

func getUserAccountWithAddress(
	t *testing.T,
	address []byte,
	nodes []*integrationTests.TestProcessorNode,
) state.UserAccountHandler {
	for _, node := range nodes {
		accShardId := node.ShardCoordinator.ComputeId(address)

		for _, helperNode := range nodes {
			if helperNode.ShardCoordinator.SelfId() == accShardId {
				acc, err := helperNode.AccntState.LoadAccount(address)
				require.Nil(t, err)
				return acc.(state.UserAccountHandler)
			}
		}
	}

	return nil
}

func getESDTDataFromKey(userAcnt state.UserAccountHandler, key []byte) (*esdt.ESDigitalToken, error) {
	esdtData := &esdt.ESDigitalToken{Value: big.NewInt(0)}
	marshaledData, err := userAcnt.DataTrieTracker().RetrieveValue(key)
	if err != nil {
		return esdtData, nil
	}

	err = integrationTests.TestMarshalizer.Unmarshal(esdtData, marshaledData)
	if err != nil {
		return nil, err
	}

	return esdtData, nil
}
