package appdir

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPaths(t *testing.T) {
	switch runtime.GOOS {
	case "windows":
		t.Setenv("LOCALAPPDATA", `C:\Users\sam\AppData\Local`)
	case "darwin":
		t.Setenv("HOME", "/Users/sam")
	default:
		t.Setenv("XDG_DATA_HOME", "/home/sam/.local/share")
		t.Setenv("HOME", "/home/sam")
	}
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(dir, "call-detect") {
		t.Fatalf("dir %s", dir)
	}
	if filepath.Base(ConfigPath(dir)) != "config.yaml" {
		t.Fatal(ConfigPath(dir))
	}
	if filepath.Base(StatusPath(dir)) != "status.json" {
		t.Fatal(StatusPath(dir))
	}
	if filepath.Base(ExePath(dir)) != ExeName() {
		t.Fatal(ExePath(dir))
	}
	if runtime.GOOS == "windows" && ExeName() != "call-detect.exe" {
		t.Fatal(ExeName())
	}
	if runtime.GOOS != "windows" && ExeName() != "call-detect" {
		t.Fatal(ExeName())
	}
	if filepath.Base(VersionPath(dir)) != "version" {
		t.Fatal(VersionPath(dir))
	}
}
