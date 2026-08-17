package status

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scallister/call-detect/internal/state"
)

func TestWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "status.json")
	s := state.Snapshot{
		Busy:       true,
		Microphone: true,
		Sources:    []string{"Discord.exe"},
		UpdatedAt:  time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC),
	}
	if err := Write(path, s); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got state.Snapshot
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if !got.Busy || !got.Microphone || got.Webcam || got.Sources[0] != "Discord.exe" {
		t.Fatalf("got %+v", got)
	}
}
