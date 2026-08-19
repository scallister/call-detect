//go:build linux

package live

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func v4lStreaming(procRoot, devRoot string) []string {
	if procRoot == "" {
		procRoot = "/proc"
	}
	if devRoot == "" {
		devRoot = "/dev"
	}
	targets := videoDevices(devRoot)
	if len(targets) == 0 {
		return nil
	}
	pids, err := os.ReadDir(procRoot)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range pids {
		if !e.IsDir() {
			continue
		}
		pid := e.Name()
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}
		fdDir := filepath.Join(procRoot, pid, "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}
		for _, fd := range fds {
			dest, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil {
				continue
			}
			if !targets[normalizeDev(dest)] {
				continue
			}
			if name := linuxProcessName(procRoot, pid); name != "" {
				names = append(names, name)
			}
			break
		}
	}
	return uniqueNames(names)
}

func videoDevices(devRoot string) map[string]bool {
	out := map[string]bool{}
	ents, err := os.ReadDir(devRoot)
	if err != nil {
		return out
	}
	for _, e := range ents {
		name := e.Name()
		if !strings.HasPrefix(name, "video") {
			continue
		}
		rest := strings.TrimPrefix(name, "video")
		if rest == "" {
			continue
		}
		if _, err := strconv.Atoi(rest); err != nil {
			continue
		}
		p := filepath.Join(devRoot, name)
		if resolved, err := filepath.EvalSymlinks(p); err == nil {
			p = resolved
		}
		out[normalizeDev(p)] = true
	}
	return out
}

func normalizeDev(p string) string {
	p = strings.TrimSpace(p)
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		p = resolved
	}
	return filepath.Clean(p)
}

func linuxProcessName(procRoot, pid string) string {
	if raw, err := os.ReadFile(filepath.Join(procRoot, pid, "exe")); err == nil {
		if s := strings.TrimSpace(string(raw)); s != "" {
			return filepath.Base(s)
		}
	}
	if dest, err := os.Readlink(filepath.Join(procRoot, pid, "exe")); err == nil {
		if dest != "" {
			return filepath.Base(strings.TrimSuffix(dest, " (deleted)"))
		}
	}
	if raw, err := os.ReadFile(filepath.Join(procRoot, pid, "comm")); err == nil {
		return strings.TrimSpace(string(raw))
	}
	return ""
}
