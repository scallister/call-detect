package version

import "testing"

func TestCompare(t *testing.T) {
	t.Parallel()
	cases := []struct {
		a, b string
		want int
	}{
		{"v0.0.5", "v0.0.4", 1},
		{"0.0.4", "v0.0.5", -1},
		{"v0.0.5", "v0.0.5", 0},
		{"v0.1.0", "v0.0.9", 1},
		{"v0.0.5", "", 1},
		{"v0.0.5", "dev", 1},
		{"dev", "v0.0.4", 0},
		{"", "v0.0.4", 0},
		{"dev", "dev", 0},
	}
	for _, tc := range cases {
		if got := Compare(tc.a, tc.b); got != tc.want {
			t.Fatalf("Compare(%q, %q)=%d want %d", tc.a, tc.b, got, tc.want)
		}
	}
	if !Newer("v0.0.5", "0.0.4") {
		t.Fatal("expected newer")
	}
}

func TestDisplay(t *testing.T) {
	t.Parallel()
	if Display("") != "unknown" {
		t.Fatal(Display(""))
	}
	if Display("v0.0.5") != "v0.0.5" {
		t.Fatal(Display("v0.0.5"))
	}
}
