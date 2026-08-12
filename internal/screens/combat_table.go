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
	return gs.PctY(handTopPct) - feedGapAboveCards - feedHeight() - firingGap - cardHeight
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
func tablePitch(gs *state.GlobalState, n int) int {
	full := cardWidth + cardGap
	if n < 2 {
		return full
	}
	if (n-1)*full+cardWidth <= tableHalfWidth(gs) {
		return full
	}
	return (tableHalfWidth(gs) - cardWidth) / (n - 1)
}

// playedSeatAt is where the nth of the player's played cards sits: **left-aligned**, the whole
// row placed at once from the screen's left inset.
//
// **The pitch is a function of the round's total, never of how many have landed so far.** Both
// hands are dealt to the table the moment DUEL! is pressed, so the total is known — and a pitch
// that grew with the row would shuffle every card already sitting there sideways each time
// another arrived, which reads as the layout twitching rather than as a card being played.
func playedSeatAt(gs *state.GlobalState, n, total int) image.Point {
	if total < 1 {
		total = 1
	}
	return image.Pt(tableInset+n*tablePitch(gs, total), tableRowTop(gs))
}

// enemySeatAt is where the nth of the opponent's queued cards sits: **right-aligned**, the
// whole row placed at once so its last card is flush with the right inset.
//
// **The two rows are anchored differently on purpose.** The opponent's queue is known in full
// the moment the round starts, so it is laid out as a finished hand against its own edge; the
// player's is revealed a card at a time. Pinning both to the outside edges is what makes them
// face each other rather than drift toward the middle as they fill.
func enemySeatAt(gs *state.GlobalState, n, total int) image.Point {
	if total < 1 {
		total = 1
	}
	pitch := tablePitch(gs, total)
	width := (total-1)*pitch + cardWidth
	left := gs.ScreenWidth - tableInset - width

	return image.Pt(left+n*pitch, tableRowTop(gs))
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

// drawEnemyQueue draws the opponent's hand for this round, right-aligned in the table band.
//
// **It is drawn face up and that is temporary** *(2026-08-12)*. `concealEnemy` is still the
// screen's single predicate for hiding the opponent's intentions and the Action Flow pane still
// obeys it; this row deliberately does not, because the owner asked to see the cards first and
// decide what to hide afterwards. **The lever is already built when that day comes**:
// `cards.Spec.FaceDown` draws a back instead of a face and the draw pile is a stack of them, so
// concealing a card here is one field rather than a second drawing path.
//
// **Nothing is drawn while the player is planning.** `enemyActions` holds *last* round's plan
// until DUEL! is pressed and re-plans it, so a row drawn during planning would be a confident
// picture of the wrong round — which is worse than an empty band. This is also what makes the
// two rows appear and disappear together.
//
// **Every card is elementless**, because `data/enemy_cards.json` is: MECHANICS.md has affixes
// transforming a basic deck into an element and none of that exists yet. So they draw with the
// neutral mid-grey border, which is the truth rather than a placeholder.
func (s *CombatScene) drawEnemyQueue(gs *state.GlobalState, screen *ebiten.Image) {
	if s.planning() {
		return
	}

	queue := s.enemyQueueOrder()
	for i, c := range queue {
		at := lift(enemySeatAt(gs, i, len(queue)), i == s.enemyFiringSeat)
		drawCard(gs, screen, at, cards.Hand, c, true, false, s.enemy.Str)
	}
}
