package transaction

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoutingOfTransactionsInShards(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 5
	nodesPerShard := 1
	numMetachainNodes := 1

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)
	mintValue := big.NewInt(1000000000000000000)
	integrationTests.MintAllNodes(nodes, mintValue)
	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Node.Stop()
		}
	}()

	for i := 0; i < numOfShards; i++ {
		txs := generateTransactionsInAllConfigurations(nodes, uint32(numOfShards))
		require.Equal(t, numOfShards*numOfShards, len(txs))

		dispatchNode := getNodeOnShard(uint32(i), nodes)

		_, err := dispatchNode.Node.SendBulkTransactions(txs)
		require.Nil(t, err)
	}

	fmt.Println("waiting for txs to be disseminated")
	time.Sleep(integrationTests.StepDelay)

	//expectedNumTxs is computed in the following manner:
	//- if the node sends all numOfShards*numOfShards txs, then it will have in the pool
	//  (numOfShards + numOfShards - 1 -> both sender and destination) txs related to its shard
	//- if the node will have to receive all those generated txs, it will receive only those txs
	//  where the node is the sender + the tx that originated from shard x and sender node was also on shard x,
	//  so numOfShards + 1 in total
	//- since all shards emmit numOfShards*numOfShards, expectedNumTxs is computed as:
	expectedNumTxs := (numOfShards + numOfShards - 1) + (numOfShards+1)*(numOfShards-1)
	checkTransactionsInPool(t, nodes, expectedNumTxs)
}

func generateTransactionsInAllConfigurations(nodes []*integrationTests.TestProcessorNode, numOfShards uint32) []*transaction.Transaction {
	txs := make([]*transaction.Transaction, 0, numOfShards*numOfShards)

	for i := uint32(0); i < numOfShards; i++ {
		for j := uint32(0); j < numOfShards; j++ {
			senderNode := getNodeOnShard(i, nodes)
			destNode := getNodeOnShard(j, nodes)

			tx := generateTx(senderNode.OwnAccount.SkTxSign, destNode.OwnAccount.PkTxSign, senderNode.OwnAccount.Nonce)
			senderNode.OwnAccount.Nonce++
			txs = append(txs, tx)
		}
	}

	return txs
}

func getNodeOnShard(shardId uint32, nodes []*integrationTests.TestProcessorNode) *integrationTests.TestProcessorNode {
	for _, n := range nodes {
		if n.ShardCoordinator.SelfId() == shardId {
			return n
		}
	}

	return nil
}

func checkTransactionsInPool(t *testing.T, nodes []*integrationTests.TestProcessorNode, numExpected int) {
	for _, n := range nodes {
		if n.ShardCoordinator.SelfId() == core.MetachainShardId {
			assert.Equal(t, int32(0), n.CounterTxRecv)
			continue
		}

		assert.Equal(t, int32(numExpected), n.CounterTxRecv)
		checkTransactionsInPoolAreForTheNode(t, n)
	}
}

func checkTransactionsInPoolAreForTheNode(t *testing.T, n *integrationTests.TestProcessorNode) {
	n.ReceivedTransactions.Range(func(key, value interface{}) bool {
		tx, ok := value.(*transaction.Transaction)
		if !ok {
			assert.Fail(t, "found a non transaction object, key: "+hex.EncodeToString(key.([]byte)))
			return false
		}

		senderBuff := tx.SndAddr
		recvBuff := tx.RcvAddr

		sender, _ := integrationTests.TestAddressConverter.CreateAddressFromPublicKeyBytes(senderBuff)
		recv, _ := integrationTests.TestAddressConverter.CreateAddressFromPublicKeyBytes(recvBuff)

		senderShardId := n.ShardCoordinator.ComputeId(sender)
		recvShardId := n.ShardCoordinator.ComputeId(recv)
		isForCurrentShard := senderShardId == n.ShardCoordinator.SelfId() || recvShardId == n.ShardCoordinator.SelfId()
		if !isForCurrentShard {
			assert.Fail(t, fmt.Sprintf("found a transaction wrongly dispatched, key: %s, tx: %v",
				hex.EncodeToString(key.([]byte)),
				tx,
			),
			)
			return false
		}

		return true
	})
}
