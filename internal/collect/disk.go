package collect

import (
	"context"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

type Disk struct {
	Mountpoint  string
	Device      string
	Filesystem  string
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
}

var pseudoFilesystems = map[string]bool{
	"autofs":          true,
	"binfmt_misc":     true,
	"bpf":             true,
	"cgroup":          true,
	"cgroup2":         true,
	"configfs":        true,
	"debugfs":         true,
	"devpts":          true,
	"devtmpfs":        true,
	"efivarfs":        true,
	"fuse.gvfsd-fuse": true,
	"fusectl":         true,
	"hugetlbfs":       true,
	"mqueue":          true,
	"nsfs":            true,
	"overlay":         true,
	"proc":            true,
	"pstore":          true,
	"ramfs":           true,
	"securityfs":      true,
	"squashfs":        true,
	"sysfs":           true,
	"tmpfs":           true,
	"tracefs":         true,
}

func (c *Collector) collectDisks(ctx context.Context) []Disk {
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		c.unavailable("disk", err)
		return nil
	}

	disks := make([]Disk, 0, len(partitions))
	for _, partition := range partitions {
		if skipMount(partition.Fstype, partition.Mountpoint) {
			continue
		}
		usage, err := disk.UsageWithContext(ctx, filepath.Join(c.hostRoot, partition.Mountpoint))
		if err != nil || usage.Total == 0 {
			continue
		}
		disks = append(disks, Disk{
			Mountpoint:  partition.Mountpoint,
			Device:      partition.Device,
			Filesystem:  partition.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
		})
	}

	sort.Slice(disks, func(a, b int) bool { return disks[a].Mountpoint < disks[b].Mountpoint })
	return disks
}

func skipMount(filesystem, mountpoint string) bool {
	if pseudoFilesystems[filesystem] {
		return true
	}
	return strings.HasPrefix(mountpoint, "/var/lib/docker/") ||
		strings.HasPrefix(mountpoint, "/snap/") ||
		strings.HasPrefix(mountpoint, "/run/")
}
