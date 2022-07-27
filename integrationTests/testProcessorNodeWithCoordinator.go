package integrationTests

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data/endProcess"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/hashing/blake2b"
	crypto "github.com/ElrondNetwork/elrond-go-crypto"
	"github.com/ElrondNetwork/elrond-go-crypto/signing"
	"github.com/ElrondNetwork/elrond-go-crypto/signing/ed25519"
	ed25519SingleSig "github.com/ElrondNetwork/elrond-go-crypto/signing/ed25519/singlesig"
	"github.com/ElrondNetwork/elrond-go-crypto/signing/mcl"
	multisig2 "github.com/ElrondNetwork/elrond-go-crypto/signing/mcl/multisig"
	"github.com/ElrondNetwork/elrond-go-crypto/signing/multisig"
	"github.com/ElrondNetwork/elrond-go/integrationTests/mock"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/sharding/nodesCoordinator"
	"github.com/ElrondNetwork/elrond-go/storage/lrucache"
	"github.com/ElrondNetwork/elrond-go/testscommon"
)

type nodeKeys struct {
	TxSignKeyGen     crypto.KeyGenerator
	TxSignSk         crypto.PrivateKey
	TxSignPk         crypto.PublicKey
	TxSignPkBytes    []byte
	BlockSignKeyGen  crypto.KeyGenerator
	BlockSignSk      crypto.PrivateKey
	BlockSignPk      crypto.PublicKey
	BlockSignPkBytes []byte
}

func pubKeysMapFromKeysMap(ncp map[uint32][]*nodeKeys) map[uint32][]string {
	keysMap := make(map[uint32][]string)

	for shardId, keys := range ncp {
		shardKeys := make([]string, len(keys))
		for i, nk := range keys {
			shardKeys[i] = string(nk.BlockSignPkBytes)
		}
		keysMap[shardId] = shardKeys
	}

	return keysMap
}

// CreateProcessorNodesWithNodesCoordinator creates a map of nodes with a valid nodes coordinator implementation
// keeping the consistency of generated keys
func CreateProcessorNodesWithNodesCoordinator(
	rewardsAddrsAssignments map[uint32][]uint32,
	shardConsensusGroupSize int,
	metaConsensusGroupSize int,
) (map[uint32][]*TestProcessorNode, uint32) {

	ncp, nbShards := createNodesCryptoParams(rewardsAddrsAssignments)
	cp := CreateCryptoParams(len(ncp[0]), len(ncp[core.MetachainShardId]), nbShards)
	pubKeys := PubKeysMapFromKeysMap(cp.Keys)
	validatorsMap := GenValidatorsFromPubKeys(pubKeys, nbShards)
	validatorsMapForNodesCoordinator, _ := nodesCoordinator.NodesInfoToValidators(validatorsMap)

	cpWaiting := CreateCryptoParams(1, 1, nbShards)
	pubKeysWaiting := PubKeysMapFromKeysMap(cpWaiting.Keys)
	waitingMap := GenValidatorsFromPubKeys(pubKeysWaiting, nbShards)
	waitingMapForNodesCoordinator, _ := nodesCoordinator.NodesInfoToValidators(waitingMap)

	nodesSetup := &mock.NodesSetupStub{InitialNodesInfoCalled: func() (m map[uint32][]nodesCoordinator.GenesisNodeInfoHandler, m2 map[uint32][]nodesCoordinator.GenesisNodeInfoHandler) {
		return validatorsMap, waitingMap
	}}

	ncp, numShards := createNodesCryptoParams(rewardsAddrsAssignments)

	completeNodesList := make([]Connectable, 0)
	nodesMap := make(map[uint32][]*TestProcessorNode)
	for shardId, validatorList := range validatorsMap {
		nodesList := make([]*TestProcessorNode, len(validatorList))
		for i, v := range validatorList {
			cache, _ := lrucache.NewCache(10000)
			argumentsNodesCoordinator := nodesCoordinator.ArgNodesCoordinator{
				ShardConsensusGroupSize: shardConsensusGroupSize,
				MetaConsensusGroupSize:  metaConsensusGroupSize,
				Marshalizer:             TestMarshalizer,
				Hasher:                  TestHasher,
				ShardIDAsObserver:       shardId,
				NbShards:                numShards,
				EligibleNodes:           validatorsMapForNodesCoordinator,
				WaitingNodes:            waitingMapForNodesCoordinator,
				SelfPublicKey:           v.PubKeyBytes(),
				ConsensusGroupCache:     cache,
				ShuffledOutHandler:      &mock.ShuffledOutHandlerStub{},
				ChanStopNode:            endProcess.GetDummyEndProcessChannel(),
				IsFullArchive:           false,
				EnableEpochsHandler:     &testscommon.EnableEpochsHandlerStub{},
			}

			nodesCoordinatorInstance, err := nodesCoordinator.NewIndexHashedNodesCoordinator(argumentsNodesCoordinator)
			if err != nil {
				fmt.Println("error creating node coordinator")
			}

			blsHasher, _ := blake2b.NewBlake2bWithSize(hashing.BlsHashSize)
			llsig := &multisig2.BlsMultiSigner{Hasher: blsHasher}

			pubKeysMap := pubKeysMapFromKeysMap(ncp)
			kp := ncp[shardId][i]
			multiSigner, err := multisig.NewBLSMultisig(
				llsig,
				pubKeysMap[shardId],
				kp.BlockSignSk,
				kp.BlockSignKeyGen,
				uint16(i),
			)
			if err != nil {
				fmt.Printf("error generating multisigner: %s\n", err)
				return nil, 0
			}

			ownAccount := &TestWalletAccount{
				SingleSigner:      createTestSingleSigner(),
				BlockSingleSigner: createTestSingleSigner(),
				SkTxSign:          kp.TxSignSk,
				PkTxSign:          kp.TxSignPk,
				PkTxSignBytes:     kp.TxSignPkBytes,
				KeygenTxSign:      kp.TxSignKeyGen,
				KeygenBlockSign:   kp.BlockSignKeyGen,
				Nonce:             0,
				Balance:           nil,
			}
			ownAccount.Address = kp.TxSignPkBytes

			nodesList[i] = NewTestProcessorNode(ArgTestProcessorNode{
				MaxShards:            numShards,
				NodeShardId:          shardId,
				TxSignPrivKeyShardId: shardId,
				NodeKeys: &TestKeyPair{
					Sk: kp.BlockSignSk,
					Pk: kp.BlockSignPk,
				},
				NodesSetup:       nodesSetup,
				NodesCoordinator: nodesCoordinatorInstance,
				MultiSigner:      multiSigner,
				OwnAccount:       ownAccount,
			})

			completeNodesList = append(completeNodesList, nodesList[i])
		}
		nodesMap[shardId] = nodesList
	}

	ConnectNodes(completeNodesList)

	return nodesMap, numShards
}

func createTestSingleSigner() crypto.SingleSigner {
	return &ed25519SingleSig.Ed25519Signer{}
}

func createNodesCryptoParams(rewardsAddrsAssignments map[uint32][]uint32) (map[uint32][]*nodeKeys, uint32) {
	numShards := uint32(0)
	suiteBlock := mcl.NewSuiteBLS12()
	suiteTx := ed25519.NewEd25519()

	blockSignKeyGen := signing.NewKeyGenerator(suiteBlock)
	txSignKeyGen := signing.NewKeyGenerator(suiteTx)

	// we need to first precompute the num shard ID
	for shardID := range rewardsAddrsAssignments {
		foundAHigherShardID := shardID != core.MetachainShardId && shardID > numShards
		if foundAHigherShardID {
			numShards = shardID
		}
	}
	// we need to increment this as the numShards is actually the max shard at this moment
	numShards++

	ncp := make(map[uint32][]*nodeKeys)
	for shardID, assignments := range rewardsAddrsAssignments {
		ncp[shardID] = createShardNodeKeys(blockSignKeyGen, txSignKeyGen, assignments, shardID, numShards)
	}

	return ncp, numShards
}

func createShardNodeKeys(
	blockSignKeyGen crypto.KeyGenerator,
	txSignKeyGen crypto.KeyGenerator,
	assignments []uint32,
	shardID uint32,
	numShards uint32,
) []*nodeKeys {
	shardCoordinator, _ := sharding.NewMultiShardCoordinator(numShards, shardID)
	keys := make([]*nodeKeys, len(assignments))
	for i, addrShardID := range assignments {
		keys[i] = &nodeKeys{
			BlockSignKeyGen: blockSignKeyGen,
			TxSignKeyGen:    txSignKeyGen,
		}
		keys[i].TxSignSk, keys[i].TxSignPk = generateSkAndPkInShard(keys[i].TxSignKeyGen, shardCoordinator, addrShardID)
		keys[i].BlockSignSk, keys[i].BlockSignPk = blockSignKeyGen.GeneratePair()

		keys[i].BlockSignPkBytes, _ = keys[i].BlockSignPk.ToByteArray()
		keys[i].TxSignPkBytes, _ = keys[i].TxSignPk.ToByteArray()
	}

	return keys
}

func generateSkAndPkInShard(
	keyGen crypto.KeyGenerator,
	shardCoordinator sharding.Coordinator,
	addrShardID uint32,
) (crypto.PrivateKey, crypto.PublicKey) {
	sk, pk := keyGen.GeneratePair()
	for {
		pkBytes, _ := pk.ToByteArray()
		if shardCoordinator.ComputeId(pkBytes) == addrShardID {
			break
		}
		sk, pk = keyGen.GeneratePair()
	}

	return sk, pk
}
