// Command enemysheet renders every creature in data/enemies.json to a PNG and writes an HTML
// page showing each one beside the deck it fights with and the stats it fights on.
//
//	go run ./tools/enemysheet
//
// It exists because the roster is the least reviewable catalogue in the game: ninety-six records,
// met one at a time, three rooms to a floor, and a creature's whole personality is a deck the
// player only ever sees the played half of. Seeing whether floor five's creatures are actually
// dearer than floor four's meant reading JSON.
//
// Everything about how the page is built is in tools/roster, which the boss sheet shares — see
// that package's comment for why these two are one sheet over two pools rather than two sheets.
package main

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/curiousjc/ascend-duel/tools/roster"
)

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "enemysheet"),
		"directory to write the PNGs and index.html into")
	flag.Parse()

	if err := roster.Run(roster.EnemyPool, *dir); err != nil {
		log.Fatal(err)
	}
}
