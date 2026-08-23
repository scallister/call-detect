// Package project holds public URLs for the GitHub repo and releases.
package project

import "runtime"

const (
	// RepoURL is the project homepage.
	RepoURL = "https://github.com/scallister/call-detect"
	// LatestReleaseURL is the GitHub page for the newest release.
	LatestReleaseURL = "https://github.com/scallister/call-detect/releases/latest"
	// LatestExeURL always downloads the newest call-detect.exe.
	LatestExeURL = "https://github.com/scallister/call-detect/releases/latest/download/call-detect.exe"
)

// AssetURL overrides LatestAssetURL in tests. Empty means the real GitHub URL.
var AssetURL string

// AssetName is the GitHub release filename for goos/goarch.
func AssetName(goos, goarch string) string {
	if goos == "windows" {
		return "call-detect.exe"
	}
	switch goos + "/" + goarch {
	case "linux/amd64":
		return "call-detect-linux-amd64"
	case "linux/arm64":
		return "call-detect-linux-arm64"
	case "darwin/amd64":
		return "call-detect-darwin-amd64"
	case "darwin/arm64":
		return "call-detect-darwin-arm64"
	default:
		return "call-detect-" + goos + "-" + goarch
	}
}

// LatestAssetURL is the download URL for this platform's newest release binary.
func LatestAssetURL() string {
	if AssetURL != "" {
		return AssetURL
	}
	return LatestReleaseURL + "/download/" + AssetName(runtime.GOOS, runtime.GOARCH)
}
