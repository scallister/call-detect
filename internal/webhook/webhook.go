// Package webhook POSTs snapshot JSON to an optional URL.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/scallister/call-detect/internal/state"
)

type statusError struct {
	status int
	msg    string
}

func (e *statusError) Error() string { return e.msg }

func (e *statusError) retryable() bool {
	return e.status >= 500 || e.status == http.StatusTooManyRequests
}

// Client POSTs snapshots. An empty URL is a no-op.
type Client struct {
	URL      string
	HTTP     *http.Client
	MaxTries int
	Backoff  time.Duration

	mu sync.Mutex
}

// GetURL returns the current destination.
func (c *Client) GetURL() string {
	return c.url()
}

// SetURL changes the destination. Empty disables POSTs.
func (c *Client) SetURL(url string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.URL = url
	c.mu.Unlock()
}

func (c *Client) url() string {
	if c == nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.URL
}

func (c *Client) enabled() bool {
	return c.url() != ""
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 5 * time.Second}
}

func (c *Client) tries() int {
	if c.MaxTries > 0 {
		return c.MaxTries
	}
	return 3
}

func (c *Client) backoff() time.Duration {
	if c.Backoff > 0 {
		return c.Backoff
	}
	return 500 * time.Millisecond
}

// Post sends s as JSON. It retries transient failures a few times.
func (c *Client) Post(ctx context.Context, s state.Snapshot) error {
	if !c.enabled() {
		return nil
	}
	body, err := json.Marshal(s)
	if err != nil {
		return err
	}

	var last error
	delay := c.backoff()
	for attempt := 1; attempt <= c.tries(); attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		last = c.doOnce(ctx, body)
		if last == nil {
			return nil
		}
		var se *statusError
		if errors.As(last, &se) && !se.retryable() {
			return last
		}
		if attempt == c.tries() {
			break
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		delay *= 2
	}
	return last
}

func (c *Client) doOnce(ctx context.Context, body []byte) error {
	dest := c.url()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dest, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return &statusError{
		status: resp.StatusCode,
		msg:    fmt.Sprintf("webhook %s: %s", dest, resp.Status),
	}
}
