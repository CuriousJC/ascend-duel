package screens

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The sealed goods: what a bag holds, where the row puts them, and the one thing about the
// dialog that would be a lock-up rather than a bug — a stage with nothing clickable in it.
//
// **The three are looked up rather than named** *(2026-09-14)*, since the catalog is
// data/goods.json: a test naming "bag" would fail on a renamed record instead of on a broken shop.

// goodHolding is the record that holds one catalog, or a fatal — every test below wants one and
// none of them can say anything useful without it.
func goodHolding(t *testing.T, c session.GoodContents) session.Good {
	t.Helper()
	g, ok := session.GoodHolding(c)
	if !ok {
		t.Fatalf("no good in goods.json holds %s", c.Noun())
	}
	return g
}

func TestABagHoldsFourDifferentStones(t *testing.T) {
	gs := testRun()

	bag := goodHolding(t, session.ContentsStones)

	got := dealStones(gs, bag.Record, bag.Size)
	if len(got) != bag.Size {
		t.Fatalf("a bag holds %d stones, want %d", len(got), bag.Size)
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

	vial := goodHolding(t, session.ContentsEssences)

	got := dealVialEssences(gs, vial.Record, vial.Size)
	if len(got) != vial.Size {
		t.Fatalf("a vial holds %d essences, want %d", len(got), vial.Size)
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

	bag := goodHolding(t, session.ContentsStones)

	a, b := dealStones(gs, bag.Record, bag.Size), dealStones(gs, bag.Record, bag.Size)
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

	bag := goodHolding(t, session.ContentsStones)
	vial := goodHolding(t, session.ContentsEssences)

	// Drawing a bag and a vial must not have consumed anything the shelf reads.
	dealStones(gs, bag.Record, bag.Size)
	dealVialEssences(gs, vial.Record, vial.Size)

	after := shelfKeys(dealShelf(gs, shopRNG(gs, seeds.ShopStock)))
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("the shelf changed after a bag was drawn: %v then %v", before, after)
		}
	}

	// And the vial is not the reward screen's offer: two draws off one stream would be the same
	// four essences in the same order.
	reward := dealEssences(gs)
	can := dealVialEssences(gs, vial.Record, vial.Size)
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
		for _, key := range offered {
			if _, ok := session.GoodByKey(key); !ok {
				t.Errorf("fight %d put %q on the shelf, which is no record", fight, key)
			}
		}
	}
}

// The pack seats are the pane's, so a good is drawn and clicked where the row solved for it.
func TestTheGoodsStandInTheirPane(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}
	bagKey := goodHolding(t, session.ContentsStones).Record
	vialKey := goodHolding(t, session.ContentsEssences).Record
	sackKey := goodHolding(t, session.ContentsRunes).Record
	s := &ShopScene{shelf: make([]shelfItem, shelfSize), offered: []string{bagKey, vialKey}}

	bag, can := s.goodSlot(gs, bagKey), s.goodSlot(gs, vialKey)
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
	if got := s.goodSlot(gs, sackKey); !got.Empty() {
		t.Errorf("a pack that was not offered has a seat at %v", got)
	}
}

// **A dialog with nothing clickable in it is a lock-up**, and it is the one failure this feature
// can have that is worse than doing nothing: the only exit is a card.
func TestADialogAlwaysHasSomethingToClick(t *testing.T) {
	gs := testRun()

	var g goods
	for _, good := range session.Goods() {
		g.open(gs, good)
		if g.count() == 0 {
			t.Errorf("%s opened with nothing in it", good.Record)
		}
		if good.Contains == session.ContentsEssences && len(g.offer) == 0 {
			t.Errorf("%s opened with no cards to aim at", good.Record)
		}
		g.reset()
	}
}

// Every good says the same thing in its dialog and in its tooltip, and every figure in both is the
// record's own. **A price quoted in one place and charged in another is the failure this catches**,
// and it is the reason none of those strings is authored.
//
// **The face is not one of the places any more** *(owner's call, 2026-09-15)*: a sealed good is a
// picture and a price, and everything it used to write across its lower half is in the tooltip. So
// the count and the noun are checked there, which is where a player now reads them.
func TestAGoodSaysTheSameThingEverywhere(t *testing.T) {
	for _, good := range session.Goods() {
		if good.Title == "" || good.Hint == "" {
			t.Errorf("%s opens a dialog with no title or no hint", good.Record)
		}

		title, lines := goodTip(good)
		if title != good.Name {
			t.Errorf("%s is headed %q on the shelf and %q in its tooltip", good.Record, good.Name, title)
		}
		got := strings.Join(lines, " ")
		if want := fmt.Sprintf("%d %s", good.Size, good.Contains.Noun()); !strings.Contains(got, want) {
			t.Errorf("%s holds %q and its tooltip says %q", good.Record, want, got)
		}
		if !strings.Contains(got, fmt.Sprintf("%d vitae", good.Price)) {
			t.Errorf("%s costs %d and its tooltip says %q", good.Record, good.Price, got)
		}
	}
}

// **Both rows are up at once and the card is chosen first**, which is the gesture every consumable
// in the game now shares. What has to hold is that an essence is dead until a card it can change is
// selected — an essence that looked available and did nothing is the failure this predicate exists for.
func TestAEssenceIsDeadUntilACardIsSelected(t *testing.T) {
	gs := testRun()

	var g goods
	g.open(gs, goodHolding(t, session.ContentsEssences))

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

// An essence taken from the vial shows what it did before it lands: the dialog stays up through the
// flight, the change and the hold, and the deck is not touched until that is over.
//
// **This is the reward screen's rule, and it arrived here on 2026-09-16** — the vial used to apply
// on the click and close, so the alteration happened between two frames.
// An essence taken from the vial eats the card that was selected, and nothing else.
func TestTheCanAppliesTheEssenceToTheSelectedCard(t *testing.T) {
	gs := testRun()

	var g goods
	g.open(gs, goodHolding(t, session.ContentsEssences))
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

	if g.stage != goodsShowing {
		t.Fatal("the essence was taken and the dialog did not show what it did")
	}
	if now, ok := gs.Run.Card(idx); gs.Run.Size() != size || (ok && now != before) {
		t.Error("the deck was altered while the result was still on screen")
	}

	for i := 0; g.openNow(); i++ {
		if i > 10000 {
			t.Fatal("the dialog never closed")
		}
		g.tickShowing(gs)
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
	g.open(gs, goodHolding(t, session.ContentsStones))
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
	g.open(gs, goodHolding(t, session.ContentsEssences))

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

// **Two sizes of one catalog must not hold nested contents** *(2026-09-15)*.
//
// This is the whole reason `seeds.ForFightSeat` exists. Every deal is `shuffle(catalog)[:size]`, so
// two goods sharing a stream would hand the small bag literally the first three rocks of the large
// one — at a lower price, in the same shop, on the same fight. Nothing in play could show that: the
// contents are sealed until bought, and a player who bought both would see an overlap and read it
// as luck. `loadGoods` used to refuse a second good per catalog outright to prevent it; this is what
// replaced that rule, so this test is what holds the replacement up.
func TestTwoSizesOfOneCatalogDealDifferently(t *testing.T) {
	gs := testRun()

	var bags []session.Good
	for _, key := range session.GoodKeys() {
		if g, ok := session.GoodByKey(key); ok && g.Contains == session.ContentsStones {
			bags = append(bags, g)
		}
	}
	if len(bags) < 2 {
		t.Skip("only one bag of rocks is authored; nothing to compare")
	}

	small, large := bags[0], bags[len(bags)-1]
	a := dealStones(gs, small.Record, small.Size)
	b := dealStones(gs, large.Record, large.Size)

	nested := len(a) > 0 && len(a) <= len(b)
	for i := range a {
		if a[i].Record != b[i].Record {
			nested = false
			break
		}
	}
	if nested {
		t.Errorf("%s is the first %d stones of %s — the two seats share a shuffle",
			small.Record, len(a), large.Record)
	}
}

// **A shelf never spends both seats on one catalog** *(2026-09-15)*. Two seats and three catalogs
// means a shelf showing a small bag beside a large one has drawn one decision twice and pushed the
// essences and the runes off that visit entirely.
func TestAShelfNeverHoldsTwoOfOneCatalog(t *testing.T) {
	gs := testRun()

	// Walk a good many rolls: the collision is a property of the shuffle, so one deal proves little.
	for fight := 0; fight < 50; fight++ {
		rng := rand.New(rand.NewSource(int64(fight) * 7919))
		keys := dealPacks(rng)

		held := map[session.GoodContents]string{}
		for _, key := range keys {
			g, ok := session.GoodByKey(key)
			if !ok {
				t.Fatalf("the shelf holds %q, which is in no catalog", key)
			}
			if first, clash := held[g.Contains]; clash {
				t.Fatalf("roll %d stands %s beside %s — both %s",
					fight, first, g.Record, g.Contains.Noun())
			}
			held[g.Contains] = g.Record
		}
		if len(keys) != packsOffered {
			t.Errorf("roll %d filled %d of %d seats", fight, len(keys), packsOffered)
		}
	}
	_ = gs
}
