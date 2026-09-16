package screens

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// The shield count on the duelist card, kept in step with playback rather than with the model.
//
// **It is `shownLife`'s pattern, and the file it replaced held the same pattern for banked action
// points** *(2026-08-31)*. `combat.Duelist.Shields` is not written until the round's end state is
// adopted, so a pip row reading the model would fill up a whole opposing turn after the card that
// filled it and empty a whole turn after the attack that ate it. The engine's arithmetic is
// untouched; what this adds is the same figure arriving on the beat it is drawn.
//
// **It cannot change an outcome.** The round was decided before a frame of it was drawn, and this
// is a view over what the log already says.
//
// **The pips fly, as of 2026-09-02** *(owner's call)*, and they fly on the beat the card that
// raises them is *scored* rather than on the beat the engine raises them. A defend card joins a
// hand like any other card and pays a 0 into the sum; that 0 flying out of it used to be the whole
// of what the card appeared to do, with the shield turning up several beats later in the defend
// phase, by which time the card that bought it had stopped being the thing on screen. So the
// shields leave the card with its figure: **what a card creates shows while the card is being
// scored.**
//
// **The flight predicts and the announcement corrects.** `noteShields` writes the standing count
// absolutely off `KindRaised`, so the raise arriving later either agrees with what is already drawn
// or fixes it — there is no double count to guard against, and a mispredicted pip lives for a few
// beats rather than for the round. The cap is `maxShieldPips`, because a prediction has to know
// the ceiling the raise will be held to.
//
// **It cannot change an outcome**, like everything else on this screen that moves.

// shieldFlight is a defend card's pips on their way to the card that will carry them.
//
// **It carries a count rather than one pip per flight.** Block raises two and Guard three, and
// three objects crossing the same gap on the same beat would read as three cards having been
// played rather than as one card worth three.
type shieldFlight struct {
	// side is whose card the pips are landing on, and seat is the hand seat they set off from.
	side combat.Side
	seat int

	// count is how many pips arrive, already capped.
	count int

	// element is which shield drawing the pips are: their card's own, so the pip that crosses the
	// screen is the mark that was sitting in that card's corner. **It is the element rather than a
	// color** *(2026-09-16)*, since the marks are authored per element now and there is nothing
	// left to tint.
	element cards.Element

	// ink is the color the pips are drawn in: their card's element, the same color that card
	// wears round its border and on its own corner mark. **Cosmetic** *(owner's call,
	// 2026-09-02)* — nothing about a shield depends on which element raised it, and a fire ward and
	// an ice ward stop the same attack. What it buys is that the thing crossing the screen looks
	// like the card it came off.
	ink color.RGBA

	// standing is the count the row shows once the pips land, or -1 for a flight that adds its
	// own count to whatever is there.
	//
	// **Both cases exist because the pips can set off before or after the engine has spoken.** A
	// defend card scored into a hand flies several beats before the raise is announced, so there
	// is no authoritative figure yet and the flight adds; a raise announced with nothing having
	// flown carries `KindRaised.Life`, which is the count itself, and the row takes it outright.
	standing int

	// landed says the arrival has been paid into the shown count, so a flight still on screen
	// during its hold cannot pay twice.
	landed bool

	t travel
}

func (f *shieldFlight) tick()        { f.t.tick() }
func (f shieldFlight) done() bool    { return f.t.done() }
func (f shieldFlight) arrived() bool { return f.t.age >= shieldFlyTicks() }

// The flight's clock, in the game's own beats so it slows down and speeds up with the round.
// **It is the hand dialog's term beat**, because it sets off on that beat and should land inside
// it: a shield still crossing the screen while the next figure flies out of the next card would
// read as belonging to that one.
func shieldFlyTicks() int  { return beat(22, 25) }
func shieldHoldTicks() int { return beat(4, 25) }

// row is one side's shield row, and nil-safe for a side outside the two.
func (s *CombatScene) row(side combat.Side) *shieldRow {
	if side < 0 || int(side) >= len(s.theater.shieldRows) {
		return &shieldRow{}
	}
	return &s.theater.shieldRows[side]
}

// noteShieldRaise flies the pips for an announced raise, and it is **the only place pips fly from**
// as of 2026-09-15 *(owner's call)*.
//
// The defend phase used to run last, so a defense scored into a hand paid a visible 0 into the sum
// and then did the thing it was for several beats later, on a card the player had stopped watching
// — which is why the pips were made to leave with the figure instead, and why this function existed
// only for the turn that formed no hand at all. The phases flipped (see combat.Categories), so the
// raise now has a beat of its own in front of the sum: the card lifts, the pips fly, the next
// defense goes up, and only then do the attacks score. The sum's copy of this is gone.
//
// The card that raised them is the one lit right now, which is the card the announcement is about.
// It reports whether it flew them, so the log can tell a raise it drew from one it did not.
//
// **The count is clamped to the row rather than to the rules.** A duelist holds as many shields as
// the turn paid for and the row draws maxShieldPips of them, so a raise past the row's end flies
// nothing rather than flying a pip with nowhere to land — see shownShields, and CLAUDE.md on the
// row being a separate number from the engine's.
func (s *CombatScene) noteShieldRaise(e combat.Event) bool {
	if e.Kind != combat.KindRaised || e.Amount <= 0 {
		return false
	}
	seat, ok := s.firingSeat(e.Side)
	if !ok || s.row(e.Side).flew(seat) {
		return false
	}

	// The raise names what is standing after its own card, so what this card put up is the
	// difference — and the row can only show maxShieldPips of it.
	count := e.Amount
	if room := maxShieldPips - (e.Life - e.Amount); count > room {
		count = room
	}
	if count <= 0 {
		return false
	}

	s.flyShields(shieldFlight{
		side: e.Side, seat: seat, count: count, standing: e.Life,
		element: s.handCardElement(e.Side, seat),
		ink:     s.handCardInk(e.Side, seat),
	})
	return true
}

// firingSeat is the seat of the card lit on one side right now, and false for none. **The last of
// them**, because a defense is lit alone and an attack phase lights a set the hand then narrows.
func (s *CombatScene) firingSeat(side combat.Side) (int, bool) {
	seats := s.theater.firingSeats
	if side == combat.SideB {
		seats = s.theater.enemyFiringSeats
	}
	if len(seats) == 0 {
		return 0, false
	}
	return seats[len(seats)-1], true
}

// flyShields raises one flight and records the seat it left, so nothing sends the same card's pips
// twice.
func (s *CombatScene) flyShields(f shieldFlight) {
	f.t = newTravel(0, shieldFlyTicks()+shieldHoldTicks())
	s.theater.shields = append(s.theater.shields, f)
	s.row(f.side).noteFlight(f.seat)
}

// landShields pays every arrived flight into the row, once each.
//
// **A predicted flight adds and an announced one sets.** A flight raised with a card's figure knows
// what its own card raised and nothing else, so it appends; a raise that flew its own pips carries
// the count itself and the row takes it. A wrong guess is corrected by the next announcement rather
// than compounded.
func (s *CombatScene) landShields() {
	for i := range s.theater.shields {
		f := &s.theater.shields[i]
		if f.landed || !f.arrived() {
			continue
		}
		f.landed = true

		row := s.row(f.side)
		row.fitTo(s.modelShields(f.side))
		if f.standing < 0 {
			row.add(f.element, f.count)
			continue
		}
		// **The announcement brings a count; the flight brings the color.** Every pip this
		// flight is responsible for wears its card's element — all `count` of them, not just the
		// newest, or a brace announcing two would land one colored pip and one bare white mark.
		row.raiseTo(f.standing, f.element)
		for i := 0; i < f.count && i < row.count(); i++ {
			row.pips[row.count()-1-i] = f.element
		}
	}
}

// shownShieldElements is the element of each standing pip, so the card can draw the right shield —
// and its length is the count, which is the whole point of the row being one list.
func (s *CombatScene) shownShieldElements(side combat.Side) []cards.Element {
	return s.row(side).pips
}

// modelShields is what the engine has standing for a side right now, which is what the row falls
// back to before any event this round has spoken.
func (s *CombatScene) modelShields(side combat.Side) int {
	c := s.fighter
	if side == combat.SideB {
		c = s.enemy
	}
	if c == nil {
		return 0
	}
	return c.Duelist.Shields
}

// shieldsRaisedBy is how many pips the card in a table seat will raise, and 0 for anything that is
// not a shield card.
//
// **It reads the card on the table, not the event.** A hand event names its terms as seats, and the
// seat is what the sum's figure is flying out of — so asking the same seat is what keeps the pips
// and the figure belonging to one card.
func (s *CombatScene) shieldsRaisedBy(side combat.Side, seat int) int {
	var card combat.Card
	switch {
	case side == combat.SideB:
		if seat < 0 || seat >= len(s.theater.enemyDealt) {
			return 0
		}
		card = s.theater.enemyDealt[seat].card
	default:
		if seat < 0 || seat >= len(s.theater.resolved) {
			return 0
		}
		card = s.theater.resolved[seat].card
	}
	if combat.ConceptOf(card.Concept).Verb != combat.VerbShield {
		return 0
	}
	return card.Amount()
}

// noteShields records what an announced shield event leaves standing.
//
// **The three kinds carry their figure differently, and an expiry is the odd one.** A raise says
// how many it added in `Amount` and what is standing in `Life`, the same split `KindDamage` makes
// between the blow and the life left; a block's `Amount` is what is *left* after it ate one; an
// expiry's is what lapsed, read before the clear — so the row it leaves behind is always empty.
//
// **A raise may only raise and a block may lower**, which is the row's one rule about who is
// allowed to take a shield away. See shieldRow.raiseTo.
func (s *CombatScene) noteShields(e combat.Event) {
	switch e.Kind {
	case combat.KindRaised:
		// The card being announced is the one lit right now, so its element is the shield any pip
		// this raise adds should be wearing.
		el := cards.Basic
		if seat, ok := s.firingSeat(e.Side); ok {
			el = s.handCardElement(e.Side, seat)
		}
		s.row(e.Side).raiseTo(e.Life, el)
	case combat.KindBlocked:
		s.row(e.Target).hold(e.Amount, cards.Basic)
	case combat.KindExpired:
		// **An expiry empties the row whatever it says.** Its `Amount` is the count read *before*
		// the shields were cleared — how many lapsed, not how many are left — so a row taking it
		// the way it takes a block's would keep drawing every shield that had just gone.
		s.row(e.Target).hold(0, cards.Basic)
	}
}

// shownShields is how many shields to draw on a side's card: what playback has reached if anything
// this round has said so, and the adopted model otherwise.
func (s *CombatScene) shownShields(side combat.Side, model int) int {
	row := s.row(side)
	row.fitTo(model)
	return row.count()
}

// The pips' journey, drawn.
//
// **They are the card's own mark, not a new picture.** The pip is the same shield drawing the
// corner of every defend card carries, in the same element, so what leaves the card, what crosses
// the screen and what lands are one drawing seen three times.
const (
	// shieldFromScale and shieldToScale are how big a pip is at each end. It **shrinks into the
	// card**, the damage figure's grammar: a thing flying into something goes away into it.
	shieldFromScale = 1.6
	shieldToScale   = 1.0

	// shieldSpread is how far apart two pips of one flight ride, in pixels at full size, so a
	// Block reads as two things rather than as one thicker one.
	shieldSpread = 26
)

// drawShields draws every pip in the air.
func (s *CombatScene) drawShields(gs *state.GlobalState, screen *ebiten.Image) {
	for _, f := range s.theater.shields {
		from, ok := s.shieldOrigin(gs, f)
		if !ok {
			continue
		}
		to := s.shieldTarget(gs, f)

		p := easeOut(clamp01(float64(f.t.age) / float64(shieldFlyTicks())))
		scale := shieldFromScale + (shieldToScale-shieldFromScale)*p
		alpha := shieldAlpha(f)

		for i := 0; i < f.count; i++ {
			// The pips of one flight fan out around the line they travel, closing as they land —
			// so two arrive as two and stack as the row they are joining.
			offset := (float64(i) - float64(f.count-1)/2) * shieldSpread * (1 - p)
			at := image.Pt(
				from.X+int(float64(to.X-from.X)*p+offset),
				from.Y+int(float64(to.Y-from.Y)*p),
			)
			drawShieldPip(screen, at, scale, alpha, f.element)
		}
	}
}

// shieldAlpha holds a pip solid for the flight and fades it over the hold, the damage figure's
// rule: what is left behind is the row it joined, not the thing that joined it.
func shieldAlpha(f shieldFlight) float32 {
	if !f.arrived() {
		return 1
	}
	held := float64(f.t.age-shieldFlyTicks()) / float64(shieldHoldTicks())
	return float32(clamp01(1 - held))
}

// drawShieldPip blits one mark, centered on a point, in its card's own element.
//
// **Nothing is tinted any more** *(2026-09-16)*. The pip used to be a near-white drawing multiplied
// by the card's element color, which was the only way to get five shields out of one picture; there
// are now five shields, authored, and the pip draws the one the card is showing. An element with no
// drawing takes the neutral shield rather than nothing, on cards.MarkArtKey's terms.
func drawShieldPip(screen *ebiten.Image, at image.Point, scale float64, alpha float32, e cards.Element) {
	img := systems.ArtMarkImage(shieldPipKey(e), shieldPipSize, shieldPipSize)
	if img == nil {
		return
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(at.X), float64(at.Y))
	op.ColorScale.ScaleAlpha(alpha)
	screen.DrawImage(img, op)
}

// shieldPipSize is the pip's drawn size, which is the card's own form box — the pip is that mark
// leaving that corner, so a second figure here would be the two drifting apart.
const shieldPipSize = 32

// shieldPipKey is the shield drawing for one element, neutral for anything that is not one of the
// five. Spelled here rather than exported from internal/cards because the key is a fact about the
// asset family and both packages build it the same way off a form and an element.
func shieldPipKey(e cards.Element) string {
	switch e {
	case cards.Fire, cards.Ice, cards.Lightning, cards.Earth, cards.Arcane:
		return "formdefend-" + e.String()
	}
	return "formdefend-neutral"
}

// shieldOrigin is the seat the pips leave: the card being scored, exactly where its own figure
// sets off from.
func (s *CombatScene) shieldOrigin(gs *state.GlobalState, f shieldFlight) (image.Point, bool) {
	seats := len(s.theater.resolved)
	if f.side == combat.SideB {
		seats = len(s.theater.enemyDealt)
	}
	if seats == 0 {
		return image.Point{}, false
	}

	// **A seat the row no longer holds still flies, from the row's first card** *(2026-09-02)*.
	// It used to draw nothing at all, and `landShields` paid the pips in regardless — so the row
	// filled with a pip that had never crossed the screen, which is exactly the "some fly and some
	// do not" the flight exists to prevent. Something traveling from slightly the wrong card is a
	// far smaller lie than a pip appearing out of nothing.
	seat := f.seat
	if seat < 0 || seat >= seats {
		seat = 0
	}
	return s.handCardCenter(gs, f.side, seat), true
}

// shieldTarget is where the pips land: **the row along the bottom of the fighter card**, not the
// middle of it *(owner's call, 2026-09-02)*.
//
// **A pip is joining a row; a damage figure is hitting a card.** The two gestures were the same
// journey to the same point, which made a shield read as something being done *to* the duelist. It
// now lands in the seat it is about to occupy, so the arrival and the row filling are one event.
//
// The row's own offset comes from the card style — `EffectTop`, the band the enemy's status badges
// use and the pips share — so a card re-laid out moves the target with it.
func (s *CombatScene) shieldTarget(gs *state.GlobalState, f shieldFlight) image.Point {
	r, style := s.duelistCardRect(gs), cards.DuelistStyle
	if f.side == combat.SideB {
		r, style = s.enemyCardRect(gs), cards.EnemyStyle
	}
	return image.Pt(
		(r.Min.X+r.Max.X)/2,
		r.Min.Y+style.EffectTop+style.EffectSize/2,
	)
}
