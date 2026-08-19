// Package detect combines ConsentStore records with live audio and camera activity.
package detect

import (
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/scallister/call-detect/internal/consentstore"
	"github.com/scallister/call-detect/internal/state"
)

// Audio is a live session snapshot. Err set means the enumerator failed.
type Audio struct {
	Capture []string
	Render  []string
	Err     error
	// Authoritative means Capture is ground truth and is not intersected
	// with ConsentStore (macOS and Linux). Windows leaves this false so
	// stale ConsentStore rows still need a live capture session.
	Authoritative bool
}

// Camera is a live camera-streaming snapshot from the Windows sensor activity
// monitor. Err set means the monitor failed and Confirm should fall back.
type Camera struct {
	Streaming []string
	Err       error
}

// Confirm builds a snapshot.
//
// Microphone: ConsentStore in-use apps that also have an active capture
// session, unless Audio.Authoritative is set (then Capture is used alone).
// If audio enumeration failed, ConsentStore is used alone.
//
// Webcam: processes currently streaming a camera (sensor activity monitor).
// That is independent of audio, so a browser preview with no microphone still
// counts. If the camera monitor failed, webcam falls back to ConsentStore
// intersected with a capture or render session (or ConsentStore alone if audio
// also failed).
//
// If both live sources failed, Confirm uses ConsentStore only.
func Confirm(mic, cam []consentstore.Usage, audio Audio, camera Camera, now time.Time) state.Snapshot {
	if audio.Err != nil && camera.Err != nil {
		return state.FromUsages(mic, cam, now)
	}

	micApps := confirmMic(mic, audio)
	camApps := confirmCam(cam, audio, camera)
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

func confirmMic(mic []consentstore.Usage, audio Audio) []string {
	if audio.Err != nil {
		return consentstore.InUseApps(mic)
	}
	if audio.Authoritative {
		return unique(append([]string{}, audio.Capture...))
	}
	capture := indexApps(audio.Capture)
	var apps []string
	for _, name := range consentstore.InUseApps(mic) {
		if capture[normApp(name)] {
			apps = append(apps, name)
		}
	}
	return apps
}

func confirmCam(cam []consentstore.Usage, audio Audio, camera Camera) []string {
	if camera.Err == nil {
		return unique(append([]string{}, camera.Streaming...))
	}
	if audio.Err != nil {
		return consentstore.InUseApps(cam)
	}
	capture := indexApps(audio.Capture)
	render := indexApps(audio.Render)
	var apps []string
	for _, name := range consentstore.InUseApps(cam) {
		key := normApp(name)
		if capture[key] || render[key] {
			apps = append(apps, name)
		}
	}
	return apps
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
