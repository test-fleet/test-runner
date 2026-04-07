package subscriber

import (
	"io"
	"log"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/test-fleet/test-runner/internal/config"
	"github.com/test-fleet/test-runner/pkg/models"
)

func TestNewSubscriber(t *testing.T) {
	cfg := &config.Config{Channel: "test-channel"}
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	jobChan := make(chan *models.Job)
	logger := log.New(io.Discard, "", 0)

	sub := NewSubscriber(cfg, client, jobChan, logger)

	assert.NotNil(t, sub)
	assert.Equal(t, cfg, sub.cfg)
	assert.Equal(t, client, sub.client)
	assert.Equal(t, jobChan, sub.jobChan)
	assert.Equal(t, logger, sub.logger)
}

func TestParseJob_ValidPayload(t *testing.T) {
	sub := &Subscriber{}

	payload := `{
		"jobId": "job-123",
		"type": "http",
		"runId": "run-456",
		"scene": {
			"id": "scene-1",
			"name": "Test Scene",
			"timeout": 5000,
			"variables": {}
		},
		"frames": []
	}`

	job, err := sub.parseJob(payload)

	require.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, "job-123", job.JobID)
	assert.Equal(t, "http", job.Type)
	assert.Equal(t, "run-456", job.RunID)
	assert.Equal(t, "scene-1", job.Scene.ID)
	assert.Equal(t, "Test Scene", job.Scene.Name)
	assert.Equal(t, 5000, job.Scene.Timeout)
}

func TestParseJob_WithFrames(t *testing.T) {
	sub := &Subscriber{}

	payload := `{
		"jobId": "job-with-frames",
		"type": "http",
		"runId": "run-789",
		"scene": {
			"id": "scene-2",
			"name": "Scene With Frames",
			"timeout": 10000,
			"variables": {
				"baseUrl": {"value": "https://api.example.com", "type": "string"}
			}
		},
		"frames": [
			{
				"_id": "frame-1",
				"name": "Login",
				"order": 1,
				"enabled": true,
				"request": {
					"method": "POST",
					"url": "https://api.example.com/login",
					"headers": {"Content-Type": "application/json"},
					"body": "{\"user\": \"test\"}",
					"timeout": 5000
				},
				"extractors": [],
				"assertions": []
			}
		]
	}`

	job, err := sub.parseJob(payload)

	require.NoError(t, err)
	assert.Equal(t, "job-with-frames", job.JobID)
	assert.Len(t, job.Frames, 1)
	assert.Equal(t, "Login", job.Frames[0].Name)
	assert.Equal(t, 1, job.Frames[0].Order)
	assert.Equal(t, "POST", job.Frames[0].Request.Method)
}

func TestParseJob_WithVariables(t *testing.T) {
	sub := &Subscriber{}

	payload := `{
		"jobId": "job-vars",
		"scene": {
			"timeout": 5000,
			"variables": {
				"token": {"value": "abc123", "type": "string"},
				"userId": {"value": 42, "type": "number"},
				"active": {"value": true, "type": "boolean"}
			}
		},
		"frames": []
	}`

	job, err := sub.parseJob(payload)

	require.NoError(t, err)
	assert.Equal(t, "abc123", job.Scene.Variables["token"].Value)
	assert.Equal(t, "string", job.Scene.Variables["token"].Type)
	assert.Equal(t, float64(42), job.Scene.Variables["userId"].Value)
	assert.Equal(t, true, job.Scene.Variables["active"].Value)
}

func TestParseJob_InvalidJSON(t *testing.T) {
	sub := &Subscriber{}

	payload := `{invalid json here}`

	job, err := sub.parseJob(payload)

	assert.Error(t, err)
	assert.NotNil(t, job) // returns &models.Job{} on error
}

func TestParseJob_EmptyObject(t *testing.T) {
	sub := &Subscriber{}

	payload := `{}`

	job, err := sub.parseJob(payload)

	require.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, "", job.JobID)
	assert.Equal(t, "", job.Type)
	assert.Nil(t, job.Scene.Variables)
}

func TestParseJob_EmptyString(t *testing.T) {
	sub := &Subscriber{}

	_, err := sub.parseJob("")

	assert.Error(t, err)
}
