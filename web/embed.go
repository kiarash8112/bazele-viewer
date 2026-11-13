// Package web holds the built frontend (web/dist), embedded into the viewer.
//
// Build it before `bazel run //cmd/viewer`:
//
//	npm --prefix web ci && npm --prefix web run build
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// FS returns the built frontend, rooted at its index.html.
func FS() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err) // "dist" is a fixed, valid path.
	}
	return sub
}
