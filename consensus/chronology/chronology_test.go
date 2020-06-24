package chronology_test

import (
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/data/endProcess"

	"github.com/ElrondNetwork/elrond-go/core/alarm"

	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/consensus/chronology"
	"github.com/ElrondNetwork/elrond-go/consensus/mock"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/stretchr/testify/assert"
)

func initSubroundHandlerMock() *mock.SubroundHandlerMock {
	srm := &mock.SubroundHandlerMock{}
	srm.CurrentCalled = func() int {
		return 0
	}
	srm.NextCalled = func() int {
		return 1
	}
	srm.DoWorkCalled = func(rounder consensus.Rounder) bool {
		return false
	}
	srm.NameCalled = func() string {
		return "(TEST)"
	}
	return srm
}

func TestChronology_NewChronologyNilRounderShouldFail(t *testing.T) {
	t.Parallel()
	syncTimerMock := &mock.SyncTimerMock{}
	genesisTime := time.Now()
	chr, err := chronology.NewChronology(
		genesisTime,
		nil,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	assert.Nil(t, chr)
	assert.Equal(t, err, chronology.ErrNilRounder)
}

func TestChronology_NewChronologyNilSyncerShouldFail(t *testing.T) {
	t.Parallel()
	rounderMock := &mock.RounderMock{}
	genesisTime := time.Now()
	chr, err := chronology.NewChronology(
		genesisTime,
		rounderMock,
		nil,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	assert.Nil(t, chr)
	assert.Equal(t, err, chronology.ErrNilSyncTimer)
}

func TestChronology_NewChronologyNilAlarmSchedulerShouldFail(t *testing.T) {
	t.Parallel()
	rounderMock := &mock.RounderMock{}
	genesisTime := time.Now()
	chr, err := chronology.NewChronology(
		genesisTime,
		rounderMock,
		&mock.SyncTimerMock{},
		nil,
		make(chan endProcess.ArgEndProcess),
	)

	assert.Nil(t, chr)
	assert.Equal(t, err, chronology.ErrNilAlarmScheduler)
}

func TestChronology_NewChronologyNilChanShouldFail(t *testing.T) {
	t.Parallel()
	rounderMock := &mock.RounderMock{}
	genesisTime := time.Now()
	chr, err := chronology.NewChronology(
		genesisTime,
		rounderMock,
		&mock.SyncTimerMock{},
		alarm.NewAlarmScheduler(),
		nil,
	)

	assert.Nil(t, chr)
	assert.Equal(t, err, chronology.ErrNilEndProcessChan)
}

func TestChronology_NewChronologyShouldWork(t *testing.T) {
	t.Parallel()
	rounderMock := &mock.RounderMock{}
	syncTimerMock := &mock.SyncTimerMock{}
	genesisTime := time.Now()
	chr, err := chronology.NewChronology(
		genesisTime,
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	assert.Nil(t, err)
	assert.False(t, check.IfNil(chr))
}

func TestChronology_AddSubroundShouldWork(t *testing.T) {
	t.Parallel()
	rounderMock := &mock.RounderMock{}
	syncTimerMock := &mock.SyncTimerMock{}
	genesisTime := time.Now()
	chr, _ := chronology.NewChronology(
		genesisTime,
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	chr.AddSubround(initSubroundHandlerMock())
	chr.AddSubround(initSubroundHandlerMock())
	chr.AddSubround(initSubroundHandlerMock())

	assert.Equal(t, 3, len(chr.SubroundHandlers()))
}

func TestChronology_RemoveAllSubroundsShouldReturnEmptySubroundHandlersArray(t *testing.T) {
	t.Parallel()
	rounderMock := &mock.RounderMock{}
	syncTimerMock := &mock.SyncTimerMock{}
	genesisTime := time.Now()
	chr, _ := chronology.NewChronology(
		genesisTime,
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	chr.AddSubround(initSubroundHandlerMock())
	chr.AddSubround(initSubroundHandlerMock())
	chr.AddSubround(initSubroundHandlerMock())

	assert.Equal(t, 3, len(chr.SubroundHandlers()))
	chr.RemoveAllSubrounds()
	assert.Equal(t, 0, len(chr.SubroundHandlers()))
}

func TestChronology_StartRoundShouldReturnWhenRoundIndexIsNegative(t *testing.T) {
	t.Parallel()
	rounderMock := &mock.RounderMock{}
	rounderMock.IndexCalled = func() int64 {
		return -1
	}
	rounderMock.BeforeGenesisCalled = func() bool {
		return true
	}
	syncTimerMock := &mock.SyncTimerMock{}
	genesisTime := time.Now()
	chr, _ := chronology.NewChronology(
		genesisTime,
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	srm := initSubroundHandlerMock()
	chr.AddSubround(srm)
	chr.SetSubroundId(0)
	chr.StartRound()

	assert.Equal(t, srm.Current(), chr.SubroundId())
}

func TestChronology_StartRoundShouldReturnWhenLoadSubroundHandlerReturnsNil(t *testing.T) {
	t.Parallel()
	rounderMock := &mock.RounderMock{}
	syncTimerMock := &mock.SyncTimerMock{}
	genesisTime := time.Now()
	chr, _ := chronology.NewChronology(
		genesisTime,
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	initSubroundHandlerMock()
	chr.StartRound()

	assert.Equal(t, -1, chr.SubroundId())
}

func TestChronology_StartRoundShouldReturnWhenDoWorkReturnsFalse(t *testing.T) {
	t.Parallel()
	rounderMock := &mock.RounderMock{}
	rounderMock.UpdateRound(rounderMock.TimeStamp(), rounderMock.TimeStamp().Add(rounderMock.TimeDuration()))
	syncTimerMock := &mock.SyncTimerMock{}
	genesisTime := time.Now()
	chr, _ := chronology.NewChronology(
		genesisTime,
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	srm := initSubroundHandlerMock()
	chr.AddSubround(srm)
	chr.SetSubroundId(0)
	chr.StartRound()

	assert.Equal(t, -1, chr.SubroundId())
}

func TestChronology_StartRoundShouldWork(t *testing.T) {
	t.Parallel()
	rounderMock := &mock.RounderMock{}
	rounderMock.UpdateRound(rounderMock.TimeStamp(), rounderMock.TimeStamp().Add(rounderMock.TimeDuration()))
	syncTimerMock := &mock.SyncTimerMock{}
	genesisTime := time.Now()
	chr, _ := chronology.NewChronology(
		genesisTime,
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	srm := initSubroundHandlerMock()
	srm.DoWorkCalled = func(rounder consensus.Rounder) bool {
		return true
	}
	chr.AddSubround(srm)
	chr.SetSubroundId(0)
	chr.StartRound()

	assert.Equal(t, srm.Next(), chr.SubroundId())
}

func TestChronology_UpdateRoundShouldInitRound(t *testing.T) {
	t.Parallel()
	rounderMock := &mock.RounderMock{}
	syncTimerMock := &mock.SyncTimerMock{}
	genesisTime := time.Now()
	chr, _ := chronology.NewChronology(
		genesisTime,
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	srm := initSubroundHandlerMock()
	chr.AddSubround(srm)
	chr.UpdateRound()

	assert.Equal(t, srm.Current(), chr.SubroundId())
}

func TestChronology_LoadSubrounderShouldReturnNilWhenSubroundHandlerNotExists(t *testing.T) {
	t.Parallel()
	rounderMock := &mock.RounderMock{}
	syncTimerMock := &mock.SyncTimerMock{}
	genesisTime := time.Now()
	chr, _ := chronology.NewChronology(
		genesisTime,
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	assert.Nil(t, chr.LoadSubroundHandler(0))
}

func TestChronology_LoadSubrounderShouldReturnNilWhenIndexIsOutOfBound(t *testing.T) {
	t.Parallel()
	rounderMock := &mock.RounderMock{}
	syncTimerMock := &mock.SyncTimerMock{}
	genesisTime := time.Now()
	chr, _ := chronology.NewChronology(
		genesisTime,
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	chr.AddSubround(initSubroundHandlerMock())
	chr.SetSubroundHandlers(make([]consensus.SubroundHandler, 0))

	assert.Nil(t, chr.LoadSubroundHandler(0))
}

func TestChronology_InitRoundShouldNotSetSubroundWhenRoundIndexIsNegative(t *testing.T) {
	t.Parallel()
	rounderMock := &mock.RounderMock{}
	syncTimerMock := &mock.SyncTimerMock{}
	chr, _ := chronology.NewChronology(
		syncTimerMock.CurrentTime(),
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	chr.AddSubround(initSubroundHandlerMock())
	rounderMock.IndexCalled = func() int64 {
		return -1
	}
	rounderMock.BeforeGenesisCalled = func() bool {
		return true
	}
	chr.InitRound()

	assert.Equal(t, -1, chr.SubroundId())
}

func TestChronology_InitRoundShouldSetSubroundWhenRoundIndexIsPositive(t *testing.T) {
	t.Parallel()
	rounderMock := &mock.RounderMock{}
	rounderMock.UpdateRound(rounderMock.TimeStamp(), rounderMock.TimeStamp().Add(rounderMock.TimeDuration()))
	syncTimerMock := &mock.SyncTimerMock{}
	chr, _ := chronology.NewChronology(
		syncTimerMock.CurrentTime(),
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	sr := initSubroundHandlerMock()
	chr.AddSubround(sr)
	chr.InitRound()

	assert.Equal(t, sr.Current(), chr.SubroundId())
}

func TestChronology_StartRoundShouldNotUpdateRoundWhenCurrentRoundIsNotFinished(t *testing.T) {
	t.Parallel()
	rounderMock := &mock.RounderMock{}
	syncTimerMock := &mock.SyncTimerMock{}
	chr, _ := chronology.NewChronology(
		syncTimerMock.CurrentTime(),
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	chr.SetSubroundId(0)
	chr.StartRound()

	assert.Equal(t, int64(0), rounderMock.Index())
}

func TestChronology_StartRoundShouldUpdateRoundWhenCurrentRoundIsFinished(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{}
	syncTimerMock := &mock.SyncTimerMock{}
	chr, _ := chronology.NewChronology(
		syncTimerMock.CurrentTime(),
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	chr.SetSubroundId(-1)
	chr.StartRound()

	assert.Equal(t, int64(1), rounderMock.Index())
}

func TestChronology_SetAppStatusHandlerWithNilValueShouldErr(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{}
	syncTimerMock := &mock.SyncTimerMock{}
	chr, _ := chronology.NewChronology(
		syncTimerMock.CurrentTime(),
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)
	err := chr.SetAppStatusHandler(nil)

	assert.Equal(t, err, chronology.ErrNilAppStatusHandler)
}

func TestChronology_SetAppStatusHandlerWithOkValueShouldPass(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{}
	syncTimerMock := &mock.SyncTimerMock{}
	chr, _ := chronology.NewChronology(
		syncTimerMock.CurrentTime(),
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	err := chr.SetAppStatusHandler(&mock.AppStatusHandlerMock{})

	assert.Nil(t, err)
}

func TestChronology_CheckIfStatusHandlerWorks(t *testing.T) {
	t.Parallel()

	chanDone := make(chan bool, 2)
	rounderMock := &mock.RounderMock{}
	syncTimerMock := &mock.SyncTimerMock{}
	chr, _ := chronology.NewChronology(
		syncTimerMock.CurrentTime(),
		rounderMock,
		syncTimerMock,
		alarm.NewAlarmScheduler(),
		make(chan endProcess.ArgEndProcess),
	)

	err := chr.SetAppStatusHandler(&mock.AppStatusHandlerStub{
		SetUInt64ValueHandler: func(key string, value uint64) {
			chanDone <- true
		},
	})

	assert.Nil(t, err)

	srm := initSubroundHandlerMock()
	srm.DoWorkCalled = func(rounder consensus.Rounder) bool {
		return true
	}

	chr.AddSubround(srm)
	chr.StartRound()

	select {
	case <-chanDone:
	case <-time.After(1 * time.Second):
		assert.Fail(t, "AppStatusHandler not working")
	}
}
