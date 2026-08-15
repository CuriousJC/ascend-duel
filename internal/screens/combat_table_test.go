package screens

import (
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
	"github.com/curiousjc/ascend-duel/internal/entities"
)

// The table's geometry, which is arithmetic and needs no window — the same narrow exception
// the other tests in this package take. Nothing here creates an ebiten.Image.
//
// What these guard is the one property the arrangement exists for: **two hands that face each
// other and never touch.** Five cards a side do not fit in a screen at full size, so the rows
// overlap within themselves; the moment they overlap *into* each other the picture stops being
// a confrontation and becomes one row of ten.

// tableSeats is the widest row the rules can produce. Asked of the rules rather than written
// down, because a ring raising the cap is an expected change and must not quietly break the
// layout instead of failing here.
func tableSeats() int { return combat.Duelist{}.MaxActions() }

func TestTheTwoHandsNeverReachEachOther(t *testing.T) {
	gs := testState()

	for n := 1; n <= tableSeats(); n++ {
		player := playedSeatAt(gs, n-1, n)
		enemy := enemySeatAt(gs, 0, n)

		playerRight := player.X + cardWidth
		if playerRight >= enemy.X {
			t.Errorf("%d cards a side: the player's row ends at x=%d and the opponent's starts at x=%d",
				n, playerRight, enemy.X)
		}
	}
}

func TestEachRowIsPinnedToItsOwnEdge(t *testing.T) {
	gs := testState()

	for n := 1; n <= tableSeats(); n++ {
		// The player's grows rightward from the left inset.
		if got := playedSeatAt(gs, 0, n).X; got != tableInset {
			t.Errorf("%d cards: the player's row starts at x=%d, want the %dpx inset", n, got, tableInset)
		}

		// The opponent's is right-aligned, so its *last* card is flush with the right inset
		// whatever the count. That is what makes a hand of two hug its own edge rather than
		// drift toward the middle.
		last := enemySeatAt(gs, n-1, n).X + cardWidth
		if want := gs.ScreenWidth - tableInset; last != want {
			t.Errorf("%d cards: the opponent's row ends at x=%d, want %d", n, last, want)
		}
	}
}

func TestTheTableSitsBetweenTheRingRowAndTheFeed(t *testing.T) {
	gs := testState()
	s := &CombatScene{}

	top := tableRowTop(gs)

	// Below the top row, whose lowest ink is the ring count under the rule.
	ringBottom := s.ringPaneRect(gs).Max.Y + ringRuleWidth + ringCountTopGap + ringCountSize
	if top < ringBottom {
		t.Errorf("the table starts at y=%d, into the ring row's count ending at y=%d", top, ringBottom)
	}

	// And clear of the Resolution feed, which is the one thing a played card may never cover.
	feedTop := gs.PctY(handTopPct) - feedGapAboveCards - feedHeight()
	if bottom := top + cardHeight; bottom > feedTop {
		t.Errorf("the table ends at y=%d, into the Resolution feed at y=%d", bottom, feedTop)
	}
}

func TestTheOpponentsRowIsInResolutionOrder(t *testing.T) {
	// **The row must say what will happen, not what was planned.** ResolutionOrder regroups a
	// turn into prepare, then attacks, then defenses, so a queue planned attack-first comes out
	// of the planner in one order and resolves in another.
	s := &CombatScene{
		enemyActions: combat.PlainCards(combat.Strike, combat.Gather, combat.Brace),
	}

	got := s.enemyQueueOrder()
	want := combat.PlainCards(combat.Gather, combat.Strike, combat.Brace)

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

	const total = 4
	first := playedSeatAt(gs, 0, total)
	for n := 1; n < total; n++ {
		if got := playedSeatAt(gs, 0, total); got != first {
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
			{actionCard: actionCard{Action: combat.Strike, Element: combat.Fire}, selected: true},
			{actionCard: actionCard{Action: combat.Gather, Element: combat.Ice}, selected: true},
			{actionCard: actionCard{Action: combat.Brace, Element: combat.Earth}, selected: true},
		},
		fighterActions: []combat.Card{
			combat.Of(combat.Strike, combat.Fire),
			combat.Of(combat.Gather, combat.Ice),
			combat.Of(combat.Brace, combat.Earth),
		},
	}
	s.seatPlayedCards()

	// Prepare first, then the attack, then the defense — and each seat holds the card the
	// player actually selected for it, not the one in the same position in the hand.
	// The elements come along, so a seat holding the right concept in the wrong colour fails
	// too — which is the whole reason the hand and the queue are one type now.
	want := []combat.Card{
		combat.Of(combat.Gather, combat.Ice),
		combat.Of(combat.Strike, combat.Fire),
		combat.Of(combat.Brace, combat.Earth),
	}
	if len(s.resolved) != len(want) {
		t.Fatalf("%d cards were seated, want %d", len(s.resolved), len(want))
	}
	for i, c := range want {
		if got := s.resolved[i].card; got != c {
			t.Errorf("seat %d holds %v, want %v", i, got, c)
		}
	}

	// And every seat knows which hand slot it came from, which is what the end-of-round throw
	// and the hand row's own hiding both read.
	for i, r := range s.resolved {
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

	player := playedSeatAt(gs, 1, 3)
	enemy := enemySeatAt(gs, 1, 3)

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

func TestOnlyOneSideOfTheTableIsLitAtATime(t *testing.T) {
	// A turn is contiguous per side, so the lit cards walk the left row and then the right
	// one. The event that lights one side is the event that unlights the other, which is why
	// neither row has to know the other exists.
	s := &CombatScene{
		fighterActions: combat.PlainCards(combat.Strike),
		enemyActions:   combat.PlainCards(combat.Jab),
		log: []combat.Event{
			{Kind: combat.KindAction, Side: combat.SideA, Action: combat.Strike},
			{Kind: combat.KindAction, Side: combat.SideB, Action: combat.Jab},
		},
	}

	// The cursor points at the event being applied, not past it — advancePlayback calls
	// applyEvent before it increments. currentSlot counts inclusively for that reason.
	s.cursor = 0
	s.noteResolved(s.log[0])
	if !sameSeats(s.firingSeats, []int{0}) || len(s.enemyFiringSeats) != 0 {
		t.Errorf("after the player's card: player %v, enemy %v — want [0] and none",
			s.firingSeats, s.enemyFiringSeats)
	}

	s.cursor = 1
	s.noteResolved(s.log[1])
	if len(s.firingSeats) != 0 || !sameSeats(s.enemyFiringSeats, []int{0}) {
		t.Errorf("after the opponent's card: player %v, enemy %v — want none and [0]",
			s.firingSeats, s.enemyFiringSeats)
	}
}

func TestTheWholeAttackHandIsRaisedAndTheComboKeepsWhatEarnedIt(t *testing.T) {
	// **A turn lands one blow, so the whole hand goes up on the first announcement** — not one
	// card per beat, which read as one attack per card. The combo then drops the ones that earned
	// nothing, so what is left standing is what the feed's single line is about.
	s := &CombatScene{
		hand: []paletteCard{
			{actionCard: actionCard{Action: combat.Strike, Element: combat.Fire}, selected: true},
			{actionCard: actionCard{Action: combat.Strike, Element: combat.Ice}, selected: true},
			{actionCard: actionCard{Action: combat.Jab, Element: combat.Basic}, selected: true},
		},
		fighterActions: []combat.Card{
			combat.Of(combat.Strike, combat.Fire),
			combat.Of(combat.Strike, combat.Ice),
			combat.Of(combat.Jab, combat.Basic),
		},
		log: []combat.Event{
			{Kind: combat.KindAction, Side: combat.SideA, Action: combat.Strike},
			{Kind: combat.KindAction, Side: combat.SideA, Action: combat.Strike},
			{Kind: combat.KindAction, Side: combat.SideA, Action: combat.Jab},
		},
	}
	s.seatPlayedCards()

	// One announcement, the whole hand up. The beats that follow say how long the phase takes,
	// not which card is acting — no single card is.
	s.cursor = 0
	s.noteResolved(s.log[0])
	if !sameSeats(s.firingSeats, []int{0, 1, 2}) {
		t.Errorf("the first announcement raised %v, want all three cards up at once", s.firingSeats)
	}

	// And the rest of the phase names the same set rather than adding to it.
	for i := 1; i < len(s.log); i++ {
		s.cursor = i
		s.noteResolved(s.log[i])
	}
	if !sameSeats(s.firingSeats, []int{0, 1, 2}) {
		t.Errorf("the attack phase ended with %v raised, want all three cards up", s.firingSeats)
	}

	// The Jab built no hand, so the combo takes it back down — and brackets only the pair.
	combo := combat.Event{Kind: combat.KindCombo, Side: combat.SideA, ComboCardCount: 2}
	combo.ComboCards[0], combo.ComboCards[1] = 0, 1
	s.noteCombo(combo)

	if !sameSeats(s.firingSeats, []int{0, 1}) {
		t.Errorf("the combo left %v raised, want only the two cards that formed it", s.firingSeats)
	}
	if !s.resolved[0].combo || !s.resolved[1].combo || s.resolved[2].combo {
		t.Errorf("bracketed %v, want the first two only",
			[]bool{s.resolved[0].combo, s.resolved[1].combo, s.resolved[2].combo})
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

	r := resolvedCard{travel: newTravel(0, riseTicks), handIndex: 2, handCount: handSize}
	from := slotAt(gs, 2, handSize)
	to := playedSeatAt(gs, 1, 3)

	if got := r.at(gs, 1, 3, false); got != from {
		t.Errorf("a card that has not set off is at %v, want its hand slot %v", got, from)
	}

	r.age = riseTicks
	if got := r.at(gs, 1, 3, false); got != to {
		t.Errorf("a landed card is at %v, want its seat %v", got, to)
	}

	// Lifted while it resolves, and only once it has landed — a card still arriving is already
	// the most moving thing on screen.
	if got := r.at(gs, 1, 3, true); got.Y != to.Y-tableFireLift {
		t.Errorf("a firing card sits at y=%d, want %d", got.Y, to.Y-tableFireLift)
	}
	r.age = 0
	if got := r.at(gs, 1, 3, true); got != from {
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
		enemyActions: combat.PlainCards(combat.Strike, combat.Gather, combat.Brace),
	}
	s.seatEnemyCards()

	want := combat.PlainCards(combat.Gather, combat.Strike, combat.Brace)
	if len(s.enemyDealt) != len(want) {
		t.Fatalf("%d cards were seated, want %d", len(s.enemyDealt), len(want))
	}
	for i, c := range want {
		if got := s.enemyDealt[i].card; got != c {
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

	d := dealtCard{travel: newTravel(0, riseTicks)}
	from := s.enemyCardRect(gs).Min
	to := enemySeatAt(gs, 1, 3)

	if got := s.enemyCardAt(gs, d, 1, 3, false); got != from {
		t.Errorf("a card that has not set off is at %v, want the enemy card at %v", got, from)
	}

	d.age = riseTicks
	if got := s.enemyCardAt(gs, d, 1, 3, false); got != to {
		t.Errorf("a landed card is at %v, want its seat %v", got, to)
	}

	// Lifted only once it has landed, exactly as the player's row does it.
	if got := s.enemyCardAt(gs, d, 1, 3, true); got.Y != to.Y-tableFireLift {
		t.Errorf("a firing card sits at y=%d, want %d", got.Y, to.Y-tableFireLift)
	}
	d.age = 0
	if got := s.enemyCardAt(gs, d, 1, 3, false); got != from {
		t.Errorf("a card back on the pad is at %v, want %v", got, from)
	}
}

func TestBothRowsUseTheSameArrivalClock(t *testing.T) {
	// The two sides deal at the same speed and stagger the same way, or the table reads as one
	// row arriving and one row appearing. Both take riseTicks and flightStaggerPer from the
	// same constants, and both count with the same travel — this is what stops a later change
	// to one of them being made twice.
	s := &CombatScene{
		hand: []paletteCard{
			{actionCard: combat.Plain(combat.Strike), selected: true},
			{actionCard: combat.Plain(combat.Jab), selected: true},
		},
		fighterActions: combat.PlainCards(combat.Strike, combat.Jab),
		enemyActions:   combat.PlainCards(combat.Strike, combat.Jab),
	}
	s.seatPlayedCards()
	s.seatEnemyCards()

	if len(s.resolved) != len(s.enemyDealt) {
		t.Fatalf("%d player seats against %d enemy seats", len(s.resolved), len(s.enemyDealt))
	}
	for i := range s.resolved {
		if got, want := s.enemyDealt[i].travel, s.resolved[i].travel; got != want {
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
	s.enemyPile = decks.NewEnemyPile(decks.EnemySeed, decks.EnemyHandSize)
	s.enemy = &entities.Combatant{
		Duelist: combat.Duelist{Str: 5, Spd: 10, MaxLife: 60, CurrentLife: 60},
		Style:   combat.StyleBrute,
	}
	s.fighter = &entities.Combatant{
		Duelist: combat.Duelist{Str: 10, Spd: 20, MaxLife: 60, CurrentLife: 60},
	}

	s.planEnemyRound()

	planned := append([]combat.Card(nil), s.enemyActions...)
	if len(planned) == 0 {
		t.Fatal("the opponent planned nothing to look at")
	}

	// What is on the table is what was planned, in resolution order.
	if len(s.enemyDealt) != len(planned) {
		t.Fatalf("%d cards on the table against a plan of %d", len(s.enemyDealt), len(planned))
	}
	for i, c := range s.enemyQueueOrder() {
		if got := s.enemyDealt[i].card; got != c {
			t.Errorf("seat %d holds %v, want %v", i, got, c)
		}
	}
}

// selecting builds a scene whose hand is these cards, all selected, with the queue derived from
// them the way syncQueue does. The player is left alive with no log, which is what planning() is.
func selecting(cards ...combat.Card) *CombatScene {
	s := &CombatScene{
		fighter: &entities.Combatant{
			Duelist: combat.Duelist{Str: 10, Spd: 10, MaxLife: 60, CurrentLife: 60},
		},
		enemy: &entities.Combatant{
			Duelist: combat.Duelist{Str: 5, Spd: 10, MaxLife: 60, CurrentLife: 60},
		},
	}
	for _, c := range cards {
		s.hand = append(s.hand, paletteCard{actionCard: c, selected: true})
		s.fighterActions = append(s.fighterActions, c)
	}
	return s
}

func TestAHandPreviewsTheMomentItIsSelected(t *testing.T) {
	// **The preview is the resolver's own answer**, so what is ringed while choosing is what
	// fires. Three Strikes are a flurry the instant the third is picked, not when DUEL! is pressed.
	s := selecting(
		combat.Of(combat.Strike, combat.Fire),
		combat.Of(combat.Strike, combat.Ice),
		combat.Of(combat.Strike, combat.Basic),
	)

	blow, ok := s.previewAttack()
	if !ok {
		t.Fatal("three Strikes previewed no combo")
	}
	if len(s.previewHandSlots(blow)) != 3 {
		t.Errorf("the preview rings %v, want all three cards", s.previewHandSlots(blow))
	}

	// And the feed says so in the same words the fired line will use.
	rows := s.planningLines()
	if len(rows) != 2 || rows[0].swatch != comboSwatch {
		t.Fatalf("the pane holds %d rows while planning, want the combo line and the prompt", len(rows))
	}
	if want := "COMBO! " + comboNameOf(blow.Hand, blow.Mix); !strings.HasPrefix(rows[0].prefix, want) {
		t.Errorf("the preview reads %q, want it to open %q", rows[0].prefix, want)
	}
}

func TestOneAttackIsNotACombo(t *testing.T) {
	// A lone attack forms no hand, and announcing COMBO! over one Strike would empty the word.
	s := selecting(combat.Of(combat.Strike, combat.Fire))

	if _, ok := s.previewAttack(); ok {
		t.Error("one Strike previewed a combo")
	}
	if rows := s.planningLines(); len(rows) != 1 {
		t.Errorf("the pane holds %d rows, want the prompt alone", len(rows))
	}
}

func TestThePreviewNamesTheCardsInTheHandNotInTheQueue(t *testing.T) {
	// **Attack.Cards indexes the turn, which is in resolution order**; the hand is in whatever
	// order the player left it. A Gather queued first resolves first, so the pair the preview
	// rings is hand slots 1 and 2 — reading the turn's indices straight would ring slots 0 and 1.
	s := selecting(
		combat.Of(combat.Gather, combat.Basic),
		combat.Of(combat.Strike, combat.Fire),
		combat.Of(combat.Strike, combat.Ice),
	)

	blow, ok := s.previewAttack()
	if !ok {
		t.Fatal("a pair behind a Gather previewed no combo")
	}
	if got := s.previewHandSlots(blow); !sameSeats(got, []int{1, 2}) {
		t.Errorf("the preview rings hand slots %v, want the two Strikes at 1 and 2", got)
	}
}

func TestThePreviewIsGoneOnceTheRoundIsRunning(t *testing.T) {
	// planning() is the single predicate for "the queue may still be edited", and a preview of a
	// round that is already resolving would be a proposal drawn over a record.
	s := selecting(
		combat.Of(combat.Strike, combat.Fire),
		combat.Of(combat.Strike, combat.Ice),
	)
	s.log = []combat.Event{{Kind: combat.KindRoundStart}}
	s.cursor = 0

	if _, ok := s.previewAttack(); ok {
		t.Error("the hand still previewed a combo while the round was playing back")
	}
}

func TestANewPlanArrivesWithNothingRaised(t *testing.T) {
	// A raised card means "this is firing now". The seat lists are written by playback, and a
	// round that ended with the opponent's second card up would leave it up under the *next*
	// plan — cards standing as though they had been committed, dropping again at DUEL!.
	s := &CombatScene{}
	s.enemyPile = decks.NewEnemyPile(decks.EnemySeed, decks.EnemyHandSize)
	s.enemy = &entities.Combatant{
		Duelist: combat.Duelist{Str: 5, Spd: 10, MaxLife: 60, CurrentLife: 60},
		Style:   combat.StyleBrute,
	}
	s.fighter = &entities.Combatant{
		Duelist: combat.Duelist{Str: 10, Spd: 20, MaxLife: 60, CurrentLife: 60},
	}

	// Where the last round's playback left them.
	s.firingSeats = []int{0, 1}
	s.enemyFiringSeats = []int{1}

	s.planEnemyRound()

	if len(s.firingSeats) != 0 || len(s.enemyFiringSeats) != 0 {
		t.Errorf("the new plan arrived with %v and %v raised, want nothing lit",
			s.firingSeats, s.enemyFiringSeats)
	}
}

func TestADeadDuelistKeepsTheRoundThatKilledItOnTheTable(t *testing.T) {
	// The row stays on the table when a duel ends — it is the round the player is looking at
	// the result of — and nothing is drawn from a pile for a fight that is over.
	s := &CombatScene{}
	s.enemyPile = decks.NewEnemyPile(decks.EnemySeed, decks.EnemyHandSize)
	s.enemy = &entities.Combatant{
		Duelist: combat.Duelist{Str: 5, Spd: 10, MaxLife: 60, CurrentLife: 0},
		Style:   combat.StyleBrute,
	}
	s.fighter = &entities.Combatant{
		Duelist: combat.Duelist{Str: 10, Spd: 20, MaxLife: 60, CurrentLife: 60},
	}

	// The killing blow is still raised, and stays raised.
	s.enemyFiringSeats = []int{0}

	s.planEnemyRound()

	if len(s.enemyActions) != 0 || len(s.enemyDealt) != 0 {
		t.Errorf("a dead opponent planned %v and seated %d cards", s.enemyActions, len(s.enemyDealt))
	}
	if len(s.enemyFiringSeats) != 1 {
		t.Errorf("the finished round was cleared off the table: %v", s.enemyFiringSeats)
	}
}
