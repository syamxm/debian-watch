package monitor

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/syamxm/debian-watch/internal/collect"
	"github.com/syamxm/debian-watch/internal/docker"
	"github.com/syamxm/debian-watch/internal/history"
)

type View struct {
	Snapshot collect.Snapshot
	Docker   docker.State
	History  Series
}

type Series struct {
	CPU    []float64
	Memory []float64
	NetRx  []float64
	NetTx  []float64
}

type Monitor struct {
	collector *collect.Collector
	docker    *docker.Monitor
	interval  time.Duration
	log       *slog.Logger

	mu       sync.RWMutex
	snapshot collect.Snapshot

	cpu    *history.Ring
	memory *history.Ring
	netRx  *history.Ring
	netTx  *history.Ring
}

func New(
	collector *collect.Collector,
	dockerMonitor *docker.Monitor,
	interval time.Duration,
	historySize int,
	log *slog.Logger,
) *Monitor {
	return &Monitor{
		collector: collector,
		docker:    dockerMonitor,
		interval:  interval,
		log:       log,
		cpu:       history.NewRing(historySize),
		memory:    history.NewRing(historySize),
		netRx:     history.NewRing(historySize),
		netTx:     history.NewRing(historySize),
	}
}

func (m *Monitor) Run(ctx context.Context) {
	m.Sample(ctx)
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.Sample(ctx)
		}
	}
}

func (m *Monitor) View() View {
	m.mu.RLock()
	snapshot := m.snapshot
	m.mu.RUnlock()

	return View{
		Snapshot: snapshot,
		Docker:   m.docker.State(),
		History: Series{
			CPU:    m.cpu.Values(),
			Memory: m.memory.Values(),
			NetRx:  m.netRx.Values(),
			NetTx:  m.netTx.Values(),
		},
	}
}

func (m *Monitor) Sample(ctx context.Context) {
	snapshot := m.collector.Snapshot(ctx)

	m.mu.Lock()
	m.snapshot = snapshot
	m.mu.Unlock()

	if snapshot.CPU.Available {
		m.cpu.Add(snapshot.CPU.TotalPercent)
	}
	if snapshot.Memory.Available {
		m.memory.Add(snapshot.Memory.UsedPercent)
	}
	if snapshot.Network.Available {
		m.netRx.Add(snapshot.Network.TotalRx)
		m.netTx.Add(snapshot.Network.TotalTx)
	}
}
