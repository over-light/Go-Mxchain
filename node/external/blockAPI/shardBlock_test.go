package blockAPI

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/data/api"
	"github.com/ElrondNetwork/elrond-go-core/data/block"
	outportcore "github.com/ElrondNetwork/elrond-go-core/data/outport"
	"github.com/ElrondNetwork/elrond-go-core/data/transaction"
	"github.com/ElrondNetwork/elrond-go/common"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/node/mock"
	"github.com/ElrondNetwork/elrond-go/outport/process/alteredaccounts/shared"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/ElrondNetwork/elrond-go/testscommon/dblookupext"
	"github.com/ElrondNetwork/elrond-go/testscommon/genericMocks"
	"github.com/ElrondNetwork/elrond-go/testscommon/state"
	storageMocks "github.com/ElrondNetwork/elrond-go/testscommon/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createMockShardAPIProcessor(
	shardID uint32,
	blockHeaderHash []byte,
	storerMock *genericMocks.StorerMock,
	withHistory bool,
	withKey bool,
) *shardAPIBlockProcessor {
	return newShardApiBlockProcessor(&ArgAPIBlockProcessor{
		APITransactionHandler: &mock.TransactionAPIHandlerStub{},
		SelfShardID:           shardID,
		Marshalizer:           &mock.MarshalizerFake{},
		Store: &storageMocks.ChainStorerStub{
			GetStorerCalled: func(unitType dataRetriever.UnitType) (storage.Storer, error) {
				return storerMock, nil
			},
			GetCalled: func(unitType dataRetriever.UnitType, key []byte) ([]byte, error) {
				if withKey {
					return storerMock.Get(key)
				}
				return blockHeaderHash, nil
			},
		},
		Uint64ByteSliceConverter: mock.NewNonceHashConverterMock(),
		HistoryRepo: &dblookupext.HistoryRepositoryStub{
			GetEpochByHashCalled: func(hash []byte) (uint32, error) {
				return 1, nil
			},
			IsEnabledCalled: func() bool {
				return withHistory
			},
		},
		ReceiptsRepository:      &testscommon.ReceiptsRepositoryStub{},
		AddressPubkeyConverter:  &testscommon.PubkeyConverterMock{},
		AlteredAccountsProvider: &testscommon.AlteredAccountsProviderStub{},
		AccountsRepository:      &state.AccountsRepositoryStub{},
	}, nil)
}

func TestShardAPIBlockProcessor_GetBlockByHashInvalidHashShouldErr(t *testing.T) {
	t.Parallel()

	shardID := uint32(3)
	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")

	storerMock := genericMocks.NewStorerMock()

	shardAPIBlockProcessor := createMockShardAPIProcessor(
		shardID,
		headerHash,
		storerMock,
		true,
		false,
	)

	blk, err := shardAPIBlockProcessor.GetBlockByHash([]byte("invalidHash"), api.BlockQueryOptions{})
	assert.Nil(t, blk)
	assert.Error(t, err)
}

func TestShardAPIBlockProcessor_GetBlockByNonceInvalidNonceShouldErr(t *testing.T) {
	t.Parallel()

	shardID := uint32(3)
	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")

	storerMock := genericMocks.NewStorerMock()

	shardAPIBlockProcessor := createMockShardAPIProcessor(
		shardID,
		headerHash,
		storerMock,
		true,
		false,
	)

	blk, err := shardAPIBlockProcessor.GetBlockByNonce(100, api.BlockQueryOptions{})
	assert.Nil(t, blk)
	assert.Error(t, err)
}

func TestShardAPIBlockProcessor_GetBlockByRoundInvalidRoundShouldErr(t *testing.T) {
	t.Parallel()

	shardID := uint32(3)
	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")

	storerMock := genericMocks.NewStorerMock()

	shardAPIBlockProcessor := createMockShardAPIProcessor(
		shardID,
		headerHash,
		storerMock,
		true,
		true,
	)

	blk, err := shardAPIBlockProcessor.GetBlockByRound(100, api.BlockQueryOptions{})
	assert.Nil(t, blk)
	assert.Error(t, err)
}

func TestShardAPIBlockProcessor_GetBlockByHashFromNormalNode(t *testing.T) {
	t.Parallel()

	nonce := uint64(1)
	round := uint64(2)
	epoch := uint32(1)
	shardID := uint32(3)
	miniblockHeader := []byte("miniBlockHash")
	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")

	storerMock := genericMocks.NewStorerMock()
	uint64Converter := mock.NewNonceHashConverterMock()

	shardAPIBlockProcessor := createMockShardAPIProcessor(
		shardID,
		headerHash,
		storerMock,
		false,
		true,
	)

	header := &block.Header{
		Nonce:   nonce,
		Round:   round,
		ShardID: shardID,
		Epoch:   epoch,
		MiniBlockHeaders: []block.MiniBlockHeader{
			{Hash: miniblockHeader, TxCount: 1},
		},
		AccumulatedFees: big.NewInt(0),
		DeveloperFees:   big.NewInt(0),
	}
	headerBytes, _ := json.Marshal(header)
	_ = storerMock.Put(headerHash, headerBytes)

	nonceBytes := uint64Converter.ToByteSlice(nonce)
	_ = storerMock.Put(nonceBytes, headerHash)

	expectedBlock := &api.Block{
		Nonce:  nonce,
		Round:  round,
		Shard:  shardID,
		Epoch:  epoch,
		Hash:   hex.EncodeToString(headerHash),
		NumTxs: 1,
		MiniBlocks: []*api.MiniBlock{
			{
				Hash:                    hex.EncodeToString(miniblockHeader),
				Type:                    block.TxBlock.String(),
				ProcessingType:          block.Normal.String(),
				ConstructionState:       block.Final.String(),
				IndexOfFirstTxProcessed: 0,
				IndexOfLastTxProcessed:  0,
			},
		},
		AccumulatedFees: "0",
		DeveloperFees:   "0",
		Status:          BlockStatusOnChain,
	}

	blk, err := shardAPIBlockProcessor.GetBlockByHash(headerHash, api.BlockQueryOptions{})
	assert.Nil(t, err)
	assert.Equal(t, expectedBlock, blk)
}

func TestShardAPIBlockProcessor_GetBlockByNonceFromHistoryNode(t *testing.T) {
	t.Parallel()

	nonce := uint64(1)
	round := uint64(2)
	epoch := uint32(1)
	shardID := uint32(3)
	miniblockHeader := []byte("miniBlockHash")
	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")

	storerMock := genericMocks.NewStorerMockWithEpoch(epoch)

	shardAPIBlockProcessor := createMockShardAPIProcessor(
		shardID,
		headerHash,
		storerMock,
		true,
		false,
	)

	header := &block.Header{
		Nonce:   nonce,
		Round:   round,
		ShardID: shardID,
		Epoch:   epoch,
		MiniBlockHeaders: []block.MiniBlockHeader{
			{Hash: miniblockHeader, TxCount: 1},
		},
		AccumulatedFees: big.NewInt(100),
		DeveloperFees:   big.NewInt(50),
	}
	headerBytes, _ := json.Marshal(header)
	_ = storerMock.Put(headerHash, headerBytes)

	expectedBlock := &api.Block{
		Nonce:  nonce,
		Round:  round,
		Shard:  shardID,
		Epoch:  epoch,
		Hash:   hex.EncodeToString(headerHash),
		NumTxs: 1,
		MiniBlocks: []*api.MiniBlock{
			{
				Hash:                    hex.EncodeToString(miniblockHeader),
				Type:                    block.TxBlock.String(),
				ProcessingType:          block.Normal.String(),
				ConstructionState:       block.Final.String(),
				IndexOfFirstTxProcessed: 0,
				IndexOfLastTxProcessed:  0,
			},
		},
		AccumulatedFees: "100",
		DeveloperFees:   "50",
		Status:          BlockStatusOnChain,
	}

	blk, err := shardAPIBlockProcessor.GetBlockByNonce(1, api.BlockQueryOptions{})
	assert.Nil(t, err)
	assert.Equal(t, expectedBlock, blk)
}

func TestShardAPIBlockProcessor_GetBlockByRoundFromStorer(t *testing.T) {
	t.Parallel()

	nonce := uint64(1)
	round := uint64(2)
	epoch := uint32(1)
	shardID := uint32(3)
	miniblockHeader := []byte("miniBlockHash")
	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")

	storerMock := genericMocks.NewStorerMockWithEpoch(epoch)

	shardAPIBlockProcessor := createMockShardAPIProcessor(
		shardID,
		headerHash,
		storerMock,
		true,
		true,
	)

	header := &block.Header{
		Nonce:   nonce,
		Round:   round,
		ShardID: shardID,
		Epoch:   epoch,
		MiniBlockHeaders: []block.MiniBlockHeader{
			{Hash: miniblockHeader, TxCount: 1},
		},
		AccumulatedFees: big.NewInt(100),
		DeveloperFees:   big.NewInt(50),
	}
	headerBytes, _ := json.Marshal(header)
	_ = storerMock.Put(headerHash, headerBytes)

	uint64Converter := shardAPIBlockProcessor.uint64ByteSliceConverter
	roundBytes := uint64Converter.ToByteSlice(round)
	nonceBytes := uint64Converter.ToByteSlice(nonce)
	_ = storerMock.Put(roundBytes, headerHash)
	_ = storerMock.Put(nonceBytes, headerHash)

	expectedBlock := &api.Block{
		Nonce:  nonce,
		Round:  round,
		Shard:  shardID,
		Epoch:  epoch,
		Hash:   hex.EncodeToString(headerHash),
		NumTxs: 1,
		MiniBlocks: []*api.MiniBlock{
			{
				Hash:                    hex.EncodeToString(miniblockHeader),
				Type:                    block.TxBlock.String(),
				ProcessingType:          block.Normal.String(),
				ConstructionState:       block.Final.String(),
				IndexOfFirstTxProcessed: 0,
				IndexOfLastTxProcessed:  0,
			},
		},
		AccumulatedFees: "100",
		DeveloperFees:   "50",
		Status:          BlockStatusOnChain,
	}

	blk, err := shardAPIBlockProcessor.GetBlockByRound(round, api.BlockQueryOptions{})
	assert.Nil(t, err)
	assert.Equal(t, expectedBlock, blk)
}

func TestShardAPIBlockProcessor_GetBlockByHashFromHistoryNodeStatusReverted(t *testing.T) {
	t.Parallel()

	nonce := uint64(1)
	round := uint64(2)
	epoch := uint32(1)
	shardID := uint32(3)
	miniblockHeader := []byte("miniBlockHash")
	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")

	storerMock := genericMocks.NewStorerMockWithEpoch(1)
	uint64Converter := mock.NewNonceHashConverterMock()

	shardAPIBlockProcessor := createMockShardAPIProcessor(
		shardID,
		headerHash,
		storerMock,
		true,
		true,
	)

	header := &block.Header{
		Nonce:   nonce,
		Round:   round,
		ShardID: shardID,
		Epoch:   epoch,
		MiniBlockHeaders: []block.MiniBlockHeader{
			{Hash: miniblockHeader, TxCount: 1},
		},
		AccumulatedFees: big.NewInt(100),
		DeveloperFees:   big.NewInt(50),
	}
	headerBytes, _ := json.Marshal(header)
	_ = storerMock.Put(headerHash, headerBytes)

	nonceBytes := uint64Converter.ToByteSlice(nonce)
	correctHash := []byte("correct-hash")
	_ = storerMock.Put(nonceBytes, correctHash)

	expectedBlock := &api.Block{
		Nonce:  nonce,
		Round:  round,
		Shard:  shardID,
		Epoch:  epoch,
		Hash:   hex.EncodeToString(headerHash),
		NumTxs: 1,
		MiniBlocks: []*api.MiniBlock{
			{
				Hash:                    hex.EncodeToString(miniblockHeader),
				Type:                    block.TxBlock.String(),
				ProcessingType:          block.Normal.String(),
				ConstructionState:       block.Final.String(),
				IndexOfFirstTxProcessed: 0,
				IndexOfLastTxProcessed:  0,
			},
		},
		AccumulatedFees: "100",
		DeveloperFees:   "50",
		Status:          BlockStatusReverted,
	}

	blk, err := shardAPIBlockProcessor.GetBlockByHash(headerHash, api.BlockQueryOptions{})
	assert.Nil(t, err)
	assert.Equal(t, expectedBlock, blk)
}

func TestShardAPIBlockProcessor_GetAlteredAccountsForBlock(t *testing.T) {
	t.Parallel()

	t.Run("header not found in storage - should err", func(t *testing.T) {
		t.Parallel()

		headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")

		storerMock := genericMocks.NewStorerMockWithEpoch(1)
		metaAPIBlockProc := createMockShardAPIProcessor(
			0,
			headerHash,
			storerMock,
			true,
			true,
		)

		res, err := metaAPIBlockProc.GetAlteredAccountsForBlock(api.GetAlteredAccountsForBlockOptions{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
		require.Nil(t, res)
	})

	t.Run("get altered account by block hash - should work", func(t *testing.T) {
		t.Parallel()

		marshaller := &testscommon.MarshalizerMock{}
		headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")
		mbHash := []byte("mb-hash")
		txHash0, txHash1 := []byte("tx-hash-0"), []byte("tx-hash-1")

		mbhReserved := block.MiniBlockHeaderReserved{}

		mbhReserved.IndexOfLastTxProcessed = 1
		reserved, _ := mbhReserved.Marshal()

		metaBlock := &block.Header{
			Nonce: 37,
			Epoch: 1,
			MiniBlockHeaders: []block.MiniBlockHeader{
				{
					Hash:     mbHash,
					Reserved: reserved,
				},
			},
		}
		miniBlock := &block.MiniBlock{
			TxHashes: [][]byte{txHash0, txHash1},
		}
		tx0 := &transaction.Transaction{
			SndAddr: []byte("addr0"),
			RcvAddr: []byte("addr1"),
		}
		tx1 := &transaction.Transaction{
			SndAddr: []byte("addr2"),
			RcvAddr: []byte("addr3"),
		}
		miniBlockBytes, _ := marshaller.Marshal(miniBlock)
		metaBlockBytes, _ := marshaller.Marshal(metaBlock)
		tx0Bytes, _ := marshaller.Marshal(tx0)
		tx1Bytes, _ := marshaller.Marshal(tx1)

		storerMock := genericMocks.NewStorerMockWithEpoch(1)
		_ = storerMock.Put(headerHash, metaBlockBytes)
		_ = storerMock.Put(mbHash, miniBlockBytes)
		_ = storerMock.Put(txHash0, tx0Bytes)
		_ = storerMock.Put(txHash1, tx1Bytes)

		metaAPIBlockProc := createMockShardAPIProcessor(
			0,
			headerHash,
			storerMock,
			true,
			true,
		)

		metaAPIBlockProc.apiTransactionHandler = &mock.TransactionAPIHandlerStub{
			UnmarshalTransactionCalled: func(txBytes []byte, _ transaction.TxType) (*transaction.ApiTransactionResult, error) {
				var tx transaction.Transaction
				_ = marshaller.Unmarshal(&tx, txBytes)

				return &transaction.ApiTransactionResult{
					Type:     "normal",
					Sender:   hex.EncodeToString(tx.SndAddr),
					Receiver: hex.EncodeToString(tx.RcvAddr),
				}, nil
			},
		}
		metaAPIBlockProc.txStatusComputer = &mock.StatusComputerStub{}

		metaAPIBlockProc.logsFacade = &testscommon.LogsFacadeStub{
			IncludeLogsInTransactionsCalled: func(_ []*transaction.ApiTransactionResult, _ [][]byte, _ uint32) error {
				return nil
			},
		}
		metaAPIBlockProc.alteredAccountsProvider = &testscommon.AlteredAccountsProviderStub{
			ExtractAlteredAccountsFromPoolCalled: func(txPool *outportcore.Pool, options shared.AlteredAccountsOptions) (map[string]*outportcore.AlteredAccount, error) {
				retMap := map[string]*outportcore.AlteredAccount{}
				for _, tx := range txPool.Txs {
					retMap[string(tx.GetSndAddr())] = &outportcore.AlteredAccount{
						Address: string(tx.GetSndAddr()),
						Balance: "10",
					}
				}

				return retMap, nil
			},
		}

		res, err := metaAPIBlockProc.GetAlteredAccountsForBlock(api.GetAlteredAccountsForBlockOptions{
			GetBlockParameters: api.GetBlockParameters{
				RequestType: api.BlockFetchTypeByHash,
				Hash:        hex.EncodeToString(headerHash),
			},
		})
		require.NoError(t, err)
		require.True(t, areAlteredAccountsResponsesTheSame(&common.AlteredAccountsForBlockAPIResponse{
			Accounts: []*common.AlteredAccountAPIResponse{
				{
					Address: "addr0",
					Balance: "10",
				},
				{
					Address: "addr2",
					Balance: "10",
				},
			},
		}, res))
	})
}

func areAlteredAccountsResponsesTheSame(first *common.AlteredAccountsForBlockAPIResponse, second *common.AlteredAccountsForBlockAPIResponse) bool {
	if len(first.Accounts) != len(second.Accounts) {
		return false
	}

	for _, firstAcc := range first.Accounts {
		found := false
		for _, secondAcc := range second.Accounts {
			if firstAcc.Address == secondAcc.Address && firstAcc.Balance == secondAcc.Balance {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}
