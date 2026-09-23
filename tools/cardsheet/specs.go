package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
)

// titled capitalises a rarity for the caption and the filename. **Hand-rolled rather than
// strings.Title**, which is deprecated, or golang.org/x/text/cases, which is a dependency for
// three words that are known to be ASCII.
func titled(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// What the sheet renders, kept apart from how it renders it.
//
// **These are written out rather than read from data/duelist_cards.json and internal/combat.**
// The tool could import both and build the real deck, and it deliberately does not: a
// review sheet that derives its contents from the rules can only show what the rules
// currently produce, and the whole point is to look at cards the game cannot deal yet —
// a fifth cost tier, a relic, a border color nothing uses. It is a drawing-board, not a
// report.
//
// The cost of that is drift: the names and costs below are a snapshot of the nineteen
// concepts as of 2026-09-06. If they stop matching the deck it makes the sheet a worse
// preview but never a wrong one, because every pixel still comes from cards.Render.

// formNotes says what mark each form actually draws, for the caption. Worth spelling out on
// the sheet, because a bare "S" in a corner is obvious to whoever wired it and opaque a month
// later — and doubly so for Slash, which carries a D.
var formNotes = map[cards.Form]string{
	cards.FormStab:   `a spear, tinted by the card's element`,
	cards.FormSlash:  `a sword, tinted by the card's element`,
	cards.FormCrush:  `an axe, tinted by the card's element`,
	cards.FormDefend: `a shield, tinted by the card's element`,
}

// concept is one of the nineteen, with the cost, form, effect text and amount the rules give it.
//
// **The amount came back on 2026-09-16**, having been dropped when the damage badge was removed in
// 2026-08-14. The badge returned, carrying the multiplier for an attack and a stack of shields for
// a defense, so the sheet has to know the figure again — it is the same snapshot-of-the-rules this
// table already is for costs and names, and it is written out here rather than read from
// `data/duelist_cards.json` for the reason at the top of this file: the sheet is a drawing-board
// and has to be able to draw a card the rules cannot deal.
type concept struct {
	name string
	form cards.Form
	cost int
	text string

	// amount is the card's own figure, as the rules write it: a percentage for an attack, where 25
	// is a quarter of DMG and 400 is four times it, and a plain count for a defense.
	amount int

	// shield says which of the two amount is. A defense stacks that many shields in the corner; an
	// attack draws one badge carrying its multiplier.
	shield bool
}

// The nineteen concepts, in duelist_cards.json's order, which is grid order: three attack forms
// of five tiers, then the defenses.
//
// **Three forms by five tiers, and the tiers cost and hit the same in each** *(2026-08-24)*.
// 0 AP is a quarter, 1 AP is half, 2 AP is one, 3 AP is two, 4 AP is four, in Stab and Slash and
// Crush alike. So a form is *which* pair you are building rather than a stronger or weaker way to
// build one, and the only thing separating Skewer from Cleave is what it pairs with.
//
// **The defenses are a four-rung ladder of their own** *(2026-09-06)*, Flinch through Guard, with
// the shield count where the attacks have a damage multiplier — so an Exalt or a Debase walks it too.
//
// **The outer rungs ship at zero copies** — both ends of each attack form, and Flinch and Guard —
// so they are on this sheet and not in any deck. This
// is where they get looked at, and there are two specific things to look for: a 0 AP card draws an
// empty cost column, and a 4 AP card is the only one that stacks four ticks.
//
// **The effect text is a snapshot of `cardEffects` in internal/screens**, under the same rule
// as the names and costs above it: the tool does not import the game so it can draw cards the
// rules cannot deal. It is the longest strings here that matter — the sheet is where an
// overlong line is *seen* rather than merely failing a test.
var concepts = []concept{
	{"Poke", cards.FormStab, 0, "", 25, false},
	{"Jab", cards.FormStab, 1, "", 50, false},
	{"Thrust", cards.FormStab, 2, "", 100, false},
	{"Skewer", cards.FormStab, 3, "", 300, false},
	{"Impale", cards.FormStab, 4, "", 400, false},

	{"Nick", cards.FormSlash, 0, "", 25, false},
	{"Cut", cards.FormSlash, 1, "", 50, false},
	{"Slice", cards.FormSlash, 2, "", 100, false},
	{"Cleave", cards.FormSlash, 3, "", 300, false},
	{"Sever", cards.FormSlash, 4, "", 400, false},

	{"Tap", cards.FormCrush, 0, "", 25, false},
	{"Thump", cards.FormCrush, 1, "", 50, false},
	{"Bash", cards.FormCrush, 2, "", 100, false},
	{"Smash", cards.FormCrush, 3, "", 300, false},
	{"Pulverize", cards.FormCrush, 4, "", 400, false},

	{"Flinch", cards.FormDefend, 0, "", 1, true},
	{"Brace", cards.FormDefend, 1, "", 1, true},
	{"Block", cards.FormDefend, 2, "", 2, true},
	{"Guard", cards.FormDefend, 3, "", 3, true},
}

// realCards is **all nineteen concepts at hand size**, one element after another so the row
// also walks the border colors.
//
// **It was a spread of six until 2026-08-14**, chosen to break the layout: the longest name,
// the biggest damage badge, both cost extremes. That was the right row while the only thing
// varying between concepts was furniture the grid rows above already covered. It is the wrong
// row now that every card carries its own paragraph — the wording is the thing being reviewed,
// and a sample of six of twelve meant half of it could only be read in the source — including
// whichever strings were longest, which are the ones a layout breaks on.
func realCards() []cards.Spec {
	elements := cards.Elements()

	out := make([]cards.Spec, 0, len(concepts))
	for i, c := range concepts {
		out = append(out, specFor(c.name, elements[i%len(elements)]))
	}
	return out
}

// realDeckRow is one element's worth of the deck: every concept, which is a rung wider each end
// than the overlay's row for that element holds when nothing has been drawn yet — the zero-copy
// ends are on the sheet precisely because no deck starts with them.
func realDeckRow(e cards.Element) []cards.Spec {
	out := make([]cards.Spec, 0, len(concepts))
	for _, c := range concepts {
		out = append(out, specOf(c, e))
	}
	return out
}

func specFor(name string, e cards.Element) cards.Spec {
	for _, c := range concepts {
		if c.name == name {
			return specOf(c, e)
		}
	}
	return cards.Spec{Name: name, Element: e, Enabled: true}
}

// specOf is the one place a concept and an element become a face, so every table on the page draws
// the card the same way and the sheet cannot disagree with itself.
//
// **It reads the real catalog for the picture.** `data/card_art.json` is one record per card per
// element, and the whole point of the sheet is to look at what the game will actually show — so a
// pairing nobody has drawn yet comes back with no art and draws as the card always did, exactly as
// it does in a duel. See data.DefaultCardArt, which is empty on purpose.
func specOf(c concept, e cards.Element) cards.Spec {
	spec := cards.Spec{
		Name: c.name, Form: c.form, Text: c.text,
		Highlights: cards.ElementHighlights(c.text),
		Cost:       c.cost, Element: e, Enabled: true,
		Art: cardArtwork(c.name, e),
	}
	if c.shield {
		spec.Shields = c.amount
	} else {
		spec.Badge, spec.BadgePct = multiplier(c.amount), c.amount
	}
	return spec
}

// multiplier writes an attack's amount the way the card does: a fraction under one, a whole number
// at or above it.
//
// **A snapshot of `screens.damageMultiplier`**, under the same rule the names, costs and effect
// text are under — the sheet does not import the game, so that it can draw a card the rules cannot
// deal. If the two ever disagree it makes the sheet a worse preview, never a wrong one, because
// every pixel still comes from cards.Render.
func multiplier(pct int) string {
	switch pct {
	case 25:
		return "1/4"
	case 50:
		return "1/2"
	case 75:
		return "3/4"
	}
	if pct%100 == 0 {
		return strconv.Itoa(pct / 100)
	}
	return strconv.FormatFloat(float64(pct)/100, 'g', -1, 64)
}

// cardArt and cardImages are the catalog and the picture bank, loaded once.
var (
	cardArt    = data.LoadCardArt()
	cardImages = assets.LoadImageData()
	artCache   = map[string]image.Image{}
)

// cardArtwork is the full-bleed picture for one card in one element, or nil for a pairing nobody
// has drawn. **Through the catalog's own ArtKey**, so the fallback the sheet shows is the fallback
// the game shows.
func cardArtwork(name string, e cards.Element) image.Image {
	rec, ok := cardArt[data.CardArtKey(name, e.String())]
	if !ok {
		return nil
	}
	key := rec.ArtKey()
	if key == "" {
		return nil
	}
	if img, seen := artCache[key]; seen {
		return img
	}
	raw := cardImages[key]
	if len(raw) == 0 {
		fmt.Printf("cardsheet: %s names no file %q\n", rec.CardArtRecord, key)
		artCache[key] = nil
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		fmt.Printf("cardsheet: decoding %q: %v\n", key, err)
		img = nil
	}
	artCache[key] = img
	return img
}

// selected and disabled are the two states the sheet draws a real card in. Written as
// modifiers rather than as three separate literals so the card underneath is provably the
// same one in all three cells — which is the whole point of a state row.
func selected(s cards.Spec) cards.Spec { s.Selected = true; return s }

func disabled(s cards.Spec) cards.Spec { s.Enabled = false; return s }

// relicSpecs is the first pass at a relic, in the card format.
//
// The art is assets/relic/dmgx-fire.png.
//
// No cost dashes, no category glyph, no damage badge: a relic is not played from a hand
// and has no phase. What it keeps is the footprint, the corners and the border, so it
// reads as the same game.
//
// **It draws one card per rarity** *(2026-09-13)*, because the border is what a rarity says now
// and three colors side by side is the only way to review whether they are actually telling
// each other apart. One picture is used for all of them on purpose: the art is the variable being
// held still so the ring is the thing being compared.
func relicSpecs() ([]cards.Spec, error) {
	art, err := loadPNG("dmgx-fire")
	if err != nil {
		return nil, err
	}
	// **The name is set and never drawn.** A relic card is a full-bleed picture with no title, so
	// Spec.Name is what a mark's pattern is derived from and what names the file — which is why
	// these are named for the rarity rather than for the relic: three cards called "Fire" would be
	// one file written three times.
	var out []cards.Spec
	for _, r := range data.Rarities() {
		out = append(out, cards.Spec{
			Name: titled(string(r)), Element: cards.Relic, Rarity: r,
			Art: art, Enabled: true,
		})
	}

	// The same relic mid-drag. **Not "not equipped"** — a relic you do not have is not
	// shown at all, so that state does not exist to draw. Being carried by the cursor
	// does exist, and it is the one thing a relic in a card format has to look like
	// besides sitting still.
	return append(out, cards.Spec{
		Name: "Rare", Element: cards.Relic, Rarity: data.Rare,
		Art: art, Enabled: true, Dragging: true,
	}), nil
}

// enemySpecs is the opponent in the card format, at four states of health and four counts of
// status badge.
//
// **The badges are drawn at every count from none to four** *(2026-08-16)*, because the row is
// centered and closes up as it fills — the same property the relic row has, and the same failure
// available: a row laid out against the maximum leaves a single badge hard left. Twenty pixels
// is small enough that this is a thing to look at rather than to reason about.
//
// **Four states of health, because the bar is the other thing on this card that moves**, and a bar is
// exactly the widget where full and empty both look fine and the middle is where the
// arithmetic is wrong — so full, most, a third and nearly dead. The names are drawn from
// `data/enemies.json`, but the life totals here are illustrative: the sheet draws what a
// card *can* look like, not what any particular fight has reached.
//
// The keys are portrait filenames rather than hand-written asset names, because the 96
// portraits are embedded as a glob. A renamed file breaks this list, and that is the
// documented cost of not maintaining 192 lines of embed directives.
func enemySpecs() ([]cards.Spec, error) {
	each := []struct {
		name     string
		key      string
		life, of int

		// The status badges along the bottom, by assets key. **The counts are what matter
		// here** — one badge and four are different layouts, and a row that is right at four
		// and off-center at one is the failure mode the sheet exists to catch by eye.
		effects []string
	}{
		{"Giant Rat", "giantrat-portrait", 50, 50, nil},
		{"Dragonfly", "dragonfly-portrait", 33, 60, []string{"fireeffect_png"}},
		{"Ogre Warlord", "ogrewarlord-portrait", 4, 160, []string{
			"fireeffect_png", "frozeneffect_png"}},
		{"Bio-Titan Omega", "biotitan-omega-portrait", 137, 200, []string{
			"fireeffect_png", "frozeneffect_png", "thundereffect_png", "eartheffect_png",
			"defaulteffect_png"}},
	}

	var specs []cards.Spec
	for _, e := range each {
		art, err := loadPNG(e.key)
		if err != nil {
			return nil, err
		}
		spec := cards.Spec{
			Name: e.name, Element: cards.Basic, Art: art,
			Life: e.life, MaxLife: e.of, Enabled: true,
		}
		for i, key := range e.effects {
			badge, err := loadPNG(key)
			if err != nil {
				return nil, err
			}
			spec.Effects[i] = badge
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

// duelistSpecs is the player in the card format, at four points in a run.
//
// **The figures are the only thing on this card that moves**, so the spread is chosen to
// stretch them rather than to be typical: the starting duelist, one that has taken damage and
// spent less of its budget, a long name against a two-digit column, and a nearly-dead one to put the
// health bar near empty beside a full one. The last is where a bar's arithmetic goes wrong.
//
// Same caveat as the concepts above: these numbers are illustrative and not read from
// data/duelists.json. The sheet draws what a card *can* look like.
func duelistSpecs() []cards.Spec {
	each := []struct {
		name           string
		dmg, ap, vitae int
		life, of       int
	}{
		{"Duelist", 10, 6, 5, 120, 120},
		{"Duelist", 10, 8, 3, 74, 120},
		{"Stormcaller", 24, 11, 12, 96, 180},
		{"Duelist", 7, 4, 0, 6, 120},
	}

	out := make([]cards.Spec, 0, len(each))
	for _, d := range each {
		spec := cards.Spec{
			Name: d.name, Element: cards.Basic,
			Life: d.life, MaxLife: d.of, Enabled: true,
		}
		spec.Stats[0] = cards.StatLine{Label: "DMG", Value: fmt.Sprintf("%d", d.dmg)}
		spec.Stats[1] = cards.StatLine{Label: "AP", Value: fmt.Sprintf("%d", d.ap)}
		spec.Stats[2] = cards.StatLine{Label: "Vitae", Value: fmt.Sprintf("%d", d.vitae)}
		out = append(out, spec)
	}
	return out
}

func loadPNG(key string) (image.Image, error) {
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
