// Package detect combines ConsentStore records with live audio sessions.
package detect

import (
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/scallister/call-detect/internal/consentstore"
	"github.com/scallister/call-detect/internal/state"
)

// Audio is a live WASAPI session snapshot. Err set means the enumerator failed
// and Confirm should fall back to ConsentStore only.
type Audio struct {
	Capture []string
	Render  []string
	Err     error
}

// Confirm builds a snapshot. Apps that only appear in ConsentStore (common
// after Discord or a browser has held a device) are ignored unless they also
// have an active capture session. If audio enumeration failed, ConsentStore
// is used alone.
func Confirm(mic, cam []consentstore.Usage, audio Audio, now time.Time) state.Snapshot {
	if audio.Err != nil {
		return state.FromUsages(mic, cam, now)
	}
	capture := indexApps(audio.Capture)
	render := indexApps(audio.Render)

	var micApps, camApps []string
	for _, name := range consentstore.InUseApps(mic) {
		if capture[normApp(name)] {
			micApps = append(micApps, name)
		}
	}
	for _, name := range consentstore.InUseApps(cam) {
		key := normApp(name)
		if capture[key] || render[key] {
			camApps = append(camApps, name)
		}
	}

	sources := unique(append(append([]string{}, micApps...), camApps...))
	micOn := len(micApps) > 0
	camOn := len(camApps) > 0
	return state.Snapshot{
		Busy:       micOn || camOn,
		Microphone: micOn,
		Webcam:     camOn,
		Sources:    sources,
		UpdatedAt:  now.UTC(),
	}
}

func normApp(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, `\`, "/")
	if i := strings.LastIndex(name, "#"); i >= 0 {
		name = name[i+1:]
	}
	return strings.ToLower(filepath.Base(name))
}

func indexApps(names []string) map[string]bool {
	out := make(map[string]bool, len(names))
	for _, n := range names {
		if key := normApp(n); key != "" {
			out[key] = true
		}
	}
	return out
}

func unique(in []string) []string {
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
	if out == nil {
		return []string{}
	}
	slices.Sort(out)
	return out
}
