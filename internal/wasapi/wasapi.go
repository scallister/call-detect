// Package wasapi lists processes with active Windows audio sessions.
package wasapi

import "github.com/scallister/call-detect/internal/detect"

// Source reads live capture and render sessions.
type Source struct{}

// Sessions returns active session image names. On Windows a successful
// enumeration is Authoritative. On non-Windows, Err is set.
func (Source) Sessions() detect.Audio {
	return listSessions()
}

// sessionProcessName labels an active non-system capture session.
// A live session whose process image cannot be read still counts as "unknown".
func sessionProcessName(name string, err error) string {
	if err != nil || name == "" {
		return "unknown"
	}
	return name
}
