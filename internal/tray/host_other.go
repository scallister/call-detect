//go:build !windows

package tray

import (
	"sync"

	"github.com/scallister/call-detect/internal/state"
)

type hostImpl struct {
	done chan struct{}
	once sync.Once
}

func newHostImpl() hostImpl {
	return hostImpl{done: make(chan struct{})}
}

func (h *hostImpl) update(state.Snapshot) {}

func (h *hostImpl) run(ready func()) {
	if ready != nil {
		ready()
	}
	<-h.done
}

func (h *hostImpl) quit() {
	h.once.Do(func() { close(h.done) })
}
