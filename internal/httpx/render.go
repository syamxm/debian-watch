package httpx

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

type Renderer struct {
	pages map[string]*template.Template
}

func NewRenderer(fsys fs.FS) (*Renderer, error) {
	pageFiles, err := fs.Glob(fsys, "templates/pages/*.html")
	if err != nil {
		return nil, err
	}
	if len(pageFiles) == 0 {
		return nil, fmt.Errorf("no page templates found")
	}

	pages := make(map[string]*template.Template, len(pageFiles))
	for _, file := range pageFiles {
		name := strings.TrimSuffix(path.Base(file), ".html")
		tmpl, err := template.New("base.html").Funcs(templateFuncs).
			ParseFS(fsys, "templates/base.html", file)
		if err != nil {
			return nil, fmt.Errorf("parse page %q: %w", name, err)
		}
		pages[name] = tmpl
	}
	return &Renderer{pages: pages}, nil
}

func (r *Renderer) Page(w http.ResponseWriter, status int, page string, data any) error {
	return r.execute(w, status, page, "base", data)
}

func (r *Renderer) Block(w http.ResponseWriter, status int, page, block string, data any) error {
	return r.execute(w, status, page, block, data)
}

func (r *Renderer) execute(w http.ResponseWriter, status int, page, block string, data any) error {
	tmpl, ok := r.pages[page]
	if !ok {
		return fmt.Errorf("unknown page %q", page)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, block, data); err != nil {
		return fmt.Errorf("render %s/%s: %w", page, block, err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := buf.WriteTo(w)
	return err
}

var templateFuncs = template.FuncMap{
	"pct":        formatPercent,
	"avg":        formatLoadAverage,
	"bytes":      formatBytes,
	"rate":       formatRate,
	"uptime":     formatUptime,
	"ts":         formatTimestamp,
	"temp":       formatTemperature,
	"level":      usageLevel,
	"tempLevel":  temperatureLevel,
	"meterStyle": meterStyle,
	"sparkline":  sparkline,
}

func formatPercent(value float64) string {
	return fmt.Sprintf("%.1f", value)
}

func formatLoadAverage(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func formatBytes(value uint64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	size := float64(value)
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	index := -1
	for size >= unit && index < len(units)-1 {
		size /= unit
		index++
	}
	return fmt.Sprintf("%.1f %s", size, units[index])
}

func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	parts := make([]string, 0, 3)
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 || (days > 0 && minutes > 0) {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	return strings.Join(parts, " ")
}

func formatTimestamp(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func usageLevel(percent float64) string {
	switch {
	case percent >= 90:
		return "hot"
	case percent >= 70:
		return "warm"
	default:
		return "cool"
	}
}

func formatRate(bytesPerSecond float64) string {
	if bytesPerSecond < 0 {
		bytesPerSecond = 0
	}
	return formatBytes(uint64(bytesPerSecond)) + "/s"
}

func formatTemperature(celsius float64) string {
	return fmt.Sprintf("%.1f°C", celsius)
}

func temperatureLevel(celsius, high float64) string {
	warn, crit := 65.0, 80.0
	if high > 0 {
		warn, crit = high*0.8, high
	}
	switch {
	case celsius >= crit:
		return "hot"
	case celsius >= warn:
		return "warm"
	default:
		return "cool"
	}
}

const sparklineWidth = 60

var sparklineRunes = []rune("▁▂▃▄▅▆▇█")

func sparkline(values []float64) string {
	if len(values) == 0 {
		return ""
	}
	if len(values) > sparklineWidth {
		values = values[len(values)-sparklineWidth:]
	}

	max := 0.0
	for _, value := range values {
		if value > max {
			max = value
		}
	}

	var builder strings.Builder
	builder.Grow(len(values) * 3)
	for _, value := range values {
		if max <= 0 {
			builder.WriteRune(sparklineRunes[0])
			continue
		}
		index := int(value / max * float64(len(sparklineRunes)-1))
		if index < 0 {
			index = 0
		}
		if index >= len(sparklineRunes) {
			index = len(sparklineRunes) - 1
		}
		builder.WriteRune(sparklineRunes[index])
	}
	return builder.String()
}

func meterStyle(percent float64) template.CSS {
	ratio := percent / 100
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return template.CSS(fmt.Sprintf("--v:%.4f", ratio))
}
