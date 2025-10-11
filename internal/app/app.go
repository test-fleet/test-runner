package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/test-fleet/test-runner/internal/config"
	"github.com/test-fleet/test-runner/internal/heartbeat"
	"github.com/test-fleet/test-runner/internal/runner"
	"github.com/test-fleet/test-runner/internal/subscriber"
	"github.com/test-fleet/test-runner/internal/worker"
)

func Run() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	hbLogger := log.New(os.Stderr, "Heartbeat Client: ", log.LstdFlags)
	httpClient := &http.Client{
		Timeout: 15 * time.Second,
	}
	heartbeatClient := heartbeat.NewClient(cfg, hbLogger, httpClient)
	go heartbeatClient.Run(ctx)

	opts, err := redis.ParseURL(cfg.RedisUrl)
	if err != nil {
		log.Fatalf("err: failed to parse redis url %v", err)
	}
	client := redis.NewClient(opts)
	rctx := context.Background()
	if err := client.Ping(rctx).Err(); err != nil {
		log.Fatalf("err: failed to ping redis server %v", err)
	}

	jobChan := make(chan string)
	resChan := make(chan bool)

	subLogger := log.New(os.Stderr, "Redis client: ", log.LstdFlags)
	sub := subscriber.NewSubscriber(cfg, client, jobChan, subLogger)

	runLogger := log.New(os.Stderr, "Test Runner: ", log.LstdFlags)
	runner := runner.NewTestRunner(runLogger)

	workerLogger := log.New(os.Stderr, "Workers: ", log.LstdFlags)
	workers := worker.NewWorkerPool(
		workerLogger,
		jobChan,
		resChan,
		cfg.MaxWorkers,
		*runner,
	)
	workers.Start(ctx)

	go func() {
		if err := sub.Subscribe(ctx); err != nil && err != context.Canceled {
			log.Fatalf("err: subscriber error %v", err)
		}
		close(jobChan)
		log.Println("shutting down subscriber")
	}()

	go func() {
		workers.Wait()
		close(resChan)
		log.Println("shutting down workers")
	}()

	resultsLogger := log.New(os.Stderr, "Reporter: ", log.LstdFlags)
	go func() {
		for result := range resChan {
			resultsLogger.Printf("Received test result: %v", result)
			// TODO: Send result to API via HTTP
		}
		resultsLogger.Println("Results channel closed, result processor exiting")
	}()

	<-ctx.Done()
	log.Println("Shutdown signal received, waiting for cleanup...")
}
