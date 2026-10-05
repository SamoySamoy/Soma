// Package web embeds the built single-page app (web/dist) into the binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist returns the built app, rooted at web/dist. Before the first
// `go tool task web:build` it holds only a placeholder and the server
// explains how to build it.
func Dist() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		// fs.Sub only fails for an invalid path, and "dist" is a constant.
		panic(err)
	}
	return sub
}
