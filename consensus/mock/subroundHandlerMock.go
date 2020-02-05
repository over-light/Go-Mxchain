package mock

import (
	"github.com/ElrondNetwork/elrond-go/consensus"
)

// SubroundHandlerMock -
type SubroundHandlerMock struct {
	DoWorkCalled    func(rounder consensus.Rounder) bool
	PreviousCalled  func() int
	NextCalled      func() int
	CurrentCalled   func() int
	StartTimeCalled func() int64
	EndTimeCalled   func() int64
	NameCalled      func() string
	JobCalled       func() bool
	CheckCalled     func() bool
}

// DoWork -
func (srm *SubroundHandlerMock) DoWork(rounder consensus.Rounder) bool {
	return srm.DoWorkCalled(rounder)
}

// Previous -
func (srm *SubroundHandlerMock) Previous() int {
	return srm.PreviousCalled()
}

// Next -
func (srm *SubroundHandlerMock) Next() int {
	return srm.NextCalled()
}

// Current -
func (srm *SubroundHandlerMock) Current() int {
	return srm.CurrentCalled()
}

// StartTime -
func (srm *SubroundHandlerMock) StartTime() int64 {
	return srm.StartTimeCalled()
}

// EndTime -
func (srm *SubroundHandlerMock) EndTime() int64 {
	return srm.EndTimeCalled()
}

// Name -
func (srm *SubroundHandlerMock) Name() string {
	return srm.NameCalled()
}

// IsInterfaceNil returns true if there is no value under the interface
func (srm *SubroundHandlerMock) IsInterfaceNil() bool {
	if srm == nil {
		return true
	}
	return false
}
