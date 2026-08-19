package live

import (
	"path/filepath"
	"strings"
)

func cleanApp(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, `\`, "/")
	return filepath.Base(s)
}

func uniqueNames(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = cleanApp(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func filterHelperNames(names []string) []string {
	skip := map[string]struct{}{
		"pipewire":       {},
		"wireplumber":    {},
		"pipewire-pulse": {},
	}
	var out []string
	for _, n := range names {
		if _, ok := skip[strings.ToLower(n)]; ok {
			continue
		}
		out = append(out, n)
	}
	return out
}

func firstProp(props map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(props[k]); v != "" {
			return v
		}
	}
	return ""
}
