//go:build !windows

package camera

import (
	"fmt"

	"github.com/scallister/call-detect/internal/detect"
)

func listStreaming() detect.Camera {
	return detect.Camera{Err: fmt.Errorf("camera monitor is only available on Windows")}
}
