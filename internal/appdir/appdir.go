// Package appdir locates the per-user data directory.
package appdir

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const dirName = "call-detect"

// Dir returns the per-user data directory for this OS.
func Dir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, dirName), nil
		}
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("user data dir: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", dirName), nil
	default:
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, dirName), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("user data dir: %w", err)
		}
		return filepath.Join(home, ".local", "share", dirName), nil
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

// ExeName is the installed binary name on this OS.
func ExeName() string {
	if runtime.GOOS == "windows" {
		return "call-detect.exe"
	}
	return "call-detect"
}

// ExePath is the installed executable path inside Dir.
func ExePath(dir string) string {
	return filepath.Join(dir, ExeName())
}

// VersionPath is the installed release string inside Dir.
func VersionPath(dir string) string {
	return filepath.Join(dir, "version")
}
