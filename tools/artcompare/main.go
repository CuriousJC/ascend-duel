// Command artcompare draws every record of an art catalog once per candidate set of pictures and
// writes a page that puts them side by side, so a batch is judged record by record rather than
// all at once.
//
//	go run ./tools/artcompare                      # the card catalog, every set it finds
//	go run ./tools/artcompare -catalog relic
//	go run ./tools/artcompare -catalog relic -set warm=artreview/relic-warm -set cool=artreview/relic-cool
//
// # Why it exists
//
// A replacement batch is not one decision. Ninety-five pictures arrive together, some better than
// what they replace and some not, and the only way to see which is which is to look at them
// against each other — which is impossible the moment a batch is installed, because the picture it
// replaced is gone. **And a generator produces options**: two prompts, three temperatures, a
// palette tried both ways. So the page takes any number of sets, not two.
//
// # It cannot lie about the game
//
// Every cell is cards.Render at the catalog's own style — cards.Hand for a playing card,
// cards.RelicStyle for a relic, cards.EssenceStyle for the rest — which is the same call the
// game and the review sheets make, with the picture read off disk instead of out of the
// embedded bank. So what is compared is the card as it will be dealt, type and all: a playing
// card's picture is drawn under five pieces of near-black type and an essence's under the
// sentence it prints, and art that reads well bare can lose all of that there.
//
// Only Spec.Art differs between the columns of a row. Everything else is held constant, so
// anything the eye catches is the art.
//
// # The sets, and where they live
//
// A set is a label and a directory of PNGs keyed by filename stem — the same stems the catalog's
// `Art` fields already name, so a candidate batch is a directory of files named like the
// installed ones and nothing has to be renamed to review it.
//
//   - `current` is always the installed art — `assets/card`, `assets/relic` and so on — unless a
//     -set of that name replaces it.
//   - Every directory under `artreview/` named `<catalog>-<label>` is picked up as the set
//     `<label>`. So dropping a batch in `artreview/card-warm/` is the whole of the setup.
//   - -set label=dir, repeatable, replaces that discovery outright, for a directory kept
//     somewhere else.
//
// **`artreview/` is gitignored**, candidate batches and built page alike, which is what lets this
// be a committed tool: the code regenerates from a bare clone and the pictures it is arguing
// about do not belong in history. See .gitignore, where it is anchored for /trace/'s reason.
//
// # It is not one of the committed sheets
//
// Everything under docs/sheets/ is a report on the shipped catalog. This is a page about a
// decision that is being made once, over pictures that are not in the repo — so it writes into
// artreview/ and is deliberately absent from tools/sheets. The output directory is rebuilt on
// every run and can be deleted whenever.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/internal/cards"
)

// ground is the combat screen's background, so a card is judged against what it actually sits
// on. The literal value from combat.go's fill, exactly as tools/cardsheet quotes it. A light
// page flatters an off-white card enormously.
const ground = "#323232"

// reviewRoot is where candidate sets are looked for and the page is written. One ignored
// directory rather than several, so a clone has one thing to not have.
const reviewRoot = "artreview"

// installed is the label the shipped pictures are given. Named rather than positional, because a
// page of three columns has to say which one is the game as it stands.
const installed = "current"

// sets is the -set flag: repeatable, `label=dir`.
type sets []set

// set is one column of the page: what to call it and where its pictures are.
type set struct {
	Label string
	Dir   string
}

func (s *sets) String() string {
	var parts []string
	for _, one := range *s {
		parts = append(parts, one.Label+"="+one.Dir)
	}
	return strings.Join(parts, ",")
}

func (s *sets) Set(v string) error {
	label, dir, ok := strings.Cut(v, "=")
	if !ok || label == "" || dir == "" {
		return fmt.Errorf("want label=dir, got %q", v)
	}
	for _, one := range *s {
		if one.Label == label {
			return fmt.Errorf("the set %q is named twice", label)
		}
	}
	*s = append(*s, set{Label: label, Dir: filepath.Clean(dir)})
	return nil
}

func main() {
	var chosen sets
	name := flag.String("catalog", "card",
		"which art family to review: "+strings.Join(catalogNames(), ", "))
	flag.Var(&chosen, "set", "a set of pictures as label=dir, repeatable; replaces the sets found under "+reviewRoot)
	dir := flag.String("dir", "", "directory to write the PNGs and index.html into (default "+reviewRoot+"/out-<catalog>)")
	flag.Parse()

	cat, ok := catalogs[*name]
	if !ok {
		log.Fatalf("artcompare: no catalog %q; the vocabulary is %s",
			*name, strings.Join(catalogNames(), ", "))
	}
	// Cleaned so the installed directory is spelled the way every other path on the page is —
	// the copy list compares a set's directory against this one to know whether a pick is
	// already installed, and `assets/card` against `assets\card` is a comparison that fails
	// silently on Windows.
	cat.Dir = filepath.Clean(cat.Dir)

	out := *dir
	if out == "" {
		out = filepath.Join(reviewRoot, "out-"+cat.Name)
	}

	if err := run(cat, chosen, out); err != nil {
		log.Fatal(err)
	}
}

func run(cat *catalog, chosen sets, dir string) error {
	found, err := setsFor(cat, chosen)
	if err != nil {
		return err
	}
	if len(found) < 2 {
		// One column is not a comparison. It is still written — a page of the installed art
		// alone is a contact sheet, and saying so is more use than refusing to run.
		fmt.Printf("artcompare: only %d set to show; drop a batch in %s/%s-<label>/ for a comparison\n",
			len(found), reviewRoot, cat.Name)
	}

	subjects, err := cat.Subjects()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("making %s: %w", dir, err)
	}
	faces, err := cards.NewFaces(assets.LoadFontData()["kubasta"])
	if err != nil {
		return err
	}

	p := page{
		Ground: ground, Catalog: cat.Name, File: cat.File,
		Sets: found, Dir: cat.Dir,
	}
	for _, one := range found {
		p.Labels = append(p.Labels, one.Label)
	}

	// The open group is held by index rather than by pointer: appending to p.Groups can move the
	// slice's backing array, and a pointer into the old one would collect rows nobody ever sees.
	at := -1
	for _, sub := range subjects {
		if sub.Stem == "" {
			// A record with no Art names no picture, so there is nothing to compare. Counted
			// and listed at the foot of the page rather than drawn as identical bare cards —
			// that list is the catalog's own backlog, which is worth seeing here.
			p.Undrawn = append(p.Undrawn, sub.Key)
			continue
		}
		if at < 0 || p.Groups[at].Name != sub.Group {
			p.Groups = append(p.Groups, group{Name: sub.Group})
			at = len(p.Groups) - 1
		}

		row := row{Key: sub.Key, Stem: sub.Stem, Caption: sub.Caption}
		for _, one := range found {
			art, err := artOf(one.Dir, sub.Stem)
			if err != nil {
				return err
			}
			if art == nil {
				row.Cells = append(row.Cells, cell{Label: one.Label})
				continue
			}
			spec := sub.Spec
			spec.Art = art
			c, err := write(dir, faces, spec, sub.Style,
				sub.Stem+"-"+one.Label+".png", one.Label)
			if err != nil {
				return err
			}
			row.Cells = append(row.Cells, c)
		}
		p.Groups[at].Rows = append(p.Groups[at].Rows, row)
		p.Count++
	}

	f, err := os.Create(filepath.Join(dir, "index.html"))
	if err != nil {
		return fmt.Errorf("creating index.html: %w", err)
	}
	defer f.Close()
	if err := tmpl.Execute(f, p); err != nil {
		return fmt.Errorf("writing index.html: %w", err)
	}

	fmt.Printf("artcompare: %s — %d records across %d %s (%s), %d undrawn; wrote %s\n",
		cat.Name, p.Count, len(found), plural(len(found), "set"),
		strings.Join(p.Labels, ", "), len(p.Undrawn),
		filepath.Join(dir, "index.html"))
	return nil
}

// setsFor is the columns of the page: the installed art, then whatever candidates were found or
// named. A -set list replaces the discovery outright rather than adding to it, so a run can be
// narrowed to two of five batches without moving directories about.
func setsFor(cat *catalog, chosen sets) ([]set, error) {
	if len(chosen) > 0 {
		for _, one := range chosen {
			if _, err := os.Stat(one.Dir); err != nil {
				return nil, fmt.Errorf("the set %q: %w", one.Label, err)
			}
		}
		return chosen, nil
	}

	out := []set{{Label: installed, Dir: cat.Dir}}
	entries, err := os.ReadDir(reviewRoot)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", reviewRoot, err)
	}

	prefix := cat.Name + "-"
	var found []set
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		label := strings.TrimPrefix(e.Name(), prefix)
		if label == "" {
			continue
		}
		found = append(found, set{Label: label, Dir: filepath.Join(reviewRoot, e.Name())})
	}
	// Sorted, so two runs of the tool put the columns in the same order and the page can be
	// read against the last one. os.ReadDir is already sorted; this says it is meant.
	sort.Slice(found, func(i, j int) bool { return found[i].Label < found[j].Label })
	return append(out, found...), nil
}

// plural is the one line that keeps the summary readable when a run finds a single set.
func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// multiplier writes an attack's amount the way the card does: a fraction under one, a whole
// number at or above it. A snapshot of screens.damageMultiplier, as tools/cardsheet's copy is.
func multiplier(pct int) string {
	if pct%100 == 0 {
		return strconv.Itoa(pct / 100)
	}
	return strconv.FormatFloat(float64(pct)/100, 'g', -1, 64)
}

// artCache is keyed by path rather than by stem, since every set holds the same names and a
// cache of stems would hand one set's picture to another.
var artCache = map[string]image.Image{}

// artOf decodes one picture off disk, or nil when that set does not hold it. **Off disk rather
// than out of assets.LoadImageData**, which is the whole reason this tool works at all: only one
// of the sets is embedded in the binary, and after an install it is the wrong one.
func artOf(dir, stem string) (image.Image, error) {
	path := filepath.Join(dir, stem+".png")
	if img, seen := artCache[path]; seen {
		return img, nil
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		artCache[path] = nil
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	// **A picture that will not decode is reported and treated as absent**, rather than failing
	// the run. A candidate batch comes out of a generator and can hold a truncated or zero-byte
	// file; a tool that stopped on the first one would show none of the ninety-four that are
	// fine, and the empty cell on the page is how the bad file gets noticed at all.
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		fmt.Printf("artcompare: %s will not decode (%d bytes): %v\n", path, len(raw), err)
		artCache[path] = nil
		return nil, nil
	}
	artCache[path] = img
	return img, nil
}

func write(dir string, f *cards.Faces, s cards.Spec, st cards.Style, name, label string) (cell, error) {
	img, err := cards.Render(s, st, f)
	if err != nil {
		return cell{}, fmt.Errorf("rendering %s: %w", name, err)
	}
	out, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return cell{}, fmt.Errorf("creating %s: %w", name, err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		return cell{}, fmt.Errorf("encoding %s: %w", name, err)
	}
	return cell{
		File: name, Label: label,
		Width: st.Width + st.Bleed, Height: st.Height + st.Bleed,
	}, nil
}
