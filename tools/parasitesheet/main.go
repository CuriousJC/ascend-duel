// Command parasitesheet renders every parasite in data/parasites.json to a PNG and writes an HTML
// page that shows each one beside the rule it actually fires.
//
//	go run ./tools/parasitesheet
//
// It exists for the reason tools/ringsheet does, and the catalogue being small today is not an
// argument against it. A parasite is the least readable record in `data/` — a `Target`, a `Rider`,
// a `Value` and a `Count`, where which of those the rules read depends entirely on the target, and
// three of the four are refused outright on the targets that do not read them. The sentence the
// card prints is authored separately and checked against none of it.
//
// **So the page's whole job is putting the authored line and the resolved rule side by side.**
// That is the ring sheet's job too, and it is the one review a growing catalogue needs from the
// first record rather than the fortieth.
//
// # It is a report, not a drawing-board
//
// This reads the real file, through internal/session, which means the catalogue is *validated*
// before anything is drawn: an unknown target, a rider named on a target that reads none, a count
// past `MaxParasiteTargets`, a swap naming a card this build has not registered — all panic at
// init exactly as they would in the game. A parasite this page refuses to draw is a parasite the
// game refuses to start with.
//
// # What to look at
//
// **The line against the rule.** `Text` is printed verbatim on the face and nothing checks it
// against the target beside it. "eat two cards" over a record carrying `Count: 1` is the failure
// this page exists to make visible.
//
// **How many cards each one asks for.** The board piece shows targets side by side and
// `MaxParasiteTargets` is two, so the counts here are the whole of what the picker ever has to
// lay out. A catalogue drifting towards two-target parasites is a layout decision being made by
// accident.
//
// **Which targets nobody has authored into.** The vocabulary is closed and every target gets a
// heading whether or not the file uses it, so an unused one is a mechanic built and never reached
// for — which is a design question rather than a bug.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/parasitesheet/` and **committed**
// *(owner's call, 2026-08-23)*, on the same terms as every other sheet. A clone opens
// `docs/sheets/index.html`.
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

// ground is screens.screenGround, the cream a parasite is actually offered on.
const ground = "#e2d0b0"

// wormArtKey is the picture every parasite draws, and it is the constant of the same name in
// internal/screens — the parasite borrows the worm's placeholder, because a card with no face at
// all would be worse than one wearing a placeholder its sibling already wears.
//
// **Keys are not file paths**: the file is assets/worm/default-worm.png and LoadImageData files it
// under this.
const wormArtKey = "defaultworm_png"

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "parasitesheet"),
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

	// session.Parasites is already the sorted order the bucket draws from, so two runs of the tool
	// produce the same page and a new parasite lands in one predictable place.
	parasites := session.Parasites()

	page := page{
		Ground:      ground,
		Style:       styleFacts(cards.WormStyle),
		Count:       len(parasites),
		BucketSize:  session.BucketSize(),
		BucketPrice: session.BucketPrice(),
		MaxTargets:  session.MaxParasiteTargets,
	}
	if page.Count > 0 {
		page.Share = fmt.Sprintf("%.1f", float64(page.BucketSize)*100/float64(page.Count))
	}

	var plates []plate
	for _, p := range parasites {
		cell, err := write(dir, faces, specFor(p, art, true, false),
			"parasite-"+p.Record+".png", p.Name)
		if err != nil {
			return err
		}
		plates = append(plates, plate{
			Cell:   cell,
			Record: p.Record,
			Name:   p.Name,
			Text:   p.Text,
			Target: p.Target.String(),
			Cards:  cardsWanted(p),
			Value:  valueOf(p),
			Rule:   ruleLine(p),
		})
	}

	page.Groups = groupByTarget(plates)

	// The three states a parasite card is drawn in, which is one more than a worm has. **Selected
	// is a state here and is not one there**: a parasite is armed first and aimed second, so the
	// board piece has to say which one is in hand while the player picks what it eats.
	if len(parasites) > 0 {
		first := parasites[0]
		for _, s := range []struct {
			name            string
			label           string
			enabled, chosen bool
		}{
			{"rest", first.Name + " — in the bucket", true, false},
			{"selected", first.Name + " — armed, picking its targets", true, true},
			{"disabled", first.Name + " — unusable this turn", false, false},
		} {
			cell, err := write(dir, faces, specFor(first, art, s.enabled, s.chosen),
				"state-"+s.name+".png", s.label)
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

	fmt.Printf("wrote %s and %d PNGs — %d parasites, %d drawn from a %d-vitae bucket, %s%% of the catalogue a seat\n",
		out, len(plates)+len(page.States), page.Count, page.BucketSize, page.BucketPrice, page.Share)
	for _, g := range page.Groups {
		fmt.Printf("  %-8s %2d parasites\n", g.Target, g.Count)
	}
	return nil
}

// specFor is a parasite as the card the bucket draws, and it fills the same fields
// screens.parasiteSpec does: a name, the line, no form and no cost. **Basic, not a colour** — a
// parasite grants no element, so its border is the mid grey `cards.BorderOf` gives `basic`.
func specFor(p session.Parasite, art image.Image, enabled, selected bool) cards.Spec {
	return cards.Spec{
		Name:     p.Name,
		Form:     cards.FormNone,
		Cost:     0,
		Element:  cards.Basic,
		Art:      art,
		Text:     p.Text,
		Enabled:  enabled,
		Selected: selected,
	}
}

// ruleLine is what the parasite does, in the file's own vocabulary.
//
// **Deliberately not prose**, for ringsheet's reason: the sentence a player reads is Text, printed
// beside this, and generating a second English sentence would give the page two descriptions and
// no way to tell which one the game agrees with.
func ruleLine(p session.Parasite) string {
	switch p.Target {
	case session.ParasiteRider:
		return fmt.Sprintf("rider %s, value %d", p.Rider, p.Number)
	case session.ParasiteRemove:
		return fmt.Sprintf("removes %d card(s) from the run", p.Count)
	case session.ParasiteSwap:
		return "becomes " + combat.Of(p.Concept, combat.Basic).Label()
	case session.ParasiteVitae:
		return fmt.Sprintf("+%d vitae, touching no card", p.Number)
	default:
		return p.Target.String()
	}
}

// valueOf is the record's value as the page prints it, or empty for the target that takes none.
// Read off the resolved parasite rather than the JSON, so it is what the rules hold.
func valueOf(p session.Parasite) string {
	switch p.Target {
	case session.ParasiteRider, session.ParasiteVitae:
		return strconv.Itoa(p.Number)
	case session.ParasiteSwap:
		return combat.Of(p.Concept, combat.Basic).Label()
	default:
		return ""
	}
}

// cardsWanted is how many cards the picker will ask for, said in words for the one that asks for
// none — a parasite that touches no card is a decision rather than an omission.
func cardsWanted(p session.Parasite) string {
	if p.Count == 0 {
		return "none — touches no card"
	}
	return strconv.Itoa(p.Count)
}

// groupByTarget splits the catalogue by what a parasite does, in session.ParasiteTargets' order.
//
// **An empty group still gets a heading**, exactly as the worm sheet's do: the vocabulary is
// closed, so a target nobody has authored into is a mechanic built and never reached for. That is
// worth seeing rather than a section to omit — and with four records against four targets it is
// most of what this page currently has to say.
func groupByTarget(plates []plate) []group {
	out := make([]group, 0, len(session.ParasiteTargets()))
	for _, t := range session.ParasiteTargets() {
		g := group{Target: t.String()}
		for _, p := range plates {
			if p.Target == t.String() {
				g.Parasites = append(g.Parasites, p)
			}
		}
		g.Count = len(g.Parasites)
		out = append(out, g)
	}
	return out
}

// write renders one card, saves it, and returns what the page needs to show it.
func write(dir string, f *cards.Faces, s cards.Spec, name, label string) (cell, error) {
	img, err := cards.Render(s, cards.WormStyle, f)
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
		Width: cards.WormStyle.Width, Height: cards.WormStyle.Height,
	}, nil
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

// plate is one parasite: the card, and everything the file says about it.
type plate struct {
	Cell   cell
	Record string
	Name   string
	Text   string
	Target string
	Cards  string
	Value  string
	Rule   string
}

// group is one target's worth of the catalogue.
type group struct {
	Target    string
	Count     int
	Parasites []plate
}

type page struct {
	Ground      string
	Style       map[string]int
	Count       int
	BucketSize  int
	BucketPrice int
	MaxTargets  int
	Share       string
	Groups      []group
	States      []cell
}
