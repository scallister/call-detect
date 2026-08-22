package project

import (
	"context"
	"net/http"
	"net/http/httptest"
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
