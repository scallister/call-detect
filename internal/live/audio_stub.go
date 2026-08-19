//go:build !windows && !linux && !darwin

package live

import (
	"fmt"
	"runtime"

	"github.com/scallister/call-detect/internal/detect"
)

func listAudio() detect.Audio {
	return detect.Audio{Err: fmt.Errorf("audio sessions are not supported on %s", runtime.GOOS), Authoritative: true}
}
