package consentstore

import (
	"testing"
	"time"
)

func TestInUse(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		start uint64
		stop  uint64
		want  bool
	}{
		{name: "idle zeros", start: 0, stop: 0, want: false},
		{name: "open session", start: 100, stop: 0, want: true},
		{name: "start after stop", start: 200, stop: 100, want: true},
		{name: "closed session", start: 100, stop: 200, want: false},
		{name: "stop only", start: 0, stop: 50, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := InUse(tc.start, tc.stop); got != tc.want {
				t.Fatalf("InUse(%d, %d) = %v, want %v", tc.start, tc.stop, got, tc.want)
			}
		})
	}
}

func TestFiletimeToTime(t *testing.T) {
	t.Parallel()
	if !FiletimeToTime(0).IsZero() {
		t.Fatal("zero FILETIME should be zero time")
	}
	// 2020-01-01 00:00:00 UTC = 132223104000000000
	got := FiletimeToTime(132223104000000000)
	want := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestDisplayName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{in: `C:#Users#sam#AppData#Local#Discord#app-1.0.9#Discord.exe`, want: "Discord.exe"},
		{in: `C:#Program Files#Google#Chrome#Application#chrome.exe`, want: "chrome.exe"},
		{in: "Microsoft.Windows.Photos_8wekyb3d8bbwe", want: "Microsoft.Windows.Photos_8wekyb3d8bbwe"},
		{in: "", want: ""},
		{in: "###foo.exe", want: "foo.exe"},
	}
	for _, tc := range cases {
		if got := DisplayName(tc.in); got != tc.want {
			t.Fatalf("DisplayName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseAndCollect(t *testing.T) {
	t.Parallel()
	entries := []Entry{
		{KeyName: `C:#Tools#Discord.exe`, Start: 100, Stop: 0},
		{KeyName: "SomeStoreApp", Start: 10, Stop: 20},
		{KeyName: `C:#Tools#Discord.exe`, Start: 50, Stop: 0},
		{KeyName: "", Start: 1, Stop: 0},
	}
	usages := ParseAll(entries)
	if len(usages) != 3 {
		t.Fatalf("ParseAll len = %d, want 3", len(usages))
	}
	if !usages[0].InUse || usages[0].App != "Discord.exe" {
		t.Fatalf("first usage = %+v", usages[0])
	}
	if usages[1].InUse {
		t.Fatal("closed session should not be in use")
	}
	if !AnyInUse(usages) {
		t.Fatal("expected any in use")
	}
	apps := InUseApps(usages)
	if len(apps) != 1 || apps[0] != "Discord.exe" {
		t.Fatalf("InUseApps = %v", apps)
	}
}

func TestMemoryStore(t *testing.T) {
	t.Parallel()
	m := &Memory{Entries: map[string][]Entry{
		CapabilityMicrophone: {{KeyName: "zoom.exe", Start: 1, Stop: 0}},
		CapabilityWebcam:     {{KeyName: "zoom.exe", Start: 1, Stop: 2}},
	}}
	mic, err := m.List(CapabilityMicrophone)
	if err != nil || !AnyInUse(ParseAll(mic)) {
		t.Fatalf("mic: %v %+v", err, mic)
	}
	cam, err := m.List(CapabilityWebcam)
	if err != nil || AnyInUse(ParseAll(cam)) {
		t.Fatalf("cam: %v %+v", err, cam)
	}
}
