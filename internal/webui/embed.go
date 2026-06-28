// Package webui embeds the built React SPA (Vite output in dist/). When the SPA
// has not been built, FS reports ok=false and the server serves a placeholder.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// FS returns the embedded SPA filesystem and whether a real build is present.
func FS() (fs.FS, bool) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, false
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, false
	}
	return sub, true
}
