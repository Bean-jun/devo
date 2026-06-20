package web

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var Dist embed.FS

func StaticFS() (fs.FS, error) {
	return fs.Sub(Dist, "dist")
}
