package tray

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestStartReadyInvokesCallback(t *testing.T) {
	t.Parallel()
	var called atomic.Bool
	done := make(chan struct{})
	startReady(func() {
		called.Store(true)
		close(done)
	})
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ready was not called")
	}
	if !called.Load() {
		t.Fatal("ready was not called")
	}
}

func TestStartReadyNil(t *testing.T) {
	t.Parallel()
	startReady(nil)
}

func TestHostRunStartsReady(t *testing.T) {
	h := New()
	ready := make(chan struct{})
	done := make(chan struct{})
	go func() {
		h.Run(func() { close(ready) })
		close(done)
	}()
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("Run did not start ready")
	}
	h.Quit()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after Quit")
	}
}
