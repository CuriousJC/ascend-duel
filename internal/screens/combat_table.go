package screens

// The table: two hands of cards facing each other across the middle of the screen.
//
// **Added 2026-08-12**, and it is the first thing on this screen that shows the round as a
// confrontation rather than as a list. The player's played cards line up on the left, the
// opponent's queued cards on the right, both full size, both in the order they will resolve —
// a hand opposing a hand, the way two hands are laid out at a card table.
//
// It replaced a pile in the bottom-left corner. That pile was the round's history and it was
// legible, but it was history *only*: it grew in a corner with nothing opposite it, so the one
// thing a duel is about — my five cards against yours — was something the player had to
// assemble from a pane of sentences.
//
// **Presentation only, the same constraint the flights live under.** Nothing here is consulted
// by a rule, a predicate or a button's enabled state, and both rows are built from what
// `combat.ResolutionOrder` already decided.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// How far each row sits from its own edge of the screen, and how much clear air is kept
	// between the two rows when both are full.
	//
	// **The gap is what stops the two hands becoming one row of ten.** Five cards a side is
	// the cap today (combat.MaxActions), and five full-size cards do not fit in half a screen
	// without overlapping — so the rows overlap *within themselves* and never into each other.
	// The alternative was shrinking the cards, and there is no card style between Hand and
	// Mini; a smaller card is a different drawing, not a scaled one.
	tableInset = 24
	tableGap   = 48

	// tableGroupGap is the clear air a row leaves between its attacks and its plans.
	//
	// **The round has two phases and the row now says so** *(2026-08-15)*. Every card in a row
	// resolves in `combat.ResolutionOrder`, which puts a turn's attacks first and its plans
	// second — so the boundary is already there and was invisible, and a row of five cards read
	// as one undifferentiated sequence when it is really "this is what I swing, and then this is
	// what I do about theirs".
	//
	// It is deliberately smaller than `cardGap` doubled: enough to read as a break, not enough to
	// read as two separate rows. And it is spent out of the same half-width the cards have, so a
	// full row overlaps by a little more rather than running past the middle.
	tableGroupGap = 26

	// tableFireLift is how far the card currently resolving rises out of its row.
	//
	// **The one thing the row cannot say by itself.** A card used to fly to the middle of the
	// screen, hold there to be read, and then drop into the pile — which was the reading
	// moment, and it cannot survive here: the middle of the screen is now where the opponent's
	// hand is, so a card crossing it would pass over the cards it is being played against.
	// Lifting in place says the same thing in the space the row already owns, and it borrows
	// the gesture selection already uses in the hand — see selectedNudge.
	tableFireLift = 22
)

// tableRowTop is the y both rows sit at: as low as the band allows while still clearing the
// Resolution feed.
//
// **Measured off the feed's collapsed top, not the expanded one.** A row that jumped whenever
// the box was held would be a layout reacting to an input it has nothing to do with. An
// expanded feed reaches up past this and is drawn *under* the cards — see the draw-order
// ranking in combat.go, where that is the one thing deliberately given up.
//
// This is where `firingAt` used to put a single card, and the number is unchanged; what moved
// is that a whole row lives here now instead of one card passing through.
func tableRowTop(gs *state.GlobalState) int {
	return gs.PctY(handTopPct) - mathBandGapAboveCards - mathBandHeight - firingGap - cardHeight
}

// playerTableCentre is the middle of the player's half of the table, on the row's own centre
// line: **the space their played cards land in**, and empty for the whole of the planning phase
// that leads up to them landing there.
//
// It is where the planned hand is named — see drawPlannedHand — and it is derived from the row
// rather than written down, so the name follows the table if the table moves.
func playerTableCentre(gs *state.GlobalState) image.Point {
	return image.Pt(tableInset+tableHalfWidth(gs)/2, tableRowTop(gs)+cardHeight/2)
}

// tableHalfWidth is how much room one hand has: the screen less both insets and the gap in
// the middle, halved.
func tableHalfWidth(gs *state.GlobalState) int {
	return (gs.ScreenWidth - 2*tableInset - tableGap) / 2
}

// tablePitch is how far apart two cards in a row start.
//
// **Side by side until they do not fit, then overlapped — the same shape as handPitch**, and
// the derived form matters for the same reason: the cap on how many actions a round holds is
// a rule that a ring or a brand is expected to raise, so a pitch that only worked for five
// would put a rendering bug behind a balance change.
//
// **The group gap is taken out of the room before the pitch is worked out**, which is what makes
// the split cost overlap rather than width: a row that spent the gap on top of a full-width
// layout would run past the middle of the screen and into the other hand.
func tablePitch(gs *state.GlobalState, n, split int) int {
	full := cardWidth + cardGap
	if n < 2 {
		return full
	}
	room := tableHalfWidth(gs) - groupGapFor(n, split)
	if (n-1)*full+cardWidth <= room {
		return full
	}
	return (room - cardWidth) / (n - 1)
}

// groupGapFor is the extra space a row spends on its attack/plan break: one gap, or none if the
// row is all one kind.
//
// `split` is the index the plans start at, which `combat.ResolutionOrder` has already decided —
// see playedSplit and enemySplit. A split of 0 or of n means every card in the row is on the same
// side of the boundary, so there is nothing to separate.
func groupGapFor(n, split int) int {
	if split <= 0 || split >= n {
		return 0
	}
	return tableGroupGap
}

// groupShiftFor is how far the nth card is pushed along by the break: the whole gap once the row
// has crossed into its plans, nothing before that.
func groupShiftFor(n, split, i int) int {
	if i < split {
		return 0
	}
	return groupGapFor(n, split)
}

// playedSeatAt is where the nth of the player's played cards sits: **left-aligned**, the whole
// row placed at once from the screen's left inset.
//
// **The pitch is a function of the round's total, never of how many have landed so far.** Both
// hands are dealt to the table the moment DUEL! is pressed, so the total is known — and a pitch
// that grew with the row would shuffle every card already sitting there sideways each time
// another arrived, which reads as the layout twitching rather than as a card being played.
func playedSeatAt(gs *state.GlobalState, n, total, split int) image.Point {
	if total < 1 {
		total = 1
	}
	x := tableInset + n*tablePitch(gs, total, split) + groupShiftFor(total, split, n)
	return image.Pt(x, tableRowTop(gs))
}

// enemySeatAt is where the nth of the opponent's queued cards sits: **right-aligned**, the
// whole row placed at once so its last card is flush with the right inset.
//
// **The two rows are anchored differently on purpose.** The opponent's queue is known in full
// the moment the round starts, so it is laid out as a finished hand against its own edge; the
// player's is revealed a card at a time. Pinning both to the outside edges is what makes them
// face each other rather than drift toward the middle as they fill.
func enemySeatAt(gs *state.GlobalState, n, total, split int) image.Point {
	if total < 1 {
		total = 1
	}
	pitch := tablePitch(gs, total, split)
	width := (total-1)*pitch + cardWidth + groupGapFor(total, split)
	left := gs.ScreenWidth - tableInset - width

	return image.Pt(left+n*pitch+groupShiftFor(total, split, n), tableRowTop(gs))
}

// splitOf is where a row laid out in resolution order stops being attacks and starts being plans.
//
// **It is found by scanning rather than counted while the row is built**, because the row is
// built from `combat.ResolutionOrder` and that is the authority on the boundary — a screen
// keeping its own tally would be a second answer to the same question. It returns len(cards) for
// a row with no plans in it, which `groupGapFor` reads as "no break".
func splitOf(cards []combat.Card) int {
	for i, c := range cards {
		if c.Category() == combat.CategoryPlan {
			return i
		}
	}
	return len(cards)
}

// playedSplit and enemySplit are splitOf over the two rows' own card lists.
func (s *CombatScene) playedSplit() int {
	for i, r := range s.resolved {
		if r.card.Category() == combat.CategoryPlan {
			return i
		}
	}
	return len(s.resolved)
}

func (s *CombatScene) enemySplit() int {
	for i, d := range s.enemyDealt {
		if d.card.Category() == combat.CategoryPlan {
			return i
		}
	}
	return len(s.enemyDealt)
}

// lift raises a seat by tableFireLift, which is how either row says "this is the card
// resolving right now".
//
// **One function so the two sides cannot drift apart** *(2026-08-12)*. The lift arrived on the
// player's row alone and the opponent's cards sat still through their whole turn, which read as
// the enemy's hand being scenery rather than the other half of the round. The gesture has to be
// the same on both sides or it stops meaning "this one" and starts meaning "yours".
func lift(at image.Point, firing bool) image.Point {
	if firing {
		at.Y -= tableFireLift
	}
	return at
}

// enemyQueueOrder is the opponent's queued actions **in the order they will resolve**, which is
// not the order the planner put them in.
//
// `combat.ResolutionOrder` regroups a turn into prepare, then attacks, then defenses, and it is
// the single authority on that — so this asks it rather than sorting a copy, exactly as the
// Resolution pane and `ResolveRound` do. A row that showed the planner's order would be a
// picture of something that never happens.
func (s *CombatScene) enemyQueueOrder() []combat.Card {
	order := combat.ResolutionOrder(s.fighterActions, s.enemyActions)

	out := make([]combat.Card, 0, len(s.enemyActions))
	for _, slot := range order {
		if slot.Side != combat.SideB || slot.Index >= len(s.enemyActions) {
			continue
		}
		out = append(out, s.enemyActions[slot.Index])
	}
	return out
}

// dealtCard is one of the opponent's cards on its way to, or sitting in, its seat.
//
// **The row got state on 2026-08-12** when it started being laid out during planning. It was
// drawn straight from `enemyQueueOrder` every frame, which was enough while it only ever
// appeared fully formed at DUEL!; a row that arrives has to remember how far along each card is.
type dealtCard struct {
	travel

	card combat.Card
}

// seatEnemyCards lays the opponent's whole queue out and sets it flying in.
//
// **Called when the opponent plans, which is now the start of the planning phase** rather than
// the moment DUEL! is pressed — see startRound and Init. That is the whole point of the change:
// the player picks their round against a hand they can see.
func (s *CombatScene) seatEnemyCards() {
	queue := s.enemyQueueOrder()

	s.enemyDealt = make([]dealtCard, 0, len(queue))
	for i, c := range queue {
		s.enemyDealt = append(s.enemyDealt, dealtCard{
			travel: newTravel(i*flightStaggerPer, riseTicks),
			card:   c,
		})
	}
}

// at is where one of the opponent's cards is drawn right now: waiting to set off, flying in
// from the enemy's own card, sitting in its seat, or lifted out of it because it is resolving.
//
// **They fly in from the enemy card in the top-right corner**, which is the opponent itself —
// the mirror of the player's cards coming out of their hand. There is no enemy draw pile on
// screen to deal from and inventing one would be a second thing to explain; the fighter card is
// already the thing on screen that *is* the opponent, so cards coming out of it read as theirs
// without a caption.
func (s *CombatScene) enemyCardAt(gs *state.GlobalState, d dealtCard, seat, total, split int, firing bool) image.Point {
	from := s.enemyCardRect(gs).Min
	to := enemySeatAt(gs, seat, total, split)

	switch {
	case d.waiting():
		return from
	case !d.done():
		return lerpPoint(from, to, easeOut(d.progress()))
	}

	// The lift only after landing, exactly as the player's row does it — a card still arriving
	// is already the most moving thing on screen and lifting it too would blur the two beats.
	return lift(to, firing)
}

// drawEnemyQueue draws the opponent's hand for this round, right-aligned in the table band.
//
// **It is drawn during planning, and that is the point of the row** *(2026-08-12)*. It used to
// appear only once DUEL! was pressed, because `enemyActions` held *last* round's plan until then
// and a row drawn during planning would have been a confident picture of the wrong round. The
// opponent now plans at the start of the planning phase instead, so the row is this round's and
// the player chooses their own cards against it.
//
// **What that gives up: the opponent's intentions are no longer hidden.** `concealEnemy` is
// still the screen's single concealment predicate and the Action Flow pane still obeys it, but
// with the cards face up on the table there is nothing left for it to hide — this is an
// open-information duel now, on the owner's call. **The lever is still built**:
// `cards.Spec.FaceDown` draws a back instead of a face, so hiding this row again is one field
// rather than a second drawing path.
//
// **Elements come through from the deck**, and every card is basic because
// `data/enemy_cards.json` is: MECHANICS.md has affixes transforming a basic deck into an element
// and none of that exists yet. So they draw with the neutral mid-grey border, which is the truth
// rather than a placeholder.
func (s *CombatScene) drawEnemyQueue(gs *state.GlobalState, screen *ebiten.Image) {
	split := s.enemySplit()
	for i, d := range s.enemyDealt {
		at := s.enemyCardAt(gs, d, i, len(s.enemyDealt), split, lit(s.enemyFiringSeats, i))
		// **The opponent's own cost, not the player's** — a discount ring is the player's and a
		// queued enemy card printing a discounted price would be the screen telling a lie about
		// whose ring it is.
		drawCard(gs, screen, at, cards.Hand, d.card, s.enemy.CardCost(d.card), true, false)
	}
}
