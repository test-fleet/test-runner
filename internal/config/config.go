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
	Channel           string
	MaxWorkers        int
	HeartbeatInterval time.Duration
}

const (
	envRedisUrl          = "REDIS_URL"
	envServerUrl         = "CONTROL_SERVER_URL"
	envApiKey            = "API_KEY"
	envApiSecret         = "API_SECRET"
	envHeartbeatInterval = "HEARTBEAT_INTERVAL"
	envRedisChannel      = "REDIS_CHANNEL"
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

	redisChannel := os.Getenv(envRedisChannel)
	if redisChannel == "" {
		return nil, fmt.Errorf("err: env variable (%s) not found", envRedisChannel)
	}

	heartbeatInterval := os.Getenv(envHeartbeatInterval)
	if heartbeatInterval == "" {
		heartbeatInterval = "3"
	}

	runnerName := os.Getenv("RUNNER_NAME")
	if runnerName == "" {
		runnerName = "unnamed-runner"
	}

	intervalSec, err := strconv.Atoi(heartbeatInterval)
	if err != nil {
		return nil, err
	}

	maxWorkers := os.Getenv("MAX_WORKERS")
	if maxWorkers == "" {
		maxWorkers = "10"
	}

	maxWorkersNum, err := strconv.Atoi(maxWorkers)
	if err != nil {
		return nil, err
	}

	return &Config{
		RedisUrl:          redisUrl,
		ControlServerUrl:  serverUrl,
		ApiKey:            apiKey,
		ApiSecret:         apiSecret,
		Channel:           redisChannel,
		MaxWorkers:        maxWorkersNum,
		HeartbeatInterval: time.Duration(intervalSec) * time.Second,
	}, nil
}
