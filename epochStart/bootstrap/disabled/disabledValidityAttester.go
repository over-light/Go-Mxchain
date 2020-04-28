package disabled

import (
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/process"
)

var _ process.ValidityAttester = (*validityAttester)(nil)

type validityAttester struct {
}

// NewValidityAttester returns a new instance of validityAttester
func NewValidityAttester() *validityAttester {
	return &validityAttester{}
}

// CheckBlockAgainstFinal -
func (v *validityAttester) CheckBlockAgainstFinal(_ data.HeaderHandler) error {
	return nil
}

// CheckBlockAgainstRounder -
func (v *validityAttester) CheckBlockAgainstRounder(_ data.HeaderHandler) error {
	return nil
}

// IsInterfaceNil -
func (v *validityAttester) IsInterfaceNil() bool {
	return v == nil
}
