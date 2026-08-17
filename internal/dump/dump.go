// Package dump formats a ConsentStore snapshot for --dump.
package dump

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/scallister/call-detect/internal/consentstore"
	"github.com/scallister/call-detect/internal/detect"
	"github.com/scallister/call-detect/internal/state"
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

// WriteAudio prints live WASAPI sessions.
func WriteAudio(w io.Writer, audio detect.Audio) error {
	if _, err := fmt.Fprintln(w, "\nActive audio sessions"); err != nil {
		return err
	}
	if audio.Err != nil {
		_, err := fmt.Fprintf(w, "  (unavailable: %v)\n", audio.Err)
		return err
	}
	if err := writeNames(w, "  capture", audio.Capture); err != nil {
		return err
	}
	return writeNames(w, "  render ", audio.Render)
}

// WriteCamera prints processes currently streaming a camera.
func WriteCamera(w io.Writer, cam detect.Camera) error {
	if _, err := fmt.Fprintln(w, "\nStreaming cameras"); err != nil {
		return err
	}
	if cam.Err != nil {
		_, err := fmt.Fprintf(w, "  (unavailable: %v)\n", cam.Err)
		return err
	}
	return writeNames(w, "  processes", cam.Streaming)
}

// WriteResult prints the confirmed snapshot.
func WriteResult(w io.Writer, snap state.Snapshot) error {
	_, err := fmt.Fprintf(w, "\nResult  busy=%v  microphone=%v  webcam=%v  sources=%s\n",
		snap.Busy, snap.Microphone, snap.Webcam, formatSources(snap.Sources))
	return err
}

func writeNames(w io.Writer, label string, names []string) error {
	if len(names) == 0 {
		_, err := fmt.Fprintf(w, "%s  (none)\n", label)
		return err
	}
	_, err := fmt.Fprintf(w, "%s  %s\n", label, strings.Join(names, ", "))
	return err
}

func formatSources(names []string) string {
	if len(names) == 0 {
		return "(none)"
	}
	return strings.Join(names, ", ")
}
