//go:build windows

package live

import (
	"github.com/scallister/call-detect/internal/camera"
	"github.com/scallister/call-detect/internal/detect"
)

func listCamera() detect.Camera {
	return camera.Source{}.Streaming()
}
