// Package roster renders one opponent-catalogue review sheet: every record in a pool, drawn
// as the card the game draws, beside the deck it fights with and the stats it fights on.
//
// # Why this is a library and the other sheets are not
//
// `tools/sheets` says the four existing sheets share nothing but the words `png.Encode`, and
// that a library between them would exist to be a seam that command already is. That argument
// is still right and this is not a counter-example to it: the enemy sheet and the boss sheet
// are not two sheets, they are **one sheet over two pools**. The two catalogues carry the same
// fields, are drawn by the same style, and are read to answer the same question. Copying four
// hundred lines so that the second one could differ in a heading and a floor field is how two
// pages that must agree quietly stop agreeing — a boss sheet that had not learned about a new
// column would show the game as it was.
//
// So the pool is the parameter and everything else is here. `tools/enemysheet` and
// `tools/bosssheet` are the two commands, and each is a `Pool` plus a `main`.
//
// # It is a report, like the ring and worm sheets
//
// It reads `data/enemies.json` and `data/bosses.json` rather than writing its own contents out.
// It also imports `internal/decks`, which registers every enemy and boss concept at init — so a
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
)

// Ground is the cream an opponent card is actually drawn on — `screens.screenGround`, the fill
// behind both corners of the combat screen. Judging a card against a white browser page would be
// the same failure as previewing art at a scale the game does not use.
const Ground = "#e2d0b0"

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

// Pool is one catalogue: what it is called, and how to read it.
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

	// Entries reads the catalogue, in the order the page should show it.
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

	Portrait string

	DMG     int
	Actions int
	HP      int

	// Group is the sort key the page's sections are cut on, and Floors is how that section is
	// written. A creature has a band and a boss has a single floor; both reduce to a heading and
	// a number to order it by.
	Group  int
	Floors string

	Affixes []string
	Cards   []data.CardData
}

// EnemyPool is every creature in data/enemies.json, in EnemyOrder — shallowest floor first.
var EnemyPool = Pool{
	Name:       "enemy",
	Title:      "Enemy sheet",
	GroupLabel: "floor band",
	Blurb: "Every creature in the roster, grouped by the floors it may appear on: its card as " +
		"the game draws it, its stat line, and the deck it fights with. The roster is " +
		"hand-assigned, so the question this page answers is whether a floor's creatures are " +
		"actually dearer than the floor below it.",
	Entries: func() []Entry {
		records := data.LoadEnemies()
		out := make([]Entry, 0, len(records))
		for _, key := range data.EnemyOrder(records) {
			r := records[key]
			out = append(out, Entry{
				Record: r.EnemyRecord, Name: r.Name, Portrait: r.Portrait,
				DMG: r.DMG, Actions: r.Actions, HP: r.HP,
				Group: r.ValidFloors[0], Floors: floorBand(r.ValidFloors),
				Affixes: r.AvailableAffixes, Cards: r.Cards,
			})
		}
		return out
	},
}

// BossPool is the thirty stairway protectors, in BossOrder — lowest floor first.
var BossPool = Pool{
	Name:       "boss",
	Title:      "Boss sheet",
	GroupLabel: "floor",
	Blurb: "The thirty stairway protectors, one of whom stands in the third room of every " +
		"floor: the card, the stat line and the deck. A boss is pitched above the creatures of " +
		"its own floor — roughly 1.6x their HP and 1.3x their DMG — so this page is read against " +
		"the enemy sheet's band of the same number.",
	Entries: func() []Entry {
		records := data.LoadBosses()
		out := make([]Entry, 0, len(records))
		for _, key := range data.BossOrder(records) {
			r := records[key]
			out = append(out, Entry{
				Record: r.BossRecord, Name: r.Name, Title: r.Title, Portrait: r.Portrait,
				DMG: r.DMG, Actions: r.Actions, HP: r.HP,
				Group: r.Floor, Floors: "Floor " + strconv.Itoa(r.Floor),
				Affixes: r.AvailableAffixes, Cards: r.Cards,
			})
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
	pg := page{
		Ground:     Ground,
		Title:      p.Title,
		Blurb:      p.Blurb,
		GroupLabel: p.GroupLabel,
		Count:      len(entries),
		Style:      styleFacts(cards.EnemyStyle),
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
			Affixes: strings.Join(e.Affixes, ", "),
			Deck:    len(decks.EnemyCards(e.Record)),
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
		fmt.Printf("  %-14s %2d record%s — HP %d-%d, DMG %d-%d, AP %d-%d\n",
			g.Label, len(g.Plates), plural, g.MinHP, g.MaxHP, g.MinDMG, g.MaxDMG, g.MinAP, g.MaxAP)
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
		img, err := cards.Render(cardSpec(c), cards.Hand, f)
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
// on a combatant, and nothing in a catalogue has been in one.
func opponentSpec(e Entry, art image.Image) cards.Spec {
	return cards.Spec{
		Name:    e.Name,
		Element: cards.Basic,
		Art:     art,
		Life:    e.HP,
		MaxLife: e.HP,
		Enabled: true,
	}
}

// cardSpec is one of an opponent's cards, drawn in the hand style the player's cards use.
//
// **The hand style on purpose**, even though nobody ever holds one of these: it is the only style
// that shows a cost column and an effect band, which is the whole of what there is to review about
// an enemy card. Drawing it in the style the player's deck is drawn in is also what lets the two
// be compared, which is the actual balance question.
func cardSpec(c data.CardData) cards.Spec {
	return cards.Spec{
		Name:    c.Label,
		Form:    form(c.Form),
		Cost:    c.Cost,
		Element: element(c.Elements),
		Text:    effectText(c),
		Enabled: true,
	}
}

// effectText is what the card face says it does.
//
// **A copy of `screens.cardEffect`'s unheld branch**, and knowingly so: that function reads the
// holder's rings and a worm's scaling, neither of which an enemy card has — an enemy wears no
// rings and no worm reaches its deck. Importing `internal/screens` to reach the real one would
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
	case combat.VerbDefend:
		return "Cuts damage by " + strconv.Itoa(c.Amount) + "%"
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

// element is which colour to draw the card in. **Empty means basic**, exactly as the deck builder
// reads it — see internal/decks. Only the first is drawn: a concept shipping in several colours is
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
		return nil, fmt.Errorf("no embedded portrait called %q", key)
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
