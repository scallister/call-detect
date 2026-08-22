package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseVersion(t *testing.T) {
	t.Parallel()
	maj, min, pat, bld := parseVersion("v0.1.0")
	if maj != 0 || min != 1 || pat != 0 || bld != 0 {
		t.Fatalf("v0.1.0 -> %d.%d.%d.%d", maj, min, pat, bld)
	}
	maj, min, pat, bld = parseVersion("dev")
	if maj != 0 || min != 0 || pat != 0 || bld != 0 {
		t.Fatalf("dev -> %d.%d.%d.%d", maj, min, pat, bld)
	}
	maj, min, pat, bld = parseVersion("1.2.3.4")
	if maj != 1 || min != 2 || pat != 3 || bld != 4 {
		t.Fatalf("1.2.3.4 -> %d.%d.%d.%d", maj, min, pat, bld)
	}
}

func TestWriteSyso(t *testing.T) {
	t.Parallel()
	out := filepath.Join(t.TempDir(), "rsrc_windows_amd64.syso")
	if err := writeSyso("v0.1.0", out); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(out)
	if err != nil || st.Size() == 0 {
		t.Fatalf("syso: %v %+v", err, st)
	}
}
