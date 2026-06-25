package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
)

// webhookClient Webhook 客户端
var webhookClient = &http.Client{
	Timeout: 30 * time.Second,
}

// SendWebhook 发送 Webhook 回调
func SendWebhook(ctx context.Context, payload *WebhookPayload) {
	if payload == nil || payload.TaskID == "" {
		return
	}

	// 获取任务信息以获取 webhook URL
	info, ok := GetTask(payload.TaskID)
	if !ok || info.Webhook == "" {
		return
	}

	webhookURL := info.Webhook

	// Async send with retries.
	go func() {
		var logger *log.Logger
		if ctx != nil {
			logger = log.FromContext(ctx).With("task_id", payload.TaskID)
		} else {
			logger = log.Default().With("task_id", payload.TaskID)
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			logger.Errorf("Failed to marshal webhook payload: %v", err)
			return
		}

		// 重试 3 次
		for i := range 3 {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, webhookURL, bytes.NewBuffer(payloadBytes))
			if err != nil {
				logger.Errorf("Failed to create webhook request: %v", err)
				return
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "SaveAny-Bot/1.0")

			resp, err := webhookClient.Do(req)
			if err != nil {
				logger.Warnf("Webhook request failed (attempt %d/3): %v", i+1, err)
				time.Sleep(time.Second * time.Duration(i+1))
				continue
			}
			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				logger.Debugf("Webhook sent successfully: %s", webhookURL)
				return
			}

			logger.Warnf("Webhook returned non-2xx status (attempt %d/3): %d", i+1, resp.StatusCode)
			time.Sleep(time.Second * time.Duration(i+1))
		}

		logger.Errorf("Failed to send webhook after 3 attempts")
	}()
}

// CreateWebhookPayload creates a Webhook payload.
func CreateWebhookPayload(taskID string, taskType string, status TaskStatus, storage, path string, err error) *WebhookPayload {
	payload := &WebhookPayload{
		TaskID:  taskID,
		Type:    taskType,
		Status:  status,
		Storage: storage,
		Path:    path,
	}

	if status == TaskStatusCompleted || status == TaskStatusFailed {
		now := time.Now()
		payload.CompletedAt = &now
	}

	if err != nil {
		payload.Error = err.Error()
	}

	return payload
}
