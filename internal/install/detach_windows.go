//go:build windows

package install

import "os/exec"

func detach(*exec.Cmd) {}
