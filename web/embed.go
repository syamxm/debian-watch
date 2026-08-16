// Package web holds the embedded templates and static assets so the compiled
// binary needs nothing from the filesystem at runtime.
package web

import "embed"

//go:embed templates static
var Files embed.FS
