package project

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LatestAPI is the GitHub releases/latest endpoint. Tests may override it.
var LatestAPI = "https://api.github.com/repos/scallister/call-detect/releases/latest"

var httpClient = &http.Client{Timeout: 8 * time.Second}

type latestRelease struct {
	TagName string `json:"tag_name"`
}

// LatestTag returns the newest GitHub release tag (for example v0.1.0).
func LatestTag(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LatestAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "call-detect")
	req.Header.Set("Accept", "application/vnd.github+json")
	res, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("check for updates: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("check for updates: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("check for updates: GitHub returned %s", res.Status)
	}
	var rel latestRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return "", fmt.Errorf("check for updates: %w", err)
	}
	tag := strings.TrimSpace(rel.TagName)
	if tag == "" {
		return "", fmt.Errorf("check for updates: missing release tag")
	}
	return tag, nil
}
