package trigger

import "github.com/ElrondNetwork/elrond-go/update"

const HardforkTriggerString = hardforkTriggerString
const PayloadSeparator = dataSeparator
const HardforkGracePeriod = hardforkGracePeriod

func (t *trigger) SetTimeHandler(handler func() int64) {
	t.getTimestampHandler = handler
}

func (t *trigger) Closers() []update.Closer {
	t.mutClosers.RLock()
	defer t.mutClosers.RUnlock()

	return t.closers
}

func (t *trigger) TriggerExecuting() bool {
	return t.triggerExecuting
}

func (t *trigger) Epoch() uint32 {
	return t.epoch
}

func (t *trigger) SetReceivedExecutingEpoch(
	triggerReceived bool,
	triggerExecuting bool,
	epoch uint32,
) {
	t.triggerExecuting = triggerExecuting
	t.triggerReceived = triggerReceived
	t.epoch = epoch
}

func (t *trigger) ComputeTriggerStartOfEpoch(epoch uint32) bool {
	return t.computeTriggerStartOfEpoch(epoch)
}
