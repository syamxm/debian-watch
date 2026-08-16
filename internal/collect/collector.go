// Package collect gathers host metrics. Every metric group degrades
// independently: an unavailable data source marks its group unavailable
// instead of failing the whole snapshot.
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

// Collector keeps the previous network sample to derive throughput, so it is
// not safe for concurrent use. The monitor calls it from a single goroutine.
type Collector struct {
	log      *slog.Logger
	cpuModel string
	lastNet  netSample
}

// New primes the CPU sampler so the first snapshot reports a real delta
// rather than usage since boot.
func New(ctx context.Context, log *slog.Logger) *Collector {
	c := &Collector{log: log}
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
