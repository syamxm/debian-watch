package httpx

import (
	"testing"
	"time"

	"github.com/syamxm/debian-watch/web"
)

func TestFormatBytes(t *testing.T) {
	cases := map[uint64]string{
		0:                "0 B",
		512:              "512 B",
		1024:             "1.0 KiB",
		1536:             "1.5 KiB",
		16 * 1024 * 1024: "16.0 MiB",
		8 << 30:          "8.0 GiB",
	}
	for input, want := range cases {
		if got := formatBytes(input); got != want {
			t.Errorf("formatBytes(%d) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatUptime(t *testing.T) {
	cases := map[time.Duration]string{
		90 * time.Minute:              "1h 30m",
		49*time.Hour + 5*time.Minute:  "2d 1h 5m",
		2*time.Hour + 121*time.Minute: "4h 1m",
		13 * 24 * time.Hour:           "13d",
		25 * time.Hour:                "1d 1h",
		0:                             "0m",
		30 * time.Second:              "0m",
	}
	for input, want := range cases {
		if got := formatUptime(input); got != want {
			t.Errorf("formatUptime(%s) = %q, want %q", input, got, want)
		}
	}
}

func TestUsageLevel(t *testing.T) {
	cases := map[float64]string{0: "cool", 69.9: "cool", 70: "warm", 89.9: "warm", 90: "hot", 100: "hot"}
	for input, want := range cases {
		if got := usageLevel(input); got != want {
			t.Errorf("usageLevel(%v) = %q, want %q", input, got, want)
		}
	}
}

func TestMeterStyleClamps(t *testing.T) {
	cases := map[float64]string{-5: "--v:0.0000", 42.5: "--v:0.4250", 150: "--v:1.0000"}
	for input, want := range cases {
		if got := string(meterStyle(input)); got != want {
			t.Errorf("meterStyle(%v) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatRate(t *testing.T) {
	cases := map[float64]string{0: "0 B/s", 2048: "2.0 KiB/s", -5: "0 B/s"}
	for input, want := range cases {
		if got := formatRate(input); got != want {
			t.Errorf("formatRate(%v) = %q, want %q", input, got, want)
		}
	}
}

func TestTemperatureLevel(t *testing.T) {
	cases := []struct {
		celsius, high float64
		want          string
	}{
		{40, 0, "cool"},
		{70, 0, "warm"},
		{85, 0, "hot"},
		{70, 100, "cool"},
		{85, 100, "warm"},
		{100, 100, "hot"},
	}
	for _, tc := range cases {
		if got := temperatureLevel(tc.celsius, tc.high); got != tc.want {
			t.Errorf("temperatureLevel(%v, %v) = %q, want %q", tc.celsius, tc.high, got, tc.want)
		}
	}
}

func TestSparkline(t *testing.T) {
	if got := sparkline(nil); got != "" {
		t.Errorf("sparkline(nil) = %q, want empty", got)
	}
	if got := sparkline([]float64{0, 0, 0}); got != "▁▁▁" {
		t.Errorf("sparkline(zeros) = %q, want ▁▁▁", got)
	}
	if got := sparkline([]float64{0, 50, 100}); got != "▁▄█" {
		t.Errorf("sparkline(ramp) = %q, want ▁▄█", got)
	}
}

func TestSparklineTruncatesToWidth(t *testing.T) {
	values := make([]float64, sparklineWidth+40)
	for i := range values {
		values[i] = float64(i)
	}

	got := []rune(sparkline(values))
	if len(got) != sparklineWidth {
		t.Fatalf("sparkline width = %d, want %d", len(got), sparklineWidth)
	}
	if got[len(got)-1] != '█' {
		t.Error("the newest sample should be the tallest in a rising series")
	}
}

func TestRendererParsesEveryPage(t *testing.T) {
	renderer, err := NewRenderer(web.Files)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	for _, page := range []string{"signin", "dashboard", "docker", "memory", "system"} {
		if _, ok := renderer.pages[page]; !ok {
			t.Errorf("page %q not parsed", page)
		}
	}
}
