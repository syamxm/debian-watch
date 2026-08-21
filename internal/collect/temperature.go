package collect

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/shirou/gopsutil/v4/sensors"
)

type Temperatures struct {
	Available bool
	Sensors   []Sensor
}

type Sensor struct {
	Label   string
	Detail  string
	Celsius float64
	High    float64
}

// sensorLabels maps hwmon driver names to what the hardware actually is.
// Order matters: the first matching prefix wins.
var sensorLabels = []struct{ prefix, label string }{
	{"k10temp", "CPU"},
	{"zenpower", "CPU"},
	{"coretemp", "CPU"},
	{"cpu_thermal", "CPU"},
	{"cpu-thermal", "CPU"},
	{"amdgpu", "Integrated Graphics"},
	{"radeon", "Integrated Graphics"},
	{"i915", "Integrated Graphics"},
	{"intel_gpu", "Integrated Graphics"},
	{"nvidia", "GPU"},
	{"nouveau", "GPU"},
	{"nvme", "NVMe SSD"},
	{"drivetemp", "Disk"},
	{"acpitz", "Motherboard"},
	{"pch", "Chipset"},
	{"iwlwifi", "Wi-Fi"},
	{"mt7921", "Wi-Fi"},
	{"ath", "Wi-Fi"},
	{"bat", "Battery"},
}

func (c *Collector) collectTemperatures(ctx context.Context) Temperatures {
	readings, err := sensors.TemperaturesWithContext(ctx)
	if err != nil && len(readings) == 0 {
		c.unavailable("temperature", err)
		return Temperatures{}
	}

	temps := Temperatures{Available: true}
	for _, reading := range readings {
		if reading.Temperature <= 0 {
			continue
		}
		detail := strings.TrimSuffix(reading.SensorKey, "_input")
		if redundantSensor(detail) {
			continue
		}
		temps.Sensors = append(temps.Sensors, Sensor{
			Label:   friendlySensorLabel(detail),
			Detail:  detail,
			Celsius: reading.Temperature,
			High:    reading.High,
		})
	}
	if len(temps.Sensors) == 0 {
		return Temperatures{}
	}

	sort.Slice(temps.Sensors, func(a, b int) bool {
		if temps.Sensors[a].Label != temps.Sensors[b].Label {
			return temps.Sensors[a].Label < temps.Sensors[b].Label
		}
		return temps.Sensors[a].Detail < temps.Sensors[b].Detail
	})
	numberDuplicates(temps.Sensors)
	return temps
}

// redundantSensor drops probes that duplicate a headline reading. NVMe drives
// expose per-die sensors alongside Composite, which is the figure every other
// tool reports.
func redundantSensor(sensorKey string) bool {
	return strings.HasPrefix(sensorKey, "nvme_sensor_")
}

func friendlySensorLabel(sensorKey string) string {
	key := strings.ToLower(sensorKey)
	for _, mapping := range sensorLabels {
		if strings.HasPrefix(key, mapping.prefix) {
			return mapping.label
		}
	}
	return sensorKey
}

// numberDuplicates disambiguates hardware that reports the same friendly name,
// such as two NVMe drives, so the panel does not show identical rows.
func numberDuplicates(list []Sensor) {
	counts := make(map[string]int, len(list))
	for _, sensor := range list {
		counts[sensor.Label]++
	}

	seen := make(map[string]int, len(counts))
	for i, sensor := range list {
		if counts[sensor.Label] < 2 {
			continue
		}
		seen[sensor.Label]++
		list[i].Label = fmt.Sprintf("%s %d", sensor.Label, seen[sensor.Label])
	}
}
