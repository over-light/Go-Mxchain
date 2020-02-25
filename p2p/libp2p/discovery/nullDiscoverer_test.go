package discovery_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/p2p/libp2p/discovery"
	"github.com/stretchr/testify/assert"
)

func TestNullDiscoverer(t *testing.T) {
	t.Parallel()

	nd := discovery.NewNullDiscoverer()

	assert.False(t, check.IfNil(nd))
	assert.Equal(t, discovery.NullName, nd.Name())
	assert.Nil(t, nd.Bootstrap())
	assert.Equal(t, 0, len(nd.ReconnectToNetwork()))
}
