package dump

import (
	"strings"
	"testing"
	"time"

	"github.com/scallister/call-detect/internal/consentstore"
	"github.com/scallister/call-detect/internal/detect"
	"github.com/scallister/call-detect/internal/state"
)

func TestWrite(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	mic := []consentstore.Usage{
		{App: "Discord.exe", InUse: true, Start: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)},
	}
	if err := Write(&b, mic, nil); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	if !strings.Contains(got, "IN USE") || !strings.Contains(got, "Discord.exe") {
		t.Fatalf("mic: %s", got)
	}
	if !strings.Contains(got, "(no records)") {
		t.Fatalf("empty webcam: %s", got)
	}
}

func TestWriteAudio(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	if err := WriteAudio(&b, detect.Audio{Capture: []string{"Discord.exe"}}, state.Snapshot{Busy: true, Microphone: true, Sources: []string{"Discord.exe"}}); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	if !strings.Contains(got, "Discord.exe") || !strings.Contains(got, "busy=true") {
		t.Fatalf("%s", got)
	}
}
