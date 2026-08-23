package project

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestURLs(t *testing.T) {
	t.Parallel()
	if !strings.HasPrefix(RepoURL, "https://github.com/scallister/call-detect") {
		t.Fatalf("RepoURL %s", RepoURL)
	}
	if LatestReleaseURL != RepoURL+"/releases/latest" {
		t.Fatalf("LatestReleaseURL %s", LatestReleaseURL)
	}
	if LatestExeURL != LatestReleaseURL+"/download/call-detect.exe" {
		t.Fatalf("LatestExeURL %s", LatestExeURL)
	}
	if LatestAssetURL() != LatestReleaseURL+"/download/"+AssetName(runtime.GOOS, runtime.GOARCH) {
		t.Fatalf("LatestAssetURL %s", LatestAssetURL())
	}
}

func TestLatestTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing User-Agent")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v0.1.0"}`))
	}))
	t.Cleanup(srv.Close)

	prev := LatestAPI
	LatestAPI = srv.URL
	t.Cleanup(func() { LatestAPI = prev })

	tag, err := LatestTag(context.Background())
	if err != nil || tag != "v0.1.0" {
		t.Fatalf("got %q %v", tag, err)
	}
}

func TestAssetName(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"windows/amd64": "call-detect.exe",
		"windows/arm64": "call-detect.exe",
		"linux/amd64":   "call-detect-linux-amd64",
		"linux/arm64":   "call-detect-linux-arm64",
		"darwin/amd64":  "call-detect-darwin-amd64",
		"darwin/arm64":  "call-detect-darwin-arm64",
	}
	for pair, want := range cases {
		goos, goarch, _ := strings.Cut(pair, "/")
		if got := AssetName(goos, goarch); got != want {
			t.Fatalf("%s: %s", pair, got)
		}
	}
}

func TestDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing User-Agent")
		}
		_, _ = w.Write([]byte("binary"))
	}))
	t.Cleanup(srv.Close)
	prev := AssetURL
	AssetURL = srv.URL
	t.Cleanup(func() { AssetURL = prev })

	dest := filepath.Join(t.TempDir(), "call-detect.new")
	if err := Download(context.Background(), dest); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(dest)
	if err != nil || string(raw) != "binary" {
		t.Fatalf("got %q %v", raw, err)
	}
}

func TestLatestTagError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)

	prev := LatestAPI
	LatestAPI = srv.URL
	t.Cleanup(func() { LatestAPI = prev })

	if _, err := LatestTag(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
