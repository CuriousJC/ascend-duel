package screens

import (
	"fmt"
	"image"
	"math"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The draw pile made visible, and the cards travelling to and from it.
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
	// Where the stack sits: the far end of the button strip, away from Discard and DUEL!
	// because it changes nothing about the round.
	//
	// **Measured in from the right edge, not as a percentage** *(2026-08-11)*, which makes it
	// the horizontal counterpart of the top being measured down from the AP bar. It was 88%,
	// which put 132 pixels of empty screen to its right for no reason anybody chose — a
	// percentage says where a thing sits relative to the whole, and what this actually wants
	// to say is "against the edge, with a margin".
	//
	// The margin is to the card's own edge, and TestDeckStackClearsTheAPBarAndTheScreen holds
	// it against the screen.
	//
	// **It was 10 until 2026-08-11**, when the pile took over stating its own count. The
	// margin is now the strip that count is written in, and it is *derived* from the three
	// things in that strip rather than being a number that happens to fit: the text's margin
	// from the screen edge, the width reserved for the text, and the gap between the pile and
	// it. Nudging the pile is therefore a change to whichever of the three is actually wrong.
	deckStackRightMargin = deckCountRightMargin + deckCountReserve + deckStackToCountGap

	// Where the count sits in that strip: right-aligned this far in from the screen's edge,
	// with its baseline on the pile's bottom edge.
	//
	// **The reserve is measured, not guessed** — `60/60` is 41.2 pixels of kubasta at 22 — and
	// it is a fixed width for the same reason apFigureReserve is: the pile's placement must
	// not shift as the numerator loses a digit.
	deckCountRightMargin = 14
	deckCountReserve     = 42
	deckCountSize        = 22

	// How far the pile stands off its own count. Halved from the 41 pixels the first version
	// left, which read as two things near each other rather than one labelled thing.
	deckStackToCountGap = 20
	deckStackDepth      = 3 // backs drawn behind the front one, to read as a pile
	deckStackStep       = 3 // pixels each one is offset up and left

	// deckStackBottomInset hangs the pile off the **bottom of the screen** *(2026-08-12)*,
	// where it used to be measured down from the action-point bar.
	//
	// **The anchor moved because what constrains it moved.** The bar was the constraint while
	// the strip below it was 86 pixels and a 54-pixel pile only just fitted; the bar has since
	// come down with the hand and there is slack under it, so measuring from above left the
	// pile floating in the middle of that slack with a band of empty screen beneath it. What
	// the pile actually wants to say now is "against the bottom edge, with a margin" — the
	// vertical counterpart of deckStackRightMargin, and the same correction that constant made.
	//
	// **Ten, because that is where the mute button's bottom edge is** — see
	// internal/game/chrome.go, which is chrome and cannot be read from here, so the number is
	// shared by being the same number rather than by being derived. The discard badge's bottom
	// lands four pixels lower at 954, since it hangs off a button strip placed as a percentage;
	// four pixels reads as one line and chasing it exactly would mean taking the strip off
	// percentages for no other reason.
	//
	// TestDeckStackClearsTheAPBarAndTheScreen fails rather than letting the pile run off the
	// bottom.
	deckStackBottomInset = 10

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
var (
	// How long a card takes to travel, and how far apart the drawn ones set off. Both in
	// ticks at 60 TPS: about a third of a second each, overlapping.
	flightTicks      = beat(4, 5)
	flightStaggerPer = beat(1, 6)

	// riseTicks is how long a card takes to fly from the hand to its seat on the table.
	//
	// **The hold and fall beats went with the pile** *(2026-08-12)*. A card used to rise out of
	// the hand, hold in the middle of the screen to be read, then drop into a corner — three
	// beats, because the destination was not somewhere you could read a card. The table *is*
	// the readable place, so there is one beat now: out of the hand and into its seat, where it
	// stays for the rest of the round. What the hold used to say — "this is the one resolving"
	// — is said by tableFireLift instead.
	riseTicks = beat(3, 5)

	// sort, or the row closing up after cards were spent. Shorter than the other three
	// journeys because it is the shortest one: a few inches across the row rather than a
	// trip across the screen, and a long ease over that distance reads as sluggish rather
	// than as deliberate.
	slideTicks = beat(1, 2)
)

// cardFlight is one card in the air between the hand and the draw pile. Purely something to
// look at.
//
// **It stores an index and a row size, not a coordinate.** The hand re-lays out the instant
// a card leaves it, so a discarded card's origin no longer exists by the time it is drawn —
// slotAt takes the pair back and returns the rectangle that used to be there. It also means
// a flight survives the window being resized, which a cached pixel position would not.
type cardFlight struct {
	travel

	card actionCard

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
	s.theatre.flights = append(s.theatre.flights, f)
}

// handSlide is a card moving from one slot in the hand to another: what a sort looks like,
// and what a card left standing does when the cards around it are spent.
//
// **It is the fourth mover and it stores no coordinates either**, for the reason travel's own
// comment gives — both ends are `slotAt` calls made fresh every frame. What it does store that
// the others do not is a row size at *each* end, because the two can differ: at the end of a
// round a surviving card slides out of the row it was in and into the smaller or refilled one
// that replaced it, and a single count could only describe one of those.
//
// **It never travels between the hand and anywhere else.** A card going to the pile, to the
// table or back is one of the three journeys above; this is the one that begins and ends in
// the row.
type handSlide struct {
	travel

	card actionCard

	// selected lifts both ends by selectedNudge, so a queued card slides along the raised
	// line it is already on rather than dropping into the row and jumping back out of it.
	selected bool

	fromIndex, fromCount int
	toIndex, toCount     int
}

// addSlide queues one, and drops any slide already heading for the same slot.
//
// **Pressing a second sort button before the first has landed is the case this exists for.**
// Two slides converging on one slot would draw the card twice and leave `slidingTo` suppressing
// the row's own copy until the later of them finished. The new arrangement is the true one, so
// the older claim on that slot loses.
func (s *CombatScene) addSlide(sl handSlide) {
	kept := s.theatre.slides[:0]
	for _, old := range s.theatre.slides {
		if old.toIndex != sl.toIndex {
			kept = append(kept, old)
		}
	}
	s.theatre.slides = append(kept, sl)
}

// slidingTo reports whether a card is currently sliding into hand slot i, so the row can leave
// that slot empty until it lands. Exactly like inboundTo, and hiding a drawing rather than a
// card for exactly the same reason: the hand is already in its new order.
func (s *CombatScene) slidingTo(i int) bool {
	for _, sl := range s.theatre.slides {
		if sl.toIndex == i {
			return true
		}
	}
	return false
}

// drawSlides draws the cards moving within the row.
//
// **Flat, at full size, with no flip, spin or scale** — and that is the whole gesture. The other
// three journeys cross the screen and dramatise it; this one is a card shuffling a few inches
// sideways into place, and anything more would make re-sorting a hand look like an event.
func (s *CombatScene) drawSlides(gs *state.GlobalState, screen *ebiten.Image) {
	for _, sl := range s.theatre.slides {
		if sl.waiting() {
			continue
		}
		t := easeOut(sl.progress())

		from, to := slotAt(gs, sl.fromIndex, sl.fromCount), slotAt(gs, sl.toIndex, sl.toCount)
		if sl.selected {
			from.Y -= selectedNudge
			to.Y -= selectedNudge
		}

		var geo ebiten.GeoM
		geo.Translate(
			float64(from.X)+(float64(to.X)-float64(from.X))*t,
			float64(from.Y)+(float64(to.Y)-float64(from.Y))*t,
		)
		drawFlyingCard(gs, screen, cardSpec(sl.card, heldBy(s.fighter.Duelist, sl.card), true, sl.selected), cards.Hand, geo)
	}
}

// inboundTo reports whether a card is currently flying into hand slot i, so the row can
// leave that slot empty until it lands.
//
// **The card is already in the hand** — spendSelected put it there before this file saw it,
// which is the whole reason the budget, the queue and every predicate stayed simple. What
// is suppressed is the *drawing* of a card that is on screen somewhere else, which is a
// view concern and lives here.
func (s *CombatScene) inboundTo(i int) bool {
	for _, f := range s.theatre.flights {
		if !f.outbound && f.index == i {
			return true
		}
	}
	return false
}

// deckStackRect is the front card of the pile: the one that is drawn on top and the one a
// click is tested against.
//
// **Sized against the strip below the action-point bar, which is what forced cards.Stack to
// be smaller than Mini.** When the pile arrived that strip was 86 pixels, so a 132-pixel
// half-size card did not fit here however much one would read better. See the Stack style for
// why a *back* survives that where a face would not.
//
// **Hung off the bottom edge since 2026-08-12**, where it used to be measured down from the
// bar. Both corners are now margins from the screen's own edges, which is what the pile
// actually wants to say; see deckStackBottomInset for why the anchor moved and why the
// constraint it has to satisfy is checked by a test rather than by arithmetic here.
func deckStackRect(gs *state.GlobalState) image.Rectangle {
	w, h := cards.Stack.Width, cards.Stack.Height

	top := gs.ScreenHeight - deckStackBottomInset - h
	left := gs.PctX(100) - deckStackRightMargin - w

	return image.Rect(left, top, left+w, top+h)
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

	// Sitting on the pile's bottom edge rather than centred on its face: the count and the
	// cards then share a line, and the pile reads as the thing the number is about.
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(gs.PctX(100)-deckCountRightMargin), float64(front.Max.Y))
	op.PrimaryAlign = text.AlignEnd
	op.SecondaryAlign = text.AlignEnd
	op.ColorScale.ScaleWithColor(groundInk)
	text.Draw(screen, fmt.Sprintf("%d/%d", len(s.deck), s.deckSize()),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: deckCountSize}, op)
}

// drawCardBack blits a face-down card at rest.
//
// Separate from drawCard because that one builds its Spec from an actionCard, and a back
// has no card behind it to build from — asking it for one would mean inventing a Strike
// nobody holds just to throw every field away. Separate from drawFlyingCard because a
// resting card must not be filtered.
func (s *CombatScene) drawCardBack(gs *state.GlobalState, screen *ebiten.Image, at image.Point, st cards.Style) {
	img := cardImage(gs, s.backSpec(), st)
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(at.X), float64(at.Y))
	screen.DrawImage(img, op)
}

// drawFlights draws every card in the air, over the row and the panes and under the overlay.
func (s *CombatScene) drawFlights(gs *state.GlobalState, screen *ebiten.Image) {
	for _, f := range s.theatre.flights {
		if f.waiting() {
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
	t := easeIn(f.progress())

	from := slotAt(gs, f.index, f.count)
	if f.fromTable {
		from = playedSeatAt(gs, f.index, f.count, f.split)
	}
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

	drawFlyingCard(gs, screen, cardSpec(f.card, heldBy(s.fighter.Duelist, f.card), true, false), cards.Hand, geo)
}

// drawInbound deals a card from the stack into its slot, turning it face up on the way.
//
// **The turn is a horizontal squash, not a rotation in depth.** Ebitengine's GeoM is a 2D
// affine transform, and affine transforms cannot do perspective — a card that genuinely
// foreshortened would need a Kage shader or per-vertex work. Scaling x from 1 to 0 and back
// while swapping the face for the back at the midpoint is the standard flat version of the
// same gesture, it costs one multiplication, and at this speed it reads correctly.
func (s *CombatScene) drawInbound(gs *state.GlobalState, screen *ebiten.Image, f cardFlight) {
	t := easeOut(f.progress())

	stack := deckStackRect(gs)
	to := slotAt(gs, f.index, f.count)

	// The stack is a small card and the hand is a full-size one, so the journey scales as
	// well as travels. Landing is at exactly 1, which is what keeps a resting card the same
	// blit it has always been — filtering a pixel-art glyph is fine in motion and not at rest.
	startScale := float64(cards.Stack.Width) / float64(cardWidth)
	scale := startScale + (1-startScale)*t

	x := float64(stack.Min.X) + (float64(to.X)-float64(stack.Min.X))*t
	y := float64(stack.Min.Y) + (float64(to.Y)-float64(stack.Min.Y))*t

	// The flip runs across the whole journey: out of the pile as a back, edge-on halfway,
	// face up as it lands. Unlinked from the easing on purpose, so the turn is even while
	// the travel decelerates.
	raw := f.progress()
	faceDown := raw < 0.5
	flip := math.Abs(1 - 2*raw)

	style, spec := cards.Hand, cardSpec(f.card, heldBy(s.fighter.Duelist, f.card), true, false)
	if faceDown {
		spec = s.backSpec()
	}

	var geo ebiten.GeoM
	geo.Translate(-cardWidth/2, -cardHeight/2)
	geo.Scale(scale*flip, scale)
	geo.Translate(x+cardWidth/2, y+cardHeight/2)

	// A face-down card is drawn from the Stack style, which is the size the back is
	// authored at; the geometry above is written in Hand units, so it is scaled up to match
	// before the flight's own transform applies.
	if faceDown {
		var back ebiten.GeoM
		back.Scale(float64(cardWidth)/float64(cards.Stack.Width),
			float64(cardHeight)/float64(cards.Stack.Height))
		back.Concat(geo)
		geo = back
		style = cards.Stack
	}

	drawFlyingCard(gs, screen, spec, style, geo)
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
	travel

	card actionCard

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
	s.theatre.resolved = nil

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

		s.theatre.resolved = append(s.theatre.resolved, resolvedCard{
			travel:    newTravel(len(s.theatre.resolved)*flightStaggerPer, riseTicks),
			card:      s.hand[hand].actionCard,
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

	mine, theirs := &s.theatre.firingSeats, &s.theatre.enemyFiringSeats
	if side == combat.SideB {
		mine, theirs = &s.theatre.enemyFiringSeats, &s.theatre.firingSeats
	}

	// **A solo attacker lifts one card at a time, and that is the whole point of it**
	// *(2026-08-17)*. Raising the set says "these cards are one blow", which is exactly what an
	// enemy's turn is not any more: three cards swing three times, in order, and the card that is
	// up is the card that is hitting.
	if combat.Plain(e.Action).Category() != combat.CategoryAttack || s.soloAttacker(side) {
		*mine = []int{seat}
	} else {
		*mine = attackSeats(order, side)
	}
	*theirs = nil
}

// attackSeats is every seat in one side's row holding an attack card.
//
// It counts along the same walk as noteResolved and seatPlayedCards — a seat is a position in
// that side's own row, not in the round — so a turn that opens with a Prepare still numbers its
// Strike as seat 1.
func attackSeats(order []combat.Slot, side combat.Side) []int {
	var (
		seats []int
		seat  int
	)
	for _, slot := range order {
		if slot.Side != side {
			continue
		}
		if slot.Card.Category() == combat.CategoryAttack {
			seats = append(seats, seat)
		}
		seat++
	}
	return seats
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

// noteHand narrows what is raised to the cards the hand was made of.
//
// The engine names them — see Event.HandCards — so this is a lookup, not a search. Nothing
// here knows what a Flurry is or how many cards one takes, which is what stops the table
// disagreeing with the hand that actually fired.
//
// **The cards need not be adjacent.** A counted hand like Two Pair is two cards, a card that
// earned nothing, and two more, so this takes the seats it is given rather than a span between
// the first and the last.
//
// **Raising is the whole of what says which cards earned it** *(2026-08-19)*. Every attack card
// of the turn is lifted by the time this arrives; the ones that built no hand drop back into the
// row here, and the yellow ring that used to be drawn round the survivors is gone — the shout
// names the hand and the row shows which cards are standing.
func (s *CombatScene) noteHand(e combat.Event) {
	if e.Kind != combat.KindHand {
		return
	}

	seats := make([]int, 0, e.HandCardCount)
	seats = append(seats, e.HandCards[:e.HandCardCount]...)

	if e.Side == combat.SideB {
		s.theatre.enemyFiringSeats = seats
		return
	}

	s.theatre.firingSeats = seats
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
	for _, r := range s.theatre.resolved {
		if r.handIndex == i {
			return true
		}
	}
	return false
}

// playedSeatOf finds the table seat of the card that came from hand slot i, so a card leaving
// at the end of the round sets off from where it actually is.
func (s *CombatScene) playedSeatOf(handIndex int) (int, bool) {
	for i, r := range s.theatre.resolved {
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
	case r.waiting():
		return from
	case !r.done():
		return lerpPoint(from, to, easeOut(r.progress()))
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
	for i, r := range s.theatre.resolved {
		at := r.at(gs, i, len(s.theatre.resolved), split, lit(s.theatre.firingSeats, i))
		drawCard(gs, screen, at, cards.Hand, r.card, heldBy(s.fighter.Duelist, r.card), true, false)
	}
}
