package metachain

import (
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/epochStart/mock"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/ElrondNetwork/elrond-go/vm/factory"
	"github.com/stretchr/testify/assert"
)

func TestNewEpochStartRewardsCreator_NilShardCoordinator(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	args.ShardCoordinator = nil

	rwd, err := NewEpochStartRewardsCreator(args)

	assert.True(t, check.IfNil(rwd))
	assert.Equal(t, epochStart.ErrNilShardCoordinator, err)
}

func TestNewEpochStartRewardsCreator_NilPubkeyConverter(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	args.PubkeyConverter = nil

	rwd, err := NewEpochStartRewardsCreator(args)

	assert.True(t, check.IfNil(rwd))
	assert.Equal(t, epochStart.ErrNilPubkeyConverter, err)
}

func TestNewEpochStartRewardsCreator_NilRewardsStorage(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	args.RewardsStorage = nil

	rwd, err := NewEpochStartRewardsCreator(args)

	assert.True(t, check.IfNil(rwd))
	assert.Equal(t, epochStart.ErrNilStorage, err)
}

func TestNewEpochStartRewardsCreator_NilMiniBlockStorage(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	args.MiniBlockStorage = nil

	rwd, err := NewEpochStartRewardsCreator(args)

	assert.True(t, check.IfNil(rwd))
	assert.Equal(t, epochStart.ErrNilStorage, err)
}

func TestNewEpochStartRewardsCreator_NilHasher(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	args.Hasher = nil

	rwd, err := NewEpochStartRewardsCreator(args)

	assert.True(t, check.IfNil(rwd))
	assert.Equal(t, epochStart.ErrNilHasher, err)
}

func TestNewEpochStartRewardsCreator_NilMarshalizer(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	args.Marshalizer = nil

	rwd, err := NewEpochStartRewardsCreator(args)

	assert.True(t, check.IfNil(rwd))
	assert.Equal(t, epochStart.ErrNilMarshalizer, err)
}

func TestNewEpochStartRewardsCreator_EmptyCommunityAddress(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	args.CommunityAddress = ""

	rwd, err := NewEpochStartRewardsCreator(args)
	assert.True(t, check.IfNil(rwd))
	assert.Equal(t, epochStart.ErrNilCommunityAddress, err)
}

func TestNewEpochStartRewardsCreator_InvalidCommunityAddress(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	args.CommunityAddress = "xyz" // not a hex string

	rwd, err := NewEpochStartRewardsCreator(args)
	assert.True(t, check.IfNil(rwd))
	assert.NotNil(t, err)
}

func TestNewEpochStartRewardsCreator_OkValsShouldWork(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	rwd, err := NewEpochStartRewardsCreator(args)

	assert.False(t, check.IfNil(rwd))
	assert.Nil(t, err)
}

func TestRewardsCreator_CreateRewardsMiniBlocks(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	rwd, _ := NewEpochStartRewardsCreator(args)

	mb := &block.MetaBlock{
		EpochStart: getDefaultEpochStart(),
	}
	valInfo := make(map[uint32][]*state.ValidatorInfo)
	valInfo[0] = []*state.ValidatorInfo{
		{
			PublicKey:       []byte("pubkey"),
			ShardId:         0,
			AccumulatedFees: big.NewInt(100),
		},
	}
	bdy, err := rwd.CreateRewardsMiniBlocks(mb, valInfo)
	assert.Nil(t, err)
	assert.NotNil(t, bdy)
}

func TestRewardsCreator_VerifyRewardsMiniBlocksHashDoesNotMatch(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	rwd, _ := NewEpochStartRewardsCreator(args)

	bdy := block.MiniBlock{
		TxHashes:        [][]byte{},
		ReceiverShardID: 0,
		SenderShardID:   core.MetachainShardId,
		Type:            block.RewardsBlock,
	}
	mbh := block.MiniBlockHeader{
		Hash:            nil,
		SenderShardID:   core.MetachainShardId,
		ReceiverShardID: 0,
		TxCount:         1,
		Type:            block.RewardsBlock,
	}
	mbHash, _ := core.CalculateHash(&marshal.JsonMarshalizer{}, &mock.HasherMock{}, bdy)
	mbh.Hash = mbHash

	mb := &block.MetaBlock{
		EpochStart: getDefaultEpochStart(),
		MiniBlockHeaders: []block.MiniBlockHeader{
			mbh,
		},
	}
	valInfo := make(map[uint32][]*state.ValidatorInfo)
	valInfo[0] = []*state.ValidatorInfo{
		{
			PublicKey:       []byte("pubkey"),
			ShardId:         0,
			AccumulatedFees: big.NewInt(100),
		},
	}

	err := rwd.VerifyRewardsMiniBlocks(mb, valInfo)
	assert.Equal(t, epochStart.ErrRewardMiniBlockHashDoesNotMatch, err)
}

func TestRewardsCreator_VerifyRewardsMiniBlocksRewardsMbNumDoesNotMatch(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	rwd, _ := NewEpochStartRewardsCreator(args)
	rwdTx := rewardTx.RewardTx{
		Round:   0,
		Value:   big.NewInt(100),
		RcvAddr: []byte{},
		Epoch:   0,
	}
	rwdTxHash, _ := core.CalculateHash(&marshal.JsonMarshalizer{}, &mock.HasherMock{}, rwdTx)

	communityRewardTx := rewardTx.RewardTx{
		Round:   0,
		Value:   big.NewInt(50),
		RcvAddr: []byte{17},
		Epoch:   0,
	}
	commRwdTxHash, _ := core.CalculateHash(&marshal.JsonMarshalizer{}, &mock.HasherMock{}, communityRewardTx)

	bdy := block.MiniBlock{
		TxHashes:        [][]byte{commRwdTxHash, rwdTxHash},
		ReceiverShardID: 0,
		SenderShardID:   core.MetachainShardId,
		Type:            block.RewardsBlock,
	}

	mbh := block.MiniBlockHeader{
		Hash:            nil,
		SenderShardID:   core.MetachainShardId,
		ReceiverShardID: 0,
		TxCount:         2,
		Type:            block.RewardsBlock,
	}
	mbHash, _ := core.CalculateHash(&marshal.JsonMarshalizer{}, &mock.HasherMock{}, bdy)
	mbh.Hash = mbHash

	mb := &block.MetaBlock{
		EpochStart: getDefaultEpochStart(),
		MiniBlockHeaders: []block.MiniBlockHeader{
			mbh,
			mbh,
		},
	}
	valInfo := make(map[uint32][]*state.ValidatorInfo)
	valInfo[0] = []*state.ValidatorInfo{
		{
			PublicKey:       []byte("pubkey"),
			ShardId:         0,
			AccumulatedFees: big.NewInt(100),
			LeaderSuccess:   1,
		},
	}

	err := rwd.VerifyRewardsMiniBlocks(mb, valInfo)
	assert.Equal(t, epochStart.ErrRewardMiniBlocksNumDoesNotMatch, err)
}

func TestRewardsCreator_VerifyRewardsMiniBlocksShouldWork(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	rwd, _ := NewEpochStartRewardsCreator(args)
	rwdTx := rewardTx.RewardTx{
		Round:   0,
		Value:   big.NewInt(100),
		RcvAddr: []byte{},
		Epoch:   0,
	}
	rwdTxHash, _ := core.CalculateHash(&marshal.JsonMarshalizer{}, &mock.HasherMock{}, rwdTx)

	communityRewardTx := rewardTx.RewardTx{
		Round:   0,
		Value:   big.NewInt(50),
		RcvAddr: []byte{17},
		Epoch:   0,
	}
	commRwdTxHash, _ := core.CalculateHash(&marshal.JsonMarshalizer{}, &mock.HasherMock{}, communityRewardTx)

	bdy := block.MiniBlock{
		TxHashes:        [][]byte{commRwdTxHash, rwdTxHash},
		ReceiverShardID: 0,
		SenderShardID:   core.MetachainShardId,
		Type:            block.RewardsBlock,
	}
	mbh := block.MiniBlockHeader{
		Hash:            nil,
		SenderShardID:   core.MetachainShardId,
		ReceiverShardID: 0,
		TxCount:         2,
		Type:            block.RewardsBlock,
	}
	mbHash, _ := core.CalculateHash(&marshal.JsonMarshalizer{}, &mock.HasherMock{}, bdy)
	mbh.Hash = mbHash

	mb := &block.MetaBlock{
		EpochStart: getDefaultEpochStart(),
		MiniBlockHeaders: []block.MiniBlockHeader{
			mbh,
		},
	}
	valInfo := make(map[uint32][]*state.ValidatorInfo)
	valInfo[0] = []*state.ValidatorInfo{
		{
			PublicKey:       []byte("pubkey"),
			ShardId:         0,
			AccumulatedFees: big.NewInt(100),
			LeaderSuccess:   1,
		},
	}

	err := rwd.VerifyRewardsMiniBlocks(mb, valInfo)
	assert.Nil(t, err)
}

func TestRewardsCreator_VerifyRewardsMiniBlocksShouldWorkEvenIfNotAllShardsHaveRewards(t *testing.T) {
	t.Parallel()

	receivedShardID := uint32(5)
	shardCoordinator := &mock.ShardCoordinatorStub{
		ComputeIdCalled: func(address []byte) uint32 {
			return receivedShardID
		},
		NumberOfShardsCalled: func() uint32 {
			return receivedShardID + 1
		}}
	args := getRewardsArguments()
	args.ShardCoordinator = shardCoordinator
	rwd, _ := NewEpochStartRewardsCreator(args)
	rwdTx := rewardTx.RewardTx{
		Round:   0,
		Value:   big.NewInt(100),
		RcvAddr: []byte{},
		Epoch:   0,
	}
	rwdTxHash, _ := core.CalculateHash(&marshal.JsonMarshalizer{}, &mock.HasherMock{}, rwdTx)

	communityRewardTx := rewardTx.RewardTx{
		Round:   0,
		Value:   big.NewInt(50),
		RcvAddr: []byte{17},
		Epoch:   0,
	}
	commRwdTxHash, _ := core.CalculateHash(&marshal.JsonMarshalizer{}, &mock.HasherMock{}, communityRewardTx)

	bdy := block.MiniBlock{
		TxHashes:        [][]byte{commRwdTxHash, rwdTxHash},
		ReceiverShardID: receivedShardID,
		SenderShardID:   core.MetachainShardId,
		Type:            block.RewardsBlock,
	}
	mbh := block.MiniBlockHeader{
		Hash:            nil,
		SenderShardID:   core.MetachainShardId,
		ReceiverShardID: receivedShardID,
		TxCount:         2,
		Type:            block.RewardsBlock,
	}
	mbHash, _ := core.CalculateHash(&marshal.JsonMarshalizer{}, &mock.HasherMock{}, bdy)
	mbh.Hash = mbHash

	mb := &block.MetaBlock{
		EpochStart: getDefaultEpochStart(),
		MiniBlockHeaders: []block.MiniBlockHeader{
			mbh,
		},
	}
	valInfo := make(map[uint32][]*state.ValidatorInfo)
	valInfo[0] = []*state.ValidatorInfo{
		{
			PublicKey:       []byte("pubkey"),
			ShardId:         receivedShardID,
			AccumulatedFees: big.NewInt(100),
			LeaderSuccess:   1,
		},
	}

	err := rwd.VerifyRewardsMiniBlocks(mb, valInfo)
	assert.Nil(t, err)
}

func TestRewardsCreator_CreateMarshalizedData(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	rwd, _ := NewEpochStartRewardsCreator(args)

	mb := &block.MetaBlock{
		EpochStart: getDefaultEpochStart(),
	}
	valInfo := make(map[uint32][]*state.ValidatorInfo)
	valInfo[0] = []*state.ValidatorInfo{
		{
			PublicKey:       []byte("pubkey"),
			ShardId:         0,
			AccumulatedFees: big.NewInt(100),
		},
	}
	_, _ = rwd.CreateRewardsMiniBlocks(mb, valInfo)

	rwdTx := rewardTx.RewardTx{
		Round:   0,
		Value:   big.NewInt(100),
		RcvAddr: []byte{},
		Epoch:   0,
	}
	rwdTxHash, _ := core.CalculateHash(&marshal.JsonMarshalizer{}, &mock.HasherMock{}, rwdTx)

	bdy := block.Body{
		MiniBlocks: []*block.MiniBlock{
			{
				ReceiverShardID: 0,
				Type:            block.RewardsBlock,
				TxHashes:        [][]byte{rwdTxHash},
			},
		},
	}
	res := rwd.CreateMarshalizedData(&bdy)

	assert.NotNil(t, res)
}

func TestRewardsCreator_SaveTxBlockToStorage(t *testing.T) {
	t.Parallel()

	putRwdTxWasCalled := false
	putMbWasCalled := false

	args := getRewardsArguments()
	args.RewardsStorage = &mock.StorerStub{
		PutCalled: func(_, _ []byte) error {
			putRwdTxWasCalled = true
			return nil
		},
	}
	args.MiniBlockStorage = &mock.StorerStub{
		PutCalled: func(_, _ []byte) error {
			putMbWasCalled = true
			return nil
		},
	}
	rwd, _ := NewEpochStartRewardsCreator(args)

	mb := &block.MetaBlock{
		EpochStart: getDefaultEpochStart(),
	}
	valInfo := make(map[uint32][]*state.ValidatorInfo)
	valInfo[0] = []*state.ValidatorInfo{
		{
			PublicKey:       []byte("pubkey"),
			ShardId:         0,
			AccumulatedFees: big.NewInt(100),
			LeaderSuccess:   1,
		},
	}
	_, _ = rwd.CreateRewardsMiniBlocks(mb, valInfo)

	mb2 := block.MetaBlock{
		MiniBlockHeaders: []block.MiniBlockHeader{
			{
				Type: block.RewardsBlock,
			},
		},
	}
	rwdTx := rewardTx.RewardTx{
		Round:   0,
		Value:   big.NewInt(100),
		RcvAddr: []byte{},
		Epoch:   0,
	}
	rwdTxHash, _ := core.CalculateHash(&marshal.JsonMarshalizer{}, &mock.HasherMock{}, rwdTx)
	bdy := block.Body{
		MiniBlocks: []*block.MiniBlock{
			{
				ReceiverShardID: 0,
				SenderShardID:   core.MetachainShardId,
				Type:            block.RewardsBlock,
				TxHashes:        [][]byte{rwdTxHash},
			},
		},
	}
	rwd.SaveTxBlockToStorage(&mb2, &bdy)

	assert.True(t, putRwdTxWasCalled)
	assert.True(t, putMbWasCalled)
}

func TestRewardsCreator_addValidatorRewardsToMiniBlocksZeroValueShouldNotAdd(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	rwdc, _ := NewEpochStartRewardsCreator(args)

	epochStartEconomics := getDefaultEpochStart()
	epochStartEconomics.Economics.RewardsForCommunity = big.NewInt(0)
	epochStartEconomics.Economics.RewardsPerBlock = big.NewInt(0)

	mb := &block.MetaBlock{
		EpochStart: epochStartEconomics,
	}

	valInfo := make(map[uint32][]*state.ValidatorInfo)
	valInfo[0] = []*state.ValidatorInfo{
		{
			PublicKey:       []byte("pubkey"),
			ShardId:         0,
			AccumulatedFees: big.NewInt(0),
		},
	}

	rwdc.fillRewardsPerBlockPerNode(&mb.EpochStart.Economics)
	mbs, err := rwdc.CreateRewardsMiniBlocks(mb, valInfo)
	assert.Nil(t, err)
	for _, mb := range mbs {
		assert.Equal(t, 0, len(mb.TxHashes))
	}
}

func TestRewardsCreator_addValidatorRewardsToMiniBlocks(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	rwdc, _ := NewEpochStartRewardsCreator(args)

	mb := &block.MetaBlock{
		EpochStart: getDefaultEpochStart(),
	}

	miniBlocks := make(block.MiniBlockSlice, rwdc.shardCoordinator.NumberOfShards())
	miniBlocks[0] = &block.MiniBlock{}
	miniBlocks[0].SenderShardID = core.MetachainShardId
	miniBlocks[0].ReceiverShardID = 0
	miniBlocks[0].Type = block.RewardsBlock
	miniBlocks[0].TxHashes = make([][]byte, 0)

	cloneMb := &(*miniBlocks[0])
	cloneMb.TxHashes = make([][]byte, 0)
	expectedRwdTx := &rewardTx.RewardTx{
		Round:   0,
		Value:   big.NewInt(100),
		RcvAddr: []byte("pubkey"),
		Epoch:   0,
	}
	expectedRwdTxHash, _ := core.CalculateHash(&marshal.JsonMarshalizer{}, &mock.HasherMock{}, expectedRwdTx)
	cloneMb.TxHashes = append(cloneMb.TxHashes, expectedRwdTxHash)

	valInfo := make(map[uint32][]*state.ValidatorInfo)
	valInfo[0] = []*state.ValidatorInfo{
		{
			PublicKey:       []byte("pubkey"),
			ShardId:         0,
			AccumulatedFees: big.NewInt(100),
			LeaderSuccess:   1,
		},
	}

	rwdc.fillRewardsPerBlockPerNode(&mb.EpochStart.Economics)
	err := rwdc.addValidatorRewardsToMiniBlocks(valInfo, mb, miniBlocks, &rewardTx.RewardTx{})
	assert.Nil(t, err)
	assert.Equal(t, cloneMb, miniBlocks[0])
}

func TestRewardsCreator_ProtocolRewardsForValidatorFromMultipleShards(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	args.NodesConfigProvider = &mock.NodesCoordinatorStub{
		ConsensusGroupSizeCalled: func(shardID uint32) int {
			if shardID == core.MetachainShardId {
				return 400
			}
			return 63
		},
	}
	rwdc, _ := NewEpochStartRewardsCreator(args)

	mb := &block.MetaBlock{
		EpochStart: getDefaultEpochStart(),
	}

	pubkey := "pubkey"
	valInfo := make(map[uint32][]*state.ValidatorInfo)
	valInfo[0] = []*state.ValidatorInfo{
		{
			RewardAddress:              []byte(pubkey),
			ShardId:                    0,
			AccumulatedFees:            big.NewInt(100),
			NumSelectedInSuccessBlocks: 100,
			LeaderSuccess:              1,
		},
	}
	valInfo[core.MetachainShardId] = []*state.ValidatorInfo{
		{
			RewardAddress:              []byte(pubkey),
			ShardId:                    core.MetachainShardId,
			AccumulatedFees:            big.NewInt(100),
			NumSelectedInSuccessBlocks: 200,
			LeaderSuccess:              1,
		},
	}

	rwdc.fillRewardsPerBlockPerNode(&mb.EpochStart.Economics)
	rwdInfoData := rwdc.computeValidatorInfoPerRewardAddress(valInfo, &rewardTx.RewardTx{})
	assert.Equal(t, 1, len(rwdInfoData))
	rwdInfo := rwdInfoData[pubkey]
	assert.Equal(t, rwdInfo.address, pubkey)

	assert.Equal(t, rwdInfo.accumulatedFees.Cmp(big.NewInt(200)), 0)
	protocolRewards := uint64(valInfo[0][0].NumSelectedInSuccessBlocks) * (mb.EpochStart.Economics.RewardsPerBlock.Uint64() / uint64(args.NodesConfigProvider.ConsensusGroupSize(0)))
	protocolRewards += uint64(valInfo[core.MetachainShardId][0].NumSelectedInSuccessBlocks) * (mb.EpochStart.Economics.RewardsPerBlock.Uint64() / uint64(args.NodesConfigProvider.ConsensusGroupSize(core.MetachainShardId)))
	assert.Equal(t, rwdInfo.protocolRewards.Uint64(), protocolRewards)
}

func TestRewardsCreator_CreateCommunityRewardTransaction(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	rwdc, _ := NewEpochStartRewardsCreator(args)
	mb := &block.MetaBlock{
		EpochStart: getDefaultEpochStart(),
	}
	expectedRewardTx := &rewardTx.RewardTx{
		Round:   0,
		Value:   big.NewInt(50),
		RcvAddr: []byte{17},
		Epoch:   0,
	}

	rwdTx, _, err := rwdc.createCommunityRewardTransaction(mb)
	assert.Equal(t, expectedRewardTx, rwdTx)
	assert.Nil(t, err)
}

func TestRewardsCreator_AddCommunityRewardToMiniBlocks(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	rwdc, _ := NewEpochStartRewardsCreator(args)
	metaBlk := &block.MetaBlock{
		EpochStart: getDefaultEpochStart(),
	}

	miniBlocks := make(block.MiniBlockSlice, rwdc.shardCoordinator.NumberOfShards())
	miniBlocks[0] = &block.MiniBlock{}
	miniBlocks[0].SenderShardID = core.MetachainShardId
	miniBlocks[0].ReceiverShardID = 0
	miniBlocks[0].Type = block.RewardsBlock
	miniBlocks[0].TxHashes = make([][]byte, 0)

	cloneMb := &(*miniBlocks[0])
	cloneMb.TxHashes = make([][]byte, 0)
	expectedRewardTx := &rewardTx.RewardTx{
		Round:   0,
		Value:   big.NewInt(50),
		RcvAddr: []byte{17},
		Epoch:   0,
	}
	expectedRwdTxHash, _ := core.CalculateHash(&marshal.JsonMarshalizer{}, &mock.HasherMock{}, expectedRewardTx)
	cloneMb.TxHashes = append(cloneMb.TxHashes, expectedRwdTxHash)

	miniBlocks, err := rwdc.CreateRewardsMiniBlocks(metaBlk, make(map[uint32][]*state.ValidatorInfo))
	assert.Nil(t, err)
	assert.Equal(t, cloneMb, miniBlocks[0])
}

func TestRewardsCreator_ValidatorInfoWithMetaAddressAddedToCommunityReward(t *testing.T) {
	t.Parallel()

	args := getRewardsArguments()
	args.ShardCoordinator, _ = sharding.NewMultiShardCoordinator(1, core.MetachainShardId)
	rwdc, _ := NewEpochStartRewardsCreator(args)
	metaBlk := &block.MetaBlock{
		EpochStart: getDefaultEpochStart(),
	}

	valInfo := make(map[uint32][]*state.ValidatorInfo)
	valInfo[0] = []*state.ValidatorInfo{
		{
			RewardAddress:              factory.StakingSCAddress,
			ShardId:                    0,
			AccumulatedFees:            big.NewInt(100),
			NumSelectedInSuccessBlocks: 1,
			LeaderSuccess:              1,
		},
	}
	miniBlocks, err := rwdc.CreateRewardsMiniBlocks(metaBlk, valInfo)
	assert.Nil(t, err)
	assert.Equal(t, len(miniBlocks), 1)
	assert.Equal(t, len(miniBlocks[0].TxHashes), 1)

	expectedCommunityValue := big.NewInt(0).Add(metaBlk.EpochStart.Economics.RewardsForCommunity, metaBlk.EpochStart.Economics.RewardsPerBlock)
	expectedCommunityValue.Add(expectedCommunityValue, big.NewInt(100))
	communityReward, err := rwdc.currTxs.GetTx(miniBlocks[0].TxHashes[0])
	assert.Nil(t, err)
	assert.True(t, expectedCommunityValue.Cmp(communityReward.GetValue()) == 0)
}

func getDefaultEpochStart() block.EpochStart {
	return block.EpochStart{
		Economics: block.Economics{
			TotalSupply:         big.NewInt(10000),
			TotalToDistribute:   big.NewInt(10000),
			TotalNewlyMinted:    big.NewInt(10000),
			RewardsPerBlock:     big.NewInt(10000),
			NodePrice:           big.NewInt(10000),
			RewardsForCommunity: big.NewInt(50),
		},
	}
}

func getRewardsArguments() ArgsNewRewardsCreator {
	return ArgsNewRewardsCreator{
		ShardCoordinator:    mock.NewMultiShardsCoordinatorMock(2),
		PubkeyConverter:     mock.NewPubkeyConverterMock(32),
		RewardsStorage:      &mock.StorerStub{},
		MiniBlockStorage:    &mock.StorerStub{},
		Hasher:              &mock.HasherMock{},
		Marshalizer:         &mock.MarshalizerMock{},
		DataPool:            testscommon.NewPoolsHolderStub(),
		CommunityAddress:    "11", // string hex => 17 decimal
		NodesConfigProvider: &mock.NodesCoordinatorStub{},
	}
}
