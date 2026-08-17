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
	if len(IdleICO()) != len(ico) {
		t.Fatal("idle/busy size mismatch")
	}
}

func TestTooltip(t *testing.T) {
	t.Parallel()
	if got := Tooltip(SnapshotIdle()); got != "call-detect: idle" {
		t.Fatalf("idle: %q", got)
	}
}
