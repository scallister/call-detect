package main

import "testing"

func TestVersionFlag(t *testing.T) {
	if code := run([]string{"-version"}); code != 0 {
		t.Fatalf("code %d", code)
	}
}
