//go:build linux

package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxDesktopAutostart(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	exe := filepath.Join(dir, "bin", "call-detect")
	if AutostartEnabled() {
		t.Fatal("expected no desktop entry")
	}
	if err := EnableAutostart(exe); err != nil {
		t.Fatal(err)
	}
	if !AutostartEnabled() {
		t.Fatal("expected desktop entry")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "autostart", "call-detect.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if !strings.Contains(body, "Exec="+exe) || !strings.Contains(body, "Type=Application") {
		t.Fatalf("desktop: %s", body)
	}
	if err := DisableAutostart(); err != nil {
		t.Fatal(err)
	}
	if AutostartEnabled() {
		t.Fatal("expected removed")
	}
}
