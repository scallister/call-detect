// Package camera lists processes currently streaming a Windows camera.
package camera

import "github.com/scallister/call-detect/internal/detect"

// Source reads live camera-streaming processes from the sensor activity monitor.
type Source struct{}

// Streaming returns image names of processes with a live camera stream.
// On non-Windows, or if the monitor cannot start, Err is set.
func (Source) Streaming() detect.Camera {
	return listStreaming()
}
