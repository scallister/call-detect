package tray

import "github.com/scallister/call-detect/internal/state"

// Host is the notification-area icon. On Windows it owns the message loop.
type Host struct {
	impl hostImpl
}

// New creates a tray host. Call Run on the main thread on Windows.
func New() *Host {
	return &Host{impl: newHostImpl()}
}

// Update changes the icon, tooltip, and menu to match s.
func (h *Host) Update(s state.Snapshot) {
	h.impl.update(s)
}

// Run shows the icon and blocks until the user chooses Quit.
func (h *Host) Run(ready func()) {
	h.impl.run(ready)
}

// Quit asks the tray to exit. Safe to call from another goroutine.
func (h *Host) Quit() {
	h.impl.quit()
}
