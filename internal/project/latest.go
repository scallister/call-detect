package project

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LatestAPI is the GitHub releases/latest endpoint. Tests may override it.
var LatestAPI = "https://api.github.com/repos/scallister/call-detect/releases/latest"

var httpClient = &http.Client{Timeout: 8 * time.Second}

var downloadClient = &http.Client{Timeout: 2 * time.Minute}

const maxDownload = 64 << 20

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

// Download writes the current platform's latest release binary to dest.
func Download(ctx context.Context, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LatestAssetURL(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "call-detect")
	req.Header.Set("Accept", "application/octet-stream")
	res, err := downloadClient.Do(req)
	if err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1<<20))
		return fmt.Errorf("download update: GitHub returned %s", res.Status)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp := dest + ".part"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, io.LimitReader(res.Body, maxDownload+1))
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("download update: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	st, err := os.Stat(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if st.Size() > maxDownload {
		_ = os.Remove(tmp)
		return fmt.Errorf("download update: file too large")
	}
	if st.Size() == 0 {
		_ = os.Remove(tmp)
		return fmt.Errorf("download update: empty file")
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(dest)
		if err := os.Rename(tmp, dest); err != nil {
			_ = os.Remove(tmp)
			return err
		}
	}
	return os.Chmod(dest, 0o755)
}
