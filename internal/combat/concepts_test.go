package combat

import "testing"

// The 2026-08-08 concepts: Ritual, Brace, Feint and Mirror. Each gets the case that says what
// makes it differ in *kind* from the tier below it rather than merely in size — that being the
// rule the concept grid in MECHANICS.md is held to, and the rule a cost ladder quietly breaks.
//
// **Sift is deliberately untested here.** Its effect is on the hand, the hand lives on the
// scene, and this package cannot see a deck by design. It is the one concept `tools/balance`
// cannot exercise either.

// hasKind reports whether the log holds an event of the given kind.
func hasKind(events []Event, kind EventKind) bool {
	for _, e := range events {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

// kindCount is how many events of a kind the log holds.
func kindCount(events []Event, kind EventKind) int {
	n := 0
	for _, e := range events {
		if e.Kind == kind {
			n++
		}
	}
	return n
}

func TestRitualBanksMoreThanGatherAtTheSameNetRate(t *testing.T) {
	// The design claim is that Ritual is not a better Gather, it is a Gather that does not eat
	// the round: both net +1 AP per point spent, and what Ritual sells is action slots. If that
	// stops being true the card has become a strict upgrade and the 4-AP prepare tier has no
	// reason to exist.
	if gatherNet, ritualNet := gatherBonusAP-costGather, ritualBonusAP-costRitual; gatherNet != ritualNet {
		t.Errorf("Gather nets %+d AP and Ritual nets %+d — they are meant to match, so Ritual sells slots not rate",
			gatherNet, ritualNet)
	}

	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	_, aAfter, _ := ResolveRound(a, b, []ActionKind{Ritual}, nil, 1)

	if aAfter.BonusAP != ritualBonusAP {
		t.Errorf("BonusAP after a Ritual = %d, want %d", aAfter.BonusAP, ritualBonusAP)
	}
	if aAfter.ActionPoints() != baseActionPoints+ritualBonusAP {
		t.Errorf("next round's budget = %d, want %d", aAfter.ActionPoints(), baseActionPoints+ritualBonusAP)
	}
}

func TestBraceHalvesOneAttackAndIsSpent(t *testing.T) {
	// Brace is partial where Dodge is binary, and single where Guard is turn-wide. Two attacks
	// into one brace land as half, then full.
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	// B braces first. B acts second, so its brace is standing when A's next turn arrives.
	_, a, b = ResolveRound(a, b, nil, []ActionKind{Brace}, 1)
	events, _, bAfter := ResolveRound(a, b, []ActionKind{Strike, Strike}, nil, 2)

	if !hasKind(events, KindBraced) {
		t.Fatal("no KindBraced event — the brace did not apply")
	}
	if n := kindCount(events, KindBraced); n != 1 {
		t.Errorf("KindBraced fired %d times, want 1 — a brace is spent on one blow", n)
	}

	// Str 10: half a Strike is 5, then a full one is 10.
	if want := 100 - 5 - 10; bAfter.CurrentLife != want {
		t.Errorf("life after a braced Strike then a clean one = %d, want %d", bAfter.CurrentLife, want)
	}
	if bAfter.Braces != 0 {
		t.Errorf("Braces left = %d, want 0", bAfter.Braces)
	}
}

func TestBraceAndGuardBothApply(t *testing.T) {
	// Two cards bought separately, so both bite: a quarter, not a half. A rule that ignored one
	// would make the cheaper card worthless exactly when the player had committed to both.
	a := duelist(20, 0, 100)
	b := duelist(10, 0, 100)

	_, a, b = ResolveRound(a, b, nil, []ActionKind{Brace, Guard}, 1)
	_, _, bAfter := ResolveRound(a, b, []ActionKind{Strike}, nil, 2)

	if want := 100 - 20/braceDivisor/guardDivisor; bAfter.CurrentLife != want {
		t.Errorf("life after a Strike into brace+guard = %d, want %d (quartered)", bAfter.CurrentLife, want)
	}
}

func TestFeintStripsARiposteWithoutTakingTheCounter(t *testing.T) {
	// The whole point of the card. Attacking into a Riposte normally costs the attacker str/2;
	// a Feint clears it and pays nothing for the privilege.
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	_, a, b = ResolveRound(a, b, nil, []ActionKind{Riposte}, 1)
	events, aAfter, bAfter := ResolveRound(a, b, []ActionKind{Feint}, nil, 2)

	if !hasKind(events, KindStripped) {
		t.Fatal("no KindStripped event — the feint did not remove the riposte")
	}
	if hasKind(events, KindNegated) {
		t.Error("a stripped riposte must not also negate: the feint should land")
	}
	if aAfter.CurrentLife != 100 {
		t.Errorf("attacker took %d counter-damage, want 0 — a feint takes no riposte", 100-aAfter.CurrentLife)
	}
	if want := 100 - Feint.Damage(10); bAfter.CurrentLife != want {
		t.Errorf("defender life = %d, want %d — the feint should land after stripping", bAfter.CurrentLife, want)
	}
}

func TestFeintStripsRipostesBeforeDodges(t *testing.T) {
	// Matching the order they are spent in. The riposte is the one worth removing, because it is
	// the one that would otherwise have cost the attacker something.
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	_, a, b = ResolveRound(a, b, nil, []ActionKind{Dodge, Riposte}, 1)
	events, _, _ := ResolveRound(a, b, []ActionKind{Feint}, nil, 2)

	for _, e := range events {
		if e.Kind == KindStripped {
			if e.Action != Riposte {
				t.Errorf("feint stripped a %v, want a Riposte first", e.Action)
			}
			return
		}
	}
	t.Fatal("no KindStripped event")
}

func TestFeintWithNothingToStripIsStillAnAttack(t *testing.T) {
	// It must never be dead. With no negation up it is an overpriced Strike, not a no-op.
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	events, _, bAfter := ResolveRound(a, b, []ActionKind{Feint}, nil, 1)

	if hasKind(events, KindStripped) {
		t.Error("stripped something that was not there")
	}
	if want := 100 - Feint.Damage(10); bAfter.CurrentLife != want {
		t.Errorf("defender life = %d, want %d", bAfter.CurrentLife, want)
	}
}

func TestFeintStripsEvenWhenTheBlowIsMirrored(t *testing.T) {
	// The strip is unconditional on purpose. Making it depend on a card the player cannot see
	// would put a hidden interaction into a game whose whole point is reading the opponent.
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	_, a, b = ResolveRound(a, b, nil, []ActionKind{Mirror, Dodge}, 1)
	events, _, _ := ResolveRound(a, b, []ActionKind{Feint}, nil, 2)

	if !hasKind(events, KindStripped) {
		t.Error("the feint's strip did not fire behind a mirror — it is meant to be unconditional")
	}
}

func TestMirrorNegatesEveryAttackAndReflectsEachOne(t *testing.T) {
	// Guard halves a turn and Dodge stops one blow; Mirror stops all of them and sends each
	// back. It is the card that scales with what the opponent committed rather than with the
	// holder's own strength, which is what makes it a read and not a damage reduction.
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	_, a, b = ResolveRound(a, b, nil, []ActionKind{Mirror}, 1)
	events, aAfter, bAfter := ResolveRound(a, b, []ActionKind{Jab, Jab, Strike}, nil, 2)

	if n := kindCount(events, KindNegated); n != 3 {
		t.Errorf("negated %d attacks, want 3 — a mirror stops every one", n)
	}
	if bAfter.CurrentLife != 100 {
		t.Errorf("mirror holder took %d damage, want 0", 100-bAfter.CurrentLife)
	}

	want := 100 - (Jab.Damage(10) + Jab.Damage(10) + Strike.Damage(10))
	if aAfter.CurrentLife != want {
		t.Errorf("attacker life after reflection = %d, want %d", aAfter.CurrentLife, want)
	}
}

func TestMirrorIsCheckedBeforeCountedNegations(t *testing.T) {
	// A mirror must not let a Dodge be spent on a blow it was going to stop for free, or the
	// expensive card wastes the cheap one that is standing beside it.
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	_, a, b = ResolveRound(a, b, nil, []ActionKind{Mirror, Dodge}, 1)
	events, _, _ := ResolveRound(a, b, []ActionKind{Strike}, nil, 2)

	for _, e := range events {
		if e.Kind == KindNegated && e.Action != Mirror {
			t.Errorf("a %v was spent while a mirror was up", e.Action)
		}
	}
}

func TestMirrorReflectionCanKillMidTurn(t *testing.T) {
	// The reflection follows a Riposte's counter, so it has to be able to end the attacker's
	// turn the same way — otherwise the rest of a lethal turn still resolves.
	a := duelist(10, 0, 8)
	b := duelist(10, 0, 100)

	_, a, b = ResolveRound(a, b, nil, []ActionKind{Mirror}, 1)
	events, aAfter, _ := ResolveRound(a, b, []ActionKind{Strike, Strike, Strike}, nil, 2)

	if aAfter.Alive() {
		t.Fatalf("attacker survived on %d life, want killed by its own reflected Strike", aAfter.CurrentLife)
	}
	if n := damageCount(events); n != 1 {
		t.Errorf("damage events = %d, want 1 — the turn should stop at the kill", n)
	}
	if !hasKind(events, KindDefeated) {
		t.Error("no KindDefeated event for the reflected kill")
	}
}

func TestExpireDefensesClearsEveryDefensiveField(t *testing.T) {
	// expireDefenses is the one place that has to know about every defensive field, and a new
	// one left out of it would stand forever — the worst failure mode available here, and a
	// silent one. Pin the whole set rather than only the fields this session added.
	d := Duelist{Guarded: true, Ripostes: 2, Dodges: 2, Braces: 2, Mirrored: true}

	got := expireDefenses(d)

	if got.Guarded || got.Ripostes != 0 || got.Dodges != 0 || got.Braces != 0 || got.Mirrored {
		t.Errorf("expireDefenses left something standing: %+v", got)
	}
}

func TestTheConceptGridIsThreeByFourAndFilled(t *testing.T) {
	// Three categories by four cost tiers, one concept per cell. This is the structural claim
	// MECHANICS.md makes about the card set, and it is the thing that quietly breaks the first
	// time somebody adds a fifth attack or a second 2-AP defence.
	//
	// It also catches a concept falling through Cost()'s default arm, which returns a Strike's
	// price — a mistake that would otherwise hide behind an entirely plausible number.
	tiers := map[Category]map[int]ActionKind{}

	for _, a := range AllActions {
		if a.Cost() < 1 || a.Cost() > 4 {
			t.Errorf("%v costs %d, outside the 1-4 tiers the grid is built on", a, a.Cost())
		}

		cat := a.Category()
		if tiers[cat] == nil {
			tiers[cat] = map[int]ActionKind{}
		}
		if other, taken := tiers[cat][a.Cost()]; taken {
			t.Errorf("%v and %v are both %v at %d AP — the grid holds one concept per cell", other, a, cat, a.Cost())
		}
		tiers[cat][a.Cost()] = a
	}

	for _, cat := range Categories() {
		if len(tiers[cat]) != 4 {
			t.Errorf("%v has %d of 4 cost tiers filled", cat, len(tiers[cat]))
		}
	}
}

func TestParseActionRoundTripsEveryConcept(t *testing.T) {
	// cards.json names concepts by string, so this is the join between the deck data and the
	// rules. A concept whose name does not round-trip is five cards missing from the deck.
	for _, a := range AllActions {
		got, ok := ParseAction(a.String())
		if !ok {
			t.Errorf("ParseAction(%q) failed for a concept the rules define", a.String())
			continue
		}
		if got != a {
			t.Errorf("ParseAction(%q) = %v, want %v", a.String(), got, a)
		}
	}

	if _, ok := ParseAction("Fient"); ok {
		t.Error("ParseAction accepted a typo — it must report failure so a bad deck fails loudly")
	}
}
