package tray

import (
	"encoding/binary"
	"testing"
)

func TestCircleICO(t *testing.T) {
	t.Parallel()
	ico := BusyICO()
	if len(ico) < 22 {
		t.Fatalf("ico too small: %d", len(ico))
	}
	if binary.LittleEndian.Uint16(ico[2:4]) != 1 {
		t.Fatal("not an icon")
	}
	if ico[6] != iconSize || ico[7] != iconSize {
		t.Fatalf("size %d x %d", ico[6], ico[7])
	}
	off := binary.LittleEndian.Uint32(ico[18:22])
	size := binary.LittleEndian.Uint32(ico[14:18])
	if int(off+size) != len(ico) {
		t.Fatalf("offset %d size %d len %d", off, size, len(ico))
	}
	if len(IdleICO()) != len(ico) || len(ErrorICO()) != len(ico) {
		t.Fatal("idle/busy/error size mismatch")
	}
}

func TestCirclePNG(t *testing.T) {
	t.Parallel()
	png := BusyPNG()
	if len(png) < 8 || string(png[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("not png: %d", len(png))
	}
	if len(IdlePNG()) < 8 || len(ErrorPNG()) < 8 {
		t.Fatal("idle/error png")
	}
}

func TestCircleARGB(t *testing.T) {
	t.Parallel()
	n := iconSize * iconSize * 4
	if len(IdleARGB()) != n || len(BusyARGB()) != n || len(ErrorARGB()) != n {
		t.Fatal("argb size")
	}
	// Opaque green ring has at least one pixel with A=0xFF.
	busy := BusyARGB()
	found := false
	for i := 0; i < iconSize*iconSize; i++ {
		if busy[i*4] == 0xFF && busy[i*4+2] == 0xCC {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected a green ARGB pixel")
	}
}

func TestTooltip(t *testing.T) {
	t.Parallel()
	if got := Tooltip(SnapshotIdle()); got != "call-detect: idle" {
		t.Fatalf("idle: %q", got)
	}
}
