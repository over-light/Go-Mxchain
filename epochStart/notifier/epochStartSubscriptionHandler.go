package notifier

import (
	"sort"
	"sync"

	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/epochStart"
)

// EpochStartNotifier defines which actions should be done for handling new epoch's events
type EpochStartNotifier interface {
	RegisterHandler(handler epochStart.EpochStartHandler)
	UnregisterHandler(handler epochStart.EpochStartHandler)
	NotifyAll(hdr data.HeaderHandler)
	NotifyAllPrepare(hdr data.HeaderHandler)
	IsInterfaceNil() bool
}

// epochStartSubscriptionHandler will handle subscription of function and notifying them
type epochStartSubscriptionHandler struct {
	epochStartHandlers   []epochStart.EpochStartHandler
	mutEpochStartHandler sync.RWMutex
}

// NewEpochStartSubscriptionHandler returns a new instance of epochStartSubscriptionHandler
func NewEpochStartSubscriptionHandler() *epochStartSubscriptionHandler {
	return &epochStartSubscriptionHandler{
		epochStartHandlers:   make([]epochStart.EpochStartHandler, 0),
		mutEpochStartHandler: sync.RWMutex{},
	}
}

// RegisterHandler will subscribe a function so it will be called when NotifyAll method is called
func (essh *epochStartSubscriptionHandler) RegisterHandler(handler epochStart.EpochStartHandler) {
	if handler != nil {
		essh.mutEpochStartHandler.Lock()
		essh.epochStartHandlers = append(essh.epochStartHandlers, handler)
		essh.mutEpochStartHandler.Unlock()
	}
}

// UnregisterHandler will unsubscribe a function from the slice
func (essh *epochStartSubscriptionHandler) UnregisterHandler(handlerToUnregister epochStart.EpochStartHandler) {
	if handlerToUnregister != nil {
		essh.mutEpochStartHandler.Lock()
		for idx, handler := range essh.epochStartHandlers {
			if handler == handlerToUnregister {
				essh.epochStartHandlers = append(essh.epochStartHandlers[:idx], essh.epochStartHandlers[idx+1:]...)
			}
		}
		essh.mutEpochStartHandler.Unlock()
	}
}

// NotifyAll will call all the subscribed functions from the internal slice
func (essh *epochStartSubscriptionHandler) NotifyAll(hdr data.HeaderHandler) {
	essh.mutEpochStartHandler.Lock()

	sort.Slice(essh.epochStartHandlers, func(i, j int) bool {
		return essh.epochStartHandlers[i].NotifyOrder() < essh.epochStartHandlers[j].NotifyOrder()
	})

	for i := 0; i < len(essh.epochStartHandlers); i++ {
		essh.epochStartHandlers[i].EpochStartAction(hdr)
	}
	essh.mutEpochStartHandler.Unlock()
}

// NotifyAllPrepare will call all the subscribed clients to notify them that an epoch change block has been
// observed, but not yet confirmed/committed. Some components may need to do some initialisation/preparation
func (essh *epochStartSubscriptionHandler) NotifyAllPrepare(metaHeader data.HeaderHandler) {
	essh.mutEpochStartHandler.Lock()

	sort.Slice(essh.epochStartHandlers, func(i, j int) bool {
		return essh.epochStartHandlers[i].NotifyOrder() < essh.epochStartHandlers[j].NotifyOrder()
	})

	for i := 0; i < len(essh.epochStartHandlers); i++ {
		essh.epochStartHandlers[i].EpochStartPrepare(metaHeader)
	}
	essh.mutEpochStartHandler.Unlock()
}

// IsInterfaceNil -
func (essh *epochStartSubscriptionHandler) IsInterfaceNil() bool {
	return essh == nil
}
