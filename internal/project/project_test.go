package project

import (
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
