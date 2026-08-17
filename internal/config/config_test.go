package config

import (
	"os"
	"path/filepath"
	"testing"
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
