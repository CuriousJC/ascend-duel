package screens

// The relic pane: what the player is wearing, drawn as cards.
//
// **It draws what the run is wearing** *(2026-08-17)*. The worn set lives on `session.Session` —
// which is what makes it survive a fight — and every rule a relic has is in `data/relics.json` in the
// `When` / `If` / `Then` grammar. This file is the row; it no longer decides anything. **The loop
// around it landed on 2026-08-21** — see shop.go, which draws the same relic cards on a shelf and is
// the only thing that puts one on or takes one off.
//
// **It claims the band the full-height panes vacated.** Action Flow is built and not drawn,
// and Resolution left for the three-line feed above the hand on 2026-08-11, so 12–46% was
// empty. That is what paid for full-size relic cards; the alternative was a row of chips at deck
// -stack size, which would have made the relic a different object from every other card in the
// game.

import (
	"fmt"
	"image"
	"log"

	"math"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// maxRelics is the widest relic row this screen can ever be asked to draw.
//
// **It is the array's width, not the cap** *(2026-09-11)*. The two were one number until the cap
// became something a run carries — see combat.DefaultRelicSlots — and this is deliberately the
// larger of them: the row is a layout, and a layout has to survive the most relics a duelist could
// ever be holding rather than the most a shipped run is allowed to buy.
//
// **What the player is told is `relicSlots`**, which reads the run. The row saying `worn/5` while
// the duelist held a sixth is the drift this file has always been guarding against, and a constant
// can no longer say it.
const maxRelics = combat.MaxWornRelics

// relicSlots is how many relics this run may wear, which is what the fraction on the pane counts
// against and what the row is willing to draw.
//
// **The run is the authority, exactly as it is for which relics are worn.** A screen that kept its
// own number would be the one place in the game that disagrees with the shop about whether a sixth
// relic is allowed on.
func relicSlots(gs *state.GlobalState) int {
	if gs.Run == nil {
		return combat.DefaultRelicSlots
	}
	return gs.Run.RelicSlots()
}

const (
	// The gap on either side of the row: it starts where the duelist card ends and stops
	// where the enemy card begins.
	//
	// **Both edges are read off the cards themselves** *(2026-08-12)*. The right edge was a
	// hardcoded 79%, chosen to clear an enemy card centred at 88% — a percentage standing in
	// for a position it could not see, and one that would have quietly overlapped the moment
	// either card moved. It moved the next day.
	relicPaneGap = 16

	// **The row sits ten pixels below the cards on either side of it** *(2026-08-12)*, where
	// it was flush with their tops for a day.
	//
	// Flush alignment and a backing panel do not both work: the backing's top edge would land
	// exactly on the two cards' top edges, and three things sharing one line reads as a single
	// wide object with two cards embedded in it — which is the "cards trapped in a panel"
	// failure the framed version was retired for, made worse by the frame now being wider than
	// the row. Dropping the row breaks that line, and the offset is what makes the backing
	// legible as a thing *behind* the relics rather than a border *around* everything.
	relicPaneTopDrop = 10

	// The cap fraction under the pane's bottom-right corner, sized and spaced like the deck
	// pile's count, which is the same idea: a number saying how much of a fixed thing is in use.
	relicCountSize   = 22
	relicCountTopGap = 6

	// relicPaneBackPad is how far the backing extends past the row on every side. The pitch
	// puts the first card flush left and the last flush right, so with no padding the two end
	// cards would sit on the backing's edge and it would read as a border drawn around them
	// rather than as a surface they stand on.
	//
	// **It has to stay under relicPaneGap**, or the backing runs into the fighter card beside
	// it — 8 against 16 leaves half the gap still showing on each side.
	relicPaneBackPad = 8
)

// relicPaneBackColor is the surface the relics stand on: one step off `screenGround`, and
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
// **The step goes down now that the ground is light** *(2026-08-14)*. It was one step
// *lighter* than {50,50,50} for as long as the screen was dark, and kept that way it would be
// a near-white slab — the loudest thing in the band, which is the exact failure this pane was
// cut back to a fill to avoid. Which direction "one step" means is a function of the ground.
//
// **It is derived rather than written down** *(2026-09-07)*. It was a hand-picked tan for as long
// as the ground was one, and a hand-picked colour that is supposed to be "one step off the
// background" is a colour that silently stops being that the moment the background moves — which
// is exactly what the ground going blue would have done to it. Nine percent is what the tan
// actually was, kept so the pane reads as it always did.
var relicPaneBackColor = systems.ColorAtStrength(screenGround, 91)

// relicPaneRect is the row's extent: the cards' own band, running between the two corner cards
// and dropped relicPaneTopDrop below them.
//
// **It is the middle of a three-part row** — duelist card, relics, enemy card — so it takes its
// edges from its neighbours rather than from percentages of the screen. Whichever card moves,
// the row follows, and the one thing that cannot happen is a relic drawn underneath one of them.
//
// **The rectangle is the cards and the rule, not the backing.** It is what the slots are cut
// out of and what the rule and the fraction hang off; the backing is derived from it — see
// relicPaneBackRect — so growing the padding cannot silently move a relic.
//
// **It is the left half of a two-pane row since 2026-09-06** *(owner's call)*. The consumables pane
// takes a fixed two seats off the right-hand end and the relics take what is left — see topRowPanes,
// and consumablePaneWidth for what that costs a full row of five.
func (s *CombatScene) relicPaneRect(gs *state.GlobalState) image.Rectangle {
	relics, _ := s.topRowPanes(gs)
	return relics
}

// consumablePaneRect is the right half: the parasites the run is carrying. See consumables.go.
func (s *CombatScene) consumablePaneRect(gs *state.GlobalState) image.Rectangle {
	_, consumables := s.topRowPanes(gs)
	return consumables
}

// topRowPanes is the split, over the span between the two fighter cards.
//
// **The row is exactly a card deep** *(2026-09-04)*. It used to reserve a gap under itself for a
// rule with the worn count beneath it; the rule is gone and the count hangs off the backing's
// corner, so the pane ends where the cards do. That is 44 pixels the top band gives back, which is
// what let the card grow to its present height — see Hand in internal/cards/style.go.
func (s *CombatScene) topRowPanes(gs *state.GlobalState) (relics, consumables image.Rectangle) {
	left, right := relicRowSpan(gs)
	return topRowPanes(left, right, duelistCardRect(gs).Min.Y+relicPaneTopDrop)
}

// relicRowSpan is the horizontal extent of the relic row: where it starts after the duelist card
// and its caption column, and where it stops short of the enemy card.
//
// **It is a function of its own because the hand row is measured against it** *(2026-09-04,
// owner's call)*. The band the cards are dealt into used to run between two percentages of the
// screen and was therefore wider than anything above it; aligning it to this span is what makes
// the screen read as one column, at the cost of the hand overlapping itself a little more. See
// cardBandWidth.
//
// **It starts at the duelist card's right edge again** *(2026-09-04, owner's call)*. The floor
// and the room stood in a 166-pixel column here for a day; they are back under the card, and the
// row — and therefore the hand — got the width back. See towerPlaceRect.
func relicRowSpan(gs *state.GlobalState) (left, right int) {
	return duelistCardRect(gs).Max.X + relicPaneGap, enemyCardRect(gs).Min.X - relicPaneGap
}

// relicPaneBackRect is the surface drawn behind the row: the row padded on every side, and
// **deep enough to hold the rule and the fraction under it**.
//
// The fraction hanging off the bottom edge onto the bare ground would say the rule is the
// panel's floor and the number is loose underneath it, which is backwards — the count belongs
// to the row it counts.
func (s *CombatScene) relicPaneBackRect(gs *state.GlobalState) image.Rectangle {
	return relicPaneBackOf(s.relicPaneRect(gs))
}

// relicPaneBackOf is the backing for any relic row, whichever screen laid it out.
//
// **Simply the row padded, since 2026-09-04.** It used to be extended to cover the rule and the
// count, which were under the row; those are in the caption column now.
//
// **A free function since 2026-09-06**, so the build band draws the same pane the fight does — it
// drew bare cards on the shop and the reward screen, which is two screens showing the run's relics
// as something other than what the fight shows.
func relicPaneBackOf(row image.Rectangle) image.Rectangle {
	return row.Inset(-relicPaneBackPad)
}

// drawRelicPaneBack paints that surface. Flat, one step off the ground, no border and no title — see
// drawRelicPane, which is where the argument for all three is written down.
func drawRelicPaneBack(screen *ebiten.Image, row image.Rectangle) {
	back := relicPaneBackOf(row)
	vector.DrawFilledRect(screen,
		float32(back.Min.X), float32(back.Min.Y), float32(back.Dx()), float32(back.Dy()),
		relicPaneBackColor, false)
}

// drawPaneCount writes a pane's fraction on its bottom-right corner: `3/5 relics`, `1/2 held`.
//
// **One function for every pane that has one**, so the two on the top row and the two on the band
// cannot come to different conclusions about where a corner is or what size the figure is.
func drawPaneCount(gs *state.GlobalState, screen *ebiten.Image, row image.Rectangle, msg string) {
	back := relicPaneBackOf(row)
	top := back.Max.Y + relicCountTopGap

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(back.Max.X), float64(top))
	op.PrimaryAlign = text.AlignEnd
	op.ColorScale.ScaleWithColor(groundInk)
	text.Draw(screen, msg, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: relicCountSize}, op)
}

// relicSlotMaxGap is the most bare table ever left between two relic cards.
//
// **A row's pitch is capped and then the row is centred** *(2026-08-24)*. Without the cap the row
// spread to whatever pane it was handed, so a run wearing two relics put one against the duelist
// card and the other in the far corner of an empty screen — two relics reading as two unrelated
// things rather than as one build. The cap is what makes a growing row *grow*: relics sit at a
// fixed pitch and the row widens outwards from the middle as one is added, up to the point where
// five of them fill the pane and the pitch has to close up again.
//
// **26 since 2026-09-04**, from 22 — the same seven sixths the card grew by, so the row keeps its
// rhythm rather than closing up around bigger cards.
//
// **It stopped being the gap five relics leave in the pane on the same day.** That was true while
// the screen was 1280 wide: five relics filled the combat pane exactly, and the cap was read off
// them. At 1920 the pane is 1443 pixels and a full row is 1039, so there is slack even at five and
// the row is centred in it. Filling the pane again would mean a gap of 118 — most of a card of bare
// table between relics, which is precisely the "two unrelated things rather than one build" failure
// the cap was written to prevent. So the cap is now a chosen pitch rather than a derived one.
const relicSlotMaxGap = 26

// relicSlotPitch is how far apart two relic cards start, **for the number actually worn**.
//
// **The row spreads to fill the pane and closes up as it fills** *(2026-08-11)*. Three relics
// stand well apart and fully visible; five sit shoulder to shoulder with a few pixels between
// them, since 5 x 162 is 810 against roughly 825 of pane. **It overlapped by 26 pixels each
// until the cards came down a tenth in width later the same day**, and it would again the
// moment the pane narrowed or a sixth slot was ever allowed — the pitch is derived, so the row
// closes up by itself rather than by anyone redoing this arithmetic. Overlapping is the
// accepted failure mode, not shrinking: a card cannot be scaled, a smaller relic is a
// *different drawing*, and there is no relic style below this one.
//
// **The spread is capped at relicSlotMaxGap and the row is centred in the pane by relicSlotAt**, so
// a pane wider than the relics in it leaves its slack at both ends rather than between the cards.
func relicSlotPitch(r image.Rectangle, worn int) int {
	if worn < 2 {
		return 0
	}
	pitch := (r.Dx() - cards.RelicStyle.Width) / (worn - 1)
	if max := cards.RelicStyle.Width + relicSlotMaxGap; pitch > max {
		return max
	}
	return pitch
}

// relicSlotRowWidth is how much of the pane the row actually occupies: every pitch but the last,
// plus the card that sits on the final one.
func relicSlotRowWidth(r image.Rectangle, worn int) int {
	if worn < 1 {
		return 0
	}
	return (worn-1)*relicSlotPitch(r, worn) + cards.RelicStyle.Width
}

// relicSlotAt is where the i'th relic card's top-left corner sits. **Flush with the pane's top**,
// which is the top of the character block beside it — the two are aligned directly rather than
// each being inset inside a frame of its own.
//
// **Horizontally the row is centred on the pane**, which is only visible once the pitch is capped:
// a full row still starts where it always did, because there is no slack left to share out.
func relicSlotAt(r image.Rectangle, i, worn int) image.Point {
	left := r.Min.X + (r.Dx()-relicSlotRowWidth(r, worn))/2
	return image.Pt(left+i*relicSlotPitch(r, worn), r.Min.Y)
}

// wornRelics is what the player is wearing, as records, in worn order.
//
// **The run is the authority and this is the lookup** *(2026-08-17)*. `session.Session` holds the
// worn keys — in worn order, which is a rule, since relics fire left to right — and `gs.Relics` holds
// the record each key names, for its art and its name. The screen decides nothing.
//
// **A worn key with no record is reported, not ignored.** It is the failure `ParseElement` refuses to
// fall back on: a relic that quietly does not draw looks exactly like a relic that was never bought.
func wornRelics(gs *state.GlobalState) []data.RelicData {
	if gs.Run == nil {
		return nil
	}

	slots := relicSlots(gs)
	out := make([]data.RelicData, 0, slots)
	for _, key := range gs.Run.Worn() {
		if len(out) == slots {
			break
		}
		record, ok := gs.Relics[key]
		if !ok {
			log.Printf("the run is wearing %q, which is in no record", key)
			continue
		}
		out = append(out, record)
	}
	return out
}

// relicRow is the worn row as a draggable row of cards. **The lifecycle is carddrag.go's**; this is
// the half that knows the row's geometry and what a drop means.
//
// **Worn order is the order relics fire in**, so a drop here changes what a duel does — this is the
// one draggable row in the game where the gesture is a rule and not an arrangement. See
// Session.MoveRelic.
//
// **Nothing is lifted out of anything.** The run is the authority on what is worn and it is not
// touched until the drop, so `rowLift` does nothing at all and the drawing skips the seat the drag
// says is empty. The hand's row does remove its card, because there the list *is* the hand — see
// handRow, and dragRow for why the two are allowed to differ.
//
// **Every screen that shows the row builds one of these**, with its own rectangle and its own idea
// of what a click means: the combat screen's click does nothing, the shop's arms a sale.
type relicRow struct {
	rect  image.Rectangle
	worn  int
	click func(i int)
	move  func(from, to int)
}

func (r relicRow) rowLen() int { return r.worn }

func (r relicRow) rowSlot(gs *state.GlobalState, i int) image.Rectangle {
	at := relicSlotAt(r.rect, i, r.worn)
	return image.Rect(at.X, at.Y, at.X+cards.RelicStyle.Width, at.Y+cards.RelicStyle.Height)
}

// rowZone is the row's own rectangle. **Tighter than the hand's band**, deliberately: the hand
// stands alone at the bottom of the screen with nothing beside it, where this row has a duelist
// card at one end and an enemy card or a margin at the other. A zone spanning the width would make
// a drop on the duelist card a reorder.
func (r relicRow) rowZone(gs *state.GlobalState) image.Rectangle { return r.rect }

// rowDropIndex is which seat the cursor is over, measured in pitches from the row's left edge and
// from the middle of a step rather than its edge — the hand's arithmetic, over the relic row's
// pitch, because once five relics are worn these overlap too.
//
// **Clamped to a seat that exists**, unlike the hand's, which may land one past the end: nothing is
// being inserted here. Five relics reordered are still five relics.
func (r relicRow) rowDropIndex(gs *state.GlobalState) int {
	if r.worn < 2 {
		return 0
	}

	pitch := relicSlotPitch(r.rect, r.worn)
	idx := (gs.MouseX - relicSlotAt(r.rect, 0, r.worn).X + pitch/2) / pitch
	if idx < 0 {
		idx = 0
	}
	if idx > r.worn-1 {
		idx = r.worn - 1
	}
	return idx
}

// rowLift is deliberately empty. See the type comment.
func (r relicRow) rowLift(int) {}

func (r relicRow) rowReturn(from, to int) {
	if r.move != nil {
		r.move(from, to)
	}
}

func (r relicRow) rowClick(i int) {
	if r.click != nil {
		r.click(i)
	}
}

// moveWornRelic is the run's half of a reorder, and the half every screen shares. A screen holding a
// live duelist has a second half — see CombatScene.moveRelic.
func moveWornRelic(gs *state.GlobalState, from, to int) bool {
	if gs.Run == nil {
		return false
	}
	return gs.Run.MoveRelic(from, to)
}

// drawDraggedRelic draws the relic riding the cursor, over everything else on the row.
//
// **It is drawn from the run rather than from anything the drag is carrying**, which is what the
// empty rowLift buys: there is only ever one copy of what is worn, so a card in flight cannot
// disagree with the row it came out of.
func drawDraggedRelic(gs *state.GlobalState, screen *ebiten.Image, drag *cardDrag,
	counters map[string]string) {

	if !drag.dragging() {
		return
	}
	worn := wornRelics(gs)
	if drag.origin() >= len(worn) {
		return
	}

	record := worn[drag.origin()]
	drawRelicCard(gs, screen, drag.at(gs), record, counters[record.RelicRecord], true, true)
}

// relicCounter is one worn relic's accumulator, formatted for the badge in the corner of its card.
//
// **The figure is what the relic is doing, not how far it has counted** *(owner's call,
// 2026-08-26)*. Enflamed's accumulator is 50 when the relic is doing 1.5x damage, and a badge
// reading `50` would be a number in units nothing on screen explains. `combat.GrowthEffect` is what
// resolves the one numeric effect the accumulator feeds and `combat.Scaling` says whether that
// figure is a percentage — so a multiplier reads as a multiplier and flat life reads as life.
//
// **A relic that does not grow has no badge**, which is most of the catalogue: an empty string draws
// nothing. That is the whole distinction the badge is for — a card carrying one is a card whose
// number is still moving.
func relicCounter(w combat.WornRelic) string {
	return combat.CounterLabel(w)
}

// relicCounters is a badge per worn relic, by record key.
//
// **Keyed by record and not by position**, exactly as the accumulator itself is: the row is about
// to become something the player can drag into a different order, and a badge indexed by seat would
// follow the finger rather than the relic.
func relicCounters(worn []combat.WornRelic) map[string]string {
	out := make(map[string]string, len(worn))
	for _, w := range worn {
		if c := relicCounter(w); c != "" {
			out[combat.RelicOf(w.Relic).Key] = c
		}
	}
	return out
}

// runCounters is the badges as the run holds them: what a relic has banked between fights.
//
// **The two callers are the screens with no duel on them** — the reward screen's build band and the
// shop. The combat screen reads the duelist instead, and mid-blow the sum: see countersNow.
func runCounters(gs *state.GlobalState) map[string]string {
	if gs.Run == nil {
		return nil
	}
	return relicCounters(gs.Run.WornRelics())
}

// countersNow is the badges as the *round being played back* has got to.
//
// **A growing relic steps between the terms of one blow as of 2026-08-26** *(owner's call)*, so the
// duelist the screen is holding is a round behind while the sum is being read: `endOfRound` adopts
// the resolved duelist only once playback is finished. The hand dialog carries the accumulator each
// term left behind, so the row can step its badges on the beat the figure lands — which is the whole
// point of moving the growth into the sum. The player watches the number that is about to price the
// next card go up.
//
// **It reads figures the resolver produced and computes none**, exactly like the sum itself. Off the
// dialog it falls back to the duelist, which is every frame outside a blow.
func (s *CombatScene) countersNow() map[string]string {
	worn := s.fighter.Duelist.WornRelics()
	if grown, ok := s.theatre.mathBox.growthNow(combat.SideA); ok {
		worn = withGrown(worn, grown)
	}
	return relicCounters(worn)
}

// withGrown is a worn set with the accumulators one beat of a sum reached, as a copy.
//
// **A copy, because the duelist it came from is the fight's own** — this is a picture of a number
// part way through a round, and writing it back would be presentation changing an outcome.
func withGrown(worn []combat.WornRelic, grown [combat.MaxWornRelics]int) []combat.WornRelic {
	out := make([]combat.WornRelic, len(worn))
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
// and the row of relics sits inert through the one moment it is earning its place.
//
// **Side to side rather than a jump** *(owner's call)*. A card that leaps reads as being *picked*,
// which is what the selected-card lift already means in the hand and what the played row's lift
// means on the table — two vertical vocabularies already spoken for. Sideways is unused and reads as
// a thing rattling as it fires.
var (
	// relicShakeTicks is how long one shake lasts. **Under a term's own flight**, because the figures
	// arrive one after another and a shake still running when the next one starts would smear the
	// beats together.
	relicShakeTicks = beat(3, 5)

	// relicShakeSwings is how many times the card crosses its own centre. Three reads as a rattle;
	// one reads as a nudge and five as a wobble.
	relicShakeSwings = 3.0

	// relicShakeWidth is how far it travels either side at the start. **It decays to nothing** over
	// the shake, so the card settles rather than stopping mid-swing.
	relicShakeWidth = 7
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
	return int(math.Sin(p*math.Pi*2*relicShakeSwings) * (1 - p) * float64(relicShakeWidth))
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
	for i := range s.relicShake {
		s.relicShake[i].tick()
	}
	for i := range s.cardShake {
		s.cardShake[i].tick()
	}

	at := s.theatre.mathBox.at
	if at == s.shakeItem {
		return
	}
	s.shakeItem = at

	relics, card, ok := s.theatre.mathBox.shaking(combat.SideA)
	if !ok {
		return
	}
	for seat, shaking := range relics {
		if shaking && seat < len(s.relicShake) {
			s.relicShake[seat] = newTravel(0, relicShakeTicks)
		}
	}
	if card > 0 {
		s.shakePlayedCard(card - 1)
	}
}

// shakePlayedCard starts one played card rattling, growing the row of clocks if the table is holding
// more cards than it has seen before.
//
// **A slice rather than a fixed array**, unlike the relics: a worn row is capped at five by a rule,
// and the number of cards on the table is capped by an action budget that a relic can make cheaper.
func (s *CombatScene) shakePlayedCard(seat int) {
	if seat < 0 {
		return
	}
	for len(s.cardShake) <= seat {
		s.cardShake = append(s.cardShake, travel{})
	}
	s.cardShake[seat] = newTravel(0, relicShakeTicks)
}

// playedCardShake is how far sideways the played card in one seat sits this frame.
func (s *CombatScene) playedCardShake(seat int) int {
	if seat < 0 || seat >= len(s.cardShake) {
		return 0
	}
	return shakeOffset(s.cardShake[seat])
}

// relicCardCentre is the middle of one worn seat's card, which is where that relic's multiplier sets
// off from on its way into the sum.
//
// **It reads the same two functions the row is drawn with** — `relicPaneRect` and `relicSlotAt` — so a
// figure cannot set off from a seat the card is not in. That is the rule every origin on this screen
// follows; see `handCardCentre`.
func (s *CombatScene) relicCardCentre(gs *state.GlobalState, seat int) image.Point {
	at := relicSlotAt(s.relicPaneRect(gs), seat, len(wornRelics(gs)))
	return image.Pt(at.X+cards.RelicStyle.Width/2, at.Y+cards.RelicStyle.Height/2)
}

// drawRelicPane draws the backing, the relics, a rule under them, and the cap as a fraction on
// its right end.
//
// **There is still no box** *(2026-08-12)*. The backing that arrived today is a fill and not a
// frame — one step lighter than the screen, no border, no title, no hue — which is a different
// thing from the pink panel this pane started as and was stripped of on 2026-08-11. What that
// stripping went too far on is legibility of the *edges*: with a fighter card at either end of
// a row spanning most of the screen, nothing said where the middle began. See
// relicPaneBackColor.
//
// **Empty slots are not drawn.** They were the first sketch and the fraction replaced them:
// five frames of which two are dashed outlines spends the loudest thing in the row on saying
// what you have *not* got, where `3/5` says the same thing on the end of the rule.
// MECHANICS.md's "the cap is never displayed — it surfaces when you try to buy a sixth" is the
// rule this softens, and softening it is the owner's call: the fraction is the deck pile's
// idea, where a count of a fixed total is read without being looked for.
func (s *CombatScene) drawRelicPane(gs *state.GlobalState, screen *ebiten.Image) {
	r := s.relicPaneRect(gs)

	// The surface first, so everything else stands on it.
	back := s.relicPaneBackRect(gs)

	// **Flat, where the deck panel and the fight log are bevelled** *(owner's call, 2026-08-24)*.
	// It was sunken for an afternoon, on the argument that the relic cards stand *in* it. What that
	// misses is what is standing there: five bevelled cards on a bevelled tray on a bevelled
	// screen is three depths in one corner, and the cards are the thing meant to be read. The two
	// panels that keep their bevel are overlays — they cover the game, so a lit edge is what says
	// they are in front of it. This backing covers nothing.
	vector.DrawFilledRect(screen,
		float32(back.Min.X), float32(back.Min.Y), float32(back.Dx()), float32(back.Dy()),
		relicPaneBackColor, false)

	worn := wornRelics(gs)
	counters := s.countersNow()
	for i, relic := range worn {
		// **The seat a dragged relic left is drawn empty rather than closed up**, which is the
		// hand's rule too: the row keeps its width and its pitch while a card is up, so nothing
		// slides sideways under the cursor mid-drag.
		if s.relicDrag.dragging() && i == s.relicDrag.origin() {
			continue
		}

		// **The shake and the toast go together**: the card rattles and its border lights, which is
		// what says the relic is working rather than merely moving.
		at := relicSlotAt(r, i, len(worn))
		shake := shakeOffset(s.relicShake[i])
		at.X += shake

		drawRelicCard(gs, screen, at, relic, counters[relic.RelicRecord], true, !s.relicShake[i].done())
	}

	// **The count hangs off the pane's bottom-right corner** *(2026-09-04, owner's call)*, and the
	// rule that used to run under the row is gone with the trip through the caption column: the
	// backing has an edge of its own now, so a second line saying where the row ends was drawing
	// the same fact twice. See relicCountRect.
	s.drawRelicCount(gs, screen, len(worn))

	// Last, so the relic riding the cursor is over the rule and the fraction as well as the row.
	drawDraggedRelic(gs, screen, &s.relicDrag, counters)
}

// relicCountRect is where the worn count stands: **hung off the bottom-right corner of the pane
// the relics stand on** *(2026-09-04, owner's call)*. It spent a day in the caption column beside
// the duelist card, which is where the floor and the room were; both have moved back under that
// card, and a count of the relics belongs against the relics.
//
// The rectangle spans the whole backing and the figure is drawn right-aligned in it, so the number
// sits on the corner however wide the pane gets.
func (s *CombatScene) relicCountRect(gs *state.GlobalState) image.Rectangle {
	back := s.relicPaneBackRect(gs)
	top := back.Max.Y + relicCountTopGap
	return image.Rect(back.Min.X, top, back.Max.X, top+relicCountSize)
}

// drawRelicCount writes `worn / cap` on the pane's bottom-right corner.
//
// **`worn / cap`, exactly like the pile's `left / owned`.** The numerator is what moves and the
// denominator deliberately never does, so the figure is read as "three of your five fingers are
// spoken for" rather than as two unrelated numbers.
//
// **The rule is drawn even with no relics equipped**, which is deliberate — a line with an empty
// figure under it says the row exists and is empty, where nothing at all says the screen forgot to
// draw something.
//
// **It is the bare fraction, and the noun went on 2026-09-11** *(owner's call)*. What the corner
// has to say is how many seats are spoken for; the pane under it is full of relic cards, so the
// word was the figure repeating what the cards it sits on already say. The same call took
// `parasites` off the consumables pane — see drawConsumableCount, which is this figure in the same
// seat at the same size and has to stay its twin.
//
// **The denominator is relicSlots, not maxRelics** *(bug, 2026-09-11)*. It drew `0/8` — the width
// of the duelist's relic array — from the day the two numbers were split, telling the player they
// had eight fingers when the cap is five. This is precisely the drift maxRelics' own doc comment
// says it is guarding against, and it went wrong in the one place that comment points at, so the
// warning is now a line of code: the run is the authority on how many relics may be worn, and the
// constant is only ever how wide the row is willing to draw.
func (s *CombatScene) drawRelicCount(gs *state.GlobalState, screen *ebiten.Image, worn int) {
	r := s.relicCountRect(gs)

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(r.Max.X), float64(r.Min.Y))
	op.PrimaryAlign = text.AlignEnd
	op.ColorScale.ScaleWithColor(groundInk)
	text.Draw(screen, fmt.Sprintf("%d/%d", worn, relicSlots(gs)),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: relicCountSize}, op)
}

// updateRelicRow runs the drag over the worn row. Called every tick from Update.
//
// **The row is live while a round resolves, unlike the hand** *(owner's call, 2026-08-26)*. The
// hand goes dead there because a queue being replayed is not a queue you may edit; the relic row has
// no such reason. A round is decided in full by `combat.ResolveRound` before a frame of it is
// drawn, so a reorder made while it plays back cannot reach it — it lands on the next one, which is
// exactly what the player is told by watching the row move.
//
// **The one thing it must not leave behind is a disagreement.** See moveRelic.
func (s *CombatScene) updateRelicRow(gs *state.GlobalState) {
	row := s.relicRow(gs)

	// A modal covering the screen, or a tutorial step holding input elsewhere, takes the row with
	// it — cancelling rather than returning, for the reason the action box cancels.
	if s.modalUp() || !gs.CursorAllowed() {
		s.relicDrag.cancel(row)
		return
	}

	s.relicDrag.update(gs, row)
}

// relicRow is this screen's worn row, addressed by the shared drag.
//
// **A click on a relic does nothing here.** The shop is where a relic is bought and sold; on the
// combat screen the row is a thing you read and now a thing you can reorder, and a click that did
// something would be a third meaning for the same press.
func (s *CombatScene) relicRow(gs *state.GlobalState) relicRow {
	return relicRow{
		rect: s.relicPaneRect(gs),
		worn: len(wornRelics(gs)),
		move: func(from, to int) { s.moveRelic(gs, from, to) },
	}
}

// moveRelic commits a reorder to every copy of the row that exists.
//
// **There are three, and missing one is silent** *(2026-08-26)*. The run holds what is worn; the
// duelist in the fight holds their own copy with the accumulators this fight has grown; and, from
// DUEL! until the round finishes playing back, `fighterAfter` holds the duelist the round is going
// to end as — which `endOfRound` assigns over the live one. Moving only the first two would look
// right for the rest of the round and then snap back the moment it ended.
//
// **What it deliberately does not do is re-Equip.** `Session.Equip` adds a relic's stats for the
// fight, so putting the row through it again would pay every `add-hp` and `add-dmg` a second time.
// A reorder is a permutation and nothing else; `MoveRelic` moves the accumulators with their relics.
//
// **The round already resolved is not touched**, which is the whole rule this feature is under: the
// blow being played back was decided against the order the row was in when DUEL! was pressed.
func (s *CombatScene) moveRelic(gs *state.GlobalState, from, to int) {
	if !moveWornRelic(gs, from, to) {
		return
	}

	s.fighter.Duelist = s.fighter.Duelist.MoveRelic(from, to)
	s.fighterAfter = s.fighterAfter.MoveRelic(from, to)

	trace.Logf("relics", "reordered %d -> %d, worn %v", from, to, gs.Run.Worn())
}
