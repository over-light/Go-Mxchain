package core_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/stretchr/testify/assert"
)

func TestMaxInt32ShouldReturnA(t *testing.T) {
	a := int32(-1)
	b := int32(-2)
	assert.Equal(t, a, core.MaxInt32(a, b))
}

func TestMaxInt32ShouldReturnB(t *testing.T) {
	a := int32(-2)
	b := int32(-1)
	assert.Equal(t, b, core.MaxInt32(a, b))
}

func TestMinInt32ShouldReturnB(t *testing.T) {
	a := int32(-1)
	b := int32(-2)
	assert.Equal(t, b, core.MinInt32(a, b))
}

func TestMinInt32ShouldReturnA(t *testing.T) {
	a := int32(-2)
	b := int32(-1)
	assert.Equal(t, a, core.MinInt32(a, b))
}

func TestMaxUint32ShouldReturnA(t *testing.T) {
	a := uint32(11)
	b := uint32(10)
	assert.Equal(t, a, core.MaxUint32(a, b))
}

func TestMaxUint32ShouldReturnB(t *testing.T) {
	a := uint32(10)
	b := uint32(11)
	assert.Equal(t, b, core.MaxUint32(a, b))
}

func TestMinUint32ShouldReturnB(t *testing.T) {
	a := uint32(11)
	b := uint32(10)
	assert.Equal(t, b, core.MinUint32(a, b))
}

func TestMinUint32ShouldReturnA(t *testing.T) {
	a := uint32(10)
	b := uint32(11)
	assert.Equal(t, a, core.MinUint32(a, b))
}
