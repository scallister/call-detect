package camera

import "strings"

// Device-interface class GUIDs that mark a camera (ks.h / ksmedia.h),
// stored lowercase without braces for substring match on symbolic links.
var cameraInterfaceIDs = []string{
	"e5323777-f976-4f5b-9b55-b94699c46e44", // KSCATEGORY_VIDEO_CAMERA
	"6994ad05-93ef-11d0-a3cc-00a0c9223196", // KSCATEGORY_VIDEO
	"24e552d7-6523-47f7-a647-d3465bf1f5ca", // KSCATEGORY_SENSOR_CAMERA
}

const audioCategoryID = "6994ad04-93ef-11d0-a3cc-00a0c9223196" // KSCATEGORY_AUDIO

var cameraNameHints = []string{
	"camera",
	"webcam",
	"usb video",
	"video device",
	"uvc",
}

func looksLikeCamera(friendly, symlink string) bool {
	sl := strings.ToLower(symlink)
	for _, id := range cameraInterfaceIDs {
		if strings.Contains(sl, id) {
			return true
		}
	}
	f := strings.ToLower(friendly)
	for _, hint := range cameraNameHints {
		if strings.Contains(f, hint) {
			return true
		}
	}
	return false
}

// isMicrophoneSensor reports sensors that are clearly audio, so a voice-only
// call does not flip the webcam bit. Unknown sensors are treated as cameras
// (fail open) so an unusual webcam still counts.
func isMicrophoneSensor(friendly, symlink string) bool {
	if looksLikeCamera(friendly, symlink) {
		return false
	}
	if strings.Contains(strings.ToLower(symlink), audioCategoryID) {
		return true
	}
	f := strings.ToLower(strings.TrimSpace(friendly))
	if strings.Contains(f, "microphone") {
		return true
	}
	return strings.Contains(f, "mic ") || strings.HasSuffix(f, " mic") || f == "mic"
}
