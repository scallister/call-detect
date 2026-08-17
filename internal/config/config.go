// Package config loads optional settings from flags, the environment, and a file.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// File is the on-disk YAML document.
type File struct {
	WebhookURL string `yaml:"webhook_url"`
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
# busy, microphone, or webcam changes (after a short debounce).
#
# webhook_url: "http://homeassistant.local:8123/api/webhook/YOUR_WEBHOOK_ID"
`

// WriteSample writes SampleYAML if path does not already exist.
func WriteSample(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(path, []byte(SampleYAML), 0o644)
}
