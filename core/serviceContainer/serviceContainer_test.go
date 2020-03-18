package serviceContainer_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/core/check"
	elasticIndexer "github.com/ElrondNetwork/elrond-go/core/indexer"
	"github.com/ElrondNetwork/elrond-go/core/mock"
	"github.com/ElrondNetwork/elrond-go/core/serviceContainer"
	"github.com/stretchr/testify/assert"
)

func TestServiceContainer_NewServiceContainerEmpty(t *testing.T) {
	sc, err := serviceContainer.NewServiceContainer()
	assert.Nil(t, err)
	assert.NotNil(t, sc)
	assert.Nil(t, sc.Indexer())
}

func TestServiceContainer_NewServiceContainerWithIndexer(t *testing.T) {
	indexer := elasticIndexer.NewNilIndexer()
	sc, err := serviceContainer.NewServiceContainer(serviceContainer.WithIndexer(indexer))

	assert.Nil(t, err)
	assert.NotNil(t, sc)
	assert.Equal(t, indexer, sc.Indexer())
}

func TestServiceContainer_NewServiceContainerWithNilIndexer(t *testing.T) {
	sc, err := serviceContainer.NewServiceContainer(serviceContainer.WithIndexer(nil))

	assert.Nil(t, err)
	assert.NotNil(t, sc)
	assert.Nil(t, sc.Indexer())
}

func TestServiceContainer_NewServiceContainerWithTPSBenchmark(t *testing.T) {
	tpsBenchmark := &mock.TpsBenchmarkMock{}

	sc, err := serviceContainer.NewServiceContainer(serviceContainer.WithTPSBenchmark(tpsBenchmark))
	assert.Nil(t, err)
	assert.False(t, check.IfNil(sc))
	assert.Equal(t, tpsBenchmark, sc.TPSBenchmark())
}

func TestServiceContainer_NewServiceContainerWithNilTPSBenchmark(t *testing.T) {
	sc, err := serviceContainer.NewServiceContainer(serviceContainer.WithTPSBenchmark(nil))
	assert.Nil(t, err)
	assert.NotNil(t, sc)
	assert.Nil(t, sc.TPSBenchmark())
}
