package main

//go:generate go run ../winres -version dev -o rsrc_windows_amd64.syso

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/scallister/call-detect/internal/appdir"
	"github.com/scallister/call-detect/internal/config"
	"github.com/scallister/call-detect/internal/consentstore"
	"github.com/scallister/call-detect/internal/detect"
	"github.com/scallister/call-detect/internal/dump"
	"github.com/scallister/call-detect/internal/install"
	"github.com/scallister/call-detect/internal/live"
	"github.com/scallister/call-detect/internal/state"
	"github.com/scallister/call-detect/internal/tray"
	"github.com/scallister/call-detect/internal/version"
	"github.com/scallister/call-detect/internal/watch"
	"github.com/scallister/call-detect/internal/webhook"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("call-detect", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	installFlag := fs.Bool("install", false, "copy the program to the user data directory and start it at logon")
	uninstallFlag := fs.Bool("uninstall", false, "remove the logon autostart entry")
	dumpFlag := fs.Bool("dump", false, "print ConsentStore records, live audio and camera activity, and the confirmed result")
	consoleFlag := fs.Bool("console", false, "show log output in a console window")
	webhookURL := fs.String("webhook-url", "", "override webhook URL (also CALL_DETECT_WEBHOOK_URL or config.yaml)")
	configPath := fs.String("config", "", "path to config.yaml (default: user data directory)")
	versionFlag := fs.Bool("version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *versionFlag {
		fmt.Println(version.Display(version.Version))
		return 0
	}

	needConsole := *installFlag || *uninstallFlag || *dumpFlag || *consoleFlag
	if needConsole {
		enableConsole(*consoleFlag || *installFlag || *uninstallFlag || *dumpFlag)
	}

	if *installFlag {
		return cmdInstall()
	}
	if *uninstallFlag {
		return cmdUninstall()
	}

	dir, err := appdir.Ensure()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	logFile, err := setupLog(dir, *consoleFlag || *dumpFlag || runtime.GOOS != "windows")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if logFile != nil {
		defer logFile.Close()
	}

	if *dumpFlag {
		return cmdDump()
	}

	if code, done := maybeUpdate(); done {
		return code
	}

	if singletonHeld() {
		log.Print("already running")
		return 0
	}

	cfgFile := *configPath
	if cfgFile == "" {
		cfgFile = appdir.ConfigPath(dir)
	}
	file, err := config.LoadFile(cfgFile)
	if err != nil {
		log.Print(err)
		return 1
	}
	cfg := config.Resolve(*webhookURL, os.Getenv("CALL_DETECT_WEBHOOK_URL"), file)
	cfg.ConfigPath = cfgFile

	log.Printf("version %s", version.Version)
	log.Printf("config %s", cfgFile)
	if cfg.WebhookURL != "" {
		log.Printf("webhook enabled")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host := tray.New()
	hook := &webhook.Client{URL: cfg.WebhookURL}
	var lastMu sync.Mutex
	var lastSnap state.Snapshot
	var haveSnap bool
	host.SetActions(tray.Actions{
		AutostartOn: install.AutostartEnabled,
		Install: func() error {
			_, err := install.Apply()
			return err
		},
		Uninstall: install.DisableAutostart,
		WebhookURL: func() string {
			return hook.GetURL()
		},
		SetWebhookURL: func(url string) error {
			url = strings.TrimSpace(url)
			if err := config.WriteWebhook(cfgFile, url); err != nil {
				return err
			}
			hook.SetURL(url)
			if url == "" {
				log.Print("webhook disabled")
				host.SetWebhookFailed(false)
				return nil
			}
			log.Print("webhook enabled")
			lastMu.Lock()
			s, ok := lastSnap, haveSnap
			lastMu.Unlock()
			if !ok {
				host.SetWebhookFailed(false)
				return nil
			}
			if err := hook.Post(ctx, s); err != nil {
				log.Printf("webhook: %v", err)
				host.SetWebhookFailed(true)
				return nil
			}
			host.SetWebhookFailed(false)
			return nil
		},
	})
	store, audioSrc, cameraSrc := deviceSources()
	opt := watch.Options{
		Store:      store,
		Audio:      audioSrc,
		Camera:     cameraSrc,
		Debounce:   2 * time.Second,
		Poll:       time.Second,
		StatusPath: appdir.StatusPath(dir),
		Webhook:    hook,
		OnUpdate: func(s state.Snapshot, _ bool) {
			lastMu.Lock()
			lastSnap = s
			haveSnap = true
			lastMu.Unlock()
			host.Update(s)
			log.Print(tray.Tooltip(s))
		},
		OnWebhook: func(err error) {
			host.SetWebhookFailed(err != nil)
		},
	}

	tray.SetExitHook(func() {
		cancel()
		watch.PublishIdle(opt, time.Now())
	})
	quit := func() {
		tray.RunExitHook()
		host.Quit()
	}
	if install.RunningFromInstall() {
		watchRemoteQuit(quit)
	}
	notifyQuit(quit)
	watchDone := make(chan struct{})
	host.Run(func() {
		go tray.OfferRemoteUpdate(false)
		go func() {
			defer close(watchDone)
			if err := watch.Run(ctx, opt); err != nil {
				log.Print(err)
			}
			host.Quit()
		}()
	})
	cancel()
	select {
	case <-watchDone:
	case <-time.After(time.Second):
	}
	tray.RunExitHook()
	return 0
}

func setupLog(dir string, alsoStdout bool) (*os.File, error) {
	path := filepath.Join(dir, "call-detect.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	var w io.Writer = f
	if alsoStdout {
		w = io.MultiWriter(os.Stdout, f)
	}
	log.SetOutput(w)
	log.SetFlags(log.LstdFlags)
	log.SetPrefix("call-detect: ")
	return f, nil
}

func cmdInstall() int {
	if !install.Supported() {
		fmt.Fprintln(os.Stderr, "install is not available on this operating system")
		return 1
	}
	paths, err := install.Apply()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := install.Start(paths.Exe); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("Installed %s (%s)\n", paths.Exe, version.Version)
	fmt.Printf("Config    %s\n", paths.Config)
	fmt.Println("Starts automatically at logon. Edit config.yaml to set webhook_url.")
	return 0
}

func maybeUpdate() (int, bool) {
	self, err := os.Executable()
	if err != nil {
		return 0, false
	}
	dir, err := appdir.Dir()
	if err != nil {
		return 0, false
	}
	installed := appdir.ExePath(dir)
	offer, msg := install.OfferReason(self, installed, version.Version, install.ReadInstalledVersion(dir))
	if !offer {
		return 0, false
	}
	if !tray.Confirm("call-detect", msg) {
		return 0, false
	}
	if err := install.Replace(installed); err != nil {
		tray.Alert(err.Error(), true)
		return 1, true
	}
	if err := install.EnableAutostart(installed); err != nil {
		log.Print(err)
	}
	if err := install.Start(installed); err != nil {
		tray.Alert(err.Error(), true)
		return 1, true
	}
	return 0, true
}

func cmdUninstall() int {
	if !install.Supported() {
		fmt.Fprintln(os.Stderr, "uninstall is not available on this operating system")
		return 1
	}
	if err := install.DisableAutostart(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	dir, err := appdir.Dir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("Removed logon autostart.")
	fmt.Printf("Files left in %s (quit the tray icon if it is still running).\n", dir)
	return 0
}

func deviceSources() (consentstore.Store, watch.AudioSource, watch.CameraSource) {
	if runtime.GOOS == "windows" {
		return consentstore.Windows{}, live.Audio{}, live.Camera{}
	}
	return consentstore.None{}, live.Audio{}, live.Camera{}
}

func cmdDump() int {
	store, audioSrc, cameraSrc := deviceSources()
	mic, err := store.List(consentstore.CapabilityMicrophone)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cam, err := store.List(consentstore.CapabilityWebcam)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	micU := consentstore.ParseAll(mic)
	camU := consentstore.ParseAll(cam)
	if err := dump.Write(os.Stdout, micU, camU); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	audio := audioSrc.Sessions()
	camLive := cameraSrc.Streaming()
	snap := detect.Confirm(micU, camU, audio, camLive, time.Now())
	if err := dump.WriteAudio(os.Stdout, audio); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := dump.WriteCamera(os.Stdout, camLive); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := dump.WriteResult(os.Stdout, snap); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
