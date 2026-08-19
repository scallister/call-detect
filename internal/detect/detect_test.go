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

	idle := Confirm(mic, cam, Audio{}, Camera{}, now)
	if idle.Busy || idle.Microphone || idle.Webcam || len(idle.Sources) != 0 {
		t.Fatalf("stale consent should be idle: %+v", idle)
	}

	voice := Confirm(mic, cam, Audio{Capture: []string{`C:\Users\a\Discord.exe`}}, Camera{}, now)
	if !voice.Busy || !voice.Microphone || voice.Webcam {
		t.Fatalf("voice-only capture should be mic, not webcam: %+v", voice)
	}
	if !slices.Equal(voice.Sources, []string{"Discord.exe"}) {
		t.Fatalf("sources %v", voice.Sources)
	}

	video := Confirm(mic, cam, Audio{Capture: []string{`C:\Users\a\Discord.exe`}}, Camera{Streaming: []string{"Discord.exe"}}, now)
	if !video.Busy || !video.Microphone || !video.Webcam {
		t.Fatalf("capture + camera stream should confirm both: %+v", video)
	}
}

func TestConfirmBrowserWebcamWithoutAudio(t *testing.T) {
	t.Parallel()
	now := time.Now()
	cam := []consentstore.Usage{{App: "chrome.exe", InUse: true}}
	s := Confirm(nil, cam, Audio{}, Camera{Streaming: []string{"chrome.exe"}}, now)
	if !s.Busy || s.Microphone || !s.Webcam {
		t.Fatalf("video-only browser preview: %+v", s)
	}
	if !slices.Equal(s.Sources, []string{"chrome.exe"}) {
		t.Fatalf("sources %v", s.Sources)
	}
}

func TestConfirmCameraStreamWithoutConsent(t *testing.T) {
	t.Parallel()
	now := time.Now()
	s := Confirm(nil, nil, Audio{}, Camera{Streaming: []string{"msedge.exe"}}, now)
	if !s.Busy || !s.Webcam || s.Microphone {
		t.Fatalf("streaming camera is enough: %+v", s)
	}
}

func TestConfirmCameraFallbackWhenMonitorFails(t *testing.T) {
	t.Parallel()
	now := time.Now()
	cam := []consentstore.Usage{{App: "chrome.exe", InUse: true}}
	s := Confirm(nil, cam, Audio{Render: []string{"chrome.exe"}}, Camera{Err: errors.New("no monitor")}, now)
	if !s.Busy || s.Microphone || !s.Webcam {
		t.Fatalf("fallback via render: %+v", s)
	}
}

func TestConfirmFallbackOnAudioAndCameraError(t *testing.T) {
	t.Parallel()
	now := time.Now()
	mic := []consentstore.Usage{{App: "zoom.exe", InUse: true}}
	s := Confirm(mic, nil, Audio{Err: errors.New("wasapi down")}, Camera{Err: errors.New("no monitor")}, now)
	if !s.Busy || !s.Microphone {
		t.Fatalf("double fallback: %+v", s)
	}
}

func TestConfirmAudioErrorUsesConsentMicAndCameraStream(t *testing.T) {
	t.Parallel()
	now := time.Now()
	mic := []consentstore.Usage{{App: "zoom.exe", InUse: true}}
	s := Confirm(mic, nil, Audio{Err: errors.New("wasapi down")}, Camera{Streaming: []string{"chrome.exe"}}, now)
	if !s.Busy || !s.Microphone || !s.Webcam {
		t.Fatalf("%+v", s)
	}
}

func TestConfirmAuthoritativeCaptureWithoutConsent(t *testing.T) {
	t.Parallel()
	now := time.Now()
	s := Confirm(nil, nil, Audio{Capture: []string{"firefox"}, Authoritative: true}, Camera{}, now)
	if !s.Busy || !s.Microphone || s.Webcam {
		t.Fatalf("authoritative capture: %+v", s)
	}
	if !slices.Equal(s.Sources, []string{"firefox"}) {
		t.Fatalf("sources %v", s.Sources)
	}

	idle := Confirm([]consentstore.Usage{{App: "Discord.exe", InUse: true}}, nil, Audio{Authoritative: true}, Camera{}, now)
	if idle.Busy || idle.Microphone {
		t.Fatalf("authoritative empty capture ignores consent: %+v", idle)
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
