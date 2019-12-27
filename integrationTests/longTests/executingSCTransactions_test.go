package longTests

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core/statistics"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/pkg/profile"
	"github.com/stretchr/testify/assert"
)

var agarioFile = "../agar_v1_min.hex"
var stepDelay = time.Second

func TestProcessesJoinGameTheSamePlayerMultipleTimesRewardAndEndgameInMultipleRounds(t *testing.T) {
	t.Skip("this is a stress test for VM and AGAR.IO")

	p := profile.Start(profile.MemProfile, profile.ProfilePath("."), profile.NoShutdownHook)
	defer p.Stop()

	scCode, err := ioutil.ReadFile(agarioFile)
	assert.Nil(t, err)

	maxShards := uint32(1)
	numOfNodes := 1
	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()
	advertiserAddr := integrationTests.GetConnectableAddress(advertiser)

	nodes := make([]*integrationTests.TestProcessorNode, numOfNodes)
	for i := 0; i < numOfNodes; i++ {
		nodes[i] = integrationTests.NewTestProcessorNode(maxShards, 0, 0, advertiserAddr)
	}

	idxProposer := 0
	numPlayers := 100
	players := make([]*integrationTests.TestWalletAccount, numPlayers)
	for i := 0; i < numPlayers; i++ {
		players[i] = integrationTests.CreateTestWalletAccount(nodes[idxProposer].ShardCoordinator, 0)
	}

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
	time.Sleep(stepDelay)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	hardCodedSk, _ := hex.DecodeString("5561d28b0d89fa425bbbf9e49a018b5d1e4a462c03d2efce60faf9ddece2af06")
	hardCodedScResultingAddress, _ := hex.DecodeString("000000000000000001006c560111a94e434413c1cdaafbc3e1348947d1d5b3a1")
	nodes[idxProposer].LoadTxSignSkBytes(hardCodedSk)

	initialVal := big.NewInt(10000000)
	topUpValue := big.NewInt(500)
	integrationTests.MintAllNodes(nodes, initialVal)
	integrationTests.MintAllPlayers(nodes, players, initialVal)

	integrationTests.DeployScTx(nodes, idxProposer, string(scCode), factory.IELEVirtualMachine)
	time.Sleep(stepDelay)
	integrationTests.ProposeBlock(nodes, []int{idxProposer}, round, nonce)
	integrationTests.SyncBlock(t, nodes, []int{idxProposer}, round)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	numRounds := 10
	runMultipleRoundsOfTheGame(
		t,
		numRounds,
		numPlayers,
		nodes,
		players,
		topUpValue,
		hardCodedScResultingAddress,
		round,
		nonce,
		[]int{idxProposer},
	)

	integrationTests.CheckRootHashes(t, nodes, []int{idxProposer})

	time.Sleep(time.Second)
}

func TestProcessesJoinGame100PlayersMultipleTimesRewardAndEndgameInMultipleRounds(t *testing.T) {
	t.Skip("this is a stress test for VM and AGAR.IO")

	p := profile.Start(profile.MemProfile, profile.ProfilePath("."), profile.NoShutdownHook)
	defer p.Stop()

	scCode, err := ioutil.ReadFile(agarioFile)
	assert.Nil(t, err)

	maxShards := uint32(1)
	numOfNodes := 4
	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()
	advertiserAddr := integrationTests.GetConnectableAddress(advertiser)

	nodes := make([]*integrationTests.TestProcessorNode, numOfNodes)
	for i := 0; i < numOfNodes; i++ {
		nodes[i] = integrationTests.NewTestProcessorNode(maxShards, 0, 0, advertiserAddr)
	}

	idxProposer := 0
	numPlayers := 100
	players := make([]*integrationTests.TestWalletAccount, numPlayers)
	for i := 1; i < numPlayers; i++ {
		players[i] = integrationTests.CreateTestWalletAccount(nodes[idxProposer].ShardCoordinator, 0)
	}

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
	time.Sleep(stepDelay)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	hardCodedSk, _ := hex.DecodeString("5561d28b0d89fa425bbbf9e49a018b5d1e4a462c03d2efce60faf9ddece2af06")
	hardCodedScResultingAddress, _ := hex.DecodeString("000000000000000001006c560111a94e434413c1cdaafbc3e1348947d1d5b3a1")
	nodes[idxProposer].LoadTxSignSkBytes(hardCodedSk)

	initialVal := big.NewInt(10000000)
	topUpValue := big.NewInt(500)
	integrationTests.MintAllNodes(nodes, initialVal)
	integrationTests.MintAllPlayers(nodes, players, initialVal)

	integrationTests.DeployScTx(nodes, idxProposer, string(scCode), factory.IELEVirtualMachine)
	time.Sleep(stepDelay)
	integrationTests.ProposeBlock(nodes, []int{idxProposer}, round, nonce)
	integrationTests.SyncBlock(t, nodes, []int{idxProposer}, round)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	numRounds := 100
	runMultipleRoundsOfTheGame(
		t,
		numRounds,
		numPlayers,
		nodes,
		players,
		topUpValue,
		hardCodedScResultingAddress,
		round,
		nonce,
		[]int{idxProposer},
	)

	integrationTests.CheckRootHashes(t, nodes, []int{idxProposer})

	time.Sleep(time.Second)
}

func TestProcessesJoinGame100PlayersMultipleTimesRewardAndEndgameInMultipleRoundsMultiShard(t *testing.T) {
	t.Skip("this is a stress test for VM and AGAR.IO")

	p := profile.Start(profile.MemProfile, profile.ProfilePath("."), profile.NoShutdownHook)
	defer p.Stop()

	scCode, err := ioutil.ReadFile(agarioFile)
	assert.Nil(t, err)

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()
	advertiserAddr := integrationTests.GetConnectableAddress(advertiser)

	numOfNodes := 3
	maxShards := uint32(2)
	nodes := make([]*integrationTests.TestProcessorNode, numOfNodes)
	nodes[0] = integrationTests.NewTestProcessorNode(maxShards, 0, 0, advertiserAddr)
	nodes[1] = integrationTests.NewTestProcessorNode(maxShards, 1, 1, advertiserAddr)
	nodes[2] = integrationTests.NewTestProcessorNode(maxShards, sharding.MetachainShardId, 1, advertiserAddr)

	idxProposer := 0
	numPlayers := 100
	players := make([]*integrationTests.TestWalletAccount, numPlayers)
	for i := 1; i < numPlayers; i++ {
		players[i] = integrationTests.CreateTestWalletAccount(nodes[idxProposer].ShardCoordinator, 0)
	}

	idxProposers := make([]int, 3)
	idxProposers[0] = idxProposer
	idxProposers[1] = 1
	idxProposers[2] = 2

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
	time.Sleep(stepDelay)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	hardCodedSk, _ := hex.DecodeString("5561d28b0d89fa425bbbf9e49a018b5d1e4a462c03d2efce60faf9ddece2af06")
	hardCodedScResultingAddress, _ := hex.DecodeString("000000000000000001006c560111a94e434413c1cdaafbc3e1348947d1d5b3a1")
	nodes[idxProposer].LoadTxSignSkBytes(hardCodedSk)

	initialVal := big.NewInt(10000000)
	topUpValue := big.NewInt(500)
	integrationTests.MintAllNodes(nodes, initialVal)
	integrationTests.MintAllPlayers(nodes, players, initialVal)

	nrRoundsToPropagateMultiShard := 5
	if maxShards == 1 {
		nrRoundsToPropagateMultiShard = 1
	}

	integrationTests.DeployScTx(nodes, idxProposer, string(scCode), factory.IELEVirtualMachine)
	time.Sleep(stepDelay)
	for i := 0; i < nrRoundsToPropagateMultiShard; i++ {
		integrationTests.ProposeBlock(nodes, idxProposers, round, nonce)
		integrationTests.SyncBlock(t, nodes, idxProposers, round)
		round = integrationTests.IncrementAndPrintRound(round)
		nonce++
	}

	numRounds := 100
	runMultipleRoundsOfTheGame(
		t,
		numRounds,
		numPlayers,
		nodes,
		players,
		topUpValue,
		hardCodedScResultingAddress,
		round,
		nonce,
		idxProposers,
	)

	integrationTests.CheckRootHashes(t, nodes, idxProposers)

	time.Sleep(time.Second)
}

func TestProcessesJoinGame100PlayersMultipleTimesRewardAndEndgameInMultipleRoundsMultiShardMultiNode(t *testing.T) {
	t.Skip("this is a stress test for VM and AGAR.IO")

	p := profile.Start(profile.MemProfile, profile.ProfilePath("."), profile.NoShutdownHook)
	defer p.Stop()

	scCode, err := ioutil.ReadFile(agarioFile)
	assert.Nil(t, err)

	maxShards := uint32(2)
	numOfNodes := 6
	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()
	advertiserAddr := integrationTests.GetConnectableAddress(advertiser)

	nodes := make([]*integrationTests.TestProcessorNode, numOfNodes)
	nodes[0] = integrationTests.NewTestProcessorNode(maxShards, 0, 0, advertiserAddr)
	nodes[1] = integrationTests.NewTestProcessorNode(maxShards, 0, 0, advertiserAddr)
	nodes[2] = integrationTests.NewTestProcessorNode(maxShards, 1, 0, advertiserAddr)
	nodes[3] = integrationTests.NewTestProcessorNode(maxShards, 1, 0, advertiserAddr)
	nodes[4] = integrationTests.NewTestProcessorNode(maxShards, sharding.MetachainShardId, 0, advertiserAddr)
	nodes[5] = integrationTests.NewTestProcessorNode(maxShards, sharding.MetachainShardId, 0, advertiserAddr)

	idxProposer := 0
	numPlayers := 100
	players := make([]*integrationTests.TestWalletAccount, numPlayers)
	for i := 1; i < numPlayers; i++ {
		players[i] = integrationTests.CreateTestWalletAccount(nodes[idxProposer].ShardCoordinator, 0)
	}

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
	time.Sleep(stepDelay)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	hardCodedSk, _ := hex.DecodeString("5561d28b0d89fa425bbbf9e49a018b5d1e4a462c03d2efce60faf9ddece2af06")
	hardCodedScResultingAddress, _ := hex.DecodeString("000000000000000001006c560111a94e434413c1cdaafbc3e1348947d1d5b3a1")
	nodes[idxProposer].LoadTxSignSkBytes(hardCodedSk)

	initialVal := big.NewInt(10000000)
	topUpValue := big.NewInt(500)
	integrationTests.MintAllNodes(nodes, initialVal)
	integrationTests.MintAllPlayers(nodes, players, initialVal)

	nrRoundsToPropagateMultiShard := 5
	if maxShards == 1 {
		nrRoundsToPropagateMultiShard = 1
	}

	idxProposers := make([]int, 3)
	idxProposers[0] = 0
	idxProposers[1] = 2
	idxProposers[2] = 4

	integrationTests.DeployScTx(nodes, idxProposer, string(scCode), factory.IELEVirtualMachine)
	time.Sleep(stepDelay)
	for i := 0; i < nrRoundsToPropagateMultiShard; i++ {
		integrationTests.ProposeBlock(nodes, idxProposers, round, nonce)
		integrationTests.SyncBlock(t, nodes, idxProposers, round)
		round = integrationTests.IncrementAndPrintRound(round)
		nonce++
	}

	numRounds := 100
	runMultipleRoundsOfTheGame(
		t,
		numRounds,
		numPlayers,
		nodes,
		players,
		topUpValue,
		hardCodedScResultingAddress,
		round,
		nonce,
		idxProposers,
	)

	integrationTests.CheckRootHashes(t, nodes, []int{0})

	time.Sleep(time.Second)
}

func getPercentageOfValue(value *big.Int, percentage float64) *big.Int {
	x := new(big.Float).SetInt(value)
	y := big.NewFloat(percentage)

	z := new(big.Float).Mul(x, y)

	op := big.NewInt(0)
	result, _ := z.Int(op)

	return result
}

func runMultipleRoundsOfTheGame(
	t *testing.T,
	nrRounds, numPlayers int,
	nodes []*integrationTests.TestProcessorNode,
	players []*integrationTests.TestWalletAccount,
	topUpValue *big.Int,
	hardCodedScResultingAddress []byte,
	round uint64,
	nonce uint64,
	idxProposers []int,
) {
	rMonitor := &statistics.ResourceMonitor{}
	numRewardedPlayers := 10
	if numRewardedPlayers > numPlayers {
		numRewardedPlayers = numPlayers
	}

	totalWithdrawValue := big.NewInt(0).SetUint64(topUpValue.Uint64() * uint64(len(players)))
	withdrawValues := make([]*big.Int, numRewardedPlayers)
	winnerRate := 1.0 - 0.05*float64(numRewardedPlayers-1)
	withdrawValues[0] = big.NewInt(0).Set(getPercentageOfValue(totalWithdrawValue, winnerRate))
	for i := 1; i < numRewardedPlayers; i++ {
		withdrawValues[i] = big.NewInt(0).Set(getPercentageOfValue(totalWithdrawValue, 0.05))
	}

	for currentRound := 0; currentRound < nrRounds; currentRound++ {
		for _, player := range players {
			integrationTests.PlayerJoinsGame(
				nodes,
				player,
				topUpValue,
				int32(currentRound),
				hardCodedScResultingAddress,
			)
		}

		// waiting to disseminate transactions
		time.Sleep(stepDelay)

		round, nonce = integrationTests.ProposeAndSyncBlocks(t, nodes, idxProposers, round, nonce)

		for i := 0; i < numRewardedPlayers; i++ {
			integrationTests.NodeCallsRewardAndSend(nodes, idxProposers[0], players[i], withdrawValues[i], int32(currentRound), hardCodedScResultingAddress)
		}

		// waiting to disseminate transactions
		time.Sleep(stepDelay)

		round, nonce = integrationTests.ProposeAndSyncBlocks(t, nodes, idxProposers, round, nonce)

		fmt.Println(rMonitor.GenerateStatistics())
	}

	integrationTests.CheckRewardsDistribution(t, nodes, players, topUpValue, totalWithdrawValue,
		hardCodedScResultingAddress, idxProposers[0])
}
