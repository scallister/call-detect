//go:build linux

package live

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePactlJSON(t *testing.T) {
	t.Parallel()
	raw := []byte(`[
	  {"properties":{"application.process.binary":"firefox","application.name":"Firefox"}},
	  {"properties":{"application.name":"Discord"}}
	]`)
	names, err := parsePactlJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "firefox" || names[1] != "Discord" {
		t.Fatalf("%v", names)
	}
}

func TestParsePactlText(t *testing.T) {
	t.Parallel()
	raw := []byte(`Source Output #12
	application.process.binary = "chrome"
	application.name = "Google Chrome"
Source Output #13
	application.name = "Zoom"
`)
	names := parsePactlText(raw)
	if len(names) != 2 || names[0] != "chrome" || names[1] != "Zoom" {
		t.Fatalf("%v", names)
	}
}

func TestParsePWDump(t *testing.T) {
	t.Parallel()
	raw := []byte(`[
	  {"type":"PipeWire:Interface:Device"},
	  {"type":"PipeWire:Interface:Node","info":{"props":{
	    "media.class":"Stream/Input/Audio",
	    "application.process.binary":"firefox"
	  }}},
	  {"type":"PipeWire:Interface:Node","info":{"props":{
	    "media.class":"Stream/Input/Video",
	    "application.name":"firefox"
	  }}}
	]`)
	nodes, err := parsePWDump(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Fatalf("nodes %d", len(nodes))
	}
	var audio, video []string
	for _, n := range nodes {
		name := firstProp(n.props, "application.process.binary", "application.name")
		switch n.mediaClass {
		case "Stream/Input/Audio":
			audio = append(audio, name)
		case "Stream/Input/Video":
			video = append(video, name)
		}
	}
	if len(audio) != 1 || audio[0] != "firefox" || len(video) != 1 || video[0] != "firefox" {
		t.Fatalf("audio=%v video=%v", audio, video)
	}
}

func TestV4LStreaming(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dev := filepath.Join(root, "dev")
	proc := filepath.Join(root, "proc")
	if err := os.Mkdir(dev, 0o755); err != nil {
		t.Fatal(err)
	}
	video := filepath.Join(dev, "video0")
	if err := os.WriteFile(video, []byte("cam"), 0o644); err != nil {
		t.Fatal(err)
	}
	pid := filepath.Join(proc, "4242")
	if err := os.MkdirAll(filepath.Join(pid, "fd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pid, "comm"), []byte("firefox\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(video, filepath.Join(pid, "fd", "7")); err != nil {
		t.Fatal(err)
	}
	names := v4lStreaming(proc, dev)
	if len(names) != 1 || names[0] != "firefox" {
		t.Fatalf("%v", names)
	}
}
