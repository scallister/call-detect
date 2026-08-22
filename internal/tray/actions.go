package tray

import "context"

// Actions are optional tray menu callbacks wired by the main program.
type Actions struct {
	// Context is canceled when the process is quitting. Update checks
	// use it so a late GitHub response cannot show a dialog after Quit.
	Context       context.Context
	AutostartOn   func() bool
	Install       func() error
	Uninstall     func() error
	WebhookURL    func() string
	SetWebhookURL func(string) error
}
