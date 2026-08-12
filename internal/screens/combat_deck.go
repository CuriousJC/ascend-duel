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

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
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

// The hand drawn from the deck each round.
//
// Eight from five on 2026-08-04. Eight cards do not fit the screen side by side, which is
// what the overlap in handPitch is for — the hand is expected to go past eight sometimes.
//
// **Eight was sized against a 30-card deck and the deck is now 60.** That is 13% of the deck
// in hand where it used to be 27%, so consistency halved on 2026-08-08 without this number
// moving. It is left at eight deliberately, for now: Sift and `discardsPerRound` are the two
// levers meant to answer draw variance, and moving all three at once would leave no way to
// tell which one did the work. A brand growing hand size is the recorded permanent version.
const handSize = 8

// elementsByName resolves the element names in cards.json. Not a map from `element` to string —
// elementNames above already is that, and two tables for one relation is one refactor away from
// disagreeing. This walks it instead, so a new element needs one edit rather than two.
func elementByName(name string) (element, bool) {
	for i, n := range elementNames {
		if n == name {
			return element(i), true
		}
	}
	return elementBasic, false
}

// startingDeck is the deck the player opens a run with: **twelve concepts x five elements = 60
// cards**, built from `data/cards.json` rather than written out here.
//
// It became data on 2026-08-08, at the same time the concept grid was filled. The shape it
// replaced was a `concat` of `conceptDeck` calls — fine for six concepts, and a list nobody
// could count at a glance for twelve. What the JSON buys is that the deck's *size* is now a
// consequence of a file the designer can read and edit, rather than of a Go expression.
//
// **The rules did not move with it.** Cost, category and damage stay in `internal/combat`;
// `cards.json` names concepts and elements and declares a cost tier that `buildStartingDeck`
// checks against the engine. See data/card_data.go for why the tier is checked rather than
// trusted.
var startingDeck = buildStartingDeck()

// buildStartingDeck turns the data records into deck entries, in file order — which is grid
// order, which is the order the deck overlay sorts into anyway.
//
// **It panics on a bad record, and that is the right severity.** An unknown concept name, a
// cost tier that disagrees with the rules, or an element the screen does not know are all
// things that would otherwise produce a deck quietly missing five cards. A missing concept is
// a balance change nobody made on purpose, and a game that starts anyway is a game that hides
// it. This runs at package init, so it fails on launch rather than mid-duel.
func buildStartingDeck() []deckEntry {
	// Not named `cards`: that is the drawing package, imported above, and shadowing a
	// package name inside the one function that builds the deck is a trap.
	records := data.LoadDuelistCards()

	problems := data.CheckCostTiers("duelist_cards.json", records,
		func(concept string) (int, bool) {
			a, ok := combat.ParseAction(concept)
			if !ok {
				return 0, false
			}
			return a.Cost(), true
		},
		func(concept string) (string, bool) {
			a, ok := combat.ParseAction(concept)
			if !ok {
				return "", false
			}
			return a.Category().String(), true
		},
	)
	if len(problems) > 0 {
		msg := "duelist_cards.json disagrees with the rules:"
		for _, p := range problems {
			msg += "\n  " + p.Error()
		}
		panic(msg)
	}

	var out []deckEntry
	for _, c := range records {
		action, ok := combat.ParseAction(c.Concept)
		if !ok {
			panic("duelist_cards.json: unknown concept " + c.Concept)
		}
		for _, name := range c.Elements {
			e, ok := elementByName(name)
			if !ok {
				panic("duelist_cards.json: " + c.Concept + " names unknown element " + name)
			}
			out = append(out, deckEntry{actionCard{action, e}, c.Copies})
		}
	}
	return out
}

// deckSize is how many cards the starting deck actually deals, counting copies. Used by the
// trace dump and the deck overlay's heading, so neither has to recount.
func deckSize() int {
	n := 0
	for _, e := range startingDeck {
		n += e.count
	}
	return n
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
// **The cards move now and the animation catches up.** Every flight this raises is a ghost
// of a card that has already gone — the hand, the piles and the queue are correct the moment
// this returns, so planning(), the action-point budget and the row's own layout never have
// to know about a card that is neither in the hand nor out of it. Holding a card in place
// until its animation finished would have put that question into all three.
func (s *CombatScene) spendSelected() {
	// The row the discarded cards are leaving. Captured before the filter, because a flight
	// starts from the slot a card had in the hand that existed when it was thrown — and that
	// hand stops existing on the next line. See slotAt.
	leaving := len(s.hand)

	// Filtering in place over the hand's own array. Safe because kept never runs ahead of
	// the read cursor, and it keeps the surviving cards in the order the player left them.
	kept := s.hand[:0]
	for i, c := range s.hand {
		if c.selected {
			s.discard = append(s.discard, c.actionCard)

			// A card that was played is sitting in its seat on the table, not in its old hand
			// slot, so that is where it has to set off from. Sending it out of a slot it
			// visibly left ten seconds ago would make it jump back across the screen to be
			// thrown. The count goes with it: a seat's x is a function of how many cards the
			// row holds, and the row is cleared three lines below this.
			flight := cardFlight{card: c.actionCard, outbound: true, index: i, count: leaving}
			if p, ok := s.playedSeatOf(i); ok {
				flight.index, flight.count, flight.fromTable = p, len(s.resolved), true
			}
			s.addFlight(flight)
			continue
		}
		kept = append(kept, c)
	}
	s.hand = kept

	// The round's history goes with the cards it was made of. Cleared here rather than at the
	// start of the next round because this is the moment those cards actually leave, and a
	// pile outliving them would be a picture of a round that is over.
	s.resolved = nil

	// Everything appended past this point was dealt, which is what makes the drawn cards
	// identifiable without drawHand having to report them.
	dealt := len(s.hand)
	s.drawHand()
	for i := dealt; i < len(s.hand); i++ {
		s.addFlight(cardFlight{
			card:  s.hand[i].actionCard,
			index: i, count: len(s.hand),
			delay: (i - dealt) * flightStaggerPer,
		})
	}

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
// **Sift is applied here and nowhere else.** Each Sift played sends two more cards away at
// random before the refill, so a round that played one Sift replaces seven of eight cards
// rather than five: the four the player chose, the Sift itself, and two the game chose.
func (s *CombatScene) endRoundHand() {
	sifted := s.siftsResolved() * siftExtraDiscards

	s.spendSelected()
	s.siftHand(sifted)
	s.discardsLeft = discardsPerRound
}

// siftExtraDiscards is how many extra cards one Sift sends away.
//
// **The extras go at random, and that is the whole difference between Sift and the Discard
// button.** Discard is steering — you choose what leaves, four times a round. Sift is
// throughput: more of the deck flows past you and you do not pick which cards pay for it, so it
// can take a card you were holding on purpose. That is what it costs beyond its 2 AP, and it is
// why the two are not the same mechanic at different prices.
//
// It is also the reason Sift's effect lives on the screen rather than in `internal/combat`: it
// needs the deck and the hand, and both stay out of the rules package so the rules keep no
// shuffle. See MECHANICS.md.
const siftExtraDiscards = 2

// siftsResolved counts the fighter's Sifts that actually happened, read off the resolved event
// log rather than off the queue.
//
// **The log and the queue disagree, and the log is right.** A stagger takes actions off the
// front of a turn, and under phases the front of a turn is its prepares — so a Sift is among
// the first things a stagger eats. Counting the queue would sift for a card the engine deleted
// before it resolved, which is the same class of bug as a combo scoring off cards that never
// landed. `KindAction` is only emitted for actions that ran; a staggered one emits
// `KindStaggered` instead.
func (s *CombatScene) siftsResolved() int {
	n := 0
	for _, e := range s.log {
		if e.Kind == combat.KindAction && e.Side == combat.SideA && e.Action == combat.Sift {
			n++
		}
	}
	return n
}

// siftHand sends n cards away at random and deals back up. Drawn from the scene's own rng, the
// same source as the shuffle — never the package-level functions, which would make a run
// unreproducible. See the determinism rules in CLAUDE.md.
//
// It takes from the *remaining* hand, after the played cards have already gone, so a Sift never
// discards something the player just spent.
func (s *CombatScene) siftHand(n int) {
	if n <= 0 {
		return
	}

	for i := 0; i < n && len(s.hand) > 0; i++ {
		at := s.rng.Intn(len(s.hand))
		s.discard = append(s.discard, s.hand[at].actionCard)
		s.hand = append(s.hand[:at], s.hand[at+1:]...)
	}

	trace.Logf("deck", "sift sent %d card(s) away at random, hand now %s", n, handLabel(s.hand))

	s.drawHand()
	s.syncQueue()
}

// Deck overlay geometry. The panel is nearly the whole screen and stops above the button
// band, so the Deck button that closes it stays outside the panel as well as on top of it.
//
// The panel holds **every card you own**, in five rows of twelve, and nothing in it moves
// as cards shift between piles — a played card dims where it stands rather than leaving.
const (
	deckPanelLeftPct  = 4
	deckPanelRightPct = 96
	deckPanelTopPct   = 4
	// 92 rather than 86: at 86 the panel stopped short of the hand, so the tops of the cards
	// and the whole AP line sat below it, dimmed by the scrim but still visibly outside the
	// dialog. It still ends above the button band, which is what keeps the Deck button that
	// closes it outside the panel as well as drawn on top of it.
	deckPanelBottomPct = 92

	// **The grid became five overlapping rows, one per element, on 2026-08-09.**
	//
	// It was an 8x3 grid of half-size cards, which held 24 of the up-to-52 cards that can
	// sit outside the hand — so "+N more not shown" fired on every single look. That line
	// was written when the deck was 30 and could not fire, deliberately, so that growing
	// the deck would produce a visible shortfall rather than a panel that quietly lied.
	// It did its job and then kept firing.
	//
	// A half-size card (cards.Mini) overlapped to show only its left half needs 45 pixels
	// of width instead of 146. Twelve concepts per element is 585 pixels a row, five rows
	// is 684 tall, and **the whole deck now fits** — the overflow line is still there and
	// can no longer fire.
	//
	// Half rather than a third: a third-size card was 59 pixels wide and could carry
	// neither a glyph nor text, so a row was a line of coloured slivers. At 90 the
	// 22-pixel category glyph fits, and the visible left 45 pixels are exactly the glyph
	// and the cost dashes under it. A row now says what phase each card resolves in and
	// what it costs. What it still cannot say is which *concept* each card is.
	//
	// **Five rows of 132 is a tight fit and the gap is what absorbs it.** The panel gives
	// about 691 pixels between the legend and the closing hint; five rows plus four
	// 8-pixel gaps is 692, so deckGridTop moved up to 120 to buy the clearance back. A
	// sixth element would not fit and would need a different arrangement, not a smaller
	// gap.
	deckRowGap = 8

	// deckStackPitch is how far apart the cards in a row sit. **It is a constant sized for
	// a full row, never derived from how many cards are actually in one** — the overlay's
	// governing idea is that a card does not move when it is discarded, it only dims, and
	// a pitch that adapted to the count would shuffle the whole row on every draw.
	//
	// Twelve concepts at 75 is 906 pixels plus the 104-pixel label gutter, inside the
	// panel's 1177 with margin. At Mini's full 81 the cards would not overlap at all, so the
	// six pixels of overlap are what buys the row its slack — the stacking is load-bearing
	// arithmetic, not a look.
	//
	// It was Mini.Width/2, which showed 45 pixels of each card and left no room for a
	// name. Widening it to width-less-six is what let the name go on at all.
	//
	// **75 since 2026-08-11**, down from 84 with the card itself — the overlap is what is
	// held constant, not the pitch, so a pitch left behind at 84 would have opened gaps
	// between cards in an 81-pixel row. TestDeckRowFitsThePanel is what catches that.
	deckStackPitch = 75

	// deckMaxPerRow caps a row so it cannot run off the panel. Twelve is what the deck
	// holds per element — 12 concepts x 1 copy — and what deckStackPitch is sized against.
	//
	// **The cap is what gives the overflow line something to report.** Without it a row
	// simply drew every card it had and ran off the edge, and the "+N more not shown"
	// message below could never fire because nothing was ever not shown. A thirteenth
	// concept should produce a visible, honest shortfall rather than a card halfway off
	// the panel.
	deckMaxPerRow = 12

	// deckRowLabelWidth is the gutter the element name sits in, to the left of each row.
	// The cards no longer carry any text, so without this a row would be an anonymous
	// line of coloured slivers.
	deckRowLabelWidth = 104

	// Offsets down from the panel's top edge.
	deckTitleTop  = 40
	deckCountsTop = 78
	deckLegendTop = 100
	deckGridTop   = 120
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
	text.Draw(screen, "Your deck", heading, title)

	// Hyphens, not em dashes. The kubasta font has no U+2014 and draws a missing-glyph box
	// for it — the middle dot is in the font, the dash is not.
	line := func(y float32, s string) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(left+width/2), float64(top+y))
		op.PrimaryAlign = text.AlignCenter
		text.Draw(screen, s, small, op)
	}
	// The deck total is stated as well as the three piles, because with 60 cards the piles no
	// longer add up to a number anybody holds in their head — and because it is the one place
	// the size of cards.json becomes visible while playing.
	line(deckCountsTop, fmt.Sprintf("draw %d  ·  discard %d  ·  %d in hand  ·  %d owned",
		len(s.deck), len(s.discard), len(s.hand), deckSize()))
	line(deckLegendTop, "dimmed cards are in your hand or the discard - the rest are still to draw")

	s.drawPileGrid(gs, screen, left+width/2, top+deckGridTop)

	hint := &text.DrawOptions{}
	hint.GeoM.Translate(float64(left+width/2), float64(top+height-deckHintUp))
	hint.PrimaryAlign = text.AlignCenter
	text.Draw(screen, "Deck again to close", small, hint)
}

// **Attacks, then defends, then prepares; within each, cheapest first.** The rows are
// already one element each, so this is the order along a row — and it is what turns a
// row from a list into a shape: the same three runs in the same places in every row,
// so a gap is a card you have spent rather than a card you never had.
//
// Availability is the last key rather than the first, so a card's position does not
// depend on which pile it is in. That is the whole governing idea of the panel — a
// card does not move when it is played, it only dims — and sorting by pile would undo
// it. It only breaks ties between genuinely identical cards, which are interchangeable.
//
// Pulled out of drawPileGrid so it can be tested: the drawing needs a window and this does
// not, and the order along a row is a user-visible decision worth pinning.
func sortPileEntries(entries []pileEntry) {
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if ra, rb := categoryRank(a.card.action.Category()), categoryRank(b.card.action.Category()); ra != rb {
			return ra < rb
		}
		if ca, cb := a.card.action.Cost(), b.card.action.Cost(); ca != cb {
			return ca < cb
		}
		if a.card.action != b.card.action {
			return a.card.action < b.card.action
		}
		if a.card.element != b.card.element {
			return a.card.element < b.card.element
		}
		return a.available && !b.available
	})
}

// categoryRank is the order categories run in along a deck row: attack, defend, prepare.
//
// A function rather than the enum's own order, because that order is the *resolution*
// order — which phase a card acts in — and it is a rule. Reading it here would tie how the
// deck panel is arranged to how a round plays out, so that changing one silently changed
// the other.
func categoryRank(c combat.Category) int {
	switch c {
	case combat.CategoryAttack:
		return 0
	case combat.CategoryDefend:
		return 1
	case combat.CategoryPrepare:
		return 2
	default:
		return 3
	}
}

// pileEntry is one card in the overlay and whether it can still be drawn.
type pileEntry struct {
	card      actionCard
	available bool
}

// drawPileGrid lays **every card you own** into rows by element, centred on centerX.
//
// It used to show only what was outside the hand, under the heading "What is left". That
// made the panel change *shape* as a round went on: eight cards vanished at the start of
// every round and came back at the end of it, so the rows shortened and lengthened and
// nothing sat still. The point of sorting rather than showing pile order was always that
// **a card does not move when it is played, it only dims** — and leaving the hand out
// broke exactly that.
//
// So all sixty are here, always, and the dimming carries the state instead: full strength
// means still in the draw pile, washed out means in your hand or already discarded. The
// rows are now a fixed twelve long, which is what the layout was sized for anyway.
//
// Sorted, never in pile order. The draw pile is shuffled, and drawing it in order would
// hand the player their next few cards and make the shuffle pointless. This is a picture
// of what you own, not of the sequence it will arrive in.
func (s *CombatScene) drawPileGrid(gs *state.GlobalState, screen *ebiten.Image, centerX, top float32) {
	entries := make([]pileEntry, 0, len(s.deck)+len(s.discard)+len(s.hand))
	for _, c := range s.deck {
		entries = append(entries, pileEntry{c, true})
	}
	for _, c := range s.discard {
		entries = append(entries, pileEntry{c, false})
	}
	// The hand dims the same way the discard does. They are different piles but the same
	// fact from this panel's point of view — this card is not one you can still draw.
	for _, c := range s.hand {
		entries = append(entries, pileEntry{c.actionCard, false})
	}

	sortPileEntries(entries)

	// One row per element, in the fixed order internal/cards declares. Never a map range:
	// Go randomises that, and a panel whose rows swapped places between looks would be
	// unreadable.
	byElement := map[cards.Element][]pileEntry{}
	for _, e := range entries {
		art := e.card.element.art()
		byElement[art] = append(byElement[art], e)
	}

	pitch := deckStackPitch
	rowPitch := cards.Mini.Height + deckRowGap

	// Widest row sets the left edge, so the rows share one origin and the columns line up
	// down the panel rather than each row centring on its own count.
	widest := 0
	for _, group := range byElement {
		n := min(len(group), deckMaxPerRow)
		if w := (n-1)*pitch + cards.Mini.Width; w > widest {
			widest = w
		}
	}
	left := int(centerX) - (deckRowLabelWidth+widest)/2
	cardsLeft := left + deckRowLabelWidth

	shown := 0
	for i, el := range cards.Elements() {
		rowTop := int(top) + i*rowPitch

		// The element's name in the gutter, vertically centred on the row. The cards in
		// this row carry no text at all, so this is the only thing naming them.
		labelOp := &text.DrawOptions{}
		labelOp.GeoM.Translate(float64(cardsLeft-12), float64(rowTop+cards.Mini.Height/2))
		labelOp.PrimaryAlign = text.AlignEnd
		labelOp.SecondaryAlign = text.AlignCenter
		labelOp.ColorScale.ScaleWithColor(cards.BorderOf(el))
		text.Draw(screen, el.String(),
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}, labelOp)

		// **Left to right, so each card is covered on its *right* edge by the next one.**
		// This was backwards and the screenshot showed it: drawing right to left puts
		// card 0 on top of card 1, and card 1's left edge is exactly where its glyph and
		// dashes are — so every row rendered as one complete card followed by eleven
		// blank slivers.
		group := byElement[el]
		for j, e := range group {
			if j >= deckMaxPerRow {
				break
			}
			at := image.Pt(cardsLeft+j*pitch, rowTop)
			// enabled carries "can be drawn", not "can be afforded". Never selected: this
			// is an inventory, not a choice, and dimming by the round's remaining AP would
			// say something about a budget that has nothing to do with a pile you cannot
			// play from.
			drawCard(gs, screen, at, cards.Mini, e.card, e.available, false, s.fighter.Str)
		}
		shown += min(len(group), deckMaxPerRow)
	}

	if over := len(entries) - shown; over > 0 {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(centerX), float64(int(top)+len(cards.Elements())*rowPitch))
		op.PrimaryAlign = text.AlignCenter
		text.Draw(screen, fmt.Sprintf("+%d more not shown", over),
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 14}, op)
	}
}
