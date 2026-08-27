package session

import (
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
	if !run.Hold("leech") || !run.Hold("leech") {
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

	p, ok := ParasiteByKey("leech")
	if !ok {
		t.Fatal("leech is not in the catalogue")
	}
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

	p, _ := ParasiteByKey("gnaw")
	if p.Count != 2 {
		t.Fatalf("gnaw eats %d cards, and this test is about the two-card case", p.Count)
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
	run := runWith(combat.Plain(combat.Jab))
	id := ids(run)[0]

	leech, _ := ParasiteByKey("leech")
	if !run.ApplyParasite(leech, []int{id}) {
		t.Fatal("the rider was refused")
	}

	mimic, _ := ParasiteByKey("mimic")
	if !run.ApplyParasite(mimic, []int{id}) {
		t.Fatal("the swap was refused")
	}

	card, ok := run.CardByID(id)
	if !ok {
		t.Fatal("the swapped card lost its identity")
	}
	if card.Concept != mimic.Concept {
		t.Errorf("the card is %s, wanted %s",
			combat.ConceptOf(card.Concept).Label, combat.ConceptOf(mimic.Concept).Label)
	}
	if card.HealOnPlay() != leech.Number {
		t.Errorf("the swap lost the rider: heals %d", card.HealOnPlay())
	}
}

func TestAVitaeParasiteTouchesNoCard(t *testing.T) {
	run := runWith(combat.Plain(combat.Strike))
	before, size := run.Vitae(), run.Size()

	p, _ := ParasiteByKey("hoard")
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

	gnaw, _ := ParasiteByKey("gnaw")
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

	leech, _ := ParasiteByKey("leech")
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
	mimic, _ := ParasiteByKey("mimic")
	run := runWith(combat.Plain(mimic.Concept))
	id := ids(run)[0]

	if run.CanApplyParasite(mimic, []int{id}) {
		t.Error("a swap onto the card it already is was offered as a legal target")
	}
}

func TestTheBucketAndItsRidersSurviveASnapshot(t *testing.T) {
	// **The one mistake that cannot be repaired afterwards.** A resumed run one consumable lighter,
	// or holding a card whose rider stopped working, is a run the player would have to work out had
	// changed.
	run := runWith(combat.Plain(combat.Strike), combat.Plain(combat.Jab))
	id := ids(run)[0]

	leech, _ := ParasiteByKey("leech")
	if !run.ApplyParasite(leech, []int{id}) {
		t.Fatal("the rider was refused")
	}
	run.Hold("gnaw")
	run.Hold("hoard")

	snap := run.Snapshot(0)
	if len(snap.Held) != 2 || snap.Held[0] != "gnaw" || snap.Held[1] != "hoard" {
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
