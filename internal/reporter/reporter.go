package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/test-fleet/test-runner/internal/config"
	"github.com/test-fleet/test-runner/internal/utils"
	"github.com/test-fleet/test-runner/pkg/models"
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

func (c *Client) Send(result *models.SceneResult) {
	httpMethod := http.MethodPost
	httpPath := "/api/v1/results"

	jsonBody, err := json.Marshal(result)
	if err != nil {
		c.logger.Println("err: failed to marshal scene result", err)
		return
	}

	now := time.Now().UTC()
	isoTsString := now.Format(time.RFC3339)

	canonicalString, err := utils.BuildCanonicalString(httpMethod, httpPath, result, isoTsString)
	if err != nil {
		c.logger.Println("err: failed to build canonical string", err)
		return
	}
	signedCanonical := utils.SignCanonical(canonicalString, c.cfg.ApiSecret)

	httpUrl := fmt.Sprintf("%s%s", c.cfg.ControlServerUrl, httpPath)
	req, err := http.NewRequest(httpMethod, httpUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		c.logger.Println("err: failed to create request", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", c.cfg.ApiKey))
	req.Header.Set("x-request-timestamp", isoTsString)
	req.Header.Set("signature", fmt.Sprintf("sha256=%s", signedCanonical))

	res, err := c.http.Do(req)
	if err != nil {
		c.logger.Println("err: failed to send result", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		c.logger.Printf("err: result submission failed with status %d", res.StatusCode)
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			c.logger.Printf("err: failed to read res body %v", err)
		} else {
			c.logger.Println("res body:", string(bodyBytes))
		}
	}
}
