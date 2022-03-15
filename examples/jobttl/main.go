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

			// эмулируем долгое выполнение задачи
			<-time.After(200 * time.Millisecond)

			// промежуточная проверка, не закрылся ли контекст
			select {
			default:
			case <-state.Context().Done():
				return state.Context().Err()
			}

			// никогда не вызовется, т.к. функция выполняется дольше разрешенного ttl 100ms
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

		// несмотря на завершение работы по таймауту, мидлвара все равно выполнится и залогирует результат
		fmt.Println("job's result", err)

		return err
	}
}
