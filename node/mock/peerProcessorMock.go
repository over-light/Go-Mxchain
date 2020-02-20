package mock

import (
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/state"
)

// ValidatorStatisticsProcessorMock -
type ValidatorStatisticsProcessorMock struct {
	UpdatePeerStateCalled           func(header data.HeaderHandler) ([]byte, error)
	RevertPeerStateCalled           func(header data.HeaderHandler) error
	IsInterfaceNilCalled            func() bool
	RevertPeerStateToSnapshotCalled func(snapshot int) error
	GetPeerAccountCalled            func(address []byte) (state.PeerAccountHandler, error)
	CommitCalled                    func() ([]byte, error)
	RootHashCalled                  func() ([]byte, error)
	IsPruningEnabledCalled          func() bool
	SnapshotStateCalled             func([]byte)
	PruneTrieCalled                 func([]byte) error
	CancelPruneCalled               func([]byte)
}

// UpdatePeerState -
func (vsp *ValidatorStatisticsProcessorMock) UpdatePeerState(header data.HeaderHandler) ([]byte, error) {
	if vsp.UpdatePeerStateCalled != nil {
		return vsp.UpdatePeerStateCalled(header)
	}
	return nil, nil
}

// RevertPeerState -
func (vsp *ValidatorStatisticsProcessorMock) RevertPeerState(header data.HeaderHandler) error {
	if vsp.RevertPeerStateCalled != nil {
		return vsp.RevertPeerStateCalled(header)
	}
	return nil
}

// RevertPeerStateToSnapshot -
func (vsp *ValidatorStatisticsProcessorMock) RevertPeerStateToSnapshot(snapshot int) error {
	if vsp.RevertPeerStateToSnapshotCalled != nil {
		return vsp.RevertPeerStateToSnapshotCalled(snapshot)
	}
	return nil
}

// Commit -
func (vsp *ValidatorStatisticsProcessorMock) Commit() ([]byte, error) {
	if vsp.CommitCalled != nil {
		return vsp.CommitCalled()
	}
	return nil, nil
}

// RootHash -
func (vsp *ValidatorStatisticsProcessorMock) RootHash() ([]byte, error) {
	if vsp.RootHashCalled != nil {
		return vsp.RootHashCalled()
	}
	return nil, nil
}

// GetPeerAccount -
func (vsp *ValidatorStatisticsProcessorMock) GetPeerAccount(address []byte) (state.PeerAccountHandler, error) {
	if vsp.GetPeerAccountCalled != nil {
		return vsp.GetPeerAccountCalled(address)
	}

	return nil, nil
}

// IsInterfaceNil -
func (vsp *ValidatorStatisticsProcessorMock) IsInterfaceNil() bool {
	if vsp.IsInterfaceNilCalled != nil {
		return vsp.IsInterfaceNilCalled()
	}
	return false
}

// IsPruningEnabled -
func (vsp *ValidatorStatisticsProcessorMock) IsPruningEnabled() bool {
	if vsp.IsPruningEnabledCalled != nil {
		return vsp.IsPruningEnabledCalled()
	}

	return false
}

// SnapshotState -
func (vsp *ValidatorStatisticsProcessorMock) SnapshotState(rootHash []byte) {
	if vsp.SnapshotStateCalled != nil {
		vsp.SnapshotStateCalled(rootHash)
	}
}

// PruneTrie -
func (vsp *ValidatorStatisticsProcessorMock) PruneTrie(rootHash []byte) error {
	if vsp.PruneTrieCalled != nil {
		return vsp.PruneTrieCalled(rootHash)
	}

	return nil
}

// CancelPrune -
func (vsp *ValidatorStatisticsProcessorMock) CancelPrune(rootHash []byte) {
	if vsp.CancelPruneCalled != nil {
		vsp.CancelPruneCalled(rootHash)
	}
}
