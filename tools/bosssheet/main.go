// Command bosssheet renders every stairway protector in data/bosses.json to a PNG and writes an
// HTML page showing each one beside the deck it fights with and the stats it fights on.
//
//	go run ./tools/bosssheet
//
// A separate page from the enemy sheet, for the reason `bosses.json` is a separate file from
// `enemies.json`: the two pools are drawn from differently and placed by different rules, and a
// boss is met once and remembered where a creature is one of ninety-six. Reading the thirty
// together is what shows whether the floor-by-floor step up is a curve or a staircase — a
// question a page holding both pools in one list would bury.
//
// Everything about how the page is built is in tools/roster, which the enemy sheet shares.
package main

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/curiousjc/ascend-duel/tools/roster"
)

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "bosssheet"),
		"directory to write the PNGs and index.html into")
	flag.Parse()

	if err := roster.Run(roster.BossPool, *dir); err != nil {
		log.Fatal(err)
	}
}
