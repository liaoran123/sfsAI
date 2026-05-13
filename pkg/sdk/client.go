package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

type MemoryUnit struct {
	ID        string                 `json:"id,omitempty"`
	SessionID string                 `json:"session_id"`
	Content   string                 `json:"content"`
	Embedding []float32              `json:"embedding,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at,omitempty"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
}

type MemoryQuery struct {
	SessionID string    `json:"session_id"`
	TimeStart time.Time `json:"time_start,omitempty"`
	TimeEnd   time.Time `json:"time_end,omitempty"`
	TopK      int       `json:"top_k,omitempty"`
}

type SearchResult struct {
	Memories []MemoryUnit `json:"memories"`
	Count    int          `json:"count"`
}

type InsertResult struct {
	MemoryID  string    `json:"memory_id"`
	CreatedAt time.Time `json:"created_at"`
}

type StatsResult struct {
	MemoryUsageMB int    `json:"memory_usage_mb"`
	TotalRecords  int    `json:"total_records"`
	Uptime        string `json:"uptime"`
	Running       bool   `json:"running"`
}

type HealthResult struct {
	Status  string `json:"status"`
	Uptime  string `json:"uptime"`
	Version string `json:"version"`
}

func NewClient(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type Option func(*Client)

func WithAPIKey(key string) Option {
	return func(c *Client) { c.apiKey = key }
}

func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	if resp.StatusCode >= 400 {
		var e struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &e) == nil && e.Error != "" {
			return nil, fmt.Errorf("API %d: %s", resp.StatusCode, e.Error)
		}
		return nil, fmt.Errorf("API status %d", resp.StatusCode)
	}

	return respBody, nil
}
