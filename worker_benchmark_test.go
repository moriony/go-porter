package porter

import (
	"testing"
	"time"
)

func Benchmark(b *testing.B) {
	b.Run("ForLoop", func(b *testing.B) {
		data := make(chan int, 1)

		go func() {
			for {
				<-data
			}
		}()

		for i := 0; i < b.N; i++ {
			data <- i
		}
	})

	run := func(opts ...Opt) func(b *testing.B) {
		data := make(chan int, 1)

		w := NewWorker(
			func(_ State) error {
				<-data

				return nil
			},
			opts...,
		)

		return func(b *testing.B) {
			// nolint errcheck
			w.Run()

			for i := 0; i < b.N; i++ {
				data <- i
			}
		}
	}

	b.Run("Default", run())
	b.Run("WithJobTTL", run(WithMiddleware(JobTTLMiddleware(1*time.Second))))
	b.Run("WithJobID", run(WithMiddleware(JobIDMiddleware())))
}
