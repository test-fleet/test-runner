package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	httpPath := "/api/v1/runners/heartbeat"
	body := map[string]bool{"heartbeat": true}
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

	httpUrl := fmt.Sprintf("%s%s", c.cfg.ControlServerUrl, httpPath)
	req, err := http.NewRequest(httpMethod, httpUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		c.logger.Println("err: failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", c.cfg.ApiKey))
	req.Header.Set("x-request-timestamp", isoTsString)
	req.Header.Set("signature", fmt.Sprintf("sha256=%s", signedCanonical))

	res, err := c.http.Do(req)
	if err != nil {
		c.logger.Println("err: request failed", err)
		return
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		c.logger.Printf("err: heartbeat failed with status %d", res.StatusCode)
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			c.logger.Printf("err: failed to read res body %d", err)
		} else {
			bodyString := string(bodyBytes)
			c.logger.Println("res body:", bodyString)
		}
		return
	}
}
