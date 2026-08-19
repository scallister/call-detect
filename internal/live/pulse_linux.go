//go:build linux

package live

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func pulseSourceOutputs() ([]string, error) {
	raw, err := runCmd("pactl", "--format=json", "list", "source-outputs")
	if err == nil {
		names, perr := parsePactlJSON(raw)
		if perr == nil {
			return names, nil
		}
	}
	raw, err = runCmd("pactl", "list", "source-outputs")
	if err != nil {
		return nil, fmt.Errorf("pactl: %w", err)
	}
	return parsePactlText(raw), nil
}

func parsePactlJSON(raw []byte) ([]string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '[' {
		return nil, fmt.Errorf("pactl json: not an array")
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	var names []string
	for _, row := range rows {
		props := stringMap(row["properties"])
		if name := firstProp(props, "application.process.binary", "application.name", "media.name"); name != "" {
			names = append(names, name)
		}
	}
	return uniqueNames(names), nil
}

func parsePactlText(raw []byte) []string {
	var names []string
	var current map[string]string
	flush := func() {
		if current == nil {
			return
		}
		if name := firstProp(current, "application.process.binary", "application.name", "media.name"); name != "" {
			names = append(names, name)
		}
		current = nil
	}
	for _, line := range strings.Split(string(raw), "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "Source Output #") || strings.HasPrefix(trim, "Source Output ") {
			flush()
			current = map[string]string{}
			continue
		}
		if current == nil {
			continue
		}
		key, val, ok := strings.Cut(trim, "=")
		if !ok {
			key, val, ok = strings.Cut(trim, ":")
		}
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"`)
		current[key] = val
	}
	flush()
	return uniqueNames(names)
}

func stringMap(v any) map[string]string {
	out := map[string]string{}
	m, ok := v.(map[string]any)
	if !ok {
		return out
	}
	for k, raw := range m {
		if s, ok := raw.(string); ok {
			out[k] = s
		}
	}
	return out
}
