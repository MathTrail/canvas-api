package ory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Session struct {
	Identity struct {
		ID string `json:"id"`
	} `json:"identity"`
	Active bool `json:"active"`
}

// WhoAmI validates an Ory Kratos session cookie by calling the Kratos whoami
// endpoint and returns the session. The caller must forward the original request
// cookies to authenticate the session with Kratos.
func WhoAmI(ctx context.Context, kratosURL string, cookies []*http.Cookie) (*Session, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, kratosURL+"/sessions/whoami", nil)
	if err != nil {
		return nil, fmt.Errorf("build whoami request: %w", err)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp, err := http.DefaultClient.Do(req)
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
