package worker

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/test-fleet/test-runner/internal/runner"
	"github.com/test-fleet/test-runner/pkg/models"
)

func testRunner(t *testing.T) runner.TestRunner {
	t.Helper()
	return *runner.NewTestRunner(log.New(io.Discard, "", 0), "")
}

func TestNewWorkerPool(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	jobChan := make(chan *models.Job)
	resultsChan := make(chan *models.SceneResult)

	pool := NewWorkerPool(logger, jobChan, resultsChan, 5, testRunner(t))

	assert.NotNil(t, pool)
	assert.Equal(t, 5, pool.maxWorkers)
}

func TestActiveJobs_InitiallyZero(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	jobChan := make(chan *models.Job)
	resultsChan := make(chan *models.SceneResult)

	pool := NewWorkerPool(logger, jobChan, resultsChan, 3, testRunner(t))

	assert.Equal(t, 0, pool.ActiveJobs())
}

func TestWorkerPool_ContextCancellation(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	jobChan := make(chan *models.Job)
	resultsChan := make(chan *models.SceneResult, 10)

	pool := NewWorkerPool(logger, jobChan, resultsChan, 3, testRunner(t))

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)
	cancel()

	done := make(chan struct{})
	go func() {
		pool.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Workers stopped
	case <-time.After(2 * time.Second):
		t.Fatal("Workers did not stop after context cancellation")
	}
}

func TestWorkerPool_JobChannelClose(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	jobChan := make(chan *models.Job)
	resultsChan := make(chan *models.SceneResult, 10)

	pool := NewWorkerPool(logger, jobChan, resultsChan, 2, testRunner(t))
	pool.Start(context.Background())
	close(jobChan)

	done := make(chan struct{})
	go func() {
		pool.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Workers stopped
	case <-time.After(2 * time.Second):
		t.Fatal("Workers did not stop after job channel close")
	}
}

func TestWorkerPool_ProcessJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	logger := log.New(io.Discard, "", 0)
	jobChan := make(chan *models.Job, 1)
	resultsChan := make(chan *models.SceneResult, 1)

	pool := NewWorkerPool(logger, jobChan, resultsChan, 1, testRunner(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool.Start(ctx)

	job := &models.Job{
		JobID: "test-job",
		Scene: models.Scene{
			Timeout:   5000,
			Variables: map[string]models.Variable{},
		},
		Frames: []models.Frame{
			{
				Order: 1,
				Request: models.HTTPRequest{
					Method:  "GET",
					URL:     server.URL + "/test",
					Timeout: 5000,
					Headers: map[string]string{},
				},
				Extractors: []models.Extractors{},
			},
		},
	}

	jobChan <- job

	select {
	case result := <-resultsChan:
		assert.Equal(t, "passed", result.Status)
	case <-time.After(3 * time.Second):
		t.Fatal("Did not receive job result in time")
	}
}

func TestWorkerPool_MultipleJobs(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	logger := log.New(io.Discard, "", 0)
	jobChan := make(chan *models.Job, 3)
	resultsChan := make(chan *models.SceneResult, 3)

	pool := NewWorkerPool(logger, jobChan, resultsChan, 2, testRunner(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool.Start(ctx)

	for i := 0; i < 3; i++ {
		job := &models.Job{
			JobID: "test-job",
			Scene: models.Scene{
				Timeout:   5000,
				Variables: map[string]models.Variable{},
			},
			Frames: []models.Frame{
				{
					Order: 1,
					Request: models.HTTPRequest{
						Method:  "GET",
						URL:     server.URL + "/test",
						Timeout: 5000,
						Headers: map[string]string{},
					},
					Extractors: []models.Extractors{},
				},
			},
		}
		jobChan <- job
	}

	results := 0
	timeout := time.After(3 * time.Second)
	for results < 3 {
		select {
		case <-resultsChan:
			results++
		case <-timeout:
			t.Fatalf("Only received %d results out of 3", results)
		}
	}

	assert.Equal(t, 3, requestCount)
}

func TestWorkerPool_Wait(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	jobChan := make(chan *models.Job)
	resultsChan := make(chan *models.SceneResult, 10)

	pool := NewWorkerPool(logger, jobChan, resultsChan, 2, testRunner(t))

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	cancel()

	waitDone := make(chan struct{})
	go func() {
		pool.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		// Wait returned after all workers finished
	case <-time.After(2 * time.Second):
		t.Fatal("Wait() did not return after context cancellation")
	}
}
