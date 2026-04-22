package worker

import (
	"context"
	"log"
	"sync"
	"sync/atomic"

	"github.com/test-fleet/test-runner/internal/runner"
	"github.com/test-fleet/test-runner/pkg/models"
)

type WorkerPool struct {
	logger      *log.Logger
	jobChan     <-chan *models.Job
	resultsChan chan<- *models.SceneResult
	maxWorkers  int
	runner      runner.TestRunner
	wg          sync.WaitGroup
	activeJobs  atomic.Int32
}

func NewWorkerPool(
	logger *log.Logger,
	jobChan <-chan *models.Job,
	resultsChan chan<- *models.SceneResult,
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

func (w *WorkerPool) ActiveJobs() int {
	return int(w.activeJobs.Load())
}

func (w *WorkerPool) processJob(ctx context.Context, workerId int, job *models.Job) {
	w.logger.Printf("Worker %d processing job", workerId)
	w.activeJobs.Add(1)
	defer w.activeJobs.Add(-1)

	res := w.runner.Run(ctx, job)

	select {
	case w.resultsChan <- res:
		w.logger.Printf("worker %d completed job %s", workerId, job.JobID)
	case <-ctx.Done():
		w.logger.Printf("worker %d interrupted while sending results for job %s", workerId, job.JobID)
	}
}

func (w *WorkerPool) Wait() {
	w.wg.Wait()
	w.logger.Println("All workers stopped")
}
