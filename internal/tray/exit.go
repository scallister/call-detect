package tray

import "sync"

var (
	exitOnce sync.Once
	exitFn   func()
)

// SetExitHook registers a function that RunExitHook calls once when the
// process is leaving. Main uses this to POST call:false.
func SetExitHook(fn func()) {
	exitFn = fn
	exitOnce = sync.Once{}
}

// RunExitHook runs the exit hook at most once. Safe from any goroutine.
func RunExitHook() {
	exitOnce.Do(func() {
		if exitFn != nil {
			exitFn()
		}
	})
}
