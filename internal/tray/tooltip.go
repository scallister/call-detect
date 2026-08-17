package tray

import (
	"strings"

	"github.com/scallister/call-detect/internal/state"
)

// Tooltip is the notification-area hover text.
func Tooltip(s state.Snapshot) string {
	if !s.Busy {
		return "call-detect: idle"
	}
	var bits []string
	switch {
	case s.Microphone && s.Webcam:
		bits = append(bits, "mic+camera")
	case s.Microphone:
		bits = append(bits, "mic")
	case s.Webcam:
		bits = append(bits, "camera")
	}
	if len(s.Sources) > 0 {
		bits = append(bits, s.Sources[0])
	}
	if len(bits) == 0 {
		return "call-detect: on a call"
	}
	return "call-detect: on a call (" + strings.Join(bits, ", ") + ")"
}

// SnapshotIdle is a helper for tests and initial tray state.
func SnapshotIdle() state.Snapshot {
	return state.Snapshot{Sources: []string{}}
}
