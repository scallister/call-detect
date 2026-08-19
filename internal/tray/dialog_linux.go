//go:build linux

package tray

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/scallister/call-detect/internal/state"
)

func Confirm(title, text string) bool {
	args, ok := linuxConfirmArgs(title, text)
	if !ok {
		return false
	}
	return exec.Command(args[0], args[1:]...).Run() == nil
}

func Alert(text string, isErr bool) {
	args, ok := linuxAlertArgs(text, isErr)
	if !ok {
		return
	}
	_ = exec.Command(args[0], args[1:]...).Run()
}

func PromptWebhook(current string) (string, bool) {
	args, ok := linuxEntryArgs("Webhook URL", webhookPromptText(), current)
	if !ok {
		return "", false
	}
	out, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		return "", false
	}
	return strings.TrimRight(string(out), "\r\n"), true
}

func openBrowser(url string) error {
	if err := exec.Command("xdg-open", url).Start(); err != nil {
		return fmt.Errorf("open %s: %w", url, err)
	}
	return nil
}

func webhookPromptText() string {
	return "Home Assistant webhook or other POST URL.\nLeave empty to disable.\n\nExample payload posted as JSON on each change:\n" + state.ExampleJSON
}

func linuxDialog() string {
	for _, name := range []string{"zenity", "kdialog"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

func linuxConfirmArgs(title, text string) ([]string, bool) {
	bin := linuxDialog()
	if bin == "" {
		return nil, false
	}
	return confirmArgs(bin, title, text), true
}

func linuxAlertArgs(text string, isErr bool) ([]string, bool) {
	bin := linuxDialog()
	if bin == "" {
		return nil, false
	}
	return alertArgs(bin, text, isErr), true
}

func linuxEntryArgs(title, text, current string) ([]string, bool) {
	bin := linuxDialog()
	if bin == "" {
		return nil, false
	}
	return entryArgs(bin, title, text, current), true
}

func linuxListArgs(text string, choices []menuChoice) ([]string, bool) {
	bin := linuxDialog()
	if bin == "" || len(choices) == 0 {
		return nil, false
	}
	return listArgs(bin, text, choices), true
}

func isKDialog(bin string) bool {
	return strings.Contains(bin, "kdialog")
}

func confirmArgs(bin, title, text string) []string {
	if isKDialog(bin) {
		return []string{bin, "--title", title, "--yesno", text}
	}
	return []string{bin, "--question", "--title", title, "--text", text, "--no-wrap"}
}

func alertArgs(bin, text string, isErr bool) []string {
	if isKDialog(bin) {
		kind := "--msgbox"
		if isErr {
			kind = "--error"
		}
		return []string{bin, "--title", "call-detect", kind, text}
	}
	kind := "--info"
	if isErr {
		kind = "--error"
	}
	return []string{bin, kind, "--title", "call-detect", "--text", text, "--no-wrap"}
}

func entryArgs(bin, title, text, current string) []string {
	if isKDialog(bin) {
		return []string{bin, "--title", title, "--inputbox", text, current}
	}
	return []string{bin, "--entry", "--title", title, "--text", text, "--entry-text", current}
}

func listArgs(bin, text string, choices []menuChoice) []string {
	if isKDialog(bin) {
		args := []string{bin, "--title", "call-detect", "--menu", text}
		for _, c := range choices {
			args = append(args, c.ID, c.Label)
		}
		return args
	}
	args := []string{bin, "--list", "--title", "call-detect", "--text", text, "--column", "id", "--column", "Action", "--hide-column", "1", "--print-column", "1", "--hide-header"}
	for _, c := range choices {
		args = append(args, c.ID, c.Label)
	}
	return args
}

func pickMenu(info []string, choices []menuChoice) (string, bool) {
	args, ok := linuxListArgs(strings.Join(info, "\n"), choices)
	if !ok {
		return "", false
	}
	out, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		return "", false
	}
	id := strings.TrimSpace(string(bytes.TrimRight(out, "\r\n")))
	for _, c := range choices {
		if c.ID == id || c.Label == id {
			return c.ID, true
		}
	}
	return "", false
}
