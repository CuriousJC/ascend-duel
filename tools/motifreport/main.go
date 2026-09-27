// Command motifreport writes the motif report: how full each motif is, and every piece of it that
// is still to write or to paint.
//
//	go run ./tools/motifreport
//
// **It draws no pictures and writes none.** The motif sheet is where a creature is looked at; this
// page is the worklist beside it — which briefs still read TBD or nothing, which creatures and
// rooms have no picture, and which fights would be drawn on the default backdrop. So it is cheap
// to regenerate, and a full `tools/sheets` run adds one small HTML file for it.
//
// Every count is taken off `data.LoadMotifs` and off the files actually under `assets/motifs/`, so
// the page cannot disagree with the game about what exists. **A picture is found by its stem**,
// exactly as `assets.embedTree` keys it, so where under the tree a file was filed does not matter.
//
// It also prints one line per motif to stdout, for a look without opening the page.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/tools/sheetfilter"
)

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "motifreport"), "directory to write index.html into")
	art := flag.String("assets", filepath.Join("assets", "motifs"), "the tree the motif pictures are filed under")
	flag.Parse()

	if err := run(*dir, *art); err != nil {
		log.Fatal(err)
	}
}

func run(dir, art string) error {
	pictures, err := stems(art)
	if err != nil {
		return err
	}

	motifs := data.LoadMotifs()
	var plates []plate
	for _, key := range data.MotifOrder(motifs) {
		plates = append(plates, report(motifs[key], pictures))
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

	p := page{Plates: plates, Filters: sheetfilter.Bar(facets(plates))}
	for _, pl := range plates {
		p.Total.add(pl.Score)
	}
	if err := tmpl.Execute(f, p); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}

	for _, pl := range plates {
		fmt.Printf("%-14s %3d%%  briefs %s  creature art %s  rooms %s  room art %s\n",
			pl.Motif, pl.Score.Percent(), pl.Score.Briefs, pl.Score.CreatureArt, pl.Score.Rooms, pl.Score.RoomArt)
	}
	fmt.Printf("wrote %s\n", out)
	return nil
}

// stems is every picture under the tree, by the key the game looks it up by.
func stems(root string) (map[string]bool, error) {
	out := map[string]bool{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := d.Name()
		out[strings.TrimSuffix(name, filepath.Ext(name))] = true
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", root, err)
	}
	return out, nil
}

// tally is x done of y.
type tally struct{ Done, Of int }

func (t tally) String() string { return fmt.Sprintf("%d/%d", t.Done, t.Of) }

func (t *tally) count(done bool) {
	t.Of++
	if done {
		t.Done++
	}
}

// Percent is the share done, rounded down so a motif one brief short never reads 100.
func (t tally) Percent() int {
	if t.Of == 0 {
		return 0
	}
	return t.Done * 100 / t.Of
}

// score is the four things a motif is filled with. **Rooms is counted by fight, not by record**:
// a floor holds fifteen fights across the three tiers and five elements, and what the owner is
// asking is how many of them are drawn in front of something other than the default.
type score struct {
	Briefs      tally // every Draw and ElementDraw a picture is briefed from, creatures and rooms
	CreatureArt tally // one per record per affinity
	Rooms       tally // one per (tier, element) fight: does any backdrop cover it
	RoomArt     tally // one per (tier, element) fight: is a covering backdrop painted
}

func (s *score) add(o score) {
	for _, pair := range [][2]*tally{{&s.Briefs, &o.Briefs}, {&s.CreatureArt, &o.CreatureArt}, {&s.Rooms, &o.Rooms}, {&s.RoomArt, &o.RoomArt}} {
		pair[0].Done += pair[1].Done
		pair[0].Of += pair[1].Of
	}
}

// Percent is every slot at once — the one figure "how full is this motif" is answered with.
func (s score) Percent() int {
	var all tally
	all.add(s.Briefs)
	all.add(s.CreatureArt)
	all.add(s.Rooms)
	all.add(s.RoomArt)
	return all.Percent()
}

func (t *tally) add(o tally) { t.Done += o.Done; t.Of += o.Of }

// written is the loader's own test of a brief, repeated because the data package keeps it
// unexported: neither empty nor TBD.
func written(s string) bool { return s != "" && s != data.DrawUnwritten }

// report is one motif's plate.
func report(m data.MotifData, pictures map[string]bool) plate {
	p := plate{Motif: m.Motif, Name: m.Name, Floors: floors(m)}

	// The motif's own two layers.
	p.Score.Briefs.count(written(m.Draw))
	if !written(m.Draw) {
		p.Todo = append(p.Todo, todo{"brief", "the motif's own Draw"})
	}
	for _, e := range data.AffinityElements {
		ok := written(m.ElementDraw[e])
		p.Score.Briefs.count(ok)
		if !ok {
			p.Todo = append(p.Todo, todo{"brief", "the motif's ElementDraw for " + e})
		}
	}

	// The portal's two lines, which are what a player reads rather than what an artist is given —
	// counted as briefs because an unwritten one is a gap on a screen, exactly as a missing Draw is
	// a gap on a card.
	p.Score.Briefs.count(written(m.Text))
	if !written(m.Text) {
		p.Todo = append(p.Todo, todo{"brief", "the motif's portal Text"})
	}
	for _, e := range data.AffinityElements {
		ok := written(m.ElementText[e])
		p.Score.Briefs.count(ok)
		if !ok {
			p.Todo = append(p.Todo, todo{"brief", "the motif's portal ElementText for " + e})
		}
	}

	// The creatures.
	cov := data.CoverageOf(m)
	for _, r := range m.Records {
		row := creature{Record: r.Record, Name: r.FullName(), Tier: r.Tier, Brief: written(r.Draw)}
		p.Score.Briefs.count(row.Brief)
		if !row.Brief {
			p.Todo = append(p.Todo, todo{"brief", r.Record + " Draw"})
		}
		for _, e := range r.Affinities {
			have := pictures[r.ArtKey(e)]
			p.Score.CreatureArt.count(have)
			row.Elements = append(row.Elements, cell{Element: e, Pictured: have, Specific: written(r.ElementDraw[e])})
			if !have {
				p.Todo = append(p.Todo, todo{"creature art", r.ArtKey(e) + ".png"})
			}
		}
		p.Creatures = append(p.Creatures, row)
	}
	p.CreatureGrid = grid(func(ti, ai int) (string, string) {
		n, want := cov.Counts[ti][ai], data.MinCoverageFor(data.TierOrder[ti])
		if n < want {
			return fmt.Sprintf("%d", n), "hole"
		}
		return fmt.Sprintf("%d", n), "ok"
	})

	// The rooms.
	for _, b := range m.Backdrops {
		row := room{Backdrop: b.Backdrop, Name: b.Name, Tier: b.Tier, Brief: written(b.Draw)}
		p.Score.Briefs.count(row.Brief)
		if !row.Brief {
			p.Todo = append(p.Todo, todo{"brief", b.Backdrop + " Draw"})
		}
		for _, e := range b.Affinities {
			have := pictures[b.ArtKey(e)]
			dir := written(b.ElementDraw[e])
			p.Score.Briefs.count(dir)
			if !dir {
				p.Todo = append(p.Todo, todo{"brief", b.Backdrop + " ElementDraw for " + e})
			}
			row.Elements = append(row.Elements, cell{Element: e, Pictured: have, Specific: dir})
			if !have {
				p.Todo = append(p.Todo, todo{"room art", b.ArtKey(e) + ".jpg"})
			}
		}
		p.Rooms = append(p.Rooms, row)
	}
	p.RoomGrid = grid(func(ti, ai int) (string, string) {
		tier, e := data.TierOrder[ti], data.AffinityElements[ai]
		var n, painted int
		for _, b := range m.Backdrops {
			if b.Tier == tier && b.HasAffinity(e) {
				n++
				if pictures[b.ArtKey(e)] {
					painted++
				}
			}
		}
		p.Score.Rooms.count(n > 0)
		p.Score.RoomArt.count(painted > 0)
		switch {
		case n == 0:
			p.Todo = append(p.Todo, todo{"room", fmt.Sprintf("no %s room in %s — the fight draws the default", tier, e)})
			return "—", "hole"
		case painted == 0:
			return fmt.Sprintf("0/%d", n), "brief"
		default:
			return fmt.Sprintf("%d/%d", painted, n), "ok"
		}
	})

	for _, t := range p.Todo {
		p.missing = append(p.missing, t.Kind)
	}
	return p
}

// grid lays a [tier][element] figure out as rows, outermost tier first.
func grid(at func(ti, ai int) (string, string)) []gridRow {
	var rows []gridRow
	for ti, tier := range data.TierOrder {
		row := gridRow{Tier: tierLabel[tier]}
		for ai := range data.AffinityElements {
			text, state := at(ti, ai)
			row.Cells = append(row.Cells, gridCell{Text: text, State: state})
		}
		rows = append(rows, row)
	}
	return rows
}

// tierLabel is what the page calls a tier: the room, which is what the owner calls it.
var tierLabel = map[string]string{
	data.TierOuter: "outer chamber",
	data.TierInner: "inner chamber",
	data.TierBoss:  "portal room",
}

func floors(m data.MotifData) string {
	if m.ValidFloors == [2]int{} {
		return "any floor"
	}
	if m.ValidFloors[0] == m.ValidFloors[1] {
		return fmt.Sprintf("floor %d", m.ValidFloors[0])
	}
	return fmt.Sprintf("floors %d–%d", m.ValidFloors[0], m.ValidFloors[1])
}

// facets is one chip row, "missing", whose values are the kinds of work a motif still has — so
// "show me every motif still short of a room" is one click. Counted off the plates, per
// sheetfilter's rule.
func facets(plates []plate) []sheetfilter.Facet {
	f := sheetfilter.Facet{Key: "missing", Label: "still missing"}
	for _, kind := range todoKinds {
		n := 0
		for _, p := range plates {
			if p.Missing(kind) {
				n++
			}
		}
		if n > 0 {
			f.Values = append(f.Values, sheetfilter.Value{Value: token(kind), Label: kind, Count: n})
		}
	}
	return []sheetfilter.Facet{f}
}

// todoKinds is the four kinds of work, in the order the page lists them.
var todoKinds = []string{"brief", "creature art", "room", "room art"}

func token(kind string) string { return strings.ReplaceAll(kind, " ", "-") }
