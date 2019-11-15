package chronology

import (
	"fmt"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/logger"
	"github.com/ElrondNetwork/elrond-go/ntp"
	"github.com/ElrondNetwork/elrond-go/statusHandler"
)

var log = logger.DefaultLogger()

// srBeforeStartRound defines the state which exist before the start of the round
const srBeforeStartRound = -1

// chronology defines the data needed by the chronology
type chronology struct {
	genesisTime time.Time

	rounder   consensus.Rounder
	syncTimer ntp.SyncTimer

	subroundId int

	subrounds         map[int]int
	subroundHandlers  []consensus.SubroundHandler
	mutSubrounds      sync.RWMutex
	appStatusHandler  core.AppStatusHandler
	epochStartTrigger consensus.EpochStartHandler
}

// NewChronology creates a new chronology object
func NewChronology(
	genesisTime time.Time,
	rounder consensus.Rounder,
	syncTimer ntp.SyncTimer,
	epochStartTrigger consensus.EpochStartHandler,
) (*chronology, error) {

	err := checkNewChronologyParams(
		rounder,
		syncTimer,
		epochStartTrigger,
	)

	if err != nil {
		return nil, err
	}

	chr := chronology{
		genesisTime:       genesisTime,
		rounder:           rounder,
		syncTimer:         syncTimer,
		appStatusHandler:  statusHandler.NewNilStatusHandler(),
		epochStartTrigger: epochStartTrigger,
	}

	chr.subroundId = srBeforeStartRound

	chr.subrounds = make(map[int]int)
	chr.subroundHandlers = make([]consensus.SubroundHandler, 0)

	return &chr, nil
}

func checkNewChronologyParams(
	rounder consensus.Rounder,
	syncTimer ntp.SyncTimer,
	epochStartTrigger consensus.EpochStartHandler,
) error {

	if check.IfNil(rounder) {
		return ErrNilRounder
	}
	if check.IfNil(syncTimer) {
		return ErrNilSyncTimer
	}
	if check.IfNil(epochStartTrigger) {
		return ErrNilEpochStartTrigger
	}

	return nil
}

// SetAppStatusHandler will set the AppStatusHandler which will be used for monitoring
func (chr *chronology) SetAppStatusHandler(ash core.AppStatusHandler) error {
	if ash == nil || ash.IsInterfaceNil() {
		return ErrNilAppStatusHandler
	}

	chr.appStatusHandler = ash
	return nil
}

// AddSubround adds new SubroundHandler implementation to the chronology
func (chr *chronology) AddSubround(subroundHandler consensus.SubroundHandler) {
	chr.mutSubrounds.Lock()

	chr.subrounds[subroundHandler.Current()] = len(chr.subroundHandlers)
	chr.subroundHandlers = append(chr.subroundHandlers, subroundHandler)

	chr.mutSubrounds.Unlock()
}

// RemoveAllSubrounds removes all the SubroundHandler implementations added to the chronology
func (chr *chronology) RemoveAllSubrounds() {
	chr.mutSubrounds.Lock()

	chr.subrounds = make(map[int]int)
	chr.subroundHandlers = make([]consensus.SubroundHandler, 0)

	chr.mutSubrounds.Unlock()
}

// StartRounds actually starts the chronology and calls the DoWork() method of the subroundHandlers loaded
func (chr *chronology) StartRounds() {
	for {
		time.Sleep(time.Millisecond)
		chr.startRound()
	}
}

// Epoch returns the current epoch of the system
func (chr *chronology) Epoch() uint32 {
	return chr.epochStartTrigger.Epoch()
}

// startRound calls the current subround, given by the finished tasks in this round
func (chr *chronology) startRound() {
	if chr.subroundId == srBeforeStartRound {
		chr.updateRound()
	}

	if chr.rounder.Index() <= 0 {
		return
	}

	sr := chr.loadSubroundHandler(chr.subroundId)

	if sr == nil {
		return
	}

	msg := fmt.Sprintf("SUBROUND %s BEGINS", sr.Name())
	log.Info(log.Headline(msg, chr.syncTimer.FormattedCurrentTime(), "."))

	if !sr.DoWork(chr.rounder) {
		chr.subroundId = srBeforeStartRound
		return
	}

	chr.subroundId = sr.Next()
}

// updateRound updates rounds and subrounds depending of the current time and the finished tasks
func (chr *chronology) updateRound() {
	oldRoundIndex := chr.rounder.Index()
	chr.rounder.UpdateRound(chr.genesisTime, chr.syncTimer.CurrentTime())

	if oldRoundIndex != chr.rounder.Index() {
		chr.epochStartTrigger.Update(chr.rounder.Index())

		if chr.epochStartTrigger.IsEpochStart() {
			msg := fmt.Sprintf("EPOCH %d BEGINS", chr.epochStartTrigger.Epoch())
			log.Info(log.Headline(msg, chr.syncTimer.FormattedCurrentTime(), "#"))
		}

		msg := fmt.Sprintf("ROUND %d BEGINS (%d)", chr.rounder.Index(), chr.rounder.TimeStamp().Unix())
		log.Info(log.Headline(msg, chr.syncTimer.FormattedCurrentTime(), "#"))

		chr.initRound()
	}
}

// initRound is called when a new round begins and it does the necessary initialization
func (chr *chronology) initRound() {
	chr.subroundId = srBeforeStartRound

	chr.mutSubrounds.RLock()

	hasSubroundsAndGenesisTimePassed := chr.rounder.Index() > 0 && len(chr.subroundHandlers) > 0

	if hasSubroundsAndGenesisTimePassed {
		chr.subroundId = chr.subroundHandlers[0].Current()
		chr.appStatusHandler.SetUInt64Value(core.MetricCurrentRound, uint64(chr.rounder.Index()))
		chr.appStatusHandler.SetUInt64Value(core.MetricCurrentRoundTimestamp, uint64(chr.rounder.TimeStamp().Unix()))
	}

	chr.mutSubrounds.RUnlock()
}

// loadSubroundHandler returns the implementation of SubroundHandler given by the subroundId
func (chr *chronology) loadSubroundHandler(subroundId int) consensus.SubroundHandler {
	chr.mutSubrounds.RLock()
	defer chr.mutSubrounds.RUnlock()

	index, exist := chr.subrounds[subroundId]

	if !exist {
		return nil
	}

	indexIsOutOfBounds := index < 0 || index >= len(chr.subroundHandlers)

	if indexIsOutOfBounds {
		return nil
	}

	return chr.subroundHandlers[index]
}

// IsInterfaceNil returns true if there is no value under the interface
func (chr *chronology) IsInterfaceNil() bool {
	if chr == nil {
		return true
	}
	return false
}
