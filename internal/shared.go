package main

import (
	"embed"
	"path"
)

// go: embed assets
var assetFs embed.FS

var fbFont = path.Join("assets", "Roboto-Regular.ttf")
