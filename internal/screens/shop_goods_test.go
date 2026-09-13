package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The two sealed goods: what a bag holds, where the row puts them, and the one thing about the
// dialog that would be a lock-up rather than a bug — a stage with nothing clickable in it.

func TestABagHoldsFourDifferentStones(t *testing.T) {
	gs := testRun()

	got := dealStones(gs)
	if len(got) != session.BagSize() {
		t.Fatalf("a bag holds %d stones, want %d", len(got), session.BagSize())
	}

	seen := map[string]bool{}
	for _, s := range got {
		if seen[s.Record] {
			t.Errorf("the bag holds %s twice, which is a seat spent saying nothing", s.Record)
		}
		seen[s.Record] = true
	}
}

func TestACanHoldsFourDifferentEssences(t *testing.T) {
	gs := testRun()

	got := dealVialEssences(gs)
	if len(got) != session.VialSize() {
		t.Fatalf("a vial holds %d essences, want %d", len(got), session.VialSize())
	}

	seen := map[string]bool{}
	for _, w := range got {
		if seen[w.Record] {
			t.Errorf("the vial holds %s twice", w.Record)
		}
		seen[w.Record] = true
	}
}

// **The same fight walks into the same shop**, which is the rule the whole per-fight stream table
// exists for: a defeat and a retry meet the same opponent, so they had better meet the same bag.
func TestTheSameFightOpensTheSameBag(t *testing.T) {
	gs := testRun()

	a, b := dealStones(gs), dealStones(gs)
	if len(a) != len(b) {
		t.Fatalf("two draws of one fight's bag are %d and %d long", len(a), len(b))
	}
	for i := range a {
		if a[i].Record != b[i].Record {
			t.Errorf("seat %d is %s then %s", i, a[i].Record, b[i].Record)
		}
	}
}

// **The bag is not the shelf and not the reward screen's essences.** Sharing a stream would make
// authoring a stone change which relics a run was offered — the exact failure internal/seeds exists
// to prevent, and one that nothing else would catch.
func TestTheBagAndTheCanDrawFromTheirOwnStreams(t *testing.T) {
	gs := testRun()

	shelf := shelfKeys(dealShelf(gs, shopRNG(gs, seeds.ShopStock)))
	before := append([]string(nil), shelf...)

	// Drawing a bag and a vial must not have consumed anything the shelf reads.
	dealStones(gs)
	dealVialEssences(gs)

	after := shelfKeys(dealShelf(gs, shopRNG(gs, seeds.ShopStock)))
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("the shelf changed after a bag was drawn: %v then %v", before, after)
		}
	}

	// And the vial is not the reward screen's offer: two draws off one stream would be the same
	// four essences in the same order.
	reward := dealEssences(gs)
	can := dealVialEssences(gs)
	same := len(reward) > 0
	for i := range reward {
		if i >= len(can) || can[i].Record != reward[i].Record {
			same = false
			break
		}
	}
	if same {
		t.Error("the vial opened with the same essences the reward screen offered, in the same order")
	}
}

// **A visit puts up two of the three packs**, and which two is a roll of its own — so what has to
// hold is that the pane always has two different ones in it, and that they sit in the pane the row
// solved for them.
func TestAVisitOffersTwoDifferentPacks(t *testing.T) {
	gs := testRun()

	for fight := 0; fight < 8; fight++ {
		gs.Run.JumpTo(fight, 20, 40, 60)
		offered := dealPacks(shopRNG(gs, seeds.PackOffer))

		if len(offered) != packsOffered {
			t.Fatalf("fight %d put up %d packs, wanted %d", fight, len(offered), packsOffered)
		}
		if offered[0] == offered[1] {
			t.Errorf("fight %d put the same pack in both seats", fight)
		}
		for _, kind := range offered {
			if kind == goodNone {
				t.Errorf("fight %d put an empty seat on the shelf", fight)
			}
		}
	}
}

// The pack seats are the pane's, so a good is drawn and clicked where the row solved for it.
func TestTheGoodsStandInTheirPane(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}
	s := &ShopScene{shelf: make([]shelfItem, shelfSize), offered: []goodKind{goodBag, goodVial}}

	bag, can := s.goodSlot(gs, goodBag), s.goodSlot(gs, goodVial)
	if bag.Empty() || can.Empty() {
		t.Fatal("an offered pack has no seat")
	}
	if bag.Min.X >= can.Min.X {
		t.Errorf("the bag at %d is not left of the vial at %d", bag.Min.X, can.Min.X)
	}
	if bag.Min.Y != s.shelfSlot(gs, 0).Min.Y {
		t.Errorf("the packs sit at %d and the relics at %d, so the shelf is not one row",
			bag.Min.Y, s.shelfSlot(gs, 0).Min.Y)
	}

	// A pack this visit did not put up has no seat at all, which is what stops a click landing on
	// one that is not drawn.
	if got := s.goodSlot(gs, goodBucket); !got.Empty() {
		t.Errorf("a pack that was not offered has a seat at %v", got)
	}
}

// **A dialog with nothing clickable in it is a lock-up**, and it is the one failure this feature
// can have that is worse than doing nothing: the only exit is a card.
func TestADialogAlwaysHasSomethingToClick(t *testing.T) {
	gs := testRun()

	var g goods
	g.open(gs, goodBag)
	if g.count() == 0 {
		t.Error("a bag opened with nothing in it")
	}
	g.reset()

	g.open(gs, goodVial)
	if g.count() == 0 {
		t.Fatal("a vial opened with no essences in it")
	}
	if len(g.offer) == 0 {
		t.Fatal("a vial opened with no cards to aim at")
	}
	g.reset()

	g.open(gs, goodBucket)
	if g.count() == 0 {
		t.Error("a bucket opened with nothing in it")
	}
}

// **Both rows are up at once and the card is chosen first**, which is the gesture every consumable
// in the game now shares. What has to hold is that an essence is dead until a card it can change is
// selected — an essence that looked available and did nothing is the failure this predicate exists for.
func TestAEssenceIsDeadUntilACardIsSelected(t *testing.T) {
	gs := testRun()

	var g goods
	g.open(gs, goodVial)

	for _, w := range g.essences {
		if g.essenceSpendable(gs, w) {
			t.Errorf("%s was spendable with nothing selected", w.Record)
		}
	}

	// With a card picked, an essence is live exactly when the run says it can change that card.
	g.selectCard(0)
	idx, ok := g.selectedDeckIndex()
	if !ok {
		t.Fatal("selecting the first offered card left no deck index")
	}
	for _, w := range g.essences {
		if got, want := g.essenceSpendable(gs, w), gs.Run.CanApply(w, idx); got != want {
			t.Errorf("%s reads spendable=%v against CanApply=%v", w.Record, got, want)
		}
	}

	// Clicking the selected card again puts it back, and the row goes dead with it.
	g.selectCard(0)
	if _, ok := g.selectedDeckIndex(); ok {
		t.Error("clicking the selected card again left it selected")
	}
}

// An essence taken from the vial eats the card that was selected, and nothing else.
func TestTheCanAppliesTheEssenceToTheSelectedCard(t *testing.T) {
	gs := testRun()

	var g goods
	g.open(gs, goodVial)
	g.selectCard(0)

	idx, ok := g.selectedDeckIndex()
	if !ok {
		t.Fatal("nothing selected")
	}

	pick := -1
	for i, w := range g.essences {
		if g.essenceSpendable(gs, w) {
			pick = i
			break
		}
	}
	if pick < 0 {
		t.Skip("no essence in this can can change the first offered card")
	}

	before, _ := gs.Run.Card(idx)
	size := gs.Run.Size()
	g.take(gs, pick)

	if g.openNow() {
		t.Error("the dialog is still up after an essence was taken")
	}
	after, still := gs.Run.Card(idx)
	if gs.Run.Size() == size && still && after == before {
		t.Error("the essence was taken and the card it was aimed at is unchanged")
	}
}

// A stone taken from the bag is used on the spot — the whole of owning one.
func TestTakingAStoneRaisesTheRunsRung(t *testing.T) {
	gs := testRun()

	var g goods
	g.open(gs, goodBag)
	stone := g.stones[0]

	before, _ := gs.Run.HandMultiplier(stone.Hand)
	g.take(gs, 0)

	after, _ := gs.Run.HandMultiplier(stone.Hand)
	if want := before + session.StoneWorth(stone.Hand); after != want {
		t.Errorf("%s pays %d after the stone, want %d", stone.Hand, after, want)
	}
	if g.openNow() {
		t.Error("the dialog is still up after the stone was taken")
	}
}

// Both of the vial's rows have to fit inside the modal panel, with the lift a selected card takes.
// **The failure this exists for shipped once**: the offer row ended exactly on the panel's bottom
// edge, which was survivable while it was the only row on screen and was not once the essences stayed
// up beside it.
func TestTheCansTwoRowsFitInsideThePanel(t *testing.T) {
	gs := testRun()
	gs.ScreenWidth, gs.ScreenHeight = state.ScreenWidth, state.ScreenHeight

	var g goods
	g.open(gs, goodVial)

	panel := modalPanelRect(gs)
	essences := g.slot(gs, 0)
	if essences.Min.Y < panel.Min.Y+goodsHintTop {
		t.Errorf("the essences start at %d, over the hint at %d",
			essences.Min.Y, panel.Min.Y+goodsHintTop)
	}

	g.selectCard(0)
	offer := g.offerSlot(gs, 0)
	if offer.Min.Y < essences.Max.Y {
		t.Errorf("a lifted card starts at %d and the essences end at %d", offer.Min.Y, essences.Max.Y)
	}
	if offer.Max.Y > panel.Max.Y {
		t.Errorf("the offer row ends at %d, past the panel's %d", offer.Max.Y, panel.Max.Y)
	}
}
