package sync

// TestMetaBootstrap extends MetaBootstrap and is used in integration tests as it exposes some funcs
// that are not supposed to be used in production code
// Exported funcs simplify the reproduction of edge cases
type TestMetaBootstrap struct {
	*MetaBootstrap
}

// ForkChoice decides to call (or not) the rollback on the current block from the blockchain structure
func (tmb *TestMetaBootstrap) ForkChoice(revertUsingForkNonce bool) error {
	return tmb.forkChoice(revertUsingForkNonce)
}

// SetProbableHighestNonce sets the probable highest nonce in the contained fork detector
func (tmb *TestMetaBootstrap) SetProbableHighestNonce(nonce uint64) {
	forkDetector, ok := tmb.forkDetector.(*metaForkDetector)
	if !ok {
		log.Error("inner forkdetector impl is not of type metaForkDetector")
		return
	}

	forkDetector.setProbableHighestNonce(nonce)
}
