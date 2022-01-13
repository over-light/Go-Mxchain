package alteredaccounts

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-go-core/data/indexer"
	"github.com/ElrondNetwork/elrond-go-core/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go-core/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go-core/data/transaction"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/ElrondNetwork/elrond-go/testscommon/state"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/require"
)

func TestNewAlteredAccountsProvider(t *testing.T) {
	t.Parallel()

	t.Run("nil shard coordinator", func(t *testing.T) {
		t.Parallel()

		args := getMockArgs()
		args.ShardCoordinator = nil

		aap, err := NewAlteredAccountsProvider(args)
		require.Nil(t, aap)
		require.Equal(t, errNilShardCoordinator, err)
	})

	t.Run("nil address converter", func(t *testing.T) {
		t.Parallel()

		args := getMockArgs()
		args.AddressConverter = nil

		aap, err := NewAlteredAccountsProvider(args)
		require.Nil(t, aap)
		require.Equal(t, errNilPubKeyConverter, err)
	})

	t.Run("nil accounts adapter", func(t *testing.T) {
		t.Parallel()

		args := getMockArgs()
		args.AccountsDB = nil

		aap, err := NewAlteredAccountsProvider(args)
		require.Nil(t, aap)
		require.Equal(t, errNilAccountsDB, err)
	})

	t.Run("nil marshalizer", func(t *testing.T) {
		t.Parallel()

		args := getMockArgs()
		args.Marshalizer = nil

		aap, err := NewAlteredAccountsProvider(args)
		require.Nil(t, aap)
		require.Equal(t, errNilMarshalizer, err)
	})

	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		args := getMockArgs()
		aap, err := NewAlteredAccountsProvider(args)

		require.NotNil(t, aap)
		require.NoError(t, err)
	})
}

func TestAlteredAccountsProvider_ExtractAlteredAccountsFromPool(t *testing.T) {
	t.Parallel()

	t.Run("no transaction, should return empty map", testExtractAlteredAccountsFromPoolNoTransaction)
	t.Run("should return sender shard accounts", testExtractAlteredAccountsFromPoolSenderShard)
	t.Run("should return receiver shard accounts", testExtractAlteredAccountsFromPoolReceiverShard)
	t.Run("should return all addresses in self shard", testExtractAlteredAccountsFromPoolBothSenderAndReceiverShards)
	t.Run("should return addresses from scrs, invalid and rewards", testExtractAlteredAccountsFromPoolScrsInvalidRewards)
	t.Run("should check data from trie", testExtractAlteredAccountsFromPoolTrieDataChecks)
	t.Run("should include esdt data", testExtractAlteredAccountsFromPoolShouldIncludeESDT)
	t.Run("should include nft data", testExtractAlteredAccountsFromPoolShouldIncludeNFT)
	t.Run("should include receiver from tokens logs", testExtractAlteredAccountsFromPoolShouldIncludeDestinationFromTokensLogsTopics)
	t.Run("should work when an address has balance changes, esdt and nft", testExtractAlteredAccountsFromPoolAddressHasBalanceChangeEsdtAndfNft)
	t.Run("should work when an address has multiple nfts with different nonces", testExtractAlteredAccountsFromPoolAddressHasMultipleNfts)
}

func testExtractAlteredAccountsFromPoolNoTransaction(t *testing.T) {
	t.Parallel()

	args := getMockArgs()
	aap, _ := NewAlteredAccountsProvider(args)

	res, err := aap.ExtractAlteredAccountsFromPool(&indexer.Pool{})
	require.NoError(t, err)
	require.Empty(t, res)
}

func testExtractAlteredAccountsFromPoolSenderShard(t *testing.T) {
	t.Parallel()

	args := getMockArgs()
	args.ShardCoordinator = &testscommon.ShardsCoordinatorMock{
		ComputeIdCalled: func(address []byte) uint32 {
			if strings.HasPrefix(string(address), "sender") {
				return 0
			}

			return 1
		},
		SelfIDCalled: func() uint32 {
			return 0
		},
	}
	aap, _ := NewAlteredAccountsProvider(args)

	res, err := aap.ExtractAlteredAccountsFromPool(&indexer.Pool{
		Txs: map[string]data.TransactionHandler{
			"hash0": &transaction.Transaction{
				SndAddr: []byte("sender shard - tx0"),
				RcvAddr: []byte("receiver shard - tx0"),
			},
			"hash1": &transaction.Transaction{
				SndAddr: []byte("sender shard - tx1"),
				RcvAddr: []byte("receiver shard - tx1"),
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(res))

	for key := range res {
		decodedKey, _ := args.AddressConverter.Decode(key)
		require.True(t, strings.HasPrefix(string(decodedKey), "sender"))
	}
}

func testExtractAlteredAccountsFromPoolReceiverShard(t *testing.T) {
	t.Parallel()

	args := getMockArgs()
	args.ShardCoordinator = &testscommon.ShardsCoordinatorMock{
		ComputeIdCalled: func(address []byte) uint32 {
			if strings.HasPrefix(string(address), "sender") {
				return 0
			}

			return 1
		},
		SelfIDCalled: func() uint32 {
			return 1
		},
	}
	aap, _ := NewAlteredAccountsProvider(args)

	res, err := aap.ExtractAlteredAccountsFromPool(&indexer.Pool{
		Txs: map[string]data.TransactionHandler{
			"hash0": &transaction.Transaction{
				SndAddr: []byte("sender shard - tx0"),
				RcvAddr: []byte("receiver shard - tx0"),
			},
			"hash1": &transaction.Transaction{
				SndAddr: []byte("sender shard - tx1"),
				RcvAddr: []byte("receiver shard - tx1"),
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(res))

	for key := range res {
		decodedKey, _ := args.AddressConverter.Decode(key)
		require.True(t, strings.HasPrefix(string(decodedKey), "receiver"))
	}
}

func testExtractAlteredAccountsFromPoolBothSenderAndReceiverShards(t *testing.T) {
	t.Parallel()

	args := getMockArgs()
	args.ShardCoordinator = &testscommon.ShardsCoordinatorMock{
		ComputeIdCalled: func(address []byte) uint32 {
			shardIdChar := string(address)[5]

			return uint32(shardIdChar - '0')
		},
		SelfIDCalled: func() uint32 {
			return 0
		},
	}
	aap, _ := NewAlteredAccountsProvider(args)

	res, err := aap.ExtractAlteredAccountsFromPool(&indexer.Pool{
		Txs: map[string]data.TransactionHandler{
			"hash0": &transaction.Transaction{ // intra-shard 0, different addresses
				SndAddr: []byte("shard0 addr - tx0"),
				RcvAddr: []byte("shard0 addr 2 - tx0"),
			},
			"hash1": &transaction.Transaction{ // intra-shard 0, same addresses
				SndAddr: []byte("shard0 addr 3 - tx1"),
				RcvAddr: []byte("shard0 addr 3 - tx1"),
			},
			"hash2": &transaction.Transaction{ // cross-shard, sender in shard 0
				SndAddr: []byte("shard0 addr - tx2"),
				RcvAddr: []byte("shard1 - tx2"),
			},
			"hash3": &transaction.Transaction{ // cross-shard, receiver in shard 0
				SndAddr: []byte("shard1 addr - tx3"),
				RcvAddr: []byte("shard0 addr - tx3"),
			},
			"hash4": &transaction.Transaction{ // cross-shard, no address in shard 0
				SndAddr: []byte("shard2 addr - tx4"),
				RcvAddr: []byte("shard2 addr - tx3"),
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 5, len(res))

	for key := range res {
		decodedKey, _ := args.AddressConverter.Decode(key)
		require.True(t, strings.HasPrefix(string(decodedKey), "shard0"))
	}
	require.Contains(t, res, args.AddressConverter.Encode([]byte("shard0 addr - tx0")))
	require.Contains(t, res, args.AddressConverter.Encode([]byte("shard0 addr 2 - tx0")))
	require.Contains(t, res, args.AddressConverter.Encode([]byte("shard0 addr 3 - tx1")))
	require.Contains(t, res, args.AddressConverter.Encode([]byte("shard0 addr - tx2")))
	require.Contains(t, res, args.AddressConverter.Encode([]byte("shard0 addr - tx3")))
}

func testExtractAlteredAccountsFromPoolTrieDataChecks(t *testing.T) {
	t.Parallel()

	receiverInSelfShard := "receiver in shard 1"
	expectedBalance := big.NewInt(37)
	args := getMockArgs()
	args.ShardCoordinator = &testscommon.ShardsCoordinatorMock{
		ComputeIdCalled: func(address []byte) uint32 {
			if strings.HasPrefix(string(address), "sender") {
				return 0
			}

			return 1
		},
		SelfIDCalled: func() uint32 {
			return 1
		},
	}
	args.AccountsDB = &state.AccountsStub{
		LoadAccountCalled: func(container []byte) (vmcommon.AccountHandler, error) {
			return &state.UserAccountStub{
				Balance: expectedBalance,
			}, nil
		},
	}
	aap, _ := NewAlteredAccountsProvider(args)

	res, err := aap.ExtractAlteredAccountsFromPool(&indexer.Pool{
		Txs: map[string]data.TransactionHandler{
			"hash0": &transaction.Transaction{
				SndAddr: []byte("sender in shard 0"),
				RcvAddr: []byte(receiverInSelfShard),
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(res))

	expectedAddressKey := args.AddressConverter.Encode([]byte(receiverInSelfShard))
	actualAccount, found := res[expectedAddressKey]
	require.True(t, found)
	require.Equal(t, expectedAddressKey, actualAccount.Address)
	require.Equal(t, expectedBalance.String(), actualAccount.Balance)
}

func testExtractAlteredAccountsFromPoolScrsInvalidRewards(t *testing.T) {
	t.Parallel()

	expectedBalance := big.NewInt(37)
	args := getMockArgs()
	args.ShardCoordinator = &testscommon.ShardsCoordinatorMock{
		ComputeIdCalled: func(address []byte) uint32 {
			if strings.Contains(string(address), "shard 0") {
				return 0
			}

			return 1
		},
		SelfIDCalled: func() uint32 {
			return 0
		},
	}
	args.AccountsDB = &state.AccountsStub{
		LoadAccountCalled: func(container []byte) (vmcommon.AccountHandler, error) {
			return &state.UserAccountStub{
				Balance: expectedBalance,
			}, nil
		},
	}
	aap, _ := NewAlteredAccountsProvider(args)

	res, err := aap.ExtractAlteredAccountsFromPool(&indexer.Pool{
		Txs: map[string]data.TransactionHandler{
			"hash0": &transaction.Transaction{
				SndAddr: []byte("sender in shard 0 - tx 0"),
			},
		},
		Rewards: map[string]data.TransactionHandler{
			"hash1": &rewardTx.RewardTx{
				RcvAddr: []byte("receiver in shard 0 - tx 1"),
			},
		},
		Scrs: map[string]data.TransactionHandler{
			"hash2": &smartContractResult.SmartContractResult{
				SndAddr: []byte("sender in shard 0 - tx 2"),
				RcvAddr: []byte("receiver in shard 0 - tx 2"),
			},
		},
		Invalid: map[string]data.TransactionHandler{
			"hash3": &transaction.Transaction{
				SndAddr: []byte("sender in shard 0 - tx 3"),
				RcvAddr: []byte("receiver in shard 0 - tx 3"),
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, res, 6)
}

func testExtractAlteredAccountsFromPoolShouldIncludeESDT(t *testing.T) {
	t.Parallel()

	expectedToken := esdt.ESDigitalToken{
		Value: big.NewInt(37),
	}
	args := getMockArgs()
	args.AccountsDB = &state.AccountsStub{
		LoadAccountCalled: func(_ []byte) (vmcommon.AccountHandler, error) {
			return &state.UserAccountStub{
				RetrieveValueFromDataTrieTrackerCalled: func(_ []byte) ([]byte, error) {
					tokenBytes, _ := args.Marshalizer.Marshal(expectedToken)
					return tokenBytes, nil
				},
			}, nil
		},
	}
	aap, _ := NewAlteredAccountsProvider(args)

	res, err := aap.ExtractAlteredAccountsFromPool(&indexer.Pool{
		Logs: []*data.LogData{
			{
				LogHandler: &transaction.Log{
					Address: []byte("addr"),
					Events: []*transaction.Event{
						{
							Address:    []byte("addr"),
							Identifier: []byte(core.BuiltInFunctionESDTTransfer),
							Topics: [][]byte{
								[]byte("token0"),
							},
						},
						{
							Address:    []byte("addr"), // other event for the same token, to ensure it isn't added twice
							Identifier: []byte(core.BuiltInFunctionESDTTransfer),
							Topics: [][]byte{
								[]byte("token0"),
							},
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	encodedAddr := args.AddressConverter.Encode([]byte("addr"))

	require.Len(t, res, 1)
	require.Len(t, res[encodedAddr].Tokens, 1)
	require.Equal(t, &indexer.AccountTokenData{
		Identifier: "token0",
		Balance:    expectedToken.Value.String(),
		Nonce:      0,
		MetaData:   nil,
	}, res[encodedAddr].Tokens[0])
}

func testExtractAlteredAccountsFromPoolShouldIncludeNFT(t *testing.T) {
	t.Parallel()

	expectedToken := esdt.ESDigitalToken{
		Value: big.NewInt(37),
		TokenMetaData: &esdt.MetaData{
			Nonce: 38,
		},
	}
	args := getMockArgs()
	args.AccountsDB = &state.AccountsStub{
		LoadAccountCalled: func(_ []byte) (vmcommon.AccountHandler, error) {
			return &state.UserAccountStub{
				RetrieveValueFromDataTrieTrackerCalled: func(_ []byte) ([]byte, error) {
					tokenBytes, _ := args.Marshalizer.Marshal(expectedToken)
					return tokenBytes, nil
				},
			}, nil
		},
	}
	aap, _ := NewAlteredAccountsProvider(args)

	res, err := aap.ExtractAlteredAccountsFromPool(&indexer.Pool{
		Logs: []*data.LogData{
			{
				LogHandler: &transaction.Log{
					Address: []byte("addr"),
					Events: []*transaction.Event{
						{
							Address:    []byte("addr"),
							Identifier: []byte(core.BuiltInFunctionESDTNFTTransfer),
							Topics: [][]byte{
								[]byte("token0"),
								big.NewInt(38).Bytes(),
							},
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	encodedAddr := args.AddressConverter.Encode([]byte("addr"))
	require.Equal(t, &indexer.AccountTokenData{
		Identifier: "token0",
		Balance:    expectedToken.Value.String(),
		Nonce:      expectedToken.TokenMetaData.Nonce,
		MetaData:   expectedToken.TokenMetaData,
	}, res[encodedAddr].Tokens[0])
}

func testExtractAlteredAccountsFromPoolShouldIncludeDestinationFromTokensLogsTopics(t *testing.T) {
	t.Parallel()

	receiverOnDestination := []byte("receiver on destination shard")
	expectedToken := esdt.ESDigitalToken{
		Value: big.NewInt(37),
		TokenMetaData: &esdt.MetaData{
			Nonce: 38,
		},
	}
	args := getMockArgs()
	args.AccountsDB = &state.AccountsStub{
		LoadAccountCalled: func(_ []byte) (vmcommon.AccountHandler, error) {
			return &state.UserAccountStub{
				RetrieveValueFromDataTrieTrackerCalled: func(_ []byte) ([]byte, error) {
					tokenBytes, _ := args.Marshalizer.Marshal(expectedToken)
					return tokenBytes, nil
				},
			}, nil
		},
	}
	aap, _ := NewAlteredAccountsProvider(args)

	res, err := aap.ExtractAlteredAccountsFromPool(&indexer.Pool{
		Logs: []*data.LogData{
			{
				LogHandler: &transaction.Log{
					Address: []byte("addr"),
					Events: []*transaction.Event{
						{
							Address:    []byte("addr"),
							Identifier: []byte(core.BuiltInFunctionESDTNFTTransfer),
							Topics: [][]byte{
								[]byte("token0"),
								big.NewInt(38).Bytes(),
								nil,
								receiverOnDestination,
							},
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	require.Len(t, res, 2)

	mapKeyToSearch := args.AddressConverter.Encode(receiverOnDestination)
	require.Contains(t, res, mapKeyToSearch)
}

func testExtractAlteredAccountsFromPoolAddressHasBalanceChangeEsdtAndfNft(t *testing.T) {
	t.Parallel()

	expectedToken := esdt.ESDigitalToken{
		Value: big.NewInt(37),
		TokenMetaData: &esdt.MetaData{
			Nonce: 38,
		},
	}
	args := getMockArgs()
	args.AccountsDB = &state.AccountsStub{
		LoadAccountCalled: func(_ []byte) (vmcommon.AccountHandler, error) {
			return &state.UserAccountStub{
				RetrieveValueFromDataTrieTrackerCalled: func(_ []byte) ([]byte, error) {
					tokenBytes, _ := args.Marshalizer.Marshal(expectedToken)
					return tokenBytes, nil
				},
			}, nil
		},
	}
	aap, _ := NewAlteredAccountsProvider(args)

	res, err := aap.ExtractAlteredAccountsFromPool(&indexer.Pool{
		Txs: map[string]data.TransactionHandler{
			"hash0": &transaction.Transaction{
				SndAddr: []byte("addr"),
			},
		},
		Logs: []*data.LogData{
			{
				LogHandler: &transaction.Log{
					Address: []byte("addr"),
					Events: []*transaction.Event{
						{
							Address:    []byte("addr"),
							Identifier: []byte(core.BuiltInFunctionESDTTransfer),
							Topics: [][]byte{
								[]byte("esdt"),
							},
						},
						{
							Address:    []byte("addr"),
							Identifier: []byte(core.BuiltInFunctionESDTNFTTransfer),
							Topics: [][]byte{
								[]byte("nft"),
								big.NewInt(38).Bytes(),
							},
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	encodedAddr := args.AddressConverter.Encode([]byte("addr"))
	require.Len(t, res[encodedAddr].Tokens, 2)
}

func testExtractAlteredAccountsFromPoolAddressHasMultipleNfts(t *testing.T) {
	t.Parallel()

	expectedToken0 := esdt.ESDigitalToken{
		Value: big.NewInt(37),
	}
	expectedToken1 := esdt.ESDigitalToken{
		Value: big.NewInt(38),
		TokenMetaData: &esdt.MetaData{
			Nonce: 5,
			Name:  []byte("nft-0"),
		},
	}
	expectedToken2 := esdt.ESDigitalToken{
		Value: big.NewInt(37),
		TokenMetaData: &esdt.MetaData{
			Nonce: 6,
			Name:  []byte("nft-1"),
		},
	}
	args := getMockArgs()
	args.AccountsDB = &state.AccountsStub{
		LoadAccountCalled: func(_ []byte) (vmcommon.AccountHandler, error) {
			return &state.UserAccountStub{
				RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
					if strings.Contains(string(key), "esdttoken") {
						tokenBytes, _ := args.Marshalizer.Marshal(expectedToken0)
						return tokenBytes, nil
					}
					if strings.Contains(string(key), "nft-0") {
						tokenBytes, _ := args.Marshalizer.Marshal(expectedToken1)
						return tokenBytes, nil
					}
					if strings.Contains(string(key), "nft-1") {
						tokenBytes, _ := args.Marshalizer.Marshal(expectedToken2)
						return tokenBytes, nil
					}

					return nil, nil
				},
			}, nil
		},
	}
	aap, _ := NewAlteredAccountsProvider(args)

	res, err := aap.ExtractAlteredAccountsFromPool(&indexer.Pool{
		Txs: map[string]data.TransactionHandler{
			"hash0": &transaction.Transaction{
				SndAddr: []byte("addr"),
			},
		},
		Logs: []*data.LogData{
			{
				LogHandler: &transaction.Log{
					Address: []byte("addr"),
					Events: []*transaction.Event{
						{
							Address:    []byte("addr"),
							Identifier: []byte(core.BuiltInFunctionESDTTransfer),
							Topics: [][]byte{
								[]byte("esdttoken"),
							},
						},
						{
							Address:    []byte("addr"),
							Identifier: []byte(core.BuiltInFunctionESDTNFTTransfer),
							Topics: [][]byte{
								expectedToken1.TokenMetaData.Name,
								big.NewInt(0).SetUint64(expectedToken1.TokenMetaData.Nonce).Bytes(),
							},
						},
						{
							Address:    []byte("addr"),
							Identifier: []byte(core.BuiltInFunctionESDTNFTTransfer),
							Topics: [][]byte{
								expectedToken2.TokenMetaData.Name,
								big.NewInt(0).SetUint64(expectedToken2.TokenMetaData.Nonce).Bytes(),
							},
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	encodedAddr := args.AddressConverter.Encode([]byte("addr"))
	require.Len(t, res[encodedAddr].Tokens, 3)

	numChecks := 0
	for _, token := range res[encodedAddr].Tokens {
		if token.Identifier == "esdttoken" {
			require.Equal(t, &indexer.AccountTokenData{
				Identifier: "esdttoken",
				Balance:    expectedToken0.Value.String(),
				Nonce:      0,
				MetaData:   nil,
			}, token)
			numChecks++
		}
		if token.Identifier == "nft-0" {
			require.Equal(t, &indexer.AccountTokenData{
				Identifier: "nft-0",
				Balance:    expectedToken1.Value.String(),
				Nonce:      expectedToken1.TokenMetaData.Nonce,
				MetaData:   expectedToken1.TokenMetaData,
			}, token)
			numChecks++
		}
		if token.Identifier == "nft-1" {
			require.Equal(t, &indexer.AccountTokenData{
				Identifier: "nft-1",
				Balance:    expectedToken2.Value.String(),
				Nonce:      expectedToken2.TokenMetaData.Nonce,
				MetaData:   expectedToken2.TokenMetaData,
			}, token)
			numChecks++
		}
	}

	require.Equal(t, 3, numChecks)
}

func getMockArgs() ArgsAlteredAccountsProvider {
	return ArgsAlteredAccountsProvider{
		ShardCoordinator: &testscommon.ShardsCoordinatorMock{},
		AddressConverter: &testscommon.PubkeyConverterMock{},
		Marshalizer:      &testscommon.MarshalizerMock{},
		AccountsDB:       &state.AccountsStub{},
	}
}
