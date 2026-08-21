package collect

import (
	"context"
	"log/slog"

	"github.com/shirou/gopsutil/v4/cpu"
)

type CPU struct {
	Available    bool
	Model        string
	Cores        int
	TotalPercent float64
	PerCore      []float64
}

func (c *Collector) collectCPU(ctx context.Context) CPU {
	total, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil || len(total) == 0 {
		c.unavailable("cpu", err)
		return CPU{Model: c.cpuModel}
	}

	perCore, err := cpu.PercentWithContext(ctx, 0, true)
	if err != nil {
		c.unavailable("cpu.percore", err)
		perCore = nil
	}

	return CPU{
		Available:    true,
		Model:        c.cpuModel,
		Cores:        len(perCore),
		TotalPercent: total[0],
		PerCore:      perCore,
	}
}

func primeCPU(ctx context.Context) {
	_, _ = cpu.PercentWithContext(ctx, 0, false)
	_, _ = cpu.PercentWithContext(ctx, 0, true)
}

func readCPUModel(ctx context.Context, log *slog.Logger) string {
	info, err := cpu.InfoWithContext(ctx)
	if err != nil || len(info) == 0 {
		log.Warn("cpu model unavailable", "error", err)
		return "unknown"
	}
	return info[0].ModelName
}
