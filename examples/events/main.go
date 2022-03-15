package main

import (
	"fmt"
	"time"

	"github.com/moriony/go-porter"
)

func main() {
	w := porter.NewWorker(
		func(state porter.State) error {
			return nil
		},
		porter.WithSuccessTimeout(500*time.Millisecond),
		porter.WithSubscriber(func(s porter.Subscriber) {
			s.ListenRun(func(err error) {
				if err != nil {
					fmt.Println("worker run error", err)
				} else {
					fmt.Println("worker is running")
				}
			})
		}),
		porter.WithMiddleware(
			porter.JobIDMiddleware(),
			LoggingMiddleware,
		),
	)

	err := w.Run()
	if err != nil {
		fmt.Println("error", err)
	}

	time.Sleep(1 * time.Minute)
}

func LoggingMiddleware(next porter.JobFunc) porter.JobFunc {
	return func(state porter.State) error {
		err := next(state)

		fmt.Println("job's done", porter.JobIDFromState(state))

		return err
	}
}
