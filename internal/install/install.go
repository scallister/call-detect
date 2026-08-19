// Package install registers or removes per-user autostart.
package install

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/scallister/call-detect/internal/appdir"
	"github.com/scallister/call-detect/internal/config"
)

const runValueName = "call-detect"

// Paths are the files used by a local install.
type Paths struct {
	Dir    string
	Exe    string
	Config string
}

// PrepareDir creates the data directory and returns standard paths.
func PrepareDir() (Paths, error) {
	dir, err := appdir.Ensure()
	if err != nil {
		return Paths{}, err
	}
	return Paths{
		Dir:    dir,
		Exe:    appdir.ExePath(dir),
		Config: appdir.ConfigPath(dir),
	}, nil
}

// CopyExecutable copies the current process to dest.
func CopyExecutable(dest string) error {
	src, err := os.Executable()
	if err != nil {
		return fmt.Errorf("current executable: %w", err)
	}
	src, err = filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}
	if sameFile(src, dest) {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dest, 0o755)
}

// Apply copies the executable, writes a sample config if needed, and enables autostart.
func Apply() (Paths, error) {
	paths, err := PrepareDir()
	if err != nil {
		return Paths{}, err
	}
	if err := Replace(paths.Exe); err != nil {
		return paths, err
	}
	if err := WriteSampleConfig(paths.Config); err != nil {
		return paths, err
	}
	if err := EnableAutostart(paths.Exe); err != nil {
		return paths, err
	}
	return paths, nil
}

// WriteSampleConfig creates a commented config.yaml if missing.
func WriteSampleConfig(path string) error {
	return config.WriteSample(path)
}

// Start launches exe detached from the installer process.
func Start(exe string) error {
	cmd := exec.Command(exe)
	cmd.Dir = filepath.Dir(exe)
	detach(cmd)
	return cmd.Start()
}

// Supported reports whether autostart install is implemented.
func Supported() bool {
	switch runtime.GOOS {
	case "windows", "linux", "darwin":
		return true
	default:
		return false
	}
}
