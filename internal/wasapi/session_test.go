package wasapi

import (
	"errors"
	"testing"
)

func TestSessionProcessName(t *testing.T) {
	t.Parallel()
	if got := sessionProcessName("Discord.exe", nil); got != "Discord.exe" {
		t.Fatalf("named session: %q", got)
	}
	if got := sessionProcessName("", errors.New("OpenProcess: access denied")); got != "unknown" {
		t.Fatalf("image lookup failure: %q", got)
	}
	if got := sessionProcessName("", nil); got != "unknown" {
		t.Fatalf("empty image name: %q", got)
	}
}
