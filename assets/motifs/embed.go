package motifs

// embed.go — the tree, and the one call that files it into assets.LoadImageData.

import (
	"embed"

	"github.com/curiousjc/ascend-duel/assets"
)

// tree is the backdrop placeholder at the top and every motif's directory under it.
//
//go:embed *.jpg */*
var tree embed.FS

func init() {
	assets.RegisterImages(func(images map[string][]byte) {
		assets.EmbedTree(images, tree, ".")
	})
}
