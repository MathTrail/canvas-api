package centrifugo

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client publishes events to Centrifugo via its HTTP API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

type publishRequest struct {
	Channel string `json:"channel"`
	// B64Data carries Protobuf-encoded binary payload as base64.
	// Centrifugo forwards it as-is to the subscribed client.
	B64Data string `json:"b64data"`
}

// Publish sends a Protobuf-encoded binary payload to the given Centrifugo channel.
func (c *Client) Publish(ctx context.Context, channel string, data []byte) error {
	body := publishRequest{
		Channel: channel,
		B64Data: base64.StdEncoding.EncodeToString(data),
	}
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal publish request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/publish", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("build publish request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("centrifugo publish: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("centrifugo publish: unexpected status %d", resp.StatusCode)
	}
	return nil
}
