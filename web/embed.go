// Package web holds the built frontend, embedded into the viewer.
//
// Bazel builds dist/ with Vite (the //web:build target) and passes it to the
// Go compiler through embedsrcs.
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
