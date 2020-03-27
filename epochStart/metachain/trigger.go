package metachain

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/display"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/storage"
)

var log = logger.GetOrCreate("epochStart/metachain")

const minimumNonceToStartEpoch = 4

// ArgsNewMetaEpochStartTrigger defines struct needed to create a new start of epoch trigger
type ArgsNewMetaEpochStartTrigger struct {
	GenesisTime        time.Time
	Settings           *config.EpochStartConfig
	Epoch              uint32
	EpochStartRound    uint64
	EpochStartNotifier epochStart.EpochStartNotifier
	Marshalizer        marshal.Marshalizer
	Hasher             hashing.Hasher
	Storage            dataRetriever.StorageService
}

type trigger struct {
	isEpochStart                bool
	epoch                       uint32
	epochStartMeta              *block.MetaBlock
	currentRound                uint64
	epochFinalityAttestingRound uint64
	currEpochStartRound         uint64
	prevEpochStartRound         uint64
	roundsPerEpoch              uint64
	minRoundsBetweenEpochs      uint64
	epochStartMetaHash          []byte
	triggerStateKey             []byte
	epochStartTime              time.Time
	mutTrigger                  sync.RWMutex
	epochStartNotifier          epochStart.EpochStartNotifier
	metaHeaderStorage           storage.Storer
	triggerStorage              storage.Storer
	marshalizer                 marshal.Marshalizer
	hasher                      hashing.Hasher
}

// NewEpochStartTrigger creates a trigger for start of epoch
func NewEpochStartTrigger(args *ArgsNewMetaEpochStartTrigger) (*trigger, error) {
	if args == nil {
		return nil, epochStart.ErrNilArgsNewMetaEpochStartTrigger
	}
	if args.Settings == nil {
		return nil, epochStart.ErrNilEpochStartSettings
	}
	if args.Settings.RoundsPerEpoch < 1 {
		return nil, epochStart.ErrInvalidSettingsForEpochStartTrigger
	}
	if args.Settings.MinRoundsBetweenEpochs < 1 {
		return nil, epochStart.ErrInvalidSettingsForEpochStartTrigger
	}
	if args.Settings.MinRoundsBetweenEpochs > args.Settings.RoundsPerEpoch {
		return nil, epochStart.ErrInvalidSettingsForEpochStartTrigger
	}
	if check.IfNil(args.EpochStartNotifier) {
		return nil, epochStart.ErrNilEpochStartNotifier
	}
	if check.IfNil(args.Marshalizer) {
		return nil, epochStart.ErrNilMarshalizer
	}
	if check.IfNil(args.Storage) {
		return nil, epochStart.ErrNilStorageService
	}
	if check.IfNil(args.Hasher) {
		return nil, epochStart.ErrNilHasher
	}

	triggerStorage := args.Storage.GetStorer(dataRetriever.BootstrapUnit)
	if check.IfNil(triggerStorage) {
		return nil, epochStart.ErrNilTriggerStorage
	}

	metaBlockStorage := args.Storage.GetStorer(dataRetriever.MetaBlockUnit)
	if check.IfNil(triggerStorage) {
		return nil, epochStart.ErrNilMetaBlockStorage
	}

	trigStateKey := fmt.Sprintf("initial_value_epoch%d", args.Epoch)

	trigger := &trigger{
		triggerStateKey:             []byte(trigStateKey),
		roundsPerEpoch:              uint64(args.Settings.RoundsPerEpoch),
		epochStartTime:              args.GenesisTime,
		currEpochStartRound:         args.EpochStartRound,
		prevEpochStartRound:         args.EpochStartRound,
		epoch:                       args.Epoch,
		minRoundsBetweenEpochs:      uint64(args.Settings.MinRoundsBetweenEpochs),
		mutTrigger:                  sync.RWMutex{},
		epochFinalityAttestingRound: args.EpochStartRound,
		epochStartNotifier:          args.EpochStartNotifier,
		metaHeaderStorage:           metaBlockStorage,
		triggerStorage:              triggerStorage,
		marshalizer:                 args.Marshalizer,
		hasher:                      args.Hasher,
		epochStartMeta:              &block.MetaBlock{},
	}

	err := trigger.saveState(trigger.triggerStateKey)
	if err != nil {
		return nil, err
	}

	return trigger, nil
}

// IsEpochStart return true if conditions are fulfilled for start of epoch
func (t *trigger) IsEpochStart() bool {
	t.mutTrigger.RLock()
	defer t.mutTrigger.RUnlock()

	return t.isEpochStart
}

// EpochStartRound returns the start round of the current epoch
func (t *trigger) EpochStartRound() uint64 {
	t.mutTrigger.RLock()
	defer t.mutTrigger.RUnlock()

	return t.currEpochStartRound
}

// EpochFinalityAttestingRound returns the round when epoch start block was finalized
func (t *trigger) EpochFinalityAttestingRound() uint64 {
	t.mutTrigger.RLock()
	defer t.mutTrigger.RUnlock()

	return t.epochFinalityAttestingRound
}

// ForceEpochStart sets the conditions for start of epoch to true in case of edge cases
func (t *trigger) ForceEpochStart(round uint64) error {
	t.mutTrigger.Lock()
	defer t.mutTrigger.Unlock()

	if t.currentRound > round {
		return epochStart.ErrSavedRoundIsHigherThanInput
	}
	if t.currentRound == round {
		return epochStart.ErrForceEpochStartCanBeCalledOnlyOnNewRound
	}

	t.currentRound = round

	if t.currentRound-t.currEpochStartRound < t.minRoundsBetweenEpochs {
		return epochStart.ErrNotEnoughRoundsBetweenEpochs
	}

	if !t.isEpochStart {
		t.epoch += 1
	}

	t.currEpochStartRound = t.currentRound
	t.isEpochStart = true
	t.saveCurrentState(round)

	log.Debug("trigger.ForceEpochStart", "isEpochStart", t.isEpochStart)

	return nil
}

// Update processes changes in the trigger
func (t *trigger) Update(round uint64, nonce uint64) {
	t.mutTrigger.Lock()
	defer t.mutTrigger.Unlock()

	t.currentRound = round

	if t.isEpochStart {
		return
	}

	isStartEdgeCase := nonce < minimumNonceToStartEpoch
	isEpochStart := t.currentRound > t.currEpochStartRound+t.roundsPerEpoch
	shouldTriggerEpochStart := isEpochStart && !isStartEdgeCase
	if shouldTriggerEpochStart {
		t.epoch += 1
		t.isEpochStart = true
		t.prevEpochStartRound = t.currEpochStartRound
		t.currEpochStartRound = t.currentRound
		t.saveCurrentState(round)

		msg := fmt.Sprintf("EPOCH %d BEGINS IN ROUND (%d)", t.epoch, t.currentRound)
		log.Debug(display.Headline(msg, "", "#"))
		log.Debug("trigger.Update", "isEpochStart", t.isEpochStart)
		logger.SetCorrelationEpoch(t.epoch)
	}
}

// SetProcessed sets start of epoch to false and cleans underlying structure
func (t *trigger) SetProcessed(header data.HeaderHandler) {
	t.mutTrigger.Lock()
	defer t.mutTrigger.Unlock()

	metaBlock, ok := header.(*block.MetaBlock)
	if !ok {
		return
	}
	if !metaBlock.IsStartOfEpochBlock() {
		return
	}

	metaBuff, errNotCritical := t.marshalizer.Marshal(metaBlock)
	if errNotCritical != nil {
		log.Debug("SetProcessed marshal", "error", errNotCritical.Error())
	}

	metaHash := t.hasher.Compute(string(metaBuff))

	t.currEpochStartRound = metaBlock.Round
	t.epoch = metaBlock.Epoch
	t.epochStartNotifier.NotifyAllPrepare(metaBlock)
	t.epochStartNotifier.NotifyAll(metaBlock)
	t.isEpochStart = false
	t.currentRound = metaBlock.Round
	t.epochStartMeta = metaBlock
	t.epochStartMetaHash = metaHash

	t.saveCurrentState(metaBlock.Round)

	log.Debug("trigger.SetProcessed", "isEpochStart", t.isEpochStart)

	epochStartIdentifier := core.EpochStartIdentifier(metaBlock.Epoch)
	errNotCritical = t.triggerStorage.Put([]byte(epochStartIdentifier), metaBuff)
	if errNotCritical != nil {
		log.Warn("SetProcessed put into triggerStorage", "error", errNotCritical.Error())
	}

	errNotCritical = t.metaHeaderStorage.Put([]byte(epochStartIdentifier), metaBuff)
	if errNotCritical != nil {
		log.Warn("SetProcessed put into metaHdrStorage", "error", errNotCritical.Error())
	}
}

// SetFinalityAttestingRound sets the round which finalized the start of epoch block
func (t *trigger) SetFinalityAttestingRound(round uint64) {
	t.mutTrigger.Lock()
	defer t.mutTrigger.Unlock()

	if round > t.currEpochStartRound {
		t.epochFinalityAttestingRound = round
		t.saveCurrentState(round)
	}
}

// RevertStateToBlock will revert the state of the trigger to the current block
func (t *trigger) RevertStateToBlock(header data.HeaderHandler) error {
	if check.IfNil(header) {
		return epochStart.ErrNilHeaderHandler
	}

	if header.IsStartOfEpochBlock() {
		log.Debug("RevertStateToBlock with epoch start block called")
		t.SetProcessed(header)
		return nil
	}

	t.mutTrigger.RLock()
	prevMeta := t.epochStartMeta
	t.mutTrigger.RUnlock()

	currentHeaderHash, err := core.CalculateHash(t.marshalizer, t.hasher, header)
	if err != nil {
		log.Warn("RevertStateToBlock error on hashing", "error", err)
		return err
	}

	if !bytes.Equal(prevMeta.GetPrevHash(), currentHeaderHash) {
		return nil
	}

	log.Debug("RevertStateToBlock to revert behind epoch start block is called")
	err = t.revert(prevMeta)
	if err != nil {
		return err
	}

	t.mutTrigger.Lock()
	t.currentRound = header.GetRound()
	t.mutTrigger.Unlock()

	return nil
}

func (t *trigger) revert(header data.HeaderHandler) error {
	if check.IfNil(header) || !header.IsStartOfEpochBlock() || header.GetEpoch() == 0 {
		return nil
	}

	metaHdr, ok := header.(*block.MetaBlock)
	if !ok {
		log.Warn("wrong type assertion in Revert metachain trigger")
		return epochStart.ErrWrongTypeAssertion
	}

	t.mutTrigger.Lock()
	defer t.mutTrigger.Unlock()

	prevEpochStartIdentifier := core.EpochStartIdentifier(metaHdr.Epoch - 1)
	epochStartMetaBuff, err := t.metaHeaderStorage.SearchFirst([]byte(prevEpochStartIdentifier))
	if err != nil {
		log.Warn("Revert get previous meta from storage", "error", err)
		return err
	}

	epochStartMeta := &block.MetaBlock{}
	err = t.marshalizer.Unmarshal(epochStartMeta, epochStartMetaBuff)
	if err != nil {
		log.Warn("Revert unmarshal previous meta", "error", err)
		return err
	}

	epochStartIdentifier := core.EpochStartIdentifier(metaHdr.Epoch)
	errNotCritical := t.triggerStorage.Remove([]byte(epochStartIdentifier))
	if errNotCritical != nil {
		log.Debug("Revert remove from triggerStorage", "error", errNotCritical.Error())
	}

	errNotCritical = t.metaHeaderStorage.Remove([]byte(epochStartIdentifier))
	if errNotCritical != nil {
		log.Debug("Revert remove from triggerStorage", "error", errNotCritical.Error())
	}

	t.currEpochStartRound = metaHdr.EpochStart.Economics.PrevEpochStartRound
	t.epoch = metaHdr.Epoch - 1
	t.isEpochStart = false
	t.epochStartMeta = epochStartMeta

	log.Debug("trigger.revert",
		"isEpochStart", t.isEpochStart,
		"epoch", t.epoch,
		"epochStartRound", t.currEpochStartRound)

	return nil
}

// Epoch return the current epoch
func (t *trigger) Epoch() uint32 {
	t.mutTrigger.RLock()
	defer t.mutTrigger.RUnlock()

	return t.epoch
}

// RequestEpochStartIfNeeded request the needed epoch start block if metablock with new epoch was received
func (t *trigger) RequestEpochStartIfNeeded(_ data.HeaderHandler) {
}

// EpochStartMetaHdrHash returns the announcing meta header hash which created the new epoch
func (t *trigger) EpochStartMetaHdrHash() []byte {
	return t.epochStartMetaHash
}

// GetSavedStateKey returns the last saved trigger state key
func (t *trigger) GetSavedStateKey() []byte {
	return t.triggerStateKey
}

// SetEpochStartMetaHdrHash sets the epoch start meta header has
func (t *trigger) SetEpochStartMetaHdrHash(metaHdrHash []byte) {
	t.epochStartMetaHash = metaHdrHash
}

// SetCurrentEpochStartRound sets the round when the current epoch started
func (t *trigger) SetCurrentEpochStartRound(round uint64) {
	t.mutTrigger.Lock()
	t.currEpochStartRound = round
	t.currentRound = round
	t.saveCurrentState(round)
	t.mutTrigger.Unlock()
}

// IsInterfaceNil return true if underlying object is nil
func (t *trigger) IsInterfaceNil() bool {
	return t == nil
}

// needs to be called under locked mutex
func (t *trigger) saveCurrentState(round uint64) {
	t.triggerStateKey = []byte(fmt.Sprint(round))
	err := t.saveState(t.triggerStateKey)
	if err != nil {
		log.Warn("error saving trigger state", "error", err, "key", t.triggerStateKey)
	}
}
