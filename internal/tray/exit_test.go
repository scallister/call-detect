package tray

import (
	"sync/atomic"
	"testing"
)

func TestRunExitHookOnce(t *testing.T) {
	var n atomic.Int32
	SetExitHook(func() { n.Add(1) })
	t.Cleanup(func() { SetExitHook(nil) })
	RunExitHook()
	RunExitHook()
	if n.Load() != 1 {
		t.Fatalf("ran %d times", n.Load())
	}
}
