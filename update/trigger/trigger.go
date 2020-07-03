package trigger

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"time"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/endProcess"
	"github.com/ElrondNetwork/elrond-go/facade"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/update"
)

const hardforkTriggerString = "hardfork trigger"
const dataSeparator = "@"
const hardforkGracePeriod = time.Minute * 5
const epochGracePeriod = 4
const minTimeToWaitAfterHardforkInMinutes = 2
const minimumEpochForHarfork = 1

var _ facade.HardforkTrigger = (*trigger)(nil)
var log = logger.GetOrCreate("update/trigger")

// ArgHardforkTrigger contains the
type ArgHardforkTrigger struct {
	TriggerPubKeyBytes        []byte
	SelfPubKeyBytes           []byte
	Enabled                   bool
	EnabledAuthenticated      bool
	ArgumentParser            process.ArgumentsParser
	EpochProvider             update.EpochHandler
	ExportFactoryHandler      update.ExportFactoryHandler
	CloseAfterExportInMinutes uint32
	ChanStopNodeProcess       chan endProcess.ArgEndProcess
	EpochConfirmedNotifier    update.EpochChangeConfirmedNotifier
	ImportStartHandler        update.ImportStartHandler
}

// trigger implements a hardfork trigger that is able to notify a set list of handlers if this instance gets triggered
// by external events
type trigger struct {
	triggerPubKey                []byte
	selfPubKey                   []byte
	enabled                      bool
	enabledAuthenticated         bool
	isTriggerSelf                bool
	mutTriggered                 sync.RWMutex
	triggerReceived              bool
	triggerExecuting             bool
	shouldTriggerFromEpochChange bool
	recordedTriggerMessage       []byte
	epoch                        uint32
	getTimestampHandler          func() int64
	argumentParser               process.ArgumentsParser
	epochProvider                update.EpochHandler
	exportFactoryHandler         update.ExportFactoryHandler
	closeAfterInMinutes          uint32
	chanStopNodeProcess          chan endProcess.ArgEndProcess
	epochConfirmedNotifier       update.EpochChangeConfirmedNotifier
	mutClosers                   sync.RWMutex
	closers                      []update.Closer
	chanTriggerReceived          chan struct{}
	importStartHandler           update.ImportStartHandler
}

// NewTrigger returns the trigger instance
func NewTrigger(arg ArgHardforkTrigger) (*trigger, error) {
	if len(arg.TriggerPubKeyBytes) == 0 {
		return nil, fmt.Errorf("%w hardfork trigger public key bytes length is 0", update.ErrInvalidValue)
	}
	if len(arg.SelfPubKeyBytes) == 0 {
		return nil, fmt.Errorf("%w self public key bytes length is 0", update.ErrInvalidValue)
	}
	if check.IfNil(arg.ArgumentParser) {
		return nil, update.ErrNilArgumentParser
	}
	if check.IfNil(arg.EpochProvider) {
		return nil, update.ErrNilEpochHandler
	}
	if check.IfNil(arg.ExportFactoryHandler) {
		return nil, update.ErrNilExportFactoryHandler
	}
	if arg.ChanStopNodeProcess == nil {
		return nil, update.ErrNilChanStopNodeProcess
	}
	if check.IfNil(arg.EpochConfirmedNotifier) {
		return nil, update.ErrNilEpochConfirmedNotifier
	}
	if arg.CloseAfterExportInMinutes < minTimeToWaitAfterHardforkInMinutes {
		return nil, fmt.Errorf("%w, minimum time to wait in minutes: %d",
			update.ErrInvalidTimeToWaitAfterHardfork,
			minTimeToWaitAfterHardforkInMinutes,
		)
	}
	if check.IfNil(arg.ImportStartHandler) {
		return nil, update.ErrNilImportStartHandler
	}

	t := &trigger{
		enabled:              arg.Enabled,
		enabledAuthenticated: arg.EnabledAuthenticated,
		selfPubKey:           arg.SelfPubKeyBytes,
		triggerPubKey:        arg.TriggerPubKeyBytes,
		triggerReceived:      false,
		triggerExecuting:     false,
		argumentParser:       arg.ArgumentParser,
		epochProvider:        arg.EpochProvider,
		exportFactoryHandler: arg.ExportFactoryHandler,
		closeAfterInMinutes:  arg.CloseAfterExportInMinutes,
		chanStopNodeProcess:  arg.ChanStopNodeProcess,
		closers:              make([]update.Closer, 0),
		chanTriggerReceived:  make(chan struct{}, 1), //buffer with one value as there might be async calls
		importStartHandler:   arg.ImportStartHandler,
	}

	t.isTriggerSelf = bytes.Equal(arg.TriggerPubKeyBytes, arg.SelfPubKeyBytes)
	t.getTimestampHandler = t.getCurrentUnixTime
	arg.EpochConfirmedNotifier.RegisterForEpochChangeConfirmed(t.epochConfirmed)

	return t, nil
}

func (t *trigger) getCurrentUnixTime() int64 {
	return time.Now().Unix()
}

func (t *trigger) epochConfirmed(epoch uint32) {
	if !t.enabled {
		return
	}

	shouldStartHardfork := t.computeTriggerStartOfEpoch(epoch)
	if !shouldStartHardfork {
		return
	}

	t.doTrigger()
}

func (t *trigger) computeTriggerStartOfEpoch(receivedTrigger uint32) bool {
	t.mutTriggered.Lock()
	defer t.mutTriggered.Unlock()

	if !t.triggerReceived {
		return false
	}
	if t.triggerExecuting {
		return false
	}
	if receivedTrigger < t.epoch {
		return false
	}

	t.triggerExecuting = true
	return true
}

// Trigger will start the hardfork process
func (t *trigger) Trigger(epoch uint32) error {
	if !t.enabled {
		return update.ErrTriggerNotEnabled
	}

	log.Debug("hardfork trigger", "epoch", epoch)

	if epoch < minimumEpochForHarfork {
		return fmt.Errorf("%w, minimum epoch accepted is %d", update.ErrInvalidEpoch, minimumEpochForHarfork)
	}

	shouldTrigger, err := t.computeAndSetTrigger(epoch, nil) //original payload is nil because this node is the originator
	if err != nil {
		return err
	}
	if !shouldTrigger {
		log.Debug("hardfork won't trigger now, will wait for epoch change")

		return nil
	}

	t.doTrigger()

	return nil
}

// computeAndSetTrigger needs to do 2 things atomically: set the original payload and epoch and determine if the trigger
// can be called
func (t *trigger) computeAndSetTrigger(epoch uint32, originalPayload []byte) (bool, error) {
	t.mutTriggered.Lock()
	defer t.mutTriggered.Unlock()

	t.triggerReceived = true
	if t.triggerExecuting {
		return false, update.ErrTriggerAlreadyInAction
	}
	t.epoch = epoch
	if len(originalPayload) > 0 {
		t.recordedTriggerMessage = originalPayload
	}

	if epoch > t.epochProvider.MetaEpoch() {
		t.shouldTriggerFromEpochChange = true
		return false, nil
	}

	if t.shouldTriggerFromEpochChange {
		return false, nil
	}

	t.triggerExecuting = true

	//writing on the notification chan should not be blocking as to allow self to initiate the hardfork process
	select {
	case t.chanTriggerReceived <- struct{}{}:
	default:
	}

	return true, nil
}

func (t *trigger) doTrigger() {
	t.callClose()
	t.exportAll()
}

func (t *trigger) exportAll() {
	t.mutTriggered.Lock()
	defer t.mutTriggered.Unlock()

	log.Debug("hardfork trigger exportAll called")

	epoch := t.epoch

	go func() {
		exportHandler, err := t.exportFactoryHandler.Create()
		if err != nil {
			log.Error("error while creating export handler", "error", err)
			return
		}

		log.Info("started hardFork export process")
		err = exportHandler.ExportAll(epoch)
		if err != nil {
			log.Error("error while exporting data", "error", err)
			return
		}
		log.Info("finished hardFork export process")
		errNotCritical := t.importStartHandler.SetStartImport()
		if errNotCritical != nil {
			log.Error("error setting the node to start the import after the restart",
				"error", errNotCritical)
		}

		wait := time.Duration(t.closeAfterInMinutes) * time.Minute
		log.Info("node will still be active for", "time duration", wait)

		time.Sleep(wait)
		argument := endProcess.ArgEndProcess{
			Reason:      "HardForkExport",
			Description: "Node finished the export process with success",
		}
		t.chanStopNodeProcess <- argument
	}()
}

// TriggerReceived is called whenever a trigger is received from the p2p side
func (t *trigger) TriggerReceived(originalPayload []byte, data []byte, pkBytes []byte) (bool, error) {
	receivedFunction, arguments, err := t.argumentParser.ParseCallData(string(data))
	if err != nil {
		return false, nil
	}

	if receivedFunction != hardforkTriggerString {
		return false, nil
	}

	isTriggerEnabled := t.enabled && t.enabledAuthenticated
	if !isTriggerEnabled {
		//should not return error as to allow the message to get to other peers
		return true, nil
	}

	if !bytes.Equal(pkBytes, t.triggerPubKey) {
		return true, update.ErrTriggerPubKeyMismatch
	}

	if len(arguments) != 2 {
		return true, update.ErrIncorrectHardforkMessage
	}

	timestamp, err := t.getIntFromArgument(string(arguments[0]))
	if err != nil {
		return true, err
	}

	currentTimeStamp := t.getTimestampHandler()
	if timestamp+int64(hardforkGracePeriod.Seconds()) < currentTimeStamp {
		return true, fmt.Errorf("%w message timestamp out of grace period message", update.ErrIncorrectHardforkMessage)
	}

	epoch, err := t.getIntFromArgument(string(arguments[1]))
	if err != nil {
		return true, err
	}
	if epoch < minimumEpochForHarfork {
		return true, fmt.Errorf("%w, minimum epoch accepted is %d", update.ErrInvalidEpoch, minimumEpochForHarfork)
	}

	currentEpoch := int64(t.epochProvider.MetaEpoch())
	if currentEpoch-epoch > epochGracePeriod {
		return true, fmt.Errorf("%w epoch out of grace period", update.ErrIncorrectHardforkMessage)
	}

	shouldTrigger, err := t.computeAndSetTrigger(uint32(epoch), originalPayload)
	if err != nil {
		log.Debug("received trigger", "status", err)
		return true, nil
	}
	if !shouldTrigger {
		return true, nil
	}

	t.doTrigger()

	return true, nil
}

func (t *trigger) callClose() {
	log.Debug("calling close on all registered instances")

	t.mutClosers.RLock()
	for _, c := range t.closers {
		err := c.Close()
		if err != nil {
			log.Warn("error closing registered instance", "error", err)
		}
	}
	t.mutClosers.RUnlock()
}

func (t *trigger) getIntFromArgument(value string) (int64, error) {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w, convert error, `%s` is not a valid int",
			update.ErrIncorrectHardforkMessage,
			value,
		)
	}

	return n, nil
}

// IsSelfTrigger returns true if self public key is the trigger public key set in the configs
func (t *trigger) IsSelfTrigger() bool {
	return t.isTriggerSelf
}

// RecordedTriggerMessage returns the trigger message that set the trigger
func (t *trigger) RecordedTriggerMessage() ([]byte, bool) {
	t.mutTriggered.RLock()
	defer t.mutTriggered.RUnlock()

	return t.recordedTriggerMessage, t.triggerReceived
}

// CreateData creates a correct hardfork trigger message based on the identifier and the additional information
func (t *trigger) CreateData() []byte {
	t.mutTriggered.RLock()
	payload := hardforkTriggerString +
		dataSeparator + hex.EncodeToString([]byte(fmt.Sprintf("%d", t.getTimestampHandler()))) +
		dataSeparator + hex.EncodeToString([]byte(fmt.Sprintf("%d", t.epoch)))
	t.mutTriggered.RUnlock()

	return []byte(payload)
}

// AddCloser will add a closer interface on the existing list
func (t *trigger) AddCloser(closer update.Closer) error {
	if check.IfNil(closer) {
		return update.ErrNilCloser
	}

	t.mutClosers.Lock()
	t.closers = append(t.closers, closer)
	t.mutClosers.Unlock()

	return nil
}

// NotifyTriggerReceived will write a struct{}{} on the provided channel as soon as a trigger is received
// this is done to decrease the latency of the heartbeat sending system
func (t *trigger) NotifyTriggerReceived() <-chan struct{} {
	return t.chanTriggerReceived
}

// IsInterfaceNil returns true if there is no value under the interface
func (t *trigger) IsInterfaceNil() bool {
	return t == nil
}
