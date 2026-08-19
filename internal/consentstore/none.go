package consentstore

// None is a Store with no records. Used on macOS and Linux, which have no
// ConsentStore. List succeeds so the watch loop can rely on live sessions.
type None struct{}

// List returns no entries.
func (None) List(string) ([]Entry, error) {
	return nil, nil
}
