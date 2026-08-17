package dump

import (
	"strings"
	"testing"
	"time"

	"github.com/scallister/call-detect/internal/consentstore"
)

func TestWrite(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	mic := []consentstore.Usage{
		{App: "Discord.exe", InUse: true, Start: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)},
	}
	if err := Write(&b, mic, nil); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	if !strings.Contains(got, "IN USE") || !strings.Contains(got, "Discord.exe") {
		t.Fatalf("mic: %s", got)
	}
	if !strings.Contains(got, "(no records)") {
		t.Fatalf("empty webcam: %s", got)
	}
}
