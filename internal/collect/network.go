package collect

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/net"
)

type Network struct {
	Available  bool
	Interfaces []Interface
	TotalRx    float64
	TotalTx    float64
}

type Interface struct {
	Name      string
	BytesRecv uint64
	BytesSent uint64
	RxPerSec  float64
	TxPerSec  float64
}

type netSample struct {
	counters map[string]net.IOCountersStat
	taken    time.Time
}

func (c *Collector) collectNetwork(ctx context.Context) Network {
	counters, err := net.IOCountersWithContext(ctx, true)
	if err != nil {
		c.unavailable("network", err)
		return Network{}
	}

	now := time.Now()
	current := netSample{counters: make(map[string]net.IOCountersStat, len(counters)), taken: now}
	network := Network{Available: true}

	for _, counter := range counters {
		if skipInterface(counter.Name) {
			continue
		}
		current.counters[counter.Name] = counter

		iface := Interface{
			Name:      counter.Name,
			BytesRecv: counter.BytesRecv,
			BytesSent: counter.BytesSent,
		}
		if rx, tx, ok := c.rates(counter, now); ok {
			iface.RxPerSec = rx
			iface.TxPerSec = tx
			network.TotalRx += rx
			network.TotalTx += tx
		}
		network.Interfaces = append(network.Interfaces, iface)
	}

	c.lastNet = current
	sort.Slice(network.Interfaces, func(a, b int) bool {
		return network.Interfaces[a].Name < network.Interfaces[b].Name
	})
	return network
}

// rates converts cumulative counters into per-second throughput using the
// previous sample. Counter resets, which happen when an interface is
// recreated, are reported as zero rather than a spike.
func (c *Collector) rates(counter net.IOCountersStat, now time.Time) (rx, tx float64, ok bool) {
	previous, exists := c.lastNet.counters[counter.Name]
	if !exists {
		return 0, 0, false
	}
	elapsed := now.Sub(c.lastNet.taken).Seconds()
	if elapsed <= 0 {
		return 0, 0, false
	}
	if counter.BytesRecv < previous.BytesRecv || counter.BytesSent < previous.BytesSent {
		return 0, 0, false
	}
	rx = float64(counter.BytesRecv-previous.BytesRecv) / elapsed
	tx = float64(counter.BytesSent-previous.BytesSent) / elapsed
	return rx, tx, true
}

// skipInterface drops the loopback and the per-container veth pairs, which on
// a host running dozens of containers would otherwise dominate the list.
func skipInterface(name string) bool {
	if name == "lo" {
		return true
	}
	for _, prefix := range []string{"veth", "br-", "docker", "tun", "tap"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
