package dataprocessor

import (
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/statistics"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

const numMillisecondsInASecond = 1000

type tpsBenchmarkUpdater struct {
	tpsBenchmark   statistics.TPSBenchmark
	elasticIndexer process.Indexer
}

// NewTPSBenchmarkUpdater returns a new instance of tpsBenchmarkUpdater
func NewTPSBenchmarkUpdater(
	genesisNodesConfig sharding.GenesisNodesSetupHandler,
	elasticIndexer process.Indexer,
) (*tpsBenchmarkUpdater, error) {
	if check.IfNil(genesisNodesConfig) {
		return nil, ErrNilGenesisNodesSetup
	}
	if check.IfNil(elasticIndexer) {
		return nil, ErrNilElasticIndexer
	}

	tpsBenchmark, err := statistics.NewTPSBenchmark(
		genesisNodesConfig.NumberOfShards(),
		genesisNodesConfig.GetRoundDuration()/numMillisecondsInASecond,
	)
	if err != nil {
		return nil, err
	}

	return &tpsBenchmarkUpdater{
		tpsBenchmark:   tpsBenchmark,
		elasticIndexer: elasticIndexer,
	}, nil
}

// IndexTPSForMetaBlock will call the indexer's tps for a metablock
func (tbp *tpsBenchmarkUpdater) IndexTPSForMetaBlock(metaBlock *block.MetaBlock) {
	tbp.tpsBenchmark.Update(metaBlock)
	tbp.elasticIndexer.UpdateTPS(tbp.tpsBenchmark)
}

// IsInterfaceNil returns true if there is no value under the interface
func (tbp *tpsBenchmarkUpdater) IsInterfaceNil() bool {
	return tbp == nil
}
