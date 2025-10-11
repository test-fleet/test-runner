package worker

import (
	"context"
	"log"
	"sync"

	"github.com/test-fleet/test-runner/internal/runner"
)

type WorkerPool struct {
	logger      *log.Logger
	jobChan     <-chan string
	resultsChan chan<- bool
	maxWorkers  int
	runner      runner.TestRunner
	wg          sync.WaitGroup
}

func NewWorkerPool(
	logger *log.Logger,
	jobChan <-chan string,
	resultsChan chan<- bool,
	maxWorkers int,
	runner runner.TestRunner,
) *WorkerPool {
	return &WorkerPool{
		logger:      logger,
		jobChan:     jobChan,
		resultsChan: resultsChan,
		maxWorkers:  maxWorkers,
		runner:      runner,
		wg:          sync.WaitGroup{},
	}
}

func (w *WorkerPool) Start(ctx context.Context) {
	w.logger.Printf("Starting %d workers", w.maxWorkers)

	for i := 0; i < w.maxWorkers; i++ {
		w.wg.Add(1)
		go w.run(ctx, i)
	}
}

func (w *WorkerPool) run(ctx context.Context, workerId int) {
	defer w.wg.Done()

	w.logger.Printf("worker %d started work", workerId)

	for {
		select {
		case <-ctx.Done():
			w.logger.Printf("Worker %d interrupted", workerId)
			return
		case job, ok := <-w.jobChan:
			if !ok {
				w.logger.Printf("Worker %d stopping, job chan closed", workerId)
				return
			}
			w.processJob(ctx, workerId, job)
		}
	}
}

func (w *WorkerPool) processJob(ctx context.Context, workerId int, job string) {
	w.logger.Printf("Worker %d processing job", workerId)

	res := w.runner.Run(ctx, job)

	select {
	case w.resultsChan <- res:
		w.logger.Printf("worker %d completed job", workerId)
	case <-ctx.Done():
		w.logger.Printf("worker %d interrupted while sending results", workerId)
	}
}

func (w *WorkerPool) Wait() {
	w.wg.Wait()
	w.logger.Println("All workers stopped")
}
