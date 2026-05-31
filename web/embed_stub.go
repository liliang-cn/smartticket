//go:build !embedui

// Package web optionally embeds the compiled single-page frontend. Without the
// `embedui` build tag the UI is not bundled and the server runs API-only — this
// keeps `go build ./...` working without a prior frontend build. Build with
// `-tags embedui` (after `pnpm build`) for the single-binary deployment.
package web

import "io/fs"

// DistFS reports ok=false in API-only builds (no embedded frontend).
func DistFS() (fs.FS, bool) { return nil, false }
