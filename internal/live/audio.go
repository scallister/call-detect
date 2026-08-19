// Package live lists processes that currently hold the microphone or camera.
package live

import "github.com/scallister/call-detect/internal/detect"

// Audio reads live capture sessions for the current OS.
type Audio struct{}

// Sessions returns active capture (and, on Windows, render) process names.
func (Audio) Sessions() detect.Audio {
	return listAudio()
}

// Camera reads processes currently streaming a camera.
type Camera struct{}

// Streaming returns process names with a live camera stream.
func (Camera) Streaming() detect.Camera {
	return listCamera()
}
