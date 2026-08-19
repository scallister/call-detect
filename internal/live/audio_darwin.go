//go:build darwin

package live

import "github.com/scallister/call-detect/internal/detect"

func listAudio() detect.Audio {
	names, err := runningInputNames()
	if err != nil {
		return detect.Audio{Err: err, Authoritative: true}
	}
	return detect.Audio{Capture: names, Authoritative: true}
}
