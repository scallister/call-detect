package appdir

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPaths(t *testing.T) {
	t.Setenv("LOCALAPPDATA", `C:\Users\sam\AppData\Local`)
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
	if filepath.Base(ExePath(dir)) != "call-detect.exe" {
		t.Fatal(ExePath(dir))
	}
	if filepath.Base(VersionPath(dir)) != "version" {
		t.Fatal(VersionPath(dir))
	}
}
