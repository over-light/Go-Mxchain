package floodPreventers

import (
	"fmt"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/stretchr/testify/assert"
)

func TestNewNilTopicFloodPreventer(t *testing.T) {
	t.Parallel()

	ntfp := NewNilTopicFloodPreventer()

	assert.False(t, check.IfNil(ntfp))
}

func TestNilTopicFloodPreventer_FunctionsShouldNotPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r != nil {
			assert.Fail(t, fmt.Sprintf("should not have paniced %v", r))
		}
	}()

	ntfp := NewNilTopicFloodPreventer()

	ntfp.ResetForTopic("")
	ntfp.SetMaxMessagesForTopic("", 0)
	assert.True(t, ntfp.Accumulate("", ""))
}
