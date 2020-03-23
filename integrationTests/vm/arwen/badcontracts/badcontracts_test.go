package badcontracts

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/integrationTests/vm/arwen"
)

func Test_Bad_C_NoPanic(t *testing.T) {
	context := arwen.SetupTestContext(t)
	defer context.close()

	context.deploySC("./testdata/bad/bad.wasm", "")

	context.executeSC(&context.Owner, "memoryFault")
	context.executeSC(&context.Owner, "divideByZero")

	context.executeSC(&context.Owner, "badGetOwner1")
	context.executeSC(&context.Owner, "badBigIntStorageStore1")

	context.executeSC(&context.Owner, "badWriteLog1")
	context.executeSC(&context.Owner, "badWriteLog2")
	context.executeSC(&context.Owner, "badWriteLog3")
	context.executeSC(&context.Owner, "badWriteLog4")

	context.executeSC(&context.Owner, "badGetBlockHash1")
	context.executeSC(&context.Owner, "badGetBlockHash2")
	context.executeSC(&context.Owner, "badGetBlockHash3")

	context.executeSC(&context.Owner, "badRecursive")
}

func Test_Empty_C_NoPanic(t *testing.T) {
	context := arwen.SetupTestContext(t)
	defer context.close()

	context.deploySC("./testdata/bad/empty.wasm", "")
	context.executeSC(&context.Owner, "thisDoesNotExist")
}

func Test_Corrupt_NoPanic(t *testing.T) {
	context := arwen.SetupTestContext(t)
	defer context.close()

	context.deploySC("./testdata/bad/corrupt.wasm", "")
	context.executeSC(&context.Owner, "thisDoesNotExist")
}

func Test_NoMemoryDeclaration_NoPanic(t *testing.T) {
	context := arwen.SetupTestContext(t)
	defer context.close()

	context.deploySC("./testdata/bad/nomemory/nomemory.wasm", "")
	context.executeSC(&context.Owner, "memoryFault")
}

func Test_BadFunctionNames_NoPanic(t *testing.T) {
	context := arwen.SetupTestContext(t)
	defer context.close()

	context.deploySC("./testdata/bad/badFunctionNames/badFunctionNames.wasm", "")
}
