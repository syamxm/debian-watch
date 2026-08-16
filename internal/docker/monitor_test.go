package docker

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestParseHealth(t *testing.T) {
	cases := map[string]string{
		"Up 3 days (healthy)":             "healthy",
		"Up 2 minutes (unhealthy)":        "unhealthy",
		"Up 5 seconds (health: starting)": "starting",
		"Up 6 weeks":                      "",
		"Exited (0) 2 hours ago":          "",
	}
	for status, want := range cases {
		if got := parseHealth(status); got != want {
			t.Errorf("parseHealth(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestUptimeFrom(t *testing.T) {
	cases := map[string]time.Duration{
		"Up 45 seconds":        45 * time.Second,
		"Up 30 minutes":        30 * time.Minute,
		"Up 3 hours (healthy)": 3 * time.Hour,
		"Up 6 days":            6 * 24 * time.Hour,
		"Up 2 weeks":           14 * 24 * time.Hour,
		"Exited (0) ago":       0,
		"Up About an hour":     0,
	}
	for status, want := range cases {
		if got := uptimeFrom(status); got != want {
			t.Errorf("uptimeFrom(%q) = %s, want %s", status, got, want)
		}
	}
}

func TestMonitorDisabledWithoutHost(t *testing.T) {
	monitor, err := NewMonitor("", time.Second, discardLogger())
	if err != nil {
		t.Fatalf("new monitor: %v", err)
	}
	if monitor.Enabled() {
		t.Fatal("monitor must be disabled when no host is configured")
	}

	state := monitor.State()
	if state.Enabled || state.Available {
		t.Errorf("state = %+v, want disabled and unavailable", state)
	}
}

func TestMonitorMarksUnavailableWhenProxyFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	monitor, err := NewMonitor(server.URL, time.Second, discardLogger())
	if err != nil {
		t.Fatalf("new monitor: %v", err)
	}
	monitor.refresh(context.Background())

	state := monitor.State()
	if !state.Enabled {
		t.Error("state.Enabled = false, want true")
	}
	if state.Available {
		t.Error("state.Available = true, want false when the proxy errors")
	}
}

func TestMonitorCollectsContainersAndStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/containers/json":
			_, _ = w.Write([]byte(`[
				{"Id":"abc","Names":["/running-one"],"Image":"nginx","State":"running","Status":"Up 3 days (healthy)"},
				{"Id":"def","Names":["/stopped-one"],"Image":"busybox","State":"exited","Status":"Exited (0) 2 hours ago"}
			]`))
		case strings.HasSuffix(r.URL.Path, "/stats"):
			_, _ = w.Write([]byte(`{
				"cpu_stats":{"cpu_usage":{"total_usage":200},"system_cpu_usage":1200,"online_cpus":2},
				"precpu_stats":{"cpu_usage":{"total_usage":100},"system_cpu_usage":1000},
				"memory_stats":{"usage":1000,"limit":4000,"stats":{"inactive_file":200}}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	monitor, err := NewMonitor(server.URL, time.Second, discardLogger())
	if err != nil {
		t.Fatalf("new monitor: %v", err)
	}
	monitor.refresh(context.Background())

	state := monitor.State()
	if !state.Available {
		t.Fatal("state.Available = false, want true")
	}
	if len(state.Containers) != 2 {
		t.Fatalf("got %d containers, want 2", len(state.Containers))
	}
	if state.RunningCount() != 1 {
		t.Errorf("RunningCount() = %d, want 1", state.RunningCount())
	}

	running := state.Containers[0]
	if running.Name != "running-one" {
		t.Fatalf("first container = %q, want running-one (running sorts first)", running.Name)
	}
	if !running.StatsOK {
		t.Fatal("expected stats for the running container")
	}
	if running.CPUPercent != 100 {
		t.Errorf("CPUPercent = %v, want 100", running.CPUPercent)
	}
	if running.MemoryUsage != 800 {
		t.Errorf("MemoryUsage = %d, want 800", running.MemoryUsage)
	}
	if running.Health != "healthy" {
		t.Errorf("Health = %q, want healthy", running.Health)
	}

	stopped := state.Containers[1]
	if stopped.StatsOK {
		t.Error("stopped containers must not be queried for stats")
	}
}
