// Command motifsheet renders every record in data/motifs to a PNG and writes an HTML page
// showing each one beside the deck it fights with and the stats it fights on, motif by motif.
//
//	go run ./tools/motifsheet
//
// It exists because the roster is the least reviewable catalog in the game: a creature is met one
// at a time, three rooms to a floor, and its whole personality is a deck the player only ever
// sees the played half of. Whether one motif's three rooms read as a climb was a question
// answered by reading JSON.
//
// **It also carries the coverage grid**, which is the one thing about a motif that cannot be seen
// by looking at its records one at a time: a floor picks a motif and an element, so what has to
// hold is that every element can field all three rooms at least twice over. The page prints that
// grid per motif, out of the same function the loader refuses a file with.
//
// Everything about how the page is built is in tools/roster.
package main

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/curiousjc/ascend-duel/tools/roster"
)

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "motifsheet"),
		"directory to write the PNGs and index.html into")
	flag.Parse()

	if err := roster.Run(roster.MotifPool, *dir); err != nil {
		log.Fatal(err)
	}
}
