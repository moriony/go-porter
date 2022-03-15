package porter

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/google/uuid"
)

func applyMiddleware(h JobFunc, middleware ...MiddlewareFunc) JobFunc {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}

func emptyMiddleware(next JobFunc) JobFunc {
	return next
}

func WithMiddleware(middlewares ...MiddlewareFunc) Opt {
	return func(w *worker) {
		w.config.middlewares = append(w.config.middlewares, middlewares...)
	}
}

func JobTTLMiddleware(ttl time.Duration) MiddlewareFunc {
	if ttl <= 0 {
		return emptyMiddleware
	}

	return func(next JobFunc) JobFunc {
		return func(state State) error {
			result := make(chan error)
			panicChan := make(chan interface{}, 1)

			ctx, cancel := context.WithTimeout(state.Context(), ttl)
			s := state.WithContext(ctx)
			defer cancel()

			go func() {
				defer func() {
					if p := recover(); p != nil {
						panicChan <- p
					}
				}()
				result <- next(s)
			}()

			select {
			case p := <-panicChan:
				panic(p)
			case err := <-result:
				return err
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

type jobIDContextKey struct{}

var jobIDKey = jobIDContextKey{}

// JobIDMiddleware generates a unique identifier for every job
func JobIDMiddleware() MiddlewareFunc {
	return func(next JobFunc) JobFunc {
		return func(state State) error {
			id := uuid.New().String()

			return next(state.WithContext(context.WithValue(state.Context(), jobIDKey, id)))
		}
	}
}

// JobIDFromState returns job identifier from State
func JobIDFromState(state State) string {
	return JobIDFromContext(state.Context())
}

// JobIDFromContext returns job identifier from context
func JobIDFromContext(ctx context.Context) string {
	if jobID, ok := ctx.Value(jobIDKey).(string); ok {
		return jobID
	}
	return ""
}

// RecoverMiddleware recovers panic and transforms it into an error return
func RecoverMiddleware() MiddlewareFunc {
	return func(next JobFunc) JobFunc {
		return func(state State) (err error) {
			defer func() {
				if r := recover(); r != nil {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					err = fmt.Errorf("porter: panic %v\n%s", r, buf)
				}
			}()

			return next(state)
		}
	}
}
