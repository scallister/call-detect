package install

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/scallister/call-detect/internal/appdir"
	"github.com/scallister/call-detect/internal/version"
)

// ReadInstalledVersion returns the version written next to the installed exe.
func ReadInstalledVersion(dir string) string {
	raw, err := os.ReadFile(appdir.VersionPath(dir))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

// WriteInstalledVersion records ver for the installed copy.
func WriteInstalledVersion(dir, ver string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(appdir.VersionPath(dir), []byte(strings.TrimSpace(ver)+"\n"), 0o644)
}

// OfferReason reports whether self should replace the installed exe, and why.
func OfferReason(selfPath, installedPath, selfVer, instVer string) (bool, string) {
	if !fileExists(installedPath) {
		return false, ""
	}
	if sameFile(selfPath, installedPath) {
		return false, ""
	}
	if version.Newer(selfVer, instVer) {
		msg := fmt.Sprintf("This is call-detect %s. The installed copy is %s.\n\nUpdate the installed program and restart it?",
			version.Display(selfVer), version.Display(instVer))
		return true, msg
	}
	if instVer == "" && FilesDiffer(selfPath, installedPath) {
		return true, "A different call-detect is already installed.\n\nReplace it with this copy and restart?"
	}
	return false, ""
}

// Replace copies this process over dest, after asking a running copy to quit.
func Replace(dest string) error {
	if self, err := os.Executable(); err == nil && sameFile(self, dest) {
		return WriteInstalledVersion(filepath.Dir(dest), version.Version)
	}
	SignalQuit()
	if err := waitUnlocked(dest, 15*time.Second); err != nil {
		return err
	}
	if err := CopyExecutable(dest); err != nil {
		return err
	}
	return WriteInstalledVersion(filepath.Dir(dest), version.Version)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// FilesDiffer reports whether two paths are not the same bytes.
func FilesDiffer(a, b string) bool {
	ha, errA := fileSum(a)
	hb, errB := fileSum(b)
	if errA != nil || errB != nil {
		return true
	}
	return ha != hb
}

func fileSum(path string) ([sha256.Size]byte, error) {
	var none [sha256.Size]byte
	f, err := os.Open(path)
	if err != nil {
		return none, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return none, err
	}
	var out [sha256.Size]byte
	copy(out[:], h.Sum(nil))
	return out, nil
}

func waitUnlocked(path string, d time.Duration) error {
	if !fileExists(path) {
		return nil
	}
	deadline := time.Now().Add(d)
	for {
		f, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err == nil {
			f.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("installed program is still running; quit the tray icon and try again")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func sameFile(a, b string) bool {
	absA, err1 := filepath.Abs(a)
	absB, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return a == b
	}
	if resolved, err := filepath.EvalSymlinks(absA); err == nil {
		absA = resolved
	}
	if resolved, err := filepath.EvalSymlinks(absB); err == nil {
		absB = resolved
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(absA, absB)
	}
	return absA == absB
}
