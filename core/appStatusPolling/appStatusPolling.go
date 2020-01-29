package appStatusPolling

import (
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
)

// AppStatusPolling will update an AppStatusHandler by polling components at a predefined interval
type AppStatusPolling struct {
	pollingDuration     time.Duration
	mutRegisteredFunc   sync.RWMutex
	registeredFunctions []func(appStatusHandler core.AppStatusHandler)
	appStatusHandler    core.AppStatusHandler
}

// NewAppStatusPolling will return an instance of AppStatusPolling
func NewAppStatusPolling(appStatusHandler core.AppStatusHandler, pollingDurationSec int) (*AppStatusPolling, error) {
	if check.IfNil(appStatusHandler) {
		return nil, ErrNilAppStatusHandler
	}
	if pollingDurationSec < 1 {
		return nil, ErrPollingDurationNegative
	}
	return &AppStatusPolling{
		pollingDuration:  time.Duration(pollingDurationSec) * time.Second,
		appStatusHandler: appStatusHandler,
	}, nil
}

// RegisterPollingFunc will register a new handler function
func (asp *AppStatusPolling) RegisterPollingFunc(handler func(appStatusHandler core.AppStatusHandler)) error {
	if handler == nil {
		return ErrNilHandlerFunc
	}
	asp.mutRegisteredFunc.Lock()
	asp.registeredFunctions = append(asp.registeredFunctions, handler)
	asp.mutRegisteredFunc.Unlock()
	return nil
}

// Poll will notify the AppStatusHandler at a given time
func (asp *AppStatusPolling) Poll() {
	go func() {
		for {
			time.Sleep(asp.pollingDuration)

			asp.mutRegisteredFunc.RLock()
			for _, handler := range asp.registeredFunctions {
				handler(asp.appStatusHandler)
			}
			asp.mutRegisteredFunc.RUnlock()
		}
	}()
}
