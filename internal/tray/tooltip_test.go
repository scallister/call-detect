package tray

import (
	"testing"

	"github.com/scallister/call-detect/internal/state"
)

func TestTooltipBusy(t *testing.T) {
	t.Parallel()
	s := state.Snapshot{Busy: true, Microphone: true, Sources: []string{"Discord.exe"}}
	if got := Tooltip(s); got != "call-detect: on a call (mic, Discord.exe)" {
		t.Fatalf("got %q", got)
	}
	s.Webcam = true
	if got := Tooltip(s); got != "call-detect: on a call (mic+camera, Discord.exe)" {
		t.Fatalf("got %q", got)
	}
	if got := TooltipAlert(s, true, false); got != "call-detect: on a call (mic+camera, Discord.exe) — webhook failed" {
		t.Fatalf("failed: %q", got)
	}
	if got := TooltipAlert(s, false, true); got != "call-detect: on a call (mic+camera, Discord.exe) — update available" {
		t.Fatalf("update: %q", got)
	}
	if got := TooltipAlert(s, true, true); got != "call-detect: on a call (mic+camera, Discord.exe) — webhook failed — update available" {
		t.Fatalf("both: %q", got)
	}
}
