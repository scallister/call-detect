// Package watch polls the ConsentStore and publishes debounced snapshots.
package watch

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/scallister/call-detect/internal/consentstore"
	"github.com/scallister/call-detect/internal/state"
	"github.com/scallister/call-detect/internal/status"
	"github.com/scallister/call-detect/internal/webhook"
)

// Options control the poll loop.
type Options struct {
	Store      consentstore.Store
	Debounce   time.Duration
	Poll       time.Duration
	StatusPath string
	Webhook    *webhook.Client
	OnUpdate   func(s state.Snapshot, boolsChanged bool)
	Log        *log.Logger
}

// Run polls until ctx is cancelled.
func Run(ctx context.Context, opt Options) error {
	if opt.Store == nil {
		return fmt.Errorf("watch: store is required")
	}
	if opt.Poll <= 0 {
		opt.Poll = time.Second
	}
	if opt.Debounce <= 0 {
		opt.Debounce = 2 * time.Second
	}
	logger := opt.Log
	if logger == nil {
		logger = log.Default()
	}

	deb := state.Debouncer{Delay: opt.Debounce}
	tick := time.NewTicker(opt.Poll)
	defer tick.Stop()

	step := func(now time.Time) {
		micEnt, err := opt.Store.List(consentstore.CapabilityMicrophone)
		if err != nil {
			logger.Printf("microphone: %v", err)
			return
		}
		camEnt, err := opt.Store.List(consentstore.CapabilityWebcam)
		if err != nil {
			logger.Printf("webcam: %v", err)
			return
		}
		raw := state.FromUsages(consentstore.ParseAll(micEnt), consentstore.ParseAll(camEnt), now)
		res := deb.Observe(raw, now)
		if !res.Changed {
			return
		}
		if opt.StatusPath != "" {
			if err := status.Write(opt.StatusPath, res.State); err != nil {
				logger.Printf("status: %v", err)
			}
		}
		if res.BoolsChanged && opt.Webhook != nil {
			if err := opt.Webhook.Post(ctx, res.State); err != nil {
				logger.Printf("webhook: %v", err)
			}
		}
		if opt.OnUpdate != nil {
			opt.OnUpdate(res.State, res.BoolsChanged)
		}
	}

	step(time.Now())
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-tick.C:
			step(now)
		}
	}
}
