package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolvePrecedence(t *testing.T) {
	t.Parallel()
	file := File{WebhookURL: "http://file.example/hook"}
	if got := Resolve("http://flag.example/hook", "http://env.example/hook", file); got.WebhookURL != "http://flag.example/hook" {
		t.Fatalf("flag: %s", got.WebhookURL)
	}
	if got := Resolve("", "http://env.example/hook", file); got.WebhookURL != "http://env.example/hook" {
		t.Fatalf("env: %s", got.WebhookURL)
	}
	if got := Resolve("  ", "", file); got.WebhookURL != "http://file.example/hook" {
		t.Fatalf("file: %s", got.WebhookURL)
	}
	if got := Resolve("", "", File{}); got.WebhookURL != "" {
		t.Fatalf("empty: %s", got.WebhookURL)
	}
}

func TestLoadFileAndSample(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.yaml")
	f, err := LoadFile(missing)
	if err != nil || f.WebhookURL != "" {
		t.Fatalf("missing: %+v %v", f, err)
	}

	path := filepath.Join(dir, "config.yaml")
	if err := WriteSample(path); err != nil {
		t.Fatal(err)
	}
	if err := WriteSample(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != SampleYAML {
		t.Fatalf("sample changed")
	}

	if err := os.WriteFile(path, []byte("webhook_url: http://ha.example/api/webhook/abc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFile(path)
	if err != nil || got.WebhookURL != "http://ha.example/api/webhook/abc" {
		t.Fatalf("load: %+v %v", got, err)
	}
}

func TestWriteWebhook(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := WriteWebhook(path, "http://homeassistant.local:8123/api/webhook/abc"); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFile(path)
	if err != nil || got.WebhookURL != "http://homeassistant.local:8123/api/webhook/abc" {
		t.Fatalf("set: %+v %v", got, err)
	}
	if err := WriteWebhook(path, "  "); err != nil {
		t.Fatal(err)
	}
	got, err = LoadFile(path)
	if err != nil || got.WebhookURL != "" {
		t.Fatalf("clear: %+v %v", got, err)
	}
}

func TestWriteUpdatePreservesWebhook(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := WriteWebhook(path, "http://ha.example/hook"); err != nil {
		t.Fatal(err)
	}
	if err := WriteUpdate(path, "2026-08-23", "v0.1.0"); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.WebhookURL != "http://ha.example/hook" || got.UpdateLastChecked != "2026-08-23" || got.UpdateAvailable != "v0.1.0" {
		t.Fatalf("update write: %+v", got)
	}
	if err := WriteWebhook(path, "http://ha.example/other"); err != nil {
		t.Fatal(err)
	}
	got, err = LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.WebhookURL != "http://ha.example/other" || got.UpdateLastChecked != "2026-08-23" || got.UpdateAvailable != "v0.1.0" {
		t.Fatalf("webhook write kept update: %+v", got)
	}
	if err := WriteUpdate(path, "2026-09-01", ""); err != nil {
		t.Fatal(err)
	}
	got, err = LoadFile(path)
	if err != nil || got.WebhookURL != "http://ha.example/other" || got.UpdateLastChecked != "2026-09-01" || got.UpdateAvailable != "" {
		t.Fatalf("clear available: %+v %v", got, err)
	}
}

func TestDue(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 23, 15, 0, 0, 0, time.UTC)
	if !Due("", now) {
		t.Fatal("empty is due")
	}
	if !Due("not-a-date", now) {
		t.Fatal("bad date is due")
	}
	if Due("2026-08-23", now) {
		t.Fatal("same day is not due")
	}
	if Due("2026-07-25", now) {
		t.Fatal("29 days is not due")
	}
	if !Due("2026-07-24", now) {
		t.Fatal("30 days is due")
	}
	if FormatDate(now) != "2026-08-23" {
		t.Fatalf("format %s", FormatDate(now))
	}
}
