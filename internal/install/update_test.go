package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOfferReason(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	installed := filepath.Join(dir, "call-detect.exe")
	self := filepath.Join(dir, "download", "call-detect.exe")
	if err := os.MkdirAll(filepath.Dir(self), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installed, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(self, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	if ok, _ := OfferReason(installed, installed, "v0.0.5", "v0.0.4"); ok {
		t.Fatal("same path should not offer")
	}
	if ok, _ := OfferReason(self, filepath.Join(dir, "missing.exe"), "v0.0.5", ""); ok {
		t.Fatal("missing install should not offer")
	}
	ok, msg := OfferReason(self, installed, "v0.0.5", "v0.0.4")
	if !ok || msg == "" {
		t.Fatalf("newer should offer: %v %q", ok, msg)
	}
	if ok, _ := OfferReason(self, installed, "v0.0.3", "v0.0.4"); ok {
		t.Fatal("older should not offer")
	}
	if ok, _ := OfferReason(self, installed, "v0.0.4", "v0.0.4"); ok {
		t.Fatal("equal should not offer")
	}
	ok, msg = OfferReason(self, installed, "dev", "")
	if !ok || msg == "" {
		t.Fatalf("unknown installed + different file should offer: %v %q", ok, msg)
	}
}

func TestWriteReadInstalledVersion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if got := ReadInstalledVersion(dir); got != "" {
		t.Fatalf("empty dir: %q", got)
	}
	if err := WriteInstalledVersion(dir, "v0.0.5"); err != nil {
		t.Fatal(err)
	}
	if got := ReadInstalledVersion(dir); got != "v0.0.5" {
		t.Fatalf("got %q", got)
	}
}

func TestFilesDiffer(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	if err := os.WriteFile(a, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if FilesDiffer(a, b) {
		t.Fatal("same bytes")
	}
	if err := os.WriteFile(b, []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !FilesDiffer(a, b) {
		t.Fatal("different bytes")
	}
}
