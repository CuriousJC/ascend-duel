package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// The table's geometry, which is arithmetic and needs no window — the same narrow exception
// the other tests in this package take. Nothing here creates an ebiten.Image.
//
// What these guard is the one property the arrangement exists for: **two hands that face each
// other and never touch.** Five cards a side do not fit in a screen at full size, so the rows
// overlap within themselves; the moment they overlap *into* each other the picture stops being
// a confrontation and becomes one row of ten.

// tableSeats is the widest row the rules can produce. Asked of the rules rather than written
// down, because a relic raising the cap is an expected change and must not quietly break the
// layout instead of failing here.
func tableSeats() int { return combat.Duelist{}.MaxActions() }

func TestTheTwoHandsNeverReachEachOther(t *testing.T) {
	// **Swept across every split as well as every count** *(2026-08-15)*. A row now leaves a gap
	// between its attacks and its plans, and that gap is spent out of the same half-width — so a
	// split that widened the row instead of tightening its overlap is exactly how the two hands
	// would meet in the middle.
	gs := testState()

	for n := 1; n <= tableSeats(); n++ {
		for split := 0; split <= n; split++ {
			player := playedSeatAt(gs, n-1, n, split)
			enemy := enemySeatAt(gs, 0, n, split)

			playerRight := player.X + cardWidth
			if playerRight >= enemy.X {
				t.Errorf("%d cards a side split at %d: the player's row ends at x=%d and the opponent's starts at x=%d",
					n, split, playerRight, enemy.X)
			}
		}
	}
}

func TestARowBreaksBetweenItsAttacksAndItsPlans(t *testing.T) {
	// The point of the break: the card after it is further along than the pitch alone would put
	// it, and the cards before it are exactly where they always were.
	gs := testState()

	const n, split = 4, 2
	pitch := tablePitch(gs, n, split)

	for i := 0; i < split; i++ {
		want := tableInset + i*pitch
		if got := playedSeatAt(gs, i, n, split).X; got != want {
			t.Errorf("attack %d sits at x=%d, want %d — nothing before the break moves", i, got, want)
		}
	}
	for i := split; i < n; i++ {
		want := tableInset + i*pitch + tableGroupGap
		if got := playedSeatAt(gs, i, n, split).X; got != want {
			t.Errorf("plan %d sits at x=%d, want %d — the whole gap once, at the break", i, got, want)
		}
	}

	// And a row that is all one kind gets no gap at all, at either end of the range.
	for _, split := range []int{0, n} {
		if got := groupGapFor(n, split); got != 0 {
			t.Errorf("a row split at %d of %d left %dpx of break, want none", split, n, got)
		}
	}
}

func TestTheSplitIsTakenFromResolutionOrder(t *testing.T) {
	// **The boundary is the engine's, not the screen's.** ResolutionOrder puts a turn's attacks
	// first and its plans second; a row that counted its own would be a second answer to a
	// question already settled, and would drift the first time a card changed category.
	s := &CombatScene{
		enemyActions: combat.PlainCards(combat.Guard, combat.Bash, combat.Brace, combat.Jab),
	}
	s.seatEnemyCards()

	if got := s.enemySplit(); got != 2 {
		t.Errorf("the opponent's row splits at %d, want 2 — two attacks then two plans", got)
	}
	if got := splitOf(s.enemyQueueOrder()); got != s.enemySplit() {
		t.Errorf("splitOf says %d and the row says %d", got, s.enemySplit())
	}

	// A row with no plans in it splits at its end, which reads as no break.
	all := &CombatScene{enemyActions: combat.PlainCards(combat.Bash, combat.Jab)}
	all.seatEnemyCards()
	if got := all.enemySplit(); got != 2 {
		t.Errorf("an all-attack row splits at %d, want its length", got)
	}
}

func TestEachRowIsPinnedToItsOwnEdge(t *testing.T) {
	gs := testState()

	for n := 1; n <= tableSeats(); n++ {
		for split := 0; split <= n; split++ {
			// The player's grows rightward from the left inset.
			if got := playedSeatAt(gs, 0, n, split).X; got != tableInset {
				t.Errorf("%d cards split at %d: the player's row starts at x=%d, want the %dpx inset",
					n, split, got, tableInset)
			}

			// The opponent's is right-aligned, so its *last* card is flush with the right inset
			// whatever the count. That is what makes a hand of two hug its own edge rather than
			// drift toward the middle — and it has to survive the break, which is the thing most
			// likely to knock a right-aligned row off its edge.
			last := enemySeatAt(gs, n-1, n, split).X + cardWidth
			if want := gs.ScreenWidth - tableInset; last != want {
				t.Errorf("%d cards split at %d: the opponent's row ends at x=%d, want %d",
					n, split, last, want)
			}
		}
	}
}

func TestTheTableSitsBetweenTheRelicRowAndTheFeed(t *testing.T) {
	gs := testState()
	s := &CombatScene{}

	top := tableRowTop(gs)

	// Below the top row, whose lowest ink is the row of relic cards itself. **The rule and the
	// count used to hang under it and moved into the caption column on 2026-09-04**, which is the
	// 44 pixels that let the card grow to its present size — see relicPaneRect.
	relicBottom := s.relicPaneBackRect(gs).Max.Y
	if top < relicBottom {
		t.Errorf("the table starts at y=%d, into the relic row's count ending at y=%d", top, relicBottom)
	}

	// And clear of the hand. **The band the sum is written in is no longer reserved**
	// *(2026-09-04, owner's call)*: the arithmetic overlays the bottom of the played row instead of
	// pushing it up, which is what let the row drop clear of the relic pane's backing above it.
	if bottom, handTop := top+cardHeight, handTop(gs)-mathBandGapAboveCards; bottom > handTop {
		t.Errorf("the table ends at y=%d, into the hand at y=%d", bottom, handTop)
	}
}

func TestTheOpponentsRowIsInResolutionOrder(t *testing.T) {
	// **The row must say what will happen, not what was planned.** ResolutionOrder regroups a
	// turn into plans then attacks, so a queue planned attack-first comes out of the planner in
	// one order and resolves in another. (It regrouped the other way until 2026-09-15; see
	// combat.Categories.)
	s := &CombatScene{
		enemyActions: combat.PlainCards(combat.Bash, combat.Jab, combat.Brace),
	}

	got := s.enemyQueueOrder()
	want := combat.PlainCards(combat.Brace, combat.Bash, combat.Jab)

	if len(got) != len(want) {
		t.Fatalf("the row holds %d cards, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("seat %d holds %v, want %v", i, got[i], want[i])
		}
	}
}

func TestPlayedSeatsDoNotMoveAsTheRoundPlaysOut(t *testing.T) {
	// The whole row is dealt at round start, so a seat's position is a function of the round's
	// total and never of how many cards have landed. A pitch derived from the latter would
	// shuffle every seated card sideways each time another arrived.
	gs := testState()

	const total, split = 4, 2
	first := playedSeatAt(gs, 0, total, split)
	for n := 1; n < total; n++ {
		if got := playedSeatAt(gs, 0, total, split); got != first {
			t.Fatalf("seat 0 moved to %v while %d cards were down, from %v", got, n, first)
		}
	}
}

func TestSeatingWalksTheSameOrderAsPlayback(t *testing.T) {
	// seatPlayedCards lays the row out and noteResolved lights one of its seats; both count
	// along combat.ResolutionOrder. If they ever took the order from different places the lit
	// card would be the wrong one, which is exactly the bug the old per-event pile could not
	// have — so this is what replaces that safety.
	s := &CombatScene{
		hand: []paletteCard{
			{Card: combat.Card{Concept: combat.Brace, Element: combat.Ice}, selected: true},
			{Card: combat.Card{Concept: combat.Bash, Element: combat.Fire}, selected: true},
			{Card: combat.Card{Concept: combat.Jab, Element: combat.Earth}, selected: true},
		},
		fighterActions: []combat.Card{
			combat.Of(combat.Brace, combat.Ice),
			combat.Of(combat.Bash, combat.Fire),
			combat.Of(combat.Jab, combat.Earth),
		},
	}
	s.seatPlayedCards()

	// The plan first and the two attacks after — and each seat holds the card the player
	// actually selected for it, not the one in the same position in the hand.
	// The elements come along, so a seat holding the right concept in the wrong color fails
	// too — which is the whole reason the hand and the queue are one type now.
	want := []combat.Card{
		combat.Of(combat.Brace, combat.Ice),
		combat.Of(combat.Bash, combat.Fire),
		combat.Of(combat.Jab, combat.Earth),
	}
	if len(s.Theater.resolved) != len(want) {
		t.Fatalf("%d cards were seated, want %d", len(s.Theater.resolved), len(want))
	}
	for i, c := range want {
		if got := s.Theater.resolved[i].card; got != c {
			t.Errorf("seat %d holds %v, want %v", i, got, c)
		}
	}

	// And every seat knows which hand slot it came from, which is what the end-of-round throw
	// and the hand row's own hiding both read.
	for i, r := range s.Theater.resolved {
		if r.handIndex < 0 || r.handIndex >= len(s.hand) {
			t.Errorf("seat %d came from hand slot %d, which is not in a hand of %d",
				i, r.handIndex, len(s.hand))
		}
	}
}

func TestBothRowsRaiseTheCardThatIsResolving(t *testing.T) {
	// **The gesture has to be the same on both sides.** It arrived on the player's row alone
	// and the opponent's cards sat still through their whole turn, which read as the enemy's
	// hand being scenery rather than the other half of the round.
	gs := testState()

	player := playedSeatAt(gs, 1, 3, 3)
	enemy := enemySeatAt(gs, 1, 3, 3)

	if got := lift(player, true); got.Y != player.Y-tableFireLift {
		t.Errorf("a firing card on the player's row sits at y=%d, want %d", got.Y, player.Y-tableFireLift)
	}
	if got := lift(enemy, true); got.Y != enemy.Y-tableFireLift {
		t.Errorf("a firing card on the opponent's row sits at y=%d, want %d", got.Y, enemy.Y-tableFireLift)
	}

	// And only the x stays put, so a lift can never be mistaken for a card sliding along.
	if got := lift(enemy, true); got.X != enemy.X {
		t.Errorf("lifting moved a card sideways, to x=%d from %d", got.X, enemy.X)
	}
	if got := lift(enemy, false); got != enemy {
		t.Errorf("an idle card was moved to %v from %v", got, enemy)
	}
}

// pairID is a rung of two or more, which is what makes an event one whose cards the screen raises.
// Every rung is announced, the No Hand included; the lift is the one thing it goes without. See
// builtARung.
func pairID(t *testing.T) combat.HandID {
	t.Helper()
	h, ok := combat.HandByName("Pair")
	if !ok {
		t.Fatal("the catalog holds no Pair")
	}
	return h.ID
}

// **A No Hand raises nothing.** It is announced — `NO HAND!`, and the sum plays — but the lift says
// which cards *made* the rung, and that turn made none.
// Its `Blow.Rung` is one card picked by damage rather than by counting, so raising it stood a single
// card up as though it had done something while every other attack landed beside it unraised.
func TestANoHandRaisesNothing(t *testing.T) {
	none, ok := combat.HandByName("No Hand")
	if !ok {
		t.Fatal("the catalog holds no No Hand rung")
	}

	s := &CombatScene{}
	e := combat.Event{
		Kind: combat.KindHand, Side: combat.SideA,
		Hand: none.ID, HandCardCount: 3, RungCardCount: 1,
	}
	e.HandCards[0], e.HandCards[1], e.HandCards[2] = 0, 1, 2
	e.RungCards[0] = 2

	s.noteHand(e)
	if got := s.Theater.firingSeats; len(got) != 0 {
		t.Errorf("a No Hand raised %v, want nothing", got)
	}

	// And a built rung on the same shape does raise, so the stillness is the No Hand's rather than
	// the event being malformed.
	e.Hand = pairID(t)
	e.RungCardCount = 2
	e.RungCards[0], e.RungCards[1] = 0, 1
	s.noteHand(e)
	if got := s.Theater.firingSeats; !sameSeats(got, []int{0, 1}) {
		t.Errorf("a Pair raised %v, want the two cards that formed it", got)
	}
}

func TestOnlyOneSideOfTheTableIsLitAtATime(t *testing.T) {
	// A turn is contiguous per side, so the lit cards walk the left row and then the right
	// one. The event that lights one side is the event that unlights the other, which is why
	// neither row has to know the other exists.
	//
	// **The player's row is lit by the hand's announcement and the creature's by its own card.** A
	// duelist's cards do not lift on their own announcements — the raise is the announcement's,
	// once, for the rung — and a creature forms no hand, so its card is the only thing that can say
	// which blow is landing.
	s := &CombatScene{
		fighterActions: combat.PlainCards(combat.Bash),
		enemyActions:   combat.PlainCards(combat.Jab),
		enemy:          &entities.Combatant{Duelist: combat.Duelist{SoloAttacks: true}},
		log: []combat.Event{
			{Kind: combat.KindAction, Side: combat.SideA, Action: combat.Bash},
			{Kind: combat.KindAction, Side: combat.SideB, Action: combat.Jab},
		},
	}

	hand := combat.Event{
		Kind: combat.KindHand, Side: combat.SideA,
		Hand: pairID(t), RungCardCount: 1,
	}
	s.noteHand(hand)
	if !sameSeats(s.Theater.firingSeats, []int{0}) || len(s.Theater.enemyFiringSeats) != 0 {
		t.Errorf("after the player's hand: player %v, enemy %v — want [0] and none",
			s.Theater.firingSeats, s.Theater.enemyFiringSeats)
	}

	// The cursor points at the event being applied, not past it — advancePlayback calls
	// applyEvent before it increments. currentSlot counts inclusively for that reason, so the
	// player's own announcement has to stay in the log for the creature's to be the second slot.
	s.cursor = 1
	s.noteResolved(s.log[1])
	if len(s.Theater.firingSeats) != 0 || !sameSeats(s.Theater.enemyFiringSeats, []int{0}) {
		t.Errorf("after the opponent's card: player %v, enemy %v — want none and [0]",
			s.Theater.firingSeats, s.Theater.enemyFiringSeats)
	}
}

func TestOnlyTheRungIsRaisedAndOnlyOnTheAnnouncement(t *testing.T) {
	// **A turn lands one blow, and the cards that made it go up on the hand's announcement and on
	// nothing else.** A raise at an attack card's own beat lands one or two beats before the hand is
	// named, which on a turn of two shields and one attack stands the attack up alone and lets the
	// shields join it later — the attack visibly going first.
	s := &CombatScene{
		hand: []paletteCard{
			{Card: combat.Card{Concept: combat.Bash, Element: combat.Fire}, selected: true},
			{Card: combat.Card{Concept: combat.Bash, Element: combat.Ice}, selected: true},
			{Card: combat.Card{Concept: combat.Jab, Element: combat.Basic}, selected: true},
		},
		fighterActions: []combat.Card{
			combat.Of(combat.Bash, combat.Fire),
			combat.Of(combat.Bash, combat.Ice),
			combat.Of(combat.Jab, combat.Basic),
		},
		log: []combat.Event{
			{Kind: combat.KindAction, Side: combat.SideA, Action: combat.Bash},
			{Kind: combat.KindAction, Side: combat.SideA, Action: combat.Bash},
			{Kind: combat.KindAction, Side: combat.SideA, Action: combat.Jab},
		},
	}
	s.seatPlayedCards()

	// The announcements say how long the phase takes and which card is being spoken about in the
	// log. They lift nothing: no single card is acting, and the blow has not been named yet.
	for i := range s.log {
		s.cursor = i
		s.noteResolved(s.log[i])
		if got := s.Theater.firingSeats; len(got) != 0 {
			t.Fatalf("announcement %d raised %v, want nothing up before the hand is named", i, got)
		}
	}

	// The hand's announcement is what raises them, and it raises **the rung** — the cards that made
	// the hand — rather than every card that paid into the blow. The Jab is in the sum and not in
	// the Pair, so it stays down. **Raising is the whole of what says which cards earned the hand**,
	// there being no ring or bracket drawn round them, which is why this is the only assertion here.
	hand := combat.Event{
		Kind: combat.KindHand, Side: combat.SideA,
		Hand:          pairID(t),
		HandCardCount: 3, RungCardCount: 2,
	}
	hand.HandCards[0], hand.HandCards[1], hand.HandCards[2] = 0, 1, 2
	hand.RungCards[0], hand.RungCards[1] = 0, 1
	s.noteHand(hand)

	if !sameSeats(s.Theater.firingSeats, []int{0, 1}) {
		t.Errorf("the hand left %v raised, want only the two cards that formed it", s.Theater.firingSeats)
	}
}

// sameSeats compares two seat lists as lists — order included, since the seats are appended in
// the order the cards resolved.
func sameSeats(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestAPlayedCardFliesFromItsHandSlotToItsSeat(t *testing.T) {
	gs := testState()

	r := resolvedCard{Travel: ui.NewTravel(0, riseTicks()), handIndex: 2, handCount: handSize}
	from := slotAt(gs, 2, handSize)
	to := playedSeatAt(gs, 1, 3, 3)

	if got := r.at(gs, 1, 3, 3, false); got != from {
		t.Errorf("a card that has not set off is at %v, want its hand slot %v", got, from)
	}

	r.Age = riseTicks()
	if got := r.at(gs, 1, 3, 3, false); got != to {
		t.Errorf("a landed card is at %v, want its seat %v", got, to)
	}

	// Lifted while it resolves, and only once it has landed — a card still arriving is already
	// the most moving thing on screen.
	if got := r.at(gs, 1, 3, 3, true); got.Y != to.Y-tableFireLift {
		t.Errorf("a firing card sits at y=%d, want %d", got.Y, to.Y-tableFireLift)
	}
	r.Age = 0
	if got := r.at(gs, 1, 3, 3, true); got != from {
		t.Errorf("a card that has not set off was lifted: %v, want %v", got, from)
	}
}

// The opponent's row arriving during planning, 2026-08-12.

func TestTheOpponentsRowIsSeatedFromItsQueue(t *testing.T) {
	// seatEnemyCards has to lay the row out in resolution order for the same reason
	// enemyQueueOrder does: a row in the planner's order would be a picture of a round that
	// does not happen. It is the same walk, and this pins that seating uses it rather than
	// taking the queue as planned.
	s := &CombatScene{
		enemyActions: combat.PlainCards(combat.Bash, combat.Jab, combat.Brace),
	}
	s.seatEnemyCards()

	want := combat.PlainCards(combat.Brace, combat.Bash, combat.Jab)
	if len(s.Theater.enemyDealt) != len(want) {
		t.Fatalf("%d cards were seated, want %d", len(s.Theater.enemyDealt), len(want))
	}
	for i, c := range want {
		if got := s.Theater.enemyDealt[i].card; got != c {
			t.Errorf("seat %d holds %v, want %v", i, got, c)
		}
	}
}

func TestTheOpponentsCardsFlyInFromTheEnemyCard(t *testing.T) {
	// **They come out of the opponent itself**, which is the mirror of the player's cards
	// coming out of their hand. There is no enemy draw pile on screen and inventing one would
	// be a second thing to explain.
	gs := testState()
	s := &CombatScene{}

	d := dealtCard{Travel: ui.NewTravel(0, riseTicks())}
	from := ui.EnemyCardRect(gs).Min
	to := enemySeatAt(gs, 1, 3, 3)

	if got := s.enemyCardAt(gs, d, 1, 3, 3, false); got != from {
		t.Errorf("a card that has not set off is at %v, want the enemy card at %v", got, from)
	}

	d.Age = riseTicks()
	if got := s.enemyCardAt(gs, d, 1, 3, 3, false); got != to {
		t.Errorf("a landed card is at %v, want its seat %v", got, to)
	}

	// Lifted only once it has landed, exactly as the player's row does it.
	if got := s.enemyCardAt(gs, d, 1, 3, 3, true); got.Y != to.Y-tableFireLift {
		t.Errorf("a firing card sits at y=%d, want %d", got.Y, to.Y-tableFireLift)
	}
	d.Age = 0
	if got := s.enemyCardAt(gs, d, 1, 3, 3, false); got != from {
		t.Errorf("a card back on the pad is at %v, want %v", got, from)
	}
}

func TestBothRowsUseTheSameArrivalClock(t *testing.T) {
	// The two sides deal at the same speed and stagger the same way, or the table reads as one
	// row arriving and one row appearing. Both take riseTicks() and flightStaggerPer() from the
	// same constants, and both count with the same travel — this is what stops a later change
	// to one of them being made twice.
	s := &CombatScene{
		hand: []paletteCard{
			{Card: combat.Plain(combat.Bash), selected: true},
			{Card: combat.Plain(combat.Jab), selected: true},
		},
		fighterActions: combat.PlainCards(combat.Bash, combat.Jab),
		enemyActions:   combat.PlainCards(combat.Bash, combat.Jab),
	}
	s.seatPlayedCards()
	s.seatEnemyCards()

	if len(s.Theater.resolved) != len(s.Theater.enemyDealt) {
		t.Fatalf("%d player seats against %d enemy seats", len(s.Theater.resolved), len(s.Theater.enemyDealt))
	}
	for i := range s.Theater.resolved {
		if got, want := s.Theater.enemyDealt[i].Travel, s.Theater.resolved[i].Travel; got != want {
			t.Errorf("seat %d: enemy clock %+v, player clock %+v", i, got, want)
		}
	}
}

func TestTheOpponentPlansOnceAndTheTableShowsThatPlan(t *testing.T) {
	// **The row has to be the round that will actually resolve.** It was a picture of last
	// round's plan until 2026-08-12, which is why it was hidden during planning; the fix is that
	// the opponent commits at the start of the planning phase instead. If startRound ever
	// re-planned, the cards the player chose against would not be the cards they faced.
	s := &CombatScene{}
	s.enemyPile = decks.NewEnemyPile(testEnemyRecord, "", seeds.EnemyDeckPin, decks.EnemyHandSize)
	s.enemy = &entities.Combatant{
		Duelist: combat.Duelist{DMG: 5, Actions: 5, MaxLife: 60, CurrentLife: 60},
	}
	s.fighter = &entities.Combatant{
		Duelist: combat.Duelist{DMG: 10, Actions: 6, MaxLife: 60, CurrentLife: 60},
	}

	s.planEnemyRound()

	planned := append([]combat.Card(nil), s.enemyActions...)
	if len(planned) == 0 {
		t.Fatal("the opponent planned nothing to look at")
	}

	// What is on the table is what was planned, in resolution order.
	if len(s.Theater.enemyDealt) != len(planned) {
		t.Fatalf("%d cards on the table against a plan of %d", len(s.Theater.enemyDealt), len(planned))
	}
	for i, c := range s.enemyQueueOrder() {
		if got := s.Theater.enemyDealt[i].card; got != c {
			t.Errorf("seat %d holds %v, want %v", i, got, c)
		}
	}
}

// selecting builds a scene whose hand is these cards, all selected, with the queue derived from
// them the way syncQueue does. The player is left alive with no log, which is what planning() is.
func selecting(cards ...combat.Card) *CombatScene {
	s := &CombatScene{
		fighter: &entities.Combatant{
			Duelist: combat.Duelist{DMG: 10, Actions: 5, MaxLife: 60, CurrentLife: 60},
		},
		enemy: &entities.Combatant{
			Duelist: combat.Duelist{DMG: 5, Actions: 5, MaxLife: 60, CurrentLife: 60},
		},
	}
	for _, c := range cards {
		s.hand = append(s.hand, paletteCard{Card: c, selected: true})
		s.fighterActions = append(s.fighterActions, c)
	}
	return s
}

func TestAHandPreviewsTheMomentItIsSelected(t *testing.T) {
	// **The preview is the resolver's own answer**, so what is named while choosing is what
	// fires. Three Bashes are three of a kind the instant the third is picked, not when DUEL! is
	// pressed.
	s := selecting(
		combat.Of(combat.Bash, combat.Fire),
		combat.Of(combat.Bash, combat.Ice),
		combat.Of(combat.Bash, combat.Basic),
	)

	blow, _, ok := s.previewBlow()
	if !ok {
		t.Fatal("three Bashes previewed no hand")
	}
	if len(blow.Cards) != 3 {
		t.Errorf("the previewed hand is made of %v, want all three cards", blow.Cards)
	}

	// And it is named in the same words the fired shout will use.
	//
	// **This moved out of the feed on 2026-08-18.** The pane carried `HAND! Three of a Kind
	// x2` while planning; the words are now written across the band the sum will fill, by
	// `drawPlannedHand`. What has to hold either way is that the preview and the announcement are
	// one spelling — two spellings of PAIR would read as two different things happening — and
	// `handShout` is the single function both go through, so that is what is checked here rather
	// than a row of pane text.
	if got, want := handShout(blow.Hand.Name), "CARD THREE OF A KIND!"; got != want {
		t.Errorf("the planned hand reads %q, want %q", got, want)
	}
	if blow.Hand.Key != "concept-three-of-a-kind" {
		t.Errorf("three Bashes previewed %q, want the three of a kind", blow.Hand.Key)
	}
}

func TestOneAttackIsTheNoHand(t *testing.T) {
	// **A single attack is a hand and is named as one** *(2026-08-19, owner's call)*, where it used
	// to preview nothing at all. The label is on screen from the first attack card picked rather
	// than appearing only if a pair happens to form.
	s := selecting(combat.Of(combat.Bash, combat.Fire))

	blow, ok := s.previewAttack()
	if !ok {
		t.Fatal("one Bash previewed no hand")
	}
	if blow.Hand.Key != "no-hand" {
		t.Errorf("one Bash previewed %q, want the no hand", blow.Hand.Key)
	}

	// **The planned name and the fired one are one spelling**, which is what lets the banner carry
	// the word through DUEL! instead of the dialog announcing it a second time.
	if got, want := handShout(blow.Hand.Name), "NO HAND!"; got != want {
		t.Errorf("one Bash is named %q, want %q", got, want)
	}
}

func TestAQueueOfPlansNamesAHandThatLandsNothing(t *testing.T) {
	// **A hand is what you played, not what you hit with** *(owner's call, 2026-08-23)*. Plans carry
	// an element and a form now, so two of them build a hand like anything else — this was
	// `TestAQueueWithNoAttackNamesNothing` and asserted the opposite until they joined.
	//
	// **The preview naming one is the point, not a leak.** A player queuing two Prepares is forming
	// a Form Pair; what they are not doing is dealing damage with it, and the two facts have to be
	// visible together or the multiplier looks like it went missing.
	s := selecting(
		combat.Of(combat.Brace, combat.Fire),
		combat.Of(combat.Block, combat.Ice),
	)

	blow, turn, ok := s.previewBlow()
	if !ok {
		t.Fatal("two plans previewed no hand at all")
	}
	if blow.Hand.Match != combat.AxisForm {
		t.Errorf("two plans of different concepts and colors formed a %v hand, want a form hand",
			blow.Hand.Match)
	}
	// The blow is real and worth nothing: `Card.Damage` is zero for every verb that is not an
	// attack, so the multiplier multiplies nothing. That is the accepted cost of plans joining
	// hands, and it is worth pinning rather than leaving to be rediscovered.
	base := 0
	for _, i := range blow.Cards {
		base += combat.Duelist{DMG: 10}.CardDamage(turn[i].Card)
	}
	if base != 0 {
		t.Errorf("a hand of plans carries %d damage, want 0", base)
	}
}

func TestAPlanQueuedLastDoesNotHideTheHandInFrontOfIt(t *testing.T) {
	// **`Blow.Cards` indexes the turn, which is in resolution order**, not the hand — a defense
	// picked *last* resolves first, so the pair sits at turn indices 1 and 2 while it sits in hand
	// slots 0 and 1. The preview goes through `ResolutionOrder` for exactly that reason, and a
	// preview built off the hand as the player left it would miss this hand entirely.
	//
	// **It was queued the other way until 2026-09-15**, when the phase order flipped — see
	// combat.Categories. The divergence this exists to pin is the same one, mirrored: whichever
	// category leads, the card queued into the *other* one moves.
	s := selecting(
		combat.Of(combat.Bash, combat.Fire),
		combat.Of(combat.Bash, combat.Ice),
		combat.Of(combat.Brace, combat.Basic),
	)

	blow, ok := s.previewAttack()
	if !ok {
		t.Fatal("a pair in front of a Prepare previewed no hand")
	}
	if blow.Hand.Key != "pair" {
		t.Errorf("a pair in front of a Prepare previewed %q, want the pair", blow.Hand.Key)
	}
	if !sameSeats(blow.Cards, []int{1, 2}) {
		t.Errorf("the previewed hand is turn slots %v, want the two Bashes at 1 and 2", blow.Cards)
	}
}

func TestThePreviewIsGoneOnceTheRoundIsRunning(t *testing.T) {
	// planning() is the single predicate for "the queue may still be edited", and a preview of a
	// round that is already resolving would be a proposal drawn over a record.
	s := selecting(
		combat.Of(combat.Bash, combat.Fire),
		combat.Of(combat.Bash, combat.Ice),
	)
	s.log = []combat.Event{{Kind: combat.KindRoundStart}}
	s.cursor = 0

	if _, ok := s.previewAttack(); ok {
		t.Error("the hand still previewed a hand while the round was playing back")
	}
}

func TestANewPlanArrivesWithNothingRaised(t *testing.T) {
	// A raised card means "this is firing now". The seat lists are written by playback, and a
	// round that ended with the opponent's second card up would leave it up under the *next*
	// plan — cards standing as though they had been committed, dropping again at DUEL!.
	s := &CombatScene{}
	s.enemyPile = decks.NewEnemyPile(testEnemyRecord, "", seeds.EnemyDeckPin, decks.EnemyHandSize)
	s.enemy = &entities.Combatant{
		Duelist: combat.Duelist{DMG: 5, Actions: 5, MaxLife: 60, CurrentLife: 60},
	}
	s.fighter = &entities.Combatant{
		Duelist: combat.Duelist{DMG: 10, Actions: 6, MaxLife: 60, CurrentLife: 60},
	}

	// Where the last round's playback left them.
	s.Theater.firingSeats = []int{0, 1}
	s.Theater.enemyFiringSeats = []int{1}

	s.planEnemyRound()

	if len(s.Theater.firingSeats) != 0 || len(s.Theater.enemyFiringSeats) != 0 {
		t.Errorf("the new plan arrived with %v and %v raised, want nothing lit",
			s.Theater.firingSeats, s.Theater.enemyFiringSeats)
	}
}

func TestADeadDuelistKeepsTheRoundThatKilledItOnTheTable(t *testing.T) {
	// The row stays on the table when a duel ends — it is the round the player is looking at
	// the result of — and nothing is drawn from a pile for a fight that is over.
	s := &CombatScene{}
	s.enemyPile = decks.NewEnemyPile(testEnemyRecord, "", seeds.EnemyDeckPin, decks.EnemyHandSize)
	s.enemy = &entities.Combatant{
		Duelist: combat.Duelist{DMG: 5, Actions: 5, MaxLife: 60, CurrentLife: 0},
	}
	s.fighter = &entities.Combatant{
		Duelist: combat.Duelist{DMG: 10, Actions: 6, MaxLife: 60, CurrentLife: 60},
	}

	// The killing blow is still raised, and stays raised.
	s.Theater.enemyFiringSeats = []int{0}

	s.planEnemyRound()

	if len(s.enemyActions) != 0 || len(s.Theater.enemyDealt) != 0 {
		t.Errorf("a dead opponent planned %v and seated %d cards", s.enemyActions, len(s.Theater.enemyDealt))
	}
	if len(s.Theater.enemyFiringSeats) != 1 {
		t.Errorf("the finished round was cleared off the table: %v", s.Theater.enemyFiringSeats)
	}
}

// testEnemyRecord is the roster entry the table tests deal from. **A named record rather than the
// first one sorted**, so a change to the roster's order does not silently change which deck these
// tests are exercising — and an outer-chamber creature of an early motif, so the hand it draws is
// small and cheap.
const testEnemyRecord = "slimes-outer-clear"

// **A defense that already flew its pips does not rise again.** The engine resolves defenses at
// the end of the turn, several beats after the hand they were scored into — so with the pips
// leaving on the beat the card is scored, a second lift on the card's own announcement reads as
// the card firing twice. A defense that flew nothing still lifts: that is the only thing on screen
// saying which one is going up.
func TestADefenseThatAlreadyFlewDoesNotRiseAgain(t *testing.T) {
	ward, ok := combat.ConceptByKey("ward")
	if !ok {
		t.Skip("no ward concept in this build")
	}

	newScene := func() *CombatScene {
		s := &CombatScene{
			fighterActions: []combat.Card{combat.Plain(combat.Bash), combat.Plain(ward)},
			log: []combat.Event{
				{Kind: combat.KindAction, Side: combat.SideA, Action: combat.Bash},
				{Kind: combat.KindAction, Side: combat.SideA, Action: ward},
			},
		}
		s.seatPlayedCards()
		s.cursor = 0
		s.noteResolved(s.log[0])
		return s
	}

	// The ward was scored into the hand and its pips left with its figure: the attack set stays up
	// and the ward does not climb on its own beat.
	s := newScene()
	s.row(combat.SideA).NoteFlight(1)
	s.cursor = 1
	s.noteResolved(s.log[1])
	if !sameSeats(s.Theater.firingSeats, []int{0}) {
		t.Errorf("a ward whose pips already flew raised %v, want the attack seat alone",
			s.Theater.firingSeats)
	}

	// Nothing flew for this one — a turn of nothing but defenses forms no hand — so it lifts.
	s = newScene()
	s.cursor = 1
	s.noteResolved(s.log[1])
	if !sameSeats(s.Theater.firingSeats, []int{1}) {
		t.Errorf("a defense that flew nothing raised %v, want its own seat", s.Theater.firingSeats)
	}
}
