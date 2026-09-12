// Command wormsheet renders every worm in data/worms.json to a PNG and writes an HTML page
// that shows each one beside what it targets and what it says.
//
//	go run ./tools/wormsheet
//
// It exists for the reason tools/relicsheet does. A worm is offered two at a time, once per won
// fight, from a catalogue of ten — so seeing all of them in a launched game means winning five
// fights and being lucky about the shuffle. This draws all of them at once.
//
// # It is a report, not a drawing-board
//
// Same split as relicsheet against cardsheet: this reads the real file, because the question it
// answers is "what does the catalogue actually hold". It goes through internal/session, which
// means the catalogue is *validated* before anything is drawn — an unknown target, a value on a
// target that takes none, a missing one on a target that needs it, all panic at init exactly as
// they would in the game. A worm this page refuses to draw is a worm the game refuses to start
// with.
//
// # What to look at
//
// **The authored line against the rule beside it.** worms.json carries a Text field that the
// card prints verbatim and nothing checks against the rule that fires — the same hazard a relic's
// sentence carries, and this is the only place the two are visible together.
//
// **The border colours.** A worm's border carries the element it grants; the ones that grant no
// element are basic grey. How many of each is a fact about the offer, not a detail.
//
// **How much art there is not.** A worm with no `Art` of its own draws `default-worm.png` and the
// page marks it, exactly as the relic sheet marks an undrawn relic — so a column of identical
// faces reads as a backlog rather than as a bug. Its `Draw` is the subject paragraph an art
// generator would be given, and a record with neither is the backlog twice over.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/wormsheet/` and **committed**
// *(owner's call, 2026-08-23)*: the sheets are how the catalogues get reviewed, and requiring a
// Go toolchain to see one meant only whoever just changed something ever looked. A clone opens
// `docs/sheets/index.html`.
//
// The price is a directory of near-identical binaries rewritten on every run, so **regenerate
// deliberately** — `go run ./tools/sheets` does all of them and rewrites the index.
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
	"strconv"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// ground is screens.screenGround, the light slate blue a worm is actually offered on. Judging a card
// against a white browser page would be the same failure as previewing art at a scale the game
// does not use.
const ground = "#a8bcd4"

// offered is how many worms a won fight puts up. It is dealWorms' cut, written down here because
// that function lives in internal/screens and links Ebitengine. It is used for one derived
// number — the share of the offer a single worm can take — and is printed, so a stale copy is
// visible rather than silent.
const offered = 2

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "wormsheet"),
		"directory to write the PNGs and index.html into")
	flag.Parse()

	if err := run(*dir); err != nil {
		log.Fatal(err)
	}
}

func run(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("making %s: %w", dir, err)
	}

	faces, err := cards.NewFaces(assets.LoadFontData()["kubasta"])
	if err != nil {
		return err
	}

	// **The page walks the file's own order and the offer walks the sorted one.**
	// data.WormFileOrder is the motif order the catalogue is authored in — the five recolours
	// together, the two that resize a card beside each other — which is what makes the family
	// headings read as blocks. session.Worms stays the shuffle's order, and nothing on this page
	// decides an outcome, so the two never meet. Same split the relic sheet makes.
	order := data.WormFileOrder()

	page := page{
		Ground:  ground,
		Style:   styleFacts(cards.WormStyle),
		Count:   len(order),
		Offered: offered,
	}
	if page.Count > 0 {
		page.Share = fmt.Sprintf("%.1f", float64(offered)*100/float64(page.Count))
	}

	var plates []plate
	for _, key := range order {
		w, ok := session.WormByKey(key)
		if !ok {
			return fmt.Errorf("worms.json writes %q and internal/session resolved no such worm", key)
		}

		art, err := artwork(w.Art)
		if err != nil {
			return err
		}
		spec := specFor(w, art, true)
		cell, err := write(dir, faces, spec, cards.WormStyle, "worm-"+w.Record+".png", w.Name)
		if err != nil {
			return err
		}
		plates = append(plates, plate{
			Cell:    cell,
			Record:  w.Record,
			Name:    w.Name,
			Text:    w.Text,
			Target:  w.Target.String(),
			Value:   valueOf(w),
			Element: elementName(w),
			Rule:    ruleLine(w),
			Family:  w.Family,
			Draw:    w.Draw,
			Art:     w.Art,
			Default: w.Art == data.DefaultWormArt,
		})
		if w.Art == data.DefaultWormArt {
			page.Undrawn++
		}
		if w.Draw == "" {
			page.Unwritten++
		}
	}

	page.Targets = groupByTarget(plates)
	page.Families = groupByFamily(plates)

	// The two states a worm card is drawn in. **Not "chosen"** — the reward screen dims the one
	// that was not taken rather than lighting the one that was, so those are the two.
	if len(order) > 0 {
		first, _ := session.WormByKey(order[0])
		art, err := artwork(first.Art)
		if err != nil {
			return err
		}
		for _, st := range []struct {
			name    string
			label   string
			enabled bool
		}{
			{"rest", first.Name + " — on offer", true},
			{"disabled", first.Name + " — the offer not taken", false},
		} {
			cell, err := write(dir, faces, specFor(first, art, st.enabled),
				cards.WormStyle, "state-"+st.name+".png", st.label)
			if err != nil {
				return err
			}
			page.States = append(page.States, cell)
		}
	}

	out := filepath.Join(dir, "index.html")
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, page); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}

	fmt.Printf("wrote %s and %d PNGs — %d worms, %d with art of their own and %d with a subject; "+
		"%d offered a fight, %s%% of the catalogue a seat\n",
		out, len(plates)+len(page.States), page.Count,
		page.Count-page.Undrawn, page.Count-page.Unwritten, offered, page.Share)
	for _, f := range page.Families {
		fmt.Printf("  %-26s %2d %s\n", f.Name, f.Count, f.Noun)
	}
	return nil
}

// specFor is a worm as the card the reward screen draws, and it fills the same fields
// screens.wormSpec does: a name, the line, and the colour of whatever it grants. No form and no
// cost, which WormStyle draws as nothing.
func specFor(w session.Worm, art image.Image, enabled bool) cards.Spec {
	return cards.Spec{
		Name:       w.Name,
		Form:       cards.FormNone,
		Cost:       0,
		Element:    artFor(w.Element),
		Art:        art,
		Text:       w.Text,
		Highlights: cards.ElementHighlights(w.Text),
		Enabled:    enabled,
	}
}

// artFor maps the rules' element onto the drawing package's, exactly as internal/screens does.
// Two enums, because internal/cards knows how to draw a card and nothing about how a round
// resolves.
func artFor(e combat.Element) cards.Element {
	switch e {
	case combat.Fire:
		return cards.Fire
	case combat.Ice:
		return cards.Ice
	case combat.Lightning:
		return cards.Lightning
	case combat.Earth:
		return cards.Earth
	case combat.Arcane:
		return cards.Arcane
	default:
		return cards.Basic
	}
}

// ruleLine is what the worm does, in the file's own vocabulary.
//
// **Deliberately not prose**, for relicsheet's reason: the sentence a player reads is Text,
// printed beside this, and generating a second English sentence would give the page two
// descriptions and no way to tell which one the game agrees with.
func ruleLine(w session.Worm) string {
	switch w.Target {
	case session.TargetElement:
		return "element → " + w.Element.String()
	case session.TargetCost:
		return "cost " + withSign(w.Number) + " AP, floored at zero"
	case session.TargetAmount:
		return "amount × " + strconv.Itoa(w.Number) + "%"
	default:
		return w.Target.String()
	}
}

func withSign(n int) string {
	if n >= 0 {
		return "+" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// valueOf is the record's value as the page prints it, or empty for the targets that take none.
// Read off the resolved worm rather than the JSON, so it is the number the rules hold.
func valueOf(w session.Worm) string {
	switch w.Target {
	case session.TargetElement:
		return w.Element.String()
	case session.TargetCost, session.TargetAmount:
		return strconv.Itoa(w.Number)
	default:
		return ""
	}
}

// elementName is the colour the border carries, and says so in words for the worms that carry
// none — a grey border is a decision rather than an omission.
func elementName(w session.Worm) string {
	if w.Target != session.TargetElement {
		return "basic — grants no element"
	}
	return w.Element.String()
}

// groupByTarget splits the catalogue by what a worm changes, in session.WormTargets' order.
//
// **By target rather than alphabetically**, for the reason relicsheet groups by rarity: the target
// is the design axis, so what a review needs is every recolour side by side and then the count of
// everything else. An empty group still gets a heading, because a target nobody has authored into
// is a fact worth seeing rather than a section to omit.
func groupByTarget(plates []plate) []group {
	out := make([]group, 0, len(session.WormTargets()))
	for _, t := range session.WormTargets() {
		g := group{Target: t.String()}
		for _, p := range plates {
			if p.Target == t.String() {
				g.Worms = append(g.Worms, p)
			}
		}
		g.Count = len(g.Worms)
		out = append(out, g)
	}
	return out
}

// groupByFamily splits the catalogue into the motifs its records are authored in.
//
// **In first-appearance order, which is the file's order**, so the page reads as data/worms.json
// does and a worm lands where its siblings were written rather than where the alphabet puts it.
// It is the relic sheet's function over a different catalogue, and it earns its place here for the
// reason that one does: a family is a block to review at once.
//
// **The target grouping did not go — it moved to the header**, as a list of counts. Grouping by
// target was right while the target was the only axis the file had; now that a record says which
// motif it belongs to, the target is a fact about one worm and the family is a block of them. A
// target nobody has authored into is still visible, in that list.
//
// A record with no Family lands under "unfamilied" rather than being dropped — an ungrouped worm
// is a thing to see, not a thing to omit.
func groupByFamily(plates []plate) []family {
	order := make([]string, 0, 8)
	byName := map[string][]plate{}
	for _, p := range plates {
		name := p.Family
		if name == "" {
			name = "unfamilied"
		}
		if _, seen := byName[name]; !seen {
			order = append(order, name)
		}
		byName[name] = append(byName[name], p)
	}

	out := make([]family, 0, len(order))
	for _, name := range order {
		f := family{Name: name, Worms: byName[name], Count: len(byName[name])}
		f.Noun = "worms"
		if f.Count == 1 {
			f.Noun = "worm"
		}
		out = append(out, f)
	}
	return out
}

// write renders one card, saves it, and returns what the page needs to show it.
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
	return cell{File: name, Label: label, Width: st.Width, Height: st.Height}, nil
}

// artwork decodes one embedded picture. **A key that is in no embed is an error rather than a
// blank face**, unlike the game, which logs and draws the hole: a review tool that quietly drew
// nothing would be hiding exactly what it is for.
func artwork(key string) (image.Image, error) {
	raw := assets.LoadImageData()[key]
	if len(raw) == 0 {
		return nil, fmt.Errorf("no embedded image called %q", key)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", key, err)
	}
	return img, nil
}

// styleFacts is the numbers the page prints, read off the style rather than typed into the
// template, so the page cannot quote a card it is not showing.
func styleFacts(st cards.Style) map[string]int {
	return map[string]int{
		"width":        st.Width,
		"height":       st.Height,
		"cornerRadius": st.CornerRadius,
		"borderWidth":  st.BorderWidth,
		"artTop":       st.ArtTop,
		"artInset":     st.ArtInset,
		"artMaxH":      st.ArtMaxH,
		"textBandTop":  st.TextBandTop,
	}
}

type cell struct {
	File   string
	Label  string
	Width  int
	Height int
}

// plate is one worm: the card, and everything the file says about it.
type plate struct {
	Cell    cell
	Record  string
	Name    string
	Text    string
	Target  string
	Value   string
	Element string
	Rule    string

	// Family, Draw and Art are the three fields the engine ignores: the motif the record was
	// authored under, the subject paragraph an art generator is given, and the picture the card
	// actually draws. Default says that picture is the placeholder rather than one of its own.
	Family  string
	Draw    string
	Art     string
	Default bool
}

// group is one target's worth of the catalogue.
type group struct {
	Target string
	Count  int
	Worms  []plate
}

// family is one motif's worth of the catalogue: every worm authored in that block.
type family struct {
	Name  string
	Count int
	Noun  string
	Worms []plate
}

type page struct {
	Ground    string
	Style     map[string]int
	Count     int
	Offered   int
	Undrawn   int
	Unwritten int
	Share     string
	Targets   []group
	Families  []family
	States    []cell
}
