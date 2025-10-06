package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/test-fleet/test-runner/internal/config"
	"github.com/test-fleet/test-runner/internal/utils"
)

type Client struct {
	cfg    *config.Config
	logger *log.Logger
	http   *http.Client
}

func NewClient(cfg *config.Config, logger *log.Logger, httpClient *http.Client) *Client {
	return &Client{
		cfg:    cfg,
		logger: logger,
		http:   httpClient,
	}
}

func (c *Client) Run(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Println("Shutting down heartbeat client")
			return
		case <-ticker.C:
			c.sendHeartbeat()
		}
	}
}

func (c *Client) sendHeartbeat() {
	httpMethod := http.MethodPost
	httpPath := fmt.Sprintf("%s/api/v1/runners/heartbeat", c.cfg.ControlServerUrl)
	body := map[string]string{"name": c.cfg.RunnerName}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		c.logger.Println("err: failed to marshall json body", err)
	}
	now := time.Now().UTC()

	isoTsString := now.Format(time.RFC3339)
	canonicalString, err := utils.BuildCanonicalString(
		httpMethod,
		httpPath,
		body,
		isoTsString,
	)
	if err != nil {
		c.logger.Println("err: failed to create canonical string", err)
	}
	signedCanonical := utils.SignCanonical(canonicalString, c.cfg.ApiSecret)

	req, err := http.NewRequest(httpMethod, httpPath, bytes.NewBuffer(jsonBody))
	if err != nil {
		c.logger.Println("err: failed to create request", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", c.cfg.ApiKey))
	req.Header.Set("x-request-timestamp", isoTsString)
	req.Header.Set("signature", signedCanonical)

	res, err := c.http.Do(req)
	if err != nil {
		c.logger.Println("err: request failed", err)
		return
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		c.logger.Printf("err: heartbeat failed with status %d", res.StatusCode)
		return
	}
}
