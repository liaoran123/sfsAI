package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

func (c *Client) InsertMemory(sessionID, content string, embedding []float32, metadata map[string]interface{}) (*InsertResult, error) {
	body, err := c.doRequest("POST", "/api/v1/memories", map[string]interface{}{
		"session_id": sessionID,
		"content":    content,
		"embedding":  embedding,
		"metadata":   metadata,
	})
	if err != nil {
		return nil, err
	}

	var r InsertResult
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &r, nil
}

func (c *Client) SearchMemories(sessionID string, topK int, timeStart, timeEnd time.Time) (*SearchResult, error) {
	body := map[string]interface{}{"session_id": sessionID, "top_k": topK}
	if !timeStart.IsZero() {
		body["time_start"] = timeStart.Format(time.RFC3339)
	}
	if !timeEnd.IsZero() {
		body["time_end"] = timeEnd.Format(time.RFC3339)
	}

	resp, err := c.doRequest("POST", "/api/v1/memories", body)
	if err != nil {
		return nil, err
	}

	var r SearchResult
	if err := json.Unmarshal(resp, &r); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &r, nil
}

func (c *Client) GetMemory(sessionID, memoryID string) (*MemoryUnit, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/v1/memories/%s/%s", sessionID, memoryID), nil)
	if err != nil {
		return nil, err
	}

	var mem MemoryUnit
	if err := json.Unmarshal(resp, &mem); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &mem, nil
}

func (c *Client) GetRecentMemories(sessionID string, limit int, window time.Duration) (*SearchResult, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/v1/memories/recent/%s?limit=%d&window=%s", sessionID, limit, window), nil)
	if err != nil {
		return nil, err
	}

	var r SearchResult
	if err := json.Unmarshal(resp, &r); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &r, nil
}

func (c *Client) WipeMemories(sessionID string, memoryIDs ...string) error {
	_, err := c.doRequest("DELETE", "/api/v1/memories/wipe", map[string]interface{}{
		"session_id": sessionID,
		"memory_ids": memoryIDs,
	})
	return err
}

func (c *Client) CompressMemories(sessionID string, olderThan time.Duration) error {
	_, err := c.doRequest("POST", fmt.Sprintf("/api/v1/memories/compress/%s?older_than=%s", sessionID, olderThan), nil)
	return err
}

func (c *Client) SemanticSearch(sessionID string, vector []float32, topK int) (*SearchResult, error) {
	resp, err := c.doRequest("POST", "/api/v1/memories/semantic", map[string]interface{}{
		"session_id": sessionID,
		"vector":     vector,
		"top_k":      topK,
	})
	if err != nil {
		return nil, err
	}

	var r SearchResult
	if err := json.Unmarshal(resp, &r); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &r, nil
}

func (c *Client) Health() (*HealthResult, error) {
	resp, err := c.doRequest("GET", "/api/v1/health", nil)
	if err != nil {
		return nil, err
	}

	var r HealthResult
	if err := json.Unmarshal(resp, &r); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &r, nil
}

func (c *Client) GetStats() (*StatsResult, error) {
	resp, err := c.doRequest("GET", "/api/v1/stats", nil)
	if err != nil {
		return nil, err
	}

	var r StatsResult
	if err := json.Unmarshal(resp, &r); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &r, nil
}
