package interceptedBulkTx

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/stretchr/testify/assert"
)

var stepDelay = time.Second

func TestNode_GenerateSendInterceptBulkTransactionsWithMessenger(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	startingNonce := uint64(6)
	var nrOfShards uint32 = 1
	var shardID uint32 = 0
	var txSignPrivKeyShardId uint32 = 0
	nodeAddr := "0"

	n := integrationTests.NewTestProcessorNode(nrOfShards, shardID, txSignPrivKeyShardId, nodeAddr)

	defer func() {
		_ = n.Messenger.Close()
	}()

	_ = n.Messenger.Bootstrap()

	time.Sleep(integrationTests.P2pBootstrapDelay)

	//set the account's nonce to startingNonce
	_ = n.SetAccountNonce(startingNonce)
	noOfTx := 8000

	time.Sleep(stepDelay)

	wg := sync.WaitGroup{}
	wg.Add(noOfTx)

	chanDone := make(chan bool)

	go func() {
		wg.Wait()
		chanDone <- true
	}()

	mut := sync.Mutex{}
	txHashes := make([][]byte, 0)
	transactions := make([]data.TransactionHandler, 0)

	//wire up handler
	n.DataPool.Transactions().RegisterHandler(func(key []byte, value interface{}) {
		mut.Lock()
		defer mut.Unlock()

		txHashes = append(txHashes, key)

		dataStore := n.DataPool.Transactions().ShardDataStore(
			process.ShardCacherIdentifier(n.ShardCoordinator.SelfId(), n.ShardCoordinator.SelfId()),
		)
		val, _ := dataStore.Get(key)

		if val == nil {
			assert.Fail(t, fmt.Sprintf("key %s not in store?", base64.StdEncoding.EncodeToString(key)))
			return
		}

		transactions = append(transactions, val.(*transaction.Transaction))
		wg.Done()
	})

	shardId := uint32(0)
	senderPrivateKeys := []crypto.PrivateKey{n.OwnAccount.SkTxSign}
	mintingValue := big.NewInt(100000)
	integrationTests.CreateMintingForSenders([]*integrationTests.TestProcessorNode{n}, shardId, senderPrivateKeys, mintingValue)

	receiver := integrationTests.TestAddressPubkeyConverter.Encode(integrationTests.CreateRandomBytes(32))
	err := n.Node.GenerateAndSendBulkTransactions(
		receiver,
		big.NewInt(1),
		uint64(noOfTx),
		n.OwnAccount.SkTxSign,
		integrationTests.ChainID,
	)

	assert.Nil(t, err)

	select {
	case <-chanDone:
	case <-time.After(time.Second * 60):
		assert.Fail(t, "timeout")
		return
	}

	integrationTests.CheckTxPresentAndRightNonce(
		t,
		startingNonce,
		noOfTx,
		txHashes,
		transactions,
		n.DataPool.Transactions(),
		n.ShardCoordinator,
	)
}
