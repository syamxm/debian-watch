package collect

import (
	"bufio"
	"context"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const DefaultNetDevPath = "/proc/net/dev"

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

type netCounter struct {
	bytesRecv uint64
	bytesSent uint64
}

type netSample struct {
	counters map[string]netCounter
	taken    time.Time
}

func (c *Collector) collectNetwork(_ context.Context) Network {
	counters, err := readNetDev(c.netDevPath)
	if err != nil {
		c.unavailable("network", err)
		return Network{}
	}

	now := time.Now()
	network := Network{Available: true}
	for name, counter := range counters {
		iface := Interface{
			Name:      name,
			BytesRecv: counter.bytesRecv,
			BytesSent: counter.bytesSent,
		}
		if rx, tx, ok := c.rates(name, counter, now); ok {
			iface.RxPerSec = rx
			iface.TxPerSec = tx
			network.TotalRx += rx
			network.TotalTx += tx
		}
		network.Interfaces = append(network.Interfaces, iface)
	}

	c.lastNet = netSample{counters: counters, taken: now}
	sort.Slice(network.Interfaces, func(a, b int) bool {
		return network.Interfaces[a].Name < network.Interfaces[b].Name
	})
	return network
}

func (c *Collector) rates(name string, counter netCounter, now time.Time) (rx, tx float64, ok bool) {
	previous, exists := c.lastNet.counters[name]
	if !exists {
		return 0, 0, false
	}
	elapsed := now.Sub(c.lastNet.taken).Seconds()
	if elapsed <= 0 {
		return 0, 0, false
	}
	if counter.bytesRecv < previous.bytesRecv || counter.bytesSent < previous.bytesSent {
		return 0, 0, false
	}
	return float64(counter.bytesRecv-previous.bytesRecv) / elapsed,
		float64(counter.bytesSent-previous.bytesSent) / elapsed,
		true
}

func readNetDev(path string) (map[string]netCounter, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	counters := make(map[string]netCounter)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		name, values, ok := strings.Cut(scanner.Text(), ":")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if skipInterface(name) {
			continue
		}

		fields := strings.Fields(values)
		if len(fields) < 9 {
			continue
		}
		bytesRecv, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}
		bytesSent, err := strconv.ParseUint(fields[8], 10, 64)
		if err != nil {
			continue
		}
		counters[name] = netCounter{bytesRecv: bytesRecv, bytesSent: bytesSent}
	}
	return counters, scanner.Err()
}

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
