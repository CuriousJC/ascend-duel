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
// **These are written out rather than read from data/cards.json and internal/combat.**
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

// concept is one of the twelve, with the cost and category the rules give it.
type concept struct {
	name     string
	category cards.Category
	cost     int
	damage   int // zero draws no damage badge
}

// The twelve concepts, in cards.json's order, which is grid order.
var concepts = []concept{
	{"Gather", cards.CategoryPrepare, 1, 0},
	{"Sift", cards.CategoryPrepare, 2, 0},
	{"Guard", cards.CategoryDefend, 3, 0},
	{"Ritual", cards.CategoryPrepare, 4, 0},
	{"Jab", cards.CategoryAttack, 1, 4},
	{"Strike", cards.CategoryAttack, 2, 7},
	{"Feint", cards.CategoryAttack, 3, 5},
	{"Heavy", cards.CategoryAttack, 4, 14},
	{"Brace", cards.CategoryDefend, 1, 0},
	{"Dodge", cards.CategoryDefend, 2, 0},
	{"Riposte", cards.CategoryDefend, 3, 6},
	{"Mirror", cards.CategoryDefend, 4, 0},
}

// realCards is a spread chosen to break things: the longest name, the biggest damage
// figure, both cost extremes, and cards with no damage badge at all.
func realCards() []cards.Spec {
	pick := []struct {
		name string
		el   cards.Element
	}{
		{"Jab", cards.Basic},
		{"Heavy", cards.Fire},
		{"Guard", cards.Ice},
		{"Riposte", cards.Lightning},
		{"Gather", cards.Earth},
		{"Ritual", cards.Basic},
	}
	out := make([]cards.Spec, 0, len(pick))
	for _, p := range pick {
		out = append(out, specFor(p.name, p.el))
	}
	return out
}

// realDeckRow is one element's worth of the deck: all twelve concepts, which is exactly
// what the overlay's row for that element holds when nothing has been drawn yet.
func realDeckRow(e cards.Element) []cards.Spec {
	out := make([]cards.Spec, 0, len(concepts))
	for _, c := range concepts {
		out = append(out, cards.Spec{
			Name: c.name, Category: c.category,
			Damage: c.damage, Cost: c.cost, Element: e, Enabled: true,
		})
	}
	return out
}

func specFor(name string, e cards.Element) cards.Spec {
	for _, c := range concepts {
		if c.name == name {
			return cards.Spec{
				Name: c.name, Category: c.category,
				Damage: c.damage, Cost: c.cost, Element: e, Enabled: true,
			}
		}
	}
	return cards.Spec{Name: name, Element: e, Enabled: true}
}

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
