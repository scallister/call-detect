//go:build !windows

package wasapi

import (
	"fmt"

	"github.com/scallister/call-detect/internal/detect"
)

func listSessions() detect.Audio {
	return detect.Audio{Err: fmt.Errorf("audio sessions are only available on Windows")}
}
