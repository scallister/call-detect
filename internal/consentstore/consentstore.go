// Package consentstore reads Windows Capability Access Manager usage records
// for the microphone and webcam.
package consentstore

import (
	"path/filepath"
	"strings"
	"time"
)

const (
	CapabilityMicrophone = "microphone"
	CapabilityWebcam     = "webcam"
)

// Entry is one ConsentStore app key and its last-used FILETIME values.
type Entry struct {
	KeyName string
	Start   uint64
	Stop    uint64
}

// Usage is a parsed ConsentStore entry.
type Usage struct {
	App   string
	Key   string
	Start time.Time
	Stop  time.Time
	InUse bool
}

// Store lists ConsentStore entries for a capability (microphone or webcam).
type Store interface {
	List(capability string) ([]Entry, error)
}

// Memory is an in-memory Store for tests.
type Memory struct {
	Entries map[string][]Entry
}

// List returns recorded entries for capability, or nil if none.
func (m *Memory) List(capability string) ([]Entry, error) {
	if m == nil || m.Entries == nil {
		return nil, nil
	}
	return m.Entries[capability], nil
}

// InUse reports whether the last session is still open.
// Windows leaves LastUsedTimeStop at 0 while the device is held.
func InUse(start, stop uint64) bool {
	return start != 0 && (stop == 0 || start > stop)
}

const (
	filetimeTicksPerSecond = 10_000_000
	// Seconds between 1601-01-01 and the Unix epoch.
	filetimeUnixEpochDiff = 11_644_473_600
)

// FiletimeToTime converts a Windows FILETIME (100ns since 1601-01-01 UTC).
func FiletimeToTime(ft uint64) time.Time {
	if ft == 0 {
		return time.Time{}
	}
	sec := int64(ft/filetimeTicksPerSecond) - filetimeUnixEpochDiff
	nsec := int64(ft%filetimeTicksPerSecond) * 100
	return time.Unix(sec, nsec).UTC()
}

// DisplayName turns a ConsentStore key into a short app label.
// NonPackaged keys use # in place of path separators.
func DisplayName(keyName string) string {
	keyName = strings.TrimSpace(keyName)
	if keyName == "" {
		return ""
	}
	if strings.Contains(keyName, "#") {
		parts := strings.Split(keyName, "#")
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] != "" {
				return parts[i]
			}
		}
	}
	if base := filepath.Base(keyName); base != "." && base != "/" && base != "" {
		return base
	}
	return keyName
}

// ParseUsage maps a raw registry entry to Usage.
func ParseUsage(e Entry) Usage {
	return Usage{
		App:   DisplayName(e.KeyName),
		Key:   e.KeyName,
		Start: FiletimeToTime(e.Start),
		Stop:  FiletimeToTime(e.Stop),
		InUse: InUse(e.Start, e.Stop),
	}
}

// ParseAll converts entries, skipping empty key names.
func ParseAll(entries []Entry) []Usage {
	out := make([]Usage, 0, len(entries))
	for _, e := range entries {
		if strings.TrimSpace(e.KeyName) == "" {
			continue
		}
		out = append(out, ParseUsage(e))
	}
	return out
}

// InUseApps returns display names of entries that currently hold the device.
func InUseApps(usages []Usage) []string {
	seen := make(map[string]struct{})
	var names []string
	for _, u := range usages {
		if !u.InUse {
			continue
		}
		name := u.App
		if name == "" {
			name = u.Key
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

// AnyInUse reports whether any usage is currently in use.
func AnyInUse(usages []Usage) bool {
	for _, u := range usages {
		if u.InUse {
			return true
		}
	}
	return false
}
