package screens

import (
	"log"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

// The bridge between this screen and internal/cards.
//
// internal/cards draws a card into a plain Go image so that `go run ./tools/cardsheet`
// can render one without a window. That is the right trade for a review tool and the
// wrong one for a game loop: every pixel of the shape is written in Go and the text is
// rasterised, so building a card costs far too much to do sixty times a frame. This file
// is the cache that makes it affordable, and it is the only thing the game adds.

// cardKey identifies a rendered card. Two cards that key the same are the same picture.
//
// The Spec is the whole of what cards.Render looks at, so it can be the whole of the key
// — it is comparable, being strings, ints and bools. The style is not comparable in a
// useful way, so its footprint stands in for it; the two styles differ in size, and a
// third that did not would be the same card twice.
type cardKey struct {
	spec cards.Spec
	w, h int
}

// cardCache is package state rather than a field on CombatScene, because the pictures
// outlive any one visit to the screen and re-entering it should not repaint sixty cards.
// Bounded in practice by the deck: 60 cards times two sizes times three states.
var cardCache = map[cardKey]*ebiten.Image{}

// cardFaces is the font internal/cards sets its text with, built once on first use.
// nil after a failure, which is checked rather than retried — a font that will not parse
// will not parse the second time either, and retrying it every frame would turn one log
// line into thousands.
var (
	cardFaces  *cards.Faces
	facesTried bool
)

func faces(gs *state.GlobalState) *cards.Faces {
	if facesTried {
		return cardFaces
	}
	facesTried = true

	ttf := gs.FontData["kubasta"]
	if len(ttf) == 0 {
		log.Println("cards: no kubasta font data; cards will not be drawn")
		return nil
	}
	f, err := cards.NewFaces(ttf)
	if err != nil {
		log.Println(err)
		return nil
	}
	cardFaces = f
	return cardFaces
}

// cardSpec turns the screen's own types into the plain data internal/cards draws from.
//
// Damage is resolved here rather than passed through, because it is a property of the
// pairing: Damage(str) is what this card does in *these* hands. Cost is not, so it is
// read straight off the action.
func cardSpec(c actionCard, enabled, selected bool, str int) cards.Spec {
	return cards.Spec{
		Name:     c.action.String(),
		Category: category(c.action.Category()),
		Damage:   c.action.Damage(str),
		Cost:     c.action.Cost(),
		Element:  c.element.art(),
		Enabled:  enabled,
		Selected: selected,
	}
}

// cardImage returns the card for this spec, rendering and caching it on a miss.
//
// Returns nil rather than a placeholder when the font is missing. drawCard checks for
// that and draws nothing: a card-shaped hole is a bug someone will report, where a card
// drawn in a fallback font is one they will not notice until a screenshot looks wrong.
func cardImage(gs *state.GlobalState, spec cards.Spec, st cards.Style) *ebiten.Image {
	key := cardKey{spec: spec, w: st.Width, h: st.Height}
	if img, ok := cardCache[key]; ok {
		return img
	}

	f := faces(gs)
	if f == nil {
		return nil
	}
	rendered, err := cards.Render(spec, st, f)
	if err != nil {
		log.Println(err)
		cardCache[key] = nil // negative-cached, so a broken card logs once and not per frame
		return nil
	}

	img := ebiten.NewImageFromImage(rendered)
	cardCache[key] = img
	return img
}

// art maps this screen's element onto the drawing package's.
//
// The two enums are deliberately separate. `element` is expected to move into
// internal/combat once elements stop being decoration and start applying statuses (see
// combat_deck.go), and a drawing package should not be standing in the way of that move.
// The cost is this switch; TestEveryElementHasArt in the tests keeps it honest.
//
// The default is Basic rather than a panic. An unmapped element is a card in the wrong
// colour, which is a visual bug; crashing mid-duel over one would be worse.
func (e element) art() cards.Element {
	switch e {
	case elementFire:
		return cards.Fire
	case elementIce:
		return cards.Ice
	case elementLightning:
		return cards.Lightning
	case elementEarth:
		return cards.Earth
	default:
		// Includes elementPoison, which has no border colour of its own any more.
		// **Nothing deals a poison card** — cards.json contains none, and the enum member
		// survives only because MECHANICS.md lists poison as a secondary element that may
		// get cards later. If it ever does, it needs a colour in internal/cards and an arm
		// here, and TestEveryElementHasItsOwnArt will say so.
		return cards.Basic
	}
}

// category maps the rules' phase onto the drawing package's, which is drawn as a glyph:
// a sword for attack, a shield for defend, an open book for prepare.
//
// Two enums again, and for the same reason as the elements — internal/cards knows how to
// draw a card and nothing about how a round resolves. The default is CategoryNone, which
// draws no glyph at all rather than guessing at one.
func category(c combat.Category) cards.Category {
	switch c {
	case combat.CategoryPrepare:
		return cards.CategoryPrepare
	case combat.CategoryAttack:
		return cards.CategoryAttack
	case combat.CategoryDefend:
		return cards.CategoryDefend
	default:
		return cards.CategoryNone
	}
}

// Compile-time assurance that the action type still answers everything a Spec needs. If
// combat.ActionKind loses one of these, this fails here rather than in a card that
// silently renders blank.
var _ = func(a combat.ActionKind, str int) (string, string, int, int) {
	return a.String(), a.Category().String(), a.Damage(str), a.Cost()
}
