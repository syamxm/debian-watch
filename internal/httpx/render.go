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

// Renderer holds one template set per page, each built from the shared base
// plus that page's own blocks.
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

// execute renders into a buffer first so a template failure never emits a
// partially written response with a success status.
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
	"uptime":     formatUptime,
	"ts":         formatTimestamp,
	"level":      usageLevel,
	"meterStyle": meterStyle,
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
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
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

// meterStyle returns trusted CSS so the custom property survives the
// html/template style sanitiser.
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
