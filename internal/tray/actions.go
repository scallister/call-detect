package tray

// Actions are optional tray menu callbacks wired by the main program.
type Actions struct {
	AutostartOn   func() bool
	Install       func() error
	Uninstall     func() error
	WebhookURL    func() string
	SetWebhookURL func(string) error
}

func (a Actions) hasSetup() bool {
	return a.Install != nil || a.Uninstall != nil || a.SetWebhookURL != nil
}
