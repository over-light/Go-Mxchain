package integrationTests

import "github.com/ElrondNetwork/elrond-go/process"

// TestBootstrapper extends the Bootstrapper interface with some functions intended to be used only in tests
// as it simplifies the reproduction of edge cases
type TestBootstrapper interface {
	process.Bootstrapper
	RollBack(revertUsingForkNonce bool) error
	SetProbableHighestNonce(nonce uint64)
}

type BlockProcessorInitializer interface {
	InitBlockProcessor()
}
