package screens

import (
	"image"
	"image/color"
	"math"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
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
// It is animated on its own clock rather than through eventDwellTicks. That is deliberate
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
	// The margin is to the card's own edge. The highlight ring reaches
	// deckHighlightInset + deckHighlightWidth/2 = 8 further, which still clears the screen,
	// and TestDeckStackClearsTheAPBarAndTheScreen fails rather than letting it not.
	deckStackRightMargin = 10
	deckStackDepth       = 3 // backs drawn behind the front one, to read as a pile
	deckStackStep        = 3 // pixels each one is offset up and left

	// deckStackTopGap clears the action-point bar. The stack's y is measured *down from the
	// bar* rather than set as a percentage, because the bar is the thing that constrains it —
	// a percentage that happened to clear it today would silently start overlapping the
	// moment the hand's geometry moved, and at 95% it already did by three pixels.
	deckStackTopGap = 8

	// deckHighlightWidth is the ring drawn round the stack while the overlay is open, in
	// attentionYellow.
	//
	// **The overlay is the only dialog in the game and it has no other exit** — no Escape
	// key, no right click — so the one live control has to be the most obvious thing on
	// screen. The Deck button used to get that for free by being a lit button on a dead
	// screen; a stack of dark card backs does not, which is what this replaces it with.
	deckHighlightWidth = 4
	deckHighlightInset = 6

	// How long a card takes to travel, and how far apart the drawn ones set off. Both in
	// ticks at 60 TPS: about a third of a second each, overlapping.
	flightTicks      = 20
	flightStaggerPer = 4

	// outboundDriftUp is how far a discarded card rises as it leaves, and outboundSpin how
	// far it turns. A card tossed flat off the side of the table reads as a bug; a little
	// lift and rotation reads as a throw.
	outboundDriftUp = 40
	outboundSpin    = -0.42 // radians at the end of the flight
	outboundShrink  = 0.72  // scale it reaches as it goes

	// A resolving card's three beats: up out of the hand, held where it can be read, then
	// down into the pile. They have to add up to less than eventDwellTicks or a card would
	// still be moving when the next one sets off.
	riseTicks = 16
	holdTicks = 24
	fallTicks = 16

	// firingGap is how far clear of the Resolution feed a card holds while it fires.
	//
	// **Cards resolve at full size, and what they must not cover is Resolution** — the
	// written record being made at the same moment. That has not changed; which side of it
	// they hang on has. Resolution used to be a pane across the top half and a card hung
	// *below* it, over the inert hand row. Resolution is a feed above the cards now, so a
	// card holds *above* the feed instead, in the band the pane vacated.
	//
	// The hand row is still what gets covered, and that is still the deliberate reading of
	// "leave them whole sized": during playback planning() is false, nothing down there can
	// be clicked or dragged, so the space is free at exactly the moment this needs it.
	firingGap = 12

	// The pile in the bottom-left corner. Overlapped hard, like the deck overlay's rows: the
	// left edge of a card carries its border colour and its cost dashes, which is enough to
	// read the shape of a round at a glance.
	//
	// **The inset has to clear the combo ring, not just the screen edge.** The ring is drawn
	// *around* the pile, so a card sitting flush against the left margin puts its bracket off
	// the screen — which is what the first version did, and it read as the combo highlight
	// having a missing side rather than as a card being too far left.
	pileLeftInset = 20
	pilePitch     = 46
)

// attentionYellow is the screen's one "look here" colour, and it has exactly two users: the
// ring round the deck stack while its overlay is open, and the ring round the cards a combo
// was formed from.
//
// **One colour, one meaning.** Both say "this is the thing right now", and a screen with two
// different attention colours has neither — so this is a single value rather than two that
// happen to match today. If a third caller wants it, the question to answer first is whether
// it means the same thing.
//
// **It is only ever drawn as a ring, never on a card.** A card's border is its element and
// nothing else may claim it; a combo says its piece in the space *around* the cards, which
// is also the only way to mark three of them as one thing — something the Resolution pane
// cannot do and is recorded as a known gap.
var attentionYellow = color.RGBA{R: 255, G: 214, B: 0, A: 255}

// The ring's weight and how far it stands off the cards. The inset is tight because the pile
// shares the hand row's bottom edge and the action-point figure is printed ten pixels under
// it — a wider standoff draws the bracket straight through "6/6 AP".
const (
	comboRingWidth = 5
	comboRingInset = 6
)

// cardFlight is one card in the air. Purely something to look at.
//
// **It stores an index and a row size, not a coordinate.** The hand re-lays out the instant
// a card leaves it, so a discarded card's origin no longer exists by the time it is drawn —
// slotAt takes the pair back and returns the rectangle that used to be there. It also means
// a flight survives the window being resized, which a cached pixel position would not.
type cardFlight struct {
	card actionCard

	// outbound is a card leaving the hand for the discard; the other direction is a card
	// dealt from the draw pile into the slot it now occupies.
	outbound bool

	// index and count locate the slot: for an outbound card the one it left and the size of
	// the row it left, for an inbound card the one it is arriving at and the row it joins.
	index, count int

	// fromPile says index is a position in the resolved pile rather than in the hand. A card
	// that fired this round spends the rest of it in the corner, so at the end of the round
	// it is thrown from there — not from the hand slot it left long before.
	fromPile bool

	// delay holds a card on the launch pad so a handful dealt at once set off in sequence
	// rather than as a single sheet. age runs to flightTicks once the delay is spent.
	age, delay int
}

// live reports whether a flight still has anything to draw.
func (f cardFlight) live() bool { return f.age < flightTicks }

// progress is 0 at the start of the journey and 1 at the end, before easing.
func (f cardFlight) progress() float64 {
	if f.age <= 0 {
		return 0
	}
	return float64(f.age) / float64(flightTicks)
}

// addFlight queues one. Kept as a method so the two call sites in spendSelected read as
// what they are rather than as slice manipulation.
func (s *CombatScene) addFlight(f cardFlight) {
	s.flights = append(s.flights, f)
}

// updateFlights advances every card in the air and drops the ones that have landed.
//
// Called unconditionally from Update — before the branches that return early on a settled
// duel, and outside the guard that stops the deck overlay reaching the action box. A card
// mid-flight when the killing blow lands should still finish its journey, and the overlay
// covers the flight rather than cancelling it.
func (s *CombatScene) updateFlights() {
	if len(s.flights) == 0 {
		return
	}
	kept := s.flights[:0]
	for _, f := range s.flights {
		if f.delay > 0 {
			f.delay--
			kept = append(kept, f)
			continue
		}
		f.age++
		if f.live() {
			kept = append(kept, f)
		}
	}
	s.flights = kept
}

// inboundTo reports whether a card is currently flying into hand slot i, so the row can
// leave that slot empty until it lands.
//
// **The card is already in the hand** — spendSelected put it there before this file saw it,
// which is the whole reason the budget, the queue and every predicate stayed simple. What
// is suppressed is the *drawing* of a card that is on screen somewhere else, which is a
// view concern and lives here.
func (s *CombatScene) inboundTo(i int) bool {
	for _, f := range s.flights {
		if !f.outbound && f.index == i {
			return true
		}
	}
	return false
}

// deckStackRect is the front card of the pile: the one that is drawn on top and the one a
// click is tested against.
//
// **Sized and placed against the strip below the action-point bar, which is what forced
// cards.Stack to be smaller than Mini.** The hand row ends at handTopPct plus a card, the AP
// figure and bar hang below that, and what is left before the bottom edge is 86 pixels — so
// a 132-pixel half-size card does not fit here however much one would read better. See the
// Stack style for why a *back* survives that where a face would not.
//
// The top is measured down from the bar rather than set as a percentage of the screen. The
// bar is the constraint, so it should be the anchor: 95% of the height put the pile three
// pixels *through* the bar, and would have gone on doing so silently every time the hand
// geometry moved.
func deckStackRect(gs *state.GlobalState) image.Rectangle {
	w, h := cards.Stack.Width, cards.Stack.Height

	barBottom := gs.PctY(handTopPct) + cardHeight + apBarBelow + apBarHeight
	top := barBottom + deckStackTopGap + (deckStackDepth-1)*deckStackStep
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
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	if image.Pt(gs.MouseX, gs.MouseY).In(deckStackBounds(gs)) {
		s.toggleDeck()
	}
}

// drawDeckStack draws the pile, and the ring round it when the overlay is open.
//
// The depth is fixed rather than proportional to how many cards are left. A pile that
// visibly thinned would be a nice touch and a lie: the discard is folded back in and
// reshuffled the moment the draw pile empties, so "how deep is the deck" is not a fact the
// player can act on and not one worth drawing. The count beside the AP bar is the honest
// version of that number.
func (s *CombatScene) drawDeckStack(gs *state.GlobalState, screen *ebiten.Image) {
	front := deckStackRect(gs)

	if s.showDeck {
		b := deckStackBounds(gs).Inset(-deckHighlightInset)
		vector.StrokeRect(screen,
			float32(b.Min.X), float32(b.Min.Y), float32(b.Dx()), float32(b.Dy()),
			deckHighlightWidth, attentionYellow, false)
	}

	// Back to front, so the front card is the one on top and the one the click tests.
	for i := deckStackDepth - 1; i >= 0; i-- {
		off := i * deckStackStep
		s.drawCardBack(gs, screen, image.Pt(front.Min.X-off, front.Min.Y-off), cards.Stack)
	}
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
	for _, f := range s.flights {
		if f.delay > 0 {
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
	if f.fromPile {
		from = pileAt(gs, f.index)
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

	drawFlyingCard(gs, screen, cardSpec(f.card, true, false, s.fighter.Str), cards.Hand, geo)
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

	style, spec := cards.Hand, cardSpec(f.card, true, false, s.fighter.Str)
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

// drawFlyingCard blits a card through an arbitrary transform.
//
// **This is the whole of what animation needed from the card pipeline.** internal/cards
// already renders into a plain Go image and card_art.go already caches it as an
// *ebiten.Image, so a card in flight is the same cached picture the hand draws with a
// different matrix in front of it — no new renders, and not one extra cache entry, because
// the transform happens after the lookup.
//
// Linear filtering, unlike drawCard, which leaves it at the default. A card being scaled
// and turned shimmers under nearest-neighbour; a card at rest must not be filtered at all,
// because the glyphs on it are 1:1 pixel art. Flights land at scale 1 and hand the card
// back to drawCard, so the smoothing only ever applies while something is moving.
func drawFlyingCard(gs *state.GlobalState, screen *ebiten.Image, spec cards.Spec, st cards.Style, geo ebiten.GeoM) {
	img := cardImage(gs, spec, st)
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM = geo
	screen.DrawImage(img, op)
}

// resolvedCard is one of the player's cards that has fired this round, and where it is on
// its way from the hand to the pile.
//
// **The pile is the round's own history, laid out in the order it happened.** Resolution
// regroups a queue into prepare, then attacks, then defenses, so the pile grows in phase
// order without anything here knowing what a phase is — it appends in the order the engine
// played, and the order does the rest. That is the whole reason this reads as the mechanic
// rather than as an animation.
type resolvedCard struct {
	card actionCard

	// Where it came from: the hand slot it occupied and the row it belonged to, so the rise
	// starts from the card's own place. Same reason cardFlight stores the pair — the hand is
	// still holding this card, but a later discard could re-lay the row out around it.
	handIndex, handCount int

	// age drives rise, hold and fall. It stops climbing once the card has landed, so a card
	// parked in the pile costs one comparison a frame rather than growing forever.
	age int

	// combo marks a card a combo bracketed. Set when the KindCombo event plays back, from
	// the span the engine put on the event — never worked out here.
	combo bool
}

// pileAt is where the nth resolved card sits in the bottom-left corner.
//
// It shares the hand row's top edge rather than picking its own, so the pile and the hand
// read as one band with cards moving between them, and a card that has landed is the same
// size and on the same line as the cards it came from.
func pileAt(gs *state.GlobalState, n int) image.Point {
	return image.Pt(pileLeftInset+n*pilePitch, gs.PctY(handTopPct))
}

// firingAt is where a card holds while it is resolving: centred across the screen, sitting
// just above the Resolution feed.
//
// **Measured off the feed's collapsed top, not the expanded one.** A card that jumped
// whenever the box was held would be an animation reacting to an input it has nothing to do
// with. An expanded feed reaches up past this, so a firing card is drawn over its older
// lines — see the draw-order ranking in combat.go, where that is the one thing deliberately
// given up. It clears y=467, so the newest lines are never covered.
func firingAt(gs *state.GlobalState) image.Point {
	top := gs.PctY(handTopPct) - feedGapAboveCards - feedHeight()
	return image.Pt(
		gs.PctX(50)-cardWidth/2,
		top-firingGap-cardHeight,
	)
}

// noteResolved records that one of the player's cards has just fired, and starts it moving.
//
// **It asks combat.ResolutionOrder which card this event belongs to rather than counting.**
// The order regroups a queue by category, so the third card to resolve is not the third card
// in the hand, and a screen keeping its own tally would light the wrong one the first time
// somebody queued a defense before an attack. currentSlot is already the authority on how
// far through that order playback has reached.
func (s *CombatScene) noteResolved(e combat.Event) {
	if e.Kind != combat.KindAction || e.Side != combat.SideA {
		return
	}

	order := combat.ResolutionOrder(s.fighterActions, s.enemyActions)
	i, ok := s.currentSlot()
	if !ok || i >= len(order) {
		return
	}

	slot := order[i]
	if slot.Side != combat.SideA {
		return
	}

	// No card behind the queued action. The real game cannot reach this — syncQueue derives
	// the queue from the hand — but the scripted demo writes a plan straight into
	// fighterActions, and a screen that panicked or drew an arbitrary card because a harness
	// took a shortcut would be worse than one that draws nothing.
	hand, ok := s.handIndexForQueue(slot.Index)
	if !ok {
		return
	}

	s.resolved = append(s.resolved, resolvedCard{
		card:      s.hand[hand].actionCard,
		handIndex: hand,
		handCount: len(s.hand),
	})
}

// noteCombo brackets the cards a combo was formed from.
//
// The engine says which run formed it — see Event.ComboStart — so this is a slice, not a
// search. Nothing here knows what a Flurry is or how long one runs, which is what stops the
// bracket disagreeing with the combo that actually fired.
func (s *CombatScene) noteCombo(e combat.Event) {
	if e.Kind != combat.KindCombo || e.Side != combat.SideA {
		return
	}
	for i := e.ComboStart; i < e.ComboStart+e.ComboLength && i < len(s.resolved); i++ {
		s.resolved[i].combo = true
	}
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

// updateResolved advances the cards that have fired. Its own clock, like the flights.
func (s *CombatScene) updateResolved() {
	for i := range s.resolved {
		if s.resolved[i].age < riseTicks+holdTicks+fallTicks {
			s.resolved[i].age++
		}
	}
}

// resolvedInHand reports whether the card in hand slot i has already fired, so the row can
// stop drawing it — it is on screen somewhere between the hand and the pile.
//
// The card has *not* left the hand. It leaves at the end of the round, with everything else
// that was played, which is what keeps the Resolution pane able to narrate from
// fighterActions while the round is still running. This hides a drawing, exactly like
// inboundTo.
func (s *CombatScene) resolvedInHand(i int) bool {
	for _, r := range s.resolved {
		if r.handIndex == i {
			return true
		}
	}
	return false
}

// pileIndexOf finds the pile position of the card that came from hand slot i, so a card
// leaving at the end of the round sets off from where it actually is.
func (s *CombatScene) pileIndexOf(handIndex int) (int, bool) {
	for i, r := range s.resolved {
		if r.handIndex == handIndex {
			return i, true
		}
	}
	return 0, false
}

// at is where a resolved card is drawn right now: rising, holding, falling, or parked.
func (r resolvedCard) at(gs *state.GlobalState, pileIndex int) image.Point {
	from := slotAt(gs, r.handIndex, r.handCount)
	fire := firingAt(gs)
	pile := pileAt(gs, pileIndex)

	switch {
	case r.age < riseTicks:
		return lerpPoint(from, fire, easeOut(float64(r.age)/riseTicks))
	case r.age < riseTicks+holdTicks:
		return fire
	case r.age < riseTicks+holdTicks+fallTicks:
		t := float64(r.age-riseTicks-holdTicks) / fallTicks
		return lerpPoint(fire, pile, easeIn(t))
	default:
		return pile
	}
}

// firing reports whether this card is the one currently being read, which is the moment its
// combo bracket is worth drawing around it.
func (r resolvedCard) firing() bool {
	return r.age >= riseTicks && r.age < riseTicks+holdTicks
}

// drawResolvedCards draws the pile and whatever is on its way into it.
//
// Oldest first, so a newer card overlaps the one before it and the pile reads left to right
// in the order the round happened — and so the card currently rising is drawn over the
// cards already parked rather than sliding underneath them.
func (s *CombatScene) drawResolvedCards(gs *state.GlobalState, screen *ebiten.Image) {
	for i, r := range s.resolved {
		drawCard(gs, screen, r.at(gs, i), cards.Hand, r.card, true, false, s.fighter.Str)
	}

	s.drawComboBracket(gs, screen)
}

// drawComboBracket rings the cards a combo was formed from.
//
// **A ring around the group, not a colour on the cards.** A card's border is its element and
// nothing else may claim it, so a combo says its piece in the space around the cards — which
// is also the only way to say "these three together", something the Resolution pane has
// never been able to show and is recorded as a known gap.
func (s *CombatScene) drawComboBracket(gs *state.GlobalState, screen *ebiten.Image) {
	var box image.Rectangle
	for i, r := range s.resolved {
		if !r.combo {
			continue
		}
		at := r.at(gs, i)
		card := image.Rect(at.X, at.Y, at.X+cardWidth, at.Y+cardHeight)
		if box.Empty() {
			box = card
			continue
		}
		box = box.Union(card)
	}
	if box.Empty() {
		return
	}

	box = box.Inset(-comboRingInset)
	vector.StrokeRect(screen,
		float32(box.Min.X), float32(box.Min.Y), float32(box.Dx()), float32(box.Dy()),
		comboRingWidth, attentionYellow, false)
}

// lerpPoint walks between two points. Integer output because a resolved card is drawn at
// full size and scale 1 — there is no resampling to hide sub-pixel stepping, and none is
// needed at this speed.
func lerpPoint(from, to image.Point, t float64) image.Point {
	return image.Pt(
		from.X+int(float64(to.X-from.X)*t),
		from.Y+int(float64(to.Y-from.Y)*t),
	)
}

// easeOut decelerates into the destination: fast off the pile, settling into the slot.
func easeOut(t float64) float64 { return 1 - (1-t)*(1-t) }

// easeIn accelerates away: a card tossed aside picks up speed rather than drifting off at a
// constant rate, which reads as being thrown instead of as sliding.
func easeIn(t float64) float64 { return t * t }
