package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setRequiredEnvVars(t *testing.T) {
	t.Helper()
	os.Setenv("REDIS_URL", "redis://localhost:6379")
	os.Setenv("CONTROL_SERVER_URL", "http://localhost:8080")
	os.Setenv("API_KEY", "test-api-key")
	os.Setenv("API_SECRET", "test-api-secret")
	os.Setenv("REDIS_CHANNEL", "test-channel")
	t.Cleanup(func() {
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("CONTROL_SERVER_URL")
		os.Unsetenv("API_KEY")
		os.Unsetenv("API_SECRET")
		os.Unsetenv("REDIS_CHANNEL")
		os.Unsetenv("HEARTBEAT_INTERVAL")
		os.Unsetenv("RUNNER_NAME")
		os.Unsetenv("MAX_WORKERS")
	})
}

func TestLoad_AllRequiredEnvVars(t *testing.T) {
	setRequiredEnvVars(t)

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "redis://localhost:6379", cfg.RedisUrl)
	assert.Equal(t, "http://localhost:8080", cfg.ControlServerUrl)
	assert.Equal(t, "test-api-key", cfg.ApiKey)
	assert.Equal(t, "test-api-secret", cfg.ApiSecret)
	assert.Equal(t, "test-channel", cfg.Channel)
	assert.Equal(t, 3*time.Second, cfg.HeartbeatInterval) // default
	assert.Equal(t, 10, cfg.MaxWorkers)                   // default
}

func TestLoad_CustomHeartbeatInterval(t *testing.T) {
	setRequiredEnvVars(t)
	os.Setenv("HEARTBEAT_INTERVAL", "30")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, cfg.HeartbeatInterval)
}

func TestLoad_CustomMaxWorkers(t *testing.T) {
	setRequiredEnvVars(t)
	os.Setenv("MAX_WORKERS", "20")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, 20, cfg.MaxWorkers)
}

func TestLoad_MissingRedisUrl(t *testing.T) {
	setRequiredEnvVars(t)
	os.Unsetenv("REDIS_URL")

	_, err := Load()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "REDIS_URL")
}

func TestLoad_MissingControlServerUrl(t *testing.T) {
	setRequiredEnvVars(t)
	os.Unsetenv("CONTROL_SERVER_URL")

	_, err := Load()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CONTROL_SERVER_URL")
}

func TestLoad_MissingApiKey(t *testing.T) {
	setRequiredEnvVars(t)
	os.Unsetenv("API_KEY")

	_, err := Load()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API_KEY")
}

func TestLoad_MissingApiSecret(t *testing.T) {
	setRequiredEnvVars(t)
	os.Unsetenv("API_SECRET")

	_, err := Load()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API_SECRET")
}

func TestLoad_MissingRedisChannel(t *testing.T) {
	setRequiredEnvVars(t)
	os.Unsetenv("REDIS_CHANNEL")

	_, err := Load()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "REDIS_CHANNEL")
}

func TestLoad_InvalidHeartbeatInterval(t *testing.T) {
	setRequiredEnvVars(t)
	os.Setenv("HEARTBEAT_INTERVAL", "not-a-number")

	_, err := Load()

	assert.Error(t, err)
}

func TestLoad_InvalidMaxWorkers(t *testing.T) {
	setRequiredEnvVars(t)
	os.Setenv("MAX_WORKERS", "not-a-number")

	_, err := Load()

	assert.Error(t, err)
}
