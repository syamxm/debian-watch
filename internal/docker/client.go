// Package docker talks to a read-only Docker socket proxy over HTTP. Only the
// two endpoints the dashboard needs are implemented, which keeps the
// dependency surface at the standard library.
package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const requestTimeout = 5 * time.Second

type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient accepts an http(s) URL pointing at the socket proxy. The raw
// Docker socket is deliberately not supported.
func NewClient(host string) (*Client, error) {
	parsed, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("invalid docker host: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("docker host must be an http(s) proxy URL, got %q", parsed.Scheme)
	}
	return &Client{
		baseURL: strings.TrimSuffix(host, "/"),
		http:    &http.Client{Timeout: requestTimeout},
	}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	return c.get(ctx, "/_ping", nil)
}

func (c *Client) Containers(ctx context.Context) ([]containerSummary, error) {
	var summaries []containerSummary
	if err := c.get(ctx, "/containers/json?all=1", &summaries); err != nil {
		return nil, err
	}
	return summaries, nil
}

func (c *Client) Stats(ctx context.Context, id string) (statsResponse, error) {
	var stats statsResponse
	// one-shot is deliberately not used: it zeroes precpu_stats, which would
	// make every CPU figure an average since container start.
	path := "/containers/" + url.PathEscape(id) + "/stats?stream=false"
	if err := c.get(ctx, path, &stats); err != nil {
		return statsResponse{}, err
	}
	return stats, nil
}

func (c *Client) get(ctx context.Context, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("docker api %s: unexpected status %d", path, resp.StatusCode)
	}
	if target == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

type containerSummary struct {
	ID      string   `json:"Id"`
	Names   []string `json:"Names"`
	Image   string   `json:"Image"`
	State   string   `json:"State"`
	Status  string   `json:"Status"`
	Created int64    `json:"Created"`
}

type statsResponse struct {
	CPU struct {
		Usage struct {
			Total uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs  uint64 `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPU struct {
		Usage struct {
			Total uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	Memory struct {
		Usage uint64            `json:"usage"`
		Limit uint64            `json:"limit"`
		Stats map[string]uint64 `json:"stats"`
	} `json:"memory_stats"`
}

// cpuPercent mirrors the calculation docker stats performs: the container's
// share of total CPU time over the sampling window, scaled by core count.
func (s statsResponse) cpuPercent() float64 {
	if s.PreCPU.SystemUsage == 0 {
		return 0
	}
	cpuDelta := float64(s.CPU.Usage.Total) - float64(s.PreCPU.Usage.Total)
	systemDelta := float64(s.CPU.SystemUsage) - float64(s.PreCPU.SystemUsage)
	if cpuDelta <= 0 || systemDelta <= 0 {
		return 0
	}
	cores := float64(s.CPU.OnlineCPUs)
	if cores == 0 {
		cores = 1
	}
	return cpuDelta / systemDelta * cores * 100
}

// memoryUsage subtracts the page cache the way the docker CLI does, so the
// figure matches what `docker stats` reports.
func (s statsResponse) memoryUsage() uint64 {
	usage := s.Memory.Usage
	if inactive, ok := s.Memory.Stats["inactive_file"]; ok && inactive < usage {
		return usage - inactive
	}
	if cache, ok := s.Memory.Stats["cache"]; ok && cache < usage {
		return usage - cache
	}
	return usage
}
