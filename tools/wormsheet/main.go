// Command wormsheet renders every worm in data/worms.json to a PNG and writes an HTML page
// that shows each one beside what it targets and what it says.
//
//	go run ./tools/wormsheet
//
// It exists for the reason tools/ringsheet does. A worm is offered two at a time, once per won
// fight, from a catalogue of ten — so seeing all of them in a launched game means winning five
// fights and being lucky about the shuffle. This draws all of them at once.
//
// # It is a report, not a drawing-board
//
// Same split as ringsheet against cardsheet: this reads the real file, because the question it
// answers is "what does the catalogue actually hold". It goes through internal/session, which
// means the catalogue is *validated* before anything is drawn — an unknown target, a value on a
// target that takes none, a missing one on a target that needs it, all panic at init exactly as
// they would in the game. A worm this page refuses to draw is a worm the game refuses to start
// with.
//
// # What to look at
//
// **The authored line against the rule beside it.** worms.json carries a Text field that the
// card prints verbatim and nothing checks against the rule that fires — the same hazard a ring's
// sentence carries, and this is the only place the two are visible together.
//
// **The border colours.** A worm's border carries the element it grants; the ones that grant no
// element are basic grey. How many of each is a fact about the offer, not a detail.
//
// **How much art there is not.** Every worm draws default-worm.png today, so the page is mostly
// a check that the seat is the right shape.
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
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// ground is screens.screenGround, the cream a worm is actually offered on. Judging a card
// against a white browser page would be the same failure as previewing art at a scale the game
// does not use.
const ground = "#e2d0b0"

// wormArtKey is the picture every worm draws, and it is the constant of the same name in
// internal/screens. **Keys are not file paths** — the file is assets/worm/default-worm.png and
// LoadImageData files it under this.
const wormArtKey = "defaultworm_png"

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

	art, err := artwork(wormArtKey)
	if err != nil {
		return err
	}

	// session.Worms is already the sorted order the offer shuffles, so two runs of the tool
	// produce the same page and a new worm lands in one predictable place.
	worms := session.Worms()

	page := page{
		Ground:  ground,
		Style:   styleFacts(cards.WormStyle),
		Count:   len(worms),
		Offered: offered,
	}
	if len(worms) > 0 {
		page.Share = fmt.Sprintf("%.1f", float64(offered)*100/float64(len(worms)))
	}

	var plates []plate
	for _, w := range worms {
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
		})
	}

	page.Groups = groupByTarget(plates)

	// The two states a worm card is drawn in. **Not "chosen"** — the reward screen dims the one
	// that was not taken rather than lighting the one that was, so those are the two.
	if len(worms) > 0 {
		for _, st := range []struct {
			name    string
			label   string
			enabled bool
		}{
			{"rest", worms[0].Name + " — on offer", true},
			{"disabled", worms[0].Name + " — the offer not taken", false},
		} {
			cell, err := write(dir, faces, specFor(worms[0], art, st.enabled),
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

	fmt.Printf("wrote %s and %d PNGs — %d worms, %d offered a fight, %s%% of the catalogue a seat\n",
		out, len(plates)+len(page.States), page.Count, offered, page.Share)
	for _, g := range page.Groups {
		fmt.Printf("  %-10s %2d worms\n", g.Target, g.Count)
	}
	return nil
}

// specFor is a worm as the card the reward screen draws, and it fills the same fields
// screens.wormSpec does: a name, the line, and the colour of whatever it grants. No form and no
// cost, which WormStyle draws as nothing.
func specFor(w session.Worm, art image.Image, enabled bool) cards.Spec {
	return cards.Spec{
		Name:    w.Name,
		Form:    cards.FormNone,
		Cost:    0,
		Element: artFor(w.Element),
		Art:     art,
		Text:    w.Text,
		Enabled: enabled,
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
	default:
		return cards.Basic
	}
}

// ruleLine is what the worm does, in the file's own vocabulary.
//
// **Deliberately not prose**, for ringsheet's reason: the sentence a player reads is Text,
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
// **By target rather than alphabetically**, for the reason ringsheet groups by rarity: the target
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
}

// group is one target's worth of the catalogue.
type group struct {
	Target string
	Count  int
	Worms  []plate
}

type page struct {
	Ground  string
	Style   map[string]int
	Count   int
	Offered int
	Share   string
	Groups  []group
	States  []cell
}
