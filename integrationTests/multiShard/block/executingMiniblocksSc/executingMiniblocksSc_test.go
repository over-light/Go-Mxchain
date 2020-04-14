package executingMiniblocksSc

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	testBlock "github.com/ElrondNetwork/elrond-go/integrationTests/multiShard/block"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/stretchr/testify/assert"
)

var agarioFile = "../../../agarioV3.hex"

// TestShouldProcessBlocksInMultiShardArchitectureWithScTxsTopUpAndWithdrawOnlyProposers tests the following scenario:
// There are 2 shards and 1 meta, each with only one node (proposer).
// Shard 1's proposer deploys a SC. There is 1 round for proposing block that will create the SC account.
// Shard 0's proposer sends a topUp SC call tx and then there are another 6 blocks added to all blockchains.
// After that there is a first check that the topUp was made. Shard 0's proposer sends a withdraw SC call tx and after
// 12 more blocks the results are checked again
func TestProcessWithScTxsTopUpAndWithdrawOnlyProposers(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	scCode, err := ioutil.ReadFile(agarioFile)
	assert.Nil(t, err)

	maxShards := uint32(2)
	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()
	advertiserAddr := integrationTests.GetConnectableAddress(advertiser)

	nodeShard0 := integrationTests.NewTestProcessorNode(maxShards, 0, 0, advertiserAddr)
	nodeShard0.EconomicsData.SetMinGasPrice(0)

	nodeShard1 := integrationTests.NewTestProcessorNode(maxShards, 1, 1, advertiserAddr)
	nodeShard1.EconomicsData.SetMinGasPrice(0)

	hardCodedSk, _ := hex.DecodeString("5561d28b0d89fa425bbbf9e49a018b5d1e4a462c03d2efce60faf9ddece2af06")
	nodeShard1.LoadTxSignSkBytes(hardCodedSk)
	nodeMeta := integrationTests.NewTestProcessorNode(maxShards, core.MetachainShardId, 0, advertiserAddr)
	nodeMeta.EconomicsData.SetMinGasPrice(0)

	nodes := []*integrationTests.TestProcessorNode{nodeShard0, nodeShard1, nodeMeta}

	idxNodeShard0 := 0
	idxNodeShard1 := 1
	idxNodeMeta := 2
	idxProposers := []int{idxNodeShard0, idxNodeShard1, idxNodeMeta}

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	for _, n := range nodes {
		_ = n.Messenger.Bootstrap()
	}

	fmt.Println("Delaying for nodes p2p bootstrap...")
	time.Sleep(testBlock.StepDelay)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	initialVal := big.NewInt(10000000000)
	topUpValue := big.NewInt(500)
	withdrawValue := big.NewInt(10)
	integrationTests.MintAllNodes(nodes, initialVal)

	hardCodedScResultingAddress, _ := nodeShard1.BlockchainHook.NewAddress(
		nodes[idxNodeShard1].OwnAccount.Address.Bytes(),
		nodes[idxNodeShard1].OwnAccount.Nonce,
		factory.IELEVirtualMachine,
	)
	integrationTests.DeployScTx(nodes, idxNodeShard1, string(scCode), factory.IELEVirtualMachine)

	integrationTests.UpdateRound(nodes, round)
	integrationTests.ProposeBlock(nodes, idxProposers, round, nonce)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	integrationTests.NodeDoesTopUp(nodes, idxNodeShard0, topUpValue, hardCodedScResultingAddress)

	roundsToWait := 6
	for i := 0; i < roundsToWait; i++ {
		integrationTests.UpdateRound(nodes, round)
		integrationTests.ProposeBlock(nodes, idxProposers, round, nonce)
		round = integrationTests.IncrementAndPrintRound(round)
		nonce++
	}

	nodeWithSc := nodes[idxNodeShard1]
	nodeWithCaller := nodes[idxNodeShard0]

	integrationTests.CheckScTopUp(t, nodeWithSc, topUpValue, hardCodedScResultingAddress)
	integrationTests.CheckScBalanceOf(t, nodeWithSc, nodeWithCaller, topUpValue, hardCodedScResultingAddress)
	integrationTests.CheckSenderBalanceOkAfterTopUp(t, nodeWithCaller, initialVal, topUpValue)

	integrationTests.NodeDoesWithdraw(nodes, idxNodeShard0, withdrawValue, hardCodedScResultingAddress)

	roundsToWait = 12
	for i := 0; i < roundsToWait; i++ {
		integrationTests.UpdateRound(nodes, round)
		integrationTests.ProposeBlock(nodes, idxProposers, round, nonce)
		round = integrationTests.IncrementAndPrintRound(round)
		nonce++
	}

	expectedSC := integrationTests.CheckBalanceIsDoneCorrectlySCSideAndReturnExpectedVal(t, nodes, idxNodeShard1, topUpValue, withdrawValue, hardCodedScResultingAddress)
	integrationTests.CheckScBalanceOf(t, nodeWithSc, nodeWithCaller, expectedSC, hardCodedScResultingAddress)
	integrationTests.CheckSenderBalanceOkAfterTopUpAndWithdraw(t, nodeWithCaller, initialVal, topUpValue, withdrawValue)
}

// TestShouldProcessBlocksInMultiShardArchitectureWithScTxsJoinAndRewardProposersAndValidators tests the following scenario:
// There are 2 shards and 1 meta, each with one proposer and one validator.
// Shard 1's proposer deploys a SC. There is 1 round for proposing block that will create the SC account.
// Shard 0's proposer sends a joinGame SC call tx and then there are another 6 blocks added to all blockchains.
// After that there is a first check that the joinGame was made. Shard 1's proposer sends a rewardAndSendFunds SC call
// tx and after 6 more blocks the results are checked again
func TestProcessWithScTxsJoinAndRewardTwoNodesInShard(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	scCode, err := ioutil.ReadFile(agarioFile)
	assert.Nil(t, err)

	maxShards := uint32(2)
	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()
	advertiserAddr := integrationTests.GetConnectableAddress(advertiser)
	nodeProposerShard0 := integrationTests.NewTestProcessorNode(
		maxShards,
		0,
		0,
		advertiserAddr,
	)
	nodeProposerShard0.EconomicsData.SetMinGasPrice(0)

	nodeValidatorShard0 := integrationTests.NewTestProcessorNode(
		maxShards,
		0,
		0,
		advertiserAddr,
	)
	nodeValidatorShard0.EconomicsData.SetMinGasPrice(0)

	nodeProposerShard1 := integrationTests.NewTestProcessorNode(
		maxShards,
		1,
		1,
		advertiserAddr,
	)
	nodeProposerShard1.EconomicsData.SetMinGasPrice(0)

	hardCodedSk, _ := hex.DecodeString("5561d28b0d89fa425bbbf9e49a018b5d1e4a462c03d2efce60faf9ddece2af06")
	hardCodedScResultingAddress, _ := hex.DecodeString("00000000000000000100f79b7a0bb9c9b78e2f2abc03c81c1ab32b4a56114849")
	nodeProposerShard1.LoadTxSignSkBytes(hardCodedSk)

	nodeValidatorShard1 := integrationTests.NewTestProcessorNode(
		maxShards,
		1,
		1,
		advertiserAddr,
	)
	nodeValidatorShard1.EconomicsData.SetMinGasPrice(0)

	nodeProposerMeta := integrationTests.NewTestProcessorNode(
		maxShards,
		core.MetachainShardId,
		0,
		advertiserAddr,
	)
	nodeProposerMeta.EconomicsData.SetMinGasPrice(0)

	nodeValidatorMeta := integrationTests.NewTestProcessorNode(
		maxShards,
		core.MetachainShardId,
		0,
		advertiserAddr,
	)
	nodeValidatorMeta.EconomicsData.SetMinGasPrice(0)

	nodes := []*integrationTests.TestProcessorNode{
		nodeProposerShard0,
		nodeProposerShard1,
		nodeProposerMeta,
		nodeValidatorShard0,
		nodeValidatorShard1,
		nodeValidatorMeta,
	}

	idxProposerShard0 := 0
	idxProposerShard1 := 1
	idxProposerMeta := 2
	idxProposers := []int{idxProposerShard0, idxProposerShard1, idxProposerMeta}
	idxValidators := []int{3, 4, 5}

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	for _, n := range nodes {
		_ = n.Messenger.Bootstrap()
	}

	fmt.Println("Delaying for nodes p2p bootstrap...")
	time.Sleep(testBlock.StepDelay)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	initialVal := big.NewInt(10000000)
	topUpValue := big.NewInt(500)
	rewardValue := big.NewInt(10)
	integrationTests.MintAllNodes(nodes, initialVal)

	integrationTests.DeployScTx(nodes, idxProposerShard1, string(scCode), factory.IELEVirtualMachine)

	round, nonce = integrationTests.ProposeAndSyncOneBlock(t, nodes, idxProposers, round, nonce)

	integrationTests.PlayerJoinsGame(
		nodes,
		nodes[idxProposerShard0].OwnAccount,
		topUpValue,
		100,
		hardCodedScResultingAddress,
	)

	roundsToWait := 7
	for i := 0; i < roundsToWait; i++ {
		round, nonce = integrationTests.ProposeAndSyncOneBlock(t, nodes, idxProposers, round, nonce)
		idxValidators, idxProposers = idxProposers, idxValidators
	}

	nodeWithSc := nodes[idxProposerShard1]
	nodeWithCaller := nodes[idxProposerShard0]

	integrationTests.CheckScTopUp(t, nodeWithSc, topUpValue, hardCodedScResultingAddress)
	integrationTests.CheckSenderBalanceOkAfterTopUp(t, nodeWithCaller, initialVal, topUpValue)

	integrationTests.NodeCallsRewardAndSend(
		nodes,
		idxProposerShard1,
		nodes[idxProposerShard0].OwnAccount,
		rewardValue,
		100,
		hardCodedScResultingAddress,
	)

	roundsToWait = 7
	for i := 0; i < roundsToWait; i++ {
		round, nonce = integrationTests.ProposeAndSyncOneBlock(t, nodes, idxProposers, round, nonce)
		idxValidators, idxProposers = idxProposers, idxValidators
	}

	time.Sleep(time.Second)

	_ = integrationTests.CheckBalanceIsDoneCorrectlySCSideAndReturnExpectedVal(t, nodes, idxProposerShard1, topUpValue, big.NewInt(0), hardCodedScResultingAddress)
	integrationTests.CheckSenderBalanceOkAfterTopUpAndWithdraw(t, nodeWithCaller, initialVal, topUpValue, big.NewInt(0))
	integrationTests.CheckRootHashes(t, nodes, idxProposers)
}

// TestShouldProcessWithScTxsJoinNoCommitShouldProcessedByValidators tests the following scenario:
// There are 2 shards and 1 meta, each with one proposer and one validator.
// Shard 1's proposer deploys a SC. There is 1 round for proposing block that will create the SC account.
// Shard 0's proposer sends a joinGame SC call tx, proposes a block (not committing it) and the validator
// should be able to sync it.
// Test will fail with any variant before commit d79898991f83188118a1c60003f5277bc71209e6
func TestShouldProcessWithScTxsJoinNoCommitShouldProcessedByValidators(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	scCode, err := ioutil.ReadFile(agarioFile)
	assert.Nil(t, err)

	maxShards := uint32(2)
	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()
	advertiserAddr := integrationTests.GetConnectableAddress(advertiser)

	nodeProposerShard0 := integrationTests.NewTestProcessorNode(maxShards, 0, 0, advertiserAddr)
	nodeProposerShard0.EconomicsData.SetMinGasPrice(0)
	nodeValidatorShard0 := integrationTests.NewTestProcessorNode(maxShards, 0, 0, advertiserAddr)
	nodeValidatorShard0.EconomicsData.SetMinGasPrice(0)

	nodeProposerShard1 := integrationTests.NewTestProcessorNode(maxShards, 1, 1, advertiserAddr)
	nodeProposerShard1.EconomicsData.SetMinGasPrice(0)
	hardCodedSk, _ := hex.DecodeString("5561d28b0d89fa425bbbf9e49a018b5d1e4a462c03d2efce60faf9ddece2af06")
	hardCodedScResultingAddress, _ := hex.DecodeString("00000000000000000100f79b7a0bb9c9b78e2f2abc03c81c1ab32b4a56114849")
	nodeProposerShard1.LoadTxSignSkBytes(hardCodedSk)
	nodeValidatorShard1 := integrationTests.NewTestProcessorNode(maxShards, 1, 1, advertiserAddr)
	nodeValidatorShard1.EconomicsData.SetMinGasPrice(0)

	nodeProposerMeta := integrationTests.NewTestProcessorNode(maxShards, core.MetachainShardId, 0, advertiserAddr)
	nodeProposerMeta.EconomicsData.SetMinGasPrice(0)
	nodeValidatorMeta := integrationTests.NewTestProcessorNode(maxShards, core.MetachainShardId, 0, advertiserAddr)
	nodeValidatorMeta.EconomicsData.SetMinGasPrice(0)

	nodes := []*integrationTests.TestProcessorNode{
		nodeProposerShard0,
		nodeProposerShard1,
		nodeProposerMeta,
		nodeValidatorShard0,
		nodeValidatorShard1,
		nodeValidatorMeta,
	}

	idxProposerShard0 := 0
	idxProposerShard1 := 1
	idxProposerMeta := 2
	idxProposers := []int{idxProposerShard0, idxProposerShard1, idxProposerMeta}

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	for _, n := range nodes {
		_ = n.Messenger.Bootstrap()
	}

	fmt.Println("Delaying for nodes p2p bootstrap...")
	time.Sleep(testBlock.StepDelay)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	initialVal := big.NewInt(10000000)
	topUpValue := big.NewInt(500)
	integrationTests.MintAllNodes(nodes, initialVal)

	integrationTests.DeployScTx(nodes, idxProposerShard1, string(scCode), factory.IELEVirtualMachine)
	round, nonce = integrationTests.ProposeAndSyncOneBlock(t, nodes, idxProposers, round, nonce)

	integrationTests.PlayerJoinsGame(
		nodes,
		nodes[idxProposerShard0].OwnAccount,
		topUpValue,
		100,
		hardCodedScResultingAddress,
	)

	maxRoundsToWait := 10
	for i := 0; i < maxRoundsToWait; i++ {
		round, nonce = integrationTests.ProposeAndSyncOneBlock(t, nodes, idxProposers, round, nonce)
	}

	nodeWithSc := nodes[idxProposerShard1]
	nodeWithCaller := nodes[idxProposerShard0]

	integrationTests.CheckScTopUp(t, nodeWithSc, topUpValue, hardCodedScResultingAddress)
	integrationTests.CheckSenderBalanceOkAfterTopUp(t, nodeWithCaller, initialVal, topUpValue)
}

// TestShouldSubtractTheCorrectTxFee uses the mock VM as it's gas model is predictable
// The test checks the tx fee subtraction from the sender account when deploying a SC
// It also checks the fee obtained by the leader is correct
// Test parameters: 2 shards + meta, each with 2 nodes
func TestShouldSubtractTheCorrectTxFee(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	maxShards := 2
	consensusGroupSize := 2
	nodesPerShard := 2
	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()

	// create map of shards - testNodeProcessors for metachain and shard chain
	nodesMap := integrationTests.CreateNodesWithNodesCoordinator(
		nodesPerShard,
		nodesPerShard,
		maxShards,
		consensusGroupSize,
		consensusGroupSize,
		integrationTests.GetConnectableAddress(advertiser),
	)

	for _, nodes := range nodesMap {
		integrationTests.DisplayAndStartNodes(nodes)
		integrationTests.SetEconomicsParameters(nodes, integrationTests.MaxGasLimitPerBlock, integrationTests.MinTxGasPrice, integrationTests.MinTxGasLimit)
	}

	defer func() {
		_ = advertiser.Close()
		for _, nodes := range nodesMap {
			for _, n := range nodes {
				_ = n.Messenger.Close()
			}
		}
	}()

	fmt.Println("Delaying for nodes p2p bootstrap...")
	time.Sleep(testBlock.StepDelay)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	initialVal := big.NewInt(10000000)
	senders := integrationTests.CreateSendersWithInitialBalances(nodesMap, initialVal)

	deployValue := big.NewInt(0)
	nodeShard0 := nodesMap[0][0]
	txData := "DEADBEEF@" + hex.EncodeToString(factory.InternalTestingVM) + "@00"
	dummyTx := &transaction.Transaction{
		Data: []byte(txData),
	}
	gasLimit := nodeShard0.EconomicsData.ComputeGasLimit(dummyTx)
	gasLimit += integrationTests.OpGasValueForMockVm
	gasPrice := integrationTests.MinTxGasPrice
	txNonce := uint64(0)
	owner := senders[0][0]
	ownerPk, _ := owner.GeneratePublic().ToByteArray()
	ownerAddr, _ := integrationTests.TestAddressPubkeyConverter.CreateAddressFromBytes(ownerPk)
	integrationTests.ScCallTxWithParams(
		nodeShard0,
		owner,
		txNonce,
		txData,
		deployValue,
		gasLimit,
		gasPrice,
	)

	_, _, consensusNodes := integrationTests.AllShardsProposeBlock(round, nonce, nodesMap)
	shardId0 := uint32(0)

	_ = integrationTests.IncrementAndPrintRound(round)

	// test sender account decreased its balance with gasPrice * gasLimit
	accnt, err := consensusNodes[shardId0][0].AccntState.GetExistingAccount(ownerAddr)
	assert.Nil(t, err)
	ownerAccnt := accnt.(state.UserAccountHandler)
	expectedBalance := big.NewInt(0).Set(initialVal)
	txCost := big.NewInt(0).SetUint64(gasPrice * gasLimit)
	expectedBalance.Sub(expectedBalance, txCost)
	assert.Equal(t, expectedBalance, ownerAccnt.GetBalance())

	printContainingTxs(consensusNodes[shardId0][0], consensusNodes[shardId0][0].BlockChain.GetCurrentBlockHeader().(*block.Header))
}

func printContainingTxs(tpn *integrationTests.TestProcessorNode, hdr *block.Header) {
	for _, miniblockHdr := range hdr.MiniBlockHeaders {
		miniblockBytes, err := tpn.Storage.Get(dataRetriever.MiniBlockUnit, miniblockHdr.Hash)
		if err != nil {
			fmt.Println("miniblock " + base64.StdEncoding.EncodeToString(miniblockHdr.Hash) + "not found")
			continue
		}

		miniblock := &block.MiniBlock{}
		err = integrationTests.TestMarshalizer.Unmarshal(miniblock, miniblockBytes)
		if err != nil {
			fmt.Println("can not unmarshal miniblock " + base64.StdEncoding.EncodeToString(miniblockHdr.Hash))
			continue
		}

		for _, txHash := range miniblock.TxHashes {
			txBytes := []byte("not found")

			mbType := miniblockHdr.Type
			switch mbType {
			case block.TxBlock:
				txBytes, err = tpn.Storage.Get(dataRetriever.TransactionUnit, txHash)
				if err != nil {
					fmt.Println("tx hash " + base64.StdEncoding.EncodeToString(txHash) + " not found")
					continue
				}
			case block.SmartContractResultBlock:
				txBytes, err = tpn.Storage.Get(dataRetriever.UnsignedTransactionUnit, txHash)
				if err != nil {
					fmt.Println("scr hash " + base64.StdEncoding.EncodeToString(txHash) + " not found")
					continue
				}
			case block.RewardsBlock:
				txBytes, err = tpn.Storage.Get(dataRetriever.RewardTransactionUnit, txHash)
				if err != nil {
					fmt.Println("reward hash " + base64.StdEncoding.EncodeToString(txHash) + " not found")
					continue
				}
			}

			fmt.Println(string(txBytes))
		}
	}
}
