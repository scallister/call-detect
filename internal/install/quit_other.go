//go:build !windows

package install

// SignalQuit is a no-op off Windows.
func SignalQuit() {}
