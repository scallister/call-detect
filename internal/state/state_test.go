package state

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/scallister/call-detect/internal/consentstore"
)

func TestFromUsages(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	mic := []consentstore.Usage{
		{App: "Discord.exe", InUse: true},
		{App: "chrome.exe", InUse: false},
	}
	cam := []consentstore.Usage{
		{App: "Discord.exe", InUse: true},
	}
	s := FromUsages(mic, cam, now)
	if !s.Busy || !s.Microphone || !s.Webcam {
		t.Fatalf("bools: %+v", s)
	}
	if !slices.Equal(s.Sources, []string{"Discord.exe"}) {
		t.Fatalf("sources: %v", s.Sources)
	}
	if !s.UpdatedAt.Equal(now) {
		t.Fatalf("updated_at: %s", s.UpdatedAt)
	}

	idle := FromUsages(nil, nil, now)
	if idle.Busy || idle.Microphone || idle.Webcam || idle.Sources == nil || len(idle.Sources) != 0 {
		t.Fatalf("idle: %+v", idle)
	}
}

func TestDebouncerFlickerIgnored(t *testing.T) {
	t.Parallel()
	d := Debouncer{Delay: 2 * time.Second}
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	idle := Snapshot{Sources: []string{}}
	if r := d.Observe(idle, t0); r.Changed {
		t.Fatalf("first tick should wait: %+v", r)
	}
	first := d.Observe(idle, t0.Add(2*time.Second))
	if !first.BoolsChanged || first.State.Busy {
		t.Fatalf("first: %+v", first)
	}

	r1 := d.Observe(Snapshot{Busy: true, Microphone: true, Sources: []string{"a"}}, t0.Add(3*time.Second))
	if r1.Changed || r1.State.Busy {
		t.Fatalf("should still be idle during debounce: %+v", r1)
	}

	r2 := d.Observe(idle, t0.Add(3500*time.Millisecond))
	if r2.Changed || r2.State.Busy {
		t.Fatalf("flicker back to idle: %+v", r2)
	}

	r3 := d.Observe(Snapshot{Busy: true, Microphone: true, Sources: []string{"a"}}, t0.Add(4*time.Second))
	if r3.Changed {
		t.Fatalf("new pending should not publish yet: %+v", r3)
	}

	r4 := d.Observe(Snapshot{Busy: true, Microphone: true, Sources: []string{"a"}}, t0.Add(6*time.Second))
	if !r4.BoolsChanged || !r4.State.Busy || r4.State.Sources[0] != "a" {
		t.Fatalf("expected publish: %+v", r4)
	}
}

func TestDebouncerSourcesOnly(t *testing.T) {
	t.Parallel()
	d := Debouncer{Delay: time.Second}
	t0 := time.Now()
	busy := Snapshot{Busy: true, Microphone: true, Sources: []string{"Discord.exe"}}
	d.Observe(busy, t0)
	d.Observe(busy, t0.Add(time.Second))
	r := d.Observe(Snapshot{Busy: true, Microphone: true, Sources: []string{"chrome.exe"}}, t0.Add(time.Second+time.Millisecond))
	if !r.Changed || r.BoolsChanged {
		t.Fatalf("sources-only: %+v", r)
	}
	if !slices.Equal(r.State.Sources, []string{"chrome.exe"}) {
		t.Fatalf("sources: %v", r.State.Sources)
	}
}

func TestExampleJSON(t *testing.T) {
	t.Parallel()
	var s Snapshot
	if err := json.Unmarshal([]byte(ExampleJSON), &s); err != nil {
		t.Fatal(err)
	}
	if !s.Busy || !s.Microphone || s.Webcam || !slices.Equal(s.Sources, []string{"Discord.exe"}) {
		t.Fatalf("example payload: %+v", s)
	}
	if s.UpdatedAt.IsZero() {
		t.Fatal("updated_at")
	}
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"call":true`) {
		t.Fatalf("json: %s", raw)
	}
	if strings.Contains(string(raw), `"busy"`) {
		t.Fatalf("legacy busy key: %s", raw)
	}
}
