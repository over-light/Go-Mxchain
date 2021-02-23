package transaction

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCostResponseStructure(t *testing.T) {
	t.Parallel()

	costResponse := &CostResponse{
		GasUnits:   10000,
		RetMessage: "",
	}

	costResponseBytes, err := json.Marshal(costResponse)
	assert.Nil(t, err)
	assert.NotNil(t, costResponseBytes)

	costResponseMap := make(map[string]interface{})
	err = json.Unmarshal(costResponseBytes, &costResponseMap)
	require.Nil(t, err)

	// DO NOT CHANGE THIS CONST IN ORDER TO KEEP THE BACKWARDS COMPATIBILITY
	keyForGasUnits := "txGasUnits"
	gasUnitsFloat, ok := costResponseMap[keyForGasUnits].(float64)
	require.True(t, ok)
	require.Equal(t, uint64(10000), uint64(gasUnitsFloat))
}
