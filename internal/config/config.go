package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	RedisUrl          string
	ControlServerUrl  string
	ApiKey            string
	ApiSecret         string
	RunnerName        string
	HeartbeatInterval time.Duration
}

const (
	envRedisUrl          = "REDIS_URL"
	envServerUrl         = "CONTROL_SERVER_URL"
	envApiKey            = "API_KEY"
	envApiSecret         = "API_SECRET"
	envHeartbeatInterval = "HEARTBEAT_INTERVAL"
)

func Load() (*Config, error) {
	redisUrl := os.Getenv(envRedisUrl)
	if redisUrl == "" {
		return nil, fmt.Errorf("err: env variable (%s) not found", envRedisUrl)
	}

	serverUrl := os.Getenv(envServerUrl)
	if serverUrl == "" {
		return nil, fmt.Errorf("err: env variable (%s) not found", envServerUrl)
	}

	apiKey := os.Getenv(envApiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("err: env variable (%s) not found", envApiKey)
	}

	apiSecret := os.Getenv(envApiSecret)
	if apiSecret == "" {
		return nil, fmt.Errorf("err: env variable (%s) not found", envApiSecret)
	}

	heartbeatInterval := os.Getenv(envHeartbeatInterval)
	if heartbeatInterval == "" {
		heartbeatInterval = "15"
	}

	runnerName := os.Getenv("RUNNER_NAME")
	if runnerName == "" {
		runnerName = "unnamed-runner"
	}

	intervalSec, err := strconv.Atoi(heartbeatInterval)
	if err != nil {
		return nil, err
	}

	return &Config{
		RedisUrl:          redisUrl,
		ControlServerUrl:  serverUrl,
		ApiKey:            apiKey,
		ApiSecret:         apiSecret,
		HeartbeatInterval: time.Duration(intervalSec) * time.Second,
	}, nil
}
