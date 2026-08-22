// Package version holds the release string baked in at build time.
package version

import (
	"strconv"
	"strings"
)

// Version is set by -ldflags on release builds (for example v0.0.5).
var Version = "dev"

// Display returns a short label for UI text.
func Display(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "unknown"
	}
	return v
}

// Normalize strips a leading v and surrounding space.
func Normalize(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

func release(v string) bool {
	v = Normalize(v)
	return v != "" && v != "dev"
}

// IsRelease reports whether v looks like a tagged release (not empty or "dev").
func IsRelease(v string) bool {
	return release(v)
}

// Compare returns 1 if a is a newer release than b, -1 if older, 0 if equal
// or not comparable (dev / empty).
func Compare(a, b string) int {
	if !release(a) && !release(b) {
		return 0
	}
	if !release(a) {
		return 0
	}
	if !release(b) {
		return 1
	}
	ap := parts(Normalize(a))
	bp := parts(Normalize(b))
	n := len(ap)
	if len(bp) > n {
		n = len(bp)
	}
	for i := 0; i < n; i++ {
		av, bv := 0, 0
		if i < len(ap) {
			av = ap[i]
		}
		if i < len(bp) {
			bv = bp[i]
		}
		if av > bv {
			return 1
		}
		if av < bv {
			return -1
		}
	}
	return 0
}

// Newer reports whether a is a newer release than b.
func Newer(a, b string) bool {
	return Compare(a, b) > 0
}

func parts(v string) []int {
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	raw := strings.Split(v, ".")
	out := make([]int, 0, len(raw))
	for _, s := range raw {
		n, err := strconv.Atoi(s)
		if err != nil {
			break
		}
		out = append(out, n)
	}
	return out
}
