package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"video-uploader-agent/internal/config"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type UploadSuccessRequest struct {
	OrderID   string `json:"order_id"`
	FileName  string `json:"file_name"`
	ObjectKey string `json:"object_key"`
	FileSize  int64  `json:"file_size"`
	Status    string `json:"status"`
}

type UploadFailedRequest struct {
	OrderID      string `json:"order_id"`
	FileName     string `json:"file_name"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
}

func NewClient(cfg *config.Config) *Client {
	timeout := time.Duration(cfg.Backend.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &Client{
		baseURL: strings.TrimRight(cfg.Backend.BaseURL, "/"),
		apiKey:  cfg.Backend.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) NotifyUploadSuccess(req UploadSuccessRequest) error {
	return c.postJSON("/internal/uploads/complete", req)
}

func (c *Client) NotifyUploadFailed(req UploadFailedRequest) error {
	return c.postJSON("/internal/uploads/failed", req)
}

func (c *Client) postJSON(path string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("backend returned status %d", resp.StatusCode)
	}

	return nil
}
