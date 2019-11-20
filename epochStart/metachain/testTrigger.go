package metachain

import "github.com/ElrondNetwork/elrond-go/epochStart"

// TestTrigger extends start of epoch trigger and is used in integration tests as it exposes some functions
// that are not supposed to be used in production code
// Exported functions simplify the reproduction of edge cases
type TestTrigger struct {
	*trigger
}

// SetTrigger sets the start of epoch trigger
func (t *TestTrigger) SetTrigger(triggerHandler epochStart.TriggerHandler) {
	actualTrigger, ok := triggerHandler.(*trigger)
	if !ok {
		return
	}

	t.trigger = actualTrigger
}

// SetRoundsPerEpoch sets the number of round between epochs
func (t *TestTrigger) SetRoundsPerEpoch(roundsPerEpoch int64) {
	t.roundsPerEpoch = roundsPerEpoch
	if t.minRoundsBetweenEpochs > t.roundsPerEpoch {
		t.minRoundsBetweenEpochs = t.roundsPerEpoch - 1
	}
}

// GetRoundsPerEpoch gets the number of rounds per epoch
func (t *TestTrigger) GetRoundsPerEpoch() int64 {
	return t.roundsPerEpoch
}
