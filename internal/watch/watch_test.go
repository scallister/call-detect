package watch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/scallister/call-detect/internal/consentstore"
	"github.com/scallister/call-detect/internal/detect"
	"github.com/scallister/call-detect/internal/state"
	"github.com/scallister/call-detect/internal/webhook"
)

type staticAudio struct{ a detect.Audio }

func (s staticAudio) Sessions() detect.Audio { return s.a }

type staticCamera struct{ c detect.Camera }

func (s staticCamera) Streaming() detect.Camera { return s.c }

type memStore struct {
	mu      sync.Mutex
	entries map[string][]consentstore.Entry
}

func (m *memStore) List(capability string) ([]consentstore.Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]consentstore.Entry(nil), m.entries[capability]...), nil
}

func (m *memStore) setMic(inUse bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.entries == nil {
		m.entries = map[string][]consentstore.Entry{}
	}
	stop := uint64(0)
	if !inUse {
		stop = 2
	}
	m.entries[consentstore.CapabilityMicrophone] = []consentstore.Entry{
		{KeyName: "Discord.exe", Start: 1, Stop: stop},
	}
}

func TestRunDebouncedWebhookAndStatus(t *testing.T) {
	t.Parallel()
	var posts atomic.Int32
	var lastBusy atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var s state.Snapshot
		_ = json.NewDecoder(r.Body).Decode(&s)
		posts.Add(1)
		lastBusy.Store(s.Busy)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := &memStore{}
	store.setMic(false)
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status.json")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updates := make(chan state.Snapshot, 8)
	go func() {
		_ = Run(ctx, Options{
			Store:      store,
			Debounce:   40 * time.Millisecond,
			Poll:       10 * time.Millisecond,
			StatusPath: statusPath,
			Webhook:    &webhook.Client{URL: srv.URL, HTTP: srv.Client(), MaxTries: 1},
			OnUpdate: func(s state.Snapshot, _ bool) {
				updates <- s
			},
		})
	}()

	idle := waitSnap(t, updates, 500*time.Millisecond)
	if idle.Busy {
		t.Fatalf("expected initial idle, got %+v", idle)
	}
	if posts.Load() != 1 || lastBusy.Load() {
		t.Fatalf("initial webhook posts=%d busy=%v", posts.Load(), lastBusy.Load())
	}

	store.setMic(true)
	busy := waitBusy(t, updates, true, time.Second)
	if !busy.Microphone || busy.Sources[0] != "Discord.exe" {
		t.Fatalf("busy %+v", busy)
	}
	if posts.Load() != 2 || !lastBusy.Load() {
		t.Fatalf("busy webhook posts=%d busy=%v", posts.Load(), lastBusy.Load())
	}

	raw, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	var file state.Snapshot
	if err := json.Unmarshal(raw, &file); err != nil || !file.Busy {
		t.Fatalf("status file: %s %v", raw, err)
	}

	store.setMic(false)
	_ = waitBusy(t, updates, false, time.Second)
	if posts.Load() != 3 || lastBusy.Load() {
		t.Fatalf("idle webhook posts=%d busy=%v", posts.Load(), lastBusy.Load())
	}
}

func TestRunIgnoresConsentWithoutCapture(t *testing.T) {
	t.Parallel()
	store := &memStore{}
	store.setMic(true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := make(chan state.Snapshot, 8)
	go func() {
		_ = Run(ctx, Options{
			Store:    store,
			Audio:    staticAudio{},
			Debounce: 20 * time.Millisecond,
			Poll:     10 * time.Millisecond,
			OnUpdate: func(s state.Snapshot, _ bool) { updates <- s },
		})
	}()
	idle := waitSnap(t, updates, 500*time.Millisecond)
	if idle.Busy {
		t.Fatalf("expected idle without capture session, got %+v", idle)
	}
	select {
	case s := <-updates:
		if s.Busy {
			t.Fatalf("became busy: %+v", s)
		}
	case <-time.After(80 * time.Millisecond):
	}
}

func TestRunCameraOnlyWithoutAudio(t *testing.T) {
	t.Parallel()
	store := &memStore{entries: map[string][]consentstore.Entry{}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := make(chan state.Snapshot, 8)
	go func() {
		_ = Run(ctx, Options{
			Store:    store,
			Audio:    staticAudio{},
			Camera:   staticCamera{c: detect.Camera{Streaming: []string{"chrome.exe"}}},
			Debounce: 20 * time.Millisecond,
			Poll:     10 * time.Millisecond,
			OnUpdate: func(s state.Snapshot, _ bool) { updates <- s },
		})
	}()
	busy := waitBusy(t, updates, true, time.Second)
	if !busy.Webcam || busy.Microphone || busy.Sources[0] != "chrome.exe" {
		t.Fatalf("camera-only %+v", busy)
	}
}

func TestRunPostsImmediatelyAndReportsWebhookError(t *testing.T) {
	t.Parallel()
	var posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := make(chan state.Snapshot, 8)
	hooks := make(chan error, 8)
	go func() {
		_ = Run(ctx, Options{
			Store:    &memStore{},
			Debounce: time.Second,
			Poll:     20 * time.Millisecond,
			Webhook:  &webhook.Client{URL: srv.URL, HTTP: srv.Client(), MaxTries: 1},
			OnUpdate: func(s state.Snapshot, _ bool) { updates <- s },
			OnWebhook: func(err error) {
				hooks <- err
			},
		})
	}()
	_ = waitSnap(t, updates, 500*time.Millisecond)
	select {
	case err := <-hooks:
		if err == nil {
			t.Fatal("expected webhook error")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("no webhook callback")
	}
	if posts.Load() < 1 {
		t.Fatal("expected launch POST")
	}
}

func TestRunAuthoritativeCaptureWithoutConsent(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := make(chan state.Snapshot, 8)
	go func() {
		_ = Run(ctx, Options{
			Store:    consentstore.None{},
			Audio:    staticAudio{a: detect.Audio{Capture: []string{"firefox"}, Authoritative: true}},
			Debounce: 20 * time.Millisecond,
			Poll:     10 * time.Millisecond,
			OnUpdate: func(s state.Snapshot, _ bool) { updates <- s },
		})
	}()
	busy := waitBusy(t, updates, true, time.Second)
	if !busy.Microphone || busy.Webcam || busy.Sources[0] != "firefox" {
		t.Fatalf("authoritative %+v", busy)
	}
}

func waitSnap(t *testing.T, ch <-chan state.Snapshot, d time.Duration) state.Snapshot {
	t.Helper()
	select {
	case s := <-ch:
		return s
	case <-time.After(d):
		t.Fatal("timeout waiting for snapshot")
		return state.Snapshot{}
	}
}

func waitBusy(t *testing.T, ch <-chan state.Snapshot, busy bool, d time.Duration) state.Snapshot {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		select {
		case s := <-ch:
			if s.Busy == busy {
				return s
			}
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Fatalf("timeout waiting for busy=%v", busy)
	return state.Snapshot{}
}
