package combat

import "testing"

// A stone raises one rung of the ladder for one duelist. These are the four things that can break
// silently: the arithmetic, whose ladder is read, whether a duelist with no stones still reads the
// shipped one, and whether the resolver actually pays the raised figure.

func TestAStoneIsATenthOfTheRungsOwnMultiplier(t *testing.T) {
	for _, h := range Hands() {
		if got, want := StoneValue(h.Multiplier, 1), h.Multiplier/10; got != want {
			t.Errorf("%s: one stone is worth %d, want %d", h.Key, got, want)
		}
	}
}

// **Additive on the base, never compounding**, which is the owner's call the whole mechanic is
// priced off: the tenth stone is worth exactly what the first was.
func TestStonesStackOnTheBaseRatherThanCompounding(t *testing.T) {
	const base = 115

	if got, want := StoneValue(base, 2), 22; got != want {
		t.Errorf("two stones on %d are worth %d, want %d", base, got, want)
	}
	if got, want := StoneValue(base, 3), 33; got != want {
		t.Errorf("three stones on %d are worth %d, want %d", base, got, want)
	}
	// Compounding would give 115 -> 126 -> 138, so a second stone worth 12 rather than 11 is the
	// failure this pins.
	if StoneValue(base, 2) != 2*StoneValue(base, 1) {
		t.Error("the second stone is not worth the same as the first")
	}
}

func TestNoStonesReadsTheCatalogueUntouched(t *testing.T) {
	var d Duelist

	table := d.HandTable()
	if len(table) != len(handTable) {
		t.Fatalf("a bare duelist reads %d rungs, want %d", len(table), len(handTable))
	}
	for i := range table {
		if table[i].Multiplier != handTable[i].Multiplier {
			t.Errorf("%s pays %d for a duelist with no stones, want the catalogue's %d",
				table[i].Key, table[i].Multiplier, handTable[i].Multiplier)
		}
	}
}

func TestAStoneRaisesOnlyItsOwnRung(t *testing.T) {
	d, ok := Duelist{}.WithHandStone("concept-pair")
	if !ok {
		t.Fatal("concept-pair is not a rung the catalogue holds")
	}

	for _, h := range d.HandTable() {
		base, found := HandByName(h.Name)
		if !found {
			t.Fatalf("%s is not in the catalogue", h.Key)
		}
		want := base.Multiplier
		if h.Key == "concept-pair" {
			want += StoneValue(base.Multiplier, 1)
		}
		if h.Multiplier != want {
			t.Errorf("%s pays %d, want %d", h.Key, h.Multiplier, want)
		}
	}
}

// A stone naming a rung the catalogue has not got is refused rather than landing on seat zero,
// which is the High Card — the failure the bool on HandSlot exists to prevent.
func TestAStoneOnANonexistentRungIsRefused(t *testing.T) {
	if _, ok := HandSlot("no-such-hand"); ok {
		t.Fatal("HandSlot found a rung that does not exist")
	}
	d, ok := Duelist{}.WithHandStone("no-such-hand")
	if ok {
		t.Error("a stone was accepted for a rung the catalogue does not hold")
	}
	if d != (Duelist{}) {
		t.Error("a refused stone still changed the duelist")
	}
}

// **The blow is worth the raised figure**, which is the whole point and the thing that would break
// most quietly: the ladder could be right everywhere a screen reads it and still not reach the
// resolver.
func TestARaisedRungPaysMoreInARealRound(t *testing.T) {
	pair := twoOfAKind()

	plain := Duelist{DMG: 10, Actions: 6, MaxLife: 100, CurrentLife: 100}
	stoned, ok := plain.WithHandStone(pair.Key)
	if !ok {
		t.Fatalf("%s is not a rung the catalogue holds", pair.Key)
	}

	before := blowDamage(t, plain)
	after := blowDamage(t, stoned)

	if after <= before {
		t.Fatalf("a stone on %s dealt %d, no more than the %d it dealt without one",
			pair.Key, after, before)
	}

	// The figures are the multiplier's, so the ratio has to be exactly the two multipliers'.
	wantBefore := blowBase(plain) * pair.Multiplier / multiplierScale
	wantAfter := blowBase(plain) *
		(pair.Multiplier + StoneValue(pair.Multiplier, 1)) / multiplierScale
	if before != wantBefore || after != wantAfter {
		t.Errorf("dealt %d then %d, want %d then %d", before, after, wantBefore, wantAfter)
	}
}

// twoOfAKind is the card pair, which is the rung the test above builds.
func twoOfAKind() Hand {
	h, _ := HandByID(mustHandID("concept-pair"))
	return h
}

func mustHandID(key string) HandID {
	id, _ := HandIDForKey(key)
	return id
}

// blowDamage resolves one round of two identical Strikes and reports what landed.
func blowDamage(t *testing.T, d Duelist) int {
	t.Helper()

	target := Duelist{DMG: 10, Actions: 6, MaxLife: 500, CurrentLife: 500}
	cards := []Card{Plain(Strike), Plain(Strike)}

	_, _, after := ResolveRound(d, target, cards, nil, 1, nil)
	return target.CurrentLife - after.CurrentLife
}

// blowBase is what the two Strikes are worth before any multiplier.
func blowBase(d Duelist) int {
	return 2 * Plain(Strike).Damage(d.DMG)
}
