//go:build !windows

package install

import "fmt"

// AutostartEnabled is always false off Windows.
func AutostartEnabled() bool { return false }

// EnableAutostart is only implemented on Windows.
func EnableAutostart(exe string) error {
	return fmt.Errorf("autostart is only available on Windows")
}

// DisableAutostart is only implemented on Windows.
func DisableAutostart() error {
	return fmt.Errorf("autostart is only available on Windows")
}
