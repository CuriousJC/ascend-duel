// Command parasitesheet renders every parasite in data/parasites.json to a PNG and writes an HTML
// page that shows each one beside the rule it actually fires.
//
//	go run ./tools/parasitesheet
//
// It exists for the reason tools/relicsheet does, and the catalogue being small today is not an
// argument against it. A parasite is the least readable record in `data/` — a `Target`, a `Rider`,
// a `Value` and a `Count`, where which of those the rules read depends entirely on the target, and
// three of the four are refused outright on the targets that do not read them. The sentence the
// card prints is authored separately and checked against none of it.
//
// **So the page's whole job is putting the authored line and the resolved rule side by side.**
// That is the relic sheet's job too, and it is the one review a growing catalogue needs from the
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
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// ground is screens.screenGround, the light slate blue a parasite is actually offered on.
const ground = "#a8bcd4"

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

	// **The page walks the file's own order and the bucket walks the sorted one.**
	// data.ParasiteFileOrder is the motif order the catalogue is authored in — the five bores
	// together, the four grubs, the metals beside each other — which is what makes the family
	// headings read as blocks. session.Parasites stays the bucket's order, and nothing on this page
	// decides an outcome, so the two never meet. Same split the relic sheet makes.
	order := data.ParasiteFileOrder()

	page := page{
		Ground:      ground,
		Style:       styleFacts(cards.WormStyle),
		Count:       len(order),
		BucketSize:  session.BucketSize(),
		BucketPrice: session.BucketPrice(),
		MaxTargets:  session.MaxParasiteTargets,
	}
	if page.Count > 0 {
		page.Share = fmt.Sprintf("%.1f", float64(page.BucketSize)*100/float64(page.Count))
	}

	var plates []plate
	for _, key := range order {
		p, ok := session.ParasiteByKey(key)
		if !ok {
			return fmt.Errorf("parasites.json writes %q and internal/session resolved no such parasite", key)
		}

		art, err := artwork(p.Art)
		if err != nil {
			return err
		}
		cell, err := write(dir, faces, specFor(p, art, true, false),
			"parasite-"+p.Record+".png", p.Name)
		if err != nil {
			return err
		}
		plates = append(plates, plate{
			Cell:    cell,
			Record:  p.Record,
			Name:    p.Name,
			Text:    p.Text,
			Target:  p.Target.String(),
			Cards:   cardsWanted(p),
			Value:   valueOf(p),
			Rule:    ruleLine(p),
			Family:  p.Family,
			Draw:    p.Draw,
			Art:     p.Art,
			Default: p.Art == data.DefaultParasiteArt,
		})
		if p.Art == data.DefaultParasiteArt {
			page.Undrawn++
		}
		if p.Draw == "" {
			page.Unwritten++
		}
	}

	page.Targets = groupByTarget(plates)
	page.Families = groupByFamily(plates)

	// The three states a parasite card is drawn in, which is one more than a worm has. **Selected
	// is a state here and is not one there**: a parasite is armed first and aimed second, so the
	// board piece has to say which one is in hand while the player picks what it eats.
	if len(order) > 0 {
		first, _ := session.ParasiteByKey(order[0])
		art, err := artwork(first.Art)
		if err != nil {
			return err
		}
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

	fmt.Printf("wrote %s and %d PNGs — %d parasites, %d with art of their own and %d with a subject; "+
		"%d drawn from a %d-vitae bucket, %s%% of the catalogue a seat\n",
		out, len(plates)+len(page.States), page.Count,
		page.Count-page.Undrawn, page.Count-page.Unwritten,
		page.BucketSize, page.BucketPrice, page.Share)
	for _, f := range page.Families {
		fmt.Printf("  %-24s %2d %s\n", f.Name, f.Count, f.Noun)
	}
	return nil
}

// specFor is a parasite as the card the bucket draws, and it fills the same fields
// screens.parasiteSpec does: a name, the line, no form and no cost. **Basic, not a colour** — a
// parasite grants no element, so its border is the mid grey `cards.BorderOf` gives `basic`.
func specFor(p session.Parasite, art image.Image, enabled, selected bool) cards.Spec {
	return cards.Spec{
		Name:       p.Name,
		Form:       cards.FormNone,
		Cost:       0,
		Element:    cards.Basic,
		Art:        art,
		Text:       p.Text,
		Highlights: cards.ElementHighlights(p.Text),
		Enabled:    enabled,
		Selected:   selected,
	}
}

// ruleLine is what the parasite does, in the file's own vocabulary.
//
// **Deliberately not prose**, for relicsheet's reason: the sentence a player reads is Text, printed
// beside this, and generating a second English sentence would give the page two descriptions and
// no way to tell which one the game agrees with.
func ruleLine(p session.Parasite) string {
	switch p.Target {
	case session.ParasiteRider:
		// **The two metals read their figure as a denominator rather than as a payout**, so a bare
		// "value 5" beside them would read as five of something. They are the only riders whose
		// number is odds, which is why this is a case here and not a widening of the line below.
		switch p.Rider {
		case combat.RiderGolden:
			return fmt.Sprintf("upgrade: 1 in %d on play: +%d DMG; 1 in %d: +%d max life; else nothing",
				p.Number, combat.LuckDMG, p.Number, combat.LuckLife)
		case combat.RiderSilver:
			return fmt.Sprintf("upgrade: 1 in %d on play: +%d vitae; else nothing",
				p.Number, combat.SilverVitae)
		}
		return fmt.Sprintf("upgrade: rider %s, value %d", p.Rider, p.Number)
	case session.ParasiteRemove:
		return fmt.Sprintf("removes %d card(s) from the run", p.Count)
	case session.ParasiteSwap:
		return "becomes " + combat.Of(p.Concept, combat.Basic).Label()
	case session.ParasiteVitae:
		return fmt.Sprintf("+%d vitae, touching no card", p.Number)
	case session.ParasiteChimera:
		// **The page cannot say what it copies**, because that is a fact about a run in progress
		// and this sheet is drawn against no run at all. Saying so is better than saying nothing.
		return "fires the run's last parasite again — count and effect are that one's"
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
	if p.Target == session.ParasiteChimera {
		// Its own record names none; what it asks for comes from whatever it is copying.
		return "as many as the parasite it copies"
	}
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

// groupByFamily splits the catalogue into the motifs its records are authored in.
//
// **In first-appearance order, which is the file's order**, so the page reads as
// data/parasites.json does and a parasite lands where its siblings were written rather than where
// the alphabet puts it. It is the relic sheet's function over a different catalogue.
//
// **The target grouping did not go — it moved to the header**, as a list of counts. That grouping
// was right while the target was the only axis the file had, and it is a poor block heading now
// that twelve of the records are one target: `swap` was a third of the page under one word, and
// the motif is what tells a jar of needles from a jar of razors.
//
// A record with no Family lands under "unfamilied" rather than being dropped.
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
		f := family{Name: name, Parasites: byName[name], Count: len(byName[name])}
		f.Noun = "parasites"
		if f.Count == 1 {
			f.Noun = "parasite"
		}
		out = append(out, f)
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
	Target    string
	Count     int
	Parasites []plate
}

// family is one motif's worth of the catalogue: every parasite authored in that block.
type family struct {
	Name      string
	Count     int
	Noun      string
	Parasites []plate
}

type page struct {
	Ground      string
	Style       map[string]int
	Count       int
	BucketSize  int
	BucketPrice int
	MaxTargets  int
	Undrawn     int
	Unwritten   int
	Share       string
	Targets     []group
	Families    []family
	States      []cell
}
