package porter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWorker(t *testing.T) {
	nopJobFunc := func(state State) error {
		return nil
	}

	t.Run("Defaults", func(t *testing.T) {
		wrk := NewWorker(nopJobFunc, nil)
		w := wrk.(*worker)

		assert.Equal(t, defaultShutdownPollTimeout, w.shutdownPollTimeout)
		assert.Equal(t, defaultJobsLimit, w.config.jobsLimit)
		assert.Equal(t, time.Duration(0), w.config.idleTimeout)
		assert.Equal(t, time.Duration(0), w.config.successTimeout)
		assert.Equal(t, time.Duration(0), w.config.errorTimeout)
	})

	t.Run("CustomConfig", func(t *testing.T) {
		jobsLimit := 1
		jobTTL := 2 * time.Second
		idleTimeout := 3 * time.Second
		successTimeout := 4 * time.Second
		errorTimeout := 5 * time.Second
		shutdownPollTimeout := 6 * time.Second

		wrk := NewWorker(
			nopJobFunc,
			WithJobsLimit(jobsLimit),
			WithIdleTimeout(idleTimeout),
			WithSuccessTimeout(successTimeout),
			WithErrorTimeout(errorTimeout),
			WithShutdownPollTimeout(shutdownPollTimeout),
			WithMiddleware(
				JobTTLMiddleware(jobTTL),
			),
		)
		w := wrk.(*worker)

		assert.Equal(t, shutdownPollTimeout, w.shutdownPollTimeout)
		assert.Equal(t, jobsLimit, w.config.jobsLimit)
		assert.Equal(t, idleTimeout, w.config.idleTimeout)
		assert.Equal(t, successTimeout, w.config.successTimeout)
		assert.Equal(t, errorTimeout, w.config.errorTimeout)
	})
}

func TestWorker_Run(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		w := NewWorker(
			func(state State) error {
				return nil
			},
			WithSuccessTimeout(1*time.Second),
		)

		assert.NoError(t, w.Run())
		assert.Equal(t, ErrAlreadyRunning, w.Run())
	})

	t.Run("Race", func(t *testing.T) {
		var count int64

		w := NewWorker(
			func(state State) error {
				return nil
			},
			WithSuccessTimeout(1*time.Second),
		)

		wg := sync.WaitGroup{}
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				if err := w.Run(); err == nil {
					atomic.AddInt64(&count, 1)
				}
				wg.Done()
			}()
		}
		wg.Wait()

		assert.Equal(t, int64(1), count)
	})
}

func TestWorker_Shutdown(t *testing.T) {
	t.Run("NotRunning", func(t *testing.T) {
		w := NewWorker(
			func(state State) error {
				return nil
			},
			WithSuccessTimeout(1*time.Second),
		)

		assert.Equal(t, ErrWorkerClosed, w.Shutdown(context.Background()))
	})

	t.Run("ContextDeadline", func(t *testing.T) {
		const shutdownPoolTimeout = 200 * time.Millisecond
		eventHandled := false

		w := NewWorker(
			func(state State) error {
				return nil
			},
			WithSuccessTimeout(1*time.Second),
			WithShutdownPollTimeout(shutdownPoolTimeout),
			WithSubscriber(func(s Subscriber) {
				s.ListenShutdown(func(err error) {
					eventHandled = true
					assert.Equal(t, context.DeadlineExceeded, err)
				})
			}),
		)

		assert.Nil(t, w.(*worker).done)
		assert.NoError(t, w.Run())
		assert.NotNil(t, w.(*worker).done)

		ctx, cancel := context.WithTimeout(context.Background(), shutdownPoolTimeout/2)
		defer cancel()

		w.(*worker).done = make(chan struct{})
		assert.Equal(t, context.DeadlineExceeded, w.Shutdown(ctx))
		assert.True(t, eventHandled)
	})

	t.Run("Success", func(t *testing.T) {
		const shutdownPoolTimeout = 500 * time.Millisecond
		eventHandled := false

		w := NewWorker(
			func(state State) error {
				return nil
			},
			WithSuccessTimeout(1*time.Second),
			WithShutdownPollTimeout(shutdownPoolTimeout),
			WithSubscriber(func(s Subscriber) {
				s.ListenShutdown(func(_ error) {
					eventHandled = true
				})
			}),
		)

		ctx, cancel := context.WithTimeout(context.Background(), shutdownPoolTimeout/2)
		defer cancel()

		assert.NoError(t, w.Run())
		assert.Nil(t, w.Shutdown(ctx))
		assert.True(t, eventHandled)
	})

	t.Run("Race", func(t *testing.T) {
		var count int64

		w := NewWorker(
			func(state State) error {
				return nil
			},
			WithSuccessTimeout(1*time.Second),
		)

		assert.NoError(t, w.Run())

		wg := sync.WaitGroup{}
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				if err := w.Shutdown(context.Background()); err == nil {
					atomic.AddInt64(&count, 1)
				}
				wg.Done()
			}()
		}
		wg.Wait()

		assert.Equal(t, int64(1), count)
	})
}

func TestWorker_PanicHandle(t *testing.T) {
	t.Run("CustomMiddleware", func(t *testing.T) {
		t.Run("WithJobsLimit_Before", func(t *testing.T) {
			wg := sync.WaitGroup{}

			w := NewWorker(
				func(state State) error {
					panic("test panic")
				},
				WithErrorTimeout(1*time.Second),
				WithSuccessTimeout(1*time.Second),
				WithJobsLimit(1),
				WithMiddleware(func(next JobFunc) JobFunc {
					return func(state State) error {
						defer wg.Done()
						defer func() {
							r := recover()
							assert.NotNil(t, r)
							assert.Contains(t, fmt.Sprintf("%v", r), "test panic")
						}()

						return next(state)
					}
				}),
			)

			wg.Add(1)
			assert.NoError(t, w.Run())
			wg.Wait()
		})

		t.Run("WithJobsLimit_After", func(t *testing.T) {
			wg := sync.WaitGroup{}

			w := NewWorker(
				func(state State) error {
					panic("test panic")
				},
				WithErrorTimeout(1*time.Second),
				WithSuccessTimeout(1*time.Second),
				WithMiddleware(func(next JobFunc) JobFunc {
					return func(state State) error {
						defer wg.Done()
						defer func() {
							r := recover()
							assert.NotNil(t, r)
							assert.Contains(t, fmt.Sprintf("%v", r), "test panic")
						}()

						return next(state)
					}
				}),
				WithJobsLimit(1),
			)

			wg.Add(1)
			assert.NoError(t, w.Run())
			wg.Wait()
		})

		t.Run("WithoutJobsLimit", func(t *testing.T) {
			wg := sync.WaitGroup{}

			w := NewWorker(
				func(state State) error {
					panic("test panic")
				},
				WithErrorTimeout(1*time.Second),
				WithSuccessTimeout(1*time.Second),
				WithMiddleware(func(next JobFunc) JobFunc {
					return func(state State) error {
						defer wg.Done()
						defer func() {
							r := recover()
							assert.NotNil(t, r)
							assert.Contains(t, fmt.Sprintf("%v", r), "test panic")
						}()

						return next(state)
					}
				}),
			)

			wg.Add(1)
			assert.NoError(t, w.Run())
			wg.Wait()
		})
	})
}

func Test_getTimeout(t *testing.T) {
	config := workerConfig{
		errorTimeout:   1,
		successTimeout: 2,
		idleTimeout:    3,
	}

	t.Run("IdleTimeout", func(t *testing.T) {
		timeout := getTimeout(config, ErrIdleJob)
		assert.Equal(t, config.idleTimeout, timeout)
	})

	t.Run("SuccessTimeout", func(t *testing.T) {
		timeout := getTimeout(config, nil)
		assert.Equal(t, config.successTimeout, timeout)
	})

	t.Run("ErrorTimeout", func(t *testing.T) {
		timeout := getTimeout(config, errors.New("test"))
		assert.Equal(t, config.errorTimeout, timeout)
	})
}
