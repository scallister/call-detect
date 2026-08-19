//go:build darwin

package live

import "github.com/scallister/call-detect/internal/detect"

func listCamera() detect.Camera {
	names, err := runningCameraNames()
	if err != nil {
		return detect.Camera{Err: err}
	}
	return detect.Camera{Streaming: names}
}
