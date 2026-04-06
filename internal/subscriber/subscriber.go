package subscriber

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/test-fleet/test-runner/internal/config"
	"github.com/test-fleet/test-runner/pkg/models"
)

type Subscriber struct {
	cfg     *config.Config
	client  *redis.Client
	jobChan chan *models.Job
	logger  *log.Logger
}

func NewSubscriber(cfg *config.Config, client *redis.Client, jobChan chan *models.Job, logger *log.Logger) *Subscriber {
	return &Subscriber{
		client:  client,
		cfg:     cfg,
		jobChan: jobChan,
		logger:  logger,
	}
}

func (s *Subscriber) Subscribe(ctx context.Context) error {
	pubsub := s.client.Subscribe(ctx, s.cfg.Channel)
	defer pubsub.Close()

	s.logger.Printf("Subscribed to channel %s", s.cfg.Channel)

	for {
		select {
		case <-ctx.Done():
			log.Println("Subcriber shutting down")
			return ctx.Err()

		case msg := <-pubsub.Channel():
			job, err := s.parseJob(msg.Payload) // parse job here
			if err != nil {
				return err
			}
			select {
			case s.jobChan <- job:
				s.logger.Printf("job %s recieved", job.JobID)

			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func (s *Subscriber) parseJob(payload string) (*models.Job, error) { // convert payload to data model
	jobBytes := []byte(payload)

	var job models.Job

	err := json.Unmarshal(jobBytes, &job)
	if err != nil {
		return &models.Job{}, err
	}

	return &job, nil
}
