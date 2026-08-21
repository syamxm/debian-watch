package docker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientRejectsNonHTTPHosts(t *testing.T) {
	for _, host := range []string{"unix:///var/run/docker.sock", "tcp://docker:2375", "/var/run/docker.sock"} {
		if _, err := NewClient(host); err == nil {
			t.Errorf("NewClient(%q) should be rejected", host)
		}
	}
}

func TestContainersDecodesSummaries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/containers/json" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"Id":"abc","Names":["/vaultwarden"],"Image":"vaultwarden/server","State":"running","Status":"Up 3 days (healthy)"},
			{"Id":"def","Names":["/old-job"],"Image":"busybox","State":"exited","Status":"Exited (0) 2 hours ago"}
		]`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	summaries, err := client.Containers(context.Background())
	if err != nil {
		t.Fatalf("containers: %v", err)
	}

	if len(summaries) != 2 {
		t.Fatalf("got %d summaries, want 2", len(summaries))
	}
	if name := containerName(summaries[0].Names); name != "vaultwarden" {
		t.Errorf("name = %q, want vaultwarden", name)
	}
}

func TestGetReportsNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if _, err := client.Containers(context.Background()); err == nil {
		t.Fatal("expected an error for a 403 response")
	}
}

func TestCPUPercent(t *testing.T) {
	var stats statsResponse
	stats.CPU.Usage.Total = 200
	stats.CPU.SystemUsage = 1200
	stats.CPU.OnlineCPUs = 4
	stats.PreCPU.Usage.Total = 100
	stats.PreCPU.SystemUsage = 1000

	if got := stats.cpuPercent(); got != 200 {
		t.Errorf("cpuPercent() = %v, want 200", got)
	}
}

func TestCPUPercentHandlesFirstSample(t *testing.T) {
	var stats statsResponse
	stats.CPU.Usage.Total = 100
	stats.CPU.SystemUsage = 1000

	if got := stats.cpuPercent(); got != 0 {
		t.Errorf("cpuPercent() = %v, want 0 when there is no previous sample", got)
	}
}

func TestMemoryUsageSubtractsPageCache(t *testing.T) {
	var stats statsResponse
	stats.Memory.Usage = 1000
	stats.Memory.Stats = map[string]uint64{"inactive_file": 400}

	if got := stats.memoryUsage(); got != 600 {
		t.Errorf("memoryUsage() = %d, want 600", got)
	}
}

func TestMemoryUsageWithoutCacheStats(t *testing.T) {
	var stats statsResponse
	stats.Memory.Usage = 1000

	if got := stats.memoryUsage(); got != 1000 {
		t.Errorf("memoryUsage() = %d, want 1000", got)
	}
}
