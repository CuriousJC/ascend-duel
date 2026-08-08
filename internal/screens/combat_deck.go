package screens

// The cards and the deck they come out of: what a card is made of, what the starting deck
// holds, the four piles and the movement between them, and the overlay that shows them.
//
// Split out of combat.go on 2026-08-07. The deck lives on the scene rather than in
// internal/combat on purpose — see CLAUDE.md on determinism. That is what keeps the rules
// package pure, testable and free of a shuffle, and it is why this file exists at all.

import (
	"fmt"
	"image"
	"sort"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"image/color"
)

// element is what a card is made of.
//
// **Still colour and nothing else in the code** — no rule reads it and ResolveRound never
// sees it, which is why it lives here on the screen rather than in internal/combat.
//
// That is no longer the design. Elements became mechanical on 2026-08-05: each applies a
// status to the opponent (ice cuts their AP, lightning adds a miss chance, fire is a damage
// over time, earth blunts their damage) and a matching ring discounts cards of that element.
// See MECHANICS.md. When that lands the type has to cross into internal/combat — cost stops
// being a property of the card and becomes a property of the pairing, the way Damage(str)
// already is — and ResolveRound, ResolutionOrder, the planners and every test in that package
// grow with it. Cheap to move now, less so later.
//
// Primary elements get cards; secondary ones (poison, force, hunger) do not, and where they
// appear is still open. Poison is in the starting deck only because it predates the split.
//
// This is what the reserved colour was reserved for: the glyphs were given one hueless
// palette on 2026-08-03 specifically so colour would still be free to mean something when
// elements arrived.
type element int

const (
	elementBasic element = iota

	// Primary. One card of each per concept, and a ring for each.
	elementFire
	elementIce
	elementLightning
	elementEarth

	// Secondary. No cards yet.
	elementPoison
)

// elementColors is the surface colour of a card, at full strength — drawCard scales it
// down for the resting, selected and unaffordable states.
//
// White is the *absence* of an element rather than a colour of its own. A plain card makes
// no claim, so the coloured ones are the ones that catch the eye, which is the whole point
// of painting them before the mechanic exists.
//
// An array rather than a map: nothing here iterates, but a map in this package is one
// refactor away from something that does, and Go randomises that order.
var elementColors = [...]color.RGBA{
	elementBasic:     {R: 235, G: 235, B: 235, A: 255},
	elementFire:      {R: 235, G: 120, B: 45, A: 255},
	elementIce:       {R: 80, G: 155, B: 230, A: 255},
	elementLightning: {R: 240, G: 205, B: 55, A: 255},
	elementEarth:     {R: 150, G: 105, B: 60, A: 255},
	elementPoison:    {R: 70, G: 140, B: 60, A: 255},
}

func (e element) color() color.RGBA { return elementColors[e] }

// elementNames is for tracing and for anything that has to say an element out loud. An
// array indexed by the constant rather than a map, for the same reason as elementColors.
var elementNames = [...]string{
	elementBasic:     "basic",
	elementFire:      "fire",
	elementIce:       "ice",
	elementLightning: "lightning",
	elementEarth:     "earth",
	elementPoison:    "poison",
}

func (e element) String() string {
	if int(e) >= len(elementNames) {
		return "?"
	}
	return elementNames[e]
}

// actionCard is one instance in the piles. A card used to *be* a combat.ActionKind, so two
// Strikes were the same value and indistinguishable; an element makes them differ, so the
// deck, hand and discard hold structs even while the element does nothing but paint.
type actionCard struct {
	action  combat.ActionKind
	element element
}

// The hand drawn from the deck each round. Quick is not in the deck — the deck is what the
// player owns, and which of the actions the rules define actually appear is a deck-building
// question rather than a rules one.
//
// Eight from five on 2026-08-04. Eight cards do not fit the screen side by side, which is
// what the overlap in handPitch is for — the hand is expected to go past eight sometimes.
// Eight against the 30-card deck below is roughly the ratio it was sized for.
const handSize = 8

// primaries is the element set that gets cards, in the order a concept's five are built.
// Secondary elements — poison, force, hunger — deliberately have none; where they appear is
// still open. See MECHANICS.md.
var primaries = []element{elementBasic, elementFire, elementIce, elementLightning, elementEarth}

// conceptDeck is one card of a concept in every primary element — the unit the deck is built
// from. Decided 2026-08-05: **a concept ships as five cards**, and that is the rule for
// adding concepts rather than a description of this particular deck. A new concept arrives as
// a set of five, never as a lone card.
func conceptDeck(action combat.ActionKind) []deckEntry {
	entries := make([]deckEntry, 0, len(primaries))
	for _, e := range primaries {
		entries = append(entries, deckEntry{actionCard{action, e}, 1})
	}
	return entries
}

// The starting deck: every concept, five cards each, grouped by the category it resolves in.
//
// **Six concepts, 30 cards**, split evenly 10 setup / 10 attack / 10 defend. That even split
// is a consequence of the concept list rather than a target — Guard moved from defend to
// setup on 2026-08-06 and Parry was dropped the same day, which is what turned the lopsided
// 1/2/4 the design started from into 2/2/2. The 5-setup/10-attack/20-defend shape recorded
// in MECHANICS.md is the thing that went; the two-thirds-defensive theory went with it.
//
// Quick is still homeless: it is an ActionKind with a cost and damage and no concept, so it
// has no five cards to arrive as.
var startingDeck = concat(
	// Setup.
	conceptDeck(combat.Gather),
	conceptDeck(combat.Guard),

	// Attack.
	conceptDeck(combat.Strike),
	conceptDeck(combat.Heavy),

	// Defend.
	conceptDeck(combat.Dodge),
	conceptDeck(combat.Riposte),
)

func concat(groups ...[]deckEntry) []deckEntry {
	var out []deckEntry
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// deckEntry is one line of a deck list: a card and how many copies of it.
type deckEntry struct {
	card  actionCard
	count int
}

// deckSeedName is which catalogued opening hand a launch deals. **Set it to whatever the
// screen is currently being worked on with** — the names and what each deals are in seeds.go,
// and `go run ./tools/seeds` prints them.
//
// Naming it rather than writing a bare number is the whole point: `deckSeed = 15` says nothing
// about why 15, and the next person to change it has no way to know they have just stopped
// dealing the hand a demo depended on.
const deckSeedName = "strike-flurry"

// deckSeed fixes the shuffle so every launch deals the same cards, which is what makes a
// layout problem reproducible while the screen is being built. It is a placeholder for the
// per-run seed described in TODO.md — when Session state exists this reads from there, and
// the rest of the deck code does not change.
//
// A var rather than a const so a build-tagged file can point it at a different catalogue
// entry in `init` — which is how the scripted demo picks its hand without the game growing a
// flag it would have to keep.
var deckSeed = seedFor(deckSeedName)

// discardSelected throws the selected cards away: they leave the hand for the discard
// pile, and the hand is dealt back up to size from the draw pile.
//
// Selection does double duty here — it is both "queue this for the round" and "this is
// the one I mean" — so throwing a card out costs the action points it was holding until
// the moment it leaves. That is a consequence of overloading selection rather than a
// designed cost; see TODO.md.
func (s *CombatScene) discardSelected() {
	if !s.planning() || s.discardsLeft <= 0 {
		return
	}
	s.discardsLeft--
	s.spendSelected()

	trace.Logf("input", "discard pressed, %d left this round, hand now %s",
		s.discardsLeft, handLabel(s.hand))
}

// spendSelected moves every selected card to the discard pile, deals the hand back up to
// size, and rebuilds the queue from what is left.
//
// **Selected is the only thing that leaves a hand.** Both ways a card can go — thrown away
// by Discard, or played by DUEL! — are the same movement over the same predicate, so they
// are one function rather than two that have to be kept in agreement.
func (s *CombatScene) spendSelected() {
	// Filtering in place over the hand's own array. Safe because kept never runs ahead of
	// the read cursor, and it keeps the surviving cards in the order the player left them.
	kept := s.hand[:0]
	for _, c := range s.hand {
		if c.selected {
			s.discard = append(s.discard, c.actionCard)
			continue
		}
		kept = append(kept, c)
	}
	s.hand = kept

	s.drawHand()
	s.syncQueue()
}

// toggleDeck shows or hides the deck overlay.
func (s *CombatScene) toggleDeck() {
	s.showDeck = !s.showDeck
	trace.Logf("input", "deck overlay %v (draw %d, discard %d)",
		s.showDeck, len(s.deck), len(s.discard))
}

// resetDeck builds the starting deck, shuffles it, empties the discard and deals an
// opening hand.
func (s *CombatScene) resetDeck() {
	s.deck = s.deck[:0]
	for _, e := range startingDeck {
		for i := 0; i < e.count; i++ {
			s.deck = append(s.deck, e.card)
		}
	}

	s.discard = s.discard[:0]
	s.hand = s.hand[:0]

	s.shuffleDeck()
	s.drawHand()
}

// shuffleDeck shuffles the draw pile using the scene's own source. Never rand.Shuffle,
// which draws from the global source and would make the deal unreproducible.
func (s *CombatScene) shuffleDeck() {
	s.rng.Shuffle(len(s.deck), func(i, j int) {
		s.deck[i], s.deck[j] = s.deck[j], s.deck[i]
	})
}

// drawHand fills the hand up to handSize, reshuffling the discard back into the draw pile
// when it runs dry. A hand can come up short only if every card the player owns is already
// in it, which cannot happen with a deck larger than the hand.
func (s *CombatScene) drawHand() {
	for len(s.hand) < handSize {
		if len(s.deck) == 0 {
			if len(s.discard) == 0 {
				return
			}
			s.deck = append(s.deck, s.discard...)
			s.discard = s.discard[:0]
			s.shuffleDeck()
		}

		last := len(s.deck) - 1
		s.hand = append(s.hand, paletteCard{actionCard: s.deck[last]})
		s.deck = s.deck[:last]
	}
}

// endRoundHand spends what was played and refills. **Only the cards that were actually
// played leave** — anything still sitting unselected in the hand stays exactly where it is,
// and the draw tops the hand back up to size.
//
// This reverses the decision recorded in TODO.md that the whole hand discarded every round,
// played or not, on the grounds that a hand kept back would let a plan be prepared once and
// repeated. What that actually produced was a hand you could not build on: cards you had
// deliberately held were taken away for having been held, so the only way to keep anything
// was to play it. Refilling only what was used is what makes a hand something you shape
// across rounds rather than a fresh deal you react to.
//
// It also gives Discard a real job. A card you never want now sits in your hand until you
// throw it out, so the discard button is how you clear it rather than a shortcut for
// something the round boundary was going to do anyway.
func (s *CombatScene) endRoundHand() {
	s.spendSelected()
	s.discardsLeft = discardsPerRound
}

// Deck overlay geometry. The panel is nearly the whole screen and stops above the button
// band, so the Deck button that closes it stays outside the panel as well as on top of it.
//
// The grid holds every card that is not in hand, which is at most deck minus handSize —
// 22 of the 30 today. Both piles are drawn into one fixed grid, so nothing below moves as
// cards shift from one pile to the other.
const (
	deckPanelLeftPct  = 4
	deckPanelRightPct = 96
	deckPanelTopPct   = 4
	// 92 rather than 86: at 86 the panel stopped short of the hand, so the tops of the cards
	// and the whole AP line sat below it, dimmed by the scrim but still visibly outside the
	// dialog. It still ends above the button band, which is what keeps the Deck button that
	// closes it outside the panel as well as drawn on top of it.
	deckPanelBottomPct = 92

	// Eight columns of 138 plus the gaps comes to 1160 inside the panel's 1177. Three rows
	// of 186-tall cards comes to 578 inside the ~713 the panel has below its heading, which
	// is 24 slots against the 22 cards that can be outside the hand.
	//
	// The third row is paid for by the card losing its initiative glyph: a two-glyph column
	// is 50 pixels shorter than a three-glyph one, and the deck doubled to 30 cards on the
	// same day. Without that the overflow line would fire on every look.
	deckGridColumns = 8
	deckGridGap     = 8
	deckGridRows    = 3

	// Offsets down from the panel's top edge.
	deckTitleTop  = 40
	deckCountsTop = 78
	deckLegendTop = 100
	deckGridTop   = 132
	deckHintUp    = 22 // hint's distance up from the panel's bottom edge
)

// drawDeckOverlay covers the screen with what is left to draw.
//
// The piles are drawn as the cards themselves at half size rather than as a table of
// counts. A count could say "six Strikes"; it could not say which of them are fire and
// which are plain, and with elements on the cards that is most of what the player wants to
// know. Twenty half-size cards fit on the panel at once, so the whole deck is one look.
//
// Both piles in one grid, with the discarded ones dimmed. The discard belongs here because
// those cards are coming back — a reshuffle folds the pile in, so "what is left" honestly
// means both piles — and merging them is what lets a card stay in place and simply dim
// when it is spent. See drawPileGrid.
func (s *CombatScene) drawDeckOverlay(gs *state.GlobalState, screen *ebiten.Image) {
	if !s.showDeck {
		return
	}

	// A scrim over everything, so the panel reads as covering the screen rather than
	// floating on it, and so the cards underneath look as inert as they now are.
	bounds := screen.Bounds()
	vector.DrawFilledRect(screen, 0, 0,
		float32(bounds.Dx()), float32(bounds.Dy()),
		color.RGBA{A: 190}, false)

	left, top := float32(gs.PctX(deckPanelLeftPct)), float32(gs.PctY(deckPanelTopPct))
	width := float32(gs.PctX(deckPanelRightPct)) - left
	height := float32(gs.PctY(deckPanelBottomPct)) - top

	vector.DrawFilledRect(screen, left, top, width, height,
		color.RGBA{R: 30, G: 30, B: 38, A: 255}, false)
	vector.StrokeRect(screen, left, top, width, height, 2, apBarColor, false)

	heading := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 28}
	small := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 14}

	title := &text.DrawOptions{}
	title.GeoM.Translate(float64(left+width/2), float64(top+deckTitleTop))
	title.PrimaryAlign = text.AlignCenter
	text.Draw(screen, "What is left", heading, title)

	// Hyphens, not em dashes. The kubasta font has no U+2014 and draws a missing-glyph box
	// for it — the middle dot is in the font, the dash is not.
	line := func(y float32, s string) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(left+width/2), float64(top+y))
		op.PrimaryAlign = text.AlignCenter
		text.Draw(screen, s, small, op)
	}
	line(deckCountsTop, fmt.Sprintf("draw %d  ·  discard %d  ·  %d in hand",
		len(s.deck), len(s.discard), len(s.hand)))
	line(deckLegendTop, "dimmed cards are in the discard - they return on the next reshuffle")

	s.drawPileGrid(gs, screen, left+width/2, top+deckGridTop)

	hint := &text.DrawOptions{}
	hint.GeoM.Translate(float64(left+width/2), float64(top+height-deckHintUp))
	hint.PrimaryAlign = text.AlignCenter
	text.Draw(screen, "Deck again to close", small, hint)
}

// pileEntry is one card in the overlay and whether it can be drawn right now.
type pileEntry struct {
	card      actionCard
	available bool
}

// drawPileGrid lays every card outside the hand into one grid, centred on centerX.
//
// One grid rather than a section per pile. Two sections of full-height cards do not fit the
// panel, and the single grid turns out to be the better picture anyway: sorting by kind and
// element rather than by which pile a card is in means **a card does not move when it is
// discarded, it only dims**. Watching your deck drain in place reads far better than
// watching cards jump between two lists.
//
// Sorted, never in pile order. The draw pile is shuffled, and drawing it in order would hand
// the player their next five cards and make the shuffle pointless. This is a picture of what
// the piles hold, not of their sequence.
func (s *CombatScene) drawPileGrid(gs *state.GlobalState, screen *ebiten.Image, centerX, top float32) {
	entries := make([]pileEntry, 0, len(s.deck)+len(s.discard))
	for _, c := range s.deck {
		entries = append(entries, pileEntry{c, true})
	}
	for _, c := range s.discard {
		entries = append(entries, pileEntry{c, false})
	}

	// Availability is the last key rather than the first, so identical cards sit together
	// and the drawable one of a pair comes first. That is what keeps a card's position
	// steady as it moves from the draw pile to the discard.
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.card.action != b.card.action {
			return a.card.action < b.card.action
		}
		if a.card.element != b.card.element {
			return a.card.element < b.card.element
		}
		return a.available && !b.available
	})

	pitchX := deckCardStyle.width + deckGridGap
	pitchY := deckCardStyle.height + deckGridGap
	rowLeft := int(centerX) - (deckGridColumns*pitchX-deckGridGap)/2
	slots := deckGridColumns * deckGridRows

	for i, e := range entries {
		if i >= slots {
			break
		}

		at := image.Pt(
			rowLeft+(i%deckGridColumns)*pitchX,
			int(top)+(i/deckGridColumns)*pitchY)

		// enabled carries "can be drawn", not "can be afforded". Never selected: this is an
		// inventory, not a choice, and dimming by the round's remaining AP would say
		// something about a budget that has nothing to do with a pile you cannot play from.
		drawCard(gs, screen, at, deckCardStyle, e.card, e.available, false, s.fighter.Str)
	}

	// The grid holds 24 and the deck puts at most 22 outside the hand, so this cannot fire
	// today — but deckbuilding will grow the deck, and a panel that silently drops the
	// overflow would be a picture that lies about what you own.
	if over := len(entries) - slots; over > 0 {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(centerX), float64(int(top)+deckGridRows*pitchY))
		op.PrimaryAlign = text.AlignCenter
		text.Draw(screen, fmt.Sprintf("+%d more not shown", over),
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 14}, op)
	}
}
