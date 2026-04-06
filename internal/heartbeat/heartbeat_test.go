package heartbeat

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/test-fleet/test-runner/internal/config"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		ApiKey:            "test-key",
		ApiSecret:         "test-secret",
		ControlServerUrl:  "http://localhost:8080",
		HeartbeatInterval: 30 * time.Second,
	}
	logger := log.New(io.Discard, "", 0)
	httpClient := &http.Client{}

	client := NewClient(cfg, logger, httpClient)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.cfg != cfg {
		t.Error("Client config not set correctly")
	}
	if client.logger != logger {
		t.Error("Client logger not set correctly")
	}
	if client.http != httpClient {
		t.Error("Client http client not set correctly")
	}
}

func TestSendHeartbeat_Success(t *testing.T) {
	// Create a test server that expects the heartbeat request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/runners/heartbeat" {
			t.Errorf("Expected path /api/v1/runners/heartbeat, got %s", r.URL.Path)
		}

		// Verify headers are present
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Missing or incorrect Content-Type header")
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("Missing Authorization header")
		}
		if r.Header.Get("x-request-timestamp") == "" {
			t.Error("Missing x-request-timestamp header")
		}
		if r.Header.Get("signature") == "" {
			t.Error("Missing signature header")
		}

		// Verify body
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "heartbeat") {
			t.Error("Request body doesn't contain heartbeat field")
		}

		// Return success
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		ApiKey:            "test-key",
		ApiSecret:         "test-secret",
		ControlServerUrl:  server.URL,
		HeartbeatInterval: 30 * time.Second,
	}
	logger := log.New(io.Discard, "", 0)
	httpClient := server.Client()

	client := NewClient(cfg, logger, httpClient)
	client.sendHeartbeat()
	// If we get here without panic, test passes
}

func TestSendHeartbeat_ServerError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		ApiKey:            "test-key",
		ApiSecret:         "test-secret",
		ControlServerUrl:  server.URL,
		HeartbeatInterval: 30 * time.Second,
	}
	logger := log.New(io.Discard, "", 0)
	httpClient := server.Client()

	client := NewClient(cfg, logger, httpClient)
	// Should not panic, just log the error
	client.sendHeartbeat()
}

func TestRun_ContextCancellation(t *testing.T) {
	// Create a test server
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		ApiKey:            "test-key",
		ApiSecret:         "test-secret",
		ControlServerUrl:  server.URL,
		HeartbeatInterval: 50 * time.Millisecond, // Short interval for testing
	}
	logger := log.New(io.Discard, "", 0)
	httpClient := server.Client()

	client := NewClient(cfg, logger, httpClient)

	ctx, cancel := context.WithCancel(context.Background())

	// Run client in goroutine
	done := make(chan bool)
	go func() {
		client.Run(ctx)
		done <- true
	}()

	// Let it send a couple heartbeats
	time.Sleep(150 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for Run to exit
	select {
	case <-done:
		// Success - Run exited when context was cancelled
	case <-time.After(1 * time.Second):
		t.Fatal("Run() did not exit after context cancellation")
	}

	if requestCount == 0 {
		t.Error("Expected at least one heartbeat to be sent")
	}
}

func TestRun_SendsPeriodicHeartbeats(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		ApiKey:            "test-key",
		ApiSecret:         "test-secret",
		ControlServerUrl:  server.URL,
		HeartbeatInterval: 50 * time.Millisecond,
	}
	logger := log.New(io.Discard, "", 0)
	httpClient := server.Client()

	client := NewClient(cfg, logger, httpClient)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	client.Run(ctx)

	// Should have sent approximately 3-4 heartbeats in 200ms with 50ms interval
	if requestCount < 2 {
		t.Errorf("Expected at least 2 heartbeats, got %d", requestCount)
	}
	if requestCount > 6 {
		t.Errorf("Expected no more than 6 heartbeats, got %d", requestCount)
	}
}
