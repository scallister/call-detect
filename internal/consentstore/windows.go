//go:build windows

package consentstore

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const consentRoot = `Software\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore`

// Windows reads HKCU ConsentStore keys for the logged-in user.
type Windows struct{}

// List returns packaged and NonPackaged entries for a capability.
func (Windows) List(capability string) ([]Entry, error) {
	path := consentRoot + `\` + capability
	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.READ)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil, nil
		}
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer key.Close()

	var out []Entry
	names, err := key.ReadSubKeyNames(-1)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", path, err)
	}
	for _, name := range names {
		if strings.EqualFold(name, "NonPackaged") {
			nested, err := listChildren(key, name)
			if err != nil {
				return nil, err
			}
			out = append(out, nested...)
			continue
		}
		e, ok, err := readEntry(key, name)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, e)
		}
	}
	return out, nil
}

func listChildren(parent registry.Key, name string) ([]Entry, error) {
	key, err := registry.OpenKey(parent, name, registry.READ)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil, nil
		}
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	defer key.Close()

	names, err := key.ReadSubKeyNames(-1)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", name, err)
	}
	var out []Entry
	for _, child := range names {
		e, ok, err := readEntry(key, child)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, e)
		}
	}
	return out, nil
}

func readEntry(parent registry.Key, name string) (Entry, bool, error) {
	key, err := registry.OpenKey(parent, name, registry.READ)
	if err != nil {
		if err == registry.ErrNotExist {
			return Entry{}, false, nil
		}
		return Entry{}, false, fmt.Errorf("open %s: %w", name, err)
	}
	defer key.Close()

	start, _, err := key.GetIntegerValue("LastUsedTimeStart")
	if err == registry.ErrNotExist {
		return Entry{}, false, nil
	}
	if err != nil {
		return Entry{}, false, fmt.Errorf("%s LastUsedTimeStart: %w", name, err)
	}
	stop, _, err := key.GetIntegerValue("LastUsedTimeStop")
	if err == registry.ErrNotExist {
		stop = 0
		err = nil
	}
	if err != nil {
		return Entry{}, false, fmt.Errorf("%s LastUsedTimeStop: %w", name, err)
	}
	return Entry{KeyName: name, Start: start, Stop: stop}, true, nil
}
