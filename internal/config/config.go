// Package config loads optional settings from flags, the environment, and a file.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// CheckInterval is the minimum time between silent GitHub update checks.
const CheckInterval = 30 * 24 * time.Hour

// DateLayout is the on-disk form of update_last_checked.
const DateLayout = "2006-01-02"

// File is the on-disk YAML document.
type File struct {
	WebhookURL        string `yaml:"webhook_url"`
	UpdateLastChecked string `yaml:"update_last_checked"`
	UpdateAvailable   string `yaml:"update_available"`
}

// Values is the resolved runtime config.
type Values struct {
	WebhookURL string
	ConfigPath string
}

// LoadFile reads path. A missing file is an empty config.
func LoadFile(path string) (File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return File{}, nil
		}
		return File{}, fmt.Errorf("read config: %w", err)
	}
	var f File
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return File{}, fmt.Errorf("parse config: %w", err)
	}
	f.WebhookURL = strings.TrimSpace(f.WebhookURL)
	f.UpdateLastChecked = strings.TrimSpace(f.UpdateLastChecked)
	f.UpdateAvailable = strings.TrimSpace(f.UpdateAvailable)
	return f, nil
}

// Resolve picks webhook URL in order: flag, CALL_DETECT_WEBHOOK_URL, file.
func Resolve(flagURL, envURL string, file File) Values {
	url := strings.TrimSpace(flagURL)
	if url == "" {
		url = strings.TrimSpace(envURL)
	}
	if url == "" {
		url = strings.TrimSpace(file.WebhookURL)
	}
	return Values{WebhookURL: url}
}

// SampleYAML is written on first install when no config exists.
const SampleYAML = `# Optional webhook. When set, call-detect POSTs JSON whenever
# call, microphone, or webcam changes (after a short debounce).
#
# webhook_url: "http://homeassistant.local:8123/api/webhook/YOUR_WEBHOOK_ID"
`

// Due reports whether a silent update check should run.
func Due(lastChecked string, now time.Time) bool {
	s := strings.TrimSpace(lastChecked)
	if s == "" {
		return true
	}
	t, err := time.ParseInLocation(DateLayout, s, now.Location())
	if err != nil {
		return true
	}
	return !now.Before(t.Add(CheckInterval))
}

// FormatDate writes now as YYYY-MM-DD.
func FormatDate(now time.Time) string {
	return now.Format(DateLayout)
}

// WriteWebhook writes path with webhook_url set (or commented out if empty).
// Existing update fields are preserved.
func WriteWebhook(path, url string) error {
	f, err := LoadFile(path)
	if err != nil {
		return err
	}
	f.WebhookURL = url
	return writeFile(path, f)
}

// WriteUpdate records the last check date and any newer release tag.
// Existing webhook_url is preserved.
func WriteUpdate(path, lastChecked, available string) error {
	f, err := LoadFile(path)
	if err != nil {
		return err
	}
	f.UpdateLastChecked = lastChecked
	f.UpdateAvailable = available
	return writeFile(path, f)
}

func writeFile(path string, f File) error {
	f.WebhookURL = strings.TrimSpace(f.WebhookURL)
	f.UpdateLastChecked = strings.TrimSpace(f.UpdateLastChecked)
	f.UpdateAvailable = strings.TrimSpace(f.UpdateAvailable)

	var b strings.Builder
	b.WriteString(`# Optional webhook. When set, call-detect POSTs JSON whenever
# call, microphone, or webcam changes (after a short debounce).
`)
	if f.WebhookURL == "" {
		b.WriteString(`# webhook_url: "http://homeassistant.local:8123/api/webhook/YOUR_WEBHOOK_ID"
`)
	} else {
		enc, err := yaml.Marshal(map[string]string{"webhook_url": f.WebhookURL})
		if err != nil {
			return err
		}
		b.Write(enc)
	}
	extra := struct {
		UpdateLastChecked string `yaml:"update_last_checked,omitempty"`
		UpdateAvailable   string `yaml:"update_available,omitempty"`
	}{f.UpdateLastChecked, f.UpdateAvailable}
	if extra.UpdateLastChecked != "" || extra.UpdateAvailable != "" {
		enc, err := yaml.Marshal(extra)
		if err != nil {
			return err
		}
		b.Write(enc)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// WriteSample writes SampleYAML if path does not already exist.
func WriteSample(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(path, []byte(SampleYAML), 0o644)
}
