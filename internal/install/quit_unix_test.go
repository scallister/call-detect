//go:build unix

package install

import "testing"

func TestPidFromLock(t *testing.T) {
	t.Parallel()
	if _, ok := pidFromLock(nil); ok {
		t.Fatal("empty")
	}
	if _, ok := pidFromLock([]byte("nope")); ok {
		t.Fatal("garbage")
	}
	pid, ok := pidFromLock([]byte("  4321\n"))
	if !ok || pid != 4321 {
		t.Fatalf("got %d %v", pid, ok)
	}
}
