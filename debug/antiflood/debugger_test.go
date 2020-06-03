package antiflood

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/debug"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAntifloodDebugger_InvalidCacheSizeShouldError(t *testing.T) {
	t.Parallel()

	d, err := NewAntifloodDebugger(config.AntifloodDebugConfig{
		CacheSize:                  -1,
		IntervalAutoPrintInSeconds: 10,
	})

	assert.True(t, check.IfNil(d))
	assert.NotNil(t, err)
}

func TestNewAntifloodDebugger_InvalidAutoPrintIntervalShouldError(t *testing.T) {
	t.Parallel()

	d, err := NewAntifloodDebugger(config.AntifloodDebugConfig{
		CacheSize:                  100,
		IntervalAutoPrintInSeconds: 0,
	})

	assert.True(t, check.IfNil(d))
	assert.True(t, errors.Is(err, debug.ErrInvalidValue))
}

func TestNewAntifloodDebugger_ShouldWork(t *testing.T) {
	t.Parallel()

	d, err := NewAntifloodDebugger(config.AntifloodDebugConfig{
		CacheSize:                  100,
		IntervalAutoPrintInSeconds: 10,
	})

	assert.False(t, check.IfNil(d))
	assert.Nil(t, err)
}

//------- AddData

func TestAntifloodDebugger_AddDataNotExistingShouldAdd(t *testing.T) {
	t.Parallel()

	d, _ := NewAntifloodDebugger(config.AntifloodDebugConfig{
		CacheSize:                  100,
		IntervalAutoPrintInSeconds: 1,
	})

	pid := core.PeerID("pid")
	topic := "topic"
	numRejected := uint32(272)
	sizeRejected := uint64(7272)
	d.AddData(pid, topic, numRejected, sizeRejected, true)

	assert.Equal(t, 1, d.cache.Len())
	ev := d.GetData([]byte(string(pid) + topic))
	require.NotNil(t, ev)
	assert.Equal(t, pid, ev.pid)
	assert.Equal(t, topic, ev.topic)
	assert.Equal(t, numRejected, ev.numRejected)
	assert.Equal(t, sizeRejected, ev.sizeRejected)
	assert.True(t, ev.isBlackListed)
}

func TestAntifloodDebugger_AddDataExistingShouldChange(t *testing.T) {
	t.Parallel()

	d, _ := NewAntifloodDebugger(config.AntifloodDebugConfig{
		CacheSize:                  100,
		IntervalAutoPrintInSeconds: 1,
	})

	pid := core.PeerID("pid")
	topic := "topic"
	numRejected := uint32(272)
	sizeRejected := uint64(7272)
	d.AddData(pid, topic, numRejected, sizeRejected, true)
	d.AddData(pid, topic, numRejected, sizeRejected, false)

	assert.Equal(t, 1, d.cache.Len())
	ev := d.GetData([]byte(string(pid) + topic))
	require.NotNil(t, ev)
	assert.Equal(t, pid, ev.pid)
	assert.Equal(t, topic, ev.topic)
	assert.Equal(t, 2*numRejected, ev.numRejected)
	assert.Equal(t, 2*sizeRejected, ev.sizeRejected)
	assert.False(t, ev.isBlackListed)
}

func TestAntifloodDebugger_PrintShouldWork(t *testing.T) {
	t.Parallel()

	d, _ := NewAntifloodDebugger(config.AntifloodDebugConfig{
		CacheSize:                  100,
		IntervalAutoPrintInSeconds: 1,
	})

	pid1 := core.PeerID("pid1")
	pid2 := core.PeerID("pid2")
	numPid1Printed := int32(0)
	numPid2Printed := int32(0)
	d.printEventHandler = func(data string) {
		if strings.Contains(data, pid1.Pretty()) {
			atomic.AddInt32(&numPid1Printed, 1)
		}
		if strings.Contains(data, pid2.Pretty()) {
			atomic.AddInt32(&numPid2Printed, 1)
		}
	}

	topic := "topic"
	numRejected := uint32(272)
	sizeRejected := uint64(7272)
	d.AddData(pid1, topic, numRejected, sizeRejected, true)
	d.AddData(pid2, topic, numRejected, sizeRejected, false)

	time.Sleep(time.Millisecond * 1500)

	assert.Equal(t, int32(1), atomic.LoadInt32(&numPid1Printed))
	assert.Equal(t, int32(1), atomic.LoadInt32(&numPid2Printed))
}

func TestAntifloodDebugger_NothingToPrintShouldWork(t *testing.T) {
	t.Parallel()

	d, _ := NewAntifloodDebugger(config.AntifloodDebugConfig{
		CacheSize:                  100,
		IntervalAutoPrintInSeconds: 1,
	})

	numPrinted := int32(0)
	d.printEventHandler = func(data string) {
		atomic.AddInt32(&numPrinted, 1)
	}

	time.Sleep(time.Millisecond * 2500)

	assert.Equal(t, int32(0), atomic.LoadInt32(&numPrinted))
}

func TestAntifloodDebugger_CloseShouldWork(t *testing.T) {
	t.Parallel()

	d, _ := NewAntifloodDebugger(config.AntifloodDebugConfig{
		CacheSize:                  100,
		IntervalAutoPrintInSeconds: 1,
	})
	numPrinted := int32(0)
	d.printEventHandler = func(data string) {
		atomic.AddInt32(&numPrinted, 1)
	}
	d.AddData("", "", 0, 0, true)

	time.Sleep(time.Millisecond * 2500)
	assert.True(t, atomic.LoadInt32(&numPrinted) > 0)

	err := d.Close()
	assert.Nil(t, err)
	time.Sleep(time.Millisecond * 10)
	atomic.StoreInt32(&numPrinted, 0)

	time.Sleep(time.Millisecond * 2500)

	assert.Equal(t, int32(0), atomic.LoadInt32(&numPrinted))
}
