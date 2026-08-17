package camera

import "testing"

func TestLooksLikeCamera(t *testing.T) {
	t.Parallel()
	cases := []struct {
		friendly, symlink string
		want              bool
	}{
		{"HD Webcam", "", true},
		{"Integrated Camera", "", true},
		{"USB Video Device", "", true},
		{"OBS Virtual Camera", "", true},
		{"", `\\?\USB#VID_1234#{e5323777-f976-4f5b-9b55-b94699c46e44}\GLOBAL`, true},
		{"Microphone Array", "", false},
		{"", `\\?\USB#VID_1234#{6994ad04-93ef-11d0-a3cc-00a0c9223196}\GLOBAL`, false},
		{"", "", false},
	}
	for _, tc := range cases {
		if got := looksLikeCamera(tc.friendly, tc.symlink); got != tc.want {
			t.Fatalf("looksLikeCamera(%q, %q)=%v want %v", tc.friendly, tc.symlink, got, tc.want)
		}
	}
}

func TestIsMicrophoneSensor(t *testing.T) {
	t.Parallel()
	if !isMicrophoneSensor("Microphone Array", "") {
		t.Fatal("expected microphone")
	}
	if isMicrophoneSensor("HD Webcam", "") {
		t.Fatal("webcam is not a microphone")
	}
	if isMicrophoneSensor("IR Camera", `\\?\USB#{6994ad04-93ef-11d0-a3cc-00a0c9223196}`) {
		t.Fatal("camera name wins over audio GUID")
	}
	if isMicrophoneSensor("Contoso Capture", "") {
		t.Fatal("unknown non-mic sensor should fail open as a camera")
	}
}
