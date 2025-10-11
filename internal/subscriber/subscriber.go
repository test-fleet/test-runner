package subscriber

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/test-fleet/test-runner/internal/config"
)

type Subscriber struct {
	cfg     *config.Config
	client  *redis.Client
	jobChan chan string
	logger  *log.Logger
}

func NewSubscriber(cfg *config.Config, client *redis.Client, jobChan chan string, logger *log.Logger) *Subscriber {
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
			job := msg.Payload
			_ = s.parseJob(msg.Payload) // parse job here
			select {
			case s.jobChan <- job:
				s.logger.Println("job added to chan")

			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func (s *Subscriber) parseJob(payload string) error { // convert payload to data model
	_ = payload
	return nil
}
