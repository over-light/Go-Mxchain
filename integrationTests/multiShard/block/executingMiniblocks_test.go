package block

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/sharding/mock"
	"github.com/stretchr/testify/assert"
)

func TestShouldProcessBlocksInMultiShardArchitecture(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	fmt.Println("Setup nodes...")
	numOfShards := 6
	nodesPerShard := 3
	numMetachainNodes := 1

	idxProposers := []int{0, 3, 6, 9, 12, 15, 18}
	senderShard := uint32(0)
	recvShards := []uint32{1, 2}
	round := uint64(0)
	nonce := uint64(0)

	valMinting := big.NewInt(100000)
	valToTransferPerTx := big.NewInt(2)

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)
	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Node.Stop()
		}
	}()

	fmt.Println("Generating private keys for senders and receivers...")
	generateCoordinator, _ := sharding.NewMultiShardCoordinator(uint32(numOfShards), 0)
	txToGenerateInEachMiniBlock := 3

	proposerNode := nodes[0]

	//sender shard keys, receivers  keys
	sendersPrivateKeys := make([]crypto.PrivateKey, 3)
	receiversPublicKeys := make(map[uint32][]crypto.PublicKey)
	for i := 0; i < txToGenerateInEachMiniBlock; i++ {
		sendersPrivateKeys[i], _, _ = integrationTests.GenerateSkAndPkInShard(generateCoordinator, senderShard)
		//receivers in same shard with the sender
		_, pk, _ := integrationTests.GenerateSkAndPkInShard(generateCoordinator, senderShard)
		receiversPublicKeys[senderShard] = append(receiversPublicKeys[senderShard], pk)
		//receivers in other shards
		for _, shardId := range recvShards {
			_, pk, _ = integrationTests.GenerateSkAndPkInShard(generateCoordinator, shardId)
			receiversPublicKeys[shardId] = append(receiversPublicKeys[shardId], pk)
		}
	}

	fmt.Println("Minting sender addresses...")
	integrationTests.CreateMintingForSenders(nodes, senderShard, sendersPrivateKeys, valMinting)

	fmt.Println("Generating transactions...")
	integrationTests.GenerateAndDisseminateTxs(
		proposerNode,
		sendersPrivateKeys,
		receiversPublicKeys,
		valToTransferPerTx,
		integrationTests.MinTxGasPrice,
		integrationTests.MinTxGasLimit,
	)
	fmt.Println("Delaying for disseminating transactions...")
	time.Sleep(time.Second * 5)

	round = integrationTests.IncrementAndPrintRound(round)
	nonce++
	roundsToWait := 6
	for i := 0; i < roundsToWait; i++ {
		round, nonce = integrationTests.ProposeAndSyncOneBlock(t, nodes, idxProposers, round, nonce)
	}

	gasPricePerTxBigInt := big.NewInt(0).SetUint64(integrationTests.MinTxGasPrice)
	gasLimitPerTxBigInt := big.NewInt(0).SetUint64(integrationTests.MinTxGasLimit)
	gasValue := big.NewInt(0).Mul(gasPricePerTxBigInt, gasLimitPerTxBigInt)
	totalValuePerTx := big.NewInt(0).Add(gasValue, valToTransferPerTx)
	fmt.Println("Test nodes from proposer shard to have the correct balances...")
	for _, n := range nodes {
		isNodeInSenderShard := n.ShardCoordinator.SelfId() == senderShard
		if !isNodeInSenderShard {
			continue
		}

		//test sender balances
		for _, sk := range sendersPrivateKeys {
			valTransferred := big.NewInt(0).Mul(totalValuePerTx, big.NewInt(int64(len(receiversPublicKeys))))
			valRemaining := big.NewInt(0).Sub(valMinting, valTransferred)
			integrationTests.TestPrivateKeyHasBalance(t, n, sk, valRemaining)
		}
		//test receiver balances from same shard
		for _, pk := range receiversPublicKeys[proposerNode.ShardCoordinator.SelfId()] {
			integrationTests.TestPublicKeyHasBalance(t, n, pk, valToTransferPerTx)
		}
	}

	fmt.Println("Test nodes from receiver shards to have the correct balances...")
	for _, n := range nodes {
		isNodeInReceiverShardAndNotProposer := false
		for _, shardId := range recvShards {
			if n.ShardCoordinator.SelfId() == shardId {
				isNodeInReceiverShardAndNotProposer = true
				break
			}
		}
		if !isNodeInReceiverShardAndNotProposer {
			continue
		}

		//test receiver balances from same shard
		for _, pk := range receiversPublicKeys[n.ShardCoordinator.SelfId()] {
			integrationTests.TestPublicKeyHasBalance(t, n, pk, valToTransferPerTx)
		}
	}
}

func TestSimpleTransactionsWithMoreGasWhichYieldInReceiptsInMultiShardedEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 2
	nodesPerShard := 3
	numMetachainNodes := 2

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)

	minGasLimit := uint64(10000)
	for _, node := range nodes {
		node.EconomicsData.SetMinGasLimit(minGasLimit)
	}

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Node.Stop()
		}
	}()

	initialVal := big.NewInt(10000000)
	sendValue := big.NewInt(5)
	integrationTests.MintAllNodes(nodes, initialVal)
	receiverAddress := []byte("12345678901234567890123456789012")

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	gasLimit := minGasLimit * 2
	time.Sleep(time.Second)
	nrRoundsToTest := 10
	for i := 0; i <= nrRoundsToTest; i++ {
		integrationTests.UpdateRound(nodes, round)
		integrationTests.ProposeBlock(nodes, idxProposers, round, nonce)
		integrationTests.SyncBlock(t, nodes, idxProposers, round)
		round = integrationTests.IncrementAndPrintRound(round)
		nonce++

		for _, node := range nodes {
			integrationTests.CreateAndSendTransactionWithGasLimit(node, sendValue, gasLimit, receiverAddress, []byte(""))
		}

		time.Sleep(2 * time.Second)
	}

	time.Sleep(time.Second)

	txGasNeed := nodes[0].EconomicsData.GetMinGasLimit()
	txGasPrice := nodes[0].EconomicsData.GetMinGasPrice()

	oneTxCost := big.NewInt(0).Add(sendValue, big.NewInt(0).SetUint64(txGasNeed*txGasPrice))
	txTotalCost := big.NewInt(0).Mul(oneTxCost, big.NewInt(int64(nrRoundsToTest)))

	expectedBalance := big.NewInt(0).Sub(initialVal, txTotalCost)
	for _, verifierNode := range nodes {
		for _, node := range nodes {
			accWrp, err := verifierNode.AccntState.GetExistingAccount(node.OwnAccount.Address)
			if err != nil {
				continue
			}

			account, _ := accWrp.(*state.Account)
			assert.Equal(t, expectedBalance, account.Balance)
		}
	}
}

func TestSimpleTransactionsWithMoreValueThanBalanceYieldReceiptsInMultiShardedEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 2
	nodesPerShard := 2
	numMetachainNodes := 2

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)

	minGasLimit := uint64(10000)
	for _, node := range nodes {
		node.EconomicsData.SetMinGasLimit(minGasLimit)
	}

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Node.Stop()
		}
	}()

	nrTxsToSend := uint64(10)
	initialVal := big.NewInt(0).SetUint64(nrTxsToSend * minGasLimit * integrationTests.MinTxGasPrice)
	halfInitVal := big.NewInt(0).Div(initialVal, big.NewInt(2))
	integrationTests.MintAllNodes(nodes, initialVal)
	receiverAddress := []byte("12345678901234567890123456789012")

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	for _, node := range nodes {
		for j := uint64(0); j < nrTxsToSend; j++ {
			integrationTests.CreateAndSendTransactionWithGasLimit(node, halfInitVal, minGasLimit, receiverAddress, []byte(""))
		}
	}

	time.Sleep(2 * time.Second)

	integrationTests.UpdateRound(nodes, round)
	integrationTests.ProposeBlock(nodes, idxProposers, round, nonce)
	integrationTests.SyncBlock(t, nodes, idxProposers, round)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	for _, node := range nodes {
		if node.ShardCoordinator.SelfId() == sharding.MetachainShardId {
			continue
		}

		bodyHandler := node.BlockChain.GetCurrentBlockBody()
		body, _ := bodyHandler.(block.Body)
		numInvalid := 0
		for _, mb := range body {
			if mb.Type == block.InvalidBlock {
				numInvalid++
			}
		}
		assert.Equal(t, 1, numInvalid)
	}

	time.Sleep(time.Second)
	numRoundsToTest := 6
	for i := 0; i < numRoundsToTest; i++ {
		integrationTests.UpdateRound(nodes, round)
		integrationTests.ProposeBlock(nodes, idxProposers, round, nonce)
		integrationTests.SyncBlock(t, nodes, idxProposers, round)
		round = integrationTests.IncrementAndPrintRound(round)
		nonce++

		time.Sleep(time.Second)
	}

	time.Sleep(time.Second)

	expectedReceiverValue := big.NewInt(0).Mul(big.NewInt(int64(len(nodes))), halfInitVal)
	for _, verifierNode := range nodes {
		for _, node := range nodes {
			accWrp, err := verifierNode.AccntState.GetExistingAccount(node.OwnAccount.Address)
			if err != nil {
				continue
			}

			account, _ := accWrp.(*state.Account)
			assert.Equal(t, big.NewInt(0), account.Balance)
		}

		receiver, _ := integrationTests.TestAddressConverter.CreateAddressFromPublicKeyBytes(receiverAddress)
		accWrp, err := verifierNode.AccntState.GetExistingAccount(receiver)
		if err != nil {
			continue
		}

		account, _ := accWrp.(*state.Account)
		assert.Equal(t, expectedReceiverValue, account.Balance)
	}
}

func TestExecuteBlocksWithGapsBetweenBlocks(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}
	nodesPerShard := 2
	nbMetaNodes := 400
	nbShards := 1
	consensusGroupSize := 63

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()

	seedAddress := integrationTests.GetConnectableAddress(advertiser)

	getCounter := 0
	putCounter := 0

	cacheMap := make(map[string]interface{})
	cache := &mock.NodesCoordinatorCacheMock{
		PutCalled: func(key []byte, value interface{}) (evicted bool) {
			putCounter++
			cacheMap[string(key)] = value
			return false
		},
		GetCalled: func(key []byte) (value interface{}, ok bool) {
			getCounter++
			val, ok := cacheMap[string(key)]
			if ok {
				return val, true
			}
			return nil, false
		},
	}

	// create map of shard - testNodeProcessors for metachain and shard chain
	nodesMap := integrationTests.CreateNodesWithNodesCoordinatorWithCacher(
		nodesPerShard,
		nbMetaNodes,
		nbShards,
		2,
		consensusGroupSize,
		seedAddress,
		cache,
	)

	roundsPerEpoch := uint64(1000)
	maxGasLimitPerBlock := uint64(100000)
	gasPrice := uint64(10)
	gasLimit := uint64(100)
	for _, nodes := range nodesMap {
		integrationTests.SetEconomicsParameters(nodes, maxGasLimitPerBlock, gasPrice, gasLimit)
		integrationTests.DisplayAndStartNodes(nodes[0:1])

		for _, node := range nodes[0:1] {
			node.EpochStartTrigger.SetRoundsPerEpoch(roundsPerEpoch)
		}
	}

	defer func() {
		_ = advertiser.Close()
		for _, nodes := range nodesMap {
			for _, n := range nodes {
				_ = n.Node.Stop()
			}
		}
	}()

	round := uint64(1)
	roundDifference := 10
	nonce := uint64(1)

	randomness := generateInitialRandomness(uint32(nbShards))
	_, _, _, randomness = integrationTests.AllShardsProposeBlock(round, nonce, randomness, nodesMap)

	round += uint64(roundDifference)
	nonce++
	putCounter = 0
	_, _, _, _ = integrationTests.AllShardsProposeBlock(round, nonce, randomness, nodesMap)

	callsFromShardZero := 2
	callsForPreviousBlockComputation := 1

	assert.Equal(t, roundDifference+callsForPreviousBlockComputation+callsFromShardZero, putCounter)
}
