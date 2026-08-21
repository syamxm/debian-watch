package collect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkipMount(t *testing.T) {
	cases := []struct {
		filesystem string
		mountpoint string
		want       bool
	}{
		{"ext4", "/", false},
		{"btrfs", "/mnt/data", false},
		{"xfs", "/srv", false},
		{"tmpfs", "/dev/shm", true},
		{"overlay", "/var/lib/docker/overlay2/abc/merged", true},
		{"ext4", "/var/lib/docker/volumes", true},
		{"squashfs", "/snap/core/1", true},
		{"proc", "/proc", true},
	}
	for _, tc := range cases {
		if got := skipMount(tc.filesystem, tc.mountpoint); got != tc.want {
			t.Errorf("skipMount(%q, %q) = %v, want %v", tc.filesystem, tc.mountpoint, got, tc.want)
		}
	}
}

func TestSkipInterface(t *testing.T) {
	cases := map[string]bool{
		"eth0":        false,
		"enp3s0":      false,
		"wlan0":       false,
		"lo":          true,
		"veth1a2b3c4": true,
		"br-9f8e7d":   true,
		"docker0":     true,
		"tun0":        true,
	}
	for name, want := range cases {
		if got := skipInterface(name); got != want {
			t.Errorf("skipInterface(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestReadNetDev(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dev")
	content := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1000      10    0    0    0     0          0         0     1000      10    0    0    0     0       0          0
  eth0: 5000      50    0    0    0     0          0         0     2500      25    0    0    0     0       0          0
veth123: 700       7    0    0    0     0          0         0      300       3    0    0    0     0       0          0
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	counters, err := readNetDev(path)
	if err != nil {
		t.Fatalf("readNetDev: %v", err)
	}

	if len(counters) != 1 {
		t.Fatalf("got %d interfaces, want 1 (lo and veth are filtered)", len(counters))
	}
	eth0, ok := counters["eth0"]
	if !ok {
		t.Fatal("eth0 missing")
	}
	if eth0.bytesRecv != 5000 || eth0.bytesSent != 2500 {
		t.Errorf("eth0 = %+v, want recv 5000 sent 2500", eth0)
	}
}

func TestReadNetDevMissingFile(t *testing.T) {
	if _, err := readNetDev(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("expected an error for a missing counter file")
	}
}
