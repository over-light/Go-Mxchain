package smartContract

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/ElrondNetwork/elrond-go/integrationTests/vm"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	factory2 "github.com/ElrondNetwork/elrond-go/vm/factory"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSCCallingIntraShard(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 1
	nodesPerShard := 2
	numMetachainNodes := 0

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
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

	initialVal := big.NewInt(10000000000000)
	initialVal.Mul(initialVal, initialVal)
	fmt.Printf("Initial minted sum: %s\n", initialVal.String())
	integrationTests.MintAllNodes(nodes, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// mint smart contract holders
	firstSCOwner := []byte("12345678901234567890123456789000")
	secondSCOwner := []byte("99945678901234567890123456789001")

	mintPubKey(firstSCOwner, initialVal, nodes)
	mintPubKey(secondSCOwner, initialVal, nodes)

	// deploy the smart contracts
	firstSCAddress := putDeploySCToDataPool("./testdata/first/first.wasm", firstSCOwner, 0, big.NewInt(50), "", nodes)
	//000000000000000005005d3d53b5d0fcf07d222170978932166ee9f3972d3030
	secondSCAddress := putDeploySCToDataPool("./testdata/second/second.wasm", secondSCOwner, 0, big.NewInt(50), "", nodes)
	//00000000000000000500017cc09151c48b99e2a1522fb70a5118ad4cb26c3031

	// Run two rounds, so the two SmartContracts get deployed.
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 2, nonce, round, idxProposers)

	time.Sleep(time.Second)

	// Create transactions that invoke "doSomething" from the second SC, which
	// will execute an "asyncCall" to a method in the first SC which counts how
	// many times it has been called. There will be as many transactions as there
	// are nodes.
	for _, node := range nodes {
		txData := "doSomething"
		integrationTests.CreateAndSendTransaction(node, big.NewInt(50), secondSCAddress, txData)
	}

	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 3, nonce, round, idxProposers)

	// verify how many times was the first SC called
	for index, node := range nodes {
		numCalled := vm.GetIntValueFromSC(nil, node.AccntState, firstSCAddress, "numCalled", nil)
		assert.NotNil(t, numCalled)
		if numCalled != nil {
			assert.Equal(t, uint64(len(nodes)), numCalled.Uint64(), fmt.Sprintf("Node %d", index))
		}
	}
}

func TestScDeployAndChangeScOwner(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	sleepDuration := time.Second
	numShards := 2
	nodesPerShard := 2
	numMetachainNodes := 2

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)

	idxProposers := make([]int, numShards+1)
	for i := 0; i < numShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numShards] = numShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := big.NewInt(1000000000)
	integrationTests.MintAllNodes(nodes, initialVal)

	firstSCOwner := nodes[0].OwnAccount.Address.Bytes()

	// deploy the smart contracts
	firstSCAddress := putDeploySCToDataPool("./testdata/counter.wasm", firstSCOwner, 0, big.NewInt(50), "", nodes)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++
	integrationTests.ProposeBlock(nodes, idxProposers, round, nonce)
	integrationTests.SyncBlock(t, nodes, idxProposers, round)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// make smart contract call to shard 1 which will do in shard 0
	for _, node := range nodes {
		txData := "increment"
		for i := 0; i < 10; i++ {
			integrationTests.CreateAndSendTransaction(node, big.NewInt(0), firstSCAddress, txData)
		}
	}

	time.Sleep(sleepDuration)

	numRoundsToPropagateMultiShard := 15
	for i := 0; i < numRoundsToPropagateMultiShard; i++ {
		integrationTests.UpdateRound(nodes, round)
		integrationTests.ProposeBlock(nodes, idxProposers, round, nonce)
		integrationTests.SyncBlock(t, nodes, idxProposers, round)
		round = integrationTests.IncrementAndPrintRound(round)
		nonce++
	}

	address, _ := integrationTests.TestAddressPubkeyConverter.CreateAddressFromBytes(firstSCAddress)
	shId := nodes[0].ShardCoordinator.ComputeId(address)
	for _, node := range nodes {
		if node.ShardCoordinator.SelfId() != shId {
			continue
		}

		numCalled := vm.GetIntValueFromSC(nil, node.AccntState, firstSCAddress, "get", nil)
		require.NotNil(t, numCalled)
	}

	account := getAccountFromAddrBytes(nodes[0].AccntState, nodes[0].OwnAccount.Address.Bytes())
	require.Equal(t, big.NewInt(0), account.GetDeveloperReward())

	newOwnerAddress := []byte("12345678123456781234567812345678")
	txData := "ChangeOwnerAddress" + "@" + hex.EncodeToString(newOwnerAddress)
	integrationTests.CreateAndSendTransaction(nodes[0], big.NewInt(0), firstSCAddress, txData)

	for i := 0; i < numRoundsToPropagateMultiShard; i++ {
		integrationTests.UpdateRound(nodes, round)
		integrationTests.ProposeBlock(nodes, idxProposers, round, nonce)
		integrationTests.SyncBlock(t, nodes, idxProposers, round)
		round = integrationTests.IncrementAndPrintRound(round)
		nonce++
	}

	// check new owner address is set
	account = getAccountFromAddrBytes(nodes[0].AccntState, firstSCAddress)
	require.Equal(t, newOwnerAddress, account.GetOwnerAddress())
	require.True(t, account.GetDeveloperReward().Cmp(big.NewInt(0)) == 1)
}

func TestScDeployAndClaimSmartContractDeveloperRewards(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numShards := 2
	nodesPerShard := 2
	numMetachainNodes := 2

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)

	idxProposers := make([]int, numShards+1)
	for i := 0; i < numShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numShards] = numShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := big.NewInt(1000000000)
	integrationTests.MintAllNodes(nodes, initialVal)

	firstSCOwner := nodes[0].OwnAccount.Address.Bytes()

	// deploy the smart contracts
	firstSCAddress := putDeploySCToDataPool("./testdata/counter.wasm", firstSCOwner, 0, big.NewInt(50), "", nodes)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++
	integrationTests.ProposeBlock(nodes, idxProposers, round, nonce)
	integrationTests.SyncBlock(t, nodes, idxProposers, round)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// make smart contract call to shard 1 which will do in shard 0
	for _, node := range nodes {
		txData := "increment"
		for i := 0; i < 10; i++ {
			integrationTests.CreateAndSendTransaction(node, big.NewInt(0), firstSCAddress, txData)
		}
	}

	time.Sleep(time.Second)

	numRoundsToPropagateMultiShard := 15
	for i := 0; i < numRoundsToPropagateMultiShard; i++ {
		integrationTests.UpdateRound(nodes, round)
		integrationTests.ProposeBlock(nodes, idxProposers, round, nonce)
		integrationTests.SyncBlock(t, nodes, idxProposers, round)
		round = integrationTests.IncrementAndPrintRound(round)
		nonce++
	}

	address, _ := integrationTests.TestAddressPubkeyConverter.CreateAddressFromBytes(firstSCAddress)
	shId := nodes[0].ShardCoordinator.ComputeId(address)
	for _, node := range nodes {
		if node.ShardCoordinator.SelfId() != shId {
			continue
		}

		numCalled := vm.GetIntValueFromSC(nil, node.AccntState, firstSCAddress, "get", nil)
		require.NotNil(t, numCalled)
	}

	account := getAccountFromAddrBytes(nodes[0].AccntState, nodes[0].OwnAccount.Address.Bytes())
	require.Equal(t, big.NewInt(0), account.GetDeveloperReward())
	fmt.Println("smart contract owner before claim", account.GetBalance())
	oldOwnerBalance := big.NewInt(0).Set(account.GetBalance())

	account = getAccountFromAddrBytes(nodes[0].AccntState, firstSCAddress)
	fmt.Println("smart contract rewards balance", account.GetDeveloperReward())

	for _, node := range nodes {
		node.EconomicsData.SetGasPerDataByte(0)
		node.EconomicsData.SetMinGasLimit(0)
		node.EconomicsData.SetMinGasPrice(0)
	}

	txData := "ClaimDeveloperRewards"
	integrationTests.CreateAndSendTransaction(nodes[0], big.NewInt(0), firstSCAddress, txData)

	for i := 0; i < numRoundsToPropagateMultiShard; i++ {
		integrationTests.UpdateRound(nodes, round)
		integrationTests.ProposeBlock(nodes, idxProposers, round, nonce)
		integrationTests.SyncBlock(t, nodes, idxProposers, round)
		round = integrationTests.IncrementAndPrintRound(round)
		nonce++
	}

	account = getAccountFromAddrBytes(nodes[0].AccntState, nodes[0].OwnAccount.Address.Bytes())
	fmt.Println("smart contract owner after claim", account.GetBalance())
	require.True(t, account.GetBalance().Cmp(oldOwnerBalance) == 1)
}

func getAccountFromAddrBytes(accState state.AccountsAdapter, address []byte) state.UserAccountHandler {
	addrCont, _ := integrationTests.TestAddressPubkeyConverter.CreateAddressFromBytes(address)
	sndrAcc, _ := accState.GetExistingAccount(addrCont)

	sndAccSt, _ := sndrAcc.(state.UserAccountHandler)

	return sndAccSt
}

func TestSCCallingInCrossShard(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 2
	nodesPerShard := 3
	numMetachainNodes := 3

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
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

	initialVal := big.NewInt(10000000000000)
	initialVal.Mul(initialVal, initialVal)
	fmt.Printf("Initial minted sum: %s\n", initialVal.String())
	integrationTests.MintAllNodes(nodes, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// mint smart contract holders
	firstSCOwner := []byte("12345678901234567890123456789000")
	secondSCOwner := []byte("99945678901234567890123456789001")

	mintPubKey(firstSCOwner, initialVal, nodes)
	mintPubKey(secondSCOwner, initialVal, nodes)

	// deploy the smart contracts
	firstSCAddress := putDeploySCToDataPool("./testdata/first/first.wasm", firstSCOwner, 0, big.NewInt(50), "", nodes)
	//000000000000000005005d3d53b5d0fcf07d222170978932166ee9f3972d3030
	secondSCAddress := putDeploySCToDataPool("./testdata/second/second.wasm", secondSCOwner, 0, big.NewInt(50), "", nodes)
	//00000000000000000500017cc09151c48b99e2a1522fb70a5118ad4cb26c3031

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 1, nonce, round, idxProposers)

	// make smart contract call to shard 1 which will do in shard 0
	for _, node := range nodes {
		txData := "doSomething"
		integrationTests.CreateAndSendTransaction(node, big.NewInt(50), secondSCAddress, txData)
	}

	time.Sleep(time.Second)

	nrRoundsToPropagateMultiShard := 10
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)

	// verify how many times was shard 0 and shard 1 called
	address, _ := integrationTests.TestAddressPubkeyConverter.CreateAddressFromBytes(firstSCAddress)
	shId := nodes[0].ShardCoordinator.ComputeId(address)
	for index, node := range nodes {
		if node.ShardCoordinator.SelfId() != shId {
			continue
		}

		numCalled := vm.GetIntValueFromSC(nil, node.AccntState, firstSCAddress, "numCalled", nil)
		assert.NotNil(t, numCalled)
		if numCalled != nil {
			assert.Equal(t, uint64(len(nodes)), numCalled.Uint64(), fmt.Sprintf("Node %d, Shard %d", index, shId))
		}
	}
}

func TestSCCallingInCrossShardDelegationMock(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 2
	nodesPerShard := 3
	numMetachainNodes := 3
	shardConsensusGroupSize := 2
	metaConsensusGroupSize := 2

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()

	nodesMap := integrationTests.CreateNodesWithNodesCoordinator(
		nodesPerShard,
		numMetachainNodes,
		numOfShards,
		shardConsensusGroupSize,
		metaConsensusGroupSize,
		integrationTests.GetConnectableAddress(advertiser),
	)

	nodes := make([]*integrationTests.TestProcessorNode, 0)
	idxProposers := make([]int, numOfShards+1)

	for _, nds := range nodesMap {
		nodes = append(nodes, nds...)
	}

	for _, nds := range nodesMap {
		idx, err := getNodeIndex(nodes, nds[0])
		assert.Nil(t, err)

		idxProposers = append(idxProposers, idx)
	}

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := big.NewInt(1000000000)
	integrationTests.MintAllNodes(nodes, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// mint smart contract holders
	delegateSCOwner := []byte("12345678901234567890123456789002")
	stakerBLSKey, _ := hex.DecodeString(strings.Repeat("a", 256))

	mintPubKey(delegateSCOwner, initialVal, nodes)

	// deploy the smart contracts
	delegateSCAddress := putDeploySCToDataPool("./testdata/delegate-mock/delegate.wasm", delegateSCOwner, 0, big.NewInt(50), "", nodes)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 1, nonce, round, idxProposers)

	// one node calls to stake all the money from the delegation - that's how the contract is :D
	node := nodes[0]
	txData := "sendToStaking"
	integrationTests.CreateAndSendTransaction(node, node.EconomicsData.GenesisNodePrice(), delegateSCAddress, txData)

	time.Sleep(time.Second)

	nrRoundsToPropagateMultiShard := 10
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)

	time.Sleep(time.Second)
	// verify system smart contract has the value
	for _, n := range nodes {
		if n.ShardCoordinator.SelfId() != core.MetachainShardId {
			continue
		}
		scQuery := &process.SCQuery{
			ScAddress: factory2.StakingSCAddress,
			FuncName:  "isStaked",
			Arguments: [][]byte{stakerBLSKey},
		}
		vmOutput, _ := n.SCQueryService.ExecuteQuery(scQuery)

		assert.NotNil(t, vmOutput)
		if vmOutput != nil {
			assert.Equal(t, vmOutput.ReturnCode, vmcommon.Ok)
		}
	}
}

func TestSCCallingInCrossShardDelegation(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 2
	nodesPerShard := 3
	numMetachainNodes := 3
	shardConsensusGroupSize := 2
	metaConsensusGroupSize := 2

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()

	nodesMap := integrationTests.CreateNodesWithNodesCoordinator(
		nodesPerShard,
		numMetachainNodes,
		numOfShards,
		shardConsensusGroupSize,
		metaConsensusGroupSize,
		integrationTests.GetConnectableAddress(advertiser),
	)

	nodes := make([]*integrationTests.TestProcessorNode, 0)
	idxProposers := make([]int, numOfShards+1)

	for _, nds := range nodesMap {
		nodes = append(nodes, nds...)
	}

	for _, nds := range nodesMap {
		idx, err := getNodeIndex(nodes, nds[0])
		assert.Nil(t, err)

		idxProposers = append(idxProposers, idx)
	}

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := big.NewInt(1000000000)
	integrationTests.MintAllNodes(nodes, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// mint smart contract holders
	shardNode := findAnyShardNode(nodes)
	delegateSCOwner := shardNode.OwnAccount.Address.Bytes()
	totalStake := shardNode.EconomicsData.GenesisNodePrice()
	node_share_per_10000 := 3000
	stakerBLSKey, _ := hex.DecodeString(strings.Repeat("a", 128*2))
	stakerBLSSignature, _ := hex.DecodeString(strings.Repeat("c", 32*2))

	// deploy the delegation smart contract
	delegateSCAddress := putDeploySCToDataPool(
		"./testdata/delegate/delegation.wasm", delegateSCOwner, 0, big.NewInt(0),
		fmt.Sprintf("@%x@%x@%s", totalStake, node_share_per_10000, hex.EncodeToString(factory2.AuctionSCAddress)),
		nodes)
	shardNode.OwnAccount.Nonce++

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 1, nonce, round, idxProposers)

	// set BLS keys in the contract
	setBlsTxData := "setBlsKeys@1@" + hex.EncodeToString(stakerBLSKey)
	integrationTests.CreateAndSendTransaction(shardNode, big.NewInt(0), delegateSCAddress, setBlsTxData)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 1, nonce, round, idxProposers)

	// stake some coin!
	// here the node account fills all the required stake
	stakeTxData := "stake"
	integrationTests.CreateAndSendTransaction(shardNode, totalStake, delegateSCAddress, stakeTxData)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 1, nonce, round, idxProposers)

	// activate the delegation, this involves a call async to auction
	activateTxData := "activate@" + hex.EncodeToString(stakerBLSSignature)
	integrationTests.CreateAndSendTransaction(shardNode, big.NewInt(0), delegateSCAddress, activateTxData)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 1, nonce, round, idxProposers)

	time.Sleep(time.Second)

	nrRoundsToPropagateMultiShard := 10
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)

	time.Sleep(time.Second)

	// check that delegation contract was correctly initialized by querying for total stake
	scQuery1 := &process.SCQuery{
		ScAddress: delegateSCAddress,
		FuncName:  "getTotalStake",
		Arguments: [][]byte{},
	}
	vmOutput1, _ := shardNode.SCQueryService.ExecuteQuery(scQuery1)
	assert.NotNil(t, vmOutput1)
	assert.Equal(t, len(vmOutput1.ReturnData), 1)
	assert.True(t, totalStake.Cmp(big.NewInt(0).SetBytes(vmOutput1.ReturnData[0])) == 0)

	// check that BLS keys were correctly set to contract
	scQuery2 := &process.SCQuery{
		ScAddress: delegateSCAddress,
		FuncName:  "getBlsKeys",
		Arguments: [][]byte{},
	}
	vmOutput2, _ := shardNode.SCQueryService.ExecuteQuery(scQuery2)
	assert.NotNil(t, vmOutput2)
	assert.Equal(t, len(vmOutput2.ReturnData), 1)
	assert.True(t, bytes.Equal(stakerBLSKey, vmOutput2.ReturnData[0]))

	// check that the staking transaction worked
	scQuery3 := &process.SCQuery{
		ScAddress: delegateSCAddress,
		FuncName:  "getUnfilledStake",
		Arguments: [][]byte{},
	}
	vmOutput3, _ := shardNode.SCQueryService.ExecuteQuery(scQuery3)
	assert.NotNil(t, vmOutput3)
	assert.Equal(t, len(vmOutput3.ReturnData), 1)
	assert.True(t, len(vmOutput3.ReturnData[0]) == 0) // unfilled stake == 0

	// check that the staking system smart contract has the value
	for _, node := range nodes {
		if node.ShardCoordinator.SelfId() != core.MetachainShardId {
			continue
		}
		scQuery := &process.SCQuery{
			ScAddress: factory2.StakingSCAddress,
			FuncName:  "isStaked",
			Arguments: [][]byte{stakerBLSKey},
		}
		vmOutput, _ := node.SCQueryService.ExecuteQuery(scQuery)

		assert.NotNil(t, vmOutput)
		if vmOutput != nil {
			assert.Equal(t, vmOutput.ReturnCode, vmcommon.Ok)
		}
	}
}

func getNodeIndex(nodeList []*integrationTests.TestProcessorNode, node *integrationTests.TestProcessorNode) (int, error) {
	for i := range nodeList {
		if node == nodeList[i] {
			return i, nil
		}
	}

	return 0, errors.New("no such node in list")
}

func putDeploySCToDataPool(
	fileName string,
	pubkey []byte,
	nonce uint64,
	transferOnDeploy *big.Int,
	initArgs string,
	nodes []*integrationTests.TestProcessorNode,
) []byte {
	scCode, _ := ioutil.ReadFile(fileName)
	scCodeString := hex.EncodeToString(scCode)
	scCodeMetadataString := "0000"

	blockChainHook := nodes[0].BlockchainHook

	scAddressBytes, _ := blockChainHook.NewAddress(pubkey, nonce, factory.ArwenVirtualMachine)

	tx := &transaction.Transaction{
		Nonce:    nonce,
		Value:    new(big.Int).Set(transferOnDeploy),
		RcvAddr:  make([]byte, 32),
		SndAddr:  pubkey,
		GasPrice: nodes[0].EconomicsData.GetMinGasPrice(),
		GasLimit: nodes[0].EconomicsData.MaxGasLimitPerBlock() - 1,
		Data:     []byte(scCodeString + "@" + hex.EncodeToString(factory.ArwenVirtualMachine) + "@" + scCodeMetadataString + initArgs),
	}
	txHash, _ := core.CalculateHash(integrationTests.TestMarshalizer, integrationTests.TestHasher, tx)

	address, _ := integrationTests.TestAddressPubkeyConverter.CreateAddressFromBytes(pubkey)
	shId := nodes[0].ShardCoordinator.ComputeId(address)

	for _, node := range nodes {
		if node.ShardCoordinator.SelfId() != shId {
			continue
		}
		strCache := process.ShardCacherIdentifier(shId, shId)
		node.DataPool.Transactions().AddData(txHash, tx, strCache)
	}

	return scAddressBytes
}

func mintPubKey(
	pubkey []byte,
	initialVal *big.Int,
	nodes []*integrationTests.TestProcessorNode,
) {
	address, _ := integrationTests.TestAddressPubkeyConverter.CreateAddressFromBytes(pubkey)
	shId := nodes[0].ShardCoordinator.ComputeId(address)
	for _, node := range nodes {
		if node.ShardCoordinator.SelfId() != shId {
			continue
		}
		integrationTests.MintAddress(node.AccntState, pubkey, initialVal)
	}
}

func findAnyShardNode(nodes []*integrationTests.TestProcessorNode) *integrationTests.TestProcessorNode {
	for _, node := range nodes {
		if node.ShardCoordinator.SelfId() != core.MetachainShardId {
			return node
		}
	}
	panic("no shard nodes found in test")
}
