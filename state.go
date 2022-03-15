package porter

import (
	"context"
)

type State interface {
	Context() context.Context
	WithContext(ctx context.Context) State
}

type state struct {
	ctx context.Context
}

func (s *state) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}

func (s *state) WithContext(ctx context.Context) State {
	if ctx == nil {
		return s
	}
	s2 := new(state)
	*s2 = *s
	s2.ctx = ctx
	return s2
}
