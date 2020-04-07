package builtInFunctions

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"
)

func createMockArguments() ArgsCreateBuiltInFunctionContainer {
	gasMap := make(map[string]map[string]uint64)
	fillGasMapInternal(gasMap, 1)

	args := ArgsCreateBuiltInFunctionContainer{
		GasMap:               gasMap,
		MapDNSAddresses:      make(map[string]struct{}),
		EnableUserNameChange: false,
	}

	return args
}

func fillGasMapInternal(gasMap map[string]map[string]uint64, value uint64) map[string]map[string]uint64 {
	gasMap[core.BaseOperationCost] = fillGasMapBaseOperationCosts(value)
	gasMap[core.BuiltInCost] = fillGasMapBuiltInCosts(value)

	return gasMap
}

func fillGasMapBaseOperationCosts(value uint64) map[string]uint64 {
	gasMap := make(map[string]uint64)
	gasMap["StorePerByte"] = value
	gasMap["DataCopyPerByte"] = value
	gasMap["ReleasePerByte"] = value
	gasMap["PersistPerByte"] = value
	gasMap["CompilePerByte"] = value

	return gasMap
}

func fillGasMapBuiltInCosts(value uint64) map[string]uint64 {
	gasMap := make(map[string]uint64)
	gasMap["ClaimDeveloperRewards"] = value
	gasMap["ChangeOwnerAddress"] = value
	gasMap["SaveUserName"] = value

	return gasMap
}

func TestCreateBuiltInFunctionContainer_Errors(t *testing.T) {
	t.Parallel()

}
