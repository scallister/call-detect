//go:build linux

package tray

import (
	"strings"
	"testing"
)

func TestLinuxDialogArgs(t *testing.T) {
	t.Parallel()
	if args := confirmArgs("/usr/bin/zenity", "T", "Q"); args[1] != "--question" || args[5] != "Q" {
		t.Fatalf("zenity confirm: %v", args)
	}
	if args := confirmArgs("/usr/bin/kdialog", "T", "Q"); args[1] != "--title" || args[3] != "--yesno" {
		t.Fatalf("kdialog confirm: %v", args)
	}
	if args := alertArgs("/usr/bin/zenity", "boom", true); args[1] != "--error" {
		t.Fatalf("zenity error: %v", args)
	}
	if args := entryArgs("/usr/bin/zenity", "Webhook URL", "text", "http://x"); !containsPair(args, "--entry-text", "http://x") {
		t.Fatalf("zenity entry: %v", args)
	}
	choices := []menuChoice{{"quit", "Quit"}}
	args := listArgs("/usr/bin/zenity", "Idle", choices)
	if !containsPair(args, "--print-column", "1") || args[len(args)-2] != "quit" {
		t.Fatalf("zenity list: %v", args)
	}
	args = listArgs("/usr/bin/kdialog", "Idle", choices)
	if args[3] != "--menu" || args[len(args)-2] != "quit" {
		t.Fatalf("kdialog list: %v", args)
	}
}

func TestEscapeMenuLabel(t *testing.T) {
	t.Parallel()
	if got := escapeMenuLabel("Set webhook URL..."); got != "Set webhook URL..." {
		t.Fatal(got)
	}
	if got := escapeMenuLabel("on_a_call"); got != "on__a__call" {
		t.Fatal(got)
	}
}

func containsPair(args []string, k, v string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == k && args[i+1] == v {
			return true
		}
	}
	return strings.Contains(strings.Join(args, " "), k+" "+v)
}
