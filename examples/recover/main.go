package main

import (
	"fmt"
	"time"

	"github.com/moriony/go-porter"
)

func main() {
	w := porter.NewWorker(
		func(state porter.State) error {
			panic("something went wrong")
		},
		porter.WithErrorTimeout(5*time.Second),
		porter.WithMiddleware(
			LoggingMiddleware,
			porter.RecoverMiddleware(),
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
		if err != nil {
			fmt.Println("jobs's done with error")
		}

		return err
	}
}
