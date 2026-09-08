package session

import (
	"math/rand"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// runWith is a run holding exactly these cards, with identities minted.
func runWith(cards ...combat.Card) *Session { return New(cards) }

// ids is every identity the run holds, in deck order.
func ids(s *Session) []int {
	out := make([]int, 0, s.Size())
	for _, c := range s.Deck() {
		out = append(out, c.ID)
	}
	return out
}

// anyWithTarget is a parasite from the shipped catalogue that does this, whichever one it is.
//
// **A test's subject is the grammar, not the record.** Every assertion in this file already reads
// off the resolved parasite — `p.Number`, `p.Count`, `p.Concept` — so the record key was the one
// place a rename could break a test that was not about that record at all, and it broke seven of
// them at once when `rockshower` became `rockbeetle`.
//
// **Deterministic, because `Parasites()` is sorted.** The first match is the same one every run,
// which is what stops this being a test that quietly changes what it exercises.
func anyWithTarget(t *testing.T, target ParasiteTarget) Parasite {
	t.Helper()
	for _, p := range Parasites() {
		if p.Target == target {
			return p
		}
	}
	t.Fatalf("the catalogue holds no %s parasite, so nothing exercises it", target)
	return Parasite{}
}

// otherThan is any concept that is not this one, for a test that needs a card a swap would change.
func otherThan(c combat.ConceptID) combat.ConceptID {
	if c != combat.Strike {
		return combat.Strike
	}
	return combat.Jab
}

// anyWithRider is anyWithTarget for the one target whose behaviour is chosen by a second field.
func anyWithRider(t *testing.T, kind combat.RiderKind) Parasite {
	t.Helper()
	for _, p := range Parasites() {
		if p.Target == ParasiteRider && p.Rider == kind {
			return p
		}
	}
	t.Fatalf("the catalogue attaches no %s rider, so nothing exercises it", kind)
	return Parasite{}
}

// The catalogue's own promises. A bad record panics at init, so by the time a test runs the file
// has already been validated — what is left worth checking is that the shipped file actually
// exercises the grammar rather than four records of one shape.
func TestTheParasiteCatalogueCoversEveryTarget(t *testing.T) {
	seen := map[ParasiteTarget]bool{}
	for _, p := range Parasites() {
		seen[p.Target] = true
	}
	for _, target := range ParasiteTargets() {
		if !seen[target] {
			t.Errorf("no parasite targets %s, so nothing in the shipped file exercises it", target)
		}
	}
}

func TestEveryParasiteTargetHasANameThatParsesBack(t *testing.T) {
	for _, target := range ParasiteTargets() {
		got, ok := ParseParasiteTarget(target.String())
		if !ok || got != target {
			t.Errorf("target %d spells itself %q, which parses back as %d/%v",
				target, target.String(), got, ok)
		}
	}
	if _, ok := ParseParasiteTarget("no-such-target"); ok {
		t.Error("an unknown target name resolved to something")
	}
}

func TestARecordNamingSomethingTheRulesLackIsRefused(t *testing.T) {
	// **The refusal is the safety property.** A parasite that quietly attached nothing, or turned a
	// card into a concept the rules have not registered, is a mechanic nobody designed — and it
	// would be discovered by a player rather than by a build.
	cases := map[string]data.ParasiteData{
		"an unknown target": {ParasiteRecord: "x", Name: "X", Text: "t", Target: "nibble", Count: 1},
		"an unknown rider":  {ParasiteRecord: "x", Name: "X", Text: "t", Target: "rider", Rider: "grow-a-hat", Value: "1", Count: 1},
		"an unknown card":   {ParasiteRecord: "x", Name: "X", Text: "t", Target: "swap", Value: "Kerfuffle", Count: 1},
		"a rider on remove": {ParasiteRecord: "x", Name: "X", Text: "t", Target: "remove", Rider: "heal-on-play", Count: 1},
		"vitae with a card": {ParasiteRecord: "x", Name: "X", Text: "t", Target: "vitae", Value: "5", Count: 1},
		"remove with none":  {ParasiteRecord: "x", Name: "X", Text: "t", Target: "remove", Count: 0},
		"too many targets":  {ParasiteRecord: "x", Name: "X", Text: "t", Target: "remove", Count: MaxParasiteTargets + 1},
		"no text":           {ParasiteRecord: "x", Name: "X", Target: "remove", Count: 1},
	}
	for name, rec := range cases {
		if _, err := resolveParasite(rec); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestTheBucketHoldsWhatIsPutInIt(t *testing.T) {
	run := runWith(combat.Plain(combat.Strike))

	if run.HoldCount() != 0 {
		t.Fatalf("a fresh run started holding %d parasites", run.HoldCount())
	}
	one := anyWithRider(t, combat.RiderHealOnPlay).Record
	if !run.Hold(one) || !run.Hold(one) {
		t.Fatal("the bucket refused a parasite the catalogue has")
	}
	if run.HoldCount() != 2 {
		t.Errorf("two of the same parasite counted as %d", run.HoldCount())
	}
	if run.Hold("no-such-parasite") {
		t.Error("the bucket took a parasite the catalogue has not got")
	}

	if !run.Drop(0) || run.HoldCount() != 1 {
		t.Errorf("dropping one left %d", run.HoldCount())
	}
	if run.Drop(5) {
		t.Error("dropping a position the bucket has not got reported success")
	}
}

func TestARiderParasiteAttachesToTheCardItNames(t *testing.T) {
	run := runWith(combat.Plain(combat.Strike), combat.Plain(combat.Jab))
	held := ids(run)

	p := anyWithRider(t, combat.RiderHealOnPlay)
	if !run.ApplyParasite(p, []int{held[0]}) {
		t.Fatal("a legal rider was refused")
	}

	first, _ := run.CardByID(held[0])
	second, _ := run.CardByID(held[1])
	if first.HealOnPlay() != p.Number {
		t.Errorf("the named card heals %d, wanted %d", first.HealOnPlay(), p.Number)
	}
	if second.RiderCount() != 0 {
		t.Error("a card that was not named picked up a rider")
	}
}

func TestARemoveParasiteEatsBothOfItsTargets(t *testing.T) {
	// **The two-card case is the one worms never had**, and it is where an index-based
	// implementation goes wrong: removing the first shifts the second.
	run := runWith(combat.Plain(combat.Strike), combat.Plain(combat.Jab), combat.Plain(combat.Poke))
	held := ids(run)

	p := anyWithTarget(t, ParasiteRemove)
	if p.Count != 2 {
		t.Fatalf("%s eats %d cards, and this test is about the two-card case", p.Record, p.Count)
	}
	if !run.ApplyParasite(p, []int{held[0], held[2]}) {
		t.Fatal("a legal two-card removal was refused")
	}

	if run.Size() != 1 {
		t.Fatalf("a deck of three lost two cards and holds %d", run.Size())
	}
	if _, ok := run.CardByID(held[1]); !ok {
		t.Error("the card that was not named is the one that went")
	}
}

func TestASwapKeepsTheCardsIdentityAndItsRiders(t *testing.T) {
	// **A card the player has already spent parasites on stays the card they invested in.** If a
	// swap minted a new identity the riders would go with it, and a player would watch an
	// investment vanish because they changed what the card was.
	// **The card starts as something the swap is not**, since a swap onto the card it already is
	// is refused — so the starting concept is derived from the parasite rather than named.
	effigy := anyWithTarget(t, ParasiteSwap)
	run := runWith(combat.Plain(otherThan(effigy.Concept)))
	id := ids(run)[0]

	leech := anyWithRider(t, combat.RiderHealOnPlay)
	if !run.ApplyParasite(leech, []int{id}) {
		t.Fatal("the rider was refused")
	}

	if !run.ApplyParasite(effigy, []int{id}) {
		t.Fatal("the swap was refused")
	}

	card, ok := run.CardByID(id)
	if !ok {
		t.Fatal("the swapped card lost its identity")
	}
	if card.Concept != effigy.Concept {
		t.Errorf("the card is %s, wanted %s",
			combat.ConceptOf(card.Concept).Label, combat.ConceptOf(effigy.Concept).Label)
	}
	if card.HealOnPlay() != leech.Number {
		t.Errorf("the swap lost the rider: heals %d", card.HealOnPlay())
	}
}

func TestAGraftMakesTheLeftCardTheRightCardWhole(t *testing.T) {
	// **"BECOMES" is not a partial verb** *(owner's call, 2026-09-08)*. It copied the concept alone,
	// so grafting a fire Cut onto an ice Jab produced a card whose name said it had become the
	// right-hand card and whose colour said it had not.
	graft := anyWithTarget(t, ParasiteClone)
	run := runWith(
		combat.Card{Concept: combat.Jab, Element: combat.Ice},
		combat.Card{Concept: combat.Strike, Element: combat.Fire},
	)
	left, right := ids(run)[0], ids(run)[1]

	// A rider on the right-hand card, so the test says what "whole" means rather than checking one
	// extra field: everything the right card is travels, not a list somebody has to keep current.
	leech := anyWithRider(t, combat.RiderHealOnPlay)
	if !run.ApplyParasite(leech, []int{right}) {
		t.Fatal("the rider was refused")
	}

	if !run.ApplyParasite(graft, []int{left, right}) {
		t.Fatal("the graft was refused")
	}

	got, ok := run.CardByID(left)
	if !ok {
		t.Fatal("the grafted card lost its identity")
	}
	want, _ := run.CardByID(right)

	// **The identity is the one thing that does not travel**, so the run still holds two cards, each
	// findable by the handle it has always had.
	if got.ID != left {
		t.Errorf("the graft moved the card's identity to %d", got.ID)
	}
	got.ID, want.ID = 0, 0
	if got != want {
		t.Errorf("the left card is %+v, wanted the right card %+v", got, want)
	}
}

func TestAGraftOntoAnIdenticalCardDoesNothing(t *testing.T) {
	// The pair check compares everything the apply copies. Two cards alike in every way but their
	// identity would leave the deck exactly as it was found, so the pick is refused rather than
	// spending the parasite on nothing.
	graft := anyWithTarget(t, ParasiteClone)
	run := runWith(
		combat.Card{Concept: combat.Jab, Element: combat.Ice},
		combat.Card{Concept: combat.Jab, Element: combat.Ice},
	)
	if run.CanApplyParasite(graft, ids(run)) {
		t.Error("a graft between two identical cards was offered")
	}
}

func TestAGraftBetweenTwoColoursOfOneCardIsOffered(t *testing.T) {
	// The pick a player reaching for this most obviously wants, and the one the concept-only check
	// used to refuse: same name, different colour.
	graft := anyWithTarget(t, ParasiteClone)
	run := runWith(
		combat.Card{Concept: combat.Jab, Element: combat.Ice},
		combat.Card{Concept: combat.Jab, Element: combat.Fire},
	)
	if !run.CanApplyParasite(graft, ids(run)) {
		t.Error("a graft between two colours of one card was refused")
	}
}

func TestAVitaeParasiteTouchesNoCard(t *testing.T) {
	run := runWith(combat.Plain(combat.Strike))
	before, size := run.Vitae(), run.Size()

	p := anyWithTarget(t, ParasiteVitae)
	if !run.ApplyParasite(p, nil) {
		t.Fatal("a parasite that needs no target was refused")
	}
	if run.Vitae() != before+p.Number {
		t.Errorf("the purse went %d to %d, wanted %d", before, run.Vitae(), before+p.Number)
	}
	if run.Size() != size {
		t.Errorf("a purse parasite changed the deck size to %d", run.Size())
	}
}

func TestAParasiteRefusesTheWrongNumberOfTargets(t *testing.T) {
	run := runWith(combat.Plain(combat.Strike), combat.Plain(combat.Jab))
	held := ids(run)

	gnaw := anyWithTarget(t, ParasiteRemove)
	if run.ApplyParasite(gnaw, []int{held[0]}) {
		t.Error("a two-card parasite fired on one card")
	}
	if run.ApplyParasite(gnaw, []int{held[0], held[0]}) {
		t.Error("a two-card parasite fired on one card named twice")
	}
	if run.ApplyParasite(gnaw, []int{held[0], 9999}) {
		t.Error("a parasite fired on an identity the run has not got")
	}
	if run.Size() != 2 {
		t.Errorf("a refused parasite still changed the deck: %d cards left", run.Size())
	}
}

func TestARiderIsRefusedOnACardWithNoRoom(t *testing.T) {
	// The board piece asks before it offers, so a card with a full row of badges is dim rather
	// than a click that silently does nothing.
	run := runWith(combat.Plain(combat.Strike))
	id := ids(run)[0]

	leech := anyWithRider(t, combat.RiderHealOnPlay)
	for i := 0; i < combat.MaxCardRiders; i++ {
		if !run.ApplyParasite(leech, []int{id}) {
			t.Fatalf("rider %d was refused", i+1)
		}
	}
	if run.CanApplyParasite(leech, []int{id}) {
		t.Error("a full card was offered as a legal target")
	}
	if run.ApplyParasite(leech, []int{id}) {
		t.Error("a full card took another rider")
	}
}

func TestASwapOntoTheCardItAlreadyIsDoesNothing(t *testing.T) {
	effigy := anyWithTarget(t, ParasiteSwap)
	run := runWith(combat.Plain(effigy.Concept))
	id := ids(run)[0]

	if run.CanApplyParasite(effigy, []int{id}) {
		t.Error("a swap onto the card it already is was offered as a legal target")
	}
}

func TestTheBucketAndItsRidersSurviveASnapshot(t *testing.T) {
	// **The one mistake that cannot be repaired afterwards.** A resumed run one consumable lighter,
	// or holding a card whose rider stopped working, is a run the player would have to work out had
	// changed.
	run := runWith(combat.Plain(combat.Strike), combat.Plain(combat.Jab))
	id := ids(run)[0]

	leech := anyWithRider(t, combat.RiderHealOnPlay)
	if !run.ApplyParasite(leech, []int{id}) {
		t.Fatal("the rider was refused")
	}
	// **The two records are found by target rather than named**, so this goes on testing that the
	// bucket round-trips in acquisition order rather than that two particular parasites exist.
	first := anyWithTarget(t, ParasiteRemove).Record
	second := anyWithTarget(t, ParasiteVitae).Record
	run.Hold(first)
	run.Hold(second)

	snap := run.Snapshot(0)
	if len(snap.Held) != 2 || snap.Held[0] != first || snap.Held[1] != second {
		t.Errorf("the bucket was written as %v, wanted acquisition order", snap.Held)
	}

	back, _, err := Resume(nil, nil, snap)
	if err != nil {
		t.Fatalf("the run would not resume: %v", err)
	}
	if back.HoldCount() != 2 {
		t.Errorf("the resumed run holds %d parasites", back.HoldCount())
	}
	card, ok := back.CardByID(id)
	if !ok {
		t.Fatal("the ridden card did not come back")
	}
	if card.HealOnPlay() != leech.Number {
		t.Errorf("the resumed card heals %d, wanted %d", card.HealOnPlay(), leech.Number)
	}
}

func TestARockShowerCarriesEveryStoneItDrawsRatherThanPlacingThem(t *testing.T) {
	// **They go into the pouch, not onto the ladder** *(owner's call, 2026-09-02)*. A shower hands
	// over consumables to be spent or sold later; the run decides which rungs it raises.
	run := runWith(combat.Plain(combat.Strike))
	p := anyWithTarget(t, ParasiteStones)

	if !run.ApplyParasiteRolling(p, nil, rand.New(rand.NewSource(1))) {
		t.Fatal("a rock shower was refused")
	}

	if got := run.CarryCount(); got != p.Number {
		t.Errorf("a shower of %d put %d stones in the pouch", p.Number, got)
	}
	placed := 0
	for _, n := range run.StoneCounts() {
		placed += n
	}
	if placed != 0 {
		t.Errorf("a shower placed %d stones on the ladder, and should have placed none", placed)
	}
	if got := len(run.Granted()); got != p.Number {
		t.Errorf("the receipt shows %d stones, wanted %d", got, p.Number)
	}
}

func TestACarriedStoneIsSpentOntoItsOwnRung(t *testing.T) {
	run := runWith(combat.Plain(combat.Strike))
	p := anyWithTarget(t, ParasiteStones)
	run.ApplyParasiteRolling(p, nil, rand.New(rand.NewSource(3)))

	first, _ := StoneByKey(run.Carried()[0])
	before := run.StonesOn(first.Hand)

	if !run.SpendCarried(0) {
		t.Fatal("a carried stone would not be spent")
	}
	if got := run.StonesOn(first.Hand); got != before+1 {
		t.Errorf("%s went on rung %s and left it at %d, wanted %d",
			first.Record, first.Hand, got, before+1)
	}
	if run.CarryCount() != p.Number-1 {
		t.Errorf("the pouch holds %d after spending one of %d", run.CarryCount(), p.Number)
	}
}

func TestASoldStonePaysAndNeverReachesTheLadder(t *testing.T) {
	run := runWith(combat.Plain(combat.Strike))
	p := anyWithTarget(t, ParasiteStones)
	run.ApplyParasiteRolling(p, nil, rand.New(rand.NewSource(5)))

	sold, _ := StoneByKey(run.Carried()[0])
	purse := run.Vitae()

	if !run.SellCarried(0) {
		t.Fatal("a carried stone would not be sold")
	}
	if got := run.Vitae(); got != purse+StoneSalePrice {
		t.Errorf("selling a stone paid %d, wanted %d", got-purse, StoneSalePrice)
	}
	if got := run.StonesOn(sold.Hand); got != 0 {
		t.Errorf("a sold stone left %d on rung %s", got, sold.Hand)
	}
}

func TestThePouchSurvivesASnapshot(t *testing.T) {
	// **A run resumed one consumable lighter is a run the player would have to work out had
	// changed** — the rule the bucket and the worn rings are both under.
	run := runWith(combat.Plain(combat.Strike))
	p := anyWithTarget(t, ParasiteStones)
	run.ApplyParasiteRolling(p, nil, rand.New(rand.NewSource(9)))
	want := run.Carried()

	back, _, err := Resume(nil, nil, run.Snapshot(0))
	if err != nil {
		t.Fatalf("a run carrying stones would not resume: %v", err)
	}
	got := back.Carried()
	if len(got) != len(want) {
		t.Fatalf("the pouch came back holding %d of %d stones", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("pouch seat %d came back as %s, wanted %s", i, got[i], want[i])
		}
	}
}

func TestARockShowerDrawsWithoutRepeats(t *testing.T) {
	// A seat spent showing the same rock twice says nothing, which is the bag's own argument.
	run := runWith(combat.Plain(combat.Strike))
	p := anyWithTarget(t, ParasiteStones)

	if !run.ApplyParasiteRolling(p, nil, rand.New(rand.NewSource(7))) {
		t.Fatal("a rock shower was refused")
	}

	seen := map[string]bool{}
	for _, st := range run.Granted() {
		if seen[st.Record] {
			t.Errorf("%s was handed over twice in one shower", st.Record)
		}
		seen[st.Record] = true
	}
}

func TestARockShowerWithNoSourceIsRefusedRatherThanRolledTheSameWayTwice(t *testing.T) {
	// **Refused outright rather than falling back to a default draw.** A consumable quietly
	// handing out the same three rocks every time is a mechanic nobody designed, and it would be
	// invisible — see ApplyParasiteRolling.
	run := runWith(combat.Plain(combat.Strike))
	p := anyWithTarget(t, ParasiteStones)

	if run.ApplyParasite(p, nil) {
		t.Error("a rock shower rolled with no source")
	}
}

func TestTwoShowersFromDifferentSourcesCanDifferAndOneSourceIsRepeatable(t *testing.T) {
	// The stream is the caller's, so what this pins is that the run does not smuggle in a source
	// of its own: the same seed twice is the same three stones.
	first := runWith(combat.Plain(combat.Strike))
	second := runWith(combat.Plain(combat.Strike))
	p := anyWithTarget(t, ParasiteStones)

	first.ApplyParasiteRolling(p, nil, rand.New(rand.NewSource(42)))
	second.ApplyParasiteRolling(p, nil, rand.New(rand.NewSource(42)))

	a, b := first.Granted(), second.Granted()
	if len(a) != len(b) {
		t.Fatalf("one seed drew %d stones and the other %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Record != b[i].Record {
			t.Errorf("seat %d drew %s once and %s the other time", i, a[i].Record, b[i].Record)
		}
	}
}

// The cap is a rule, not a label on a pane. **`MaxHeld` is what the top row draws as `held/2`** —
// see internal/screens/consumables.go — and a bucket that took a third would make that fraction a
// lie on the one screen the player reads their build off.
func TestTheBucketRefusesMoreThanItHolds(t *testing.T) {
	run := runWith(combat.Plain(combat.Strike))

	filler := anyWithRider(t, combat.RiderHealOnPlay).Record
	spare := anyWithTarget(t, ParasiteRemove).Record

	for i := 0; i < MaxHeld; i++ {
		if !run.Hold(filler) {
			t.Fatalf("the bucket refused parasite %d of %d", i+1, MaxHeld)
		}
	}
	if !run.HoldFull() {
		t.Errorf("a bucket holding %d of %d does not report itself full", run.HoldCount(), MaxHeld)
	}
	if run.Hold(spare) {
		t.Errorf("a full bucket took a %dth parasite", MaxHeld+1)
	}
	if run.HoldCount() != MaxHeld {
		t.Errorf("the bucket holds %d, past the cap of %d", run.HoldCount(), MaxHeld)
	}

	// **Dropping one makes room again**, which is what makes the cap a bound on carrying rather
	// than on ever acquiring.
	if !run.Drop(0) || run.HoldFull() {
		t.Errorf("a bucket with one spent still reports itself full at %d", run.HoldCount())
	}
	if !run.Hold(spare) || run.HoldCount() != MaxHeld {
		t.Errorf("the freed seat did not take a parasite: %d held", run.HoldCount())
	}
}
