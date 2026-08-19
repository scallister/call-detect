//go:build !unix

package main

import (
	"os"
	"syscall"
)

var quitSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}
