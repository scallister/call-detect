// Package watch polls the ConsentStore and publishes debounced snapshots.
package watch

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/scallister/call-detect/internal/consentstore"
	"github.com/scallister/call-detect/internal/detect"
	"github.com/scallister/call-detect/internal/state"
	"github.com/scallister/call-detect/internal/status"
	"github.com/scallister/call-detect/internal/webhook"
)

// AudioSource lists live audio sessions. Optional; if nil, ConsentStore is used alone.
type AudioSource interface {
	Sessions() detect.Audio
}

// CameraSource lists processes currently streaming a camera. Optional; if nil,
// webcam confirmation falls back to ConsentStore and audio sessions.
type CameraSource interface {
	Streaming() detect.Camera
}

var (
	errNoAudio  = fmt.Errorf("audio source not configured")
	errNoCamera = fmt.Errorf("camera source not configured")
)

// Options control the poll loop.
type Options struct {
	Store      consentstore.Store
	Audio      AudioSource
	Camera     CameraSource
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
	loggedAudioErr := false
	loggedCameraErr := false

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
		mic := consentstore.ParseAll(micEnt)
		cam := consentstore.ParseAll(camEnt)
		audio := detect.Audio{Err: errNoAudio}
		if opt.Audio != nil {
			audio = opt.Audio.Sessions()
			if audio.Err != nil && !loggedAudioErr {
				logger.Printf("audio sessions: %v", audio.Err)
				loggedAudioErr = true
			}
		}
		camera := detect.Camera{Err: errNoCamera}
		if opt.Camera != nil {
			camera = opt.Camera.Streaming()
			if camera.Err != nil && !loggedCameraErr {
				logger.Printf("camera monitor: %v", camera.Err)
				loggedCameraErr = true
			}
		}
		raw := detect.Confirm(mic, cam, audio, camera, now)
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
