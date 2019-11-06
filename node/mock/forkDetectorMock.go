package mock

import (
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/process"
)

// ForkDetectorMock is a mock implementation for the ForkDetector interface
type ForkDetectorMock struct {
	AddHeaderCalled                 func(header data.HeaderHandler, hash []byte, state process.BlockHeaderState, finalHeaders []data.HeaderHandler, finalHeadersHashes [][]byte, isNotarizedShardStuck bool) error
	RemoveHeadersCalled             func(nonce uint64, hash []byte)
	CheckForkCalled                 func() *process.ForkInfo
	GetHighestFinalBlockNonceCalled func() uint64
	ProbableHighestNonceCalled      func() uint64
	ResetProbableHighestNonceCalled func()
	ResetForkCalled                 func()
}

// AddHeader is a mock implementation for AddHeader
func (f *ForkDetectorMock) AddHeader(header data.HeaderHandler, hash []byte, state process.BlockHeaderState, finalHeaders []data.HeaderHandler, finalHeadersHashes [][]byte, isNotarizedShardStuck bool) error {
	return f.AddHeaderCalled(header, hash, state, finalHeaders, finalHeadersHashes, isNotarizedShardStuck)
}

// RemoveHeaders is a mock implementation for RemoveHeaders
func (f *ForkDetectorMock) RemoveHeaders(nonce uint64, hash []byte) {
	f.RemoveHeadersCalled(nonce, hash)
}

// CheckFork is a mock implementation for CheckFork
func (f *ForkDetectorMock) CheckFork() *process.ForkInfo {
	return f.CheckForkCalled()
}

// GetHighestFinalBlockNonce is a mock implementation for GetHighestFinalBlockNonce
func (f *ForkDetectorMock) GetHighestFinalBlockNonce() uint64 {
	return f.GetHighestFinalBlockNonceCalled()
}

// ProbableHighestNonce is a mock implementation for GetProbableHighestNonce
func (f *ForkDetectorMock) ProbableHighestNonce() uint64 {
	return f.ProbableHighestNonceCalled()
}

func (fdm *ForkDetectorMock) ResetProbableHighestNonce() {
	fdm.ResetProbableHighestNonceCalled()
}

func (fdm *ForkDetectorMock) ResetFork() {
	fdm.ResetForkCalled()
}

// IsInterfaceNil returns true if there is no value under the interface
func (fdm *ForkDetectorMock) IsInterfaceNil() bool {
	if fdm == nil {
		return true
	}
	return false
}
