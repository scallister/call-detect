package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/scallister/call-detect/internal/appdir"
	"github.com/scallister/call-detect/internal/config"
	"github.com/scallister/call-detect/internal/consentstore"
	"github.com/scallister/call-detect/internal/detect"
	"github.com/scallister/call-detect/internal/dump"
	"github.com/scallister/call-detect/internal/install"
	"github.com/scallister/call-detect/internal/state"
	"github.com/scallister/call-detect/internal/tray"
	"github.com/scallister/call-detect/internal/wasapi"
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
	dumpFlag := fs.Bool("dump", false, "print ConsentStore records, live audio sessions, and the confirmed result")
	consoleFlag := fs.Bool("console", false, "show log output in a console window")
	webhookURL := fs.String("webhook-url", "", "override webhook URL (also CALL_DETECT_WEBHOOK_URL or config.yaml)")
	configPath := fs.String("config", "", "path to config.yaml (default: user data directory)")
	if err := fs.Parse(args); err != nil {
		return 2
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

	logFile, err := setupLog(dir, *consoleFlag || *dumpFlag)
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

	if runtime.GOOS != "windows" {
		log.Print("call-detect is a Windows program")
		return 1
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

	log.Printf("config %s", cfgFile)
	if cfg.WebhookURL != "" {
		log.Printf("webhook enabled")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host := tray.New()
	opt := watch.Options{
		Store:      consentstore.Windows{},
		Audio:      wasapi.Source{},
		Debounce:   2 * time.Second,
		Poll:       time.Second,
		StatusPath: appdir.StatusPath(dir),
		OnUpdate: func(s state.Snapshot, _ bool) {
			host.Update(s)
			log.Print(tray.Tooltip(s))
		},
	}
	if cfg.WebhookURL != "" {
		opt.Webhook = &webhook.Client{URL: cfg.WebhookURL}
	}

	host.Run(func() {
		go func() {
			if err := watch.Run(ctx, opt); err != nil {
				log.Print(err)
			}
			host.Quit()
		}()
	})
	cancel()
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
		fmt.Fprintln(os.Stderr, "install is only available on Windows")
		return 1
	}
	paths, err := install.PrepareDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := install.CopyExecutable(paths.Exe); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := install.WriteSampleConfig(paths.Config); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := install.EnableAutostart(paths.Exe); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := install.Start(paths.Exe); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("Installed %s\n", paths.Exe)
	fmt.Printf("Config    %s\n", paths.Config)
	fmt.Println("Starts automatically at logon. Edit config.yaml to set webhook_url.")
	return 0
}

func cmdUninstall() int {
	if !install.Supported() {
		fmt.Fprintln(os.Stderr, "uninstall is only available on Windows")
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

func cmdDump() int {
	store := consentstore.Windows{}
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
	audio := wasapi.Source{}.Sessions()
	snap := detect.Confirm(micU, camU, audio, time.Now())
	if err := dump.WriteAudio(os.Stdout, audio, snap); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
