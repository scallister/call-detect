package tray

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/scallister/call-detect/internal/project"
	"github.com/scallister/call-detect/internal/state"
	"github.com/scallister/call-detect/internal/version"
)

func TestActionChoices(t *testing.T) {
	t.Parallel()
	got := actionChoices(Actions{})
	if len(got) != 3 || got[0].ID != menuUpdate || got[1].ID != menuGitHub || got[2].ID != menuQuit {
		t.Fatalf("default: %+v", got)
	}
	got = actionChoices(Actions{
		Install:       func() error { return nil },
		Uninstall:     func() error { return nil },
		SetWebhookURL: func(string) error { return nil },
		AutostartOn:   func() bool { return false },
	})
	if len(got) != 5 || got[0].ID != menuInstall || got[1].ID != menuWebhook || got[2].ID != menuUpdate {
		t.Fatalf("not installed: %+v", got)
	}
	got = actionChoices(Actions{
		Install:     func() error { return nil },
		Uninstall:   func() error { return nil },
		AutostartOn: func() bool { return true },
	})
	if len(got) != 4 || got[0].ID != menuUninstall || got[1].ID != menuUpdate {
		t.Fatalf("installed: %+v", got)
	}
}

func TestStatusLines(t *testing.T) {
	t.Parallel()
	s := state.Snapshot{Busy: true, Microphone: true, Sources: []string{"Firefox"}}
	lines := statusLines(s, true)
	if lines[0] != "On a call — webhook failed" {
		t.Fatalf("status: %q", lines[0])
	}
	if lines[1] != "Microphone: yes" || lines[2] != "Webcam: no" {
		t.Fatalf("bools: %v", lines)
	}
	if lines[3] != "Sources: Firefox" {
		t.Fatalf("sources: %q", lines[3])
	}
	if lines[4] != "Version: dev" {
		t.Fatalf("version: %q", lines[4])
	}
}

func TestHandleMenuQuit(t *testing.T) {
	t.Parallel()
	var quit bool
	handleMenuID(Actions{}, menuQuit, func() { quit = true })
	if !quit {
		t.Fatal("quit")
	}
}

func TestOfferRemoteUpdateSkipsDev(t *testing.T) {
	// Version is "dev" in tests; a silent check must not hit the network.
	OfferRemoteUpdate(context.Background(), false)
}

func TestOfferRemoteUpdateHonorsCancel(t *testing.T) {
	prevVer := version.Version
	version.Version = "v0.0.7"
	t.Cleanup(func() { version.Version = prevVer })

	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	prevAPI := project.LatestAPI
	project.LatestAPI = srv.URL
	t.Cleanup(func() { project.LatestAPI = prevAPI })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		OfferRemoteUpdate(ctx, false)
		close(done)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("server was not reached")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("did not return after cancel")
	}
}
