// Command artsheet writes one page listing every picture in assets/, grouped by the directory it
// lives in, each one drawn at its own size with its filename under it.
//
//	go run ./tools/artsheet
//
// # It is the only sheet that renders nothing
//
// Every other page under docs/sheets/ is a report on a catalog: it builds each record's card with
// cards.Render and writes a PNG per record, which is what makes those pages heavy and what makes
// them stale the moment a record is retuned. This one answers a different question — "what
// pictures does this repo actually hold, and what is each file called" — and the answer is already
// on disk. So it writes no image at all: every cell is an <img> pointing into assets/, and the
// page is a few tens of kilobytes of HTML over art that is committed once.
//
// That is also why it is not a substitute for the targeted sheets and links to them instead. A
// picture here is the raw file; a picture there is the card as it will be dealt, with type over
// it, which is the only way to judge whether art survives the card. Use this to find a file and
// that one to review it.
//
// # The categories come off the filesystem
//
// A directory under assets/ holding at least one image is a category, counted rather than typed
// in — the same rule a chip bar's values are under. `known` adds a label and a link to the sheet
// that reviews that catalog properly, and a directory missing from it still appears, under its own
// name with no link. So a new asset directory shows up here the day it is created.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/curiousjc/ascend-duel/tools/sheetfilter"
)

// imageExt is what counts as a picture. assets/ also holds fonts and a MIDI file, and a page of
// broken image icons would be worse than leaving them off.
var imageExt = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true}

// known is the label and the deeper review for each asset directory. **A directory missing from
// here is still drawn**, under its own name and with no link — the categories are counted off the
// filesystem, and this table only decorates them.
var known = map[string]struct {
	Label string
	Sheet string // the review sheet that draws these as cards, relative to docs/sheets/artsheet/
	Note  string
}{
	"card":    {"Playing cards", "../cardsheet/index.html", "The duelist's own deck."},
	"relic":   {"Relics", "../relicsheet/index.html", "Full-bleed card faces."},
	"essence": {"Essences", "../essencesheet/index.html", "Full-bleed card faces."},
	"rune":    {"Runes", "../runesheet/index.html", "Full-bleed card faces."},
	"stone":   {"Stones", "../stonesheet/index.html", "Full-bleed card faces."},
	"other":   {"Other cards", "../goodsheet/index.html", "Potions, sealed goods and the brand."},
	"enemy":   {"Creatures and bosses", "../motifsheet/index.html", "One picture per record per element."},
	"damage":  {"Damage badges", "../badgesheet/index.html", "The numeral is drawn into the badge."},
	"form":    {"Form marks and cost ticks", "../marksheet/index.html", "One per form per element, plus a neutral set."},
	"upgrade": {"Upgrade art", "../upgradesheet/index.html", "The one rider that keeps a picture for its ink."},
	"effect":  {"Status badges", "", "Drawn along the bottom of the enemy card."},
	"texture": {"Form materials", "", "Seamless tiles, kept inside the glyph shapes of a form word."},
	"game":    {"Interface", "", "The title screens and the cog."},
}

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "artsheet"),
		"directory to write index.html into")
	assets := flag.String("assets", "assets", "the directory of pictures to list")
	flag.Parse()

	if err := run(*dir, *assets); err != nil {
		log.Fatal(err)
	}
}

func run(dir, assets string) error {
	cats, err := categories(assets)
	if err != nil {
		return err
	}
	if len(cats) == 0 {
		return fmt.Errorf("no pictures found under %s", assets)
	}

	// The page lives at <dir>/index.html and the pictures stay where they are, so every src is a
	// walk back up to the repo root and down into assets/. Computed rather than written down: a
	// sheet written somewhere else with -dir still points at real files.
	rel, err := filepath.Rel(dir, assets)
	if err != nil {
		return fmt.Errorf("locating %s from %s: %w", assets, dir, err)
	}
	base := filepath.ToSlash(rel)

	for i := range cats {
		for j := range cats[i].Files {
			cats[i].Files[j].Src = path.Join(base, cats[i].Key, cats[i].Files[j].Name)
		}
	}

	var total int
	facet := sheetfilter.Facet{Key: "category", Label: "category"}
	for _, c := range cats {
		total += len(c.Files)
		facet.Values = append(facet.Values,
			sheetfilter.Value{Value: c.Key, Label: c.Label, Count: len(c.Files)})
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("making %s: %w", dir, err)
	}
	out := filepath.Join(dir, "index.html")
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	defer f.Close()

	p := page{
		Categories: cats,
		Total:      total,
		Filters:    sheetfilter.Bar([]sheetfilter.Facet{facet}),
	}
	if err := tmpl.Execute(f, p); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}

	fmt.Printf("%d pictures in %d categories\n", total, len(cats))
	fmt.Printf("wrote %s\n", out)
	return nil
}

// categories reads assets/ one directory deep. A directory with no picture in it is left off
// rather than drawn empty, which is the stone sheet's rule for a rung nobody authored a stone for.
func categories(assets string) ([]category, error) {
	ents, err := os.ReadDir(assets)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", assets, err)
	}

	var out []category
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		files, err := pictures(filepath.Join(assets, e.Name()))
		if err != nil {
			return nil, err
		}
		if len(files) == 0 {
			continue
		}
		c := category{Key: e.Name(), Label: e.Name(), Files: files}
		if k, ok := known[e.Name()]; ok {
			c.Label, c.Sheet, c.Note = k.Label, k.Sheet, k.Note
		}
		out = append(out, c)
	}

	// Biggest first: the question this page is opened with is usually about the catalog with the
	// most files in it, and a reader scrolling past thirteen headings alphabetically meets the
	// three-file ones first.
	sort.SliceStable(out, func(i, j int) bool { return len(out[i].Files) > len(out[j].Files) })
	return out, nil
}

func pictures(dir string) ([]file, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	var out []file
	for _, e := range ents {
		if e.IsDir() || !imageExt[strings.ToLower(filepath.Ext(e.Name()))] {
			continue
		}
		out = append(out, file{Name: e.Name(), Stem: strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

type category struct {
	Key   string
	Label string
	Sheet string
	Note  string
	Files []file
}

type file struct {
	Name string // the filename, which is what a data record's Art field writes
	Stem string
	Src  string // where the page reaches it, relative to the page
}
