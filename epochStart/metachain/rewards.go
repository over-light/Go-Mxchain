package metachain

import (
	"bytes"
	"math/big"
	"sort"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/dataPool"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
)

var _ process.EpochStartRewardsCreator = (*rewardsCreator)(nil)

var zero = big.NewInt(0)

// ArgsNewRewardsCreator defines the arguments structure needed to create a new rewards creator
type ArgsNewRewardsCreator struct {
	ShardCoordinator              sharding.Coordinator
	PubkeyConverter               core.PubkeyConverter
	RewardsStorage                storage.Storer
	MiniBlockStorage              storage.Storer
	Hasher                        hashing.Hasher
	Marshalizer                   marshal.Marshalizer
	DataPool                      dataRetriever.PoolsHolder
	ProtocolSustainabilityAddress string
	NodesConfigProvider           epochStart.NodesConfigProvider
}

type rewardsCreator struct {
	currTxs                       dataRetriever.TransactionCacher
	shardCoordinator              sharding.Coordinator
	pubkeyConverter               core.PubkeyConverter
	rewardsStorage                storage.Storer
	miniBlockStorage              storage.Storer
	protocolSustainabilityAddress []byte
	nodesConfigProvider           epochStart.NodesConfigProvider

	hasher                         hashing.Hasher
	marshalizer                    marshal.Marshalizer
	dataPool                       dataRetriever.PoolsHolder
	mapRewardsPerBlockPerValidator map[uint32]*big.Int
}

type rewardInfoData struct {
	accumulatedFees *big.Int
	address         string
	protocolRewards *big.Int
}

// NewEpochStartRewardsCreator creates a new rewards creator object
func NewEpochStartRewardsCreator(args ArgsNewRewardsCreator) (*rewardsCreator, error) {
	if check.IfNil(args.ShardCoordinator) {
		return nil, epochStart.ErrNilShardCoordinator
	}
	if check.IfNil(args.PubkeyConverter) {
		return nil, epochStart.ErrNilPubkeyConverter
	}
	if check.IfNil(args.RewardsStorage) {
		return nil, epochStart.ErrNilStorage
	}
	if check.IfNil(args.Marshalizer) {
		return nil, epochStart.ErrNilMarshalizer
	}
	if check.IfNil(args.Hasher) {
		return nil, epochStart.ErrNilHasher
	}
	if check.IfNil(args.MiniBlockStorage) {
		return nil, epochStart.ErrNilStorage
	}
	if check.IfNil(args.DataPool) {
		return nil, epochStart.ErrNilDataPoolsHolder
	}
	if len(args.ProtocolSustainabilityAddress) == 0 {
		return nil, epochStart.ErrNilProtocolSustainabilityAddress
	}
	if check.IfNil(args.NodesConfigProvider) {
		return nil, epochStart.ErrNilNodesConfigProvider
	}

	address, err := args.PubkeyConverter.Decode(args.ProtocolSustainabilityAddress)
	if err != nil {
		log.Warn("invalid protocol sustainability reward address", "err", err, "provided address", args.ProtocolSustainabilityAddress)
		return nil, err
	}
	protocolSustainabilityShardID := args.ShardCoordinator.ComputeId(address)
	if protocolSustainabilityShardID == core.MetachainShardId {
		return nil, epochStart.ErrProtocolSustainabilityAddressInMetachain
	}

	currTxsCache, err := dataPool.NewCurrentBlockPool()
	if err != nil {
		return nil, err
	}

	rc := &rewardsCreator{
		currTxs:                       currTxsCache,
		shardCoordinator:              args.ShardCoordinator,
		pubkeyConverter:               args.PubkeyConverter,
		rewardsStorage:                args.RewardsStorage,
		hasher:                        args.Hasher,
		marshalizer:                   args.Marshalizer,
		miniBlockStorage:              args.MiniBlockStorage,
		dataPool:                      args.DataPool,
		protocolSustainabilityAddress: address,
		nodesConfigProvider:           args.NodesConfigProvider,
	}

	return rc, nil
}

// CreateBlockStarted announces block creation started and cleans inside data
func (rc *rewardsCreator) clean() {
	rc.mapRewardsPerBlockPerValidator = make(map[uint32]*big.Int)
	rc.currTxs.Clean()
}

// CreateRewardsMiniBlocks creates the rewards miniblocks according to economics data and validator info
func (rc *rewardsCreator) CreateRewardsMiniBlocks(metaBlock *block.MetaBlock, validatorsInfo map[uint32][]*state.ValidatorInfo) (block.MiniBlockSlice, error) {
	if check.IfNil(metaBlock) {
		return nil, epochStart.ErrNilHeaderHandler
	}

	rc.clean()

	miniBlocks := make(block.MiniBlockSlice, rc.shardCoordinator.NumberOfShards())
	for i := uint32(0); i < rc.shardCoordinator.NumberOfShards(); i++ {
		miniBlocks[i] = &block.MiniBlock{}
		miniBlocks[i].SenderShardID = core.MetachainShardId
		miniBlocks[i].ReceiverShardID = i
		miniBlocks[i].Type = block.RewardsBlock
		miniBlocks[i].TxHashes = make([][]byte, 0)
	}

	protocolSustainabilityRwdTx, protocolSustainabilityShardId, err := rc.createProtocolSustainabilityRewardTransaction(metaBlock)
	if err != nil {
		return nil, err
	}

	rc.fillRewardsPerBlockPerNode(&metaBlock.EpochStart.Economics)
	err = rc.addValidatorRewardsToMiniBlocks(validatorsInfo, metaBlock, miniBlocks, protocolSustainabilityRwdTx)
	if err != nil {
		return nil, err
	}

	if protocolSustainabilityRwdTx.Value.Cmp(zero) > 0 {
		protocolSustainabilityRwdHash, errHash := core.CalculateHash(rc.marshalizer, rc.hasher, protocolSustainabilityRwdTx)
		if errHash != nil {
			return nil, errHash
		}

		rc.currTxs.AddTx(protocolSustainabilityRwdHash, protocolSustainabilityRwdTx)
		miniBlocks[protocolSustainabilityShardId].TxHashes = append(miniBlocks[protocolSustainabilityShardId].TxHashes, protocolSustainabilityRwdHash)
	}

	for shId := uint32(0); shId < rc.shardCoordinator.NumberOfShards(); shId++ {
		sort.Slice(miniBlocks[shId].TxHashes, func(i, j int) bool {
			return bytes.Compare(miniBlocks[shId].TxHashes[i], miniBlocks[shId].TxHashes[j]) < 0
		})
	}

	finalMiniBlocks := make(block.MiniBlockSlice, 0)
	for i := uint32(0); i < rc.shardCoordinator.NumberOfShards(); i++ {
		if len(miniBlocks[i].TxHashes) > 0 {
			finalMiniBlocks = append(finalMiniBlocks, miniBlocks[i])
		}
	}

	return finalMiniBlocks, nil
}

func (rc *rewardsCreator) fillRewardsPerBlockPerNode(economicsData *block.Economics) {
	rc.mapRewardsPerBlockPerValidator = make(map[uint32]*big.Int)
	for i := uint32(0); i < rc.shardCoordinator.NumberOfShards(); i++ {
		consensusSize := big.NewInt(int64(rc.nodesConfigProvider.ConsensusGroupSize(i)))
		rc.mapRewardsPerBlockPerValidator[i] = big.NewInt(0).Div(economicsData.RewardsPerBlock, consensusSize)
		log.Debug("rewardsPerBlockPerValidator", "shardID", i, "value", rc.mapRewardsPerBlockPerValidator[i].String())
	}

	consensusSize := big.NewInt(int64(rc.nodesConfigProvider.ConsensusGroupSize(core.MetachainShardId)))
	rc.mapRewardsPerBlockPerValidator[core.MetachainShardId] = big.NewInt(0).Div(economicsData.RewardsPerBlock, consensusSize)
	log.Debug("rewardsPerBlockPerValidator", "shardID", core.MetachainShardId, "value", rc.mapRewardsPerBlockPerValidator[core.MetachainShardId].String())
}

func (rc *rewardsCreator) addValidatorRewardsToMiniBlocks(
	validatorsInfo map[uint32][]*state.ValidatorInfo,
	metaBlock *block.MetaBlock,
	miniBlocks block.MiniBlockSlice,
	protocolSustainabilityRwdTx *rewardTx.RewardTx,
) error {
	rwdAddrValidatorInfo := rc.computeValidatorInfoPerRewardAddress(validatorsInfo, protocolSustainabilityRwdTx)
	for _, rwdInfo := range rwdAddrValidatorInfo {
		rwdTx, rwdTxHash, err := rc.createRewardFromRwdInfo(rwdInfo, metaBlock)
		if err != nil {
			return err
		}
		if rwdTx.Value.Cmp(zero) <= 0 {
			continue
		}

		shardId := rc.shardCoordinator.ComputeId([]byte(rwdInfo.address))
		if shardId == core.MetachainShardId {
			protocolSustainabilityRwdTx.Value.Add(protocolSustainabilityRwdTx.Value, rwdTx.Value)
			continue
		}

		rc.currTxs.AddTx(rwdTxHash, rwdTx)
		miniBlocks[shardId].TxHashes = append(miniBlocks[shardId].TxHashes, rwdTxHash)
	}

	return nil
}

func (rc *rewardsCreator) createProtocolSustainabilityRewardTransaction(
	metaBlock *block.MetaBlock,
) (*rewardTx.RewardTx, uint32, error) {

	shardID := rc.shardCoordinator.ComputeId(rc.protocolSustainabilityAddress)
	protocolSustainabilityRwdTx := &rewardTx.RewardTx{
		Round:   metaBlock.GetRound(),
		Value:   big.NewInt(0).Set(metaBlock.EpochStart.Economics.RewardsForProtocolSustainability),
		RcvAddr: rc.protocolSustainabilityAddress,
		Epoch:   metaBlock.Epoch,
	}

	return protocolSustainabilityRwdTx, shardID, nil
}

func (rc *rewardsCreator) computeValidatorInfoPerRewardAddress(
	validatorsInfo map[uint32][]*state.ValidatorInfo,
	protocolSustainabilityRwd *rewardTx.RewardTx,
) map[string]*rewardInfoData {

	rwdAddrValidatorInfo := make(map[string]*rewardInfoData)

	for _, shardValidatorsInfo := range validatorsInfo {
		for _, validatorInfo := range shardValidatorsInfo {
			rewardsPerBlockPerNodeForShard := rc.mapRewardsPerBlockPerValidator[validatorInfo.ShardId]
			protocolRewardValue := big.NewInt(0).Mul(rewardsPerBlockPerNodeForShard, big.NewInt(0).SetUint64(uint64(validatorInfo.NumSelectedInSuccessBlocks)))

			if validatorInfo.LeaderSuccess == 0 && validatorInfo.ValidatorFailure == 0 {
				protocolSustainabilityRwd.Value.Add(protocolSustainabilityRwd.Value, protocolRewardValue)
				continue
			}

			rwdInfo, ok := rwdAddrValidatorInfo[string(validatorInfo.RewardAddress)]
			if !ok {
				rwdInfo = &rewardInfoData{
					accumulatedFees: big.NewInt(0),
					protocolRewards: big.NewInt(0),
					address:         string(validatorInfo.RewardAddress),
				}
				rwdAddrValidatorInfo[string(validatorInfo.RewardAddress)] = rwdInfo
			}

			rwdInfo.accumulatedFees.Add(rwdInfo.accumulatedFees, validatorInfo.AccumulatedFees)
			rwdInfo.protocolRewards.Add(rwdInfo.protocolRewards, protocolRewardValue)
		}
	}

	return rwdAddrValidatorInfo
}

func (rc *rewardsCreator) createRewardFromRwdInfo(
	rwdInfo *rewardInfoData,
	metaBlock *block.MetaBlock,
) (*rewardTx.RewardTx, []byte, error) {
	rwdTx := &rewardTx.RewardTx{
		Round:   metaBlock.GetRound(),
		Value:   big.NewInt(0).Add(rwdInfo.accumulatedFees, rwdInfo.protocolRewards),
		RcvAddr: []byte(rwdInfo.address),
		Epoch:   metaBlock.Epoch,
	}

	rwdTxHash, err := core.CalculateHash(rc.marshalizer, rc.hasher, rwdTx)
	if err != nil {
		return nil, nil, err
	}

	//TODO change this to trace
	log.Debug("rewardTx",
		"address", []byte(rwdInfo.address),
		"value", rwdTx.Value.String(),
		"hash", rwdTxHash,
		"accumulatedFees", rwdInfo.accumulatedFees,
		"protocolRewards", rwdInfo.protocolRewards,
	)

	return rwdTx, rwdTxHash, nil
}

// VerifyRewardsMiniBlocks verifies if received rewards miniblocks are correct
func (rc *rewardsCreator) VerifyRewardsMiniBlocks(metaBlock *block.MetaBlock, validatorsInfo map[uint32][]*state.ValidatorInfo) error {
	if check.IfNil(metaBlock) {
		return epochStart.ErrNilHeaderHandler
	}

	createdMiniBlocks, err := rc.CreateRewardsMiniBlocks(metaBlock, validatorsInfo)
	if err != nil {
		return err
	}

	numReceivedRewardsMBs := 0
	for _, miniBlockHdr := range metaBlock.MiniBlockHeaders {
		if miniBlockHdr.Type != block.RewardsBlock {
			continue
		}

		numReceivedRewardsMBs++
		createdMiniBlock := getMiniBlockWithReceiverShardID(miniBlockHdr.ReceiverShardID, createdMiniBlocks)
		if createdMiniBlock == nil {
			return epochStart.ErrRewardMiniBlockHashDoesNotMatch
		}

		createdMBHash, errComputeHash := core.CalculateHash(rc.marshalizer, rc.hasher, createdMiniBlock)
		if errComputeHash != nil {
			return errComputeHash
		}

		if !bytes.Equal(createdMBHash, miniBlockHdr.Hash) {
			// TODO: add display debug prints of miniblocks contents
			return epochStart.ErrRewardMiniBlockHashDoesNotMatch
		}
	}

	if len(createdMiniBlocks) != numReceivedRewardsMBs {
		return epochStart.ErrRewardMiniBlocksNumDoesNotMatch
	}

	return nil
}

func getMiniBlockWithReceiverShardID(shardId uint32, miniBlocks block.MiniBlockSlice) *block.MiniBlock {
	for _, miniBlock := range miniBlocks {
		if miniBlock.ReceiverShardID == shardId {
			return miniBlock
		}
	}
	return nil
}

// CreateMarshalizedData creates the marshalized data to be sent to shards
func (rc *rewardsCreator) CreateMarshalizedData(body *block.Body) map[string][][]byte {
	if check.IfNil(body) {
		return nil
	}

	mrsTxs := make(map[string][][]byte)

	for _, miniBlock := range body.MiniBlocks {
		if miniBlock.Type != block.RewardsBlock {
			continue
		}

		broadcastTopic := createBroadcastTopic(rc.shardCoordinator, miniBlock.ReceiverShardID)
		if _, ok := mrsTxs[broadcastTopic]; !ok {
			mrsTxs[broadcastTopic] = make([][]byte, 0, len(miniBlock.TxHashes))
		}

		for _, txHash := range miniBlock.TxHashes {
			rwdTx, err := rc.currTxs.GetTx(txHash)
			if err != nil {
				log.Warn("rewardsCreator.CreateMarshalizedData.GetTx", "hash", txHash, "error", err)
				continue
			}

			marshalizedData, err := rc.marshalizer.Marshal(rwdTx)
			if err != nil {
				log.Error("rewardsCreator.CreateMarshalizedData.Marshal", "hash", txHash, "error", err)
				continue
			}

			mrsTxs[broadcastTopic] = append(mrsTxs[broadcastTopic], marshalizedData)
		}

		if len(mrsTxs[broadcastTopic]) == 0 {
			delete(mrsTxs, broadcastTopic)
		}
	}

	return mrsTxs
}

// GetRewardsTxs will return rewards txs MUST be called before SaveTxBlockToStorage
func (rc *rewardsCreator) GetRewardsTxs(body *block.Body) map[string]data.TransactionHandler {
	rewardsTxs := make(map[string]data.TransactionHandler)
	for _, miniBlock := range body.MiniBlocks {
		if miniBlock.Type != block.RewardsBlock {
			continue
		}

		for _, txHash := range miniBlock.TxHashes {
			rwTx, err := rc.currTxs.GetTx(txHash)
			if err != nil {
				continue
			}

			rewardsTxs[string(txHash)] = rwTx
		}
	}

	return rewardsTxs
}

// SaveTxBlockToStorage saves created data to storage
func (rc *rewardsCreator) SaveTxBlockToStorage(_ *block.MetaBlock, body *block.Body) {
	if check.IfNil(body) {
		return
	}

	for _, miniBlock := range body.MiniBlocks {
		if miniBlock.Type != block.RewardsBlock {
			continue
		}

		for _, txHash := range miniBlock.TxHashes {
			rwdTx, err := rc.currTxs.GetTx(txHash)
			if err != nil {
				continue
			}

			marshalizedData, err := rc.marshalizer.Marshal(rwdTx)
			if err != nil {
				continue
			}

			_ = rc.rewardsStorage.Put(txHash, marshalizedData)
		}

		marshalizedData, err := rc.marshalizer.Marshal(miniBlock)
		if err != nil {
			continue
		}

		mbHash := rc.hasher.Compute(string(marshalizedData))
		_ = rc.miniBlockStorage.Put(mbHash, marshalizedData)
	}
}

// DeleteTxsFromStorage deletes data from storage
func (rc *rewardsCreator) DeleteTxsFromStorage(metaBlock *block.MetaBlock, body *block.Body) {
	if check.IfNil(metaBlock) || check.IfNil(body) {
		return
	}

	for _, miniBlock := range body.MiniBlocks {
		if miniBlock.Type != block.RewardsBlock {
			continue
		}

		for _, txHash := range miniBlock.TxHashes {
			_ = rc.rewardsStorage.Remove(txHash)
		}
	}

	for _, mbHeader := range metaBlock.MiniBlockHeaders {
		if mbHeader.Type == block.RewardsBlock {
			_ = rc.miniBlockStorage.Remove(mbHeader.Hash)
		}
	}
}

// IsInterfaceNil return true if underlying object is nil
func (rc *rewardsCreator) IsInterfaceNil() bool {
	return rc == nil
}

func createBroadcastTopic(shardC sharding.Coordinator, destShId uint32) string {
	transactionTopic := factory.RewardsTransactionTopic +
		shardC.CommunicationIdentifier(destShId)
	return transactionTopic
}

// RemoveBlockDataFromPools removes block info from pools
func (rc *rewardsCreator) RemoveBlockDataFromPools(metaBlock *block.MetaBlock, body *block.Body) {
	if check.IfNil(metaBlock) || check.IfNil(body) {
		return
	}

	transactionsPool := rc.dataPool.Transactions()
	miniBlocksPool := rc.dataPool.MiniBlocks()

	for _, miniBlock := range body.MiniBlocks {
		if miniBlock.Type != block.RewardsBlock {
			continue
		}

		strCache := process.ShardCacherIdentifier(miniBlock.SenderShardID, miniBlock.ReceiverShardID)
		transactionsPool.RemoveSetOfDataFromPool(miniBlock.TxHashes, strCache)
	}

	for _, mbHeader := range metaBlock.MiniBlockHeaders {
		if mbHeader.Type != block.RewardsBlock {
			continue
		}

		miniBlocksPool.Remove(mbHeader.Hash)

		log.Trace("RemoveBlockDataFromPools",
			"hash", mbHeader.Hash,
			"type", mbHeader.Type,
			"sender", mbHeader.SenderShardID,
			"receiver", mbHeader.ReceiverShardID,
			"num txs", mbHeader.TxCount)
	}
}
