package screens

import (
	"fmt"
	"image"
	"math"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The draw pile made visible, and the cards traveling to and from it.
//
// Two things live here because they are one idea: a stack of card backs where the Deck
// button used to be, and the cards that fly out of the hand and back in from that stack.
// The pile is the thing the movement is *about*, so drawing one without the other would be
// a count changing next to an animation with no source.
//
// **Presentation only, and the constraint is the same one that governs trace, idle and the
// debug flags: it may never change an outcome.** The logical move has already happened by
// the time anything here runs — see spendSelected — so every flight is a ghost of a card
// that has already left. Nothing in this file is consulted by a rule, a predicate or a
// button's enabled state.
//
// It is animated on its own clock rather than through beatTicks. That is deliberate
// and recorded in TODO.md: playback pacing is one constant with one caller precisely so a
// new event kind cannot silently inherit a timing nobody chose, and card movement is not an
// event.

const (
	// deckCountSize is the count written beside the pile. **`60/60` is 41.2 pixels of kubasta at
	// 22**, which is what the strip it used to be right-aligned into was sized for; it is
	// right-aligned on the pile's own edge now — see deckCaptionRect.
	deckCountSize = 22

	deckStackDepth = 3 // backs drawn behind the front one, to read as a pile
	deckStackStep  = 3 // pixels each one is offset up and left

	// **The pile stands in the duelist card's column, at the size of every other card**
	// *(2026-09-04, owner's call)*. It was a half-size back in the bottom-right corner, which made
	// it the one card on the screen drawn small — and it stood in the corner the enemy card, the
	// sort column and DUEL! now line up on. Left-aligned under the duelist card and the floor
	// caption, it is the bottom of a column that reads as one thing: who you are, where you are,
	// what is left to draw.
	//
	// **The corner is the pile's again** *(2026-09-04)*. The frame's cog and ledger stood there
	// and have moved to the control column on the right — see screens.ControlColumnLeft — so what
	// the pile is measured against is the screen's own bottom edge, with its count under it.
	deckStackBottomInset = 10

	// deckCaptionGap is the air between the pile and the two things hung off it: the sack button
	// on the line above, the count on the line below. **Neither is beside it**, because what is to
	// its right at that height is the action-point bar, which spans the whole hand.
	deckCaptionGap = 6

	// outboundDriftUp is how far a discarded card rises as it leaves, and outboundSpin how
	// far it turns. A card tossed flat off the side of the table reads as a bug; a little
	// lift and rotation reads as a throw.
	outboundDriftUp = 40
	outboundSpin    = -0.42 // radians at the end of the flight
	outboundShrink  = 0.72  // scale it reaches as it goes

	//
	// **Cards are played at full size, and what they must not cover is the band the blow's
	// arithmetic is written across.** That has not changed through three arrangements of this
	// screen; see tableRowTop, which is the one place it is applied now.
	firingGap = 12
)

// Every journey a card makes, **as fractions of the one playback speed** *(2026-08-19)* — see
// `beat`. They reproduce the numbers they were tuned to at a speed of 25, so this changed nothing
// on the day it landed; what it changes is that they move with the round from here.
//
// **Cards move during planning as well as during playback**, which is the thing to know before
// turning the speed down: a discard leaving the hand and a card dealt back into it are on this
// clock too, and no round is playing while they happen. That is the trade for a single speed —
// the alternative is a second constant for movement, which is two numbers to keep in step on a
// screen where most movement *is* the round.
// How long a card takes to travel, and how far apart the drawn ones set off. Both in
// ticks at 60 TPS: about a third of a second each, overlapping.
func flightTicks() int      { return ui.Beat(4, 5) }
func flightStaggerPer() int { return ui.Beat(1, 6) }

// riseTicks() is how long a card takes to fly from the hand to its seat on the table.
//
// **The hold and fall beats went with the pile** *(2026-08-12)*. A card used to rise out of
// the hand, hold in the middle of the screen to be read, then drop into a corner — three
// beats, because the destination was not somewhere you could read a card. The table *is*
// the readable place, so there is one beat now: out of the hand and into its seat, where it
// stays for the rest of the round. What the hold used to say — "this is the one resolving"
// — is said by tableFireLift instead.
func riseTicks() int { return ui.Beat(3, 5) }

// cardFlight is one card in the air between the hand and the draw pile. Purely something to
// look at.
//
// **It stores an index and a row size, not a coordinate.** The hand re-lays out the instant
// a card leaves it, so a discarded card's origin no longer exists by the time it is drawn —
// slotAt takes the pair back and returns the rectangle that used to be there. It also means
// a flight survives the window being resized, which a cached pixel position would not.
type cardFlight struct {
	ui.Travel

	card combat.Card

	// outbound is a card leaving the hand for the discard; the other direction is a card
	// dealt from the draw pile into the slot it now occupies.
	outbound bool

	// index and count locate the slot: for an outbound card the one it left and the size of
	// the row it left, for an inbound card the one it is arriving at and the row it joins.
	index, count int

	// fromTable says the pair locates a seat on the table rather than a slot in the hand. A
	// card played this round spends the rest of it face up on the left of the table, so at the
	// end of the round it is thrown from there — not from the hand slot it left long before.
	//
	// split goes with it: the table row breaks between its attacks and its plans, so a seat
	// cannot be located without knowing where that break was. It is unread for a hand slot.
	fromTable bool
	split     int
}

// addFlight queues one. Kept as a method so the two call sites in spendSelected read as
// what they are rather than as slice manipulation.
func (s *CombatScene) addFlight(f cardFlight) {
	s.Theater.flights = append(s.Theater.flights, f)
}

// The hand's slides are cardSlide, and the mover is cardslide.go — shared with the essence screen's
// offer row since 2026-09-05. What stays here is what a *hand* slide is: the seat it uses, and the
// fact that a queued card is drawn standing proud of the row.

// addSlide queues one against the hand's row.
func (s *CombatScene) addSlide(sl ui.CardSlide) {
	s.Theater.slides = ui.AddCardSlide(s.Theater.slides, sl)
}

// slidingTo reports whether a card is currently sliding into hand slot i, so the row can leave
// that slot empty until it lands.
func (s *CombatScene) slidingTo(i int) bool { return ui.SlideInto(s.Theater.slides, i) }

// drawSlides draws the cards moving within the hand.
func (s *CombatScene) drawSlides(gs *state.GlobalState, screen *ebiten.Image) {
	ui.DrawCardSlides(gs, screen, s.Theater.slides,
		func(gs *state.GlobalState, i, count int) image.Point { return slotAt(gs, i, count) },
		func(sl ui.CardSlide) cards.Spec {
			return ui.CardSpec(sl.Card, ui.HeldBy(s.fighter.Duelist, sl.Card), true, sl.ToLift > 0)
		})
}

// inboundTo reports whether a card is currently flying into hand slot i, so the row can
// leave that slot empty until it lands.
//
// **The card is already in the hand** — spendSelected put it there before this file saw it,
// which is the whole reason the budget, the queue and every predicate stayed simple. What
// is suppressed is the *drawing* of a card that is on screen somewhere else, which is a
// view concern and lives here.
func (s *CombatScene) inboundTo(i int) bool {
	for _, f := range s.Theater.flights {
		if !f.outbound && f.index == i {
			return true
		}
	}
	return false
}

// deckStackRect is the front card of the pile: the one that is drawn on top and the one a
// click is tested against.
//
// **It is cards.Stack at three quarters, in the duelist's column** *(2026-09-04, owner's call)*.
// The style was a fifth — a back drawn small, because the strip it used to stand in was 86 pixels
// deep — and then briefly the full card, which in a column of its own read as the biggest thing on
// the screen. Three quarters is a card that is plainly a card and plainly not in play.
//
// Left-aligned with the duelist card above it, with the count on the line underneath.
func deckStackRect(gs *state.GlobalState) image.Rectangle {
	w, h := cards.Stack.Width, cards.Stack.Height

	bottom := gs.ScreenHeight - deckStackBottomInset - deckCountSize - deckCaptionGap
	left := gs.PctX(ui.DuelistCardLeftPct)

	return image.Rect(left, bottom-h, left+w, bottom)
}

// deckCountRect is the line under the pile, where the count is written left-aligned with it
// *(2026-09-04, owner's call)*.
func deckCountRect(gs *state.GlobalState) image.Rectangle {
	pile := deckStackRect(gs)
	top := pile.Max.Y + deckCaptionGap
	return image.Rect(pile.Min.X, top, pile.Max.X, top+deckCountSize)
}

// deckCaptionRect is the line above the pile: the sack button at its left end and the count at
// its right, both hung off the pile's own edges.
//
// **Above rather than beside**, which is the whole of what the move into the column cost. The pile
// used to have the width of the corner to spread into; it now fills its column, and what is to its
// right at that height is the action-point bar.
func deckCaptionRect(gs *state.GlobalState) image.Rectangle {
	// The pile's *bounds*, not its front card: the backs are drawn up and to the left, so the
	// front card's top edge is not the pile's.
	pile := deckStackBounds(gs)
	bottom := pile.Min.Y - deckCaptionGap
	return image.Rect(pile.Min.X, bottom-ui.PileSlotSize, pile.Max.X, bottom)
}

// deckStackBounds is the whole pile including the backs behind the front one, which is what
// the highlight is drawn around.
func deckStackBounds(gs *state.GlobalState) image.Rectangle {
	r := deckStackRect(gs)
	back := (deckStackDepth - 1) * deckStackStep
	return image.Rect(r.Min.X-back, r.Min.Y-back, r.Max.X, r.Max.Y)
}

// updateDeckStack handles the click that opens and closes the overlay.
//
// **It runs whether or not the overlay is up**, because it is the only thing that closes
// one. That is the same rule the Deck button lived under, and the reason Draw renders the
// stack a second time on top of the overlay.
func (s *CombatScene) updateDeckStack(gs *state.GlobalState) {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || !gs.CursorAllowed() {
		return
	}
	if image.Pt(gs.MouseX, gs.MouseY).In(deckStackBounds(gs)) {
		s.toggleDeck()
	}
}

// drawDeckStack draws the pile, the count beside it, and the ring round it when the overlay
// is open.
//
// The depth is fixed rather than proportional to how many cards are left. A pile that
// visibly thinned would be a nice touch and a lie: the discard is folded back in and
// reshuffled the moment the draw pile empties, so "how deep is the deck" is not a fact the
// player can act on and not one worth drawing. **The written count is the honest version of
// that number, and it now lives here** *(2026-08-11)* — it used to be `deck 45 · discard 7`
// on a line under the hand, which is gone.
//
// **It is a fraction over everything you own, not over the draw pile's starting size.** The
// denominator is `deckSize()` and never moves, so the numerator alone says how far through
// the deck the round is, and the rest — what is in the discard — is the subtraction the
// player can do at a glance: owned, less what is left to draw, less the eight in hand.
// A denominator that changed as cards were spent would make the fraction unreadable at
// exactly the moment it is worth reading.
func (s *CombatScene) drawDeckStack(gs *state.GlobalState, screen *ebiten.Image) {
	front := deckStackRect(gs)

	// Back to front, so the front card is the one on top and the one the click tests.
	for i := deckStackDepth - 1; i >= 0; i-- {
		off := i * deckStackStep
		s.drawCardBack(gs, screen, image.Pt(front.Min.X-off, front.Min.Y-off), cards.Stack)
	}

	// **Under the pile, left-aligned with it** *(2026-09-04, owner's call)*. It shares an edge with
	// the cards either way, which is what makes the pile read as the thing the number is about; what
	// the line underneath buys is that the count does not have to find room beside a card in a
	// column exactly one card wide.
	count := deckCountRect(gs)
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(count.Min.X), float64(count.Min.Y))
	op.ColorScale.ScaleWithColor(ui.GroundInk)
	text.Draw(screen, fmt.Sprintf("%d/%d", len(s.deck), s.deckSize()),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: deckCountSize}, op)
}

// drawCardBack blits a face-down card at rest.
//
// Separate from drawCard because that one builds its Spec from an combat.Card, and a back
// has no card behind it to build from — asking it for one would mean inventing a Bash
// nobody holds just to throw every field away. Separate from drawFlyingCard because a
// resting card must not be filtered.
func (s *CombatScene) drawCardBack(gs *state.GlobalState, screen *ebiten.Image, at image.Point, st cards.Style) {
	img := ui.CardImage(gs, s.backSpec(), st)
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(at.X), float64(at.Y))
	screen.DrawImage(img, op)
}

// drawFlights draws every card in the air, over the row and the panes and under the overlay.
func (s *CombatScene) drawFlights(gs *state.GlobalState, screen *ebiten.Image) {
	for _, f := range s.Theater.flights {
		if f.Waiting() {
			continue
		}
		if f.outbound {
			s.drawOutbound(gs, screen, f)
			continue
		}
		s.drawInbound(gs, screen, f)
	}
}

// drawOutbound throws a discarded card off to the left: it accelerates away, lifts, turns
// and shrinks. Face up the whole way — you know what you threw away, and a card that turned
// over on its way out would be hiding information the player already has.
func (s *CombatScene) drawOutbound(gs *state.GlobalState, screen *ebiten.Image, f cardFlight) {
	t := ui.EaseIn(f.Progress())

	from := slotAt(gs, f.index, f.count)
	if f.fromTable {
		from = playedSeatAt(gs, f.index, f.count, f.split)
	}

	ui.DrawFlyingCard(gs, screen,
		ui.CardSpec(f.card, ui.HeldBy(s.fighter.Duelist, f.card), true, false),
		cards.Hand, outboundGeoM(from, t))
}

// outboundGeoM is the throw: accelerating away to the left, lifting, turning and shrinking.
//
// **Split out so the animation gallery can drive it from its own origin** *(2026-09-15)*, which is
// the same reason drawDealtCard takes a point. A gallery reproducing the arithmetic would be a
// second copy of the gesture, and the one thing a review page must not do is show a gesture the
// game does not have.
func outboundGeoM(from image.Point, t float64) ebiten.GeoM {
	// Off the left edge by a whole card, so it is gone rather than clipped.
	toX := -cardWidth
	x := float64(from.X) + (float64(toX)-float64(from.X))*t
	y := float64(from.Y) - outboundDriftUp*t

	scale := 1 + (outboundShrink-1)*t

	var geo ebiten.GeoM
	geo.Translate(-cardWidth/2, -cardHeight/2)
	geo.Scale(scale, scale)
	geo.Rotate(outboundSpin * t)
	geo.Translate(x+cardWidth/2, y+cardHeight/2)
	return geo
}

// drawInbound deals a card from the stack into its slot, turning it face up on the way.
//
// **The journey is drawDealtCard's**, shared with the flip cascade's own deal; what is here is the
// face, which for an ordinary refill is simply the card.
func (s *CombatScene) drawInbound(gs *state.GlobalState, screen *ebiten.Image, f cardFlight) {
	to := slotAt(gs, f.index, f.count)
	face := ui.CardSpec(f.card, ui.HeldBy(s.fighter.Duelist, f.card), true, false)
	drawDealtCard(gs, screen, deckStackRect(gs).Min, to, f.Progress(), face, s.backSpec())
}

// drawDealtCard is the deal itself: a card out of the pile, growing to hand size, turning face up
// on the way, landing in `to`.
//
// **It takes a face and two points rather than a card**, which is the whole of what it costs to be
// shared. The
// hand's own deal draws the card it is dealing; the flip cascade's deal draws the card *as the pile
// holds it*, before any ring has touched it, and then morphs that face through the rings where it
// stands. One journey, two callers, no second copy of the flip-and-scale arithmetic.
//
// **The turn is a horizontal squash, not a rotation in depth.** Ebitengine's GeoM is a 2D affine
// transform, and affine transforms cannot do perspective — a card that genuinely foreshortened
// would need a Kage shader or per-vertex work. Scaling x from 1 to 0 and back while swapping the
// face for the back at the midpoint is the standard flat version of the same gesture, it costs one
// multiplication, and at this speed it reads correctly.
func drawDealtCard(gs *state.GlobalState, screen *ebiten.Image, from, to image.Point, raw float64, face, back cards.Spec) {
	t := ui.EaseOut(raw)

	// The stack is a small card and the hand is a full-size one, so the journey scales as
	// well as travels. Landing is at exactly 1, which is what keeps a resting card the same
	// blit it has always been — filtering a pixel-art glyph is fine in motion and not at rest.
	startScale := float64(cards.Stack.Width) / float64(cardWidth)
	scale := startScale + (1-startScale)*t

	x := float64(from.X) + (float64(to.X)-float64(from.X))*t
	y := float64(from.Y) - (float64(from.Y)-float64(to.Y))*t

	// The flip runs across the whole journey: out of the pile as a back, edge-on halfway,
	// face up as it lands. Unlinked from the easing on purpose, so the turn is even while
	// the travel decelerates.
	faceDown := raw < 0.5
	flip := math.Abs(1 - 2*raw)

	style, spec := cards.Hand, face
	if faceDown {
		spec = back
	}

	var geo ebiten.GeoM
	geo.Translate(-cardWidth/2, -cardHeight/2)
	geo.Scale(scale*flip, scale)
	geo.Translate(x+cardWidth/2, y+cardHeight/2)

	// A face-down card is drawn from the Stack style, which is the size the back is
	// authored at; the geometry above is written in Hand units, so it is scaled up to match
	// before the flight's own transform applies.
	if faceDown {
		var b ebiten.GeoM
		b.Scale(float64(cardWidth)/float64(cards.Stack.Width),
			float64(cardHeight)/float64(cards.Stack.Height))
		b.Concat(geo)
		geo = b
		style = cards.Stack
	}

	ui.DrawFlyingCard(gs, screen, spec, style, geo)
}

// resolvedCard is one of the player's cards for this round, on its way from the hand to its
// seat on the table.
//
// **The row is the round in the order it will happen.** Resolution regroups a queue into
// prepare, then attacks, then defenses, so the row is laid out in phase order without anything
// here knowing what a phase is — it is built by walking what the engine returned, and the order
// does the rest. That is the whole reason this reads as the mechanic rather than as an
// animation.
type resolvedCard struct {
	ui.Travel

	card combat.Card

	// Where it came from: the hand slot it occupied and the row it belonged to, so the flight
	// starts from the card's own place. Same reason cardFlight stores the pair — the hand is
	// still holding this card, but a later discard could re-lay the row out around it.
	handIndex, handCount int
}

// seatPlayedCards deals the player's whole queue to the table, in resolution order.
//
// **Called once, when the round starts, rather than a card at a time as each fires**
// *(2026-08-12)*. The two hands are laid out facing each other and the opponent's is known in
// full at that moment, so a player's row that assembled itself over the following seconds would
// be one hand against half of another. What playback drives now is which card is *lit*, not
// which cards exist — see firingSeat.
//
// **It asks combat.ResolutionOrder for the order rather than taking the queue as planned.** The
// order regroups by category, so the third card to resolve is not the third card selected, and
// a row in selection order would be a confident picture of a round that does not happen.
func (s *CombatScene) seatPlayedCards() {
	s.Theater.resolved = nil

	for _, slot := range combat.ResolutionOrder(s.fighterActions, s.enemyActions) {
		if slot.Side != combat.SideA {
			continue
		}

		// No card behind the queued action, which nothing should now be able to produce:
		// syncQueue derives the queue by walking the hand, and as of 2026-08-12 the scripted demo
		// selects through `toggle` like a player rather than writing `fighterActions` directly.
		//
		// **The shortcut it used to take is exactly what this guard was hiding.** A queue with no
		// cards behind it drew an empty half of the table while the Resolution feed narrated the
		// Duelist attacking — the guard did its job and the demo lied anyway. It is kept because
		// drawing nothing is still the right answer if something reaches here, and taken out of
		// the load-bearing position it was in.
		hand, ok := s.handIndexForQueue(slot.Index)
		if !ok {
			continue
		}

		s.Theater.resolved = append(s.Theater.resolved, resolvedCard{
			Travel:    ui.NewTravel(len(s.Theater.resolved)*flightStaggerPer(), riseTicks()),
			card:      s.hand[hand].Card,
			handIndex: hand,
			handCount: len(s.hand),
		})
	}
}

// noteResolved lights the card that has just fired, on whichever side played it.
//
// **It asks combat.ResolutionOrder which card this event belongs to rather than counting.**
// The order regroups a queue by category, so the third card to resolve is not the third card
// in the hand, and a screen keeping its own tally would light the wrong one the first time
// somebody queued a defense before an attack. currentSlot is already the authority on how
// far through that order playback has reached.
//
// The seat is *counted along the same walk the rows were laid out by* — seatPlayedCards for
// the player, enemyQueueOrder for the opponent — so a lit seat and a drawn card cannot
// disagree about which card is which. Both sides take their positions from the one ordering,
// which is why one function covers both rather than two that would have to be kept in step.
//
// **One side is lit at a time, and it is whichever fired last.** A turn is contiguous per side,
// so the lit cards walk the left row and then the right one; nothing has to clear the other
// side's seats because the event that lights one side is the event that unlights the other.
//
// **The whole attack hand goes up at once; everything else replaces** *(2026-08-15)*. A turn
// lands one blow, and the blow is the set — so the first attack announcement raises every attack
// card of that turn rather than each one climbing on its own beat. What the beats then say is how
// long the phase takes, not which card is acting, because no single card is: watching four cards
// rise one at a time reads as four attacks, which is the model this replaced.
//
// It is recomputed rather than accumulated, so every later announcement in the phase names the
// same set and the list cannot drift. noteHand then drops whichever of them earned nothing.
// A prepare or a defend is its own beat and takes the row on its own.
func (s *CombatScene) noteResolved(e combat.Event) {
	if e.Kind != combat.KindAction {
		return
	}

	order := combat.ResolutionOrder(s.fighterActions, s.enemyActions)
	i, ok := s.currentSlot()
	if !ok || i >= len(order) {
		return
	}

	side := order[i].Side
	seat := 0
	for _, slot := range order[:i] {
		if slot.Side == side {
			seat++
		}
	}

	mine, theirs := &s.Theater.firingSeats, &s.Theater.enemyFiringSeats
	if side == combat.SideB {
		mine, theirs = &s.Theater.enemyFiringSeats, &s.Theater.firingSeats
	}

	// **Nothing on a duelist's turn lifts on a card's own announcement.** A lift says "this card is
	// acting now", and a duelist's turn has exactly one moment entitled to say that: the hand's
	// announcement. See noteHand, which raises the cards that made the rung on the `KindHand` beat,
	// and advancePlayback, which puts them back down before the tally. **The turn reads shields,
	// then the announcement, then the sum**, and a second gesture ahead of the announcement reads
	// as whichever card it lifted having gone first.
	//
	// The defend phase says what it has to say as one bundle of pips out of their own cards — see
	// noteShieldRaise, and MECHANICS.md §Shields — so a defense has nothing left to add by moving.
	//
	// **A solo attacker is the exception and that is the whole point of it** *(2026-08-17)*.
	// Raising a set says "these cards are one blow", which is exactly what an enemy's turn is not:
	// three cards swing three times, in order, no hand is ever named, and the card that is up is
	// the card that is hitting. It has no other beat to be lifted on.
	switch {
	case s.soloAttacker(side) && combat.Plain(e.Action).Category() == combat.CategoryAttack:
		*mine = []int{seat}
	default:
		*mine = nil
	}
	*theirs = nil
}

// lit reports whether a seat is one of the ones currently raised.
func lit(seats []int, seat int) bool {
	for _, s := range seats {
		if s == seat {
			return true
		}
	}
	return false
}

// noteHand raises the cards the hand was made of, on the beat it is announced.
//
// The engine names them — see Event.RungCards — so this is a lookup, not a search. Nothing
// here knows what a Flurry is or how many cards one takes, which is what stops the table
// disagreeing with the hand that actually fired.
//
// **The rung, not the blow.** `HandCards` is every attack the turn played, so on a Pair formed by
// two shields beside one attack it holds the attack too — and raising it would stand up a card that
// made no part of the hand being announced. `RungCards` is the narrower list and is what the lift
// means.
//
// **The cards need not be adjacent.** A counted hand like Two Pair is two cards, a card that
// earned nothing, and two more, so this takes the seats it is given rather than a span between
// the first and the last.
//
// **Raising is the whole of what says which cards earned it**: the shout names the hand and the row
// shows which cards made it, with no ring or bracket drawn round them. **And it ends with the
// announcement** — advancePlayback puts them down when the tally starts, because a row held up
// through the arithmetic is the announcement still being made while it is read out.
func (s *CombatScene) noteHand(e combat.Event) {
	if e.Kind != combat.KindHand {
		return
	}

	// **The No Hand raises nothing.** That rung is announced like any other — it carries a
	// multiplier, a stone raises it, a relic can name it — but it is the turn that built *no* hand,
	// so there is no set of cards that made it. Its `Blow.Rung` is one card picked by damage rather
	// than by counting, and raising that says the card did something while every other attack in
	// the turn lands beside it unraised. See builtARung.
	var seats []int
	if builtARung(e) {
		seats = make([]int, 0, e.RungCardCount)
		seats = append(seats, e.RungCards[:e.RungCardCount]...)
	}

	if e.Side == combat.SideB {
		s.Theater.enemyFiringSeats = seats
		return
	}

	s.Theater.firingSeats = seats
}

// handIndexForQueue maps a position in the player's queue to the hand slot holding it.
//
// syncQueue builds fighterActions by walking the hand and taking the selected cards in
// order, so the nth queued action is the nth selected card. This is the inverse of that one
// loop, and it is written next to the thing that needs it rather than cached, because the
// hand can be re-ordered by a drag between rounds.
func (s *CombatScene) handIndexForQueue(n int) (int, bool) {
	seen := 0
	for i, c := range s.hand {
		if !c.selected {
			continue
		}
		if seen == n {
			return i, true
		}
		seen++
	}
	return 0, false
}

// resolvedInHand reports whether the card in hand slot i has already fired, so the row can
// stop drawing it — it is on screen somewhere between the hand and the pile.
//
// The card has *not* left the hand. It leaves at the end of the round, with everything else
// that was played, which is what keeps the Resolution pane able to narrate from
// fighterActions while the round is still running. This hides a drawing, exactly like
// inboundTo.
func (s *CombatScene) resolvedInHand(i int) bool {
	for _, r := range s.Theater.resolved {
		if r.handIndex == i {
			return true
		}
	}
	return false
}

// playedSeatOf finds the table seat of the card that came from hand slot i, so a card leaving
// at the end of the round sets off from where it actually is.
func (s *CombatScene) playedSeatOf(handIndex int) (int, bool) {
	for i, r := range s.Theater.resolved {
		if r.handIndex == handIndex {
			return i, true
		}
	}
	return 0, false
}

// at is where a played card is drawn right now: waiting its turn to set off, flying to its
// seat, sitting in it, or lifted out of it because it is the card currently resolving.
func (r resolvedCard) at(gs *state.GlobalState, seat, total, split int, firing bool) image.Point {
	from := slotAt(gs, r.handIndex, r.handCount)
	to := playedSeatAt(gs, seat, total, split)

	switch {
	case r.Waiting():
		return from
	case !r.Done():
		return ui.LerpPoint(from, to, ui.EaseOut(r.Progress()))
	}

	// **The lift is applied after the card has landed, never during the flight.** A card still
	// arriving is already the most moving thing on screen, and lifting it as well would make
	// the two beats — dealt to the table, then played — impossible to tell apart. The
	// opponent's cards do not fly, so their row applies the same lift unconditionally.
	return lift(to, firing)
}

// drawPlayedCards draws the player's side of the table and whatever is still flying into it.
//
// Left to right in resolution order, so a later card overlaps the one before it and the row
// reads in the order the round happens — and so a card still arriving is drawn over the ones
// already seated rather than sliding underneath them.
func (s *CombatScene) drawPlayedCards(gs *state.GlobalState, screen *ebiten.Image) {
	split := s.playedSplit()
	for i, r := range s.Theater.resolved {
		at := r.at(gs, i, len(s.Theater.resolved), split, lit(s.Theater.firingSeats, i))

		// **The card rattles as its own figure is written into the sum** *(owner's call,
		// 2026-08-26)*. Sideways, where the lift above is vertical: the lift says this card built
		// the hand and stays for the whole blow, and the shake says this card is paying *now*. Two
		// vocabularies on one card, which is why the shake could not also be a jump.
		at.X += s.playedCardShake(i)

		ui.DrawCard(gs, screen, at, cards.Hand, r.card, ui.HeldBy(s.fighter.Duelist, r.card), true, false)
	}
}
