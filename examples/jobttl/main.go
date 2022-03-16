package main

import (
	"fmt"
	"time"

	"github.com/moriony/go-porter"
)

func main() {
	w := porter.NewWorker(
		func(state porter.State) error {
			fmt.Println("started", porter.JobIDFromState(state))

			// emulate a long task execution
			<-time.After(200 * time.Millisecond)

			// intermediate check if the context has closed
			select {
			default:
			case <-state.Context().Done():
				return state.Context().Err()
			}

			// will never be called, because the function runs longer than the allowed ttl 100ms
			fmt.Println("finished", porter.JobIDFromState(state))

			return nil
		},
		porter.WithSuccessTimeout(1*time.Second),
		porter.WithMiddleware(
			porter.JobTTLMiddleware(100*time.Millisecond),
			porter.JobIDMiddleware(),
			LoggingMiddleware,
		),
	)

	err := w.Run()
	if err != nil {
		fmt.Println("error", err)
	}

	time.Sleep(1 * time.Second)
}

func LoggingMiddleware(next porter.JobFunc) porter.JobFunc {
	return func(state porter.State) error {
		err := next(state)

		// despite the completion of work on a timeout, the middleware will still be executed and log the result
		fmt.Println("job's result", err)

		return err
	}
}
