package tray

import (
	"strings"

	"github.com/scallister/call-detect/internal/project"
	"github.com/scallister/call-detect/internal/state"
)

const (
	menuInstall   = "install"
	menuUninstall = "uninstall"
	menuWebhook   = "webhook"
	menuGitHub    = "github"
	menuQuit      = "quit"
)

type menuChoice struct {
	ID    string
	Label string
}

func statusText(s state.Snapshot, failed bool) string {
	line := statusLine(s)
	if failed {
		return line + " — webhook failed"
	}
	return line
}

func actionChoices(a Actions) []menuChoice {
	var out []menuChoice
	if a.Install != nil && (a.AutostartOn == nil || !a.AutostartOn()) {
		out = append(out, menuChoice{menuInstall, "Install (start at logon)"})
	}
	if a.Uninstall != nil && (a.AutostartOn == nil || a.AutostartOn()) {
		out = append(out, menuChoice{menuUninstall, "Uninstall (remove logon startup)"})
	}
	if a.SetWebhookURL != nil {
		out = append(out, menuChoice{menuWebhook, "Set webhook URL..."})
	}
	out = append(out, menuChoice{menuGitHub, "GitHub..."})
	out = append(out, menuChoice{menuQuit, "Quit"})
	return out
}

func handleMenuID(a Actions, id string, quit func()) {
	switch id {
	case menuInstall:
		if a.Install == nil {
			return
		}
		if err := a.Install(); err != nil {
			Alert(err.Error(), true)
			return
		}
		Alert("Installed. call-detect will start at logon.", false)
	case menuUninstall:
		if a.Uninstall == nil {
			return
		}
		if err := a.Uninstall(); err != nil {
			Alert(err.Error(), true)
			return
		}
		Alert("Removed logon startup. The tray icon will keep running until you choose Quit.", false)
	case menuWebhook:
		if a.SetWebhookURL == nil {
			return
		}
		cur := ""
		if a.WebhookURL != nil {
			cur = a.WebhookURL()
		}
		url, ok := PromptWebhook(cur)
		if !ok {
			return
		}
		if err := a.SetWebhookURL(url); err != nil {
			Alert(err.Error(), true)
			return
		}
		if url == "" {
			Alert("Webhook disabled.", false)
			return
		}
		Alert("Webhook URL saved. Changes apply immediately.", false)
	case menuGitHub:
		if err := openBrowser(project.RepoURL); err != nil {
			Alert(err.Error(), true)
		}
	case menuQuit:
		if quit != nil {
			quit()
		}
	}
}

func statusLine(s state.Snapshot) string {
	if s.Busy {
		return "On a call"
	}
	return "Idle"
}

func boolLine(name string, on bool) string {
	if on {
		return name + ": yes"
	}
	return name + ": no"
}

func joinSources(src []string) string {
	return strings.Join(src, ", ")
}

func statusLines(s state.Snapshot, failed bool) []string {
	lines := []string{
		statusText(s, failed),
		boolLine("Microphone", s.Microphone),
		boolLine("Webcam", s.Webcam),
	}
	if len(s.Sources) > 0 {
		lines = append(lines, "Sources: "+joinSources(s.Sources))
	} else {
		lines = append(lines, "Sources: (none)")
	}
	return lines
}
