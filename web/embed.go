//go:build embedui

// Package web optionally embeds the compiled single-page frontend (web/dist)
// into the server binary. It is built with the `embedui` tag (after the
// frontend has been built with `pnpm build`) to produce a true single-binary
// deployment that serves both the API and the console from one process.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// DistFS returns the embedded frontend rooted at its top level (so "index.html"
// and "assets/..." resolve directly). It reports ok=false when the build did
// not bundle a compiled frontend.
func DistFS() (fs.FS, bool) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, false
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, false
	}
	return sub, true
}
