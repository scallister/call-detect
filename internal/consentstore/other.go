//go:build !windows

package consentstore

import "fmt"

// Windows is a stub on non-Windows systems.
type Windows struct{}

// List always returns an error: ConsentStore exists only on Windows.
func (Windows) List(capability string) ([]Entry, error) {
	return nil, fmt.Errorf("consent store %q is only available on Windows", capability)
}
