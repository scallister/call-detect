//go:build !windows && !unix

package install

// SignalQuit is a no-op on platforms without a singleton quit signal.
func SignalQuit() {}
