package polynetworkbridge

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/stretchr/testify/require"
)

func TestBridgeSetupAndBurn(t *testing.T) {
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

	ownerNode := nodes[0]
	shard := nodes[0:nodesPerShard]

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

	// tokenManagerPath := "../testdata/polynetworkbridge/esdt_token_manager.wasm"
	tokenManagerPath := "/home/camil.bancioiu/Work/Elrond/elrond-go/integrationTests/multiShard/smartContract/testdata/polynetworkbridge/esdt_token_manager.wasm"

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 2, nonce, round, idxProposers)

	blockChainHook := ownerNode.BlockchainHook
	scAddressBytes, _ := blockChainHook.NewAddress(
		ownerNode.OwnAccount.Address,
		ownerNode.OwnAccount.Nonce,
		factory.ArwenVirtualMachine,
	)

	scCode, err := ioutil.ReadFile(tokenManagerPath)
	if err != nil {
		panic(fmt.Sprintf("putDeploySCToDataPool(): %s", err))
	}

	scCodeString := hex.EncodeToString(scCode)
	scCodeMetadataString := "0000"

	deploymentData := scCodeString + "@" + hex.EncodeToString(factory.ArwenVirtualMachine) + "@" + scCodeMetadataString

	integrationTests.CreateAndSendTransaction(
		ownerNode,
		shard,
		big.NewInt(0),
		make([]byte, 32),
		deploymentData,
		100000,
	)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 1, nonce, round, idxProposers)

	txValue := big.NewInt(1000)
	txData := "performWrappedEgldIssue@05"
	integrationTests.CreateAndSendTransaction(
		ownerNode,
		shard,
		txValue,
		scAddressBytes,
		txData,
		integrationTests.AdditionalGasLimit,
	)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 6, nonce, round, idxProposers)

	scQuery := &process.SCQuery{
		CallerAddr: ownerNode.OwnAccount.Address,
		ScAddress:  scAddressBytes,
		FuncName:   "getWrappedEgldTokenIdentifier",
		Arguments:  [][]byte{},
	}
	vmOutput, err := ownerNode.SCQueryService.ExecuteQuery(scQuery)
	require.Nil(t, err)
	require.NotNil(t, vmOutput)
	require.NotZero(t, len(vmOutput.ReturnData[0]))

	tokenIdentifier := vmOutput.ReturnData[0]
	require.Equal(t, []byte("WEGLD"), tokenIdentifier[:5])

	txValue = big.NewInt(0)
	txData = "burnEsdtToken@" + hex.EncodeToString(tokenIdentifier) + "@" + "8AC7230489E80000"
	integrationTests.CreateAndSendTransaction(
		ownerNode,
		shard,
		txValue,
		scAddressBytes,
		txData,
		integrationTests.AdditionalGasLimit,
	)
	logger.SetLogLevel("*:TRACE")
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 6, nonce, round, idxProposers)
}
