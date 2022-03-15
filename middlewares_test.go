package porter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJobIDMiddleware(t *testing.T) {
	fn := applyMiddleware(
		func(s State) error {
			jobID := JobIDFromState(s)
			assert.NotEmpty(t, jobID)
			assert.Equal(t, jobID, JobIDFromContext(s.Context()))

			return nil
		},
		JobIDMiddleware(),
	)

	assert.NoError(t, fn(&state{}))
}

func TestRecoverMiddleware(t *testing.T) {
	fn := applyMiddleware(
		func(s State) error {
			panic("test")
		},
		RecoverMiddleware(),
	)

	err := fn(&state{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "porter: panic test")
}
