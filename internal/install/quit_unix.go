//go:build unix

package install

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/scallister/call-detect/internal/appdir"
)

// SignalQuit asks a running copy to exit by SIGTERM to the lock-file PID.
func SignalQuit() {
	dir, err := appdir.Dir()
	if err != nil {
		return
	}
	raw, err := os.ReadFile(filepath.Join(dir, "lock"))
	if err != nil {
		return
	}
	pid, ok := pidFromLock(raw)
	if !ok || pid == os.Getpid() {
		return
	}
	_ = unix.Kill(pid, unix.SIGTERM)
}

func pidFromLock(raw []byte) (int, bool) {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return 0, false
	}
	pid, err := strconv.Atoi(s)
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}
