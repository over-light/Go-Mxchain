package epochChangeWithNodesShufflingAndRater

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/display"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/ElrondNetwork/elrond-go/integrationTests/multiShard/endOfEpoch"
	"github.com/ElrondNetwork/elrond-go/process/rating"
)

func TestEpochChangeWithNodesShufflingAndRater(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	_ = display.SetDisplayByteSlice(display.ToHexShort)

	nodesPerShard := 1
	nbMetaNodes := 1
	nbShards := 1
	consensusGroupSize := 1
	maxGasLimitPerBlock := uint64(100000)

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()

	seedAddress := integrationTests.GetConnectableAddress(advertiser)

	rater, _ := rating.NewBlockSigningRater(integrationTests.CreateEconomicsData().RatingsData())

	coordinatorFactory := &integrationTests.IndexHashedNodesCoordinatorWithRaterFactory{
		RaterHandler: rater,
	}

	// create map of shard - testNodeProcessors for metachain and shard chain
	nodesMap := integrationTests.CreateNodesWithNodesCoordinatorFactory(
		nodesPerShard,
		nbMetaNodes,
		nbShards,
		consensusGroupSize,
		consensusGroupSize,
		seedAddress,
		coordinatorFactory,
	)

	gasPrice := uint64(10)
	gasLimit := uint64(100)
	valToTransfer := big.NewInt(100)
	nbTxsPerShard := uint32(100)
	mintValue := big.NewInt(1000000)

	defer func() {
		_ = advertiser.Close()
		for _, nodes := range nodesMap {
			for _, n := range nodes {
				_ = n.Node.Stop()
			}
		}
	}()

	roundsPerEpoch := uint64(7)
	for _, nodes := range nodesMap {
		integrationTests.SetEconomicsParameters(nodes, maxGasLimitPerBlock, gasPrice, gasLimit)
		integrationTests.DisplayAndStartNodes(nodes)
		for _, node := range nodes {
			node.EpochStartTrigger.SetRoundsPerEpoch(roundsPerEpoch)
		}
	}

	integrationTests.GenerateIntraShardTransactions(nodesMap, nbTxsPerShard, mintValue, valToTransfer, gasPrice, gasLimit)

	round := uint64(1)
	nonce := uint64(1)
	nbBlocksToProduce := uint64(20)
	expectedLastEpoch := uint32(nbBlocksToProduce / roundsPerEpoch)
	var consensusNodes map[uint32][]*integrationTests.TestProcessorNode

	for i := uint64(0); i < nbBlocksToProduce; i++ {
		for _, nodes := range nodesMap {
			integrationTests.UpdateRound(nodes, round)
		}

		_, _, consensusNodes = integrationTests.AllShardsProposeBlock(round, nonce, nodesMap)
		indexesProposers := endOfEpoch.GetBlockProposersIndexes(consensusNodes, nodesMap)
		integrationTests.SyncAllShardsWithRoundBlock(t, nodesMap, indexesProposers, round)
		round++
		nonce++

		time.Sleep(5 * time.Second)
	}

	for _, nodes := range nodesMap {
		endOfEpoch.VerifyIfNodesHasCorrectEpoch(t, expectedLastEpoch, nodes)
	}
}
