package collect

import (
	"context"

	"github.com/shirou/gopsutil/v4/mem"
)

type Memory struct {
	Available   bool
	Total       uint64
	Used        uint64
	Free        uint64
	Cached      uint64
	Buffers     uint64
	UsedPercent float64
}

type Swap struct {
	Available   bool
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
}

func (c *Collector) collectMemory(ctx context.Context) Memory {
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		c.unavailable("memory", err)
		return Memory{}
	}
	return Memory{
		Available:   true,
		Total:       v.Total,
		Used:        v.Used,
		Free:        v.Available,
		Cached:      v.Cached,
		Buffers:     v.Buffers,
		UsedPercent: v.UsedPercent,
	}
}

func (c *Collector) collectSwap(ctx context.Context) Swap {
	s, err := mem.SwapMemoryWithContext(ctx)
	if err != nil {
		c.unavailable("swap", err)
		return Swap{}
	}
	return Swap{
		Available:   true,
		Total:       s.Total,
		Used:        s.Used,
		Free:        s.Free,
		UsedPercent: s.UsedPercent,
	}
}
