package collect

import (
	"context"
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
	Celsius float64
	High    float64
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
		temps.Sensors = append(temps.Sensors, Sensor{
			Label:   strings.TrimSuffix(reading.SensorKey, "_input"),
			Celsius: reading.Temperature,
			High:    reading.High,
		})
	}
	if len(temps.Sensors) == 0 {
		return Temperatures{}
	}

	sort.Slice(temps.Sensors, func(a, b int) bool { return temps.Sensors[a].Label < temps.Sensors[b].Label })
	return temps
}
