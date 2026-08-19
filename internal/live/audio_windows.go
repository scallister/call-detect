//go:build windows

package live

import (
	"github.com/scallister/call-detect/internal/detect"
	"github.com/scallister/call-detect/internal/wasapi"
)

func listAudio() detect.Audio {
	return wasapi.Source{}.Sessions()
}
