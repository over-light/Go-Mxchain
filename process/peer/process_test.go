package peer_test

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/process/peer"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/stretchr/testify/assert"
)

const (
	validatorIncreaseRatingStep     = int32(1)
	validatorDecreaseRatingStep     = int32(-2)
	proposerIncreaseRatingStep      = int32(2)
	proposerDecreaseRatingStep      = int32(-4)
	metaValidatorIncreaseRatingStep = int32(3)
	metaValidatorDecreaseRatingStep = int32(-4)
	metaProposerIncreaseRatingStep  = int32(5)
	metaProposerDecreaseRatingStep  = int32(-10)
	minRating                       = uint32(1)
	maxRating                       = uint32(100)
	startRating                     = uint32(50)
)

func createMockArguments() peer.ArgValidatorStatisticsProcessor {
	economicsData, _ := economics.NewEconomicsData(
		&config.EconomicsConfig{
			GlobalSettings: config.GlobalSettings{
				TotalSupply:      "2000000000000000000000",
				MinimumInflation: 0,
				MaximumInflation: 0.5,
			},
			RewardsSettings: config.RewardsSettings{
				LeaderPercentage: 0.10,
			},
			FeeSettings: config.FeeSettings{
				MaxGasLimitPerBlock:  "10000000",
				MinGasPrice:          "10",
				MinGasLimit:          "10",
				GasPerDataByte:       "1",
				DataLimitForBaseCalc: "10000",
			},
			ValidatorSettings: config.ValidatorSettings{
				GenesisNodePrice:         "500",
				UnBondPeriod:             "5",
				TotalSupply:              "200000000000",
				MinStepValue:             "100000",
				NumNodes:                 1000,
				AuctionEnableNonce:       "100000",
				StakeEnableNonce:         "100000",
				NumRoundsWithoutBleed:    "1000",
				MaximumPercentageToBleed: "0.5",
				BleedPercentagePerRound:  "0.00001",
				UnJailValue:              "1000",
			},
		},
	)

	arguments := peer.ArgValidatorStatisticsProcessor{
		Marshalizer: &mock.MarshalizerMock{},
		DataPool: &mock.PoolsHolderStub{
			HeadersCalled: func() dataRetriever.HeadersPool {
				return nil
			},
		},
		StorageService:      &mock.ChainStorerMock{},
		NodesCoordinator:    &mock.NodesCoordinatorMock{},
		ShardCoordinator:    mock.NewOneShardCoordinatorMock(),
		AdrConv:             &mock.AddressConverterMock{},
		PeerAdapter:         getAccountsMock(),
		StakeValue:          economicsData.GenesisNodePrice(),
		Rater:               createMockRater(),
		RewardsHandler:      economicsData,
		MaxComputableRounds: 1000,
		StartEpoch:          0,
	}
	return arguments
}

func createMockRater() *mock.RaterMock {
	rater := mock.GetNewMockRater()
	rater.MinRating = minRating
	rater.MaxRating = maxRating
	rater.StartRating = startRating
	rater.IncreaseProposer = proposerIncreaseRatingStep
	rater.DecreaseProposer = proposerDecreaseRatingStep
	rater.IncreaseValidator = validatorIncreaseRatingStep
	rater.DecreaseValidator = validatorDecreaseRatingStep
	rater.MetaIncreaseProposer = metaProposerIncreaseRatingStep
	rater.MetaDecreaseProposer = metaProposerDecreaseRatingStep
	rater.MetaIncreaseValidator = metaValidatorIncreaseRatingStep
	rater.MetaDecreaseValidator = metaValidatorDecreaseRatingStep
	return rater
}

func createMockCache() map[string]data.HeaderHandler {
	return make(map[string]data.HeaderHandler)
}

func TestNewValidatorStatisticsProcessor_NilPeerAdaptersShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArguments()
	arguments.PeerAdapter = nil
	validatorStatistics, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Nil(t, validatorStatistics)
	assert.Equal(t, process.ErrNilPeerAccountsAdapter, err)
}

func TestNewValidatorStatisticsProcessor_NilAddressConverterShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArguments()
	arguments.AdrConv = nil
	validatorStatistics, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Nil(t, validatorStatistics)
	assert.Equal(t, process.ErrNilAddressConverter, err)
}

func TestNewValidatorStatisticsProcessor_NilNodesCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArguments()
	arguments.NodesCoordinator = nil
	validatorStatistics, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Nil(t, validatorStatistics)
	assert.Equal(t, process.ErrNilNodesCoordinator, err)
}

func TestNewValidatorStatisticsProcessor_NilShardCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArguments()
	arguments.ShardCoordinator = nil
	validatorStatistics, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Nil(t, validatorStatistics)
	assert.Equal(t, process.ErrNilShardCoordinator, err)
}

func TestNewValidatorStatisticsProcessor_NilStorageShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArguments()
	arguments.StorageService = nil
	validatorStatistics, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Nil(t, validatorStatistics)
	assert.Equal(t, process.ErrNilStorage, err)
}

func TestNewValidatorStatisticsProcessor_ZeroMaxComputableRoundsShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArguments()
	arguments.MaxComputableRounds = 0
	validatorStatistics, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Nil(t, validatorStatistics)
	assert.Equal(t, process.ErrZeroMaxComputableRounds, err)
}

func TestNewValidatorStatisticsProcessor_NilStakeValueShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArguments()
	arguments.StakeValue = nil
	validatorStatistics, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Nil(t, validatorStatistics)
	assert.Equal(t, process.ErrNilEconomicsData, err)
}

func TestNewValidatorStatisticsProcessor_NilRaterShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArguments()
	arguments.Rater = nil
	validatorStatistics, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Nil(t, validatorStatistics)
	assert.Equal(t, process.ErrNilRater, err)
}

func TestNewValidatorStatisticsProcessor_NilRewardsHandlerShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArguments()
	arguments.RewardsHandler = nil
	validatorStatistics, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Nil(t, validatorStatistics)
	assert.Equal(t, process.ErrNilRewardsHandler, err)
}
func TestNewValidatorStatisticsProcessor_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArguments()
	arguments.Marshalizer = nil
	validatorStatistics, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Nil(t, validatorStatistics)
	assert.Equal(t, process.ErrNilMarshalizer, err)
}

func TestNewValidatorStatisticsProcessor_NilDataPoolShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArguments()
	arguments.DataPool = nil
	validatorStatistics, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Nil(t, validatorStatistics)
	assert.Equal(t, process.ErrNilDataPoolHolder, err)
}

func TestNewValidatorStatisticsProcessor(t *testing.T) {
	t.Parallel()

	arguments := createMockArguments()
	validatorStatistics, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.NotNil(t, validatorStatistics)
	assert.Nil(t, err)
}

func TestValidatorStatisticsProcessor_SaveInitialStateErrOnWrongAddressConverter(t *testing.T) {
	t.Parallel()

	addressErr := errors.New("hex address error")
	addressConverter := &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (container state.AddressContainer, e error) {
			return nil, addressErr
		},
	}

	arguments := createMockArguments()
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		GetAllEligibleValidatorsPublicKeysCalled: func() (map[uint32][][]byte, error) {
			keys := make(map[uint32][][]byte)
			keys[0] = make([][]byte, 0)
			keys[0] = append(keys[0], []byte("aaaa"))
			return keys, nil
		},
		GetValidatorWithPublicKeyCalled: func(publicKey []byte, _ uint32) (sharding.Validator, uint32, error) {
			validator, _ := sharding.NewValidator(publicKey, publicKey)
			return validator, 0, nil
		},
	}

	arguments.AdrConv = addressConverter
	validatorStatistics, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Equal(t, addressErr, err)
	assert.Nil(t, validatorStatistics)
}

func TestValidatorStatisticsProcessor_SaveInitialStateErrOnGetAccountFail(t *testing.T) {
	t.Parallel()

	adapterError := errors.New("account error")
	peerAdapters := &mock.AccountsStub{
		LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
			return nil, adapterError
		},
	}

	addressConverter := &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (container state.AddressContainer, e error) {
			return &mock.AddressMock{}, nil
		},
	}

	arguments := createMockArguments()
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		GetAllEligibleValidatorsPublicKeysCalled: func() (map[uint32][][]byte, error) {
			keys := make(map[uint32][][]byte)
			keys[0] = make([][]byte, 0)
			keys[0] = append(keys[0], []byte("aaaa"))
			return keys, nil
		},
		GetValidatorWithPublicKeyCalled: func(publicKey []byte, _ uint32) (sharding.Validator, uint32, error) {
			validator, _ := sharding.NewValidator(publicKey, publicKey)
			return validator, 0, nil
		},
	}

	arguments.PeerAdapter = peerAdapters
	arguments.AdrConv = addressConverter
	_, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Equal(t, adapterError, err)
}

func TestValidatorStatisticsProcessor_SaveInitialStateGetAccountReturnsInvalid(t *testing.T) {
	t.Parallel()

	peerAdapter := &mock.AccountsStub{
		LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
			return &mock.AccountWrapMock{}, nil
		},
	}

	addressConverter := &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (container state.AddressContainer, e error) {
			return &mock.AddressMock{}, nil
		},
	}

	arguments := createMockArguments()
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		GetAllEligibleValidatorsPublicKeysCalled: func() (map[uint32][][]byte, error) {
			keys := make(map[uint32][][]byte)
			keys[0] = make([][]byte, 0)
			keys[0] = append(keys[0], []byte("aaaa"))
			return keys, nil
		},
		GetValidatorWithPublicKeyCalled: func(publicKey []byte, _ uint32) (sharding.Validator, uint32, error) {
			validator, _ := sharding.NewValidator(publicKey, publicKey)
			return validator, 0, nil
		},
	}

	arguments.PeerAdapter = peerAdapter
	arguments.AdrConv = addressConverter
	_, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Equal(t, process.ErrInvalidPeerAccount, err)
}

func TestValidatorStatisticsProcessor_SaveInitialStateSetAddressErrors(t *testing.T) {
	t.Parallel()

	saveAccountError := errors.New("save account error")
	peerAccount, _ := state.NewPeerAccount(&mock.AddressMock{})
	peerAdapter := &mock.AccountsStub{
		LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
			return peerAccount, nil
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			return saveAccountError
		},
	}

	addressConverter := &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (container state.AddressContainer, e error) {
			return &mock.AddressMock{}, nil
		},
	}

	arguments := createMockArguments()
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		GetAllEligibleValidatorsPublicKeysCalled: func() (map[uint32][][]byte, error) {
			keys := make(map[uint32][][]byte)
			keys[0] = make([][]byte, 0)
			keys[0] = append(keys[0], []byte("aaaa"))
			return keys, nil
		},
		GetValidatorWithPublicKeyCalled: func(publicKey []byte, _ uint32) (sharding.Validator, uint32, error) {
			validator, _ := sharding.NewValidator(publicKey, publicKey)
			return validator, 0, nil
		},
	}

	arguments.PeerAdapter = peerAdapter
	arguments.AdrConv = addressConverter
	_, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Equal(t, saveAccountError, err)
}

func TestValidatorStatisticsProcessor_SaveInitialStateCommitErrors(t *testing.T) {
	t.Parallel()

	commitError := errors.New("commit error")
	peerAccount, _ := state.NewPeerAccount(&mock.AddressMock{})
	peerAdapter := &mock.AccountsStub{
		LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
			return peerAccount, nil
		},
		CommitCalled: func() (bytes []byte, e error) {
			return nil, commitError
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			return nil
		},
	}

	addressConverter := &mock.AddressConverterStub{
		CreateAddressFromHexCalled: func(hexAddress string) (container state.AddressContainer, e error) {
			return &mock.AddressMock{}, nil
		},
	}

	arguments := createMockArguments()
	arguments.PeerAdapter = peerAdapter
	arguments.AdrConv = addressConverter
	_, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Equal(t, commitError, err)
}

func TestValidatorStatisticsProcessor_SaveInitialStateCommit(t *testing.T) {
	t.Parallel()

	peerAccount, _ := state.NewPeerAccount(&mock.AddressMock{})
	peerAdapter := &mock.AccountsStub{
		LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
			return peerAccount, nil
		},
		CommitCalled: func() (bytes []byte, e error) {
			return nil, nil
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			return nil
		},
	}

	addressConverter := &mock.AddressConverterStub{
		CreateAddressFromHexCalled: func(hexAddress string) (container state.AddressContainer, e error) {
			return &mock.AddressMock{}, nil
		},
	}

	arguments := createMockArguments()
	arguments.PeerAdapter = peerAdapter
	arguments.AdrConv = addressConverter
	_, err := peer.NewValidatorStatisticsProcessor(arguments)

	assert.Nil(t, err)
}

func TestValidatorStatisticsProcessor_SaveInitialStateCommitsEligibleAndWaiting(t *testing.T) {
	t.Parallel()

	arguments, waitingMap, eligibleMap, actualMap := createCustomArgumentsForSaveInitialState()

	_, err := peer.NewValidatorStatisticsProcessor(arguments)

	//verify 6 saves
	for _, validators := range eligibleMap {
		for _, val := range validators {
			assert.Equal(t, 6, actualMap[string(val.PubKey())])
		}
	}

	for _, validators := range waitingMap {
		for _, val := range validators {
			assert.Equal(t, 6, actualMap[string(val.PubKey())])
		}
	}

	assert.Nil(t, err)
}

func TestValidatorStatisticsProcessor_UpdatePeerStateReturnsRootHashForGenesis(t *testing.T) {
	t.Parallel()

	expectedRootHash := []byte("root hash")
	peerAdapter := getAccountsMock()
	peerAdapter.RootHashCalled = func() (bytes []byte, e error) {
		return expectedRootHash, nil
	}

	arguments := createMockArguments()
	arguments.PeerAdapter = peerAdapter
	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)

	header := getMetaHeaderHandler([]byte("header"))
	header.Nonce = 0
	rootHash, err := validatorStatistics.UpdatePeerState(header, createMockCache())

	assert.Nil(t, err)
	assert.Equal(t, expectedRootHash, rootHash)
}

func TestValidatorStatisticsProcessor_UpdatePeerStateReturnsErrForRootHashErr(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("expected error")
	peerAdapter := getAccountsMock()
	peerAdapter.RootHashCalled = func() (bytes []byte, e error) {
		return nil, expectedError
	}

	arguments := createMockArguments()
	arguments.PeerAdapter = peerAdapter
	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)

	header := getMetaHeaderHandler([]byte("header"))
	header.Nonce = 0
	_, err := validatorStatistics.UpdatePeerState(header, createMockCache())

	assert.Equal(t, expectedError, err)
}

func TestValidatorStatisticsProcessor_UpdatePeerStateComputeValidatorErrShouldError(t *testing.T) {
	t.Parallel()

	computeValidatorsErr := errors.New("compute validators error")

	arguments := createMockArguments()
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validatorsGroup []sharding.Validator, err error) {
			return nil, computeValidatorsErr
		},
	}
	arguments.DataPool = &mock.PoolsHolderStub{
		HeadersCalled: func() dataRetriever.HeadersPool {
			return &mock.HeadersCacherStub{
				GetHeaderByHashCalled: func(hash []byte) (handler data.HeaderHandler, e error) {
					return getMetaHeaderHandler([]byte("header")), nil
				},
			}
		},
	}
	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)

	header := getMetaHeaderHandler([]byte("header"))
	_, err := validatorStatistics.UpdatePeerState(header, createMockCache())

	assert.Equal(t, computeValidatorsErr, err)
}

func TestValidatorStatisticsProcessor_UpdatePeerStateCreateAddressFromPublicKeyBytesErr(t *testing.T) {
	t.Parallel()

	createAddressErr := errors.New("create address error")

	arguments := createMockArguments()
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validatorsGroup []sharding.Validator, err error) {
			return []sharding.Validator{&mock.ValidatorMock{}}, nil
		},
	}
	arguments.AdrConv = &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (container state.AddressContainer, e error) {
			return nil, createAddressErr
		},
	}
	arguments.DataPool = &mock.PoolsHolderStub{
		HeadersCalled: func() dataRetriever.HeadersPool {
			return &mock.HeadersCacherStub{
				GetHeaderByHashCalled: func(hash []byte) (handler data.HeaderHandler, e error) {
					return getMetaHeaderHandler([]byte("header")), nil
				},
			}
		},
	}

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)

	header := getMetaHeaderHandler([]byte("header"))
	_, err := validatorStatistics.UpdatePeerState(header, createMockCache())

	assert.Equal(t, createAddressErr, err)
}

func TestValidatorStatisticsProcessor_UpdatePeerStateGetExistingAccountErr(t *testing.T) {
	t.Parallel()

	existingAccountErr := errors.New("existing account err")
	adapter := getAccountsMock()
	adapter.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return nil, existingAccountErr
	}

	arguments := createMockArguments()
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validatorsGroup []sharding.Validator, err error) {
			return []sharding.Validator{&mock.ValidatorMock{}}, nil
		},
	}
	arguments.AdrConv = &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (container state.AddressContainer, e error) {
			return &mock.AddressMock{}, nil
		},
	}
	arguments.DataPool = &mock.PoolsHolderStub{
		HeadersCalled: func() dataRetriever.HeadersPool {
			return &mock.HeadersCacherStub{
				GetHeaderByHashCalled: func(hash []byte) (handler data.HeaderHandler, e error) {
					return getMetaHeaderHandler([]byte("header")), nil
				},
			}
		},
	}
	arguments.PeerAdapter = adapter
	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)

	header := getMetaHeaderHandler([]byte("header"))
	_, err := validatorStatistics.UpdatePeerState(header, createMockCache())

	assert.Equal(t, existingAccountErr, err)
}

func TestValidatorStatisticsProcessor_UpdatePeerStateGetExistingAccountInvalidType(t *testing.T) {
	t.Parallel()

	adapter := getAccountsMock()
	adapter.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return &mock.AccountWrapMock{}, nil
	}

	arguments := createMockArguments()
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validatorsGroup []sharding.Validator, err error) {
			return []sharding.Validator{&mock.ValidatorMock{}}, nil
		},
	}
	arguments.AdrConv = &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (container state.AddressContainer, e error) {
			return &mock.AddressMock{}, nil
		},
	}
	arguments.DataPool = &mock.PoolsHolderStub{
		HeadersCalled: func() dataRetriever.HeadersPool {
			return &mock.HeadersCacherStub{
				GetHeaderByHashCalled: func(hash []byte) (handler data.HeaderHandler, e error) {
					return getMetaHeaderHandler([]byte("header")), nil
				},
			}
		},
	}
	arguments.PeerAdapter = adapter
	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)

	header := getMetaHeaderHandler([]byte("header"))
	_, err := validatorStatistics.UpdatePeerState(header, createMockCache())

	assert.Equal(t, process.ErrInvalidPeerAccount, err)
}

func TestValidatorStatisticsProcessor_UpdatePeerStateGetHeaderError(t *testing.T) {
	t.Parallel()

	getHeaderError := errors.New("get header error")
	adapter := getAccountsMock()
	marshalizer := &mock.MarshalizerStub{}

	adapter.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return state.NewPeerAccount(addressContainer)
	}
	shardCoordinatorMock := mock.NewOneShardCoordinatorMock()

	arguments := createMockArguments()
	arguments.Marshalizer = marshalizer
	arguments.DataPool = &mock.PoolsHolderStub{
		HeadersCalled: func() dataRetriever.HeadersPool {
			return &mock.HeadersCacherStub{}
		},
	}
	arguments.StorageService = &mock.ChainStorerMock{
		GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
			return &mock.StorerStub{
				GetCalled: func(key []byte) (bytes []byte, e error) {
					return nil, getHeaderError
				},
			}
		},
	}
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validatorsGroup []sharding.Validator, err error) {
			return []sharding.Validator{&mock.ValidatorMock{}, &mock.ValidatorMock{}}, nil
		},
	}
	arguments.ShardCoordinator = shardCoordinatorMock
	arguments.AdrConv = &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (container state.AddressContainer, e error) {
			return &mock.AddressMock{}, nil
		},
	}
	arguments.PeerAdapter = adapter
	arguments.Rater = mock.GetNewMockRater()
	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)

	header := getMetaHeaderHandler([]byte("header"))
	header.Nonce = 2
	_, err := validatorStatistics.UpdatePeerState(header, createMockCache())

	assert.Equal(t, process.ErrMissingHeader, err)
}

func TestValidatorStatisticsProcessor_UpdatePeerStateGetHeaderUnmarshalError(t *testing.T) {
	t.Parallel()

	getHeaderUnmarshalError := errors.New("get header unmarshal error")
	adapter := getAccountsMock()
	marshalizer := &mock.MarshalizerStub{
		UnmarshalCalled: func(obj interface{}, buff []byte) error {
			return getHeaderUnmarshalError
		},
	}

	adapter.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return state.NewPeerAccount(addressContainer)
	}
	shardCoordinatorMock := mock.NewOneShardCoordinatorMock()

	arguments := createMockArguments()
	arguments.Marshalizer = marshalizer
	arguments.DataPool = &mock.PoolsHolderStub{
		HeadersCalled: func() dataRetriever.HeadersPool {
			return &mock.HeadersCacherStub{}
		},
	}
	arguments.StorageService = &mock.ChainStorerMock{
		GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
			return &mock.StorerStub{
				GetCalled: func(key []byte) (bytes []byte, e error) {
					return nil, nil
				},
			}
		},
	}
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validatorsGroup []sharding.Validator, err error) {
			return []sharding.Validator{&mock.ValidatorMock{}, &mock.ValidatorMock{}}, nil
		},
	}
	arguments.ShardCoordinator = shardCoordinatorMock
	arguments.AdrConv = &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (container state.AddressContainer, e error) {
			return &mock.AddressMock{}, nil
		},
	}
	arguments.PeerAdapter = adapter
	arguments.Rater = mock.GetNewMockRater()

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)

	header := getMetaHeaderHandler([]byte("header"))
	header.Nonce = 2
	_, err := validatorStatistics.UpdatePeerState(header, createMockCache())

	assert.Equal(t, process.ErrUnmarshalWithoutSuccess, err)
}

func TestValidatorStatisticsProcessor_UpdatePeerStateCallsIncrease(t *testing.T) {
	t.Parallel()

	adapter := getAccountsMock()
	increaseLeaderCalled := false
	increaseValidatorCalled := false
	marshalizer := &mock.MarshalizerStub{}

	adapter.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return &mock.PeerAccountHandlerMock{
			IncreaseLeaderSuccessRateCalled: func(value uint32) {
				increaseLeaderCalled = true
			},
			IncreaseValidatorSuccessRateCalled: func(value uint32) {
				increaseValidatorCalled = true
			},
		}, nil
	}
	adapter.RootHashCalled = func() (bytes []byte, e error) {
		return nil, nil
	}
	shardCoordinatorMock := mock.NewOneShardCoordinatorMock()

	arguments := createMockArguments()
	arguments.Marshalizer = marshalizer
	arguments.DataPool = &mock.PoolsHolderStub{
		HeadersCalled: func() dataRetriever.HeadersPool {
			return &mock.HeadersCacherStub{}
		},
	}
	arguments.StorageService = &mock.ChainStorerMock{
		GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
			return &mock.StorerStub{
				GetCalled: func(key []byte) (bytes []byte, e error) {
					return nil, nil
				},
			}
		},
	}
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validatorsGroup []sharding.Validator, err error) {
			return []sharding.Validator{&mock.ValidatorMock{}, &mock.ValidatorMock{}}, nil
		},
	}
	arguments.ShardCoordinator = shardCoordinatorMock
	arguments.AdrConv = &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (container state.AddressContainer, e error) {
			return &mock.AddressMock{}, nil
		},
	}
	arguments.PeerAdapter = adapter
	arguments.Rater = mock.GetNewMockRater()
	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)

	header := getMetaHeaderHandler([]byte("header"))
	header.PubKeysBitmap = []byte{255, 0}

	marshalizer.UnmarshalCalled = func(obj interface{}, buff []byte) error {
		switch v := obj.(type) {
		case *block.MetaBlock:
			*v = block.MetaBlock{
				PubKeysBitmap:   []byte{255, 255},
				AccumulatedFees: big.NewInt(0),
			}
		case *block.Header:
			*v = block.Header{
				AccumulatedFees: big.NewInt(0),
			}
		default:
			fmt.Println(v)
		}

		return nil
	}

	_, err := validatorStatistics.UpdatePeerState(header, createMockCache())

	assert.Nil(t, err)
	assert.True(t, increaseLeaderCalled)
	assert.True(t, increaseValidatorCalled)
}

func TestValidatorStatisticsProcessor_UpdatePeerStateCheckForMissedBlocksErr(t *testing.T) {
	t.Parallel()

	adapter := getAccountsMock()
	missedBlocksErr := errors.New("missed blocks error")
	shouldErr := false
	marshalizer := &mock.MarshalizerStub{}

	adapter.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return &mock.PeerAccountHandlerMock{
			DecreaseLeaderSuccessRateCalled: func(value uint32) {
				shouldErr = true
			},
		}, nil
	}

	adapter.SaveAccountCalled = func(account state.AccountHandler) error {
		if shouldErr {
			return missedBlocksErr
		}
		return nil
	}
	adapter.RootHashCalled = func() (bytes []byte, e error) {
		return nil, nil
	}
	shardCoordinatorMock := mock.NewOneShardCoordinatorMock()

	arguments := createMockArguments()
	arguments.DataPool = &mock.PoolsHolderStub{
		HeadersCalled: func() dataRetriever.HeadersPool {
			return &mock.HeadersCacherStub{}
		},
	}
	arguments.StorageService = &mock.ChainStorerMock{
		GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
			return &mock.StorerStub{
				GetCalled: func(key []byte) (bytes []byte, e error) {
					return nil, nil
				},
			}
		},
	}
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validatorsGroup []sharding.Validator, err error) {
			return []sharding.Validator{&mock.ValidatorMock{}, &mock.ValidatorMock{}}, nil
		},
	}
	arguments.ShardCoordinator = shardCoordinatorMock
	arguments.AdrConv = &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (container state.AddressContainer, e error) {
			return &mock.AddressMock{}, nil
		},
	}
	arguments.PeerAdapter = adapter
	arguments.Marshalizer = marshalizer
	arguments.Rater = mock.GetNewMockRater()

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)

	header := getMetaHeaderHandler([]byte("header"))
	header.Nonce = 2
	header.Round = 2

	marshalizer.UnmarshalCalled = func(obj interface{}, buff []byte) error {
		switch v := obj.(type) {
		case *block.MetaBlock:
			*v = block.MetaBlock{
				Nonce:         0,
				PubKeysBitmap: []byte{0, 0},
			}
		case *block.Header:
			*v = block.Header{}
		default:
			fmt.Println(v)
		}

		return nil
	}

	_, err := validatorStatistics.UpdatePeerState(header, createMockCache())

	assert.Equal(t, missedBlocksErr, err)
}

func TestValidatorStatisticsProcessor_CheckForMissedBlocksNoMissedBlocks(t *testing.T) {
	t.Parallel()

	computeValidatorGroupCalled := false
	shardCoordinatorMock := mock.NewOneShardCoordinatorMock()

	arguments := createMockArguments()
	arguments.Marshalizer = &mock.MarshalizerMock{}
	arguments.DataPool = &mock.PoolsHolderStub{}
	arguments.StorageService = &mock.ChainStorerMock{}
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validatorsGroup []sharding.Validator, err error) {
			computeValidatorGroupCalled = true
			return nil, nil
		},
	}
	arguments.ShardCoordinator = shardCoordinatorMock
	arguments.AdrConv = &mock.AddressConverterMock{}
	arguments.PeerAdapter = getAccountsMock()

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)
	err := validatorStatistics.CheckForMissedBlocks(1, 0, []byte("prev"), 0, 0)
	assert.Nil(t, err)
	assert.False(t, computeValidatorGroupCalled)

	err = validatorStatistics.CheckForMissedBlocks(1, 1, []byte("prev"), 0, 0)
	assert.Nil(t, err)
	assert.False(t, computeValidatorGroupCalled)

	err = validatorStatistics.CheckForMissedBlocks(2, 1, []byte("prev"), 0, 0)
	assert.Nil(t, err)
	assert.False(t, computeValidatorGroupCalled)
}

func TestValidatorStatisticsProcessor_CheckForMissedBlocksErrOnComputeValidatorList(t *testing.T) {
	t.Parallel()

	computeErr := errors.New("compute err")
	shardCoordinatorMock := mock.NewOneShardCoordinatorMock()

	arguments := createMockArguments()
	arguments.Marshalizer = &mock.MarshalizerMock{}
	arguments.DataPool = &mock.PoolsHolderStub{}
	arguments.StorageService = &mock.ChainStorerMock{}
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validatorsGroup []sharding.Validator, err error) {
			return nil, computeErr
		},
	}
	arguments.ShardCoordinator = shardCoordinatorMock
	arguments.AdrConv = &mock.AddressConverterMock{}
	arguments.PeerAdapter = getAccountsMock()
	arguments.Rater = mock.GetNewMockRater()
	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)
	err := validatorStatistics.CheckForMissedBlocks(2, 0, []byte("prev"), 0, 0)
	assert.Equal(t, computeErr, err)
}

func TestValidatorStatisticsProcessor_CheckForMissedBlocksErrOnGetPeerAcc(t *testing.T) {
	t.Parallel()

	peerAccErr := errors.New("peer acc err")
	shardCoordinatorMock := mock.NewOneShardCoordinatorMock()

	arguments := createMockArguments()
	arguments.Marshalizer = &mock.MarshalizerMock{}
	arguments.DataPool = &mock.PoolsHolderStub{}
	arguments.StorageService = &mock.ChainStorerMock{}
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validatorsGroup []sharding.Validator, err error) {
			return []sharding.Validator{
				&mock.ValidatorMock{},
			}, nil
		},
	}
	arguments.ShardCoordinator = shardCoordinatorMock
	arguments.AdrConv = &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (addressContainer state.AddressContainer, e error) {
			return nil, peerAccErr
		},
	}
	arguments.PeerAdapter = getAccountsMock()
	arguments.Rater = mock.GetNewMockRater()
	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)
	err := validatorStatistics.CheckForMissedBlocks(2, 0, []byte("prev"), 0, 0)
	assert.Equal(t, peerAccErr, err)
}

func TestValidatorStatisticsProcessor_CheckForMissedBlocksErrOnDecrease(t *testing.T) {
	t.Parallel()

	decreaseErr := false
	shardCoordinatorMock := mock.NewOneShardCoordinatorMock()
	peerAdapter := getAccountsMock()
	peerAdapter.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return &mock.PeerAccountHandlerMock{
			DecreaseLeaderSuccessRateCalled: func(value uint32) {
				decreaseErr = true
			},
		}, nil
	}

	arguments := createMockArguments()
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validatorsGroup []sharding.Validator, err error) {
			return []sharding.Validator{
				&mock.ValidatorMock{},
			}, nil
		},
	}
	arguments.ShardCoordinator = shardCoordinatorMock
	arguments.AdrConv = &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (addressContainer state.AddressContainer, e error) {
			return nil, nil
		},
	}
	arguments.PeerAdapter = peerAdapter
	arguments.Rater = mock.GetNewMockRater()
	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)
	_ = validatorStatistics.CheckForMissedBlocks(2, 0, []byte("prev"), 0, 0)
	_ = validatorStatistics.UpdateMissedBlocksCounters()
	assert.True(t, decreaseErr)
}

func TestValidatorStatisticsProcessor_CheckForMissedBlocksCallsDecrease(t *testing.T) {
	t.Parallel()

	currentHeaderRound := 10
	previousHeaderRound := 4
	decreaseCount := 0
	pubKey := []byte("pubKey")
	shardCoordinatorMock := mock.NewOneShardCoordinatorMock()
	peerAdapter := getAccountsMock()
	peerAdapter.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return &mock.PeerAccountHandlerMock{
			DecreaseLeaderSuccessRateCalled: func(value uint32) {
				decreaseCount += 5
			},
		}, nil
	}

	arguments := createMockArguments()
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validatorsGroup []sharding.Validator, err error) {
			return []sharding.Validator{
				&mock.ValidatorMock{
					PubKeyCalled: func() []byte {
						return pubKey
					},
				},
			}, nil
		},
	}
	arguments.ShardCoordinator = shardCoordinatorMock
	arguments.AdrConv = &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (addressContainer state.AddressContainer, e error) {
			return nil, nil
		},
	}
	arguments.PeerAdapter = peerAdapter
	arguments.Rater = mock.GetNewMockRater()
	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)
	_ = validatorStatistics.CheckForMissedBlocks(uint64(currentHeaderRound), uint64(previousHeaderRound), []byte("prev"), 0, 0)
	counters := validatorStatistics.GetLeaderDecreaseCount(pubKey)
	_ = validatorStatistics.UpdateMissedBlocksCounters()
	assert.Equal(t, uint32(currentHeaderRound-previousHeaderRound-1), counters)
}

func TestValidatorStatisticsProcessor_CheckForMissedBlocksWithRoundDifferenceGreaterThanMaxComputableCallsDecreaseOnlyOnce(t *testing.T) {
	t.Parallel()

	currentHeaderRound := 20
	previousHeaderRound := 10
	decreaseValidatorCalls := 0
	decreaseLeaderCalls := 0
	setTempRatingCalls := 0

	validatorPublicKeys := make(map[uint32][][]byte)
	validatorPublicKeys[0] = make([][]byte, 1)
	validatorPublicKeys[0][0] = []byte("testpk")

	shardCoordinatorMock := mock.NewOneShardCoordinatorMock()
	peerAdapter := getAccountsMock()
	peerAdapter.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return &mock.PeerAccountHandlerMock{
			DecreaseLeaderSuccessRateCalled: func(value uint32) {
				decreaseLeaderCalls++
			},
			DecreaseValidatorSuccessRateCalled: func(value uint32) {
				decreaseValidatorCalls++
			},
			SetTempRatingCalled: func(value uint32) {
				setTempRatingCalls++
			},
		}, nil
	}

	arguments := createMockArguments()
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, _ uint32) (validatorsGroup []sharding.Validator, err error) {
			return []sharding.Validator{
				&mock.ValidatorMock{},
			}, nil
		},
		GetAllEligibleValidatorsPublicKeysCalled: func() (map[uint32][][]byte, error) {
			return validatorPublicKeys, nil
		},
		GetValidatorWithPublicKeyCalled: func(publicKey []byte, _ uint32) (sharding.Validator, uint32, error) {
			validator, _ := sharding.NewValidator(publicKey, publicKey)
			return validator, 0, nil
		},
	}
	arguments.ShardCoordinator = shardCoordinatorMock
	arguments.AdrConv = &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (addressContainer state.AddressContainer, e error) {
			return nil, nil
		},
	}
	arguments.PeerAdapter = peerAdapter
	arguments.Rater = mock.GetNewMockRater()
	arguments.MaxComputableRounds = 5

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)
	_ = validatorStatistics.CheckForMissedBlocks(uint64(currentHeaderRound), uint64(previousHeaderRound), []byte("prev"), 0, 0)
	assert.Equal(t, 1, decreaseLeaderCalls)
	assert.Equal(t, 1, decreaseValidatorCalls)
	assert.Equal(t, 2, setTempRatingCalls)
}

func TestValidatorStatisticsProcessor_CheckForMissedBlocksWithRoundDifferenceGreaterThanMaxComputableCallsOnlyOnce(t *testing.T) {
	t.Parallel()

	currentHeaderRound := 20
	previousHeaderRound := 10
	decreaseValidatorCalls := 0
	decreaseLeaderCalls := 0
	setTempRatingCalls := 0
	nrValidators := 1

	validatorPublicKeys := make(map[uint32][][]byte)
	validatorPublicKeys[0] = make([][]byte, nrValidators)
	for i := 0; i < nrValidators; i++ {
		validatorPublicKeys[0][i] = []byte(fmt.Sprintf("testpk_%v", i))
	}

	shardCoordinatorMock := mock.NewOneShardCoordinatorMock()
	peerAdapter := getAccountsMock()
	peerAdapter.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return &mock.PeerAccountHandlerMock{
			DecreaseLeaderSuccessRateCalled: func(value uint32) {
				decreaseLeaderCalls++
			},
			DecreaseValidatorSuccessRateCalled: func(value uint32) {
				decreaseValidatorCalls++
			},
			SetTempRatingCalled: func(value uint32) {
				setTempRatingCalls++
			},
		}, nil
	}

	arguments := createMockArguments()
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, _ uint32) (validatorsGroup []sharding.Validator, err error) {
			return []sharding.Validator{
				&mock.ValidatorMock{},
			}, nil
		},
		GetAllEligibleValidatorsPublicKeysCalled: func() (map[uint32][][]byte, error) {
			return validatorPublicKeys, nil
		},
		GetValidatorWithPublicKeyCalled: func(publicKey []byte, _ uint32) (sharding.Validator, uint32, error) {
			validator, _ := sharding.NewValidator(publicKey, publicKey)
			return validator, 0, nil
		},
	}
	arguments.ShardCoordinator = shardCoordinatorMock
	arguments.AdrConv = &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (addressContainer state.AddressContainer, e error) {
			return nil, nil
		},
	}
	arguments.PeerAdapter = peerAdapter
	arguments.Rater = mock.GetNewMockRater()
	arguments.MaxComputableRounds = 5

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)
	_ = validatorStatistics.CheckForMissedBlocks(uint64(currentHeaderRound), uint64(previousHeaderRound), []byte("prev"), 0, 0)
	assert.Equal(t, 1, decreaseLeaderCalls)
	assert.Equal(t, 1, decreaseValidatorCalls)
	assert.Equal(t, 2, setTempRatingCalls)
}

func TestValidatorStatisticsProcessor_CheckForMissedBlocksWithRoundDifferences(t *testing.T) {
	t.Parallel()

	currentHeaderRound := uint64(101)
	previousHeaderRound := uint64(1)
	maxComputableRounds := uint64(5)

	type args struct {
		currentHeaderRound  uint64
		previousHeaderRound uint64
		maxComputableRounds uint64
		nrValidators        int
		consensusGroupSize  int
	}

	type result struct {
		decreaseValidatorValue uint32
		decreaseLeaderValue    uint32
		tempRating             uint32
	}

	type testSuite struct {
		name string
		args args
		want result
	}

	rater := mock.GetNewMockRater()
	rater.StartRating = 500000
	rater.MinRating = 100000
	rater.MaxRating = 1000000
	rater.DecreaseProposer = -2000
	rater.DecreaseValidator = -10

	validators := []struct {
		validators    int
		consensusSize int
	}{
		{validators: 1, consensusSize: 1},
		{validators: 2, consensusSize: 1},
		{validators: 10, consensusSize: 1},
		{validators: 100, consensusSize: 1},
		{validators: 400, consensusSize: 1},
		{validators: 400, consensusSize: 2},
		{validators: 400, consensusSize: 10},
		{validators: 400, consensusSize: 63},
		{validators: 400, consensusSize: 400},
	}

	tests := make([]testSuite, len(validators))

	for i, nodes := range validators {
		{
			leaderProbability := computeLeaderProbability(currentHeaderRound, previousHeaderRound, nodes.validators)
			intValidatorProbability := uint32(leaderProbability*float64(nodes.consensusSize) + 1 - math.SmallestNonzeroFloat64)
			intLeaderProbability := uint32(leaderProbability + 1 - math.SmallestNonzeroFloat64)

			tests[i] = testSuite{
				args: args{
					currentHeaderRound:  currentHeaderRound,
					previousHeaderRound: previousHeaderRound,
					maxComputableRounds: maxComputableRounds,
					nrValidators:        nodes.validators,
					consensusGroupSize:  nodes.consensusSize,
				},
				want: result{
					decreaseValidatorValue: intValidatorProbability,
					decreaseLeaderValue:    intLeaderProbability,
					tempRating: uint32(int32(rater.StartRating) +
						int32(intLeaderProbability)*rater.DecreaseProposer +
						int32(intValidatorProbability)*rater.DecreaseValidator),
				},
			}
		}
	}

	for _, tt := range tests {
		ttCopy := tt
		t.Run(tt.name, func(t *testing.T) {
			decreaseLeader, decreaseValidator, rating := DoComputeMissingBlocks(
				rater,
				tt.args.nrValidators,
				tt.args.consensusGroupSize,
				tt.args.currentHeaderRound,
				tt.args.previousHeaderRound,
				tt.args.maxComputableRounds)

			res := result{
				decreaseValidatorValue: decreaseValidator,
				decreaseLeaderValue:    decreaseLeader,
				tempRating:             rating,
			}

			if res != ttCopy.want {
				t.Errorf("ComputeMissingBlocks = %v, want %v", res, ttCopy.want)
			}

			t.Logf("validators:%v, consensusSize:%v, missedRounds: %v, decreased leader: %v, decreased validator: %v, startRating: %v, endRating: %v",
				ttCopy.args.nrValidators,
				ttCopy.args.consensusGroupSize,
				ttCopy.args.currentHeaderRound-ttCopy.args.previousHeaderRound,
				ttCopy.want.decreaseLeaderValue,
				ttCopy.want.decreaseValidatorValue,
				rater.StartRating,
				ttCopy.want.tempRating,
			)

		})
	}

}

func computeLeaderProbability(
	currentHeaderRound uint64,
	previousHeaderRound uint64,
	validators int,
) float64 {
	return (float64(currentHeaderRound) - float64(previousHeaderRound) - 1) / float64(validators)
}

func DoComputeMissingBlocks(
	rater *mock.RaterMock,
	nrValidators int,
	consensusGroupSize int,
	currentHeaderRounds uint64,
	previousHeaderRound uint64,
	maxComputableRounds uint64,
) (uint32, uint32, uint32) {
	validatorPublicKeys := make(map[uint32][][]byte)
	validatorPublicKeys[0] = make([][]byte, nrValidators)
	for i := 0; i < nrValidators; i++ {
		validatorPublicKeys[0][i] = []byte(fmt.Sprintf("testpk_%v", i))
	}

	consensus := make([]sharding.Validator, consensusGroupSize)
	for i := 0; i < consensusGroupSize; i++ {
		consensus[i] = &mock.ValidatorMock{}
	}

	accountsMap := make(map[string]*mock.PeerAccountHandlerMock)
	leaderSuccesRateMap := make(map[string]uint32)
	validatorSuccesRateMap := make(map[string]uint32)
	ratingMap := make(map[string]uint32)

	shardCoordinatorMock := mock.NewOneShardCoordinatorMock()
	peerAdapter := getAccountsMock()
	peerAdapter.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		key := string(addressContainer.Bytes())
		account, found := accountsMap[key]

		if !found {
			account = &mock.PeerAccountHandlerMock{
				DecreaseLeaderSuccessRateCalled: func(value uint32) {
					leaderSuccesRateMap[key] += value
				},
				DecreaseValidatorSuccessRateCalled: func(value uint32) {
					validatorSuccesRateMap[key] += value
				},
				GetTempRatingCalled: func() uint32 {
					return ratingMap[key]
				},
				SetTempRatingCalled: func(value uint32) {
					ratingMap[key] = value
				},
			}
			accountsMap[key] = account
			leaderSuccesRateMap[key] = 0
			validatorSuccesRateMap[key] = 0
			ratingMap[key] = rater.StartRating
		}

		return account, nil
	}

	arguments := createMockArguments()
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, _ uint32) (validatorsGroup []sharding.Validator, err error) {
			return consensus, nil
		},
		GetAllEligibleValidatorsPublicKeysCalled: func() (map[uint32][][]byte, error) {
			return validatorPublicKeys, nil
		},
		ConsensusGroupSizeCalled: func(uint32) int {
			return consensusGroupSize
		},
		GetValidatorWithPublicKeyCalled: func(publicKey []byte, _ uint32) (sharding.Validator, uint32, error) {
			validator, _ := sharding.NewValidator(publicKey, publicKey)
			return validator, 0, nil
		},
	}
	arguments.ShardCoordinator = shardCoordinatorMock
	arguments.AdrConv = &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (addressContainer state.AddressContainer, e error) {
			return mock.NewAddressMock(pubKey), nil
		},
	}
	arguments.PeerAdapter = peerAdapter
	arguments.Rater = rater

	arguments.MaxComputableRounds = maxComputableRounds

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)
	_ = validatorStatistics.CheckForMissedBlocks(currentHeaderRounds, previousHeaderRound, []byte("prev"), 0, 0)

	firstKey := "testpk_0"

	return leaderSuccesRateMap[firstKey], validatorSuccesRateMap[firstKey], ratingMap[firstKey]
}

func TestValidatorStatisticsProcessor_GetMatchingPrevShardDataEmptySDReturnsNil(t *testing.T) {
	arguments := createMockArguments()

	currentShardData := block.ShardData{}
	shardInfo := make([]block.ShardData, 0)

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)
	sd := validatorStatistics.GetMatchingPrevShardData(currentShardData, shardInfo)

	assert.Nil(t, sd)
}

func TestValidatorStatisticsProcessor_GetMatchingPrevShardDataNoMatch(t *testing.T) {
	arguments := createMockArguments()

	currentShardData := block.ShardData{ShardID: 1, Nonce: 10}
	shardInfo := []block.ShardData{{ShardID: 1, Nonce: 8}, {ShardID: 2, Nonce: 9}}

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)
	sd := validatorStatistics.GetMatchingPrevShardData(currentShardData, shardInfo)

	assert.Nil(t, sd)
}

func TestValidatorStatisticsProcessor_GetMatchingPrevShardDataFindsMatch(t *testing.T) {
	arguments := createMockArguments()

	currentShardData := block.ShardData{ShardID: 1, Nonce: 10}
	shardInfo := []block.ShardData{{ShardID: 1, Nonce: 9}, {ShardID: 2, Nonce: 9}}

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)
	sd := validatorStatistics.GetMatchingPrevShardData(currentShardData, shardInfo)

	assert.NotNil(t, sd)
}

func TestValidatorStatisticsProcessor_UpdatePeerStateCallsPubKeyForValidator(t *testing.T) {
	pubKeyCalled := false
	addressCalled := false
	arguments := createMockArguments()
	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validatorsGroup []sharding.Validator, err error) {
			return []sharding.Validator{&mock.ValidatorMock{
				PubKeyCalled: func() []byte {
					pubKeyCalled = true
					return make([]byte, 0)
				},
				AddressCalled: func() []byte {
					addressCalled = true
					return make([]byte, 0)
				},
			}, &mock.ValidatorMock{}}, nil
		},
	}
	arguments.DataPool = &mock.PoolsHolderStub{
		HeadersCalled: func() dataRetriever.HeadersPool {
			return &mock.HeadersCacherStub{
				GetHeaderByHashCalled: func(hash []byte) (handler data.HeaderHandler, e error) {
					return getMetaHeaderHandler([]byte("header")), nil
				},
			}
		},
	}

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)
	header := getMetaHeaderHandler([]byte("header"))

	_, _ = validatorStatistics.UpdatePeerState(header, createMockCache())

	assert.True(t, pubKeyCalled)
	assert.False(t, addressCalled)
}

func getMetaHeaderHandler(randSeed []byte) *block.MetaBlock {
	return &block.MetaBlock{
		Nonce:           2,
		PrevRandSeed:    randSeed,
		PrevHash:        randSeed,
		PubKeysBitmap:   randSeed,
		AccumulatedFees: big.NewInt(0),
	}
}

func getAccountsMock() *mock.AccountsStub {
	return &mock.AccountsStub{
		CommitCalled: func() (bytes []byte, e error) {
			return make([]byte, 0), nil
		},
		LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
			return &mock.AccountWrapMock{}, nil
		},
	}
}

func createCustomArgumentsForSaveInitialState() (peer.ArgValidatorStatisticsProcessor, map[uint32][]sharding.Validator, map[uint32][]sharding.Validator, map[string]int) {
	arguments := createMockArguments()

	shardZeroId := uint32(0)
	eligibleValidatorsPubKeys := map[uint32][][]byte{
		shardZeroId:           {[]byte("e_pk0_shard0"), []byte("e_pk1_shard0")},
		core.MetachainShardId: {[]byte("e_pk0_meta"), []byte("e_pk1_meta")},
	}

	waitingValidatorsPubKeys := map[uint32][][]byte{
		shardZeroId:           {[]byte("w_pk0_shard0"), []byte("w_pk1_shard0")},
		core.MetachainShardId: {[]byte("w_pk0_meta"), []byte("w_pk1_meta")},
	}

	waitingMap := make(map[uint32][]sharding.Validator)
	waitingMap[core.MetachainShardId] = []sharding.Validator{
		mock.NewValidatorMock(eligibleValidatorsPubKeys[core.MetachainShardId][0], []byte("e_addr0_meta")),
		mock.NewValidatorMock(eligibleValidatorsPubKeys[core.MetachainShardId][1], []byte("e_addr1_meta")),
	}
	waitingMap[shardZeroId] = []sharding.Validator{
		mock.NewValidatorMock(eligibleValidatorsPubKeys[shardZeroId][0], []byte("e_addr0_shard0")),
		mock.NewValidatorMock(eligibleValidatorsPubKeys[shardZeroId][1], []byte("e_addr1_shard0")),
	}

	eligibleMap := make(map[uint32][]sharding.Validator)
	eligibleMap[core.MetachainShardId] = []sharding.Validator{
		mock.NewValidatorMock(waitingValidatorsPubKeys[core.MetachainShardId][0], []byte("w_addr0_meta")),
		mock.NewValidatorMock(waitingValidatorsPubKeys[core.MetachainShardId][1], []byte("w_addr1_meta")),
	}
	eligibleMap[shardZeroId] = []sharding.Validator{
		mock.NewValidatorMock(waitingValidatorsPubKeys[shardZeroId][0], []byte("w_addr0_shard0")),
		mock.NewValidatorMock(waitingValidatorsPubKeys[shardZeroId][1], []byte("w_addr1_shard0")),
	}

	arguments.NodesCoordinator = &mock.NodesCoordinatorMock{
		GetValidatorWithPublicKeyCalled: func(publicKey []byte, epoch uint32) (validator sharding.Validator, shardId uint32, err error) {
			for shardId, validators := range eligibleMap {
				for _, val := range validators {
					if bytes.Equal(val.PubKey(), publicKey) {
						return val, shardId, nil
					}
				}
			}

			for shardId, validators := range waitingMap {
				for _, val := range validators {
					if bytes.Equal(val.PubKey(), publicKey) {
						return val, shardId, nil
					}
				}
			}

			return nil, 0, nil
		},
		GetAllEligibleValidatorsPublicKeysCalled: func() (m map[uint32][][]byte, err error) {
			return eligibleValidatorsPubKeys, nil
		},
		GetAllWaitingValidatorsPublicKeysCalled: func() (m map[uint32][][]byte, err error) {
			return waitingValidatorsPubKeys, nil
		},
	}

	actualMap := make(map[string]int)

	peerAdapter := &mock.AccountsStub{
		LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
			peerAccount, _ := state.NewPeerAccount(addressContainer)
			return peerAccount, nil
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			actualMap[string(accountHandler.AddressContainer().Bytes())]++
			return nil
		},
		CommitCalled: func() (bytes []byte, e error) {
			return nil, nil
		},
	}

	addressConverter := &mock.AddressConverterStub{
		CreateAddressFromHexCalled: func(hexAddress string) (container state.AddressContainer, e error) {
			return mock.NewAddressMock([]byte(hexAddress)), nil
		},
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (container state.AddressContainer, err error) {
			return mock.NewAddressMock(pubKey), nil
		},
	}

	arguments.PeerAdapter = peerAdapter
	arguments.AdrConv = addressConverter
	return arguments, waitingMap, eligibleMap, actualMap
}

func TestValidatorStatistics_RootHashWithErrShouldReturnNil(t *testing.T) {
	hash := []byte("nonExistingRootHash")
	expectedErr := errors.New("Invalid rootHash")

	arguments := createMockArguments()

	peerAdapter := getAccountsMock()
	peerAdapter.GetAllLeavesCalled = func(rootHash []byte) (m map[string][]byte, err error) {
		return nil, expectedErr
	}
	arguments.PeerAdapter = peerAdapter

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)

	validatorInfos, err := validatorStatistics.GetValidatorInfoForRootHash(hash)
	assert.Nil(t, validatorInfos)
	assert.Equal(t, expectedErr, err)
}

func TestValidatorStatistics_ResetValidatorStatisticsAtNewEpoch(t *testing.T) {
	hash := []byte("correctRootHash")
	expectedErr := errors.New("unknown peer")
	arguments := createMockArguments()

	addrBytes0 := []byte("addr1")
	addrBytesMeta := []byte("addrM")

	pa0, _ := createPeerAccounts(addrBytes0, addrBytesMeta)

	marshalizedPa0, _ := arguments.Marshalizer.Marshal(pa0)

	validatorInfoMap := make(map[string][]byte)
	validatorInfoMap[string(addrBytes0)] = marshalizedPa0

	peerAdapter := getAccountsMock()
	peerAdapter.GetAllLeavesCalled = func(rootHash []byte) (m map[string][]byte, err error) {
		if bytes.Equal(rootHash, hash) {
			return validatorInfoMap, nil
		}
		return nil, expectedErr
	}
	peerAdapter.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, err error) {
		if bytes.Equal(pa0.GetBLSPublicKey(), addressContainer.Bytes()) {
			return pa0, nil
		}
		return nil, expectedErr
	}
	arguments.PeerAdapter = peerAdapter
	arguments.AdrConv = mock.NewAddressConverterFake(4, "")
	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)
	validatorInfos, _ := validatorStatistics.GetValidatorInfoForRootHash(hash)

	assert.NotEqual(t, pa0.GetTempRating(), pa0.GetRating())

	err := validatorStatistics.ResetValidatorStatisticsAtNewEpoch(validatorInfos)

	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(0), pa0.GetAccumulatedFees())
	assert.Equal(t, uint32(0), pa0.GetLeaderSuccessRate().NrSuccess)
	assert.Equal(t, uint32(0), pa0.GetLeaderSuccessRate().NrFailure)
	assert.Equal(t, uint32(0), pa0.GetValidatorSuccessRate().NrSuccess)
	assert.Equal(t, uint32(0), pa0.GetValidatorSuccessRate().NrFailure)
	assert.Equal(t, uint32(0), pa0.GetNumSelectedInSuccessBlocks())
	assert.Equal(t, pa0.GetTempRating(), pa0.GetRating())
}

func TestValidatorStatistics_Process(t *testing.T) {
	hash := []byte("correctRootHash")
	expectedErr := errors.New("error rootHash")
	arguments := createMockArguments()

	addrBytes0 := []byte("addr1")
	addrBytesMeta := []byte("addrMeta")

	pa0, paMeta := createPeerAccounts(addrBytes0, addrBytesMeta)

	marshalizedPa0, _ := arguments.Marshalizer.Marshal(pa0)
	marshalizedPaMeta, _ := arguments.Marshalizer.Marshal(paMeta)

	validatorInfoMap := make(map[string][]byte)
	validatorInfoMap[string(addrBytes0)] = marshalizedPa0
	validatorInfoMap[string(addrBytesMeta)] = marshalizedPaMeta

	peerAdapter := getAccountsMock()
	peerAdapter.GetAllLeavesCalled = func(rootHash []byte) (m map[string][]byte, err error) {
		if bytes.Equal(rootHash, hash) {
			return validatorInfoMap, nil
		}
		return nil, expectedErr
	}
	peerAdapter.LoadAccountCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, err error) {
		if bytes.Equal(pa0.GetBLSPublicKey(), addressContainer.Bytes()) {
			return pa0, nil
		}
		return nil, expectedErr
	}
	arguments.PeerAdapter = peerAdapter

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)
	validatorInfos, _ := validatorStatistics.GetValidatorInfoForRootHash(hash)
	vi0 := validatorInfos[0][0]
	newTempRating := uint32(25)
	vi0.TempRating = newTempRating

	assert.NotEqual(t, newTempRating, pa0.GetRating())

	err := validatorStatistics.Process(vi0)

	assert.Nil(t, err)
	assert.Equal(t, newTempRating, pa0.GetRating())
}

func TestValidatorStatistics_GetValidatorInfoForRootHash(t *testing.T) {
	hash := []byte("correctRootHash")
	expectedErr := errors.New("error rootHash")
	arguments := createMockArguments()

	addrBytes0 := []byte("addr1")
	addrBytesMeta := []byte("addrMeta")

	pa0, paMeta := createPeerAccounts(addrBytes0, addrBytesMeta)

	marshalizedPa0, _ := arguments.Marshalizer.Marshal(pa0)
	marshalizedPaMeta, _ := arguments.Marshalizer.Marshal(paMeta)

	validatorInfoMap := make(map[string][]byte)
	validatorInfoMap[string(addrBytes0)] = marshalizedPa0
	validatorInfoMap[string(addrBytesMeta)] = marshalizedPaMeta

	peerAdapter := getAccountsMock()
	peerAdapter.GetAllLeavesCalled = func(rootHash []byte) (m map[string][]byte, err error) {
		if bytes.Equal(rootHash, hash) {
			return validatorInfoMap, nil
		}
		return nil, expectedErr
	}
	arguments.PeerAdapter = peerAdapter

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)

	validatorInfos, err := validatorStatistics.GetValidatorInfoForRootHash(hash)
	assert.NotNil(t, validatorInfos)
	assert.Nil(t, err)
	assert.Equal(t, uint32(0), validatorInfos[0][0].ShardId)
	compare(t, pa0, validatorInfos[0][0])
	assert.Equal(t, core.MetachainShardId, validatorInfos[core.MetachainShardId][0].ShardId)
	compare(t, paMeta, validatorInfos[core.MetachainShardId][0])
}

func TestValidatorStatistics_ProcessValidatorInfosEndOfEpochWithNilMapShouldErr(t *testing.T) {
	arguments := createMockArguments()
	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)

	err := validatorStatistics.ProcessRatingsEndOfEpoch(nil)
	assert.Equal(t, process.ErrNilValidatorInfos, err)

	vi := make(map[uint32][]*state.ValidatorInfo)
	err = validatorStatistics.ProcessRatingsEndOfEpoch(vi)
	assert.Equal(t, process.ErrNilValidatorInfos, err)
}

func TestValidatorStatistics_ProcessValidatorInfosEndOfEpochWithNoValidatorFailureShouldNotChangeTempRating(t *testing.T) {
	arguments := createMockArguments()
	rater := createMockRater()
	rater.GetSignedBlocksThresholdCalled = func() float32 {
		return 0.025
	}
	arguments.Rater = rater

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)

	tempRating1 := uint32(75)
	tempRating2 := uint32(80)

	vi := make(map[uint32][]*state.ValidatorInfo)
	vi[core.MetachainShardId] = make([]*state.ValidatorInfo, 1)
	vi[core.MetachainShardId][0] = &state.ValidatorInfo{
		PublicKey:                  nil,
		ShardId:                    core.MetachainShardId,
		List:                       "",
		Index:                      0,
		TempRating:                 tempRating1,
		Rating:                     0,
		RewardAddress:              nil,
		LeaderSuccess:              10,
		LeaderFailure:              0,
		ValidatorSuccess:           10,
		ValidatorFailure:           0,
		NumSelectedInSuccessBlocks: 20,
		AccumulatedFees:            nil,
	}

	vi[0] = make([]*state.ValidatorInfo, 1)
	vi[0][0] = &state.ValidatorInfo{
		PublicKey:                  nil,
		ShardId:                    core.MetachainShardId,
		List:                       "",
		Index:                      0,
		TempRating:                 tempRating2,
		Rating:                     0,
		RewardAddress:              nil,
		LeaderSuccess:              10,
		LeaderFailure:              0,
		ValidatorSuccess:           10,
		ValidatorFailure:           0,
		NumSelectedInSuccessBlocks: 20,
		AccumulatedFees:            nil,
	}

	err := validatorStatistics.ProcessRatingsEndOfEpoch(vi)
	assert.Nil(t, err)
	assert.Equal(t, tempRating1, vi[core.MetachainShardId][0].TempRating)
	assert.Equal(t, tempRating2, vi[0][0].TempRating)
}

func TestValidatorStatistics_ProcessValidatorInfosEndOfEpochWithSmallValidatorFailureShouldWork(t *testing.T) {
	arguments := createMockArguments()
	rater := createMockRater()
	rater.GetSignedBlocksThresholdCalled = func() float32 {
		return 0.025
	}
	rater.MinRating = 1000
	rater.MaxRating = 10000
	arguments.Rater = rater

	updateArgumetsWithNeeded(arguments)

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)

	tempRating1 := uint32(5000)
	tempRating2 := uint32(8000)

	validatorSuccess1 := uint32(2)
	validatorFailure1 := uint32(98)
	validatorSuccess2 := uint32(1)
	validatorFailure2 := uint32(99)

	vi := make(map[uint32][]*state.ValidatorInfo)
	vi[core.MetachainShardId] = make([]*state.ValidatorInfo, 1)
	vi[core.MetachainShardId][0] = createMockValidatorInfo(core.MetachainShardId, tempRating1, validatorSuccess1, validatorFailure1)
	vi[0] = make([]*state.ValidatorInfo, 1)
	vi[0][0] = createMockValidatorInfo(0, tempRating2, validatorSuccess2, validatorFailure2)

	err := validatorStatistics.ProcessRatingsEndOfEpoch(vi)
	assert.Nil(t, err)
	expectedTempRating1 := tempRating1 - uint32(rater.MetaIncreaseValidator)*validatorFailure1
	assert.Equal(t, expectedTempRating1, vi[core.MetachainShardId][0].TempRating)
	expectedTempRating2 := tempRating2 - uint32(rater.IncreaseValidator)*validatorFailure2
	assert.Equal(t, expectedTempRating2, vi[0][0].TempRating)
}

func TestValidatorStatistics_ProcessValidatorInfosEndOfEpochWithLargeValidatorFailureBelowMinRatingShouldWork(t *testing.T) {
	arguments := createMockArguments()
	rater := createMockRater()
	rater.GetSignedBlocksThresholdCalled = func() float32 {
		return 0.025
	}
	rater.MinRating = 1000
	rater.MaxRating = 10000
	arguments.Rater = rater
	rater.MetaIncreaseValidator = 100
	rater.IncreaseValidator = 99
	updateArgumetsWithNeeded(arguments)

	tempRating1 := uint32(5000)
	tempRating2 := uint32(8000)

	validatorSuccess1 := uint32(2)
	validatorSuccess2 := uint32(1)
	validatorFailure1 := uint32(98)
	validatorFailure2 := uint32(99)

	vi := make(map[uint32][]*state.ValidatorInfo)
	vi[core.MetachainShardId] = make([]*state.ValidatorInfo, 1)
	vi[core.MetachainShardId][0] = createMockValidatorInfo(core.MetachainShardId, tempRating1, validatorSuccess1, validatorFailure1)
	vi[0] = make([]*state.ValidatorInfo, 1)
	vi[0][0] = createMockValidatorInfo(0, tempRating2, validatorSuccess2, validatorFailure2)

	validatorStatistics, _ := peer.NewValidatorStatisticsProcessor(arguments)
	err := validatorStatistics.ProcessRatingsEndOfEpoch(vi)

	assert.Nil(t, err)
	assert.Equal(t, rater.MinRating, vi[core.MetachainShardId][0].TempRating)
	assert.Equal(t, rater.MinRating, vi[0][0].TempRating)
}

func createMockValidatorInfo(shardId uint32, tempRating uint32, validatorSuccess uint32, validatorFailure uint32) *state.ValidatorInfo {
	return &state.ValidatorInfo{
		PublicKey:                  nil,
		ShardId:                    shardId,
		List:                       "",
		Index:                      0,
		TempRating:                 tempRating,
		Rating:                     0,
		RewardAddress:              nil,
		LeaderSuccess:              0,
		LeaderFailure:              0,
		ValidatorSuccess:           validatorSuccess,
		ValidatorFailure:           validatorFailure,
		NumSelectedInSuccessBlocks: validatorSuccess + validatorFailure,
		AccumulatedFees:            nil,
	}
}

func compare(t *testing.T, peerAccount state.PeerAccountHandler, validatorInfo *state.ValidatorInfo) {
	assert.Equal(t, peerAccount.GetCurrentShardId(), validatorInfo.ShardId)
	assert.Equal(t, peerAccount.GetRating(), validatorInfo.Rating)
	assert.Equal(t, peerAccount.GetTempRating(), validatorInfo.TempRating)
	assert.Equal(t, peerAccount.GetBLSPublicKey(), validatorInfo.PublicKey)
	assert.Equal(t, peerAccount.GetValidatorSuccessRate().NrFailure, validatorInfo.ValidatorFailure)
	assert.Equal(t, peerAccount.GetValidatorSuccessRate().NrSuccess, validatorInfo.ValidatorSuccess)
	assert.Equal(t, peerAccount.GetLeaderSuccessRate().NrFailure, validatorInfo.LeaderFailure)
	assert.Equal(t, peerAccount.GetLeaderSuccessRate().NrSuccess, validatorInfo.LeaderSuccess)
	assert.Equal(t, "list", validatorInfo.List)
	assert.Equal(t, uint32(0), validatorInfo.Index)
	assert.Equal(t, peerAccount.GetRewardAddress(), validatorInfo.RewardAddress)
	assert.Equal(t, peerAccount.GetAccumulatedFees(), validatorInfo.AccumulatedFees)
	assert.Equal(t, peerAccount.GetNumSelectedInSuccessBlocks(), validatorInfo.NumSelectedInSuccessBlocks)
}

func createPeerAccounts(addrBytes0 []byte, addrBytesMeta []byte) (state.PeerAccountHandler, state.PeerAccountHandler) {
	addr := mock.NewAddressMock(addrBytes0)
	pa0, _ := state.NewPeerAccount(addr)
	pa0.PeerAccountData = state.PeerAccountData{
		BLSPublicKey:     []byte("bls0"),
		SchnorrPublicKey: []byte("schnorr0"),
		RewardAddress:    []byte("reward0"),
		Stake:            big.NewInt(10),
		AccumulatedFees:  big.NewInt(11),
		JailTime: state.TimePeriod{
			StartTime: state.TimeStamp{Epoch: 1, Round: 10},
			EndTime:   state.TimeStamp{Epoch: 2, Round: 2},
		},
		PastJailTimes: []state.TimePeriod{
			{
				StartTime: state.TimeStamp{Epoch: 1, Round: 1},
				EndTime:   state.TimeStamp{Epoch: 1, Round: 2},
			},
		},
		CurrentShardId:    0,
		NextShardId:       1,
		NodeInWaitingList: false,
		UnStakedNonce:     7,
		ValidatorSuccessRate: state.SignRate{
			NrSuccess: 1,
			NrFailure: 2,
		},
		LeaderSuccessRate: state.SignRate{
			NrSuccess: 3,
			NrFailure: 4,
		},
		NumSelectedInSuccessBlocks: 5,
		Rating:                     51,
		TempRating:                 61,
		Nonce:                      7,
	}

	addr = mock.NewAddressMock(addrBytesMeta)
	paMeta, _ := state.NewPeerAccount(addr)
	paMeta.PeerAccountData = state.PeerAccountData{
		BLSPublicKey:     []byte("blsM"),
		SchnorrPublicKey: []byte("schnorrM"),
		RewardAddress:    []byte("rewardM"),
		Stake:            big.NewInt(110),
		AccumulatedFees:  big.NewInt(111),
		JailTime: state.TimePeriod{
			StartTime: state.TimeStamp{Epoch: 11, Round: 101},
			EndTime:   state.TimeStamp{Epoch: 21, Round: 21},
		},
		PastJailTimes: []state.TimePeriod{
			{
				StartTime: state.TimeStamp{Epoch: 11, Round: 11},
				EndTime:   state.TimeStamp{Epoch: 11, Round: 12},
			},
		},
		CurrentShardId:    core.MetachainShardId,
		NextShardId:       1,
		NodeInWaitingList: true,
		UnStakedNonce:     2,
		ValidatorSuccessRate: state.SignRate{
			NrSuccess: 11,
			NrFailure: 21,
		},
		LeaderSuccessRate: state.SignRate{
			NrSuccess: 31,
			NrFailure: 41,
		},
		NumSelectedInSuccessBlocks: 3,
		Rating:                     511,
		TempRating:                 611,
		Nonce:                      8,
	}
	return pa0, paMeta
}

func updateArgumetsWithNeeded(arguments peer.ArgValidatorStatisticsProcessor) {
	addrBytes0 := []byte("addr1")
	addrBytesMeta := []byte("addrMeta")

	pa0, paMeta := createPeerAccounts(addrBytes0, addrBytesMeta)

	marshalizedPa0, _ := arguments.Marshalizer.Marshal(pa0)
	marshalizedPaMeta, _ := arguments.Marshalizer.Marshal(paMeta)

	validatorInfoMap := make(map[string][]byte)
	validatorInfoMap[string(addrBytes0)] = marshalizedPa0
	validatorInfoMap[string(addrBytesMeta)] = marshalizedPaMeta
	peerAdapter := getAccountsMock()
	peerAdapter.GetAllLeavesCalled = func(rootHash []byte) (m map[string][]byte, err error) {
		return validatorInfoMap, nil
	}
	peerAdapter.GetAccountWithJournalCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, err error) {
		return pa0, nil
	}
	arguments.PeerAdapter = peerAdapter
}
