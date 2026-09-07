package session

import (
	"math/rand"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// **Both are found by target rather than by key**, on the rule the rest of this package's parasite
// tests follow: what is under test is the grammar, and a record key is the one thing here that a
// rename can break without changing any behaviour.
func luckParasite(t *testing.T) Parasite {
	t.Helper()
	return anyWithTarget(t, ParasiteLuck)
}

func chimeraParasite(t *testing.T) Parasite {
	t.Helper()
	return anyWithTarget(t, ParasiteChimera)
}

// The gamble's whole promise: one roll, three outcomes, and the two rewards never together.
func TestALuckRollPaysOneThingOrNothing(t *testing.T) {
	p := luckParasite(t)

	dmg, life, dud := 0, 0, 0
	for seed := int64(0); seed < 300; seed++ {
		s := runWith(combat.Card{})
		if !s.ApplyParasiteRolling(p, nil, rand.New(rand.NewSource(seed))) {
			t.Fatalf("seed %d: a luck parasite refused to be spent", seed)
		}

		got := s.Lucked()
		if got.DMG != 0 && got.Life != 0 {
			t.Fatalf("seed %d: one roll paid both %d DMG and %d life", seed, got.DMG, got.Life)
		}
		if got.DMG != 0 && got.DMG != LuckDMG {
			t.Fatalf("seed %d: paid %d DMG, and a win is worth %d", seed, got.DMG, LuckDMG)
		}
		if got.Life != 0 && got.Life != LuckLife {
			t.Fatalf("seed %d: paid %d life, and a win is worth %d", seed, got.Life, LuckLife)
		}

		// The run's own figures have to have moved by exactly what was announced, since the
		// announcement is all the screen ever sees.
		if s.DMGBonus() != got.DMG || s.LifeBonus() != got.Life {
			t.Fatalf("seed %d: announced %+v and the run holds %d DMG / %d life",
				seed, got, s.DMGBonus(), s.LifeBonus())
		}

		switch {
		case got.DMG != 0:
			dmg++
		case got.Life != 0:
			life++
		default:
			dud++
		}
	}

	// Not a distribution test — the point is only that all three faces are reachable, since a
	// switch that could never take a branch is the failure a seeded roll hides.
	if dmg == 0 || life == 0 || dud == 0 {
		t.Errorf("300 rolls came to %d DMG, %d life, %d nothing — a face nothing lands on", dmg, life, dud)
	}
}

// A dud is still a spending, and it still moves the cursor. Without this two empty rolls in a row
// would be seeded identically and the second could never come up different.
func TestALuckRollStepsTheCounterWhateverItPays(t *testing.T) {
	p := luckParasite(t)
	s := runWith(combat.Card{})

	for i := 1; i <= 5; i++ {
		if !s.ApplyParasiteRolling(p, nil, rand.New(rand.NewSource(int64(i)))) {
			t.Fatalf("roll %d refused", i)
		}
		if s.LuckRolls() != i {
			t.Fatalf("after %d rolls the run has counted %d", i, s.LuckRolls())
		}
	}
}

// It rolls, so it must be refused without a source rather than falling back to a default draw —
// the posture the rock shower already takes.
func TestALuckParasiteWithoutASourceIsRefused(t *testing.T) {
	s := runWith(combat.Card{})
	if s.ApplyParasite(luckParasite(t), nil) {
		t.Error("a luck parasite was spent with no source to roll against")
	}
}

func TestAChimeraWithNothingToCopyIsRefused(t *testing.T) {
	s := runWith(combat.Card{Concept: combat.Strike})
	c := chimeraParasite(t)

	if _, ok := s.Echoes(c); ok {
		t.Error("a chimera on a fresh run resolved to something")
	}
	if s.CanApplyParasite(c, nil) {
		t.Error("a chimera on a fresh run reported itself spendable")
	}
	if s.ApplyParasite(c, nil) {
		t.Error("a chimera on a fresh run was spent")
	}
}

// The echo's whole shape: what the chimera costs, asks for and does all come from the parasite
// behind it.
func TestAChimeraFiresTheLastParasiteAgain(t *testing.T) {
	hoard := anyWithTarget(t, ParasiteVitae)
	s := runWith(combat.Card{Concept: combat.Strike})

	opened := s.Vitae()
	if !s.ApplyParasite(hoard, nil) {
		t.Fatal("hoard refused")
	}
	paid := s.Vitae() - opened

	echoed, ok := s.Echoes(chimeraParasite(t))
	if !ok || echoed.Record != hoard.Record {
		t.Fatalf("a chimera behind a hoard resolves to %q/%v", echoed.Record, ok)
	}
	if !s.ApplyParasite(chimeraParasite(t), nil) {
		t.Fatal("a chimera behind a hoard refused")
	}
	if got := s.Vitae() - opened; got != paid*2 {
		t.Errorf("hoard paid %d and the pair of them came to %d", paid, got)
	}
}

// A chimera never becomes the thing to copy, so two in a row both fire what is behind them rather
// than the second copying the first into nothing.
func TestAChimeraNeverRemembersItself(t *testing.T) {
	hoard := anyWithTarget(t, ParasiteVitae)
	s := runWith(combat.Card{Concept: combat.Strike})

	if !s.ApplyParasite(hoard, nil) {
		t.Fatal("hoard refused")
	}
	for i := 0; i < 2; i++ {
		if !s.ApplyParasite(chimeraParasite(t), nil) {
			t.Fatalf("chimera %d refused", i+1)
		}
		if s.LastParasite() != hoard.Record {
			t.Fatalf("after chimera %d the run remembers %q", i+1, s.LastParasite())
		}
	}
}

// A chimera behind a card-eating parasite asks for that one's cards, by that one's rules.
func TestAChimeraInheritsTheCountOfWhatItCopies(t *testing.T) {
	bore := anyWithTarget(t, ParasiteElement)
	// **Every card starts on a colour the parasite is not**, since a recolour onto the colour a
	// card already is is refused — so the starting element is derived from the parasite rather
	// than picked, and this goes on working whichever element parasite the catalogue hands over.
	other := combat.Fire
	if bore.Element == other {
		other = combat.Ice
	}
	s := runWith(
		combat.Card{Concept: combat.Strike, Element: other},
		combat.Card{Concept: combat.Jab, Element: other},
		combat.Card{Concept: combat.Ward, Element: other},
		combat.Card{Concept: combat.Cut, Element: other},
	)

	all := ids(s)
	if !s.ApplyParasite(bore, all[:2]) {
		t.Fatal("emberbore refused")
	}

	c := chimeraParasite(t)
	if echoed, _ := s.Echoes(c); echoed.Count != bore.Count {
		t.Fatalf("a chimera behind %s asks for %d cards, not %d", bore.Record, echoed.Count, bore.Count)
	}
	// The wrong number of cards is refused exactly as it is for the parasite itself.
	if s.CanApplyParasite(c, all[2:3]) {
		t.Error("a chimera behind a two-card parasite accepted one card")
	}
	if !s.ApplyParasite(c, all[2:]) {
		t.Fatal("a chimera behind an element parasite refused two fresh cards")
	}
	for _, card := range s.Deck() {
		if card.Element != bore.Element {
			t.Fatalf("card %d is %v after two firings of %s, wanted %v",
				card.ID, card.Element, bore.Record, bore.Element)
		}
	}
}

// The memory is the run's, so it survives the snapshot a resume is rebuilt from.
func TestTheChimerasMemorySurvivesASave(t *testing.T) {
	hoard := anyWithTarget(t, ParasiteVitae)
	s := runWith(combat.Card{Concept: combat.Strike})
	if !s.ApplyParasite(hoard, nil) {
		t.Fatal("hoard refused")
	}
	if !s.ApplyParasiteRolling(luckParasite(t), nil, rand.New(rand.NewSource(1))) {
		t.Fatal("luck refused")
	}

	snap := s.Snapshot(0)
	if snap.LastParasite != luckParasite(t).Record {
		t.Fatalf("the snapshot remembers %q", snap.LastParasite)
	}
	if snap.LuckRolls != 1 {
		t.Fatalf("the snapshot counted %d rolls", snap.LuckRolls)
	}
}
