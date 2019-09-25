package integrationTests

import "github.com/ElrondNetwork/elrond-go/process"

// TestBootstrapper extends the Bootstrapper interface with some funcs intended to be used only in tests
// as it simplifies the reproduction of edge cases
type TestBootstrapper interface {
	process.Bootstrapper
	ManualRollback() error
	SetProbableHighestNonce(nonce uint64)
}
