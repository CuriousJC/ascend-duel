package screens

// Shields breaking the attacks they ate, between the player's turn and the creature's.
//
// **The engine already decided this before a frame of it was drawn.** `combat.shieldedSlots` picks
// which of a creature's blows the player's shields eat — the heaviest first — and it picks them at
// the top of the creature's turn rather than as each card arrives. That is what makes this
// drawable: the whole exchange is known at the boundary, so it can be *shown* at the boundary
// instead of dribbling out one skipped card at a time while the turn plays.
//
// # Why it is a beat of its own
//
// A shield used to be spent invisibly. The creature's card came up, nothing happened, and a line in
// the feed said a shield had eaten it — which is a rule the player paid for and never saw work.
// Worse, the pip left the row several beats after the card that would have hit them was already
// past. So this is the same argument the shield pips' own flight was built on: **what a thing does
// shows while it is being done**, and the doing is at the boundary.
//
// One beat *(owner's call, 2026-09-08)*: every blocked attack breaks at once, however many shields
// were spent. Three pips crossing in sequence would be three beats of pause before a creature that
// has not swung yet, and the thing being said — "these are the ones that will not land" — is one
// statement about the turn rather than three about cards.
//
// # Two halves, and the split is the point
//
// **The transition** is here: pips flying out of the duelist card into the attack cards they kill,
// then the break spreading across each face. It draws `cards.ShatterCracks` on the GPU because it
// changes every frame.
//
// **The final state** is `cards.MarkShattered`, baked into the card image and cached with it, and
// it is what the seat wears for the rest of the round. The handoff is exact: both halves draw the
// same geometry from the same seed, so the frame the animation ends on and the frame the baked
// mark starts on are the same picture. See internal/cards/mark.go.
//
// **It cannot change an outcome**, like everything else that moves on this screen. It holds the
// playback cursor — `combatTheatre.running` — which is pacing, and pacing is allowed.

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The break's clock, in the game's own beats so it slows down and speeds up with the round — see
// `beat` in clock.go, and the note there about every other clock on this screen being a fraction of
// the one speed.
var (
	// shatterFlyTicks is the pip crossing the table. Longer than a shield pip's own flight
	// (shieldFlyTicks) because this one crosses the *whole* table rather than travelling from a
	// card to the row beneath it, and a journey twice as far at the same speed reads as hurried.
	shatterFlyTicks = beat(30, 25)

	// shatterSpreadTicks is the break opening once the pip lands. Short: a window breaks, it does
	// not dissolve.
	shatterSpreadTicks = beat(14, 25)

	// shatterHoldTicks is the pause on the finished break before playback moves on, so the player
	// reads which cards died before the creature starts swinging with the ones that did not.
	//
	// **A whole beat and a half, which is long by this screen's standards and deliberately so**
	// *(owner's call, 2026-09-08)*. It was a third of a beat and the round played straight through
	// it: the break appeared and the creature was already swinging, so what the player saw was a
	// flicker rather than an exchange. The round has three moves in it — the duelist swings, the
	// shields break what they can reach, the creature swings with what is left — and the middle one
	// is the only one that had no beat of its own.
	//
	// It is the longest single hold on this screen and it is spent on the one thing here that is
	// *not* a card acting. Pacing, like every other clock in this file: it cannot change an
	// outcome.
	shatterHoldTicks = beat(38, 25)
)

// shieldBreak is one attack card being broken: a pip crossing to it, then the crack spreading over
// its face.
//
// **It carries the seat rather than the card.** The seat is what both halves need — where to fly
// to, and which row entry wears the mark afterwards — and the card at that seat is a lookup the
// row already answers.
type shieldBreak struct {
	// seat is the index into the opponent's table row.
	seat int

	// ink is the colour the pip is drawn in: the element of a shield the player raised, so the
	// thing crossing the table looks like what left the row. **Cosmetic**, exactly as the pips'
	// own colour is — a fire ward and an ice ward break the same attack.
	ink color.RGBA

	t travel
}

func (b *shieldBreak) tick()     { b.t.tick() }
func (b shieldBreak) done() bool { return b.t.done() }

// landed reports that the pip has arrived and the break has started opening.
func (b shieldBreak) landed() bool { return b.t.age >= shatterFlyTicks }

// spread is how far open the break is, 0 while the pip is still crossing and 1 once it is whole.
func (b shieldBreak) spread() float64 {
	if !b.landed() {
		return 0
	}
	return clamp01(float64(b.t.age-shatterFlyTicks) / float64(shatterSpreadTicks))
}

// stageShieldBreaks raises the whole exchange on the frame the creature's turn is about to start.
//
// **It reads the round it has already been handed rather than waiting for the events.** The blocks
// are `KindBlocked` events sitting further down `s.log`, and the screen is allowed to look at them
// because the round is resolved: this is a drawing of a decision, not a second decision. It is the
// one place on this screen that reads ahead of the cursor, and it is confined to this function so
// that stays true.
//
// **The pips leave the row here, several beats before `KindBlocked` says so.** That is the same
// predict-then-correct the raises use — the block's own `Amount` sets the row absolutely when it
// arrives, so a miscount lives for a few beats rather than for the round. See shield_row.go.
//
// It reports whether it staged anything, so the caller can leave the flag alone on a round with no
// blocks in it.
func (s *CombatScene) stageShieldBreaks(gs *state.GlobalState) bool {
	blocks := s.blocksAhead()
	if len(blocks) == 0 {
		return false
	}

	ink := s.brokenPipInk()
	for _, seat := range blocks {
		if seat < 0 || seat >= len(s.theatre.enemyDealt) {
			// A seat the row does not hold is dropped rather than flown to nowhere. It means the
			// engine's slot indices and this row have come apart — see blocksAhead, where the one
			// way that can happen is written down.
			continue
		}
		s.theatre.breaks = append(s.theatre.breaks, shieldBreak{
			seat: seat,
			ink:  ink,
			t:    newTravel(0, shatterFlyTicks+shatterSpreadTicks+shatterHoldTicks),
		})
	}
	if len(s.theatre.breaks) == 0 {
		return false
	}

	// The pips go now, with the things that are carrying them. The row cannot go below zero and
	// `hold` clamps, so a prediction that disagrees with the engine costs a few beats of a wrong
	// count rather than a broken row.
	row := s.row(combat.SideA)
	row.hold(row.count()-len(s.theatre.breaks), color.RGBA{})
	return true
}

// blocksAhead is which seats of the opponent's row a shield will eat this round, read off the
// resolved log.
//
// **The seat is `Event.Slot`, and it indexes the turn as it resolved.** That is the same convention
// `Event.HandCards` uses and the same one `noteHand` reads it under, so the two cannot drift apart.
// It also inherits the same known gap: a *chilled* card is taken off the front of the turn before
// the indices are handed out, while the table row still draws it — so a round in which ice takes a
// creature's card and a shield eats another would break the wrong seat. Ice is the only thing that
// can chill, no creature carries it today, and fixing it properly means the row learning what a
// chill did, which is a change to `noteResolved` rather than to this.
func (s *CombatScene) blocksAhead() []int {
	var out []int
	for i := s.cursor; i < len(s.log); i++ {
		e := s.log[i]
		if e.Kind == combat.KindRoundEnd {
			break
		}
		// The block's `Side` is whose shield was spent. The card that breaks belongs to the other
		// duelist, so a block on the player's side marks the opponent's row — which is the only
		// case the game produces, every creature being a solo attacker and none of them holding
		// shields. See combat.blockedByShield, which carries the same note.
		if e.Kind == combat.KindBlocked && e.Side == combat.SideA {
			out = append(out, e.Slot)
		}
	}
	return out
}

// brokenPipInk is the colour the crossing pips take: the newest shield in the player's row, which
// is the one most recently raised and the colour the player just watched land.
//
// **A row with no colour recorded hands back a zero**, which `drawShieldPip` reads as "as drawn" —
// the bare white mark. That is the same fallback the pips' own flight takes.
func (s *CombatScene) brokenPipInk() color.RGBA {
	pips := s.row(combat.SideA).pips
	if len(pips) == 0 {
		return color.RGBA{}
	}
	return pips[len(pips)-1]
}

// shattered reports whether a seat of the opponent's row is wearing a finished break.
//
// **A seat is shattered once its own transition has finished, not when it was staged.** The mark
// and the animation are two drawings of one thing, and both being up at once would double every
// line.
func (s *CombatScene) shattered(seat int) bool {
	return s.theatre.shatteredSeats[seat]
}

// advanceBreaks ticks every break and settles the finished ones into the persistent mark.
//
// **It is `advance` with the handoff inside the loop, and that is the whole reason it exists.** The
// generic one ticks a mover and drops it in the same pass, so a break has exactly one frame in
// which it is both finished and still in the list — and nothing outside this loop is ever standing
// in it. Noting them before the call saw nothing finished; noting them after saw nothing at all.
// Either way the mark never landed and the break vanished the instant it finished drawing.
//
// **The handoff is exact**: the frame the animation stops is the frame the baked mark starts, so
// there is no frame of a card with neither on it.
func (t *combatTheatre) advanceBreaks() []shieldBreak {
	live := t.breaks[:0]
	for i := range t.breaks {
		b := &t.breaks[i]
		b.tick()
		if !b.done() {
			live = append(live, *b)
			continue
		}
		if t.shatteredSeats == nil {
			t.shatteredSeats = map[int]bool{}
		}
		t.shatteredSeats[b.seat] = true
	}
	return live
}

// drawShieldBreaks draws every break in progress: the pip while it is crossing, and the crack
// opening once it has landed.
//
// **It is drawn after the opponent's row**, so the break sits on the card rather than under it.
func (s *CombatScene) drawShieldBreaks(gs *state.GlobalState, screen *ebiten.Image) {
	for _, b := range s.theatre.breaks {
		at, ok := s.breakSeatRect(gs, b.seat)
		if !ok {
			continue
		}
		if !b.landed() {
			s.drawBreakPip(gs, screen, b, at)
			continue
		}
		if b.done() {
			// Finished: the baked mark is drawing it now. Nothing here, or every line would be
			// drawn twice on the frame the two overlap.
			continue
		}
		drawSpreadingCracks(screen, at, s.theatre.enemyDealt[b.seat].card, b.spread())
	}
}

// drawBreakPip is the shield mark on its way across the table, shrinking as it goes.
//
// **It shrinks where the shields' own flight grows.** A pip joining the row comes toward the
// reader; this one is going away into a card, which is the difference the damage figure already
// draws between a term flying into a sum and a total flying into a health bar.
func (s *CombatScene) drawBreakPip(gs *state.GlobalState, screen *ebiten.Image,
	b shieldBreak, seat image.Rectangle) {

	from := s.shieldTarget(gs, shieldFlight{side: combat.SideA})
	to := image.Pt((seat.Min.X+seat.Max.X)/2, (seat.Min.Y+seat.Max.Y)/2)

	p := easeOut(clamp01(float64(b.t.age) / float64(shatterFlyTicks)))
	at := image.Pt(
		from.X+int(float64(to.X-from.X)*p),
		from.Y+int(float64(to.Y-from.Y)*p),
	)
	drawShieldPip(screen, at, breakPipScale(p), 1, b.ink)
}

// breakPipScale is the pip's size along its journey: full when it leaves the row, two thirds when
// it strikes.
func breakPipScale(p float64) float64 { return 1 - 0.34*p }

// drawSpreadingCracks strokes the card's own break geometry, as far as the spread has opened it.
//
// **The same lines the baked mark uses, from the same seed**, so nothing moves on the frame this
// hands over. Each crack has its own delay — the radials first, then the panes closing round them —
// and a crack part-way through its own window is drawn part-way along its length, so the break
// travels outward rather than fading in.
func drawSpreadingCracks(screen *ebiten.Image, at image.Rectangle, c combat.Card, spread float64) {
	w, h := at.Dx(), at.Dy()
	for _, crack := range cards.ShatterCracks(w, h, cards.MarkSeed(c.Label())) {
		grown := crackProgress(crack.Delay, spread)
		if grown <= 0 {
			continue
		}
		x0 := float32(at.Min.X + crack.From.X)
		y0 := float32(at.Min.Y + crack.From.Y)
		x1 := float32(at.Min.X) + float32(crack.From.X) + float32(crack.To.X-crack.From.X)*float32(grown)
		y1 := float32(at.Min.Y) + float32(crack.From.Y) + float32(crack.To.Y-crack.From.Y)*float32(grown)
		vector.StrokeLine(screen, x0, y0, x1, y1, float32(crack.Width), cards.CrackInk(), true)
	}
}

// crackProgress is how far along one crack has grown, given when it starts and how far the whole
// break has opened. **Each crack takes the same share of the window**, so a late one is as quick as
// an early one rather than being crushed into whatever is left.
func crackProgress(delay, spread float64) float64 {
	const window = 0.4
	if spread <= delay {
		return 0
	}
	return clamp01((spread - delay) / window)
}

// breakSeatRect is where one of the opponent's cards is drawn right now, as a rectangle.
//
// **It asks the row's own layout rather than recomputing a seat.** A card still flying in has not
// reached its seat, and a break drawn at the seat while the card was elsewhere would be a crack
// hanging in the air — so this reads the same `enemyCardAt` the row itself draws with.
func (s *CombatScene) breakSeatRect(gs *state.GlobalState, seat int) (image.Rectangle, bool) {
	if seat < 0 || seat >= len(s.theatre.enemyDealt) {
		return image.Rectangle{}, false
	}
	d := s.theatre.enemyDealt[seat]
	at := s.enemyCardAt(gs, d, seat, len(s.theatre.enemyDealt), s.enemySplit(),
		lit(s.theatre.enemyFiringSeats, seat))
	return image.Rect(at.X, at.Y, at.X+cardWidth, at.Y+cardHeight), true
}
