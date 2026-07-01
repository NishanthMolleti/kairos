package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const hfEmbedURL = "https://api-inference.huggingface.co/pipeline/feature-extraction/nomic-ai/nomic-embed-text-v1"

type hfEmbedRequest struct {
	Inputs string `json:"inputs"`
}

// Embed calls HuggingFace nomic-embed-text-v1 and returns a 768-dimensional vector.
func Embed(text, hfAPIKey string) ([]float32, error) {
	body, err := json.Marshal(hfEmbedRequest{Inputs: text})
	if err != nil {
		return nil, fmt.Errorf("embed: marshal: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, hfEmbedURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("embed: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+hfAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed: http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("embed: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed: HuggingFace %d: %s", resp.StatusCode, raw)
	}

	// API returns [][]float32 for batch; we send one string so take index [0].
	var outer [][]float32
	if err := json.Unmarshal(raw, &outer); err != nil {
		return nil, fmt.Errorf("embed: unmarshal: %w", err)
	}
	if len(outer) == 0 {
		return nil, fmt.Errorf("embed: empty response")
	}
	if len(outer[0]) != 768 {
		return nil, fmt.Errorf("embed: expected 768 dims, got %d", len(outer[0]))
	}
	return outer[0], nil
}
