package sharding

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/sharding/mock"
	"github.com/stretchr/testify/assert"
)

func TestSelectionBasedProvider_AddToSortedSlice(t *testing.T) {
	sbp := NewSelectionBasedProvider(&mock.HasherMock{}, 7)

	v1 := &validatorEntry{
		startIndex:     0,
		numAppearances: 2,
	}
	v2 := &validatorEntry{
		startIndex:     7,
		numAppearances: 2,
	}
	v3 := &validatorEntry{
		startIndex:     4,
		numAppearances: 3,
	}
	v4 := &validatorEntry{
		startIndex:     5,
		numAppearances: 3,
	}
	v5 := &validatorEntry{
		startIndex:     12,
		numAppearances: 3,
	}
	v6 := &validatorEntry{
		startIndex:     9,
		numAppearances: 3,
	}
	v7 := &validatorEntry{
		startIndex:     5,
		numAppearances: 3,
	}

	sbp.addToSortedSlice(v1)
	sbp.addToSortedSlice(v2)
	sbp.addToSortedSlice(v3)
	sbp.addToSortedSlice(v4)
	sbp.addToSortedSlice(v5)
	sbp.addToSortedSlice(v6)
	sbp.addToSortedSlice(v7)

	lastIndex := sbp.sortedSlice[0].startIndex
	for i := 1; i < len(sbp.sortedSlice); i++ {
		if sbp.sortedSlice[i].startIndex < lastIndex {
			assert.Fail(t, "slice is not sorted.")
		}
	}
}

func TestSelectionBasedProvider_Get(t *testing.T) {
	sbp := NewSelectionBasedProvider(&mock.HasherMock{}, 7)

	numVals := 7
	randomness := []byte("randomness")
	expElList := getExpandedEligibleList(17)
	res, err := sbp.Get(randomness, int64(numVals), expElList)

	displayVals(res)

	assert.Nil(t, err)
	assert.Equal(t, numVals, len(res))
}
