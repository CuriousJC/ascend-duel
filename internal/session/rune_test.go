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

// anyWithTarget is a rune from the shipped catalog that does this, whichever one it is.
//
// **A test's subject is the grammar, not the record.** Every assertion in this file already reads
// off the resolved rune — `p.Number`, `p.Count` — so the record key was the one place a rename
// could break a test that was not about that record at all, and it broke seven of them at once
// when `rockshower` was renamed.
//
// **Deterministic, because `Runes()` is sorted.** The first match is the same one every run,
// which is what stops this being a test that quietly changes what it exercises.
func anyWithTarget(t *testing.T, target RuneTarget) Rune {
	t.Helper()
	for _, p := range Runes() {
		if p.Target == target {
			return p
		}
	}
	t.Fatalf("the catalog holds no %s rune, so nothing exercises it", target)
	return Rune{}
}

// notOfForm is any concept the rules have that is not this form, for a test whose cards have to
// be something a form rune would actually change.
func notOfForm(t *testing.T, f combat.Form) combat.ConceptID {
	t.Helper()
	for i := 0; i < combat.ConceptCount(); i++ {
		id := combat.ConceptID(i)
		if combat.ConceptOf(id).Form != f {
			return id
		}
	}
	t.Fatalf("every concept this build has is %s", f)
	return combat.NoConcept
}

// anyWithRider is anyWithTarget for the one target whose behavior is chosen by a second field.
func anyWithRider(t *testing.T, kind combat.RiderKind) Rune {
	t.Helper()
	for _, p := range Runes() {
		if p.Target == RuneRider && p.Rider == kind {
			return p
		}
	}
	t.Fatalf("the catalog attaches no %s rider, so nothing exercises it", kind)
	return Rune{}
}

// The catalog's own promises. A bad record panics at init, so by the time a test runs the file
// has already been validated — what is left worth checking is that the shipped file actually
// exercises the grammar rather than four records of one shape.
func TestTheRuneCatalogCoversEveryTarget(t *testing.T) {
	seen := map[RuneTarget]bool{}
	for _, p := range Runes() {
		seen[p.Target] = true
	}
	for _, target := range RuneTargets() {
		if !seen[target] {
			t.Errorf("no rune targets %s, so nothing in the shipped file exercises it", target)
		}
	}
}

func TestEveryRuneTargetHasANameThatParsesBack(t *testing.T) {
	for _, target := range RuneTargets() {
		got, ok := ParseRuneTarget(target.String())
		if !ok || got != target {
			t.Errorf("target %d spells itself %q, which parses back as %d/%v",
				target, target.String(), got, ok)
		}
	}
	if _, ok := ParseRuneTarget("no-such-target"); ok {
		t.Error("an unknown target name resolved to something")
	}
}

func TestARecordNamingSomethingTheRulesLackIsRefused(t *testing.T) {
	// **The refusal is the safety property.** A rune that quietly attached nothing, or turned a
	// card into a concept the rules have not registered, is a mechanic nobody designed — and it
	// would be discovered by a player rather than by a build.
	cases := map[string]data.RuneData{
		"an unknown target": {RuneRecord: "x", Name: "X", Text: "t", Target: "nibble", Count: 1},
		"an unknown rider":  {RuneRecord: "x", Name: "X", Text: "t", Target: "rider", Rider: "grow-a-hat", Value: "1", Count: 1},
		"an unknown card":   {RuneRecord: "x", Name: "X", Text: "t", Target: "swap", Value: "Kerfuffle", Count: 1},
		"a rider on remove": {RuneRecord: "x", Name: "X", Text: "t", Target: "remove", Rider: "heal-on-play", Count: 1},
		"vitae with a card": {RuneRecord: "x", Name: "X", Text: "t", Target: "vitae", Value: "5", Count: 1},
		"remove with none":  {RuneRecord: "x", Name: "X", Text: "t", Target: "remove", Count: 0},
		"too many targets":  {RuneRecord: "x", Name: "X", Text: "t", Target: "remove", Count: MaxRuneTargets + 1},
		"no text":           {RuneRecord: "x", Name: "X", Target: "remove", Count: 1},
	}
	for name, rec := range cases {
		if _, err := resolveRune(rec); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestTheSackHoldsWhatIsPutInIt(t *testing.T) {
	run := runWith(combat.Plain(combat.Bash))

	if run.HoldCount() != 0 {
		t.Fatalf("a fresh run started holding %d runes", run.HoldCount())
	}
	one := anyWithRider(t, combat.RiderHealOnPlay).Record
	if !run.Hold(one) || !run.Hold(one) {
		t.Fatal("the sack refused a rune the catalog has")
	}
	if run.HoldCount() != 2 {
		t.Errorf("two of the same rune counted as %d", run.HoldCount())
	}
	if run.Hold("no-such-rune") {
		t.Error("the sack took a rune the catalog has not got")
	}

	if !run.Drop(0) || run.HoldCount() != 1 {
		t.Errorf("dropping one left %d", run.HoldCount())
	}
	if run.Drop(5) {
		t.Error("dropping a position the sack has not got reported success")
	}
}

func TestARiderRuneAttachesToTheCardItNames(t *testing.T) {
	run := runWith(combat.Plain(combat.Bash), combat.Plain(combat.Jab))
	held := ids(run)

	p := anyWithRider(t, combat.RiderHealOnPlay)
	if !run.ApplyRune(p, []int{held[0]}) {
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

func TestARemoveRuneEatsBothOfItsTargets(t *testing.T) {
	// **The two-card case is the one essences never had**, and it is where an index-based
	// implementation goes wrong: removing the first shifts the second.
	run := runWith(combat.Plain(combat.Bash), combat.Plain(combat.Jab), combat.Plain(combat.Poke))
	held := ids(run)

	p := anyWithTarget(t, RuneRemove)
	if p.Count != 2 {
		t.Fatalf("%s eats %d cards, and this test is about the two-card case", p.Record, p.Count)
	}
	if !run.ApplyRune(p, []int{held[0], held[2]}) {
		t.Fatal("a legal two-card removal was refused")
	}

	if run.Size() != 1 {
		t.Fatalf("a deck of three lost two cards and holds %d", run.Size())
	}
	if _, ok := run.CardByID(held[1]); !ok {
		t.Error("the card that was not named is the one that went")
	}
}

func TestAGraftMakesTheLeftCardTheRightCardWhole(t *testing.T) {
	// **"BECOMES" is not a partial verb** *(owner's call, 2026-09-08)*. It copied the concept alone,
	// so grafting a fire Cut onto an ice Jab produced a card whose name said it had become the
	// right-hand card and whose color said it had not.
	graft := anyWithTarget(t, RuneClone)
	run := runWith(
		combat.Card{Concept: combat.Jab, Element: combat.Ice},
		combat.Card{Concept: combat.Bash, Element: combat.Fire},
	)
	left, right := ids(run)[0], ids(run)[1]

	// A rider on the right-hand card, so the test says what "whole" means rather than checking one
	// extra field: everything the right card is travels, not a list somebody has to keep current.
	siphon := anyWithRider(t, combat.RiderHealOnPlay)
	if !run.ApplyRune(siphon, []int{right}) {
		t.Fatal("the rider was refused")
	}

	if !run.ApplyRune(graft, []int{left, right}) {
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
	// spending the rune on nothing.
	graft := anyWithTarget(t, RuneClone)
	run := runWith(
		combat.Card{Concept: combat.Jab, Element: combat.Ice},
		combat.Card{Concept: combat.Jab, Element: combat.Ice},
	)
	if run.CanApplyRune(graft, ids(run)) {
		t.Error("a graft between two identical cards was offered")
	}
}

func TestAGraftBetweenTwoColorsOfOneCardIsOffered(t *testing.T) {
	// The pick a player reaching for this most obviously wants, and the one the concept-only check
	// used to refuse: same name, different color.
	graft := anyWithTarget(t, RuneClone)
	run := runWith(
		combat.Card{Concept: combat.Jab, Element: combat.Ice},
		combat.Card{Concept: combat.Jab, Element: combat.Fire},
	)
	if !run.CanApplyRune(graft, ids(run)) {
		t.Error("a graft between two colors of one card was refused")
	}
}

func TestAVitaeRuneTouchesNoCard(t *testing.T) {
	run := runWith(combat.Plain(combat.Bash))
	before, size := run.Vitae(), run.Size()

	p := anyWithTarget(t, RuneVitae)
	if !run.ApplyRune(p, nil) {
		t.Fatal("a rune that needs no target was refused")
	}
	if run.Vitae() != before+p.Number {
		t.Errorf("the purse went %d to %d, wanted %d", before, run.Vitae(), before+p.Number)
	}
	if run.Size() != size {
		t.Errorf("a purse rune changed the deck size to %d", run.Size())
	}
}

func TestARuneRefusesTheWrongNumberOfTargets(t *testing.T) {
	run := runWith(combat.Plain(combat.Bash), combat.Plain(combat.Jab))
	held := ids(run)

	unmake := anyWithTarget(t, RuneRemove)
	if run.ApplyRune(unmake, []int{held[0]}) {
		t.Error("a two-card rune fired on one card")
	}
	if run.ApplyRune(unmake, []int{held[0], held[0]}) {
		t.Error("a two-card rune fired on one card named twice")
	}
	if run.ApplyRune(unmake, []int{held[0], 9999}) {
		t.Error("a rune fired on an identity the run has not got")
	}
	if run.Size() != 2 {
		t.Errorf("a refused rune still changed the deck: %d cards left", run.Size())
	}
}

// **The only illegal rider pick is the one that would change nothing** *(owner's call, 2026-09-09)*.
// It used to be a card carrying its maximum, which stopped making sense the day the maximum became
// one: a card already upgraded is the pick a player reaching for a second rune most obviously
// wants. What is refused is the same upgrade twice, which is the rule every other target is under.
func TestARiderIsRefusedOnlyWhenItWouldChangeNothing(t *testing.T) {
	run := runWith(combat.Plain(combat.Bash))
	id := ids(run)[0]

	siphon := anyWithRider(t, combat.RiderHealOnPlay)
	if !run.ApplyRune(siphon, []int{id}) {
		t.Fatal("a plain card refused its first upgrade")
	}
	if run.CanApplyRune(siphon, []int{id}) {
		t.Error("the same upgrade twice was offered as a legal target")
	}
	if run.ApplyRune(siphon, []int{id}) {
		t.Error("a card took the same upgrade twice")
	}

	// A *different* upgrade is legal and replaces the first outright.
	motley := anyWithRider(t, combat.RiderWildElement)
	if !run.CanApplyRune(motley, []int{id}) {
		t.Fatal("an upgraded card refused a different upgrade")
	}
	if !run.ApplyRune(motley, []int{id}) {
		t.Fatal("a different upgrade did not take")
	}

	card, _ := run.CardByID(id)
	if card.RiderCount() != 1 {
		t.Errorf("a card upgraded twice carries %d upgrades", card.RiderCount())
	}
	if card.HealOnPlay() != 0 {
		t.Errorf("the replaced heal still pays %d", card.HealOnPlay())
	}
	if !card.Wild(combat.AxisElement) {
		t.Error("the replacing upgrade did not take")
	}
}

// **A normal change leaves the card's upgrade exactly where it was.** That is the whole of what the
// two classes are for — see RuneChange — and it is the half a test can catch, since the
// difference between Bulwark and Golden is invisible until a *second* rune is spent.
func TestANormalChangeLeavesTheUpgradeAlone(t *testing.T) {
	// **Two cards, because the element and form runes take two.** The gold goes on the first
	// and every assertion below is about that one; the second is only somebody for the pair
	// runes to name.
	//
	// **The starting cards are derived from the form rune rather than named**, because a rune
	// is refused on a card it would not change — so a hand that happened to already be the
	// form the rune makes would fail this test for a reason that is not what it tests.
	lead := notOfForm(t, anyWithTarget(t, RuneForm).Form)
	run := runWith(combat.Plain(lead), combat.Plain(lead))
	held := ids(run)
	id := held[0]

	golden := anyWithRider(t, combat.RiderGolden)
	if !run.ApplyRune(golden, []int{id}) {
		t.Fatal("a plain card refused gold")
	}

	for _, target := range []RuneTarget{RuneElement, RuneForm} {
		p := anyWithTarget(t, target)
		if p.Change != RuneNormal {
			t.Fatalf("%s calls itself a %s change", p.Record, p.Change)
		}
		if !run.ApplyRune(p, held[:p.Count]) {
			t.Fatalf("%s was refused on a gold card", p.Record)
		}
		card, _ := run.CardByID(id)
		if card.GoldenOdds() == 0 {
			t.Errorf("%s, a normal change, took the gold off the card", p.Record)
		}
	}
}

// **Every record declares its class and the loader agrees with it.** The field is authored rather
// than derived so it is a claim the record makes; this is the check that makes the claim worth
// something — see data.RuneData.Change.
func TestEveryRuneDeclaresTheChangeItActuallyMakes(t *testing.T) {
	for _, p := range Runes() {
		want := RuneNormal
		if p.Target == RuneRider {
			want = RuneUpgrade
		}
		if p.Change != want {
			t.Errorf("%s targets %s and resolved as a %s change, want %s",
				p.Record, p.Target, p.Change, want)
		}
	}
}

// **A record whose declaration disagrees with its target is refused at load**, which is what stops
// the authored field becoming a second source of truth free to drift from the first.
func TestAMisdeclaredChangeIsRefused(t *testing.T) {
	for _, tc := range []struct {
		what   string
		record data.RuneData
	}{
		{"a swap calling itself an upgrade", data.RuneData{
			RuneRecord: "liar", Name: "Liar", Text: "X", Change: "upgrade",
			Target: "swap", Value: "Bash", Count: 1}},
		{"a rider calling itself normal", data.RuneData{
			RuneRecord: "liar", Name: "Liar", Text: "X", Change: "normal",
			Target: "rider", Rider: "heal-on-play", Value: "10", Count: 1}},
		{"a record declaring nothing", data.RuneData{
			RuneRecord: "mute", Name: "Mute", Text: "X",
			Target: "swap", Value: "Bash", Count: 1}},
		{"a record declaring a word that is not one", data.RuneData{
			RuneRecord: "odd", Name: "Odd", Text: "X", Change: "sideways",
			Target: "swap", Value: "Bash", Count: 1}},
	} {
		if _, err := resolveRune(tc.record); err == nil {
			t.Errorf("%s was accepted", tc.what)
		}
	}
}

// **A gambling rider is refused below the number of outcomes.** Under that there is no losing face
// left, and a card that pays every time it is played is a different card entirely — the mistake a
// number in a JSON file could make silently. See combat.LuckOutcomes.
func TestAGambleThatAlwaysPaysIsRefused(t *testing.T) {
	for _, rider := range []string{"golden", "silver"} {
		bad := data.RuneData{
			RuneRecord: "sure-thing", Name: "Sure Thing", Text: "X", Change: "upgrade",
			Target: "rider", Rider: rider, Value: "2", Count: 1,
		}
		if _, err := resolveRune(bad); err == nil {
			t.Errorf("a %s card on a d2 was accepted", rider)
		}
	}
}

func TestTheSackAndItsRidersSurviveASnapshot(t *testing.T) {
	// **The one mistake that cannot be repaired afterwards.** A resumed run one consumable lighter,
	// or holding a card whose rider stopped working, is a run the player would have to work out had
	// changed.
	run := runWith(combat.Plain(combat.Bash), combat.Plain(combat.Jab))
	id := ids(run)[0]

	siphon := anyWithRider(t, combat.RiderHealOnPlay)
	if !run.ApplyRune(siphon, []int{id}) {
		t.Fatal("the rider was refused")
	}
	// **The two records are found by target rather than named**, so this goes on testing that the
	// sack round-trips in acquisition order rather than that two particular runes exist.
	first := anyWithTarget(t, RuneRemove).Record
	second := anyWithTarget(t, RuneVitae).Record
	run.Hold(first)
	run.Hold(second)

	snap := run.Snapshot(0)
	if len(snap.Held) != 2 || snap.Held[0] != first || snap.Held[1] != second {
		t.Errorf("the sack was written as %v, wanted acquisition order", snap.Held)
	}

	back, _, err := Resume(nil, nil, snap)
	if err != nil {
		t.Fatalf("the run would not resume: %v", err)
	}
	if back.HoldCount() != 2 {
		t.Errorf("the resumed run holds %d runes", back.HoldCount())
	}
	card, ok := back.CardByID(id)
	if !ok {
		t.Fatal("the ridden card did not come back")
	}
	if card.HealOnPlay() != siphon.Number {
		t.Errorf("the resumed card heals %d, wanted %d", card.HealOnPlay(), siphon.Number)
	}
}

func TestARockShowerCarriesEveryStoneItDrawsRatherThanPlacingThem(t *testing.T) {
	// **They go into the pouch, not onto the ladder** *(owner's call, 2026-09-02)*. A shower hands
	// over consumables to be spent or sold later; the run decides which rungs it raises.
	run := runWith(combat.Plain(combat.Bash))
	p := anyWithTarget(t, RuneStones)

	if !run.ApplyRuneRolling(p, nil, rand.New(rand.NewSource(1))) {
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
	run := runWith(combat.Plain(combat.Bash))
	p := anyWithTarget(t, RuneStones)
	run.ApplyRuneRolling(p, nil, rand.New(rand.NewSource(3)))

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
	run := runWith(combat.Plain(combat.Bash))
	p := anyWithTarget(t, RuneStones)
	run.ApplyRuneRolling(p, nil, rand.New(rand.NewSource(5)))

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
	// changed** — the rule the sack and the worn relics are both under.
	run := runWith(combat.Plain(combat.Bash))
	p := anyWithTarget(t, RuneStones)
	run.ApplyRuneRolling(p, nil, rand.New(rand.NewSource(9)))
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
	run := runWith(combat.Plain(combat.Bash))
	p := anyWithTarget(t, RuneStones)

	if !run.ApplyRuneRolling(p, nil, rand.New(rand.NewSource(7))) {
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
	// invisible — see ApplyRuneRolling.
	run := runWith(combat.Plain(combat.Bash))
	p := anyWithTarget(t, RuneStones)

	if run.ApplyRune(p, nil) {
		t.Error("a rock shower rolled with no source")
	}
}

func TestTwoShowersFromDifferentSourcesCanDifferAndOneSourceIsRepeatable(t *testing.T) {
	// The stream is the caller's, so what this pins is that the run does not smuggle in a source
	// of its own: the same seed twice is the same three stones.
	first := runWith(combat.Plain(combat.Bash))
	second := runWith(combat.Plain(combat.Bash))
	p := anyWithTarget(t, RuneStones)

	first.ApplyRuneRolling(p, nil, rand.New(rand.NewSource(42)))
	second.ApplyRuneRolling(p, nil, rand.New(rand.NewSource(42)))

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
// see internal/screens/consumables.go — and a sack that took a third would make that fraction a
// lie on the one screen the player reads their build off.
func TestTheSackRefusesMoreThanItHolds(t *testing.T) {
	run := runWith(combat.Plain(combat.Bash))

	filler := anyWithRider(t, combat.RiderHealOnPlay).Record
	spare := anyWithTarget(t, RuneRemove).Record

	for i := 0; i < MaxHeld; i++ {
		if !run.Hold(filler) {
			t.Fatalf("the sack refused rune %d of %d", i+1, MaxHeld)
		}
	}
	if !run.HoldFull() {
		t.Errorf("a sack holding %d of %d does not report itself full", run.HoldCount(), MaxHeld)
	}
	if run.Hold(spare) {
		t.Errorf("a full sack took a %dth rune", MaxHeld+1)
	}
	if run.HoldCount() != MaxHeld {
		t.Errorf("the sack holds %d, past the cap of %d", run.HoldCount(), MaxHeld)
	}

	// **Dropping one makes room again**, which is what makes the cap a bound on carrying rather
	// than on ever acquiring.
	if !run.Drop(0) || run.HoldFull() {
		t.Errorf("a sack with one spent still reports itself full at %d", run.HoldCount())
	}
	if !run.Hold(spare) || run.HoldCount() != MaxHeld {
		t.Errorf("the freed seat did not take a rune: %d held", run.HoldCount())
	}
}

func TestTheMintedCopyIsHandedOverOnceAndOnlyOnce(t *testing.T) {
	// **The handover is about the rune that just fired, not about the last one of its kind.**
	// `Duplicated` is read by the combat screen after *every* rune it spends — see
	// CombatScene.takeRune, which seats what it finds in the dealt hand — so a copy still sitting
	// in the list when the next rune goes off is seated a second time, and a third, for every
	// rune spent afterwards. The copy is minted once and the deck is the right size throughout;
	// what duplicates is the card in the player's hand.
	run := runWith(combat.Plain(combat.Bash), combat.Plain(combat.Jab))
	held := ids(run)

	mimic := anyWithTarget(t, RuneDuplicate)
	if !run.ApplyRune(mimic, []int{held[0]}) {
		t.Fatal("a legal copy was refused")
	}
	if n := len(run.Duplicated()); n != 1 {
		t.Fatalf("one card was copied and %d were handed over", n)
	}

	graft := anyWithTarget(t, RuneClone)
	if !run.ApplyRune(graft, []int{held[0], held[1]}) {
		t.Fatal("a legal graft was refused")
	}
	if n := len(run.Duplicated()); n != 0 {
		t.Errorf("a graft minted nothing and handed over %d cards", n)
	}
}

func TestARockShowersStonesAreHandedOverOnceAndOnlyOnce(t *testing.T) {
	// The same handover, one field over: `Granted` is read after every rune the combat screen
	// spends, so stones left standing fly to the pouch again on whatever is spent next.
	run := runWith(combat.Plain(combat.Bash), combat.Plain(combat.Jab))
	held := ids(run)

	shower := anyWithTarget(t, RuneStones)
	if !run.ApplyRuneRolling(shower, nil, rand.New(rand.NewSource(1))) {
		t.Fatal("a legal rock shower was refused")
	}
	if len(run.Granted()) == 0 {
		t.Fatal("a rock shower handed over no stones")
	}

	graft := anyWithTarget(t, RuneClone)
	if !run.ApplyRune(graft, []int{held[0], held[1]}) {
		t.Fatal("a legal graft was refused")
	}
	if n := len(run.Granted()); n != 0 {
		t.Errorf("a graft drew no stones and handed over %d", n)
	}
}
