package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/test-fleet/test-runner/internal/config"
	"github.com/test-fleet/test-runner/internal/heartbeat"
	"github.com/test-fleet/test-runner/internal/reporter"
	"github.com/test-fleet/test-runner/internal/runner"
	"github.com/test-fleet/test-runner/internal/subscriber"
	"github.com/test-fleet/test-runner/internal/worker"
	"github.com/test-fleet/test-runner/pkg/models"
)

func Run() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	instanceBytes := make([]byte, 8)
	if _, err := rand.Read(instanceBytes); err != nil {
		log.Fatalf("err: failed to generate instance id: %v", err)
	}
	instanceID := hex.EncodeToString(instanceBytes)
	log.Printf("Instance ID: %s", instanceID)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	opts, err := redis.ParseURL(cfg.RedisUrl)
	if err != nil {
		log.Fatalf("err: failed to parse redis url %v", err)
	}
	client := redis.NewClient(opts)
	rctx := context.Background()
	if err := client.Ping(rctx).Err(); err != nil {
		log.Fatalf("err: failed to ping redis server %v", err)
	}

	jobChan := make(chan *models.Job)
	resChan := make(chan *models.SceneResult)

	subLogger := log.New(os.Stderr, "Redis client: ", log.LstdFlags)
	sub := subscriber.NewSubscriber(cfg, client, jobChan, subLogger)

	runLogger := log.New(os.Stderr, "Test Runner: ", log.LstdFlags)
	testRunner := runner.NewTestRunner(runLogger, cfg.RunnerName)

	workerLogger := log.New(os.Stderr, "Workers: ", log.LstdFlags)
	workers := worker.NewWorkerPool(
		workerLogger,
		jobChan,
		resChan,
		cfg.MaxWorkers,
		*testRunner,
	)
	workers.Start(ctx)

	httpClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	hbLogger := log.New(os.Stderr, "Heartbeat Client: ", log.LstdFlags)
	heartbeatClient := heartbeat.NewClient(cfg, hbLogger, httpClient, workers.ActiveJobs, instanceID, cfg.RunnerName)
	go heartbeatClient.Run(ctx)

	go func() {
		if err := sub.Subscribe(ctx); err != nil && err != context.Canceled {
			log.Fatalf("err: subscriber error %v", err)
		}
		close(jobChan)
		log.Println("shutting down subscriber")
	}()

	// Watchdog: the pub/sub connection above can silently die and retry
	// forever without ever surfacing an error (go-redis reconnects
	// internally), while the heartbeat client below keeps succeeding over
	// a separate HTTP connection — meaning the runner would look healthy
	// while never receiving new jobs. Ping Redis directly on a timer and
	// exit (letting Kubernetes restart the pod) after sustained failures.
	watchdogLogger := log.New(os.Stderr, "Redis Watchdog: ", log.LstdFlags)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		failures := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				err := client.Ping(pingCtx).Err()
				cancel()
				if err != nil {
					failures++
					watchdogLogger.Printf("err: redis ping failed (%d/3): %v", failures, err)
					if failures >= 3 {
						log.Fatalf("err: redis unreachable after %d consecutive checks, exiting", failures)
					}
					continue
				}
				failures = 0
			}
		}
	}()

	go func() {
		workers.Wait()
		close(resChan)
		log.Println("shutting down workers")
	}()

	reporterLogger := log.New(os.Stderr, "Reporter: ", log.LstdFlags)
	reporterClient := reporter.NewClient(cfg, reporterLogger, httpClient)
	go func() {
		for result := range resChan {
			reporterClient.Send(result)
		}
		reporterLogger.Println("Results channel closed, reporter exiting")
	}()

	<-ctx.Done()
	log.Println("Shutdown signal received, waiting for cleanup...")
}
