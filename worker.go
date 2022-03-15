package porter

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrAlreadyRunning = errors.New("worker already running")
	ErrWorkerClosed   = errors.New("worker is closed")
	ErrIdleJob        = errors.New("idle job")
)

const (
	defaultJobsLimit           = 1
	defaultShutdownPollTimeout = 500 * time.Millisecond
)

type Worker interface {
	Run() error
	Shutdown(ctx context.Context) error
}

type JobFunc func(state State) error

type MiddlewareFunc func(JobFunc) JobFunc

type Opt func(w *worker)

func WithJobsLimit(limit int) Opt {
	return func(w *worker) {
		if limit > 0 {
			w.config.jobsLimit = limit
		}
	}
}

func WithShutdownPollTimeout(shutdownPollTimeout time.Duration) Opt {
	return func(w *worker) {
		if shutdownPollTimeout > 0 {
			w.shutdownPollTimeout = shutdownPollTimeout
		}
	}
}

// WithRunDelay adds a delay before worker starts
func WithRunDelay(delay time.Duration) Opt {
	return func(w *worker) {
		if delay > 0 {
			w.config.delay = delay
		}
	}
}

func WithSubscriber(subscribers ...func(subscriber Subscriber)) Opt {
	return func(w *worker) {
		for _, subscribe := range subscribers {
			subscribe(w.events)
		}
	}
}

// WithErrorTimeout adds a delay after the job that returned the error
func WithErrorTimeout(timeout time.Duration) Opt {
	return func(w *worker) {
		w.config.errorTimeout = timeout
	}
}

// WithIdleTimeout adds a delay after the job that returned ErrIdleJob
func WithIdleTimeout(timeout time.Duration) Opt {
	return func(w *worker) {
		w.config.idleTimeout = timeout
	}
}

// WithSuccessTimeout adds a delay after a successful job
func WithSuccessTimeout(timeout time.Duration) Opt {
	return func(w *worker) {
		w.config.successTimeout = timeout
	}
}

func NewWorker(jobFunc JobFunc, opts ...Opt) Worker {
	w := &worker{
		jobFunc:             jobFunc,
		events:              &Dispatcher{},
		shutdownPollTimeout: defaultShutdownPollTimeout,
		config: workerConfig{
			jobsLimit: defaultJobsLimit,
		},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(w)
		}
	}

	return w
}

type worker struct {
	// Blocks concurrent start and stop of the worker
	mu sync.Mutex
	// While the channel is open, it means that the worker is working
	done <-chan struct{}
	// While the channel is open, the worker will start new tasks
	closed chan struct{}
	// Events handler
	events *Dispatcher
	// Timeout between attempts to stop the worker
	shutdownPollTimeout time.Duration
	// The task that the worker performs
	jobFunc JobFunc

	config workerConfig
}

type workerConfig struct {
	jobsLimit      int
	delay          time.Duration
	errorTimeout   time.Duration
	successTimeout time.Duration
	idleTimeout    time.Duration
	middlewares    []MiddlewareFunc
}

func (w *worker) Run() error {
	err := w.run()
	w.events.OnRun(err)

	return err
}

func (w *worker) run() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.done != nil {
		select {
		case <-w.done:
		default:
			return ErrAlreadyRunning
		}
	}

	w.closed = make(chan struct{})
	w.done = runWorker(w.jobFunc, w.config, w.closed)

	return nil
}

func (w *worker) Shutdown(ctx context.Context) error {
	err := w.shutdown(ctx)
	w.events.OnShutdown(err)

	return err
}

func (w *worker) shutdown(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed == nil {
		return ErrWorkerClosed
	}

	select {
	default:
	case <-w.closed:
		return ErrWorkerClosed
	}

	close(w.closed)

	select {
	default:
	case <-w.done:
		return ErrWorkerClosed
	}

	ticker := time.NewTicker(w.shutdownPollTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func runWorker(fn JobFunc, config workerConfig, closed <-chan struct{}) <-chan struct{} {
	done := make(chan struct{})
	jobs := make(chan struct{}, config.jobsLimit)

	go func() {
		defer func() {
			done <- struct{}{}
		}()

		if config.delay > 0 {
			select {
			case <-closed:
				return
			case <-time.After(config.delay):
			}
		}

		for {
			select {
			default:
			case <-closed:
				return
			}

			jobs <- struct{}{}

			// TODO use a worker pool to avoid running excess goroutines
			go func() {
				var err error

				defer func() {
					if timeout := getTimeout(config, err); timeout > 0 {
						select {
						case <-time.After(timeout):
						case <-closed:
						}
					}

					<-jobs
				}()

				s := &state{}
				err = applyMiddleware(fn, config.middlewares...)(s)
			}()
		}
	}()

	return done
}

func getTimeout(c workerConfig, err error) time.Duration {
	timeout := time.Duration(0)
	switch err {
	case ErrIdleJob:
		timeout = c.idleTimeout
	case nil:
		timeout = c.successTimeout
	default:
		timeout = c.errorTimeout
	}

	return timeout
}
