package tray

import "github.com/scallister/call-detect/internal/state"

// Host is the notification-area / menu-bar icon. If the desktop tray
// cannot be created, Run still keeps the process alive until Quit.
type Host struct {
	impl hostImpl
}

// startReady starts the detection loop. It must run even if the tray icon fails.
func startReady(ready func()) {
	if ready != nil {
		go ready()
	}
}

// New creates a tray host. Call Run on the main thread on Windows.
func New() *Host {
	return &Host{impl: newHostImpl()}
}

// SetActions attaches Install / Uninstall / webhook menu handlers.
func (h *Host) SetActions(a Actions) {
	h.impl.setActions(a)
}

// Update changes the icon, tooltip, and menu to match s.
func (h *Host) Update(s state.Snapshot) {
	h.impl.update(s)
}

// SetWebhookFailed turns the tray icon red when the last webhook POST failed.
func (h *Host) SetWebhookFailed(failed bool) {
	h.impl.setWebhookFailed(failed)
}

// Run shows the icon and blocks until Quit. ready always starts, even if
// the notification icon cannot be created, so detection and webhooks still run.
func (h *Host) Run(ready func()) {
	h.impl.run(ready)
}

// Quit asks the tray to exit. Safe to call from another goroutine.
func (h *Host) Quit() {
	h.impl.quit()
}
