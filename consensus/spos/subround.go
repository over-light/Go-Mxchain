package spos

import (
	"time"

	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/statusHandler"
)

var _ consensus.SubroundHandler = (*Subround)(nil)

// Subround struct contains the needed data for one Subround and the Subround properties. It defines a Subround
// with it's properties (it's ID, next Subround ID, it's duration, it's name) and also it has some handler functions
// which should be set. Job function will be the main function of this Subround, Extend function will handle the overtime
// situation of the Subround and Check function will decide if in this Subround the consensus is achieved
type Subround struct {
	ConsensusCoreHandler
	*ConsensusState

	previous  int
	current   int
	next      int
	startTime int64
	endTime   int64
	name      string
	chainID   []byte

	consensusStateChangedChannel chan bool
	executeStoredMessages        func()
	appStatusHandler             core.AppStatusHandler

	Job    func() bool          // method does the Subround Job and send the result to the peers
	Check  func() bool          // method checks if the consensus of the Subround is done
	Extend func(subroundId int) // method is called when round time is out
}

// NewSubround creates a new SubroundId object
func NewSubround(
	previous int,
	current int,
	next int,
	startTime int64,
	endTime int64,
	name string,
	consensusState *ConsensusState,
	consensusStateChangedChannel chan bool,
	executeStoredMessages func(),
	container ConsensusCoreHandler,
	chainID []byte,
) (*Subround, error) {
	err := checkNewSubroundParams(
		consensusState,
		consensusStateChangedChannel,
		executeStoredMessages,
		container,
		chainID,
	)
	if err != nil {
		return nil, err
	}

	sr := Subround{
		ConsensusCoreHandler:         container,
		ConsensusState:               consensusState,
		previous:                     previous,
		current:                      current,
		next:                         next,
		startTime:                    startTime,
		endTime:                      endTime,
		name:                         name,
		chainID:                      chainID,
		consensusStateChangedChannel: consensusStateChangedChannel,
		executeStoredMessages:        executeStoredMessages,
		Job:                          nil,
		Check:                        nil,
		Extend:                       nil,
		appStatusHandler:             statusHandler.NewNilStatusHandler(),
	}

	return &sr, nil
}

func checkNewSubroundParams(
	state *ConsensusState,
	consensusStateChangedChannel chan bool,
	executeStoredMessages func(),
	container ConsensusCoreHandler,
	chainID []byte,
) error {
	err := ValidateConsensusCore(container)
	if err != nil {
		return err
	}
	if consensusStateChangedChannel == nil {
		return ErrNilChannel
	}
	if state == nil {
		return ErrNilConsensusState
	}
	if executeStoredMessages == nil {
		return ErrNilExecuteStoredMessages
	}
	if len(chainID) == 0 {
		return ErrInvalidChainID
	}

	return nil
}

// DoWork method actually does the work of this Subround. First it tries to do the Job of the Subround then it will
// Check the consensus. If the upper time limit of this Subround is reached, the Extend method will be called before
// returning. If this method returns true the chronology will advance to the next Subround.
func (sr *Subround) DoWork(rounder consensus.Rounder) bool {
	if sr.Job == nil || sr.Check == nil {
		return false
	}

	// execute stored messages which were received in this new round but before this initialisation
	go sr.executeStoredMessages()

	startTime := rounder.TimeStamp()
	maxTime := rounder.TimeDuration() * MaxThresholdPercent / 100

	sr.Job()
	if sr.Check() {
		return true
	}

	for {
		select {
		case <-sr.consensusStateChangedChannel:
			if sr.Check() {
				return true
			}
		case <-time.After(rounder.RemainingTime(startTime, maxTime)):
			if sr.Extend != nil {
				sr.RoundCanceled = true
				sr.Extend(sr.current)
			}

			return false
		}
	}
}

// Previous method returns the ID of the previous Subround
func (sr *Subround) Previous() int {
	return sr.previous
}

// Current method returns the ID of the current Subround
func (sr *Subround) Current() int {
	return sr.current
}

// Next method returns the ID of the next Subround
func (sr *Subround) Next() int {
	return sr.next
}

// StartTime method returns the start time of the Subround
func (sr *Subround) StartTime() int64 {
	return sr.startTime
}

// EndTime method returns the upper time limit of the Subround
func (sr *Subround) EndTime() int64 {
	return sr.endTime
}

// Name method returns the name of the Subround
func (sr *Subround) Name() string {
	return sr.name
}

// ChainID method returns the current chain ID
func (sr *Subround) ChainID() []byte {
	return sr.chainID
}

// SetAppStatusHandler method sets appStatusHandler
func (sr *Subround) SetAppStatusHandler(ash core.AppStatusHandler) error {
	if check.IfNil(ash) {
		return ErrNilAppStatusHandler
	}
	sr.appStatusHandler = ash

	return nil
}

// AppStatusHandler method returns the appStatusHandler instance
func (sr *Subround) AppStatusHandler() core.AppStatusHandler {
	return sr.appStatusHandler
}

// ConsensusChannel method returns the consensus channel
func (sr *Subround) ConsensusChannel() chan bool {
	return sr.consensusStateChangedChannel
}

// UpdateValidatorPeerHonesty will increase or decrease the honesty of the given validator public key
func (sr *Subround) UpdateValidatorPeerHonesty(pk string) {
	if !sr.IsNodeInConsensusGroup(pk) {
		sr.PeerHonestyHandler().Decrease(
			sr.Rounder().Index(),
			pk,
			GetConsensusTopicID(sr.ShardCoordinator()),
			ValidatorPeerHonestyDecreaseFactor)

		return
	}

	sr.PeerHonestyHandler().Increase(
		sr.Rounder().Index(),
		pk,
		GetConsensusTopicID(sr.ShardCoordinator()),
		ValidatorPeerHonestyIncreaseFactor)
}

// UpdateLeaderPeerHonesty will increase or decrease the honesty of the given leader public key
func (sr *Subround) UpdateLeaderPeerHonesty(pk string) {
	if !sr.IsNodeLeaderInCurrentRound(pk) { // is NOT this node leader in current round?
		sr.PeerHonestyHandler().Decrease(
			sr.Rounder().Index(),
			pk,
			GetConsensusTopicID(sr.ShardCoordinator()),
			LeaderPeerHonestyDecreaseFactor)

		return
	}

	sr.PeerHonestyHandler().Increase(
		sr.Rounder().Index(),
		pk,
		GetConsensusTopicID(sr.ShardCoordinator()),
		LeaderPeerHonestyIncreaseFactor)
}

// IsInterfaceNil returns true if there is no value under the interface
func (sr *Subround) IsInterfaceNil() bool {
	return sr == nil
}
