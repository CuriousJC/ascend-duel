package screens

import (
	"testing"

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

func TestACanHoldsFourDifferentWorms(t *testing.T) {
	gs := testRun()

	got := dealCanWorms(gs)
	if len(got) != session.CanSize() {
		t.Fatalf("a can holds %d worms, want %d", len(got), session.CanSize())
	}

	seen := map[string]bool{}
	for _, w := range got {
		if seen[w.Record] {
			t.Errorf("the can holds %s twice", w.Record)
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

// **The bag is not the shelf and not the reward screen's worms.** Sharing a stream would make
// authoring a stone change which rings a run was offered — the exact failure internal/seeds exists
// to prevent, and one that nothing else would catch.
func TestTheBagAndTheCanDrawFromTheirOwnStreams(t *testing.T) {
	gs := testRun()

	shelf := shelfKeys(dealShelf(gs))
	before := append([]string(nil), shelf...)

	// Drawing a bag and a can must not have consumed anything the shelf reads.
	dealStones(gs)
	dealCanWorms(gs)

	after := shelfKeys(dealShelf(gs))
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("the shelf changed after a bag was drawn: %v then %v", before, after)
		}
	}

	// And the can is not the reward screen's offer: two draws off one stream would be the same
	// four worms in the same order.
	reward := dealWorms(gs)
	can := dealCanWorms(gs)
	same := len(reward) > 0
	for i := range reward {
		if i >= len(can) || can[i].Record != reward[i].Record {
			same = false
			break
		}
	}
	if same {
		t.Error("the can opened with the same worms the reward screen offered, in the same order")
	}
}

// The row is five full-size cards now and it still has to fit, with the two goods' price figures
// clearing the Leave button exactly as the rings' do.
func TestTheGoodsStandInTheShelfRowAndFit(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: 1280, ScreenHeight: 960}
	s := &ShopScene{shelf: make([]shelfItem, shelfSize)}

	bag, can := s.goodSlot(gs, goodBag), s.goodSlot(gs, goodCan)
	last := s.shelfSlot(gs, shelfSize-1)

	if bag.Min.X < last.Max.X {
		t.Errorf("the bag at %d overlaps the last ring ending at %d", bag.Min.X, last.Max.X)
	}
	if can.Min.X < bag.Max.X {
		t.Errorf("the can at %d overlaps the bag ending at %d", can.Min.X, bag.Max.X)
	}
	if can.Max.X > gs.ScreenWidth {
		t.Errorf("the row runs to %d, past the screen's %d", can.Max.X, gs.ScreenWidth)
	}
	if bag.Min.Y != last.Min.Y {
		t.Errorf("the bag sits at %d and the rings at %d, so the row is not one row",
			bag.Min.Y, last.Min.Y)
	}
	if bag.Max.Y+shopFigureGap+shopFigureSize >= gs.PctY(offerButtonsPct)-offerButtonHeight/2 {
		t.Error("the goods' prices run into the Leave button")
	}
}

// **A dialog with nothing clickable in it is a lock-up**, and it is the one failure this feature
// can have that is worse than doing nothing: the only exit is a card. A worm with no card it can
// change closes the dialog instead of showing an empty row.
func TestADialogAlwaysHasSomethingToClick(t *testing.T) {
	gs := testRun()

	var g goods
	g.open(gs, goodBag)
	if g.count() == 0 {
		t.Error("a bag opened with nothing in it")
	}
	g.reset()

	g.open(gs, goodCan)
	if g.count() == 0 {
		t.Fatal("a can opened with nothing in it")
	}

	// Every worm in the can either has a card it can change, or ends the dialog rather than
	// stranding it. Walking them all is what pins that the empty case is handled and not merely
	// unlikely.
	for i := range g.worms {
		var one goods
		one.open(gs, goodCan)
		one.take(gs, i)

		if one.stage == goodsAim && len(one.offer) == 0 {
			t.Errorf("%s left the dialog aiming at nothing", g.worms[i].Record)
		}
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
