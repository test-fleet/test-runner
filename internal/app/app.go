package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/test-fleet/test-runner/internal/config"
	"github.com/test-fleet/test-runner/internal/heartbeat"
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

	<-ctx.Done()
}
