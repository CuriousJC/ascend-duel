package screens

import (
	"bytes"
	"image"
	_ "image/png"
	"log"
	"strconv"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
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
// **The card carries no damage figure** *(2026-08-14)*. It used to resolve `Damage(str)` here,
// because the number a card deals is a property of the pairing rather than of the concept —
// and that is exactly what made it worth removing once the effect text arrived: "Deal 2x DMG"
// is the rule, where "14" was the rule already multiplied out by this duelist's strength and
// was the same fact said twice. `combat.ActionKind.Damage` is still what the engine resolves
// with, and the duelist card still shows a DMG stat.
func cardSpec(c actionCard, enabled, selected bool) cards.Spec {
	return cards.Spec{
		Name:     c.Action.String(),
		Category: category(c.Action.Category()),
		Cost:     c.Action.Cost(),
		Element:  artFor(c.Element),
		Text:     cardEffects[c.Action],
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

// artworkCache holds the decoded pictures that go *on* a card — enemy portraits, ring art —
// keyed by their assets name.
//
// **Decoded once and held**, for the same reason the cards themselves are cached: these are
// 320-pixel PNGs and `image.Decode` is not a per-frame operation. They are handed to
// internal/cards as plain `image.Image`, which is why they come out of `LoadImageData` as
// bytes rather than out of `LoadAssets` as *ebiten.Image — a card is drawn with no graphics
// context.
//
// **One cache for both, rather than one per kind of art.** It was `portraitCache` until the
// ring pane arrived on 2026-08-11; a second map would have been the same six lines keyed the
// same way, and the thing they have in common — a file that has to be decoded before it can
// be drawn into a card — is the whole of what either needs.
//
// A failure is cached as nil so a bad file logs once rather than sixty times a second, and
// the card then draws with no picture rather than not at all.
var artworkCache = map[string]image.Image{}

func artwork(gs *state.GlobalState, key string) image.Image {
	if key == "" {
		return nil
	}
	if img, ok := artworkCache[key]; ok {
		return img
	}

	data := gs.ImageData[key]
	if len(data) == 0 {
		log.Printf("cards: no artwork named %q", key)
		artworkCache[key] = nil
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Printf("cards: decoding artwork %q: %v", key, err)
		img = nil
	}
	artworkCache[key] = img
	return img
}

// enemySpec is the opponent as a card: its portrait, its name, and the life it has left.
//
// Life is on the Spec, so a point of damage produces a different cache entry — see the
// field's comment in internal/cards. Bounded by how many distinct life totals a fight passes
// through, which is a handful.
func enemySpec(gs *state.GlobalState, c *entities.Combatant, name string) cards.Spec {
	return cards.Spec{
		Name:    name,
		Element: cards.Basic,
		Art:     artwork(gs, c.Portrait),
		Life:    c.CurrentLife,
		MaxLife: c.MaxLife,
		Enabled: true,
	}
}

// duelistSpec is the player as a card: their name, three figures, and the life they have
// left.
//
// **DMG is asked of the rules rather than written here.** `combat.Strike.Damage(Str)` is what
// one plain attack does in these hands, which is the "1x base damage" the figure means — so a
// change to how strength becomes damage moves this number without anyone remembering that a
// card shows it. Writing `c.Str` would have been the same integer today and a lie tomorrow.
//
// AP is the live budget, `BonusAP` included, so a Gather banked last round shows up on the
// card before it is spent. Vitae is passed in rather than read off the combatant because it is
// run-level state that does not live on a duelist yet — see startingVitae.
//
// Every distinct set of figures is a cache entry, like the enemy's life. Bounded by how many
// values a fight passes through, which is a handful.
func duelistSpec(c *entities.Combatant, name string, vitae int) cards.Spec {
	spec := cards.Spec{
		Name:    name,
		Element: cards.Basic,
		Life:    c.CurrentLife,
		MaxLife: c.MaxLife,
		Enabled: true,
	}
	spec.Stats[0] = cards.StatLine{Label: "DMG", Value: strconv.Itoa(combat.Strike.Damage(c.Str))}
	spec.Stats[1] = cards.StatLine{Label: "AP", Value: strconv.Itoa(c.ActionPoints())}
	spec.Stats[2] = cards.StatLine{Label: "Vitae", Value: strconv.Itoa(vitae)}
	return spec
}

// ringSpec is an equipped ring as a card: its name and its artwork, and nothing else.
//
// **The element on the record does not reach the Spec**, deliberately. `cards.Ring` is the
// element a ring card carries, which paints the border pink whatever the ring is about — the
// one thing that must never happen is reaching for a ring thinking it is a card you can play.
// `RingData.Element` says which element the ring will eventually *discount*; it is a rule, not
// a colour, and it has nowhere to be read yet.
//
// No cost, no category, no damage: a ring is not played from a hand and has no phase.
func ringSpec(gs *state.GlobalState, r data.RingData) cards.Spec {
	return cards.Spec{
		Name:    r.Name,
		Element: cards.Ring,
		Art:     artwork(gs, r.Art),
		Enabled: true,
	}
}

// backSpec is a face-down card of this duelist's deck.
//
// **A duelist and a card back go together** *(2026-08-11)*: the plan is to offer different
// duelists as different decks, and the mark on the back is how you tell at a glance whose
// deck is on the table. The name comes from `data/duelists.json` and is parsed here rather
// than at load, because `internal/entities` must not import the drawing package — the same
// separation the element mapping below exists for.
//
// An unrecognised name falls back to the triangle and says so once. A back is cosmetic;
// refusing to draw the draw pile over one would be a worse outcome than the wrong shape.
func (s *CombatScene) backSpec() cards.Spec {
	mark, ok := cards.ParseBackMark(s.fighter.CardBack)
	if !ok && s.fighter.CardBack != "" && !warnedBack {
		warnedBack = true
		log.Printf("cards: duelist card back %q is not a mark; using %v",
			s.fighter.CardBack, mark)
	}
	return cards.Spec{FaceDown: true, Back: mark}
}

// warnedBack keeps a bad name in duelists.json to one log line rather than one per frame.
var warnedBack bool

// artFor maps the rules' element onto the drawing package's.
//
// **The two enums stay separate**, and the reason changed on 2026-08-12 rather than expiring.
// It used to be that `element` was a screen type on its way into `internal/combat` and a drawing
// package should not stand in the way of the move. The move has happened, and the separation is
// now the ordinary one `category` below has: `internal/cards` knows how to paint a border and
// nothing about what an element does to a duelist. The cost is this switch;
// TestEveryElementHasItsOwnArt keeps it honest.
//
// A free function rather than a method, because a method cannot be hung on another package's
// type — which is the one thing the collapse to `combat.Element` cost.
//
// The default is Basic rather than a panic. An unmapped element is a card in the wrong
// colour, which is a visual bug; crashing mid-duel over one would be worse.
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
		// Includes combat.Poison, which has no border colour of its own any more.
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
