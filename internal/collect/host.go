package collect

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
)

type Host struct {
	Available       bool
	Hostname        string
	OS              string
	Platform        string
	PlatformVersion string
	KernelVersion   string
	KernelArch      string
	Uptime          time.Duration
	BootTime        time.Time
}

type Load struct {
	Available bool
	Load1     float64
	Load5     float64
	Load15    float64
}

func (c *Collector) collectHost(ctx context.Context) Host {
	info, err := host.InfoWithContext(ctx)
	if err != nil {
		c.unavailable("host", err)
		return Host{}
	}
	return Host{
		Available:       true,
		Hostname:        c.hostname(info.Hostname),
		OS:              info.OS,
		Platform:        info.Platform,
		PlatformVersion: info.PlatformVersion,
		KernelVersion:   info.KernelVersion,
		KernelArch:      info.KernelArch,
		Uptime:          time.Duration(info.Uptime) * time.Second,
		BootTime:        time.Unix(int64(info.BootTime), 0),
	}
}

func (c *Collector) hostname(fallback string) string {
	if c.hostRoot == "" {
		return fallback
	}
	data, err := os.ReadFile(filepath.Join(c.hostRoot, "etc", "hostname"))
	if err != nil {
		c.log.Warn("host hostname unreadable", "error", err)
		return fallback
	}
	if name := strings.TrimSpace(string(data)); name != "" {
		return name
	}
	return fallback
}

func (c *Collector) collectLoad(ctx context.Context) Load {
	avg, err := load.AvgWithContext(ctx)
	if err != nil {
		c.unavailable("load", err)
		return Load{}
	}
	return Load{
		Available: true,
		Load1:     avg.Load1,
		Load5:     avg.Load5,
		Load15:    avg.Load15,
	}
}
