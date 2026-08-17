package detect

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/scallister/call-detect/internal/consentstore"
)

func TestConfirmIgnoresStaleConsent(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	mic := []consentstore.Usage{{App: "Discord.exe", InUse: true}}
	cam := []consentstore.Usage{{App: "Discord.exe", InUse: true}}

	idle := Confirm(mic, cam, Audio{}, now)
	if idle.Busy || idle.Microphone || idle.Webcam || len(idle.Sources) != 0 {
		t.Fatalf("stale consent should be idle: %+v", idle)
	}

	onCall := Confirm(mic, cam, Audio{Capture: []string{`C:\Users\a\Discord.exe`}}, now)
	if !onCall.Busy || !onCall.Microphone || !onCall.Webcam {
		t.Fatalf("active capture should confirm: %+v", onCall)
	}
	if !slices.Equal(onCall.Sources, []string{"Discord.exe"}) {
		t.Fatalf("sources %v", onCall.Sources)
	}
}

func TestConfirmCameraViaRender(t *testing.T) {
	t.Parallel()
	now := time.Now()
	cam := []consentstore.Usage{{App: "chrome.exe", InUse: true}}
	s := Confirm(nil, cam, Audio{Render: []string{"chrome.exe"}}, now)
	if !s.Busy || s.Microphone || !s.Webcam {
		t.Fatalf("%+v", s)
	}
}

func TestConfirmFallbackOnAudioError(t *testing.T) {
	t.Parallel()
	now := time.Now()
	mic := []consentstore.Usage{{App: "zoom.exe", InUse: true}}
	s := Confirm(mic, nil, Audio{Err: errors.New("wasapi down")}, now)
	if !s.Busy || !s.Microphone {
		t.Fatalf("fallback: %+v", s)
	}
}

func TestNormApp(t *testing.T) {
	t.Parallel()
	if got := normApp(`C:\Users\a\AppData\Local\Discord\Discord.exe`); got != "discord.exe" {
		t.Fatalf("path: %s", got)
	}
	if got := normApp("Discord.exe"); got != "discord.exe" {
		t.Fatalf("base: %s", got)
	}
}
