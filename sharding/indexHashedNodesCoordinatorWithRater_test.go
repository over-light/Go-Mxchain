package sharding_test

import (
	"fmt"
	"math/big"
	"math/rand"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/hashing/blake2b"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/sharding/mock"
	"github.com/stretchr/testify/assert"
)

func createArguments() sharding.ArgNodesCoordinator {
	nbShards := uint32(1)
	eligibleMap := createDummyNodesMap(10, nbShards, "eligible")
	waitingMap := createDummyNodesMap(3, nbShards, "waiting")
	nodeShuffler := &mock.NodeShufflerMock{}
	epochStartSubscriber := &mock.EpochStartNotifierStub{}

	arguments := sharding.ArgNodesCoordinator{
		ShardConsensusGroupSize: 1,
		MetaConsensusGroupSize:  1,
		Hasher:                  &mock.HasherMock{},
		Shuffler:                nodeShuffler,
		EpochStartSubscriber:    epochStartSubscriber,
		NbShards:                nbShards,
		EligibleNodes:           eligibleMap,
		WaitingNodes:            waitingMap,
		SelfPublicKey:           []byte("test"),
	}
	return arguments
}

func TestNewIndexHashedNodesCoordinatorWithRater_NilRaterShouldErr(t *testing.T) {
	nc, _ := sharding.NewIndexHashedNodesCoordinator(createArguments())
	ihgs, err := sharding.NewIndexHashedNodesCoordinatorWithRater(nc, nil)

	assert.Nil(t, ihgs)
	assert.Equal(t, sharding.ErrNilRater, err)
}

func TestNewIndexHashedNodesCoordinatorWithRater_NilNodesCoordinatorShouldErr(t *testing.T) {
	ihgs, err := sharding.NewIndexHashedNodesCoordinatorWithRater(nil, &mock.RaterMock{})

	assert.Nil(t, ihgs)
	assert.Equal(t, sharding.ErrNilNodesCoordinator, err)
}

func TestNewIndexHashedGroupSelectorWithRater_OkValsShouldWork(t *testing.T) {
	t.Parallel()

	nc, _ := sharding.NewIndexHashedNodesCoordinator(createArguments())
	ihgs, err := sharding.NewIndexHashedNodesCoordinatorWithRater(nc, &mock.RaterMock{})
	assert.NotNil(t, ihgs)
	assert.Nil(t, err)
}

//------- LoadEligibleList

func TestIndexHashedGroupSelectorWithRater_SetNilEligibleMapShouldErr(t *testing.T) {
	t.Parallel()
	waiting := createDummyNodesMap(2, 1, "waiting")
	nc, _ := sharding.NewIndexHashedNodesCoordinator(createArguments())
	ihgs, _ := sharding.NewIndexHashedNodesCoordinatorWithRater(nc, &mock.RaterMock{})
	assert.Equal(t, sharding.ErrNilInputNodesMap, ihgs.SetNodesPerShards(nil, waiting, 0))
}

func TestIndexHashedGroupSelectorWithRater_OkValShouldWork(t *testing.T) {
	t.Parallel()

	eligibleMap := createDummyNodesMap(3, 1, "waiting")
	waitingMap := make(map[uint32][]sharding.Validator)
	nodeShuffler := &mock.NodeShufflerMock{}
	epochStartSubscriber := &mock.EpochStartNotifierStub{}

	arguments := sharding.ArgNodesCoordinator{
		ShardConsensusGroupSize: 2,
		MetaConsensusGroupSize:  1,
		Hasher:                  &mock.HasherMock{},
		Shuffler:                nodeShuffler,
		EpochStartSubscriber:    epochStartSubscriber,
		NbShards:                1,
		EligibleNodes:           eligibleMap,
		WaitingNodes:            waitingMap,
		SelfPublicKey:           []byte("test"),
	}
	nc, err := sharding.NewIndexHashedNodesCoordinator(arguments)
	assert.Nil(t, err)

	ihgs, err := sharding.NewIndexHashedNodesCoordinatorWithRater(nc, &mock.RaterMock{})
	assert.Nil(t, err)
	readEligible, _ := ihgs.EligibleList(0)
	assert.Equal(t, eligibleMap[0], readEligible)
}

//------- functionality tests

func TestIndexHashedGroupSelectorWithRater_ComputeValidatorsGroup1ValidatorShouldCallGetRating(t *testing.T) {
	t.Parallel()

	list := []sharding.Validator{
		mock.NewValidatorMock(big.NewInt(1), 2, []byte("pk0"), []byte("addr0")),
	}

	arguments := createArguments()
	arguments.EligibleNodes[0] = list

	raterCalled := false
	rater := &mock.RaterMock{GetRatingCalled: func(string) uint32 {
		raterCalled = true
		return 1
	}}

	nc, _ := sharding.NewIndexHashedNodesCoordinator(arguments)
	ihgs, _ := sharding.NewIndexHashedNodesCoordinatorWithRater(nc, rater)
	list2, err := ihgs.ComputeValidatorsGroup([]byte("randomness"), 0, 0, 0)

	assert.Nil(t, err)
	assert.Equal(t, 1, len(list2))
	assert.Equal(t, true, raterCalled)
}

func TestIndexHashedGroupSelectorWithRater_ComputeExpandedList(t *testing.T) {
	t.Parallel()

	list := []sharding.Validator{
		mock.NewValidatorMock(big.NewInt(1), 2, []byte("pk0"), []byte("addr0")),
		mock.NewValidatorMock(big.NewInt(2), 3, []byte("pk1"), []byte("addr1")),
	}

	listMeta := []sharding.Validator{
		mock.NewValidatorMock(big.NewInt(1), 1, []byte("pkMeta1"), []byte("addrMeta1")),
		mock.NewValidatorMock(big.NewInt(1), 2, []byte("pkMeta2"), []byte("addrMeta2")),
	}

	eligibleMap := make(map[uint32][]sharding.Validator)
	eligibleMap[0] = list
	eligibleMap[core.MetachainShardId] = listMeta
	waitingMap := make(map[uint32][]sharding.Validator)
	nodeShuffler := &mock.NodeShufflerMock{}
	epochStartSubscriber := &mock.EpochStartNotifierStub{}

	arguments := sharding.ArgNodesCoordinator{
		ShardConsensusGroupSize: 2,
		MetaConsensusGroupSize:  1,
		Hasher:                  &mock.HasherMock{},
		Shuffler:                nodeShuffler,
		EpochStartSubscriber:    epochStartSubscriber,
		NbShards:                1,
		EligibleNodes:           eligibleMap,
		WaitingNodes:            waitingMap,
		SelfPublicKey:           []byte("key"),
	}

	ratingPk0 := uint32(5)
	ratingPk1 := uint32(1)
	rater := &mock.RaterMock{GetRatingCalled: func(pk string) uint32 {
		if pk == "pk0" {
			return ratingPk0
		}
		if pk == "pk1" {
			return ratingPk1
		}
		return 1
	}}

	nc, _ := sharding.NewIndexHashedNodesCoordinator(arguments)
	ihgs, _ := sharding.NewIndexHashedNodesCoordinatorWithRater(nc, rater)

	eligibleNodes, mut := ihgs.EligibleList(0)
	expandedList := ihgs.ExpandEligibleList(eligibleNodes, mut)
	assert.Equal(t, int(ratingPk0+ratingPk1), len(expandedList))

	occurences := make(map[string]uint32, 2)
	occurences["pk0"] = 0
	occurences["pk1"] = 0
	for _, validator := range expandedList {
		occurences[string(validator.PubKey())]++
	}

	assert.Equal(t, ratingPk0, occurences["pk0"])
	assert.Equal(t, ratingPk1, occurences["pk1"])
}

func BenchmarkIndexHashedGroupSelectorWithRater_ComputeValidatorsGroup21of400(b *testing.B) {
	consensusGroupSize := 21
	list := make([]sharding.Validator, 0)

	//generate 400 validators
	for i := 0; i < 400; i++ {
		list = append(list, mock.NewValidatorMock(big.NewInt(0), 0, []byte("pk"+strconv.Itoa(i)), []byte("addr"+strconv.Itoa(i))))
	}

	eligibleMap := make(map[uint32][]sharding.Validator)
	waitingMap := make(map[uint32][]sharding.Validator)
	eligibleMap[0] = list
	nodeShuffler := &mock.NodeShufflerMock{}
	epochStartSubscriber := &mock.EpochStartNotifierStub{}

	arguments := sharding.ArgNodesCoordinator{
		ShardConsensusGroupSize: consensusGroupSize,
		MetaConsensusGroupSize:  1,
		Hasher:                  &mock.HasherMock{},
		Shuffler:                nodeShuffler,
		EpochStartSubscriber:    epochStartSubscriber,
		NbShards:                1,
		EligibleNodes:           eligibleMap,
		WaitingNodes:            waitingMap,
		SelfPublicKey:           []byte("key"),
	}
	ihgs, _ := sharding.NewIndexHashedNodesCoordinator(arguments)
	ihgsRater, _ := sharding.NewIndexHashedNodesCoordinatorWithRater(ihgs, &mock.RaterMock{})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		randomness := strconv.Itoa(i)
		list2, _ := ihgsRater.ComputeValidatorsGroup([]byte(randomness), 0, 0, 0)

		assert.Equal(b, consensusGroupSize, len(list2))
	}
}

func TestIndexHashedGroupSelectorWithRater_GetValidatorWithPublicKeyShouldReturnErrNilPubKey(t *testing.T) {
	t.Parallel()

	list := []sharding.Validator{
		mock.NewValidatorMock(big.NewInt(1), 2, []byte("pk0"), []byte("addr0")),
	}
	eligibleMap := make(map[uint32][]sharding.Validator)
	waitingMap := make(map[uint32][]sharding.Validator)
	eligibleMap[0] = list
	nodeShuffler := &mock.NodeShufflerMock{}
	epochStartSubscriber := &mock.EpochStartNotifierStub{}

	arguments := sharding.ArgNodesCoordinator{
		ShardConsensusGroupSize: 1,
		MetaConsensusGroupSize:  1,
		Hasher:                  &mock.HasherMock{},
		Shuffler:                nodeShuffler,
		EpochStartSubscriber:    epochStartSubscriber,
		NbShards:                1,
		EligibleNodes:           eligibleMap,
		WaitingNodes:            waitingMap,
		SelfPublicKey:           []byte("key"),
	}
	nc, _ := sharding.NewIndexHashedNodesCoordinator(arguments)
	ihgs, _ := sharding.NewIndexHashedNodesCoordinatorWithRater(nc, &mock.RaterMock{})

	_, _, err := ihgs.GetValidatorWithPublicKey(nil, 0)
	assert.Equal(t, sharding.ErrNilPubKey, err)
}

func TestIndexHashedGroupSelectorWithRater_GetValidatorWithPublicKeyShouldReturnErrValidatorNotFound(t *testing.T) {
	t.Parallel()

	list := []sharding.Validator{
		mock.NewValidatorMock(big.NewInt(1), 2, []byte("pk0"), []byte("addr0")),
	}

	eligibleMap := make(map[uint32][]sharding.Validator)
	waitingMap := make(map[uint32][]sharding.Validator)
	eligibleMap[0] = list
	nodeShuffler := &mock.NodeShufflerMock{}
	epochStartSubscriber := &mock.EpochStartNotifierStub{}

	arguments := sharding.ArgNodesCoordinator{
		ShardConsensusGroupSize: 1,
		MetaConsensusGroupSize:  1,
		Hasher:                  &mock.HasherMock{},
		Shuffler:                nodeShuffler,
		EpochStartSubscriber:    epochStartSubscriber,
		NbShards:                1,
		EligibleNodes:           eligibleMap,
		WaitingNodes:            waitingMap,
		SelfPublicKey:           []byte("key"),
	}
	nc, _ := sharding.NewIndexHashedNodesCoordinator(arguments)
	ihgs, _ := sharding.NewIndexHashedNodesCoordinatorWithRater(nc, &mock.RaterMock{})

	_, _, err := ihgs.GetValidatorWithPublicKey([]byte("pk1"), 0)
	assert.Equal(t, sharding.ErrValidatorNotFound, err)
}

func TestIndexHashedGroupSelectorWithRater_GetValidatorWithPublicKeyShouldWork(t *testing.T) {
	t.Parallel()

	listMeta := []sharding.Validator{
		mock.NewValidatorMock(big.NewInt(1), 2, []byte("pk0_meta"), []byte("addr0_meta")),
		mock.NewValidatorMock(big.NewInt(1), 2, []byte("pk1_meta"), []byte("addr1_meta")),
		mock.NewValidatorMock(big.NewInt(1), 2, []byte("pk2_meta"), []byte("addr2_meta")),
	}
	listShard0 := []sharding.Validator{
		mock.NewValidatorMock(big.NewInt(1), 2, []byte("pk0_shard0"), []byte("addr0_shard0")),
		mock.NewValidatorMock(big.NewInt(1), 2, []byte("pk1_shard0"), []byte("addr1_shard0")),
		mock.NewValidatorMock(big.NewInt(1), 2, []byte("pk2_shard0"), []byte("addr2_shard0")),
	}
	listShard1 := []sharding.Validator{
		mock.NewValidatorMock(big.NewInt(1), 2, []byte("pk0_shard1"), []byte("addr0_shard1")),
		mock.NewValidatorMock(big.NewInt(1), 2, []byte("pk1_shard1"), []byte("addr1_shard1")),
		mock.NewValidatorMock(big.NewInt(1), 2, []byte("pk2_shard1"), []byte("addr2_shard1")),
	}

	eligibleMap := make(map[uint32][]sharding.Validator)
	waitingMap := make(map[uint32][]sharding.Validator)
	nodeShuffler := &mock.NodeShufflerMock{}
	epochStartSubscriber := &mock.EpochStartNotifierStub{}

	eligibleMap[core.MetachainShardId] = listMeta
	eligibleMap[0] = listShard0
	eligibleMap[1] = listShard1

	arguments := sharding.ArgNodesCoordinator{
		ShardConsensusGroupSize: 1,
		MetaConsensusGroupSize:  1,
		Hasher:                  &mock.HasherMock{},
		Shuffler:                nodeShuffler,
		EpochStartSubscriber:    epochStartSubscriber,
		NbShards:                2,
		EligibleNodes:           eligibleMap,
		WaitingNodes:            waitingMap,
		SelfPublicKey:           []byte("key"),
	}
	nc, _ := sharding.NewIndexHashedNodesCoordinator(arguments)
	ihgs, _ := sharding.NewIndexHashedNodesCoordinatorWithRater(nc, &mock.RaterMock{})

	validator, shardId, err := ihgs.GetValidatorWithPublicKey([]byte("pk0_meta"), 0)
	assert.Nil(t, err)
	assert.Equal(t, core.MetachainShardId, shardId)
	assert.Equal(t, []byte("addr0_meta"), validator.Address())

	validator, shardId, err = ihgs.GetValidatorWithPublicKey([]byte("pk1_shard0"), 0)
	assert.Nil(t, err)
	assert.Equal(t, uint32(0), shardId)
	assert.Equal(t, []byte("addr1_shard0"), validator.Address())

	validator, shardId, err = ihgs.GetValidatorWithPublicKey([]byte("pk2_shard1"), 0)
	assert.Nil(t, err)
	assert.Equal(t, uint32(1), shardId)
	assert.Equal(t, []byte("addr2_shard1"), validator.Address())
}

func TestIndexHashedGroupSelectorWithRater_GetAllValidatorsPublicKeys(t *testing.T) {
	t.Parallel()

	shardZeroId := uint32(0)
	shardOneId := uint32(1)
	expectedValidatorsPubKeys := map[uint32][][]byte{
		shardZeroId:           {[]byte("pk0_shard0"), []byte("pk1_shard0"), []byte("pk2_shard0")},
		shardOneId:            {[]byte("pk0_shard1"), []byte("pk1_shard1"), []byte("pk2_shard1")},
		core.MetachainShardId: {[]byte("pk0_meta"), []byte("pk1_meta"), []byte("pk2_meta")},
	}

	listMeta := []sharding.Validator{
		mock.NewValidatorMock(big.NewInt(1), 2, expectedValidatorsPubKeys[core.MetachainShardId][0], []byte("addr0_meta")),
		mock.NewValidatorMock(big.NewInt(1), 2, expectedValidatorsPubKeys[core.MetachainShardId][1], []byte("addr1_meta")),
		mock.NewValidatorMock(big.NewInt(1), 2, expectedValidatorsPubKeys[core.MetachainShardId][2], []byte("addr2_meta")),
	}
	listShard0 := []sharding.Validator{
		mock.NewValidatorMock(big.NewInt(1), 2, expectedValidatorsPubKeys[shardZeroId][0], []byte("addr0_shard0")),
		mock.NewValidatorMock(big.NewInt(1), 2, expectedValidatorsPubKeys[shardZeroId][1], []byte("addr1_shard0")),
		mock.NewValidatorMock(big.NewInt(1), 2, expectedValidatorsPubKeys[shardZeroId][2], []byte("addr2_shard0")),
	}
	listShard1 := []sharding.Validator{
		mock.NewValidatorMock(big.NewInt(1), 2, expectedValidatorsPubKeys[shardOneId][0], []byte("addr0_shard1")),
		mock.NewValidatorMock(big.NewInt(1), 2, expectedValidatorsPubKeys[shardOneId][1], []byte("addr1_shard1")),
		mock.NewValidatorMock(big.NewInt(1), 2, expectedValidatorsPubKeys[shardOneId][2], []byte("addr2_shard1")),
	}

	eligibleMap := make(map[uint32][]sharding.Validator)
	waitingMap := make(map[uint32][]sharding.Validator)
	nodeShuffler := &mock.NodeShufflerMock{}
	epochStartSubscriber := &mock.EpochStartNotifierStub{}

	eligibleMap[core.MetachainShardId] = listMeta
	eligibleMap[shardZeroId] = listShard0
	eligibleMap[shardOneId] = listShard1

	arguments := sharding.ArgNodesCoordinator{
		ShardConsensusGroupSize: 1,
		MetaConsensusGroupSize:  1,
		Hasher:                  &mock.HasherMock{},
		Shuffler:                nodeShuffler,
		EpochStartSubscriber:    epochStartSubscriber,
		ShardId:                 shardZeroId,
		NbShards:                2,
		EligibleNodes:           eligibleMap,
		WaitingNodes:            waitingMap,
		SelfPublicKey:           []byte("key"),
	}

	nc, _ := sharding.NewIndexHashedNodesCoordinator(arguments)
	ihgs, err := sharding.NewIndexHashedNodesCoordinatorWithRater(nc, &mock.RaterMock{})
	assert.Nil(t, err)

	allValidatorsPublicKeys, err := ihgs.GetAllValidatorsPublicKeys(0)
	assert.Nil(t, err)
	assert.Equal(t, expectedValidatorsPubKeys, allValidatorsPublicKeys)
}

func BenchmarkIndexHashedGroupSelectorWithRater_TestExpandList(b *testing.B) {
	m := runtime.MemStats{}
	runtime.ReadMemStats(&m)

	fmt.Println(m.TotalAlloc)

	nrNodes := 40000
	ratingSteps := 100
	array := make([]int, nrNodes*ratingSteps)
	for i := 0; i < nrNodes; i++ {
		for j := 0; j < ratingSteps; j++ {
			array[i*ratingSteps+j] = i
		}
	}

	//a := []int{1, 2, 3, 4, 5, 6, 7, 8}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(array), func(i, j int) { array[i], array[j] = array[j], array[i] })
	m2 := runtime.MemStats{}

	runtime.ReadMemStats(&m2)

	fmt.Println(m2.TotalAlloc)
	fmt.Println(fmt.Sprintf("Used %d MB", (m2.TotalAlloc-m.TotalAlloc)/1024/1024))
	//fmt.Print(array[0:100])
}

func BenchmarkIndexHashedGroupSelectorWithRater_TestHashes(b *testing.B) {
	nrElementsInList := int64(4000000)
	nrHashes := 100

	hasher := blake2b.Blake2b{}

	randomBits := ""

	for i := 0; i < nrHashes; i++ {
		randomBits = fmt.Sprintf("%s%d", randomBits, rand.Intn(2))
	}
	//computedListIndex := int64(0)
	for i := 0; i < nrHashes; i++ {
		computedHash := hasher.Compute(randomBits + fmt.Sprintf("%d", i))
		computedLargeIndex := big.NewInt(0)
		computedLargeIndex.SetBytes(computedHash)
		fmt.Println(big.NewInt(0).Mod(computedLargeIndex, big.NewInt(nrElementsInList)).Int64())
	}

	//fmt.Print(array[0:100])
}

func BenchmarkIndexHashedWithRaterGroupSelector_ComputeValidatorsGroup21of400(b *testing.B) {
	consensusGroupSize := 21
	list := make([]sharding.Validator, 0)

	//generate 400 validators
	for i := 0; i < 400; i++ {
		list = append(list, mock.NewValidatorMock(big.NewInt(0), 0, []byte("pk"+strconv.Itoa(i)), []byte("addr"+strconv.Itoa(i))))
	}

	eligibleMap := make(map[uint32][]sharding.Validator)
	waitingMap := make(map[uint32][]sharding.Validator)
	eligibleMap[0] = list
	nodeShuffler := &mock.NodeShufflerMock{}
	epochStartSubscriber := &mock.EpochStartNotifierStub{}

	arguments := sharding.ArgNodesCoordinator{
		ShardConsensusGroupSize: consensusGroupSize,
		MetaConsensusGroupSize:  1,
		Hasher:                  &mock.HasherMock{},
		Shuffler:                nodeShuffler,
		EpochStartSubscriber:    epochStartSubscriber,
		NbShards:                1,
		EligibleNodes:           eligibleMap,
		WaitingNodes:            waitingMap,
		SelfPublicKey:           []byte("key"),
	}
	ihgs, _ := sharding.NewIndexHashedNodesCoordinator(arguments)
	ihgsRater, _ := sharding.NewIndexHashedNodesCoordinatorWithRater(ihgs, &mock.RaterMock{})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		randomness := strconv.Itoa(i)
		list2, _ := ihgsRater.ComputeValidatorsGroup([]byte(randomness), 0, 0, 0)

		assert.Equal(b, consensusGroupSize, len(list2))
	}
}
