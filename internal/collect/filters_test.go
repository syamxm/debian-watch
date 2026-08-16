package collect

import "testing"

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
