package porter

import (
	"context"
)

type workerGroup struct {
	workers []Worker
}

func NewWorkerGroup(workers ...Worker) Worker {
	return &workerGroup{
		workers: workers,
	}
}

func (g *workerGroup) Run() error {
	for _, w := range g.workers {
		if err := w.Run(); err != nil {
			return err
		}
	}

	return nil
}

func (g *workerGroup) Shutdown(ctx context.Context) error {
	// TODO need to stop the workers concurrently, otherwise, it can not meet the timeout
	for _, w := range g.workers {
		if err := w.Shutdown(ctx); err != nil {
			return err
		}
	}

	return nil
}
