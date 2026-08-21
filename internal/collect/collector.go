package collect

import (
	"context"
	"log/slog"
	"time"
)

type Snapshot struct {
	Time         time.Time
	CPU          CPU
	Memory       Memory
	Swap         Swap
	Host         Host
	Load         Load
	Disks        []Disk
	Network      Network
	Temperatures Temperatures
}

type Collector struct {
	log        *slog.Logger
	cpuModel   string
	hostRoot   string
	netDevPath string
	lastNet    netSample
}

func New(ctx context.Context, log *slog.Logger, hostRoot, netDevPath string) *Collector {
	if netDevPath == "" {
		netDevPath = DefaultNetDevPath
	}
	c := &Collector{log: log, hostRoot: hostRoot, netDevPath: netDevPath}
	c.cpuModel = readCPUModel(ctx, log)
	primeCPU(ctx)
	return c
}

func (c *Collector) Snapshot(ctx context.Context) Snapshot {
	return Snapshot{
		Time:         time.Now(),
		CPU:          c.collectCPU(ctx),
		Memory:       c.collectMemory(ctx),
		Swap:         c.collectSwap(ctx),
		Host:         c.collectHost(ctx),
		Load:         c.collectLoad(ctx),
		Disks:        c.collectDisks(ctx),
		Network:      c.collectNetwork(ctx),
		Temperatures: c.collectTemperatures(ctx),
	}
}

func (c *Collector) unavailable(group string, err error) {
	c.log.Warn("metric group unavailable", "group", group, "error", err)
}
