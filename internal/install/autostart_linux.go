//go:build linux

package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func desktopPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "autostart", "call-detect.desktop"), nil
}

// EnableAutostart writes an XDG autostart desktop entry for exe.
func EnableAutostart(exe string) error {
	path, err := desktopPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body := "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=call-detect\n" +
		"Comment=Detect microphone and webcam use\n" +
		"Exec=" + desktopExec(exe) + "\n" +
		"X-GNOME-Autostart-enabled=true\n" +
		"Hidden=false\n"
	return os.WriteFile(path, []byte(body), 0o644)
}

func desktopExec(exe string) string {
	if strings.ContainsAny(exe, " \t") {
		return `"` + strings.ReplaceAll(exe, `"`, `\"`) + `"`
	}
	return exe
}

// AutostartEnabled reports whether the XDG autostart entry exists.
func AutostartEnabled() bool {
	path, err := desktopPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// DisableAutostart removes the XDG autostart entry.
func DisableAutostart() error {
	path, err := desktopPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove autostart: %w", err)
	}
	return nil
}
