package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"createdAt"`
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

func (q *Queue) StartMockWorker(ctx context.Context) {
	go q.runWorker(ctx)
}

func (q *Queue) runWorker(ctx context.Context) {
	lastID := "$"
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		streams, err := q.client.XRead(ctx, &redis.XReadArgs{
			Streams: []string{taskStream, lastID},
			Count:   1,
			Block:   5 * time.Second,
		}).Result()
		if err != nil {
			if err != redis.Nil && ctx.Err() == nil {
				log.Printf("mock agent xread failed: %v", err)
			}
			continue
		}
		for _, stream := range streams {
			for _, message := range stream.Messages {
				lastID = message.ID
				taskID, _ := message.Values["taskId"].(string)
				taskType, _ := message.Values["type"].(string)
				if taskID == "" {
					continue
				}
				go q.simulateTask(context.Background(), taskID, taskType)
			}
		}
	}
}

func (q *Queue) simulateTask(ctx context.Context, taskID string, taskType string) {
	steps := []string{"receive-task", "prepare-runtime", "execute-steps", "collect-logs", "finish"}
	for index, step := range steps {
		status := "RUNNING"
		if index == len(steps)-1 {
			status = "SUCCESS"
		}
		now := time.Now().Format(time.RFC3339)
		q.client.HSet(ctx, statusKey(taskID), map[string]any{
			"taskId":    taskID,
			"type":      taskType,
			"step":      step,
			"status":    status,
			"updatedAt": now,
		})
		q.client.RPush(ctx, logKey(taskID), fmt.Sprintf("[%s] %s %s", now, status, step))
		time.Sleep(300 * time.Millisecond)
	}
}

func statusKey(taskID string) string {
	return "ops:agent:task:" + taskID + ":status"
}

func logKey(taskID string) string {
	return "ops:agent:task:" + taskID + ":logs"
}
