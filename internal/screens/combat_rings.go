package screens

// The ring pane: what the player is wearing, drawn as cards.
//
// **It draws what the run is wearing** *(2026-08-17)*. The worn set lives on `session.Session` —
// which is what makes it survive a fight — and every rule a ring has is in `data/rings.json` in the
// `When` / `If` / `Then` grammar. This file is the row; it no longer decides anything. **The loop
// around it landed on 2026-08-21** — see shop.go, which draws the same ring cards on a shelf and is
// the only thing that puts one on or takes one off.
//
// **It claims the band the full-height panes vacated.** Action Flow is built and not drawn,
// and Resolution left for the three-line feed above the hand on 2026-08-11, so 12–46% was
// empty. That is what paid for full-size ring cards; the alternative was a row of chips at deck
// -stack size, which would have made the ring a different object from every other card in the
// game.

import (
	"fmt"
	"image"
	"log"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"math"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"image/color"
)

// maxRings is how many rings can be worn at once.
//
// **Five, and it is a rule rather than a number the layout chose** — brands are what expand it, and
// nothing else may. It reads `combat.MaxWornRings` rather than declaring a second five: the rule
// moved into the engine with the grammar on 2026-08-17, exactly as `maxSelected` did before it, and
// the row saying `worn/5` while the duelist held a sixth is the drift this prevents.
const maxRings = combat.MaxWornRings

const (
	// The gap on either side of the row: it starts where the duelist card ends and stops
	// where the enemy card begins.
	//
	// **Both edges are read off the cards themselves** *(2026-08-12)*. The right edge was a
	// hardcoded 79%, chosen to clear an enemy card centred at 88% — a percentage standing in
	// for a position it could not see, and one that would have quietly overlapped the moment
	// either card moved. It moved the next day.
	ringPaneGap = 16

	// **The row sits ten pixels below the cards on either side of it** *(2026-08-12)*, where
	// it was flush with their tops for a day.
	//
	// Flush alignment and a backing panel do not both work: the backing's top edge would land
	// exactly on the two cards' top edges, and three things sharing one line reads as a single
	// wide object with two cards embedded in it — which is the "cards trapped in a panel"
	// failure the framed version was retired for, made worse by the frame now being wider than
	// the row. Dropping the row breaks that line, and the offset is what makes the backing
	// legible as a thing *behind* the rings rather than a border *around* everything.
	ringPaneTopDrop = 10

	// How far the rule sits below the cards, and how thick it is.
	ringRuleGap   = 12
	ringRuleWidth = 2

	// The cap fraction under the bottom-right corner, sized and spaced like the deck pile's
	// count, which is the same idea: a number saying how much of a fixed thing is in use.
	ringCountSize     = 22
	ringCountTopGap   = 6
	ringCountRightPad = 2

	// ringPaneBackPad is how far the backing extends past the row on every side. The pitch
	// puts the first card flush left and the last flush right, so with no padding the two end
	// cards would sit on the backing's edge and it would read as a border drawn around them
	// rather than as a surface they stand on.
	//
	// **It has to stay under ringPaneGap**, or the backing runs into the fighter card beside
	// it — 8 against 16 leaves half the gap still showing on each side.
	ringPaneBackPad = 8
)

// ringRuleColor is the line under the rings.
//
// **Neutral rather than the ring's pink** *(2026-08-11)*. The pane was a full pink box —
// filled, bordered and titled — for its first hour and the box was the loudest thing in the
// band, competing with the five saturated pink borders standing inside it. What is left is a
// rule: it says where the row ends and nothing else, so it takes a grey that is read as
// structure rather than as a colour meaning something.
var ringRuleColor = color.RGBA{R: 120, G: 122, B: 132, A: 255}

// ringPaneBackColor is the surface the rings stand on: one step off `screenGround`, and
// nothing else.
//
// **A fill, not a frame** *(2026-08-12)*, and the distinction is the whole history of this
// pane. The first version was a full pink box — filled, bordered and titled — and it was the
// loudest thing in the band, competing with five saturated pink borders standing inside it. It
// came out the same hour, leaving the cards to be the pane. What that lost is the thing being
// put back now: with the row spanning most of the screen's width and a fighter card at either
// end, nothing said where the middle *began*.
//
// So it is the quietest possible answer to that: one step off the background, no border, no
// title, no hue. A colour that meant something would put it back in competition with the
// borders it sits behind.
//
// **The step goes down now that the ground is cream** *(2026-08-14)*. It was one step
// *lighter* than {50,50,50} for as long as the screen was dark, and kept that way it would be
// a near-white slab — the loudest thing in the band, which is the exact failure this pane was
// cut back to a fill to avoid. Which direction "one step" means is a function of the ground.
var ringPaneBackColor = color.RGBA{R: 207, G: 189, B: 156, A: 255}

// ringPaneRect is the row's extent: the cards' own band, running between the two corner cards
// and dropped ringPaneTopDrop below them.
//
// **It is the middle of a three-part row** — duelist card, rings, enemy card — so it takes its
// edges from its neighbours rather than from percentages of the screen. Whichever card moves,
// the row follows, and the one thing that cannot happen is a ring drawn underneath one of them.
//
// **The rectangle is the cards and the rule, not the backing.** It is what the slots are cut
// out of and what the rule and the fraction hang off; the backing is derived from it — see
// ringPaneBackRect — so growing the padding cannot silently move a ring.
func (s *CombatScene) ringPaneRect(gs *state.GlobalState) image.Rectangle {
	duelist, enemy := s.duelistCardRect(gs), s.enemyCardRect(gs)

	left := duelist.Max.X + ringPaneGap
	top := duelist.Min.Y + ringPaneTopDrop
	right := enemy.Min.X - ringPaneGap
	bottom := top + cards.RingStyle.Height + ringRuleGap

	return image.Rect(left, top, right, bottom)
}

// ringPaneBackRect is the surface drawn behind the row: the row padded on every side, and
// **deep enough to hold the rule and the fraction under it**.
//
// The fraction hanging off the bottom edge onto the bare ground would say the rule is the
// panel's floor and the number is loose underneath it, which is backwards — the count belongs
// to the row it counts.
func (s *CombatScene) ringPaneBackRect(gs *state.GlobalState) image.Rectangle {
	r := s.ringPaneRect(gs).Inset(-ringPaneBackPad)
	r.Max.Y = s.ringPaneRect(gs).Max.Y +
		ringRuleWidth + ringCountTopGap + ringCountSize + ringPaneBackPad
	return r
}

// ringSlotMaxGap is the most bare table ever left between two ring cards.
//
// **A row's pitch is capped and then the row is centred** *(2026-08-24)*. Without the cap the row
// spread to whatever pane it was handed, so a run wearing two rings put one against the duelist
// card and the other in the far corner of an empty screen — two rings reading as two unrelated
// things rather than as one build. The cap is what makes a growing row *grow*: rings sit at a
// fixed pitch and the row widens outwards from the middle as one is added, up to the point where
// five of them fill the pane and the pitch has to close up again.
//
// 22 is the gap five rings leave in the combat screen's pane, so the fullest row on the screen the
// pitch was originally derived from is drawn exactly where it always was.
const ringSlotMaxGap = 22

// ringSlotPitch is how far apart two ring cards start, **for the number actually worn**.
//
// **The row spreads to fill the pane and closes up as it fills** *(2026-08-11)*. Three rings
// stand well apart and fully visible; five sit shoulder to shoulder with a few pixels between
// them, since 5 x 162 is 810 against roughly 825 of pane. **It overlapped by 26 pixels each
// until the cards came down a tenth in width later the same day**, and it would again the
// moment the pane narrowed or a sixth slot was ever allowed — the pitch is derived, so the row
// closes up by itself rather than by anyone redoing this arithmetic. Overlapping is the
// accepted failure mode, not shrinking: a card cannot be scaled, a smaller ring is a
// *different drawing*, and there is no ring style below this one.
//
// **The spread is capped at ringSlotMaxGap and the row is centred in the pane by ringSlotAt**, so
// a pane wider than the rings in it leaves its slack at both ends rather than between the cards.
func ringSlotPitch(r image.Rectangle, worn int) int {
	if worn < 2 {
		return 0
	}
	pitch := (r.Dx() - cards.RingStyle.Width) / (worn - 1)
	if max := cards.RingStyle.Width + ringSlotMaxGap; pitch > max {
		return max
	}
	return pitch
}

// ringSlotRowWidth is how much of the pane the row actually occupies: every pitch but the last,
// plus the card that sits on the final one.
func ringSlotRowWidth(r image.Rectangle, worn int) int {
	if worn < 1 {
		return 0
	}
	return (worn-1)*ringSlotPitch(r, worn) + cards.RingStyle.Width
}

// ringSlotAt is where the i'th ring card's top-left corner sits. **Flush with the pane's top**,
// which is the top of the character block beside it — the two are aligned directly rather than
// each being inset inside a frame of its own.
//
// **Horizontally the row is centred on the pane**, which is only visible once the pitch is capped:
// a full row still starts where it always did, because there is no slack left to share out.
func ringSlotAt(r image.Rectangle, i, worn int) image.Point {
	left := r.Min.X + (r.Dx()-ringSlotRowWidth(r, worn))/2
	return image.Pt(left+i*ringSlotPitch(r, worn), r.Min.Y)
}

// wornRings is what the player is wearing, as records, in worn order.
//
// **The run is the authority and this is the lookup** *(2026-08-17)*. `session.Session` holds the
// worn keys — in worn order, which is a rule, since rings fire left to right — and `gs.Rings` holds
// the record each key names, for its art and its name. The screen decides nothing.
//
// **A worn key with no record is reported, not ignored.** It is the failure `ParseElement` refuses to
// fall back on: a ring that quietly does not draw looks exactly like a ring that was never bought.
func wornRings(gs *state.GlobalState) []data.RingData {
	if gs.Run == nil {
		return nil
	}

	out := make([]data.RingData, 0, maxRings)
	for _, key := range gs.Run.Worn() {
		if len(out) == maxRings {
			break
		}
		record, ok := gs.Rings[key]
		if !ok {
			log.Printf("the run is wearing %q, which is in no record", key)
			continue
		}
		out = append(out, record)
	}
	return out
}

// ringRow is the worn row as a draggable row of cards. **The lifecycle is carddrag.go's**; this is
// the half that knows the row's geometry and what a drop means.
//
// **Worn order is the order rings fire in**, so a drop here changes what a duel does — this is the
// one draggable row in the game where the gesture is a rule and not an arrangement. See
// Session.MoveRing.
//
// **Nothing is lifted out of anything.** The run is the authority on what is worn and it is not
// touched until the drop, so `rowLift` does nothing at all and the drawing skips the seat the drag
// says is empty. The hand's row does remove its card, because there the list *is* the hand — see
// handRow, and dragRow for why the two are allowed to differ.
//
// **Every screen that shows the row builds one of these**, with its own rectangle and its own idea
// of what a click means: the combat screen's click does nothing, the shop's arms a sale.
type ringRow struct {
	rect  image.Rectangle
	worn  int
	click func(i int)
	move  func(from, to int)
}

func (r ringRow) rowLen() int { return r.worn }

func (r ringRow) rowSlot(gs *state.GlobalState, i int) image.Rectangle {
	at := ringSlotAt(r.rect, i, r.worn)
	return image.Rect(at.X, at.Y, at.X+cards.RingStyle.Width, at.Y+cards.RingStyle.Height)
}

// rowZone is the row's own rectangle. **Tighter than the hand's band**, deliberately: the hand
// stands alone at the bottom of the screen with nothing beside it, where this row has a duelist
// card at one end and an enemy card or a margin at the other. A zone spanning the width would make
// a drop on the duelist card a reorder.
func (r ringRow) rowZone(gs *state.GlobalState) image.Rectangle { return r.rect }

// rowDropIndex is which seat the cursor is over, measured in pitches from the row's left edge and
// from the middle of a step rather than its edge — the hand's arithmetic, over the ring row's
// pitch, because once five rings are worn these overlap too.
//
// **Clamped to a seat that exists**, unlike the hand's, which may land one past the end: nothing is
// being inserted here. Five rings reordered are still five rings.
func (r ringRow) rowDropIndex(gs *state.GlobalState) int {
	if r.worn < 2 {
		return 0
	}

	pitch := ringSlotPitch(r.rect, r.worn)
	idx := (gs.MouseX - ringSlotAt(r.rect, 0, r.worn).X + pitch/2) / pitch
	if idx < 0 {
		idx = 0
	}
	if idx > r.worn-1 {
		idx = r.worn - 1
	}
	return idx
}

// rowLift is deliberately empty. See the type comment.
func (r ringRow) rowLift(int) {}

func (r ringRow) rowReturn(from, to int) {
	if r.move != nil {
		r.move(from, to)
	}
}

func (r ringRow) rowClick(i int) {
	if r.click != nil {
		r.click(i)
	}
}

// moveWornRing is the run's half of a reorder, and the half every screen shares. A screen holding a
// live duelist has a second half — see CombatScene.moveRing.
func moveWornRing(gs *state.GlobalState, from, to int) bool {
	if gs.Run == nil {
		return false
	}
	return gs.Run.MoveRing(from, to)
}

// drawDraggedRing draws the ring riding the cursor, over everything else on the row.
//
// **It is drawn from the run rather than from anything the drag is carrying**, which is what the
// empty rowLift buys: there is only ever one copy of what is worn, so a card in flight cannot
// disagree with the row it came out of.
func drawDraggedRing(gs *state.GlobalState, screen *ebiten.Image, drag *cardDrag,
	counters map[string]string) {

	if !drag.dragging() {
		return
	}
	worn := wornRings(gs)
	if drag.origin() >= len(worn) {
		return
	}

	record := worn[drag.origin()]
	drawRingCard(gs, screen, drag.at(gs), record, counters[record.RingRecord], true, true)
}

// ringCounter is one worn ring's accumulator, formatted for the badge in the corner of its card.
//
// **The figure is what the ring is doing, not how far it has counted** *(owner's call,
// 2026-08-26)*. Enflamed's accumulator is 50 when the ring is doing 1.5x damage, and a badge
// reading `50` would be a number in units nothing on screen explains. `combat.GrowthEffect` is what
// resolves the one numeric effect the accumulator feeds and `combat.Scaling` says whether that
// figure is a percentage — so a multiplier reads as a multiplier and flat life reads as life.
//
// **A ring that does not grow has no badge**, which is most of the catalogue: an empty string draws
// nothing. That is the whole distinction the badge is for — a card carrying one is a card whose
// number is still moving.
func ringCounter(w combat.WornRing) string {
	e, ok := combat.GrowthEffect(w)
	if !ok {
		return ""
	}
	if combat.Scaling(e.Do) {
		return fmt.Sprintf("%.1fx", float64(e.Amount)/100)
	}
	return fmt.Sprintf("%+d", e.Amount)
}

// ringCounters is a badge per worn ring, by record key.
//
// **Keyed by record and not by position**, exactly as the accumulator itself is: the row is about
// to become something the player can drag into a different order, and a badge indexed by seat would
// follow the finger rather than the ring.
func ringCounters(worn []combat.WornRing) map[string]string {
	out := make(map[string]string, len(worn))
	for _, w := range worn {
		if c := ringCounter(w); c != "" {
			out[combat.RingOf(w.Ring).Key] = c
		}
	}
	return out
}

// runCounters is the badges as the run holds them: what a ring has banked between fights.
//
// **The two callers are the screens with no duel on them** — the reward screen's build band and the
// shop. The combat screen reads the duelist instead, and mid-blow the sum: see countersNow.
func runCounters(gs *state.GlobalState) map[string]string {
	if gs.Run == nil {
		return nil
	}
	return ringCounters(gs.Run.WornRings())
}

// countersNow is the badges as the *round being played back* has got to.
//
// **A growing ring steps between the terms of one blow as of 2026-08-26** *(owner's call)*, so the
// duelist the screen is holding is a round behind while the sum is being read: `endOfRound` adopts
// the resolved duelist only once playback is finished. The hand dialog carries the accumulator each
// term left behind, so the row can step its badges on the beat the figure lands — which is the whole
// point of moving the growth into the sum. The player watches the number that is about to price the
// next card go up.
//
// **It reads figures the resolver produced and computes none**, exactly like the sum itself. Off the
// dialog it falls back to the duelist, which is every frame outside a blow.
func (s *CombatScene) countersNow() map[string]string {
	worn := s.fighter.Duelist.WornRings()
	if grown, ok := s.theatre.mathBox.growthNow(combat.SideA); ok {
		worn = withGrown(worn, grown)
	}
	return ringCounters(worn)
}

// withGrown is a worn set with the accumulators one beat of a sum reached, as a copy.
//
// **A copy, because the duelist it came from is the fight's own** — this is a picture of a number
// part way through a round, and writing it back would be presentation changing an outcome.
func withGrown(worn []combat.WornRing, grown [combat.MaxWornRings]int) []combat.WornRing {
	out := make([]combat.WornRing, len(worn))
	copy(out, worn)
	for i := range out {
		if i < len(grown) {
			out[i].Grown = grown[i]
		}
	}
	return out
}

// The shake a card makes on the beat it does its work.
//
// **A card that does work should be seen doing it** *(owner's call, 2026-08-26)*. The figures leave
// the cards and land in the sum; without the card moving, the number appears to come from nowhere
// and the row of rings sits inert through the one moment it is earning its place.
//
// **Side to side rather than a jump** *(owner's call)*. A card that leaps reads as being *picked*,
// which is what the selected-card lift already means in the hand and what the played row's lift
// means on the table — two vertical vocabularies already spoken for. Sideways is unused and reads as
// a thing rattling as it fires.
var (
	// ringShakeTicks is how long one shake lasts. **Under a term's own flight**, because the figures
	// arrive one after another and a shake still running when the next one starts would smear the
	// beats together.
	ringShakeTicks = beat(3, 5)

	// ringShakeSwings is how many times the card crosses its own centre. Three reads as a rattle;
	// one reads as a nudge and five as a wobble.
	ringShakeSwings = 3.0

	// ringShakeWidth is how far it travels either side at the start. **It decays to nothing** over
	// the shake, so the card settles rather than stopping mid-swing.
	ringShakeWidth = 7
)

// shakeOffset is how far sideways a card sits this frame: a decaying oscillation that ends where it
// started.
//
// **A decaying sine rather than an ease**, because the card has to come back to where it was and be
// still when it gets there. Every other movement in the game is a journey from one seat to another
// and eases into its destination; this one has no destination.
func shakeOffset(t travel) int {
	if t.done() {
		return 0
	}
	p := t.progress()
	return int(math.Sin(p*math.Pi*2*ringShakeSwings) * (1 - p) * float64(ringShakeWidth))
}

// tickShakes starts a shake on whatever the sum has just reached, and advances the ones already
// running. Called every tick from Update.
//
// **It fires on the beat one figure of the sum is written**, watching the hand dialog's own item
// cursor rather than keeping a second clock: the number leaving a card and the card rattling are the
// same event, and two clocks would eventually disagree about when it happened.
//
// **It may not change an outcome**, like every other thing on this screen that moves.
func (s *CombatScene) tickShakes(gs *state.GlobalState) {
	for i := range s.ringShake {
		s.ringShake[i].tick()
	}
	for i := range s.cardShake {
		s.cardShake[i].tick()
	}

	at := s.theatre.mathBox.at
	if at == s.shakeItem {
		return
	}
	s.shakeItem = at

	rings, card, ok := s.theatre.mathBox.shaking(combat.SideA)
	if !ok {
		return
	}
	for seat, shaking := range rings {
		if shaking && seat < len(s.ringShake) {
			s.ringShake[seat] = newTravel(0, ringShakeTicks)
		}
	}
	if card > 0 {
		s.shakePlayedCard(card - 1)
	}
}

// shakePlayedCard starts one played card rattling, growing the row of clocks if the table is holding
// more cards than it has seen before.
//
// **A slice rather than a fixed array**, unlike the rings: a worn row is capped at five by a rule,
// and the number of cards on the table is capped by an action budget that a ring can make cheaper.
func (s *CombatScene) shakePlayedCard(seat int) {
	if seat < 0 {
		return
	}
	for len(s.cardShake) <= seat {
		s.cardShake = append(s.cardShake, travel{})
	}
	s.cardShake[seat] = newTravel(0, ringShakeTicks)
}

// playedCardShake is how far sideways the played card in one seat sits this frame.
func (s *CombatScene) playedCardShake(seat int) int {
	if seat < 0 || seat >= len(s.cardShake) {
		return 0
	}
	return shakeOffset(s.cardShake[seat])
}

// ringCardCentre is the middle of one worn seat's card, which is where that ring's multiplier sets
// off from on its way into the sum.
//
// **It reads the same two functions the row is drawn with** — `ringPaneRect` and `ringSlotAt` — so a
// figure cannot set off from a seat the card is not in. That is the rule every origin on this screen
// follows; see `handCardCentre`.
func (s *CombatScene) ringCardCentre(gs *state.GlobalState, seat int) image.Point {
	at := ringSlotAt(s.ringPaneRect(gs), seat, len(wornRings(gs)))
	return image.Pt(at.X+cards.RingStyle.Width/2, at.Y+cards.RingStyle.Height/2)
}

// drawRingPane draws the backing, the rings, a rule under them, and the cap as a fraction on
// its right end.
//
// **There is still no box** *(2026-08-12)*. The backing that arrived today is a fill and not a
// frame — one step lighter than the screen, no border, no title, no hue — which is a different
// thing from the pink panel this pane started as and was stripped of on 2026-08-11. What that
// stripping went too far on is legibility of the *edges*: with a fighter card at either end of
// a row spanning most of the screen, nothing said where the middle began. See
// ringPaneBackColor.
//
// **Empty slots are not drawn.** They were the first sketch and the fraction replaced them:
// five frames of which two are dashed outlines spends the loudest thing in the row on saying
// what you have *not* got, where `3/5` says the same thing on the end of the rule.
// MECHANICS.md's "the cap is never displayed — it surfaces when you try to buy a sixth" is the
// rule this softens, and softening it is the owner's call: the fraction is the deck pile's
// idea, where a count of a fixed total is read without being looked for.
func (s *CombatScene) drawRingPane(gs *state.GlobalState, screen *ebiten.Image) {
	r := s.ringPaneRect(gs)

	// The surface first, so everything else stands on it.
	back := s.ringPaneBackRect(gs)

	// **Flat, where the deck panel and the fight log are bevelled** *(owner's call, 2026-08-24)*.
	// It was sunken for an afternoon, on the argument that the ring cards stand *in* it. What that
	// misses is what is standing there: five bevelled cards on a bevelled tray on a bevelled
	// screen is three depths in one corner, and the cards are the thing meant to be read. The two
	// panels that keep their bevel are overlays — they cover the game, so a lit edge is what says
	// they are in front of it. This backing covers nothing.
	vector.DrawFilledRect(screen,
		float32(back.Min.X), float32(back.Min.Y), float32(back.Dx()), float32(back.Dy()),
		ringPaneBackColor, false)

	worn := wornRings(gs)
	counters := s.countersNow()
	for i, ring := range worn {
		// **The seat a dragged ring left is drawn empty rather than closed up**, which is the
		// hand's rule too: the row keeps its width and its pitch while a card is up, so nothing
		// slides sideways under the cursor mid-drag.
		if s.ringDrag.dragging() && i == s.ringDrag.origin() {
			continue
		}

		// **The shake and the toast go together**: the card rattles and its border lights, which is
		// what says the ring is working rather than merely moving.
		at := ringSlotAt(r, i, len(worn))
		shake := shakeOffset(s.ringShake[i])
		at.X += shake

		drawRingCard(gs, screen, at, ring, counters[ring.RingRecord], true, !s.ringShake[i].done())
	}

	// The rule: the row's whole width, whatever is standing on it. **It is drawn even with no
	// rings equipped**, which is deliberate — an empty band with a line under it says the row
	// exists and is empty, where nothing at all says the screen forgot to draw something.
	vector.DrawFilledRect(screen,
		float32(r.Min.X), float32(r.Max.Y), float32(r.Dx()), ringRuleWidth,
		ringRuleColor, false)

	// **`worn / cap`, exactly like the pile's `left / owned`.** The numerator is what moves and
	// the denominator deliberately never does, so the figure is read as "three of your five
	// fingers are spoken for" rather than as two unrelated numbers.
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(r.Max.X-ringCountRightPad),
		float64(r.Max.Y+ringRuleWidth+ringCountTopGap))
	op.PrimaryAlign = text.AlignEnd
	op.ColorScale.ScaleWithColor(groundInk)
	text.Draw(screen, fmt.Sprintf("%d/%d", len(worn), maxRings),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: ringCountSize}, op)

	// Last, so the ring riding the cursor is over the rule and the fraction as well as the row.
	drawDraggedRing(gs, screen, &s.ringDrag, counters)
}

// updateRingRow runs the drag over the worn row. Called every tick from Update.
//
// **The row is live while a round resolves, unlike the hand** *(owner's call, 2026-08-26)*. The
// hand goes dead there because a queue being replayed is not a queue you may edit; the ring row has
// no such reason. A round is decided in full by `combat.ResolveRound` before a frame of it is
// drawn, so a reorder made while it plays back cannot reach it — it lands on the next one, which is
// exactly what the player is told by watching the row move.
//
// **The one thing it must not leave behind is a disagreement.** See moveRing.
func (s *CombatScene) updateRingRow(gs *state.GlobalState) {
	row := s.ringRow(gs)

	// A modal covering the screen, or a tutorial step holding input elsewhere, takes the row with
	// it — cancelling rather than returning, for the reason the action box cancels.
	if s.modalUp() || !gs.CursorAllowed() {
		s.ringDrag.cancel(row)
		return
	}

	s.ringDrag.update(gs, row)
}

// ringRow is this screen's worn row, addressed by the shared drag.
//
// **A click on a ring does nothing here.** The shop is where a ring is bought and sold; on the
// combat screen the row is a thing you read and now a thing you can reorder, and a click that did
// something would be a third meaning for the same press.
func (s *CombatScene) ringRow(gs *state.GlobalState) ringRow {
	return ringRow{
		rect: s.ringPaneRect(gs),
		worn: len(wornRings(gs)),
		move: func(from, to int) { s.moveRing(gs, from, to) },
	}
}

// moveRing commits a reorder to every copy of the row that exists.
//
// **There are three, and missing one is silent** *(2026-08-26)*. The run holds what is worn; the
// duelist in the fight holds their own copy with the accumulators this fight has grown; and, from
// DUEL! until the round finishes playing back, `fighterAfter` holds the duelist the round is going
// to end as — which `endOfRound` assigns over the live one. Moving only the first two would look
// right for the rest of the round and then snap back the moment it ended.
//
// **What it deliberately does not do is re-Equip.** `Session.Equip` adds a ring's stats for the
// fight, so putting the row through it again would pay every `add-hp` and `add-dmg` a second time.
// A reorder is a permutation and nothing else; `MoveRing` moves the accumulators with their rings.
//
// **The round already resolved is not touched**, which is the whole rule this feature is under: the
// blow being played back was decided against the order the row was in when DUEL! was pressed.
func (s *CombatScene) moveRing(gs *state.GlobalState, from, to int) {
	if !moveWornRing(gs, from, to) {
		return
	}

	s.fighter.Duelist = s.fighter.Duelist.MoveRing(from, to)
	s.fighterAfter = s.fighterAfter.MoveRing(from, to)

	trace.Logf("rings", "reordered %d -> %d, worn %v", from, to, gs.Run.Worn())
}
