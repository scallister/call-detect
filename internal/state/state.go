// Package state holds the debounced call / device snapshot.
package state

import (
	"slices"
	"time"

	"github.com/scallister/call-detect/internal/consentstore"
)

// Snapshot is the public status payload (JSON webhook and status.json).
type Snapshot struct {
	Busy       bool      `json:"busy"`
	Microphone bool      `json:"microphone"`
	Webcam     bool      `json:"webcam"`
	Sources    []string  `json:"sources"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ExampleJSON is a sample snapshot in the same shape as the webhook POST body
// and status.json. The tray webhook dialog shows this next to the URL field.
const ExampleJSON = `{
  "busy": true,
  "microphone": true,
  "webcam": false,
  "sources": ["Discord.exe"],
  "updated_at": "2026-08-17T12:00:00Z"
}`

// FromUsages builds an undebounced snapshot from ConsentStore reads.
func FromUsages(mic, cam []consentstore.Usage, now time.Time) Snapshot {
	micOn := consentstore.AnyInUse(mic)
	camOn := consentstore.AnyInUse(cam)
	sources := uniqueSorted(append(consentstore.InUseApps(mic), consentstore.InUseApps(cam)...))
	if sources == nil {
		sources = []string{}
	}
	return Snapshot{
		Busy:       micOn || camOn,
		Microphone: micOn,
		Webcam:     camOn,
		Sources:    sources,
		UpdatedAt:  now.UTC(),
	}
}

func uniqueSorted(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	slices.Sort(out)
	return out
}

// EqualBools reports whether busy / microphone / webcam match.
func EqualBools(a, b Snapshot) bool {
	return a.Busy == b.Busy && a.Microphone == b.Microphone && a.Webcam == b.Webcam
}

// Result is the debouncer output for one observation.
type Result struct {
	State        Snapshot
	Changed      bool
	BoolsChanged bool
}

// Debouncer publishes a new snapshot only after raw bools stay changed
// for Delay, so brief device grabs do not flicker outputs.
type Debouncer struct {
	Delay        time.Duration
	current      Snapshot
	pending      *Snapshot
	pendingSince time.Time
	ready        bool
}

// Observe applies one raw reading. The published state is unchanged until
// Delay elapses with a stable difference in the three bools.
func (d *Debouncer) Observe(raw Snapshot, now time.Time) Result {
	if d.Delay < 0 {
		d.Delay = 0
	}
	raw.Sources = uniqueSorted(raw.Sources)
	if raw.Sources == nil {
		raw.Sources = []string{}
	}

	if !d.ready {
		return d.commitPending(raw, now)
	}

	if EqualBools(raw, d.current) {
		d.pending = nil
		if !slices.Equal(raw.Sources, d.current.Sources) {
			d.current.Sources = raw.Sources
			d.current.UpdatedAt = now.UTC()
			return Result{State: d.current, Changed: true, BoolsChanged: false}
		}
		return Result{State: d.current}
	}

	return d.commitPending(raw, now)
}

func (d *Debouncer) commitPending(raw Snapshot, now time.Time) Result {
	if d.pending == nil || !EqualBools(*d.pending, raw) {
		p := raw
		d.pending = &p
		d.pendingSince = now
		return Result{State: d.current}
	}
	if now.Sub(d.pendingSince) < d.Delay {
		return Result{State: d.current}
	}
	d.ready = true
	d.current = raw
	d.current.UpdatedAt = now.UTC()
	d.pending = nil
	return Result{State: d.current, Changed: true, BoolsChanged: true}
}
