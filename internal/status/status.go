// Package status writes the latest snapshot to a local JSON file.
package status

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/scallister/call-detect/internal/state"
)

// Write atomically replaces path with the JSON snapshot.
func Write(path string, s state.Snapshot) error {
	if path == "" {
		return fmt.Errorf("status path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create status dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write status temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace status file: %w", err)
	}
	return nil
}
