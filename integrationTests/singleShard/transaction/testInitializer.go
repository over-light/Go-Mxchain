package transaction

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core/partitioning"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/crypto/signing"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/kyber"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/kyber/singlesig"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/multisig"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/blockchain"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/state/addressConverters"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/data/trie"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters/uint64ByteSlice"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/dataPool"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/factory/containers"
	factoryDataRetriever "github.com/ElrondNetwork/elrond-go/dataRetriever/factory/shard"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/shardedData"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/hashing/sha256"
	"github.com/ElrondNetwork/elrond-go/integrationTests/mock"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/node"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/p2p/libp2p"
	"github.com/ElrondNetwork/elrond-go/p2p/libp2p/discovery"
	"github.com/ElrondNetwork/elrond-go/p2p/loadBalancer"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/factory/shard"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/memorydb"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/btcsuite/btcd/btcec"
	libp2pCrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/stretchr/testify/assert"
)

var testMarshalizer = &marshal.JsonMarshalizer{}
var testHasher = sha256.Sha256{}

func createTestBlockChain() data.ChainHandler {
	cfgCache := storageUnit.CacheConfig{Size: 100, Type: storageUnit.LRUCache}
	badBlockCache, _ := storageUnit.NewCache(cfgCache.Type, cfgCache.Size, cfgCache.Shards)
	blockChain, _ := blockchain.NewBlockChain(
		badBlockCache,
	)

	return blockChain
}

func createMemUnit() storage.Storer {
	cache, _ := storageUnit.NewCache(storageUnit.LRUCache, 10, 1)
	persist, _ := memorydb.New()
	unit, _ := storageUnit.NewStorageUnit(cache, persist)

	return unit
}

func createTestStore() dataRetriever.StorageService {
	store := dataRetriever.NewChainStorer()
	store.AddStorer(dataRetriever.TransactionUnit, createMemUnit())
	store.AddStorer(dataRetriever.MiniBlockUnit, createMemUnit())
	store.AddStorer(dataRetriever.MetaBlockUnit, createMemUnit())
	store.AddStorer(dataRetriever.PeerChangesUnit, createMemUnit())
	store.AddStorer(dataRetriever.BlockHeaderUnit, createMemUnit())
	store.AddStorer(dataRetriever.UnsignedTransactionUnit, createMemUnit())
	store.AddStorer(dataRetriever.ShardHdrNonceHashDataUnit, createMemUnit())
	store.AddStorer(dataRetriever.MetaHdrNonceHashDataUnit, createMemUnit())

	return store
}

func createTestDataPool() dataRetriever.PoolsHolder {
	txPool, _ := shardedData.NewShardedData(storageUnit.CacheConfig{Size: 100000, Type: storageUnit.LRUCache})
	uTxPool, _ := shardedData.NewShardedData(storageUnit.CacheConfig{Size: 100000, Type: storageUnit.LRUCache})
	cacherCfg := storageUnit.CacheConfig{Size: 100, Type: storageUnit.LRUCache}
	hdrPool, _ := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)

	cacherCfg = storageUnit.CacheConfig{Size: 100000, Type: storageUnit.LRUCache}
	hdrNoncesCacher, _ := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)
	hdrNonces, _ := dataPool.NewNonceSyncMapCacher(hdrNoncesCacher, uint64ByteSlice.NewBigEndianConverter())

	cacherCfg = storageUnit.CacheConfig{Size: 100000, Type: storageUnit.LRUCache}
	txBlockBody, _ := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)

	cacherCfg = storageUnit.CacheConfig{Size: 100000, Type: storageUnit.LRUCache}
	peerChangeBlockBody, _ := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)

	cacherCfg = storageUnit.CacheConfig{Size: 100000, Type: storageUnit.LRUCache}
	metaBlocks, _ := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)

	cacherCfg = storageUnit.CacheConfig{Size: 10, Type: storageUnit.LRUCache}

	dPool, _ := dataPool.NewShardedDataPool(
		txPool,
		uTxPool,
		hdrPool,
		hdrNonces,
		txBlockBody,
		peerChangeBlockBody,
		metaBlocks,
	)

	return dPool
}

func createDummyHexAddress(chars int) string {
	if chars < 1 {
		return ""
	}

	addr := make([]byte, chars/2)
	_, _ = rand.Reader.Read(addr)

	return hex.EncodeToString(addr)
}

func createAccountsDB() *state.AccountsDB {
	marsh := &marshal.JsonMarshalizer{}
	hasher := sha256.Sha256{}
	store := createMemUnit()

	tr, _ := trie.NewTrie(store, marsh, hasher)
	adb, _ := state.NewAccountsDB(tr, sha256.Sha256{}, marsh, &mock.AccountsFactoryStub{
		CreateAccountCalled: func(address state.AddressContainer, tracker state.AccountTracker) (wrapper state.AccountHandler, e error) {
			return state.NewAccount(address, tracker)
		},
	})

	return adb
}

func createMultiSigner(
	privateKey crypto.PrivateKey,
	publicKey crypto.PublicKey,
	keyGen crypto.KeyGenerator,
	hasher hashing.Hasher,
) (crypto.MultiSigner, error) {

	publicKeys := make([]string, 1)
	pubKey, _ := publicKey.ToByteArray()
	publicKeys[0] = string(pubKey)
	multiSigner, err := multisig.NewBelNevMultisig(hasher, publicKeys, privateKey, keyGen, 0)

	return multiSigner, err
}

func createNetNode(
	dPool dataRetriever.PoolsHolder,
	accntAdapter state.AccountsAdapter,
	shardCoordinator sharding.Coordinator,
	nodesCoordinator sharding.NodesCoordinator,
) (
	*node.Node,
	p2p.Messenger,
	crypto.PrivateKey,
	dataRetriever.ResolversFinder) {

	messenger := createMessenger(context.Background())

	addrConverter, _ := addressConverters.NewPlainAddressConverter(32, "0x")

	suite := kyber.NewBlakeSHA256Ed25519()
	singleSigner := &singlesig.SchnorrSigner{}
	keyGen := signing.NewKeyGenerator(suite)
	sk, pk := keyGen.GeneratePair()
	multiSigner, _ := createMultiSigner(sk, pk, keyGen, testHasher)
	blkc := createTestBlockChain()
	store := createTestStore()
	uint64Converter := uint64ByteSlice.NewBigEndianConverter()
	dataPacker, _ := partitioning.NewSizeDataPacker(testMarshalizer)

	interceptorContainerFactory, _ := shard.NewInterceptorsContainerFactory(
		shardCoordinator,
		nodesCoordinator,
		messenger,
		store,
		testMarshalizer,
		testHasher,
		keyGen,
		singleSigner,
		multiSigner,
		dPool,
		addrConverter,
	)
	interceptorsContainer, _ := interceptorContainerFactory.Create()

	resolversContainerFactory, _ := factoryDataRetriever.NewResolversContainerFactory(
		shardCoordinator,
		messenger,
		store,
		testMarshalizer,
		dPool,
		uint64Converter,
		dataPacker,
	)
	resolversContainer, _ := resolversContainerFactory.Create()
	resolversFinder, _ := containers.NewResolversFinder(resolversContainer, shardCoordinator)

	n, _ := node.NewNode(
		node.WithMessenger(messenger),
		node.WithMarshalizer(testMarshalizer),
		node.WithHasher(testHasher),
		node.WithDataPool(dPool),
		node.WithAddressConverter(addrConverter),
		node.WithAccountsAdapter(accntAdapter),
		node.WithKeyGen(keyGen),
		node.WithShardCoordinator(shardCoordinator),
		node.WithBlockChain(blkc),
		node.WithUint64ByteSliceConverter(uint64Converter),
		node.WithMultiSigner(multiSigner),
		node.WithSingleSigner(singleSigner),
		node.WithTxSignPrivKey(sk),
		node.WithTxSignPubKey(pk),
		node.WithInterceptorsContainer(interceptorsContainer),
		node.WithResolversFinder(resolversFinder),
		node.WithDataStore(store),
		node.WithTxSingleSigner(singleSigner),
		node.WithTxStorageSize(100000),
	)

	return n, messenger, sk, resolversFinder
}

func createMessenger(ctx context.Context) p2p.Messenger {
	prvKey, _ := ecdsa.GenerateKey(btcec.S256(), rand.Reader)
	sk := (*libp2pCrypto.Secp256k1PrivateKey)(prvKey)

	libP2PMes, err := libp2p.NewNetworkMessengerOnFreePort(
		ctx,
		sk,
		nil,
		loadBalancer.NewOutgoingChannelLoadBalancer(),
		discovery.NewNullDiscoverer())

	if err != nil {
		fmt.Println(err.Error())
	}

	return libP2PMes
}

func getConnectableAddress(mes p2p.Messenger) string {
	for _, addr := range mes.Addresses() {
		if strings.Contains(addr, "circuit") || strings.Contains(addr, "169.254") {
			continue
		}

		return addr
	}

	return ""
}

func checkResults(
	t *testing.T,
	startingNonce uint64,
	noOfTxs int,
	txHashes [][]byte,
	txs []data.TransactionHandler,
	cache dataRetriever.ShardedDataCacherNotifier,
	shardCoordinator sharding.Coordinator,
) {

	if noOfTxs != len(txHashes) {
		for i := startingNonce; i < startingNonce+uint64(noOfTxs); i++ {
			found := false

			for _, txHandler := range txs {
				nonce := extractUint64ValueFromTxHandler(txHandler)
				if nonce == i {
					found = true
					break
				}
			}

			if !found {
				fmt.Printf("unsigned tx with nonce %d is missing\n", i)
			}
		}
		assert.Fail(t, fmt.Sprintf("should have been %d, got %d", noOfTxs, len(txHashes)))

		return
	}

	bitmap := make([]bool, noOfTxs+int(startingNonce))
	//set for each nonce from found tx a true flag in bitmap
	for i := 0; i < noOfTxs; i++ {
		selfId := shardCoordinator.SelfId()
		shardDataStore := cache.ShardDataStore(process.ShardCacherIdentifier(selfId, selfId))
		val, _ := shardDataStore.Get(txHashes[i])
		if val == nil {
			continue
		}

		nonce := extractUint64ValueFromTxHandler(val.(data.TransactionHandler))
		bitmap[nonce] = true
	}

	//for the first startingNonce values, the bitmap should be false
	//for the rest, true
	for i := 0; i < noOfTxs+int(startingNonce); i++ {
		if i < int(startingNonce) {
			assert.False(t, bitmap[i])
			continue
		}

		assert.True(t, bitmap[i])
	}
}

func extractUint64ValueFromTxHandler(txHandler data.TransactionHandler) uint64 {
	tx, ok := txHandler.(*transaction.Transaction)
	if ok {
		return tx.Nonce
	}

	buff, _ := hex.DecodeString(txHandler.GetData())
	return binary.BigEndian.Uint64(buff)
}
