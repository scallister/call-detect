//go:build linux

package live

import (
	"fmt"

	"github.com/scallister/call-detect/internal/detect"
)

func listAudio() detect.Audio {
	if names, err := pulseSourceOutputs(); err == nil {
		return detect.Audio{Capture: names, Authoritative: true}
	}
	if names, err := pipewireClassNames("Stream/Input/Audio"); err == nil {
		return detect.Audio{Capture: names, Authoritative: true}
	}
	return detect.Audio{
		Err:           fmt.Errorf("no PulseAudio (pactl) or PipeWire (pw-dump) capture list"),
		Authoritative: true,
	}
}
