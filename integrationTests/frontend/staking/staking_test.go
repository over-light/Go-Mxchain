package staking

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core"
	ed25519SingleSig "github.com/ElrondNetwork/elrond-go/crypto/signing/ed25519/singlesig"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/mcl/singlesig"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/testscommon/txDataBuilder"
	"github.com/ElrondNetwork/elrond-go/vm"
	"github.com/ElrondNetwork/elrond-go/vm/systemSmartContracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var log = logger.GetOrCreate("integrationtests/frontend/staking")

func TestSignatureOnStaking(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	skHexBuff, pkString, err := core.LoadSkPkFromPemFile("./testdata/key.pem", 0)
	require.Nil(t, err)

	skBuff, err := hex.DecodeString(string(skHexBuff))
	require.Nil(t, err)

	skStaking, err := integrationTests.TestKeyGenForAccounts.PrivateKeyFromByteArray(skBuff)
	require.Nil(t, err)
	pkStaking := skStaking.GeneratePublic()
	pkBuff, err := pkStaking.ToByteArray()
	require.Nil(t, err)

	stakingWalletAccount := &integrationTests.TestWalletAccount{
		SingleSigner:      &ed25519SingleSig.Ed25519Signer{},
		BlockSingleSigner: &singlesig.BlsSingleSigner{},
		SkTxSign:          skStaking,
		PkTxSign:          pkStaking,
		PkTxSignBytes:     pkBuff,
		KeygenTxSign:      integrationTests.TestKeyGenForAccounts,
		PeerSigHandler:    nil,
		Address:           pkBuff,
		Nonce:             0,
		Balance:           big.NewInt(0),
	}

	log.Info("using tx sign pk for staking", "pk", pkString)

	frontendBLSPubkey, err := hex.DecodeString("309befb6387288380edda61ce174b12d42ad161d19361dfcf7e61e6a4e812caf07e45a5a1c5c1e6e1f2f4d84d794dc16d9c9db0088397d85002194b773c30a8b7839324b3b80d9b8510fe53385ba7b7383c96a4c07810db31d84b0feefafbd03")
	require.Nil(t, err)
	frontendHexSignature := "17b1f945404c0c98d2e69a576f3635f4ebe77cd396561566afb969333b0da053e7485b61ef10311f512e3ec2f351ee95"

	numOfShards := 2
	nodesPerShard := 2
	numMetachainNodes := 2

	nodes := integrationTests.CreateNodesWithBLSSigVerifier(
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
	integrationTests.MintAllPlayers(nodes, []*integrationTests.TestWalletAccount{stakingWalletAccount}, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	///////////------- send stake tx and check sender's balance
	var txData string
	genesisBlock := nodes[0].GenesisBlocks[core.MetachainShardId]
	metaBlock := genesisBlock.(*block.MetaBlock)
	nodePrice := big.NewInt(0).Set(metaBlock.EpochStart.Economics.NodePrice)
	oneEncoded := hex.EncodeToString(big.NewInt(1).Bytes())

	pubKey := hex.EncodeToString(frontendBLSPubkey)
	txData = "stake" + "@" + oneEncoded + "@" + pubKey + "@" + frontendHexSignature
	integrationTests.PlayerSendsTransaction(
		nodes,
		stakingWalletAccount,
		vm.ValidatorSCAddress,
		nodePrice,
		txData,
		integrationTests.MinTxGasLimit+uint64(len(txData))+1+core.MinMetaTxExtraGasCost,
	)

	time.Sleep(time.Second)

	nrRoundsToPropagateMultiShard := 10
	integrationTests.AddSelfNotarizedHeaderByMetachain(nodes)
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)

	time.Sleep(time.Second)

	testStakingWasDone(t, nodes, frontendBLSPubkey)
}

func TestValidatorToDelegationManagerWithNewContract(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	skHexBuff, pkString, err := core.LoadSkPkFromPemFile("./testdata/key.pem", 0)
	require.Nil(t, err)

	skBuff, err := hex.DecodeString(string(skHexBuff))
	require.Nil(t, err)

	skStaking, err := integrationTests.TestKeyGenForAccounts.PrivateKeyFromByteArray(skBuff)
	require.Nil(t, err)
	pkStaking := skStaking.GeneratePublic()
	pkBuff, err := pkStaking.ToByteArray()
	require.Nil(t, err)

	stakingWalletAccount := &integrationTests.TestWalletAccount{
		SingleSigner:      &ed25519SingleSig.Ed25519Signer{},
		BlockSingleSigner: &singlesig.BlsSingleSigner{},
		SkTxSign:          skStaking,
		PkTxSign:          pkStaking,
		PkTxSignBytes:     pkBuff,
		KeygenTxSign:      integrationTests.TestKeyGenForAccounts,
		PeerSigHandler:    nil,
		Address:           pkBuff,
		Nonce:             0,
		Balance:           big.NewInt(0),
	}

	numOfShards := 1
	nodesPerShard := 1
	numMetachainNodes := 1

	nodes := integrationTests.CreateNodesWithBLSSigVerifier(
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
	integrationTests.MintAllPlayers(nodes, []*integrationTests.TestWalletAccount{stakingWalletAccount}, initialVal)
	saveDelegationManagerConfig(nodes)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	///////////------- send stake tx and check sender's balance
	genesisBlock := nodes[0].GenesisBlocks[core.MetachainShardId]
	metaBlock := genesisBlock.(*block.MetaBlock)
	nodePrice := big.NewInt(0).Set(metaBlock.EpochStart.Economics.NodePrice)

	log.Info("using tx sign pk for staking", "pk", pkString)

	frontendBLSPubkey, err := hex.DecodeString("309befb6387288380edda61ce174b12d42ad161d19361dfcf7e61e6a4e812caf07e45a5a1c5c1e6e1f2f4d84d794dc16d9c9db0088397d85002194b773c30a8b7839324b3b80d9b8510fe53385ba7b7383c96a4c07810db31d84b0feefafbd03")
	require.Nil(t, err)
	frontendHexSignature := "17b1f945404c0c98d2e69a576f3635f4ebe77cd396561566afb969333b0da053e7485b61ef10311f512e3ec2f351ee95"

	nonce, round = generateSendAndWaitToExecuteStakeTransaction(
		t,
		nodes,
		stakingWalletAccount,
		idxProposers,
		nodePrice,
		frontendBLSPubkey,
		frontendHexSignature,
		nonce,
		round,
	)

	time.Sleep(time.Second)

	testStakingWasDone(t, nodes, frontendBLSPubkey)

	saveDelegationContractsList(nodes)

	nonce, round = generateSendAndWaitToExecuteTransaction(
		t,
		nodes,
		stakingWalletAccount,
		idxProposers,
		"makeNewContractFromValidatorData",
		big.NewInt(0),
		[]byte{10},
		nonce,
		round)

	time.Sleep(time.Second)
	testStakingWasDone(t, nodes, frontendBLSPubkey)
	scAddressBytes, _ := hex.DecodeString("0000000000000000000100000000000000000000000000000000000002ffffff")
	testBLSKeyOwnerIsAddress(t, nodes, scAddressBytes, frontendBLSPubkey)
}

func generateSendAndWaitToExecuteStakeTransaction(
	t *testing.T,
	nodes []*integrationTests.TestProcessorNode,
	stakingWalletAccount *integrationTests.TestWalletAccount,
	idxProposers []int,
	nodePrice *big.Int,
	frontendBLSPubkey []byte,
	frontendHexSignature string,
	nonce, round uint64,
) (uint64, uint64) {
	oneEncoded := hex.EncodeToString(big.NewInt(1).Bytes())
	pubKey := hex.EncodeToString(frontendBLSPubkey)
	txData := "stake" + "@" + oneEncoded + "@" + pubKey + "@" + frontendHexSignature
	integrationTests.PlayerSendsTransaction(
		nodes,
		stakingWalletAccount,
		vm.ValidatorSCAddress,
		nodePrice,
		txData,
		integrationTests.MinTxGasLimit+uint64(len(txData))+1+core.MinMetaTxExtraGasCost,
	)
	time.Sleep(time.Second)

	nrRoundsToPropagateMultiShard := 6
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)

	return nonce, round
}

func generateSendAndWaitToExecuteTransaction(
	t *testing.T,
	nodes []*integrationTests.TestProcessorNode,
	stakingWalletAccount *integrationTests.TestWalletAccount,
	idxProposers []int,
	function string,
	value *big.Int,
	serviceFee []byte,
	nonce, round uint64,
) (uint64, uint64) {
	maxDelegationCap := []byte{0}
	txData := txDataBuilder.NewBuilder().Clear().
		Func(function).
		Bytes(maxDelegationCap).
		Bytes(serviceFee).
		ToString()

	integrationTests.PlayerSendsTransaction(
		nodes,
		stakingWalletAccount,
		vm.DelegationManagerSCAddress,
		value,
		txData,
		integrationTests.MinTxGasLimit+uint64(len(txData))+1+core.MinMetaTxExtraGasCost,
	)

	time.Sleep(time.Second)

	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, 10, nonce, round, idxProposers)

	return nonce, round
}

func TestValidatorToDelegationManagerWithMerge(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	skHexBuff, pkString, err := core.LoadSkPkFromPemFile("./testdata/key.pem", 0)
	require.Nil(t, err)

	skBuff, err := hex.DecodeString(string(skHexBuff))
	require.Nil(t, err)

	skStaking, err := integrationTests.TestKeyGenForAccounts.PrivateKeyFromByteArray(skBuff)
	require.Nil(t, err)
	pkStaking := skStaking.GeneratePublic()
	pkBuff, err := pkStaking.ToByteArray()
	require.Nil(t, err)

	stakingWalletAccount := &integrationTests.TestWalletAccount{
		SingleSigner:      &ed25519SingleSig.Ed25519Signer{},
		BlockSingleSigner: &singlesig.BlsSingleSigner{},
		SkTxSign:          skStaking,
		PkTxSign:          pkStaking,
		PkTxSignBytes:     pkBuff,
		KeygenTxSign:      integrationTests.TestKeyGenForAccounts,
		PeerSigHandler:    nil,
		Address:           pkBuff,
		Nonce:             0,
		Balance:           big.NewInt(0),
	}

	numOfShards := 1
	nodesPerShard := 1
	numMetachainNodes := 1

	nodes := integrationTests.CreateNodesWithBLSSigVerifier(
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
	integrationTests.MintAllPlayers(nodes, []*integrationTests.TestWalletAccount{stakingWalletAccount}, initialVal)
	saveDelegationManagerConfig(nodes)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	///////////------- send stake tx and check sender's balance
	var txData string
	genesisBlock := nodes[0].GenesisBlocks[core.MetachainShardId]
	metaBlock := genesisBlock.(*block.MetaBlock)
	nodePrice := big.NewInt(0).Set(metaBlock.EpochStart.Economics.NodePrice)

	log.Info("using tx sign pk for staking", "pk", pkString)

	frontendBLSPubkey, err := hex.DecodeString("309befb6387288380edda61ce174b12d42ad161d19361dfcf7e61e6a4e812caf07e45a5a1c5c1e6e1f2f4d84d794dc16d9c9db0088397d85002194b773c30a8b7839324b3b80d9b8510fe53385ba7b7383c96a4c07810db31d84b0feefafbd03")
	require.Nil(t, err)
	frontendHexSignature := "17b1f945404c0c98d2e69a576f3635f4ebe77cd396561566afb969333b0da053e7485b61ef10311f512e3ec2f351ee95"

	nonce, round = generateSendAndWaitToExecuteStakeTransaction(
		t,
		nodes,
		stakingWalletAccount,
		idxProposers,
		nodePrice,
		frontendBLSPubkey,
		frontendHexSignature,
		nonce,
		round,
	)

	time.Sleep(time.Second)

	testStakingWasDone(t, nodes, frontendBLSPubkey)

	saveDelegationContractsList(nodes)

	nonce, round = generateSendAndWaitToExecuteTransaction(
		t,
		nodes,
		stakingWalletAccount,
		idxProposers,
		"createNewDelegationContract",
		big.NewInt(10000),
		[]byte{0},
		nonce,
		round,
	)

	scAddressBytes, _ := hex.DecodeString("0000000000000000000100000000000000000000000000000000000002ffffff")
	txData = txDataBuilder.NewBuilder().Clear().
		Func("mergeValidatorDataToContract").
		Bytes(scAddressBytes).
		ToString()
	integrationTests.PlayerSendsTransaction(
		nodes,
		stakingWalletAccount,
		vm.DelegationManagerSCAddress,
		big.NewInt(0),
		txData,
		integrationTests.MinTxGasLimit+uint64(len(txData))+1+core.MinMetaTxExtraGasCost,
	)

	time.Sleep(time.Second)

	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, 10, nonce, round, idxProposers)

	time.Sleep(time.Second)
	testStakingWasDone(t, nodes, frontendBLSPubkey)
	testBLSKeyOwnerIsAddress(t, nodes, scAddressBytes, frontendBLSPubkey)
}

func testBLSKeyOwnerIsAddress(t *testing.T, nodes []*integrationTests.TestProcessorNode, address []byte, blsKey []byte) {
	for _, n := range nodes {
		if n.ShardCoordinator.SelfId() != core.MetachainShardId {
			continue
		}

		acnt, _ := n.AccntState.GetExistingAccount(vm.StakingSCAddress)
		userAcc, _ := acnt.(state.UserAccountHandler)

		marshaledData, _ := userAcc.DataTrieTracker().RetrieveValue(blsKey)
		stakingData := &systemSmartContracts.StakedDataV2_0{}
		_ = integrationTests.TestMarshalizer.Unmarshal(stakingData, marshaledData)
		assert.Equal(t, stakingData.OwnerAddress, address)
	}
}

func testStakingWasDone(t *testing.T, nodes []*integrationTests.TestProcessorNode, blsKey []byte) {
	for _, n := range nodes {
		if n.ShardCoordinator.SelfId() == core.MetachainShardId {
			checkStakeOnNode(t, n, blsKey)
		}
	}
}

func checkStakeOnNode(t *testing.T, n *integrationTests.TestProcessorNode, blsKey []byte) {
	query := &process.SCQuery{
		ScAddress: vm.StakingSCAddress,
		FuncName:  "getBLSKeyStatus",
		Arguments: [][]byte{blsKey},
	}

	vmOutput, err := n.SCQueryService.ExecuteQuery(query)
	require.Nil(t, err)
	require.NotNil(t, vmOutput)
	require.Equal(t, 1, len(vmOutput.ReturnData))
	assert.Equal(t, []byte("staked"), vmOutput.ReturnData[0])
}

const delegationManagementKey = "delegationManagement"
const delegationContractsList = "delegationContracts"

func saveDelegationManagerConfig(nodes []*integrationTests.TestProcessorNode) {
	for _, node := range nodes {
		if node.ShardCoordinator.SelfId() != core.MetachainShardId {
			continue
		}

		acc, _ := node.AccntState.LoadAccount(vm.DelegationManagerSCAddress)
		userAcc, _ := acc.(state.UserAccountHandler)

		managementData := &systemSmartContracts.DelegationManagement{
			MinDeposit:          big.NewInt(100),
			LastAddress:         vm.FirstDelegationSCAddress,
			MinDelegationAmount: big.NewInt(1),
		}
		marshaledData, _ := integrationTests.TestMarshalizer.Marshal(managementData)
		_ = userAcc.DataTrieTracker().SaveKeyValue([]byte(delegationManagementKey), marshaledData)
		_ = node.AccntState.SaveAccount(userAcc)
		_, _ = node.AccntState.Commit()
	}
}

func saveDelegationContractsList(nodes []*integrationTests.TestProcessorNode) {
	for _, node := range nodes {
		if node.ShardCoordinator.SelfId() != core.MetachainShardId {
			continue
		}

		acc, _ := node.AccntState.LoadAccount(vm.DelegationManagerSCAddress)
		userAcc, _ := acc.(state.UserAccountHandler)

		managementData := &systemSmartContracts.DelegationContractList{
			Addresses: [][]byte{[]byte("addr")},
		}
		marshaledData, _ := integrationTests.TestMarshalizer.Marshal(managementData)
		_ = userAcc.DataTrieTracker().SaveKeyValue([]byte(delegationContractsList), marshaledData)
		_ = node.AccntState.SaveAccount(userAcc)
		_, _ = node.AccntState.Commit()
	}
}
