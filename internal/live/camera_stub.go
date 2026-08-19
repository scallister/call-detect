//go:build !windows && !linux && !darwin

package live

import (
	"fmt"
	"runtime"

	"github.com/scallister/call-detect/internal/detect"
)

func listCamera() detect.Camera {
	return detect.Camera{Err: fmt.Errorf("camera monitor is not supported on %s", runtime.GOOS)}
}
