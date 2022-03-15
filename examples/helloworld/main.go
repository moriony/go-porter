package main

import (
	"fmt"
	"time"

	"github.com/moriony/go-porter"
)

func main() {
	helloFunc := func(state porter.State) error {
		fmt.Println("Hello")
		return nil
	}

	w := porter.NewWorker(
		helloFunc,
		porter.WithJobsLimit(1),
		porter.WithSuccessTimeout(500*time.Millisecond),
		porter.WithShutdownPollTimeout(100*time.Millisecond),
		porter.WithMiddleware(
			porter.JobTTLMiddleware(2*time.Second),
		),
	)

	err := w.Run()
	if err != nil {
		fmt.Println("error", err)
	}

	time.Sleep(1 * time.Minute)
}
