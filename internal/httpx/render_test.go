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

func TestRendererParsesEveryPage(t *testing.T) {
	renderer, err := NewRenderer(web.Files)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	for _, page := range []string{"signin", "dashboard", "memory", "system"} {
		if _, ok := renderer.pages[page]; !ok {
			t.Errorf("page %q not parsed", page)
		}
	}
}
