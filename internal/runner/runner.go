package runner

import (
	"context"
	"log"
	"time"
)

type Runner interface {
	Run(ctx context.Context, job string) bool // should return result type
}

type TestRunner struct {
	logger *log.Logger
}

func NewTestRunner(logger *log.Logger) *TestRunner {
	return &TestRunner{logger: logger}
}

func (e *TestRunner) Run(ctx context.Context, job string) bool {
	e.logger.Println("processing job")

	time.Sleep(10 * time.Second) // simulate work
	// test each frame here

	return true
}
