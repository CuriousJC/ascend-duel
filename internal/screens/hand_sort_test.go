package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
)

// The hand's arrangement, which is checkable without a window for the same reason the deck
// overlay's is: it is a comparison over plain values, and only the drawing needs a screen.
//
// What these pin is the part a player would notice being wrong — that a mode's leading key is
// the one it is named after, that every mode falls through to the same secondary chain, and
// that the column of buttons stands clear of the cards it arranges.

// sortHandOf arranges a bare hand, which is all sortHand touches.
func sortHandOf(mode handSort, cards ...actionCard) []paletteCard {
	hand := make([]paletteCard, 0, len(cards))
	for _, c := range cards {
		hand = append(hand, paletteCard{actionCard: c})
	}
	s := &CombatScene{hand: hand, sortMode: mode}
	s.sortHand()
	return s.hand
}

func card(a combat.ConceptID, e combat.Element) actionCard {
	return actionCard{Concept: a, Element: e}
}

func TestCostIsTheDefaultSort(t *testing.T) {
	// The zero value, so a scene nobody has pressed a button on is already in the mode the
	// owner asked to be the default. A named constant that happened to be second in the enum
	// would make an unsorted-looking hand the out-of-the-box state.
	var s CombatScene
	if s.sortMode != sortByCost {
		t.Errorf("a fresh scene sorts by %v, want %v", s.sortMode, sortByCost)
	}
}

func TestCostSortRunsCheapestFirst(t *testing.T) {
	hand := sortHandOf(sortByCost,
		card(combat.Cleave, combat.Ice),    // 3
		card(combat.Prepare, combat.Basic), // 1
		card(combat.Slash, combat.Fire),    // 2
		card(combat.Jab, combat.Earth),     // 1
	)

	last := 0
	for _, c := range hand {
		if got := c.Cost(); got < last {
			t.Fatalf("hand runs %s, which drops from %d AP to %d", handLabel(hand), last, got)
		} else {
			last = got
		}
	}
}

func TestTypeSortPutsEveryAttackBeforeEveryPlan(t *testing.T) {
	hand := sortHandOf(sortByType,
		card(combat.Defend, combat.Basic),  // plan, 3
		card(combat.Cleave, combat.Ice),    // attack, 3
		card(combat.Prepare, combat.Basic), // plan, 1
		card(combat.Jab, combat.Fire),      // attack, 1
	)

	seenPlan := false
	for _, c := range hand {
		isPlan := c.Category() == combat.CategoryPlan
		if seenPlan && !isPlan {
			t.Fatalf("hand runs %s, which puts an attack after a plan", handLabel(hand))
		}
		seenPlan = seenPlan || isPlan
	}

	// And within each group it is the cost sort, which is the whole point of the chain: a
	// type sort is a cost sort with one key in front of it.
	if hand[0].Concept != combat.Jab || hand[1].Concept != combat.Cleave {
		t.Errorf("attacks run %s, want the cheaper one first", handLabel(hand[:2]))
	}
	if hand[2].Concept != combat.Prepare || hand[3].Concept != combat.Defend {
		t.Errorf("plans run %s, want the cheaper one first", handLabel(hand[2:]))
	}
}

func TestElementSortRunsFireIceLightningEarthThenDrab(t *testing.T) {
	// Deliberately not the enum's order — combat.Basic is the zero value and leads there,
	// and the whole reason elementRank is written out is that on screen it trails.
	want := []combat.Element{combat.Fire, combat.Ice, combat.Lightning, combat.Earth, combat.Basic}

	hand := sortHandOf(sortByElement,
		card(combat.Prepare, combat.Basic),
		card(combat.Jab, combat.Earth),
		card(combat.Jab, combat.Fire),
		card(combat.Jab, combat.Lightning),
		card(combat.Jab, combat.Ice),
	)

	for i, c := range hand {
		if c.Element != want[i] {
			t.Fatalf("hand runs %s, want %v", handLabel(hand), want)
		}
	}
}

func TestEveryElementHasItsOwnRank(t *testing.T) {
	// Element is append-only, so a fifth colour arrives with no rank and would silently join
	// the default arm — sorted level with basic and to the right of everything. Failing here
	// is what stops that shipping as "the new cards are in a funny order".
	seen := map[int]combat.Element{}
	for _, e := range combat.AllElements {
		r := elementRank(e)
		if r >= len(combat.AllElements) {
			t.Errorf("%v has no rank of its own and fell through to %d", e, r)
		}
		if other, dup := seen[r]; dup {
			t.Errorf("%v and %v both rank %d", e, other, r)
		}
		seen[r] = e
	}
}

func TestEverySortFallsThroughToTheSameChain(t *testing.T) {
	// Two cards a mode's leading key cannot separate must come out in the cost chain's order,
	// whichever mode is asking. That is what makes the three arrangements read alike: only the
	// first key differs.
	cheap, dear := card(combat.Jab, combat.Fire), card(combat.Lunge, combat.Fire)

	for _, mode := range []handSort{sortByCost, sortByType, sortByElement} {
		hand := sortHandOf(mode, dear, cheap)
		if hand[0].Concept != combat.Jab {
			t.Errorf("sorting by %v ran %s, want the cost chain to break the tie",
				mode, handLabel(hand))
		}
	}
}

func TestSortingKeepsIdenticalCardsInPlace(t *testing.T) {
	// Stable, because two identical cards are told apart on screen only by which one is
	// selected — and a selected card is lifted out of the row. An unstable sort would swap
	// them and show a card moving for no reason the player can see.
	hand := []paletteCard{
		{actionCard: card(combat.Jab, combat.Fire), selected: false},
		{actionCard: card(combat.Jab, combat.Fire), selected: true},
	}
	s := &CombatScene{hand: hand}
	s.sortHand()

	if s.hand[0].selected || !s.hand[1].selected {
		t.Error("sorting reordered two identical cards, moving the selected one")
	}
}

func TestSortingRebuildsTheQueueInTheNewOrder(t *testing.T) {
	// The hand is the authority on the queue's order as well as its membership, and
	// handIndexForQueue is the inverse of that walk. Sorting without resyncing would leave the
	// two disagreeing, and the visible symptom is a hand preview ringing the wrong cards.
	s := &CombatScene{hand: []paletteCard{
		{actionCard: card(combat.Cleave, combat.Fire), selected: true}, // 3 AP
		{actionCard: card(combat.Jab, combat.Fire), selected: true},    // 1 AP
	}}
	s.syncQueue()
	s.setSort(sortByCost)

	if len(s.fighterActions) != 2 || s.fighterActions[0].Concept != combat.Jab {
		t.Fatalf("queue is %v, want it rebuilt cheapest first from the sorted hand",
			s.fighterActions)
	}
	for n := range s.fighterActions {
		h, ok := s.handIndexForQueue(n)
		if !ok || s.hand[h].actionCard != s.fighterActions[n] {
			t.Errorf("queue position %d maps to hand slot %d, which holds a different card", n, h)
		}
	}
}

func TestARefilledHandComesBackSorted(t *testing.T) {
	// (a) of the three options: the sort re-applies on every refill, so a drawn card lands
	// where it belongs rather than on the right-hand end.
	s := flightScene(selectedHand(5, 2))
	s.deck = nil
	for i := 0; i < 30; i++ {
		// Dearer than the Jabs already in hand, so an unsorted refill is visible: the drawn
		// cards would sit past the cheap ones instead of being sorted in among them.
		s.deck = append(s.deck, card(combat.Cleave, combat.Fire))
	}
	s.spendSelected()

	last := 0
	for _, c := range s.hand {
		if got := c.Cost(); got < last {
			t.Fatalf("refilled hand runs %s, which is not in cost order", handLabel(s.hand))
		} else {
			last = got
		}
	}
}

func TestAnInboundFlightPointsAtTheSlotItsCardEndedIn(t *testing.T) {
	// The sort runs before the flights are raised, so a dealt card flies to the slot it will
	// actually occupy. If this drifts, cards land in the wrong gaps and inboundTo blanks a
	// slot that has a card in it — two visible faults from one index.
	s := flightScene(selectedHand(6, 3))
	s.deck = nil
	for i := 0; i < 30; i++ {
		s.deck = append(s.deck, card(combat.Prepare, combat.Basic))
	}
	s.spendSelected()

	for _, f := range s.flights {
		if f.outbound {
			continue
		}
		if f.index < 0 || f.index >= len(s.hand) {
			t.Fatalf("a dealt card flies to slot %d, which is not in a hand of %d",
				f.index, len(s.hand))
		}
		if s.hand[f.index].actionCard != f.card {
			t.Errorf("slot %d holds %s, but the card flying into it is %s",
				f.index, cardLabel(s.hand[f.index].actionCard), cardLabel(f.card))
		}
		if f.count != len(s.hand) {
			t.Errorf("inbound flight targets a row of %d, want the %d it joins",
				f.count, len(s.hand))
		}
	}
}

func TestTheSortReportsWhereEveryCardCameFrom(t *testing.T) {
	// The permutation is what tells a card where to slide from, and two identical cards cannot
	// be told apart afterwards by looking at them — so it has to come out of the sort itself.
	// A permutation is what this checks: every position accounted for, exactly once.
	s := &CombatScene{hand: []paletteCard{
		{actionCard: card(combat.Cleave, combat.Fire)}, // 3
		{actionCard: card(combat.Jab, combat.Fire)},    // 1
		{actionCard: card(combat.Slash, combat.Fire)},  // 2
	}}
	before := append([]paletteCard(nil), s.hand...)

	order := s.sortHand()
	if len(order) != len(before) {
		t.Fatalf("the sort reported %d positions for a hand of %d", len(order), len(before))
	}

	seen := map[int]bool{}
	for to, from := range order {
		if seen[from] {
			t.Fatalf("slot %d is reported as the origin of two cards", from)
		}
		seen[from] = true

		if s.hand[to] != before[from] {
			t.Errorf("position %d says it came from %d, which held a different card", to, from)
		}
	}
}

func TestSortingSendsEveryMovedCardSliding(t *testing.T) {
	s := &CombatScene{hand: []paletteCard{
		{actionCard: card(combat.Cleave, combat.Fire)}, // 3, moves to the right
		{actionCard: card(combat.Jab, combat.Fire)},    // 1, moves to the left
	}}
	s.setSort(sortByCost)

	if len(s.slides) != 2 {
		t.Fatalf("%d cards slid, want both of them", len(s.slides))
	}
	for _, sl := range s.slides {
		if sl.fromIndex == sl.toIndex {
			t.Errorf("a card slid from slot %d to itself", sl.fromIndex)
		}
		if s.hand[sl.toIndex].actionCard != sl.card {
			t.Errorf("slot %d holds %s, but the card sliding into it is %s",
				sl.toIndex, cardLabel(s.hand[sl.toIndex].actionCard), cardLabel(sl.card))
		}
		// Both ends are in the same row here, and neither may name a slot outside it —
		// slotAt would otherwise place the card off the end of the band.
		if sl.fromCount != len(s.hand) || sl.toCount != len(s.hand) {
			t.Errorf("a slide runs between rows of %d and %d, want %d",
				sl.fromCount, sl.toCount, len(s.hand))
		}
	}
}

func TestACardThatDoesNotMoveDoesNotSlide(t *testing.T) {
	// A sort that changes nothing must not send the whole hand travelling on the spot.
	s := &CombatScene{hand: []paletteCard{
		{actionCard: card(combat.Jab, combat.Fire)},
		{actionCard: card(combat.Cleave, combat.Fire)},
	}}
	s.setSort(sortByCost)

	if len(s.slides) != 0 {
		t.Errorf("%d cards slid over an already-sorted hand", len(s.slides))
	}
}

func TestASecondSortReplacesASlideForTheSameSlot(t *testing.T) {
	// Pressing another mode before the first has landed: two slides converging on one slot
	// would draw the card twice and keep the row's own copy suppressed until the later of
	// them finished.
	s := &CombatScene{hand: []paletteCard{
		{actionCard: card(combat.Cleave, combat.Fire)},
		{actionCard: card(combat.Jab, combat.Basic)},
	}}
	s.setSort(sortByCost)
	s.setSort(sortByElement)

	seen := map[int]bool{}
	for _, sl := range s.slides {
		if seen[sl.toIndex] {
			t.Errorf("two cards are sliding into slot %d", sl.toIndex)
		}
		seen[sl.toIndex] = true
	}
}

func TestASurvivingCardSlidesAsTheRowClosesUp(t *testing.T) {
	// A card nobody played still moves: the cards around it have gone and the row re-centres
	// under it. Both ends of the slide have to name their own row size, because the two differ.
	s := flightScene(selectedHand(5, 2))
	s.spendSelected()

	if len(s.hand) != handSize {
		t.Fatalf("hand holds %d cards, want it dealt back to %d", len(s.hand), handSize)
	}
	if len(s.slides) == 0 {
		t.Fatal("no card slid, though the row went from five cards to eight under them")
	}
	for _, sl := range s.slides {
		if sl.fromCount != 5 {
			t.Errorf("a slide sets off from a row of %d, want the 5 that was there", sl.fromCount)
		}
		if sl.toCount != len(s.hand) {
			t.Errorf("a slide lands in a row of %d, want the %d it joins", sl.toCount, len(s.hand))
		}
	}
}

// settledScene is a scene whose playback has just reached the end of a round that killed
// somebody. `after` is the life each side ends on.
func settledScene(fighterLife, enemyLife int) *CombatScene {
	s := flightScene(selectedHand(5, 2))
	s.fighter, s.enemy = &entities.Combatant{}, &entities.Combatant{}

	s.fighterAfter = combat.Duelist{MaxLife: 20, CurrentLife: fighterLife}
	s.enemyAfter = combat.Duelist{MaxLife: 20, CurrentLife: enemyLife}
	s.syncQueue()
	return s
}

func TestASettledDuelFreezesTheScreenAsItStands(t *testing.T) {
	// **The last frame of a fight must not reflow.** Spending the hand takes cards out of the
	// row, and the row is what half the lower screen is measured from — handBand is a function
	// of how many cards are in it, the AP bar spans that band, and the Resolution feed's bottom
	// edge comes off the same row. So a hand spent after the killing blow collapsed the cards
	// into a narrow centred huddle and dragged the bar and the feed with it.
	for _, tc := range []struct {
		name                   string
		fighterLife, enemyLife int
	}{
		{"a win", 12, 0},
		{"a defeat", 0, 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := settledScene(tc.fighterLife, tc.enemyLife)
			hand, deck, queue := len(s.hand), len(s.deck), len(s.fighterActions)

			s.endOfRound()

			if len(s.hand) != hand {
				t.Errorf("the hand went from %d cards to %d, want it frozen", hand, len(s.hand))
			}
			if len(s.deck) != deck {
				t.Errorf("the draw pile went from %d to %d, want it untouched", deck, len(s.deck))
			}
			if len(s.fighterActions) != queue {
				t.Errorf("the queue went from %d cards to %d, want the round still on screen",
					queue, len(s.fighterActions))
			}
			if len(s.flights)+len(s.slides) != 0 {
				t.Errorf("%d cards are moving, want nothing to move once the duel is over",
					len(s.flights)+len(s.slides))
			}

			// And it does adopt the end state, which is what duelSettled and the fighter cards
			// read — freezing the layout must not freeze the result.
			if s.fighter.CurrentLife != tc.fighterLife || s.enemy.CurrentLife != tc.enemyLife {
				t.Errorf("life ended at %d/%d, want %d/%d",
					s.fighter.CurrentLife, s.enemy.CurrentLife, tc.fighterLife, tc.enemyLife)
			}
			if !s.duelSettled() {
				t.Error("the duel does not report itself settled")
			}
		})
	}
}

func TestTheSortColumnStandsClearOfTheCards(t *testing.T) {
	gs := testState()
	col := sortColumnRect(gs)

	// Against the band's right edge, and inside the screen with it.
	if col.Max.X > gs.PctX(handBandRightPct) {
		t.Errorf("the sort column reaches x=%d, past the band's right edge at %d",
			col.Max.X, gs.PctX(handBandRightPct))
	}

	// **Checked at more than one hand size**, because the row is centred on the space the
	// column leaves: a bigger hand is a wider row, and the failure this guards is the last
	// card sliding under the buttons.
	for _, n := range []int{1, 3, handSize, handSize + 2, 14} {
		band := handBand(gs, n)
		if band.Max.X+sortColumnGap > col.Min.X {
			t.Errorf("a hand of %d reaches x=%d, leaving less than %dpx before the column at %d",
				n, band.Max.X, sortColumnGap, col.Min.X)
		}
		if band.Min.X < gs.PctX(handBandLeftPct) {
			t.Errorf("a hand of %d starts at x=%d, left of the band's edge at %d",
				n, band.Min.X, gs.PctX(handBandLeftPct))
		}
	}
}

func TestTheSortColumnIsCentredOnTheCards(t *testing.T) {
	// It is centred on the cards rather than on the screen or the band, which is what the
	// owner asked for and also what keeps it reading as belonging to the row. Measured off
	// the row's own top and height, so it follows if handTopPct moves again — it has three
	// times.
	gs := testState()
	col := sortColumnRect(gs)

	cardsMid := gs.PctY(handTopPct) + cardHeight/2
	if mid := (col.Min.Y + col.Max.Y) / 2; abs(mid-cardsMid) > 1 {
		t.Errorf("the column is centred at y=%d, want the cards' centre at %d", mid, cardsMid)
	}
}

func TestThereIsOneButtonPerSortMode(t *testing.T) {
	// The column and the modes are two lists that have to stay the same length: updateSortButtons
	// indexes one by the other to decide which button is latched, so a mode with no button
	// would be unreachable and a button with no mode would panic.
	seen := map[handSort]bool{}
	for _, spec := range sortButtonSpecs {
		if seen[spec.mode] {
			t.Errorf("two buttons both select %v", spec.mode)
		}
		seen[spec.mode] = true

		if spec.label == "" {
			t.Errorf("%v has no symbol on its button", spec.mode)
		}
	}
	for _, mode := range []handSort{sortByCost, sortByType, sortByElement} {
		if !seen[mode] {
			t.Errorf("%v has no button to select it", mode)
		}
	}
}
