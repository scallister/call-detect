// Package wasapi lists processes with active Windows audio sessions.
package wasapi

import "github.com/scallister/call-detect/internal/detect"

// Source reads live capture and render sessions.
type Source struct{}

// Sessions returns active session image names. On non-Windows, Err is set.
func (Source) Sessions() detect.Audio {
	return listSessions()
}
