// Package dump formats a ConsentStore snapshot for --dump.
package dump

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/scallister/call-detect/internal/consentstore"
)

// Write prints microphone and webcam usage to w.
func Write(w io.Writer, mic, cam []consentstore.Usage) error {
	if _, err := fmt.Fprintln(w, "Microphone"); err != nil {
		return err
	}
	if err := writeUsages(w, mic); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "\nWebcam"); err != nil {
		return err
	}
	return writeUsages(w, cam)
}

func writeUsages(w io.Writer, usages []consentstore.Usage) error {
	if len(usages) == 0 {
		_, err := fmt.Fprintln(w, "  (no records)")
		return err
	}
	for _, u := range usages {
		flag := "idle"
		if u.InUse {
			flag = "IN USE"
		}
		line := fmt.Sprintf("  %-8s  %s", flag, u.App)
		if extra := formatTimes(u); extra != "" {
			line += "  " + extra
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

func formatTimes(u consentstore.Usage) string {
	var parts []string
	if !u.Start.IsZero() {
		parts = append(parts, "start="+u.Start.UTC().Format(time.RFC3339))
	}
	if !u.Stop.IsZero() {
		parts = append(parts, "stop="+u.Stop.UTC().Format(time.RFC3339))
	}
	return strings.Join(parts, " ")
}
