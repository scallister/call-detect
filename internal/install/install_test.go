package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/scallister/call-detect/internal/config"
)

func TestCopyExecutableAndSample(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dest := filepath.Join(dir, "call-detect.exe")
	if err := CopyExecutable(dest); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(dest)
	if err != nil || st.Size() == 0 {
		t.Fatalf("copied: %v %+v", err, st)
	}
	cfg := filepath.Join(dir, "config.yaml")
	if err := WriteSampleConfig(cfg); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(cfg)
	if err != nil || string(raw) != config.SampleYAML {
		t.Fatalf("sample: %s %v", raw, err)
	}
}
