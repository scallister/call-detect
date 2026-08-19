//go:build windows

package tray

import "github.com/scallister/call-detect/internal/state"

func PromptWebhook(current string) (string, bool) {
	return promptText(0, "Webhook URL", "Home Assistant webhook or other POST URL.\nLeave empty to disable.", current, state.ExampleJSON)
}

func openBrowser(url string) error {
	return openURL(url)
}
