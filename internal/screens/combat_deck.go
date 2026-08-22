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

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/scenario"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"image/color"
)

// actionCard is one instance in the piles: a concept and the element it is made of.
//
// **It is an alias for `combat.Card` as of 2026-08-12, not a struct of its own.** The screen
// used to own this *and* an unexported `element` type, on the honest grounds that neither meant
// anything to the rules — a card's colour painted a border and `ResolveRound` never saw it.
// Elements are mechanical now, so the rules own the type and the piles hold exactly what the
// engine resolves.
//
// That is what an alias is for here: the hand, the queue and the round are one type, so a card
// cannot be converted wrongly on the way between them because it is never converted at all. The
// name stays because the screen is full of it, and `actionCard` still says what a pile holds
// better than `Card` does next to `paletteCard`, `pileEntry` and `cardFlight`.
//
// **What went with the old type: `elementColors` and `element.color()`.** They were the surface
// colours from when a card *was* a coloured rectangle, and nothing had called either since the
// border took over the element on 2026-08-09 and `internal/cards` took over the drawing. The
// live colour table is `cards.BorderOf`.
type actionCard = combat.Card

// The hand drawn from the deck each round.
//
// Eight from five on 2026-08-04. Eight cards do not fit the screen side by side, which is
// what the overlap in handPitch is for — the hand is expected to go past eight sometimes.
//
// **Eight was sized against a 30-card deck and the deck is now 48.** That is 17% of the deck
// in hand where it used to be 27%, so consistency fell without this number moving.
//
// **It is the base rather than the size as of 2026-08-15**, because a Plan card widens one hand
// by two. See handTarget, which is what the refill actually reads; this is what it widens from.
// Eight is left alone deliberately: `discardsPerRound` and now Plan are the levers meant to
// answer draw variance, and moving all three at once would leave no way to tell which did the
// work. A brand growing hand size is the recorded permanent version.
const handSize = 8

// deckSize is how many cards the player owns right now, counting all three piles.
//
// **It counts rather than reading a constant** *(2026-08-17)*. It used to total the authored
// starting list, which was the same number every time and stopped being true the moment a worm
// could remove a card. The piles are conserved — nothing is created or destroyed mid-fight — so
// their sum is the run's deck size for as long as the fight lasts.
func (s *CombatScene) deckSize() int {
	return len(s.deck) + len(s.discard) + len(s.hand)
}

// deckSeedName pins every launch to one catalogued opening hand. **Empty means unpinned**,
// which is the default: the shuffles are rolled from the run seed instead and each fight deals
// a fresh deck. Set it to a name while working on something that needs a particular hand — the
// names and what each deals are in seeds.go, and `go run ./tools/seeds` prints them.
//
// Naming it rather than writing a bare number is the whole point: `deckSeed = 15` says nothing
// about why 15, and the next person to change it has no way to know they have just stopped
// dealing the hand a demo depended on.
const deckSeedName = ""

// deckSeed is the pinned shuffle, or **zero for "roll one from the run seed"** — the same
// toggle shape as `fixedRunSeed` in main.go, and the counterpart of it: that one pins which
// enemies a run meets, this one pins which cards it draws. Pinning makes a layout problem
// reproducible; unpinned is what an actual run looks like.
//
// When it is non-zero the *opponent's* shuffle is pinned too, to `seeds.EnemyDeckPin` — see
// shuffleSeeds. Half a pinned duel is not reproducible, and the scripted demo sets only this
// one.
//
// A var rather than a const so a build-tagged file can point it at a catalogue entry in
// `init` — which is how the scripted demo picks its hand without the game growing a flag it
// would have to keep.
var deckSeed = pinnedDeckSeed()

// pinnedDeckSeed reads deckSeedName, treating the empty name as no pin. seedFor panics on a
// name it does not know, which is right for a typo and wrong for "no name at all", so the
// empty case is answered here rather than by adding a not-found path to the catalogue.
func pinnedDeckSeed() int64 {
	if deckSeedName == "" {
		return 0
	}
	return seedFor(deckSeedName)
}

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
	//
	// keptFrom records where each survivor was standing, because a card that stays in the hand
	// still *moves*: the cards around it have gone and the row closes up under it. That is a
	// slide, and it needs the slot the card is leaving as much as a discard does.
	kept := s.hand[:0]
	var keptFrom []int
	for i, c := range s.hand {
		if c.selected {
			s.discard = append(s.discard, c.actionCard)

			// A card that was played is sitting in its seat on the table, not in its old hand
			// slot, so that is where it has to set off from. Sending it out of a slot it
			// visibly left ten seconds ago would make it jump back across the screen to be
			// thrown. The count goes with it: a seat's x is a function of how many cards the
			// row holds, and the row is cleared three lines below this.
			flight := cardFlight{
				travel:   newTravel(0, flightTicks),
				card:     c.actionCard,
				outbound: true,
				index:    i, count: leaving,
			}
			if p, ok := s.playedSeatOf(i); ok {
				flight.index, flight.count, flight.fromTable = p, len(s.theatre.resolved), true
				flight.split = s.playedSplit()
			}
			s.addFlight(flight)
			continue
		}
		kept = append(kept, c)
		keptFrom = append(keptFrom, i)
	}
	s.hand = kept

	// The round's history goes with the cards it was made of. Cleared here rather than at the
	// start of the next round because this is the moment those cards actually leave, and a
	// pile outliving them would be a picture of a round that is over.
	s.theatre.resolved = nil

	// Everything appended past this point was dealt, which is what makes the drawn cards
	// identifiable without drawHand having to report them.
	dealt := len(s.hand)
	s.drawHand()

	// **The sort runs before anything is animated, not after.** A dealt card lands in the slot
	// the sort gives it rather than on the right-hand end, so it flies to where it will
	// actually be sitting — and `inboundTo`, which blanks a slot that is still filling, is
	// looking at that same index.
	//
	// The permutation is what survives the rearrangement. `order[to]` is where the card now at
	// `to` came from: past `dealt` it came out of the draw pile and flies in, below it the card
	// was already in the hand and slides across the row from wherever it was standing.
	order := s.sortHand()

	staggered := 0
	for to, from := range order {
		if from >= dealt {
			s.addFlight(cardFlight{
				travel: newTravel(staggered*flightStaggerPer, flightTicks),
				card:   s.hand[to].actionCard,
				index:  to, count: len(s.hand),
			})
			staggered++
			continue
		}

		// A survivor. It has moved if its slot changed or if the row it is standing in did —
		// eight cards centred is not the same place as six centred, so a card that kept its
		// index still has ground to cover.
		was := keptFrom[from]
		if was == to && leaving == len(s.hand) {
			continue
		}
		s.addSlide(handSlide{
			travel:    newTravel(0, slideTicks),
			card:      s.hand[to].actionCard,
			selected:  s.hand[to].selected,
			fromIndex: was, fromCount: leaving,
			toIndex: to, toCount: len(s.hand),
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

// resetDeck fills the draw pile from the run, shuffles it, empties the discard and deals an
// opening hand.
//
// **The deck comes from the run now, not from the authored list** *(2026-08-17)*. That is the
// whole of what makes a worm stick: the piles are rebuilt on every `Init`, and `Init` is how the
// next fight starts, so a deck edit held anywhere on this scene would be thrown away between
// rooms.
//
// **A nil run means the starting deck**, which is not a fallback for the game — `main` always
// builds one — but for the callers that deal a hand without a run around it: `OpeningHand` in
// seeds.go, `tools/seeds`, and the flight tests. A named seed is a fact about the *starting*
// deck, so those must not read a run even when one exists.
func (s *CombatScene) resetDeck(run *session.Session) {
	s.deck = s.deck[:0]
	if run != nil {
		// **FightDeck rather than Deck**: this is the `deck-built` moment, so a flip ring recolours
		// what is dealt without touching what the run owns. See session.FightDeck.
		s.deck = append(s.deck, run.FightDeck()...)
	} else {
		s.deck = append(s.deck, session.StartingDeck()...)
	}

	s.discard = s.discard[:0]
	s.hand = s.hand[:0]

	s.shuffleDeck()
	s.drawHand()

	// **A scenario's hand is dealt over the shuffle, not through it** — the draw pile is left
	// exactly as it was, so the second hand of the fight is a normal one and the fixture is only
	// the opening. Compiled out of every normal build; see internal/scenario.
	if scenario.Active() {
		s.plugHand(scenario.Hand())
	}

	// An opening hand arrives sorted, exactly as a refilled one does. The default is cost, so
	// this is true of a scene that has never had a sort button pressed on it.
	s.sortHand()
}

// plugHand replaces the hand with an authored one. **A debug seat and nothing else** — it is
// called from one place, behind `scenario.Active()`, and it is a no-op in every build that has not
// asked for the tag.
//
// **The replaced cards go nowhere.** They are not discarded and not put back: the draw pile is
// untouched, so the round after this one refills from a deck that never knew. A fixture is meant
// to be one hand, not a rewritten deck.
func (s *CombatScene) plugHand(cards []combat.Card) {
	if len(cards) == 0 {
		return
	}

	s.hand = s.hand[:0]
	for _, c := range cards {
		s.hand = append(s.hand, paletteCard{actionCard: c})
	}
}

// shuffleDeck shuffles the draw pile using the scene's own source. Never rand.Shuffle,
// which draws from the global source and would make the deal unreproducible.
func (s *CombatScene) shuffleDeck() {
	s.rng.Shuffle(len(s.deck), func(i, j int) {
		s.deck[i], s.deck[j] = s.deck[j], s.deck[i]
	})
}

// handTarget is how many cards this round's refill draws to: the usual size, plus whatever a
// Plan banked in the round before.
//
// **This is where a Plan actually draws** *(2026-08-15)*. `internal/combat` has no deck, so a
// Plan records `BonusDraw` on the duelist and the size of the next hand is the whole of what it
// bought. It has to be the refill target rather than an immediate two cards: the hand refills to
// a fixed size at the round boundary anyway, so cards handed over mid-round would be two fewer
// drawn at the boundary and the card would do nothing at all.
//
// The engine assigns `BonusDraw` rather than adding to it, so it lasts exactly one round — a
// hand of ten, and then a hand of eight again.
//
// **A scene with no fighter draws the usual eight.** `OpeningHand` and the flight tests build a
// bare CombatScene to deal a hand without a duel around it, which is exactly the case a Plan
// cannot have applied in.
func (s *CombatScene) handTarget() int {
	if s.fighter == nil {
		return handSize
	}
	return handSize + s.fighter.BonusDraw
}

// drawHand fills the hand up to handTarget, reshuffling the discard back into the draw pile
// when it runs dry. A hand can come up short only if every card the player owns is already
// in it, which cannot happen with a deck larger than the hand.
func (s *CombatScene) drawHand() {
	for len(s.hand) < s.handTarget() {
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
// **Do not go back to discarding the whole hand each round.** The argument for it is that a
// hand kept back would let a plan be prepared once and repeated — but the *queue* already
// empties every round, so no plan repeats by default, and clearing the hand as well produces
// a hand you cannot build on: cards you deliberately held are taken away for having been
// held, so the only way to keep anything is to play it. Refilling only what was used is what
// makes a hand something you shape across rounds rather than a fresh deal you react to.
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
// The panel holds **every card you own**, in five rows, and nothing in it moves as cards shift
// between piles — a played card dims where it stands rather than leaving.
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
	deckRowGap = 6

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

	// deckMaxPerRow caps a row so it cannot run off the panel. Twelve is what the longest row
	// holds — the plans, at 3 concepts x 4 copies — and what deckStackPitch is sized against.
	// The four colour rows hold nine each.
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
	//
	// **Everything under the title came up by eight on 2026-08-15**, for a grid that briefly had
	// six rows. It is back to five — four colours and the plans — and the clearance is simply
	// spare now. TestDeckPitchMatchesTheCard is what holds the arithmetic either way.
	deckTitleTop  = 40
	deckCountsTop = 70
	deckLegendTop = 92
	deckGridTop   = 112
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
	// The deck total is stated as well as the three piles, because with 48 cards the piles no
	// longer add up to a number anybody holds in their head — and because it is the one place
	// the size of cards.json becomes visible while playing.
	line(deckCountsTop, fmt.Sprintf("draw %d  ·  discard %d  ·  %d in hand  ·  %d owned",
		len(s.deck), len(s.discard), len(s.hand), s.deckSize()))
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
		if ra, rb := formRank(a.card.Form()), formRank(b.card.Form()); ra != rb {
			return ra < rb
		}
		if ca, cb := a.card.Cost(), b.card.Cost(); ca != cb {
			return ca < cb
		}
		if a.card.Concept != b.card.Concept {
			return a.card.Concept < b.card.Concept
		}
		if a.card.Element != b.card.Element {
			return a.card.Element < b.card.Element
		}
		return a.available && !b.available
	})
}

// formRank is the order forms run in along a deck row: stab, slash, crush, then the plans.
//
// **It sorts on form rather than category** *(2026-08-15)*. Category has two values now, so
// sorting by it would put nine cards in one undifferentiated block — and the thing a player is
// looking for in this panel is how much of a form they still hold, because that is what a pair
// is counted on.
//
// A function rather than the enum's own order, because that order is grouped for the *rules* —
// it is what an expanded hand ID is derived from. Reading it here would tie how the deck panel
// is arranged to how hands are numbered, so that changing one silently changed the other.
func formRank(f combat.Form) int {
	switch f {
	case combat.FormStab:
		return 0
	case combat.FormSlash:
		return 1
	case combat.FormCrush:
		return 2
	case combat.FormPlan:
		return 3
	default:
		return 4
	}
}

// pileEntry is one card in the overlay and whether it can still be drawn.
type pileEntry struct {
	card      actionCard
	available bool
}

// deckRowElements is the colours the overlay gives a row to, in the fixed order internal/cards
// declares them.
//
// **Basic is not among them as of 2026-08-15**, because no attack card is basic any more — every
// attack ships in one of the four colours, and the only basic cards in the deck are the plans,
// which have their own row. A basic row would draw an empty gutter label over nothing at all.
//
// A function rather than a package-level slice so nothing can append to it, and derived from
// `cards.Elements()` rather than written out, so a fifth colour added to the drawing package
// arrives here without an edit.
func deckRowElements() []cards.Element {
	out := make([]cards.Element, 0, len(cards.Elements()))
	for _, e := range cards.Elements() {
		if e == cards.Basic {
			continue
		}
		out = append(out, e)
	}
	return out
}

// deckRowCount is how many rows the overlay draws: one per colour, then the plans.
//
// **The plans get a row of their own rather than sitting with the basics** *(2026-08-15)*. Every
// plan card is basic and no attack card is, so the alternative was a row holding nothing but the
// plans under a label reading "basic" — which names the colour rather than the thing, on the one
// row where the colour is the least interesting fact about the cards in it.
var deckRowCount = len(deckRowElements()) + 1

// deckPlanRow is the index of that row, and where a card goes when it is a plan. It is
// deliberately not a `cards.Element` — a plan is basic and saying otherwise would put a colour
// on a card that has none.
var deckPlanRow = deckRowCount - 1

// deckRowFor is which row a card belongs to: its colour, or the plan row.
//
// **A basic attack has nowhere to go and lands in the first row**, which is the deck list being
// wrong rather than this being lenient — `data/duelist_cards.json` ships no basic attacks and
// TestEveryCardLandsInExactlyOneDeckRow is what would catch one arriving.
func deckRowFor(c actionCard) int {
	if c.Category() == combat.CategoryPlan {
		return deckPlanRow
	}
	for i, e := range deckRowElements() {
		if e == artFor(c.Element) {
			return i
		}
	}
	return 0
}

// deckRowLabel is what the gutter says beside a row, and the colour it says it in.
func deckRowLabel(row int) (string, color.RGBA) {
	if row == deckPlanRow {
		// Neutral, because the plan row is the one row that is not a colour. It borrows the
		// basic border rather than inventing a fifth hue for a row that means "no element".
		return "plan", cards.BorderOf(cards.Basic)
	}
	e := deckRowElements()[row]
	return e.String(), cards.BorderOf(e)
}

// drawPileGrid lays **every card you own** into rows by element, plus a row of plans,
// centred on centerX.
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
// pileGrid is where every card in the panel goes, and where each row is named.
//
// **One layout, read by the drawing and by the cursor** *(2026-08-21)*. It was inline in
// drawPileGrid until the tooltip needed to know which card is under the pointer, and a second set
// of arithmetic saying where a card is drawn is exactly the bug the one-rectangle rule prevents
// everywhere else on this screen.
func (s *CombatScene) pileGrid(centerX, top float32) pileGridLayout {
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

	// One row per element in the fixed order internal/cards declares, then the plans. A slice
	// indexed by row rather than a map: Go randomises map iteration, and a panel whose rows
	// swapped places between looks would be unreadable.
	rows := make([][]pileEntry, deckRowCount)
	for _, e := range entries {
		rows[deckRowFor(e.card)] = append(rows[deckRowFor(e.card)], e)
	}

	out := pileGridLayout{rowPitch: cards.Mini.Height + deckRowGap}
	pitch := deckStackPitch

	// Widest row sets the left edge, so the rows share one origin and the columns line up
	// down the panel rather than each row centring on its own count.
	widest := 0
	for _, group := range rows {
		n := min(len(group), deckMaxPerRow)
		if w := (n-1)*pitch + cards.Mini.Width; w > widest {
			widest = w
		}
	}
	cardsLeft := int(centerX) - (deckRowLabelWidth+widest)/2 + deckRowLabelWidth

	shown := 0
	for i, group := range rows {
		rowTop := int(top) + i*out.rowPitch
		out.labels = append(out.labels, pileRowLabel{
			row: i,
			at:  image.Pt(cardsLeft-12, rowTop+cards.Mini.Height/2),
		})

		for j, e := range group {
			if j >= deckMaxPerRow {
				break
			}
			at := image.Pt(cardsLeft+j*pitch, rowTop)
			out.slots = append(out.slots, pileSlot{
				pileEntry: e,
				at:        image.Rect(at.X, at.Y, at.X+cards.Mini.Width, at.Y+cards.Mini.Height),
			})
		}
		shown += min(len(group), deckMaxPerRow)
	}
	out.hidden = len(entries) - shown
	return out
}

// pileGridLayout is the panel's geometry: where each card sits, where each row is named, how tall a
// row is, and how many cards did not fit.
type pileGridLayout struct {
	slots    []pileSlot
	labels   []pileRowLabel
	rowPitch int
	hidden   int
}

// pileSlot is one card and the rectangle it occupies. **The same rectangle is drawn in and
// hit-tested against**, which is the rule every row of cards in this game follows.
type pileSlot struct {
	pileEntry
	at image.Rectangle
}

// pileRowLabel is one row's name in the gutter, at the point it is drawn from.
type pileRowLabel struct {
	row int
	at  image.Point
}

func (s *CombatScene) drawPileGrid(gs *state.GlobalState, screen *ebiten.Image, centerX, top float32) {
	grid := s.pileGrid(centerX, top)

	for _, l := range grid.labels {
		// A mini card says its own concept but nothing about which row it is in, so this is what
		// names the row.
		name, ink := deckRowLabel(l.row)
		labelOp := &text.DrawOptions{}
		labelOp.GeoM.Translate(float64(l.at.X), float64(l.at.Y))
		labelOp.PrimaryAlign = text.AlignEnd
		labelOp.SecondaryAlign = text.AlignCenter
		labelOp.ColorScale.ScaleWithColor(ink)
		text.Draw(screen, name,
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}, labelOp)
	}

	// **Left to right, so each card is covered on its *right* edge by the next one.** This was
	// backwards and the screenshot showed it: drawing right to left puts card 0 on top of card 1,
	// and card 1's left edge is exactly where its glyph and dashes are — so every row rendered as
	// one complete card followed by eleven blank slivers.
	for _, slot := range grid.slots {
		// available carries "can be drawn", not "can be afforded". Never selected: this is an
		// inventory, not a choice, and dimming by the round's remaining AP would say something
		// about a budget that has nothing to do with a pile you cannot play from.
		drawCard(gs, screen, slot.at.Min, cards.Mini, slot.card,
			heldBy(s.fighter.Duelist, slot.card), slot.available, false)
	}

	if grid.hidden > 0 {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(centerX), float64(int(top)+deckRowCount*grid.rowPitch))
		op.PrimaryAlign = text.AlignCenter
		text.Draw(screen, fmt.Sprintf("+%d more not shown", grid.hidden),
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 14}, op)
	}
}
