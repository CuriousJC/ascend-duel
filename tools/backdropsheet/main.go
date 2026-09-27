// Command backdropsheet writes the backdrop sheet: every room a motif's fights are drawn in front
// of, once per element it is painted in, beside the brief it was painted from.
//
//	go run ./tools/backdropsheet
//
// A room is met one fight at a time and only in the element the realm dealt, so seeing a room's
// five pictures in a launched game is five realms of luck. The motif report counts which are
// painted and draws none of them; this is where they are looked at.
//
// **A thumbnail is committed, never the picture.** A backdrop is a 1920x1080 JPEG and the sheet
// would carry every one of them a second time; a reduction to a third of the width is what the
// page needs to judge a room, and each thumbnail links to the full picture under `assets/` for the
// rest. Each run deletes every thumbnail no longer on the page.
//
// **An unpainted picture is drawn as the default backdrop, marked**, because that is what the fight
// actually shows — a blank cell would be a page disagreeing with the game.
//
// Every room and every picture is taken off `data.LoadMotifs` and off the files under
// `assets/motifs/`, found by stem exactly as `assets.embedTree` keys them.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/tools/sheetfilter"
)

// thumbW and thumbH are the committed thumbnail: a third of the canvas each way, an integer
// reduction of the 1920x1080 the prompt asks for.
const thumbW, thumbH = 640, 360

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "backdropsheet"), "directory to write the page and thumbnails into")
	art := flag.String("assets", filepath.Join("assets", "motifs"), "the tree the motif pictures are filed under")
	flag.Parse()

	if err := run(*dir, *art); err != nil {
		log.Fatal(err)
	}
}

func run(dir, art string) error {
	files, err := pictures(art)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("making %s: %w", dir, err)
	}

	// Every thumbnail this run writes, so the ones it did not can be swept afterwards.
	written := map[string]bool{}
	thumb := func(key string) (string, error) {
		name := key + ".jpg"
		if written[name] {
			return name, nil
		}
		if err := reduce(files[key], filepath.Join(dir, name)); err != nil {
			return "", err
		}
		written[name] = true
		return name, nil
	}

	fallback, ok := files[data.DefaultBackgroundArt]
	if !ok {
		return fmt.Errorf("%s: no %s picture to draw an unpainted room as", art, data.DefaultBackgroundArt)
	}
	p := page{FallbackLink: link(dir, fallback)}
	motifs := data.LoadMotifs()
	for _, key := range data.MotifOrder(motifs) {
		m := motifs[key]
		if len(m.Backdrops) == 0 {
			p.Bare = append(p.Bare, m.Name)
			continue
		}
		for _, b := range m.Backdrops {
			r := room{Motif: m.Motif, MotifName: m.Name, Backdrop: b.Backdrop, Name: b.Name,
				Tier: b.Tier, TierLabel: tierLabel[b.Tier], Draw: b.Draw}
			for _, e := range b.Affinities {
				c := cell{Element: e, Key: b.ArtKey(e), Direction: b.ElementDraw[e]}
				if path, ok := files[c.Key]; ok {
					if c.Thumb, err = thumb(c.Key); err != nil {
						return err
					}
					c.Link, c.Painted = link(dir, path), true
				} else {
					// The default's thumbnail is written only when a cell draws it.
					if c.Thumb, err = thumb(data.DefaultBackgroundArt); err != nil {
						return err
					}
					c.Link = p.FallbackLink
				}
				r.Cells = append(r.Cells, c)
			}
			p.Rooms = append(p.Rooms, r)
		}
	}
	p.Filters = sheetfilter.Bar(facets(p.Rooms))

	if err := sweep(dir, written); err != nil {
		return err
	}

	out := filepath.Join(dir, "index.html")
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	defer f.Close()
	if err := tmpl.Execute(f, p); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}

	var painted, of int
	for _, r := range p.Rooms {
		for _, c := range r.Cells {
			of++
			if c.Painted {
				painted++
			}
		}
	}
	fmt.Printf("%d rooms, %d/%d pictures painted\nwrote %s\n", len(p.Rooms), painted, of, out)
	return nil
}

// pictures is every file under the tree, by the key the game looks it up by.
func pictures(root string) (map[string]string, error) {
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := d.Name()
		out[strings.TrimSuffix(name, filepath.Ext(name))] = p
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", root, err)
	}
	return out, nil
}

// reduce writes a picture out at thumbnail size.
func reduce(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	img, _, err := image.Decode(in)
	if err != nil {
		return fmt.Errorf("decoding %s: %w", src, err)
	}
	small := image.NewRGBA(image.Rect(0, 0, thumbW, thumbH))
	xdraw.CatmullRom.Scale(small, small.Bounds(), img, img.Bounds(), xdraw.Src, nil)

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	return jpeg.Encode(out, small, &jpeg.Options{Quality: 85})
}

// sweep deletes every thumbnail this run did not write: a room renamed or removed leaves nothing
// behind to be committed.
func sweep(dir string, keep map[string]bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jpg") && !keep[e.Name()] {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// link is the full picture, relative to the page, so a click in a clone opens the file itself.
func link(dir, path string) string {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

// tierLabel is what the page calls a tier: the room, which is what the owner calls it.
var tierLabel = map[string]string{
	data.TierOuter: "outer chamber",
	data.TierInner: "inner chamber",
	data.TierBoss:  "stairway",
}

// facets cuts the page on motif, tier, element and whether a picture is painted. Every value and
// count is read off the rooms, per sheetfilter's rule.
func facets(rooms []room) []sheetfilter.Facet {
	motif := sheetfilter.Facet{Key: "motif", Label: "motif"}
	tier := sheetfilter.Facet{Key: "tier", Label: "room"}
	element := sheetfilter.Facet{Key: "element", Label: "element"}
	state := sheetfilter.Facet{Key: "state", Label: "picture"}

	motifs, tiers := map[string]int{}, map[string]int{}
	elements, states := map[string]int{}, map[string]int{}
	var motifOrder []string
	for _, r := range rooms {
		if motifs[r.Motif] == 0 {
			motifOrder = append(motifOrder, r.Motif)
		}
		motifs[r.Motif]++
		tiers[r.Tier]++
		for _, c := range r.Cells {
			elements[c.Element]++
			states[c.State()]++
		}
	}
	for _, m := range motifOrder {
		motif.Values = append(motif.Values, sheetfilter.Value{Value: m, Count: motifs[m]})
	}
	for _, t := range data.TierOrder {
		if n := tiers[t]; n > 0 {
			tier.Values = append(tier.Values, sheetfilter.Value{Value: t, Label: tierLabel[t], Count: n})
		}
	}
	for _, e := range data.AffinityElements {
		if n := elements[e]; n > 0 {
			element.Values = append(element.Values, sheetfilter.Value{Value: e, Count: n})
		}
	}
	for _, s := range []string{"painted", "unpainted"} {
		if n := states[s]; n > 0 {
			state.Values = append(state.Values, sheetfilter.Value{Value: s, Count: n})
		}
	}

	var out []sheetfilter.Facet
	for _, f := range []sheetfilter.Facet{motif, tier, element, state} {
		if len(f.Values) > 0 {
			out = append(out, f)
		}
	}
	return out
}
