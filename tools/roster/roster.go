// Package roster renders one opponent-catalog review sheet: every record in a pool, drawn
// as the card the game draws, beside the deck it fights with and the stats it fights on.
//
// # Why this is a library and the other sheets are not
//
// `tools/sheets` says the other sheets share nothing but the words `png.Encode`, and that a
// library between them would exist to be a seam that command already is. That argument is still
// right, and this package is a `Pool` plus a page rather than a library between sheets:
// `tools/motifsheet` is a pool and a `main`. The shape is kept because the pool is the one thing
// a second roster view would vary, and a second view that had not learned about a new column
// would show the game as it was.
//
// # It is a report, like the relic and essence sheets
//
// It reads the motif files under `data/motifs` rather than writing its own contents out.
// It also imports `internal/decks`, which registers every creature concept at init — so a
// card naming a verb the rules do not have fails the sheet exactly as it fails the launch, and
// the deck sizes printed here are the real expanded piles rather than a `Copies` column added up
// in a template.
//
// # What to look at
//
// **The stat line against the floor.** The whole roster is hand-assigned, so the review question
// is whether floor 5's creatures are actually dearer than floor 4's — which an alphabetical list
// of ninety-six records cannot answer and a page grouped by floor can.
//
// **The deck beside the stats.** An enemy's personality is what it holds; `Copies` is the
// difficulty dial and it is sharper than it looks, since four copies of a 1 AP card in one turn
// is a Barrage at 5x. The strip shows one card per concept and the table under it says how many
// of each the pile holds.
//
// # Output
//
// One composite PNG per record — the opponent's card followed by its deck — plus an index.html,
// written under `docs/sheets/`. **One strip rather than one file per card** *(deliberate)*: a
// file per card would be about five hundred binaries rewritten on every run of `tools/sheets`,
// against the hundred and twenty-six a strip costs, for the same pixels. The sheets are committed
// and their weight is charged to git history, so the cheaper shape wins where the page reads the
// same either way.
package roster

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
	"github.com/curiousjc/ascend-duel/tools/sheetfilter"
)

// Ground is the one an opponent card is actually drawn on — `screens.screenGround`, the fill
// behind both corners of the combat screen. Judging a card against a white browser page would be
// the same failure as previewing art at a scale the game does not use.
const Ground = "#a8bcd4"

// groundRGBA is Ground as pixels, for the strip the cards are composited onto. Said twice in two
// forms rather than parsed, because a hex parser here would be four lines to avoid one constant.
var groundRGBA = color.RGBA{R: 0xe2, G: 0xd0, B: 0xb0, A: 255}

// The space between cards on a strip. The gap after the opponent's own card is wider: the card on
// the left is the creature and the rest are what it holds, and a single pitch would read as one
// row of six equals.
const (
	stripGap   = 14
	stripSplit = 34
)

// Pool is one catalog: what it is called, and how to read it.
//
// **The pool is the only thing the two commands differ by.** Everything a page shows is derived
// from the entries it hands back.
type Pool struct {
	// Name is the sheet's own noun, used in the run's report — "enemy", "boss".
	Name string

	// Title is the page's heading.
	Title string

	// Blurb is the paragraph under the heading, saying what this page is for.
	Blurb string

	// GroupLabel is what a section of the page is a group of — "floor band", "floor".
	GroupLabel string

	// Entries reads the catalog, in the order the page should show it.
	Entries func() []Entry
}

// Entry is one opponent, whole: what the file says about it and what its deck expands to.
type Entry struct {
	Record string

	// Name is what goes on the card, which for a boss is the bare first name — see BossData.Name
	// for why the title is not on it.
	Name string

	// Title is the rest of what an opponent is called, drawn under the name on the page and never
	// on the card. Empty for every creature in the roster: a creature has no title.
	Title string

	// Family is the tier this record stands in — outer, inner or the stairway's boss — and Draw is
	// the subject paragraph an art generator would be given. **Draw is authored and ignored by
	// everything that plays the game**, exactly as a relic's is.
	Family string
	Draw   string

	// Portrait is the art key drawn on the card, which is the record's own picture in the element
	// this row is showing it at.
	Portrait string

	// Element is the colour this row draws the record at, and Elements is every colour it could be
	// dealt as. The strip shows one because a record has one card face per colour and nine of them
	// in a row would be nine copies of the same reading; the list beside it is what says the
	// record is not stuck at the one drawn.
	Element  string
	Elements []string

	DMG     int
	Actions int
	HP      int

	// Group is the sort key the page's sections are cut on, and Floors is how that section is
	// written. A section is one motif, ordered by the floor its band opens on.
	Group  int
	Floors string

	// Band is the same band as a pair of floors, which is what the floor chips are matched
	// against. Written out rather than parsed back off Floors, which is prose.
	Band [2]int

	// Motif is which file this record came from, and Tier is which of a floor's three rooms it can
	// stand in.
	Motif string
	Tier  string

	Cards []data.CardData
}

// MotifPool is every record in every motif file, motif by motif, shallowest band first.
//
// **Cut by motif rather than by floor**, because a motif is the unit a floor is built from: a
// floor takes one whole motif and one element, so the question this page answers is whether one
// motif's three rooms read as a climb and whether its creatures look like each other.
var MotifPool = Pool{
	Name:       "motif",
	Title:      "Motif sheet",
	GroupLabel: "motif",
	Blurb: "The whole roster, motif by motif: every creature's card as the game draws it, its " +
		"stat line, the colours it can be dealt as, and the deck it fights with. A floor takes " +
		"one motif and one element and holds three fights, so the questions this page answers " +
		"are whether a motif's outer, inner and stairway records read as a climb, and whether " +
		"every one of its fights can be dealt at least two ways.",
	Entries: func() []Entry {
		motifs := data.LoadMotifs()
		var out []Entry
		for _, key := range data.MotifOrder(motifs) {
			m := motifs[key]
			for _, r := range m.Records {
				element := ""
				if len(r.Affinities) > 0 {
					element = r.Affinities[0]
				}
				out = append(out, Entry{
					Record: r.Record, Name: r.Name, Title: r.Title,
					Family: r.Tier, Draw: r.Draw,
					Portrait: r.ArtKey(element),
					Element:  element, Elements: r.Affinities,
					DMG: r.DMG, Actions: r.Actions, HP: r.HP,
					Group: m.ValidFloors[0], Floors: m.Name + " — " + floorBand(m.ValidFloors),
					Band:  m.ValidFloors,
					Motif: m.Motif, Tier: r.Tier,
					Cards: r.Cards,
				})
			}
		}
		return out
	},
}

// floorBand writes a [2]int range the way the page reads it. A zero range means every floor —
// see EnemyData.AllowsFloor, where a record written without the field is fightable rather than
// unreachable — and the page has to say so rather than printing "Floors 0-0".
func floorBand(f [2]int) string {
	switch {
	case f == [2]int{}:
		return "Any floor"
	case f[0] == f[1]:
		return "Floor " + strconv.Itoa(f[0])
	default:
		return "Floors " + strconv.Itoa(f[0]) + "–" + strconv.Itoa(f[1])
	}
}

// Run writes the sheet.
func Run(p Pool, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("making %s: %w", dir, err)
	}

	faces, err := cards.NewFaces(assets.LoadFontData()["kubasta"])
	if err != nil {
		return err
	}

	entries := p.Entries()
	lo, hi := span(entries)
	pg := page{
		SpanLo:     lo,
		SpanHi:     hi,
		Ground:     Ground,
		Title:      p.Title,
		Blurb:      p.Blurb,
		GroupLabel: p.GroupLabel,
		Count:      len(entries),
		Style:      styleFacts(cards.EnemyStyle),
		Filters:    sheetfilter.Bar(facetsFor(entries)),
	}

	var written int64
	for _, e := range entries {
		strip, err := stripFor(faces, e)
		if err != nil {
			return err
		}
		name := "strip-" + strings.ToLower(e.Record) + ".png"
		n, err := writePNG(filepath.Join(dir, name), strip)
		if err != nil {
			return err
		}
		written += n

		pg.add(plate{
			Entry:   e,
			Cell:    cell{File: name, Width: strip.Bounds().Dx(), Height: strip.Bounds().Dy()},
			Affixes: strings.Join(e.Elements, ", "),
			Elems:   strings.Join(e.Elements, " "),
			Deck:    len(decks.EnemyCards(e.Record, e.Element)),
			Rows:    deckRows(e.Cards),
		})
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

	fmt.Printf("wrote %s and %d strips — %d %s records, %.1f MB of PNG\n",
		out, len(entries), pg.Count, p.Name, float64(written)/(1<<20))
	for _, g := range pg.Groups {
		plural := "s"
		if len(g.Plates) == 1 {
			plural = " "
		}
		fmt.Printf("  %-14s %2d record%s — HP %d-%d, DMG %d-%d, AP %d-%d — %s\n",
			g.Label, len(g.Plates), plural, g.MinHP, g.MaxHP, g.MinDMG, g.MaxDMG,
			g.MinAP, g.MaxAP, g.Mix)
	}
	return nil
}

// stripFor composites one opponent's card and its deck onto a single picture.
//
// **One card per concept, not per copy.** A pile of six Spores is six of the same picture, and
// the count is a number the table beside the strip prints — showing it as six cards would make a
// swarm's row six times as wide as a brute's while saying nothing a figure does not.
func stripFor(f *cards.Faces, e Entry) (*image.RGBA, error) {
	art, err := artwork(e.Portrait)
	if err != nil {
		return nil, err
	}

	face, err := cards.Render(opponentSpec(e, art), cards.EnemyStyle, f)
	if err != nil {
		return nil, fmt.Errorf("rendering %s: %w", e.Record, err)
	}

	deck := make([]*image.RGBA, 0, len(e.Cards))
	for _, c := range e.Cards {
		img, err := cards.Render(cardSpec(c, e.Element), cards.Hand, f)
		if err != nil {
			return nil, fmt.Errorf("rendering %s.%s: %w", e.Record, c.Label, err)
		}
		deck = append(deck, img)
	}

	w, h := face.Bounds().Dx(), face.Bounds().Dy()
	for i, img := range deck {
		w += img.Bounds().Dx()
		if i == 0 {
			w += stripSplit
		} else {
			w += stripGap
		}
		if d := img.Bounds().Dy(); d > h {
			h = d
		}
	}

	strip := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(strip, strip.Bounds(), &image.Uniform{C: groundRGBA}, image.Point{}, draw.Src)

	x := 0
	place := func(img *image.RGBA) {
		r := image.Rect(x, 0, x+img.Bounds().Dx(), img.Bounds().Dy())
		draw.Draw(strip, r, img, img.Bounds().Min, draw.Over)
		x = r.Max.X
	}
	place(face)
	for i, img := range deck {
		if i == 0 {
			x += stripSplit
		} else {
			x += stripGap
		}
		place(img)
	}
	return strip, nil
}

// opponentSpec is the opponent as the game's own card: name, portrait, and a full health bar.
//
// **Full life rather than a sample wound**, because the figure a reviewer is reading is the
// record's HP and a bar drawn at some fraction of it would be inviting the question of which
// fraction. It carries no status badges for the same reason — a status is something a fight puts
// on a combatant, and nothing in a catalog has been in one.
func opponentSpec(e Entry, art image.Image) cards.Spec {
	return cards.Spec{
		Name:    e.Name,
		Element: cards.Basic,
		Art:     art,
		Life:    e.HP,
		MaxLife: e.HP,
		Enabled: true,

		// **The figure under the portrait, as the game writes it.** A card here missing a row the
		// combat screen draws is a page about a card that does not exist.
		PortraitStat: cards.StatLine{Label: "DMG", Value: strconv.Itoa(e.DMG)},
	}
}

// cardSpec is one of an opponent's cards, drawn in the hand style the player's cards use.
//
// **The hand style on purpose**, even though nobody ever holds one of these: it is the only style
// that shows a cost column and an effect band, which is the whole of what there is to review about
// an enemy card. Drawing it in the style the player's deck is drawn in is also what lets the two
// be compared, which is the actual balance question.
func cardSpec(c data.CardData, el string) cards.Spec {
	return cards.Spec{
		Name:       c.Label,
		Form:       form(c.Form),
		Cost:       c.Cost,
		Element:    element([]string{el}),
		Text:       effectText(c),
		Highlights: cards.ElementHighlights(effectText(c)),
		Enabled:    true,
	}
}

// effectText is what the card face says it does.
//
// **A copy of `screens.cardEffect`'s unheld branch**, and knowingly so: that function reads the
// holder's relics and an essence's scaling, neither of which an enemy card has — an enemy wears no
// relics and no essence reaches its deck. Importing `internal/screens` to reach the real one would
// pull Ebitengine into a review tool, which is the thing every sheet here is built to avoid.
// `tools/cardsheet` keeps its own snapshot of the player's wording for the same reason. If the
// two ever disagree, prose.go is the one that is right.
func effectText(c data.CardData) string {
	verb, ok := combat.ParseVerb(c.Verb)
	if !ok {
		// Unreachable in practice: importing internal/decks registers every card at init and a bad
		// verb panics there. Said anyway, so a future pool that skips registration cannot draw a
		// card claiming to hit for 0x.
		return "unknown verb " + strconv.Quote(c.Verb)
	}

	switch verb {
	case combat.VerbShield:
		return shieldText(c.Amount)
	default:
		return attackVerb(c.Form) + " for " + multiplier(c.Amount) + " DMG"
	}
}

func attackVerb(f string) string {
	switch f {
	case combat.FormStab.String():
		return "Stabs"
	case combat.FormSlash.String():
		return "Slashes"
	case combat.FormCrush.String():
		return "Crushes"
	default:
		return "Hits"
	}
}

// multiplier writes an Amount the way the card face does: percent of the wielder's DMG, so 100
// is 1x and 250 is 2.5x.
func multiplier(amount int) string {
	whole, frac := amount/100, amount%100
	switch {
	case frac == 0:
		return strconv.Itoa(whole) + "x"
	case frac%10 == 0:
		return fmt.Sprintf("%d.%dx", whole, frac/10)
	default:
		return fmt.Sprintf("%d.%02dx", whole, frac)
	}
}

// form maps a card's form name onto the drawing package's enum. **Every enemy card is formless
// today** — a form is the player's deck axis and the thing a hand is counted on — so this is the
// seat rather than a live mapping, and a record that grows one gets its corner mark for free.
func form(name string) cards.Form {
	for _, f := range cards.Forms() {
		if f != cards.FormNone && f.String() == name {
			return f
		}
	}
	return cards.FormNone
}

// element is which color to draw the card in. **Empty means basic**, exactly as the deck builder
// reads it — see internal/decks. Only the first is drawn: a concept shipping in several colors is
// several cards in the pile, and a strip showing one of each would say a swarm was a rainbow.
func element(names []string) cards.Element {
	if len(names) == 0 {
		return cards.Basic
	}
	for _, e := range cards.Elements() {
		if e.String() == names[0] {
			return e
		}
	}
	return cards.Basic
}

// deckRows is the table under a strip: one row per concept, in the order the strip draws them.
func deckRows(list []data.CardData) []row {
	out := make([]row, 0, len(list))
	for _, c := range list {
		out = append(out, row{
			Label:  c.Label,
			Verb:   c.Verb,
			Effect: effectText(c),
			Cost:   c.Cost,
			Copies: c.Copies,
		})
	}
	return out
}

// artwork decodes one embedded portrait. **A key that is in no embed is an error rather than a
// blank face**, unlike the game, which logs and draws the hole: the game must not refuse to start
// over a picture, and a review tool that quietly drew nothing would be hiding exactly what it is
// for. The portraits are keyed by filename stem — see assets/embed.go — so this is also the check
// that a renamed file and its JSON are still in step.
func artwork(key string) (image.Image, error) {
	raw := assets.LoadImageData()[key]
	if len(raw) == 0 {
		// **The placeholder rather than an error.** Nearly every record's own picture is still to
		// be generated, and a sheet that refused to build until the last one existed would be a
		// sheet nobody could use while the art was being made.
		raw = assets.LoadImageData()[data.DefaultEnemyArt]
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("no embedded picture called %q and no placeholder either", key)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", key, err)
	}
	return img, nil
}

func writePNG(path string, img image.Image) (int64, error) {
	f, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("creating %s: %w", path, err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return 0, fmt.Errorf("encoding %s: %w", path, err)
	}
	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// styleFacts is the numbers the page prints, read off the style rather than typed into the
// template, so the page cannot quote a card it is not showing.
func styleFacts(st cards.Style) map[string]int {
	return map[string]int{
		"width":     st.Width,
		"height":    st.Height,
		"artTop":    st.ArtTop,
		"artMaxH":   st.ArtMaxH,
		"nameSize":  int(st.NameSize),
		"healthTop": st.HealthBarTop,
	}
}

// shieldText is what a shield card's face says, and it is shared with internal/screens through
// nothing at all — a roster card and a played card are drawn by two packages that may not import
// each other, so the wording is written twice on purpose and kept identical by
// TestARosterCardSaysWhatAPlayedCardSays.
func shieldText(n int) string {
	if n == 1 {
		return "1 shield"
	}
	return strconv.Itoa(n) + " shields"
}
