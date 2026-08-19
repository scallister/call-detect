//go:build linux

package live

import "github.com/scallister/call-detect/internal/detect"

func listCamera() detect.Camera {
	pw, err := pipewireClassNames("Stream/Input/Video")
	if err == nil && len(pw) > 0 {
		return detect.Camera{Streaming: pw}
	}
	return detect.Camera{Streaming: filterHelperNames(v4lStreaming("", ""))}
}
