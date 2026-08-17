package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/scallister/call-detect/internal/state"
)

func TestPostEmptyURL(t *testing.T) {
	t.Parallel()
	c := &Client{}
	if err := c.Post(context.Background(), state.Snapshot{Busy: true}); err != nil {
		t.Fatal(err)
	}
}

func TestPostJSON(t *testing.T) {
	t.Parallel()
	var got state.Snapshot
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type %s", r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("json: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &Client{URL: srv.URL, HTTP: srv.Client()}
	s := state.Snapshot{
		Busy:       true,
		Microphone: true,
		Sources:    []string{"Discord.exe"},
		UpdatedAt:  time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC),
	}
	if err := c.Post(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	if !got.Busy || !got.Microphone || got.Sources[0] != "Discord.exe" {
		t.Fatalf("payload %+v", got)
	}
}

func TestPostRetriesThenOK(t *testing.T) {
	t.Parallel()
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := &Client{
		URL:      srv.URL,
		HTTP:     srv.Client(),
		MaxTries: 3,
		Backoff:  time.Millisecond,
	}
	if err := c.Post(context.Background(), state.Snapshot{Busy: true}); err != nil {
		t.Fatal(err)
	}
	if n.Load() != 3 {
		t.Fatalf("tries %d", n.Load())
	}
}

func TestPostNoRetryOn4xx(t *testing.T) {
	t.Parallel()
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c := &Client{URL: srv.URL, HTTP: srv.Client(), MaxTries: 3, Backoff: time.Millisecond}
	if err := c.Post(context.Background(), state.Snapshot{}); err == nil {
		t.Fatal("expected error")
	}
	if n.Load() != 1 {
		t.Fatalf("tries %d", n.Load())
	}
}
