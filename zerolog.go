package porter

import (
	"github.com/rs/zerolog"
)

func WithZerolog(logger *zerolog.Logger) Opt {
	return func(w *worker) {
		ZerologSubscriber(logger)(w.events)
		WithMiddleware(ZerologMiddleware(logger))(w)
	}
}

func ZerologSubscriber(logger *zerolog.Logger) func(Subscriber) {
	return func(s Subscriber) {
		s.ListenRun(func(err error) {
			if err != nil {
				logger.Error().Err(err).Msg("worker run failed")
			} else {
				logger.Info().Msg("worker running")
			}
		})

		s.ListenShutdown(func(err error) {
			if err != nil {
				logger.Error().Err(err).Msg("worker stopped with error")
			} else {
				logger.Info().Msg("worker stopped")
			}
		})
	}
}

// ZerologMiddleware logs the execution of tasks
func ZerologMiddleware(logger *zerolog.Logger) MiddlewareFunc {
	return func(next JobFunc) JobFunc {
		return func(state State) error {
			err := next(state)

			switch err {
			case ErrWorkerClosed, ErrIdleJob:
				return err
			}

			if err != nil {
				logger.Error().Err(err).Str("job_id", JobIDFromState(state)).Msg("job error")
			}

			return err
		}
	}
}
