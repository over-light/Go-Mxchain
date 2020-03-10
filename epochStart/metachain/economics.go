package metachain

import (
	"bytes"
	"math/big"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

const numberOfDaysInYear = 365.0
const numberOfSecondsInDay = 86400

type economics struct {
	marshalizer      marshal.Marshalizer
	hasher           hashing.Hasher
	store            dataRetriever.StorageService
	shardCoordinator sharding.Coordinator
	nodesCoordinator sharding.NodesCoordinator
	rewardsHandler   process.RewardsHandler
	roundTime        process.RoundTimeDurationHandler
}

// ArgsNewEpochEconomics holds the arguments needed when creating a new end of epoch economics data creator
type ArgsNewEpochEconomics struct {
	Marshalizer      marshal.Marshalizer
	Hasher           hashing.Hasher
	Store            dataRetriever.StorageService
	ShardCoordinator sharding.Coordinator
	NodesCoordinator sharding.NodesCoordinator
	RewardsHandler   process.RewardsHandler
	RoundTime        process.RoundTimeDurationHandler
}

// NewEndOfEpochEconomicsDataCreator creates a new end of epoch economics data creator object
func NewEndOfEpochEconomicsDataCreator(args ArgsNewEpochEconomics) (*economics, error) {
	if check.IfNil(args.Marshalizer) {
		return nil, epochStart.ErrNilMarshalizer
	}
	if check.IfNil(args.Hasher) {
		return nil, epochStart.ErrNilHasher
	}
	if check.IfNil(args.Store) {
		return nil, epochStart.ErrNilStorage
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, epochStart.ErrNilShardCoordinator
	}
	if check.IfNil(args.NodesCoordinator) {
		return nil, epochStart.ErrNilNodesCoordinator
	}
	if check.IfNil(args.RewardsHandler) {
		return nil, epochStart.ErrNilRewardsHandler
	}
	if check.IfNil(args.RoundTime) {
		return nil, process.ErrNilRounder
	}

	e := &economics{
		marshalizer:      args.Marshalizer,
		hasher:           args.Hasher,
		store:            args.Store,
		shardCoordinator: args.ShardCoordinator,
		nodesCoordinator: args.NodesCoordinator,
		rewardsHandler:   args.RewardsHandler,
		roundTime:        args.RoundTime,
	}
	return e, nil
}

// ComputeEndOfEpochEconomics calculates the rewards per block value for the current epoch
func (e *economics) ComputeEndOfEpochEconomics(
	metaBlock *block.MetaBlock,
) (*block.Economics, error) {
	if check.IfNil(metaBlock) {
		return nil, epochStart.ErrNilHeaderHandler
	}
	if metaBlock.AccumulatedFeesInEpoch == nil {
		return nil, epochStart.ErrNilTotalAccumulatedFeesInEpoch
	}
	if !metaBlock.IsStartOfEpochBlock() || metaBlock.Epoch < 1 {
		return nil, epochStart.ErrNotEpochStartBlock
	}

	noncesPerShardPrevEpoch, prevEpochStart, err := e.startNoncePerShardFromEpochStart(metaBlock.Epoch - 1)
	if err != nil {
		return nil, err
	}
	prevEpochEconomics := prevEpochStart.EpochStart.Economics

	noncesPerShardCurrEpoch, err := e.startNoncePerShardFromLastCrossNotarized(metaBlock.GetNonce(), metaBlock.EpochStart)
	if err != nil {
		return nil, err
	}

	roundsPassedInEpoch := metaBlock.GetRound() - prevEpochStart.GetRound()
	maxBlocksInEpoch := roundsPassedInEpoch * uint64(e.shardCoordinator.NumberOfShards()+1)
	totalNumBlocksInEpoch := e.computeNumOfTotalCreatedBlocks(noncesPerShardPrevEpoch, noncesPerShardCurrEpoch)

	inflationRate, err := e.computeInflationRate(prevEpochEconomics.TotalSupply, prevEpochEconomics.NodePrice)
	if err != nil {
		return nil, err
	}

	rwdPerBlock := e.computeRewardsPerBlock(prevEpochEconomics.TotalSupply, maxBlocksInEpoch, inflationRate)
	totalRewardsToBeDistributed := big.NewInt(0).Mul(rwdPerBlock, big.NewInt(0).SetUint64(totalNumBlocksInEpoch))

	newTokens := big.NewInt(0).Sub(totalRewardsToBeDistributed, metaBlock.AccumulatedFeesInEpoch)
	if newTokens.Cmp(big.NewInt(0)) < 0 {
		newTokens = big.NewInt(0)
		totalRewardsToBeDistributed = big.NewInt(0).Set(metaBlock.AccumulatedFeesInEpoch)
		rwdPerBlock = big.NewInt(0).Div(totalRewardsToBeDistributed, big.NewInt(0).SetUint64(totalNumBlocksInEpoch))
	}

	prevEpochStartHash, err := core.CalculateHash(e.marshalizer, e.hasher, prevEpochStart)
	if err != nil {
		return nil, err
	}

	computedEconomics := block.Economics{
		TotalSupply:            big.NewInt(0).Add(prevEpochEconomics.TotalSupply, newTokens),
		TotalToDistribute:      big.NewInt(0).Set(totalRewardsToBeDistributed),
		TotalNewlyMinted:       big.NewInt(0).Set(newTokens),
		RewardsPerBlockPerNode: e.computeRewardsPerValidatorPerBlock(rwdPerBlock),
		// TODO: get actual nodePrice from auction smart contract (currently on another feature branch, and not all features enabled)
		NodePrice:          big.NewInt(0).Set(prevEpochEconomics.NodePrice),
		PrevEpochStartHash: prevEpochStartHash,
	}

	return &computedEconomics, nil
}

// compute rewards per node per block
func (e *economics) computeRewardsPerValidatorPerBlock(rwdPerBlock *big.Int) *big.Int {
	numOfNodes := e.nodesCoordinator.GetNumTotalEligible()
	return big.NewInt(0).Div(rwdPerBlock, big.NewInt(0).SetUint64(numOfNodes))
}

// compute inflation rate from totalSupply and totalStaked
func (e *economics) computeInflationRate(_ *big.Int, _ *big.Int) (float64, error) {
	//TODO: use prevTotalSupply and nodePrice (number of eligible + number of waiting)
	// for epoch which ends now to compute inflation rate according to formula provided by L.
	return e.rewardsHandler.MaxInflationRate(), nil
}

// compute rewards per block from according to inflation rate and total supply from previous block and maxBlocksPerEpoch
func (e *economics) computeRewardsPerBlock(
	prevTotalSupply *big.Int,
	maxBlocksInEpoch uint64,
	inflationRate float64,
) *big.Int {

	inflationRatePerDay := inflationRate / numberOfDaysInYear
	roundsPerDay := numberOfSecondsInDay / uint64(e.roundTime.TimeDuration().Seconds())
	maxBlocksInADay := roundsPerDay * uint64(e.shardCoordinator.NumberOfShards()+1)

	inflationRateForEpoch := inflationRatePerDay * (float64(maxBlocksInEpoch) / float64(maxBlocksInADay))

	rewardsPerBlock := big.NewInt(0).Div(prevTotalSupply, big.NewInt(0).SetUint64(maxBlocksInEpoch))
	rewardsPerBlock = core.GetPercentageOfValue(rewardsPerBlock, inflationRateForEpoch)

	return rewardsPerBlock
}

func (e *economics) computeNumOfTotalCreatedBlocks(
	mapStartNonce map[uint32]uint64,
	mapEndNonce map[uint32]uint64,
) uint64 {
	totalNumBlocks := uint64(0)
	for shardId := uint32(0); shardId < e.shardCoordinator.NumberOfShards(); shardId++ {
		totalNumBlocks += mapEndNonce[shardId] - mapStartNonce[shardId]
	}
	totalNumBlocks += mapEndNonce[core.MetachainShardId] - mapStartNonce[core.MetachainShardId]

	return totalNumBlocks
}

func (e *economics) startNoncePerShardFromEpochStart(epoch uint32) (map[uint32]uint64, *block.MetaBlock, error) {
	mapShardIdNonce := make(map[uint32]uint64, e.shardCoordinator.NumberOfShards()+1)
	for i := uint32(0); i < e.shardCoordinator.NumberOfShards(); i++ {
		mapShardIdNonce[i] = 0
	}
	mapShardIdNonce[core.MetachainShardId] = 0

	epochStartIdentifier := core.EpochStartIdentifier(epoch)
	previousEpochStartMeta, err := process.GetMetaHeaderFromStorage([]byte(epochStartIdentifier), e.marshalizer, e.store)
	if err != nil {
		return nil, nil, err
	}

	if epoch == 0 {
		return mapShardIdNonce, previousEpochStartMeta, nil
	}

	mapShardIdNonce[core.MetachainShardId] = previousEpochStartMeta.GetNonce()
	for _, shardData := range previousEpochStartMeta.EpochStart.LastFinalizedHeaders {
		mapShardIdNonce[shardData.ShardID] = shardData.Nonce
	}

	return mapShardIdNonce, previousEpochStartMeta, nil
}

func (e *economics) startNoncePerShardFromLastCrossNotarized(metaNonce uint64, epochStart block.EpochStart) (map[uint32]uint64, error) {
	mapShardIdNonce := make(map[uint32]uint64, e.shardCoordinator.NumberOfShards()+1)
	for i := uint32(0); i < e.shardCoordinator.NumberOfShards(); i++ {
		mapShardIdNonce[i] = 0
	}
	mapShardIdNonce[core.MetachainShardId] = metaNonce

	for _, shardData := range epochStart.LastFinalizedHeaders {
		mapShardIdNonce[shardData.ShardID] = shardData.Nonce
	}

	return mapShardIdNonce, nil
}

// VerifyRewardsPerBlock checks whether rewards per block value was correctly computed
func (e *economics) VerifyRewardsPerBlock(
	metaBlock *block.MetaBlock,
) error {
	if !metaBlock.IsStartOfEpochBlock() {
		return nil
	}
	computedEconomics, err := e.ComputeEndOfEpochEconomics(metaBlock)
	if err != nil {
		return err
	}
	computedEconomicsHash, err := core.CalculateHash(e.marshalizer, e.hasher, computedEconomics)
	if err != nil {
		return err
	}

	receivedEconomics := metaBlock.EpochStart.Economics
	receivedEconomicsHash, err := core.CalculateHash(e.marshalizer, e.hasher, &receivedEconomics)
	if err != nil {
		return err
	}

	if !bytes.Equal(receivedEconomicsHash, computedEconomicsHash) {
		logEconomicsDifferences(computedEconomics, &receivedEconomics)
		return epochStart.ErrEndOfEpochEconomicsDataDoesNotMatch
	}

	return nil
}

// IsInterfaceNil returns true if underlying object is nil
func (e *economics) IsInterfaceNil() bool {
	return e == nil
}

func logEconomicsDifferences(computed *block.Economics, received *block.Economics) {
	log.Warn("VerifyRewardsPerBlock error",
		"\ncomputed total to distribute", computed.TotalToDistribute,
		"computed total newly minted", computed.TotalNewlyMinted,
		"computed total supply", computed.TotalSupply,
		"computed rewards per block per node", computed.RewardsPerBlockPerNode,
		"computed node price", computed.NodePrice,
		"\nreceived total to distribute", received.TotalToDistribute,
		"received total newly minted", received.TotalNewlyMinted,
		"received total supply", received.TotalSupply,
		"received rewards per block per node", received.RewardsPerBlockPerNode,
		"received node price", received.NodePrice,
	)
}
