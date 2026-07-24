package worker

import (
	"fmt"
	"log"

	"github.com/hibiken/asynq"

	"github.com/timeless/backend/internal/config"
)

// Client wraps asynq.Client for enqueuing background tasks from the API server.
type Client struct {
	inner *asynq.Client
}

func NewClient(cfg *config.Config) (*Client, error) {
	opt, err := asynq.ParseRedisURI(cfg.RedisURL)
	if err != nil {
		log.Printf("warning: could not parse redis URI for worker client, falling back: %v", err)
		opt = asynq.RedisClientOpt{Addr: "localhost:6379"}
	}
	c := asynq.NewClient(opt)
	return &Client{inner: c}, nil
}

func (c *Client) Close() error {
	return c.inner.Close()
}

// Enqueue puts a task on the default queue.
func (c *Client) Enqueue(taskType string, payload TaskPayload, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	t, err := NewTask(taskType, payload)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return c.inner.Enqueue(t, opts...)
}

// EnqueueCritical puts a task on the critical queue.
func (c *Client) EnqueueCritical(taskType string, payload TaskPayload) (*asynq.TaskInfo, error) {
	return c.Enqueue(taskType, payload, asynq.Queue("critical"))
}

// EnqueueLow puts a task on the low-priority queue.
func (c *Client) EnqueueLow(taskType string, payload TaskPayload) (*asynq.TaskInfo, error) {
	return c.Enqueue(taskType, payload, asynq.Queue("low"))
}
