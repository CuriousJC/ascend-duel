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
	"sort"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
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
)

// tableRowTop is the y both rows sit at: as low as it can go without reaching the hand.
//
// **The hand dialog reserves nothing here** *(owner's call)*: it is an overlay, drawn last, its lines
// under these cards and over whatever is below them. See mathLineAt.
//
// This is where `firingAt` used to put a single card; what moved is that a whole row lives here
// now instead of one card passing through.
func tableRowTop(gs *state.GlobalState) int {
	return handTop(gs) - mathBandGapAboveCards - firingGap - cardHeight
}

// tableCenter is the middle of the whole table — both halves — on the row's own center line.
//
// It is where the planned hand is named — see drawPlannedHand — and it is derived from the row
// rather than written down, so the name follows the table if the table moves.
//
// **It is centered on the screen and not on the player's half** *(2026-08-19, owner's call)*. The
// name used to sit over the space the player's own cards fly into, which kept it out of the
// opponent's row at the cost of putting the loudest word on the screen off to one side; at 80
// points and growing it was also the half's whole width. **Overlaying the opponent's cards is
// accepted rather than worked around**: those cards have been read by the time a hand is named,
// the word is up only while the round is being planned and while it plays back, and the
// alternative — shrinking the name to fit a half — is the opposite of what the size is for.
func tableCenter(gs *state.GlobalState) image.Point {
	return image.Pt(gs.ScreenWidth/2, tableRowTop(gs)+cardHeight/2)
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
// a rule that a relic or a tonic is expected to raise, so a pitch that only worked for five
// would put a rendering bug behind a balance change.
func tablePitch(gs *state.GlobalState, n int) int {
	full := cardWidth + cardGap
	if n < 2 {
		return full
	}
	room := tableHalfWidth(gs)
	if (n-1)*full+cardWidth <= room {
		return full
	}
	return (room - cardWidth) / (n - 1)
}

// playedSeatAt is where the nth of the player's played cards sits: **left-aligned**, the whole
// row placed at once from the screen's left inset. `n` is a *place* — the card's position in the
// turn as the player picked it — never a seat in the resolved turn; see playedPoint.
//
// **The pitch is a function of the round's total, never of how many have landed so far.** Both
// hands are dealt to the table the moment DUEL! is pressed, so the total is known — and a pitch
// that grew with the row would shuffle every card already sitting there sideways each time
// another arrived, which reads as the layout twitching rather than as a card being played.
func playedSeatAt(gs *state.GlobalState, n, total int) image.Point {
	if total < 1 {
		total = 1
	}
	x := tableInset + n*tablePitch(gs, total)
	return image.Pt(x, tableRowTop(gs))
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

// **A row on the table is the turn as it was picked, one hand, in one run** *(owner's call)*. The
// engine still plays a turn defenses first — `combat.ResolutionOrder` — and every event names its
// card by its seat in that resolved turn, which is what `resolved`, `enemyDealt`, `firingSeats` and
// every mover hold. **Only where a card is drawn is its place in the queue**, so the row reads left
// to right exactly as the cards were chosen, and nothing regroups between the hand and the table.
// The cards that made the multiplier are the ones on a plate.

// playedPlace is where the card in one seat of the resolved turn sits along the player's row.
func (s *CombatScene) playedPlace(seat int) int {
	if seat >= 0 && seat < len(s.Theater.resolved) {
		return s.Theater.resolved[seat].place
	}
	return seat
}

// enemyPlace is the same for the opponent's row.
func (s *CombatScene) enemyPlace(seat int) int {
	if seat >= 0 && seat < len(s.Theater.enemyDealt) {
		return s.Theater.enemyDealt[seat].place
	}
	return seat
}

// playedPoint and enemyPoint are the resting top-left of the card in one seat of the resolved turn.
func (s *CombatScene) playedPoint(gs *state.GlobalState, seat int) image.Point {
	return playedSeatAt(gs, s.playedPlace(seat), len(s.Theater.resolved))
}

func (s *CombatScene) enemyPoint(gs *state.GlobalState, seat int) image.Point {
	return enemySeatAt(gs, s.enemyPlace(seat), len(s.Theater.enemyDealt))
}

// byPlace is a row's seats in the order they sit left to right, so a later card overlaps the one
// before it on screen rather than in the resolver's order.
func byPlace(n int, place func(int) int) []int {
	seats := make([]int, n)
	for i := range seats {
		seats[i] = i
	}
	sort.SliceStable(seats, func(a, b int) bool { return place(seats[a]) < place(seats[b]) })
	return seats
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
	ui.Travel

	card combat.Card

	// place is where the card sits along the row: its position in the opponent's queue.
	place int
}

// seatEnemyCards lays the opponent's whole queue out and sets it flying in.
//
// **Called when the opponent plans, which is now the start of the planning phase** rather than
// the moment DUEL! is pressed — see startRound and Init. That is the whole point of the change:
// the player picks their round against a hand they can see.
func (s *CombatScene) seatEnemyCards() {
	var places []int
	for _, slot := range combat.ResolutionOrder(s.fighterActions, s.enemyActions) {
		if slot.Side == combat.SideB && slot.Index < len(s.enemyActions) {
			places = append(places, slot.Index)
		}
	}
	queue := s.enemyQueueOrder()

	// **A stopped attack belongs to the row it happened in.** Its seat is a number, and this is
	// where the seats stop meaning what they meant, so the round's intercepts go with the row.
	s.Theater.intercepts = nil
	s.enemyShake = nil

	s.Theater.enemyDealt = make([]dealtCard, 0, len(queue))
	for i, c := range queue {
		place := i
		if i < len(places) {
			place = places[i]
		}
		s.Theater.enemyDealt = append(s.Theater.enemyDealt, dealtCard{
			Travel: ui.NewTravel(place*flightStaggerPer(), riseTicks()),
			card:   c,
			place:  place,
		})
	}
}

// at is where one of the opponent's cards is drawn right now: waiting to set off, flying in
// from the enemy's own card, or sitting in its seat. **A card on the table never lifts** — the one
// swinging rattles sideways instead, which the row adds; see enemyCardShake.
//
// **They fly in from the enemy card in the top-right corner**, which is the opponent itself —
// the mirror of the player's cards coming out of their hand. There is no enemy draw pile on
// screen to deal from and inventing one would be a second thing to explain; the fighter card is
// already the thing on screen that *is* the opponent, so cards coming out of it read as theirs
// without a caption.
func (s *CombatScene) enemyCardAt(gs *state.GlobalState, d dealtCard, total int) image.Point {
	from := ui.EnemyCardRect(gs).Min
	to := enemySeatAt(gs, d.place, total)

	switch {
	case d.Waiting():
		return from
	case !d.Done():
		return ui.LerpPoint(from, to, ui.EaseOut(d.Progress()))
	}
	return to
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
// and none of that exists yet. So they draw with the neutral mid-gray border, which is the truth
// rather than a placeholder.
func (s *CombatScene) drawEnemyQueue(gs *state.GlobalState, screen *ebiten.Image) {
	for _, i := range byPlace(len(s.Theater.enemyDealt), s.enemyPlace) {
		d := s.Theater.enemyDealt[i]
		at := s.enemyCardAt(gs, d, len(s.Theater.enemyDealt))
		at.X += s.enemyCardShake(i)
		// **The opponent's own cost, not the player's** — a discount relic is the player's and a
		// queued enemy card printing a discounted price would be the screen telling a lie about
		// whose relic it is.
		mark := ui.MarksFor(gs, image.Rect(at.X, at.Y, at.X+cardWidth, at.Y+cardHeight))
		ui.DrawMarkedCard(gs, screen, at, cards.Hand, d.card, ui.HeldBy(s.enemy.Duelist, d.card),
			true, false, mark)
	}
}
