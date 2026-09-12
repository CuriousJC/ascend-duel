// Command scenariosheet lists every fixture in internal/scenario/scenarios.json and draws what
// each one plugs in, so the question "what can I boot into" is a page rather than a file read.
//
//	go run ./tools/scenariosheet
//
// It exists because the scenarios are the fastest way to look at anything in this game and were
// the least discoverable thing in the repo. A fixture that cuts a twenty-minute question down to a
// relaunch is worth nothing if nobody remembers it is there, and the only ways to find one were to
// read four hundred lines of JSON or to misspell a key and read the fatal line that lists them.
// Every entry here carries the command that launches it, ready to copy.
//
// # It reads the file rather than the package
//
// `internal/scenario` is behind a build tag, and the `//go:embed` deliberately lives in
// `scenario_on.go` so an untagged build carries neither the fixture nor the reader — see CLAUDE.md.
// Importing the package would mean building this tool, and `tools/sheets` with it, under
// `-tags scenario`, which would put a debug fixture into the one command that regenerates the
// committed sheets. So the tool parses the JSON itself, off disk, from the path below.
//
// **The cost is a second view of the record struct, and the tripwire is `DisallowUnknownFields`.**
// A field added to `internal/scenario` and not to this file fails the sheet loudly rather than
// being quietly left off the page — which is the failure that matters, since a page silently
// missing half a fixture is worse than no page.
//
// # What to look at
//
// **The Note against what the fixture actually plugs in.** A scenario's Note is a promise about the
// cards, the relics and the opponent, and nothing checks it — this is the only place the sentence
// and the contents are visible together, exactly as the relic sheet does for a relic's authored line.
//
// **Whether any two fixtures are the same fixture.** They accumulate, and two entries plugging in
// the same deck to look at different things are one entry with two notes.
//
// **The cards are drawn plain, riders and all.** A rider is printed as the word the file writes
// rather than painted onto the face: the wash an upgrade puts on a card is `tools/upgradesheet`'s
// subject, and reproducing it here would mean a third copy of `screens.upgradeForRider` — see that
// tool, which keeps the second one and says so.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/scenariosheet/` and **committed**, on
// the same terms as every other sheet. A clone opens `docs/sheets/index.html`.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// source is the fixture file, relative to the repo root the tool is run from.
const source = "internal/scenario/scenarios.json"

// ground is the one the cards are actually drawn on — `screens.screenGround`. Judging a fixture's
// hand against a white browser page is the same failure as previewing art at the wrong scale.
const ground = "#a8bcd4"

// placeholderArt is the face this sheet gives anything it is only naming — a parasite, a stone, a
// key the catalogues do not answer to.
//
// **It is the worm catalogue's own default rather than a fourth picture**, and it is spelled by
// reading data.DefaultWormArt rather than by writing the string again: a page that showed a
// picture no catalogue uses would be inventing art in a review tool. This sheet is about what a
// fixture *plugs in*, so a face here is a label, not a drawing to judge.
const placeholderArt = data.DefaultWormArt

// stripGap is the space between two cards in a row, and stripSplit the wider one between a row's
// sections. Same two figures tools/roster uses, for the same reason: a strip has to say where one
// group of cards stops without a caption inside the picture.
const (
	stripGap   = 10
	stripSplit = 34
)

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "scenariosheet"),
		"directory to write the PNGs and index.html into")
	src := flag.String("src", filepath.FromSlash(source), "the scenario file to read")
	flag.Parse()

	if err := run(*dir, *src); err != nil {
		log.Fatal(err)
	}
}

func run(dir, src string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("making %s: %w", dir, err)
	}

	list, err := load(src)
	if err != nil {
		return err
	}

	faces, err := cards.NewFaces(assets.LoadFontData()["kubasta"])
	if err != nil {
		return err
	}

	pg := page{Ground: ground, Source: filepath.ToSlash(src), Count: len(list)}
	written, strips := 0, 0

	for _, r := range list {
		p := plate{
			Record:  r.ScenarioRecord,
			Note:    r.Note,
			Facts:   factsOf(r),
			Relics:  relicLines(r.Relics),
			Held:    heldLines(r),
			Hand:    handLines(r),
			Deck:    deckLines(r.Deck),
			DeckSum: deckTotal(r.Deck),
		}

		for _, s := range sectionsOf(r) {
			strip, err := compose(faces, s.specs, s.style, s.splits)
			if err != nil {
				return fmt.Errorf("%s: %s: %w", r.ScenarioRecord, s.name, err)
			}
			name := r.ScenarioRecord + "-" + s.name + ".png"
			n, err := save(dir, name, strip)
			if err != nil {
				return err
			}
			written, strips = written+n, strips+1
			b := strip.Bounds()
			p.Strips = append(p.Strips, cell{
				File: name, Label: s.label, Width: b.Dx(), Height: b.Dy(),
			})
		}
		pg.Plates = append(pg.Plates, p)
	}

	out := filepath.Join(dir, "index.html")
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, pg); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}

	fmt.Printf("wrote %s and %d strips for %d scenarios, %.1f MB of PNG\n",
		out, strips, pg.Count, float64(written)/(1<<20))
	for _, p := range pg.Plates {
		fmt.Printf("  %-24s %s\n", p.Record, strings.Join(p.Facts, " · "))
	}
	return nil
}

// load reads and validates the fixture file.
//
// **Unknown fields are an error**, which is the whole guard against this tool's duplicate view of
// the record drifting from `internal/scenario`'s. See the package comment.
func load(src string) ([]record, error) {
	raw, err := os.ReadFile(src)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", src, err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var list []record
	if err := dec.Decode(&list); err != nil {
		return nil, fmt.Errorf("%s: %w -- if internal/scenario grew a field, add it to "+
			"tools/scenariosheet's record too", src, err)
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("%s holds no scenarios", src)
	}
	return list, nil
}

// record is this tool's view of one fixture. **A duplicate of internal/scenario's, knowingly**,
// held honest by DisallowUnknownFields above.
type record struct {
	ScenarioRecord string     `json:"ScenarioRecord"`
	Note           string     `json:"Note"`
	Relics         []string   `json:"Relics"`
	Hand           []handCard `json:"Hand"`
	Parasites      []string   `json:"Parasites"`
	Stones         []string   `json:"Stones"`
	Enemy          string     `json:"Enemy"`
	Screen         string     `json:"Screen"`
	Fight          int        `json:"Fight"`
	Vitae          int        `json:"Vitae"`
	Life           int        `json:"Life"`
	Deck           []deckLine `json:"Deck"`
	Seed           string     `json:"Seed"`
	Dummy          bool       `json:"Dummy"`
	Actions        int        `json:"Actions"`
	RoundLimit     int        `json:"RoundLimit"`
	RelicSlots     int        `json:"RelicSlots"`
	Teach          bool       `json:"Teach"`
}

type handCard struct {
	Card    string   `json:"Card"`
	Element string   `json:"Element"`
	Riders  []string `json:"Riders"`
}

type deckLine struct {
	Card    string   `json:"Card"`
	Element string   `json:"Element"`
	Copies  int      `json:"Copies"`
	Riders  []string `json:"Riders"`
}

// copies is how many cards one deck line actually is. Absent or zero is one, exactly as
// internal/scenario reads it.
func (d deckLine) copies() int {
	if d.Copies == 0 {
		return 1
	}
	return d.Copies
}

// section is one row of cards on a fixture's plate.
type section struct {
	name   string
	label  string
	style  cards.Style
	specs  []cards.Spec
	splits []int // indices a wider gap goes before
}

// sectionsOf is the rows a fixture has something to show in. **An empty row is omitted rather than
// drawn blank**, so the page's shape says at a glance which fixtures are about a hand and which
// are about what the run is carrying.
func sectionsOf(r record) []section {
	var out []section

	if worn := wornSpecs(r); len(worn.specs) > 0 {
		out = append(out, worn)
	}
	if len(r.Hand) > 0 {
		specs := make([]cards.Spec, 0, len(r.Hand))
		for _, c := range r.Hand {
			specs = append(specs, cardSpec(c.Card, c.Element))
		}
		out = append(out, section{name: "hand", label: "the opening hand, dealt over the shuffle",
			style: cards.Mini, specs: specs})
	}
	if len(r.Deck) > 0 {
		specs := make([]cards.Spec, 0, len(r.Deck))
		for _, l := range r.Deck {
			specs = append(specs, cardSpec(l.Card, l.Element))
		}
		out = append(out, section{name: "deck",
			label: "the replacement deck -- one card per line, the count is in the table",
			style: cards.Mini, specs: specs})
	}
	return out
}

// wornSpecs is the relics, the bucket and the pouch on one row, split apart by the wider gap.
//
// **One row rather than three**, because all three are things the run is *carrying* and a fixture
// rarely has more than a couple of each — three near-empty rows would say less than one.
func wornSpecs(r record) section {
	s := section{name: "worn", label: "worn, in the bucket, and in the pouch", style: cards.RelicStyle}

	relics := data.LoadRelics()
	for _, key := range r.Relics {
		rec, ok := relics[key]
		if !ok {
			s.specs = append(s.specs, missingSpec(key))
			continue
		}
		s.specs = append(s.specs, cards.Spec{
			Name: rec.Name, Element: cards.Relic, Art: artwork(rec.ArtKey()), Enabled: true,
		})
	}

	// **The parasites and the stones are drawn in the relic style too**, because a strip is one
	// style wide: WormStyle and RelicStyle are the same size, and mixing them in a row would be the
	// only place in the project two card formats stand side by side pretending to be a set.
	if len(r.Parasites) > 0 {
		s.splits = append(s.splits, len(s.specs))
		for _, key := range r.Parasites {
			p, ok := session.ParasiteByKey(key)
			if !ok {
				s.specs = append(s.specs, missingSpec(key))
				continue
			}
			s.specs = append(s.specs, goodSpec(p.Name))
		}
	}
	if len(r.Stones) > 0 {
		s.splits = append(s.splits, len(s.specs))
		stones := data.LoadStones()
		for _, key := range r.Stones {
			st, ok := stones[key]
			if !ok {
				s.specs = append(s.specs, missingSpec(key))
				continue
			}
			s.specs = append(s.specs, goodSpec(st.Name))
		}
	}
	return s
}

// goodSpec is a parasite or a stone as a face. **The name and nothing else** — what either one
// does is `tools/parasitesheet` and `tools/stonesheet`'s subject, and repeating their text here
// would be a third place the same sentence can go stale.
func goodSpec(name string) cards.Spec {
	return cards.Spec{Name: name, Element: cards.Relic, Art: artwork(placeholderArt), Enabled: true}
}

// missingSpec is a key nothing in the catalogues answers to. **Drawn rather than fatal**, unlike
// the game, which refuses the launch: the sheet's job is to show what the file says, and a page
// that would not render because one fixture names a deleted relic would hide the other twenty.
func missingSpec(key string) cards.Spec {
	return cards.Spec{Name: "?" + key, Element: cards.Relic, Art: artwork(placeholderArt), Enabled: false}
}

// cardSpec is one player card as the hand draws it. A key the registry does not hold is drawn
// disabled and named, for missingSpec's reason.
func cardSpec(label, element string) cards.Spec {
	id, ok := combat.ConceptByKey(label)
	if !ok {
		return cards.Spec{Name: "?" + label, Element: cards.Basic, Enabled: false}
	}
	e, _ := combat.ParseElement(element)
	c := combat.Of(id, e)
	return cards.Spec{
		Name:    c.Label(),
		Form:    formOf(c.Form()),
		Cost:    c.Cost(),
		Element: elementOf(c.Element),
		Enabled: true,
	}
}

// compose lays a row of rendered cards onto one picture, one strip per section.
//
// **A strip rather than a file per card**, for tools/roster's reason: a file per card is hundreds
// of near-identical binaries rewritten on every run for the same pixels, and these sheets are
// committed.
func compose(f *cards.Faces, specs []cards.Spec, style cards.Style, splits []int) (*image.RGBA, error) {
	imgs := make([]*image.RGBA, 0, len(specs))
	for _, s := range specs {
		img, err := cards.Render(s, style, f)
		if err != nil {
			return nil, fmt.Errorf("rendering %s: %w", s.Name, err)
		}
		imgs = append(imgs, img)
	}

	wide := map[int]bool{}
	for _, i := range splits {
		wide[i] = true
	}

	w, h := 0, 0
	for i, img := range imgs {
		if i > 0 {
			if wide[i] {
				w += stripSplit
			} else {
				w += stripGap
			}
		}
		w += img.Bounds().Dx()
		if d := img.Bounds().Dy(); d > h {
			h = d
		}
	}

	strip := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(strip, strip.Bounds(), &image.Uniform{C: groundRGBA}, image.Point{}, draw.Src)

	x := 0
	for i, img := range imgs {
		if i > 0 {
			if wide[i] {
				x += stripSplit
			} else {
				x += stripGap
			}
		}
		r := image.Rect(x, 0, x+img.Bounds().Dx(), img.Bounds().Dy())
		draw.Draw(strip, r, img, img.Bounds().Min, draw.Over)
		x = r.Max.X
	}
	return strip, nil
}

func save(dir, name string, img *image.RGBA) (int, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return 0, fmt.Errorf("encoding %s: %w", name, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), buf.Bytes(), 0o644); err != nil {
		return 0, fmt.Errorf("writing %s: %w", name, err)
	}
	return buf.Len(), nil
}

// factsOf is the run state a fixture arrives in, as short phrases. **Only what the fixture
// actually says** — a default is silence, so a plate with three facts on it is a plate that
// changed three things.
func factsOf(r record) []string {
	screen := r.Screen
	if screen == "" {
		screen = "combat"
	}
	out := []string{"opens on " + screen, "room " + strconv.Itoa(r.Fight)}
	if r.Seed != "" {
		out = append(out, "seed "+r.Seed)
	}
	if r.Enemy != "" {
		out = append(out, "against "+r.Enemy)
	}
	if r.Vitae > 0 {
		out = append(out, strconv.Itoa(r.Vitae)+" vitae")
	}
	if r.Life > 0 {
		out = append(out, strconv.Itoa(r.Life)+" life")
	}
	if r.Dummy {
		out = append(out, "DUMMY -- nothing dies")
	}
	if r.Actions > 0 {
		out = append(out, strconv.Itoa(r.Actions)+" AP a turn")
	}
	if r.RoundLimit > 0 {
		out = append(out, strconv.Itoa(r.RoundLimit)+"-round clock")
	}
	if r.RelicSlots > 0 {
		out = append(out, strconv.Itoa(r.RelicSlots)+" relic slots")
	}
	if r.Teach {
		out = append(out, "runs the tutorial")
	}
	return out
}

// relicLines names each worn relic, with the sentence it prints. The order is the fixture's, because
// worn order is a rule: relics fire left to right and two multiplicative ones do not commute.
func relicLines(keys []string) []named {
	relics := data.LoadRelics()
	out := make([]named, 0, len(keys))
	for _, key := range keys {
		rec, ok := relics[key]
		if !ok {
			out = append(out, named{Key: key, Name: "-- no such relic --"})
			continue
		}
		out = append(out, named{Key: key, Name: rec.Name, Text: oneLine(rec.Text)})
	}
	return out
}

// heldLines is the bucket and the pouch, in that order, each row saying which it is.
func heldLines(r record) []named {
	var out []named
	for _, key := range r.Parasites {
		p, ok := session.ParasiteByKey(key)
		if !ok {
			out = append(out, named{Key: key, Name: "-- no such parasite --", Kind: "bucket"})
			continue
		}
		out = append(out, named{Key: key, Name: p.Name, Text: oneLine(p.Text), Kind: "bucket"})
	}
	stones := data.LoadStones()
	for _, key := range r.Stones {
		st, ok := stones[key]
		if !ok {
			out = append(out, named{Key: key, Name: "-- no such stone --", Kind: "pouch"})
			continue
		}
		out = append(out, named{Key: key, Name: st.Name, Text: oneLine(st.Text), Kind: "pouch"})
	}
	return out
}

func handLines(r record) []line {
	out := make([]line, 0, len(r.Hand))
	for _, c := range r.Hand {
		out = append(out, line{
			Card: c.Card, Element: c.Element, Copies: 1,
			Riders: strings.Join(c.Riders, ", "), Cost: costOf(c.Card),
		})
	}
	return out
}

func deckLines(in []deckLine) []line {
	out := make([]line, 0, len(in))
	for _, l := range in {
		out = append(out, line{
			Card: l.Card, Element: l.Element, Copies: l.copies(),
			Riders: strings.Join(l.Riders, ", "), Cost: costOf(l.Card),
		})
	}
	return out
}

func deckTotal(in []deckLine) int {
	n := 0
	for _, l := range in {
		n += l.copies()
	}
	return n
}

func costOf(label string) int {
	id, ok := combat.ConceptByKey(label)
	if !ok {
		return 0
	}
	return combat.Of(id, combat.Basic).Cost()
}

// oneLine folds an authored line break, because `\n` in a card's Text is a layout instruction for
// a 128-pixel column and means nothing in a browser paragraph. See cards.WrapText.
func oneLine(s string) string { return strings.ReplaceAll(s, "\n", " ") }

// artwork decodes an embedded picture, returning nil rather than failing: a card with no face
// still shows its name, and a sheet that refused to render over one missing key would hide
// everything else. **The opposite call from the other sheets**, because this one is a picture of
// fixtures rather than of a catalogue — a fixture naming something that no longer exists is the
// finding, not a reason to have no page.
func artwork(key string) image.Image {
	raw := assets.LoadImageData()[key]
	if len(raw) == 0 {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	return img
}

func elementOf(e combat.Element) cards.Element {
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

func formOf(f combat.Form) cards.Form {
	switch f {
	case combat.FormStab:
		return cards.FormStab
	case combat.FormSlash:
		return cards.FormSlash
	case combat.FormCrush:
		return cards.FormCrush
	case combat.FormDefend:
		return cards.FormDefend
	default:
		return cards.FormNone
	}
}
