package watch

import (
	"log"
	"time"

	"github.com/scallister/call-detect/internal/state"
	"github.com/scallister/call-detect/internal/status"
)

// PublishIdle writes status.json and POSTs call:false. Used on process exit
// so anything watching the webhook turns off before this program is gone.
func PublishIdle(opt Options, now time.Time) {
	s := state.Idle(now)
	logger := opt.Log
	if logger == nil {
		logger = log.Default()
	}
	if opt.StatusPath != "" {
		if err := status.Write(opt.StatusPath, s); err != nil {
			logger.Printf("status: %v", err)
		}
	}
	if opt.Webhook == nil || !opt.Webhook.Enabled() {
		return
	}
	if err := opt.Webhook.PostFinal(s); err != nil {
		logger.Printf("webhook: %v", err)
	}
}
