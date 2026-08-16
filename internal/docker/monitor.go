package docker

import (
	"context"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxConcurrentStats = 8

type Container struct {
	Name        string
	Image       string
	State       string
	Health      string
	Uptime      time.Duration
	CPUPercent  float64
	MemoryUsage uint64
	MemoryLimit uint64
	StatsOK     bool
}

func (c Container) Running() bool { return c.State == "running" }

// State is the result of one refresh cycle. Enabled is false when no Docker
// host is configured, which disables the panel rather than reporting an error.
type State struct {
	Enabled    bool
	Available  bool
	Containers []Container
	UpdatedAt  time.Time
}

func (s State) RunningCount() int {
	count := 0
	for _, container := range s.Containers {
		if container.Running() {
			count++
		}
	}
	return count
}

type Monitor struct {
	client   *Client
	interval time.Duration
	log      *slog.Logger

	mu    sync.RWMutex
	state State
}

// NewMonitor returns a disabled monitor when host is empty, so a homeserver
// without the socket proxy still runs every other panel.
func NewMonitor(host string, interval time.Duration, log *slog.Logger) (*Monitor, error) {
	if host == "" {
		return &Monitor{log: log, interval: interval}, nil
	}
	client, err := NewClient(host)
	if err != nil {
		return nil, err
	}
	return &Monitor{
		client:   client,
		interval: interval,
		log:      log,
		state:    State{Enabled: true},
	}, nil
}

func (m *Monitor) Enabled() bool { return m.client != nil }

func (m *Monitor) State() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

func (m *Monitor) Run(ctx context.Context) {
	if !m.Enabled() {
		return
	}

	m.refresh(ctx)
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.refresh(ctx)
		}
	}
}

func (m *Monitor) refresh(ctx context.Context) {
	containers, err := m.collect(ctx)
	if err != nil {
		m.log.Warn("docker refresh failed", "error", err)
		m.mu.Lock()
		m.state = State{Enabled: true, Available: false, UpdatedAt: time.Now()}
		m.mu.Unlock()
		return
	}

	m.mu.Lock()
	m.state = State{Enabled: true, Available: true, Containers: containers, UpdatedAt: time.Now()}
	m.mu.Unlock()
}

func (m *Monitor) collect(ctx context.Context) ([]Container, error) {
	summaries, err := m.client.Containers(ctx)
	if err != nil {
		return nil, err
	}

	containers := make([]Container, len(summaries))
	semaphore := make(chan struct{}, maxConcurrentStats)
	var wg sync.WaitGroup

	for i, summary := range summaries {
		containers[i] = Container{
			Name:   containerName(summary.Names),
			Image:  summary.Image,
			State:  summary.State,
			Health: parseHealth(summary.Status),
			Uptime: uptimeFrom(summary.Status),
		}
		if summary.State != "running" {
			continue
		}

		wg.Add(1)
		go func(index int, id string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			stats, err := m.client.Stats(ctx, id)
			if err != nil {
				m.log.Debug("container stats unavailable", "container", containers[index].Name, "error", err)
				return
			}
			containers[index].CPUPercent = stats.cpuPercent()
			containers[index].MemoryUsage = stats.memoryUsage()
			containers[index].MemoryLimit = stats.Memory.Limit
			containers[index].StatsOK = true
		}(i, summary.ID)
	}
	wg.Wait()

	sort.Slice(containers, func(a, b int) bool {
		if containers[a].Running() != containers[b].Running() {
			return containers[a].Running()
		}
		return containers[a].Name < containers[b].Name
	})
	return containers, nil
}

func containerName(names []string) string {
	if len(names) == 0 {
		return "unknown"
	}
	return strings.TrimPrefix(names[0], "/")
}

// parseHealth reads the health suffix Docker appends to the status text, for
// example "Up 3 days (healthy)".
func parseHealth(status string) string {
	switch {
	case strings.Contains(status, "(healthy)"):
		return "healthy"
	case strings.Contains(status, "(unhealthy)"):
		return "unhealthy"
	case strings.Contains(status, "health: starting"):
		return "starting"
	default:
		return ""
	}
}

// uptimeFrom parses the "Up 3 days" form of the status text. Docker reports
// container age here as human text, so precision below the printed unit is
// not recoverable.
func uptimeFrom(status string) time.Duration {
	if !strings.HasPrefix(status, "Up ") {
		return 0
	}
	fields := strings.Fields(strings.TrimPrefix(status, "Up "))
	if len(fields) < 2 {
		return 0
	}

	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	switch {
	case strings.HasPrefix(fields[1], "second"):
		return time.Duration(value) * time.Second
	case strings.HasPrefix(fields[1], "minute"):
		return time.Duration(value) * time.Minute
	case strings.HasPrefix(fields[1], "hour"):
		return time.Duration(value) * time.Hour
	case strings.HasPrefix(fields[1], "day"):
		return time.Duration(value) * 24 * time.Hour
	case strings.HasPrefix(fields[1], "week"):
		return time.Duration(value) * 7 * 24 * time.Hour
	case strings.HasPrefix(fields[1], "month"):
		return time.Duration(value) * 30 * 24 * time.Hour
	case strings.HasPrefix(fields[1], "year"):
		return time.Duration(value) * 365 * 24 * time.Hour
	default:
		return 0
	}
}
