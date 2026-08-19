package tray

import (
	"testing"

	"github.com/scallister/call-detect/internal/state"
)

func TestActionChoices(t *testing.T) {
	t.Parallel()
	got := actionChoices(Actions{})
	if len(got) != 2 || got[0].ID != menuGitHub || got[1].ID != menuQuit {
		t.Fatalf("default: %+v", got)
	}
	got = actionChoices(Actions{
		Install:       func() error { return nil },
		Uninstall:     func() error { return nil },
		SetWebhookURL: func(string) error { return nil },
		AutostartOn:   func() bool { return false },
	})
	if len(got) != 4 || got[0].ID != menuInstall || got[1].ID != menuWebhook {
		t.Fatalf("not installed: %+v", got)
	}
	got = actionChoices(Actions{
		Install:     func() error { return nil },
		Uninstall:   func() error { return nil },
		AutostartOn: func() bool { return true },
	})
	if len(got) != 3 || got[0].ID != menuUninstall {
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
}

func TestHandleMenuQuit(t *testing.T) {
	t.Parallel()
	var quit bool
	handleMenuID(Actions{}, menuQuit, func() { quit = true })
	if !quit {
		t.Fatal("quit")
	}
}
