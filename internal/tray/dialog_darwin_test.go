//go:build darwin

package tray

import "testing"

func TestApplescriptQuote(t *testing.T) {
	t.Parallel()
	if got := applescriptQuote(`say "hi"`); got != `"say \"hi\""` {
		t.Fatalf("got %s", got)
	}
}
