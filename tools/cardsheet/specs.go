package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/internal/cards"
)

// What the sheet renders, kept apart from how it renders it.
//
// **These are written out rather than read from data/duelist_cards.json and internal/combat.**
// The tool could import both and build the real deck, and it deliberately does not: a
// review sheet that derives its contents from the rules can only show what the rules
// currently produce, and the whole point is to look at cards the game cannot deal yet —
// a fifth cost tier, a ring, a border colour nothing uses. It is a drawing-board, not a
// report.
//
// The cost of that is drift: the names and costs below are a snapshot of the twelve
// concepts as of 2026-08-09. If they stop matching the deck it makes the sheet a worse
// preview but never a wrong one, because every pixel still comes from cards.Render.

// glyphNames says what art each category actually draws, for the caption. Worth spelling
// out on the sheet, because "attack" and "a sword" are the kind of pairing that is
// obvious to whoever wired it and opaque a month later.
var glyphNames = map[cards.Category]string{
	cards.CategoryPrepare: "an open book",
	cards.CategoryAttack:  "a sword",
	cards.CategoryDefend:  "a kite shield",
}

// concept is one of the twelve, with the cost, category and effect text the rules give it.
//
// **No damage figure**: a card does not carry one any more, and what it deals is in the text.
type concept struct {
	name     string
	category cards.Category
	cost     int
	text     string
}

// The twelve concepts, in cards.json's order, which is grid order.
//
// **The effect text is a snapshot of `cardEffects` in internal/screens**, under the same rule
// as the names and costs above it: the tool does not import the game so it can draw cards the
// rules cannot deal. It is the longest strings here that matter — the sheet is where an
// overlong line is *seen* rather than merely failing a test.
var concepts = []concept{
	{"Gather", cards.CategoryPrepare, 1, "Bank 2 AP for next round"},
	{"Sift", cards.CategoryPrepare, 2, "Throw away 2 more cards, then refill"},
	// **Guard is a prepare, not a defend** — combat.ActionKind.Category() groups it with
	// Gather, Sift and Ritual. This list said Defend from the day it was written and the sheet
	// duly drew it with a kite shield where the game draws an open book. The drift this file
	// accepts is a card the rules cannot *deal*; a card drawn with the wrong glyph is a
	// picture that lies, which is the one thing a contact sheet may never be. Corrected
	// 2026-08-12.
	{"Guard", cards.CategoryPrepare, 3, "Halve every attack next turn"},
	{"Ritual", cards.CategoryPrepare, 4, "Bank 6 AP for next round"},
	{"Jab", cards.CategoryAttack, 1, "Deal 0.5x DMG"},
	{"Strike", cards.CategoryAttack, 2, "Deal 1x DMG"},
	{"Feint", cards.CategoryAttack, 3, "Strip a defend card, deal 1x DMG"},
	{"Heavy", cards.CategoryAttack, 4, "Deal 2x DMG"},
	{"Brace", cards.CategoryDefend, 1, "Halve 1 incoming attack"},
	{"Dodge", cards.CategoryDefend, 2, "Negate 1 incoming attack"},
	{"Riposte", cards.CategoryDefend, 3, "Negate 1 attack, deal 0.5x DMG back"},
	{"Retreat", cards.CategoryDefend, 4, "Negate 3 incoming attacks"},
}

// realCards is **all twelve concepts at hand size**, one element after another so the row
// also walks the border colours.
//
// **It was a spread of six until 2026-08-14**, chosen to break the layout: the longest name,
// the biggest damage badge, both cost extremes. That was the right row while the only thing
// varying between concepts was furniture the grid rows above already covered. It is the wrong
// row now that every card carries its own paragraph — the wording is the thing being reviewed,
// and six of twelve meant half of it could only be read in the source. Sift and Feint, the two
// longest strings in the game, were among the six that never appeared.
func realCards() []cards.Spec {
	elements := cards.Elements()

	out := make([]cards.Spec, 0, len(concepts))
	for i, c := range concepts {
		out = append(out, specFor(c.name, elements[i%len(elements)]))
	}
	return out
}

// realDeckRow is one element's worth of the deck: all twelve concepts, which is exactly
// what the overlay's row for that element holds when nothing has been drawn yet.
func realDeckRow(e cards.Element) []cards.Spec {
	out := make([]cards.Spec, 0, len(concepts))
	for _, c := range concepts {
		out = append(out, cards.Spec{
			Name: c.name, Category: c.category, Text: c.text,
			Cost: c.cost, Element: e, Enabled: true,
		})
	}
	return out
}

func specFor(name string, e cards.Element) cards.Spec {
	for _, c := range concepts {
		if c.name == name {
			return cards.Spec{
				Name: c.name, Category: c.category, Text: c.text,
				Cost: c.cost, Element: e, Enabled: true,
			}
		}
	}
	return cards.Spec{Name: name, Element: e, Enabled: true}
}

// selected and disabled are the two states the sheet draws a real card in. Written as
// modifiers rather than as three separate literals so the card underneath is provably the
// same one in all three cells — which is the whole point of a state row.
func selected(s cards.Spec) cards.Spec { s.Selected = true; return s }

func disabled(s cards.Spec) cards.Spec { s.Enabled = false; return s }

// ringSpecs is the first pass at a ring, in the card format.
//
// The art is assets/fire-ring.png. **That is first-party work** — README credits the art
// to CuriousJC and KingSherman1820, and only the sheets prefixed `tyrian_` come from the
// Tyrian set. So it carries no provenance question and is not part of the release blocker
// that set represents; it can ship.
//
// No cost dashes, no category glyph, no damage badge: a ring is not played from a hand
// and has no phase. What it keeps is the footprint, the corners and the border, so it
// reads as the same game.
func ringSpecs() ([]cards.Spec, error) {
	art, err := loadPNG("firering_png")
	if err != nil {
		return nil, err
	}
	return []cards.Spec{
		{Name: "Fire Ring", Element: cards.Ring, Art: art, Enabled: true},

		// The same ring mid-drag. **Not "not equipped"** — a ring you do not have is not
		// shown at all, so that state does not exist to draw. Being carried by the cursor
		// does exist, and it is the one thing a ring in a card format has to look like
		// besides sitting still.
		{Name: "Fire Ring", Element: cards.Ring, Art: art, Enabled: true, Dragging: true},
	}, nil
}

// enemySpecs is the opponent in the card format, at four states of health.
//
// **Four, because the health bar is the only thing on this card that moves**, and a bar is
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
	}{
		{"Giant Rat", "giantrat-portrait", 50, 50},
		{"Dragonfly", "dragonfly-portrait", 33, 60},
		{"Ogre Warlord", "ogrewarlord-portrait", 4, 160},
		{"Bio-Titan Omega", "biotitan-omega-portrait", 137, 200},
	}

	var specs []cards.Spec
	for _, e := range each {
		art, err := loadPNG(e.key)
		if err != nil {
			return nil, err
		}
		specs = append(specs, cards.Spec{
			Name: e.name, Element: cards.Basic, Art: art,
			Life: e.life, MaxLife: e.of, Enabled: true,
		})
	}
	return specs, nil
}

// duelistSpecs is the player in the card format, at four points in a run.
//
// **The figures are the only thing on this card that moves**, so the spread is chosen to
// stretch them rather than to be typical: the starting duelist, one that has taken damage and
// banked a Gather, a long name against a two-digit column, and a nearly-dead one to put the
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
