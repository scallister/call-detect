//go:build !windows && !linux && !darwin

package tray

import (
	"sync"

	"github.com/scallister/call-detect/internal/state"
)

type hostImpl struct {
	done    chan struct{}
	once    sync.Once
	actions Actions
}

func newHostImpl() hostImpl {
	return hostImpl{done: make(chan struct{})}
}

func (h *hostImpl) setActions(a Actions) {
	h.actions = a
}

func (h *hostImpl) update(state.Snapshot) {}

func (h *hostImpl) setWebhookFailed(bool) {}

func (h *hostImpl) run(ready func()) {
	startReady(ready)
	<-h.done
}

func (h *hostImpl) quit() {
	RunExitHook()
	h.once.Do(func() { close(h.done) })
}

// Confirm is only implemented on Windows, macOS, and Linux.
func Confirm(string, string) bool { return false }

// Alert is only implemented on Windows, macOS, and Linux.
func Alert(string, bool) {}

func PromptWebhook(string) (string, bool) { return "", false }

func openBrowser(url string) error {
	return errNoBrowser
}
