package ory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Session struct {
	Identity struct {
		ID string `json:"id"`
	} `json:"identity"`
	Active bool `json:"active"`
}

// Client performs Ory Kratos session validation.
type Client struct {
	url        string
	httpClient *http.Client
}

// NewClient creates a Kratos client with a connection timeout.
func NewClient(url string) *Client {
	return &Client{
		url:        url,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// WhoAmI validates an Ory Kratos session cookie by calling the Kratos whoami
// endpoint and returns the session. The caller must forward the original request
// cookies to authenticate the session with Kratos.
func (c *Client) WhoAmI(ctx context.Context, cookies []*http.Cookie) (*Session, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/sessions/whoami", nil)
	if err != nil {
		return nil, fmt.Errorf("build whoami request: %w", err)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kratos whoami: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthenticated
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kratos whoami: unexpected status %d", resp.StatusCode)
	}

	var s Session
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, fmt.Errorf("decode whoami response: %w", err)
	}
	if !s.Active {
		return nil, ErrUnauthenticated
	}
	return &s, nil
}

var ErrUnauthenticated = fmt.Errorf("unauthenticated")
