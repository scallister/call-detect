// Package appdir locates the per-user data directory.
package appdir

import (
	"fmt"
	"os"
	"path/filepath"
)

const dirName = "call-detect"

// Dir returns %LOCALAPPDATA%\call-detect on Windows, or the user cache dir elsewhere.
func Dir() (string, error) {
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		return filepath.Join(local, dirName), nil
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("user data dir: %w", err)
	}
	return filepath.Join(cache, dirName), nil
}

// Ensure creates the data directory if needed.
func Ensure() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	return dir, nil
}

// ConfigPath is the YAML config file inside Dir.
func ConfigPath(dir string) string {
	return filepath.Join(dir, "config.yaml")
}

// StatusPath is the JSON status file inside Dir.
func StatusPath(dir string) string {
	return filepath.Join(dir, "status.json")
}

// ExePath is the installed executable name inside Dir.
func ExePath(dir string) string {
	return filepath.Join(dir, "call-detect.exe")
}

// VersionPath is the installed release string inside Dir.
func VersionPath(dir string) string {
	return filepath.Join(dir, "version")
}
