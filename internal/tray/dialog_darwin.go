//go:build darwin

package tray

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/scallister/call-detect/internal/state"
)

func Confirm(title, text string) bool {
	script := fmt.Sprintf(
		`display dialog %s with title %s buttons {"No", "Yes"} default button "Yes"`,
		applescriptQuote(text), applescriptQuote(title),
	)
	return osascript(script) == nil
}

func Alert(text string, isErr bool) {
	icon := "note"
	if isErr {
		icon = "stop"
	}
	script := fmt.Sprintf(
		`display dialog %s with title "call-detect" buttons {"OK"} default button "OK" with icon %s`,
		applescriptQuote(text), icon,
	)
	_ = osascript(script)
}

func PromptWebhook(current string) (string, bool) {
	script := fmt.Sprintf(
		`set r to display dialog %s default answer %s with title "Webhook URL" buttons {"Cancel", "OK"} default button "OK"
return text returned of r`,
		applescriptQuote(webhookPromptText()), applescriptQuote(current),
	)
	out, err := osascriptOutput(script)
	if err != nil {
		return "", false
	}
	return strings.TrimRight(out, "\r\n"), true
}

func openBrowser(url string) error {
	if err := exec.Command("open", url).Start(); err != nil {
		return fmt.Errorf("open %s: %w", url, err)
	}
	return nil
}

func webhookPromptText() string {
	return "Home Assistant webhook or other POST URL.\nLeave empty to disable.\n\nExample payload posted as JSON on each change:\n" + state.ExampleJSON
}

func applescriptQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

func osascript(script string) error {
	return exec.Command("osascript", "-e", script).Run()
}

func osascriptOutput(script string) (string, error) {
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
