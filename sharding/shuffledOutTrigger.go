package sharding

import "sync"

var _ ShuffledOutHandler = (*shuffledOutTrigger)(nil)

type shuffledOutTrigger struct {
	ownPubKey         []byte
	currentShardID    uint32
	handlers          []func(newShardID uint32)
	mutHandlers       sync.RWMutex
	endProcessHandler func() error
}

// NewShuffledOutTrigger returns a new instance of shuffledOutTrigger
func NewShuffledOutTrigger(ownPubKey []byte, currentShardID uint32, endProcessHandler func() error) (*shuffledOutTrigger, error) {
	if ownPubKey == nil {
		return nil, ErrNilOwnPublicKey
	}
	if endProcessHandler == nil {
		return nil, ErrNilEndOfProcessingHandler
	}
	return &shuffledOutTrigger{
		ownPubKey:         ownPubKey,
		currentShardID:    currentShardID,
		endProcessHandler: endProcessHandler,
	}, nil
}

// Process will compare the received shard ID and the existing one and do some processing in case that the received
// shard ID is different
func (sot *shuffledOutTrigger) Process(newShardID uint32) error {
	if sot.currentShardID == newShardID {
		return nil
	}

	sot.currentShardID = newShardID
	sot.notifyAllHandlers(newShardID)
	return sot.endProcessHandler()
}

func (sot *shuffledOutTrigger) notifyAllHandlers(newShardID uint32) {
	sot.mutHandlers.RLock()
	for _, handler := range sot.handlers {
		handler(newShardID)
	}
	sot.mutHandlers.RUnlock()
}

// CurrentShardID return the current shard ID of the node
func (sot *shuffledOutTrigger) CurrentShardID() uint32 {
	return sot.currentShardID
}

// RegisterHandler will append the provided handler to the handlers slice
func (sot *shuffledOutTrigger) RegisterHandler(handler func(newShardID uint32)) {
	sot.mutHandlers.Lock()
	sot.handlers = append(sot.handlers, handler)
	sot.mutHandlers.Unlock()
}

// IsInterfaceNil returns true if there is no value under the interface
func (sot *shuffledOutTrigger) IsInterfaceNil() bool {
	return sot == nil
}
