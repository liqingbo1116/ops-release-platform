package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	taskStream = "ops:agent:tasks"
)

type Queue struct {
	client *redis.Client
}

type Task struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	Action        string            `json:"action"`
	AgentID       string            `json:"agentId,omitempty"`
	EnvironmentID string            `json:"environmentId,omitempty"`
	Payload       map[string]string `json:"payload,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
}

func NewQueue(addr string) (*Queue, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &Queue{client: client}, nil
}

func (q *Queue) Close() error {
	return q.client.Close()
}

func (q *Queue) Enqueue(ctx context.Context, task Task) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return err
	}
	return q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: taskStream,
		Values: map[string]any{
			"taskId":  task.ID,
			"type":    task.Type,
			"action":  task.Action,
			"payload": string(payload),
		},
	}).Err()
}

func (q *Queue) Status(ctx context.Context, taskID string) (map[string]string, error) {
	return q.client.HGetAll(ctx, statusKey(taskID)).Result()
}

func (q *Queue) Logs(ctx context.Context, taskID string) ([]string, error) {
	return q.client.LRange(ctx, logKey(taskID), 0, -1).Result()
}

func statusKey(taskID string) string {
	return "ops:agent:task:" + taskID + ":status"
}

func logKey(taskID string) string {
	return "ops:agent:task:" + taskID + ":logs"
}
