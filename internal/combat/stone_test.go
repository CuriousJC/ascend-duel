package combat

import (
	"reflect"
	"testing"
)

// A stone raises one rung of the ladder for one duelist. These are the four things that can break
// silently: the arithmetic, whose ladder is read, whether a duelist with no stones still reads the
// shipped one, and whether the resolver actually pays the raised figure.

func TestAStoneIsATenthOfTheRungsOwnMultiplier(t *testing.T) {
	for _, h := range Hands() {
		if h.StonePercent != 0 {
			continue
		}
		if got, want := StoneValue(h, 1), h.Multiplier/10/handStep*handStep; got != want {
			t.Errorf("%s: one stone is worth %d, want %d", h.Key, got, want)
		}
	}
}

// **The No Hand grows at three times the ladder's rate**, written on its own record: playing many
// cards that agree on nothing is the hard thing it rewards.
func TestTheNoHandGrowsAtThirtyPercent(t *testing.T) {
	for _, h := range Hands() {
		if h.Key != "no-hand" {
			continue
		}
		for n, want := range []int{0, 30, 60, 90, 120, 150} {
			if got := StoneValue(h, n); got != want {
				t.Errorf("%d stones on the No Hand are worth %d, want %d", n, got, want)
			}
		}
		return
	}
	t.Fatal("the catalog holds no No Hand")
}

// **A raised rung stays on the tenths grid** the catalog is written on, however many stones.
func TestARaisedRungStaysOnTheTenths(t *testing.T) {
	for _, h := range Hands() {
		for n := 1; n <= 10; n++ {
			if v := h.Multiplier + StoneValue(h, n); v%handStep != 0 {
				t.Errorf("%s with %d stones pays %d, off the tenths grid", h.Key, n, v)
			}
		}
	}
}

// **The total is floored, never each stone**, so a fraction one stone cannot show is carried into
// the next. A 1.4x rung and the 1.0x Pair both gain a tenth from their first stone; by the third
// the harder rung has pulled ahead.
func TestStonesCarryTheirFractionForward(t *testing.T) {
	h := Hand{Multiplier: 140}
	for n, want := range []int{0, 10, 20, 40, 50, 70} {
		if got := StoneValue(h, n); got != want {
			t.Errorf("%d stones on 140 are worth %d, want %d", n, got, want)
		}
	}
}

// **Additive on the base, never compounding**, which is the owner's call the whole mechanic is
// priced off: ten stones are worth exactly ten tenths of the catalog figure.
func TestStonesStackOnTheBaseRatherThanCompounding(t *testing.T) {
	h := Hand{Multiplier: 450}

	if got, want := StoneValue(h, 2), 90; got != want {
		t.Errorf("two stones on %d are worth %d, want %d", h.Multiplier, got, want)
	}
	// Compounding would put ten stones at about 2.6 times the base; additive is exactly double.
	if got, want := StoneValue(h, 10), 450; got != want {
		t.Errorf("ten stones on %d are worth %d, want %d", h.Multiplier, got, want)
	}
}

func TestNoStonesReadsTheCatalogUntouched(t *testing.T) {
	var d Duelist

	table := d.HandTable()
	if len(table) != len(handTable) {
		t.Fatalf("a bare duelist reads %d rungs, want %d", len(table), len(handTable))
	}
	for i := range table {
		if table[i].Multiplier != handTable[i].Multiplier {
			t.Errorf("%s pays %d for a duelist with no stones, want the catalog's %d",
				table[i].Key, table[i].Multiplier, handTable[i].Multiplier)
		}
	}
}

func TestAStoneRaisesOnlyItsOwnRung(t *testing.T) {
	d, ok := Duelist{}.WithHandStone("pair")
	if !ok {
		t.Fatal("concept-pair is not a rung the catalog holds")
	}

	for _, h := range d.HandTable() {
		base, found := HandByName(h.Name)
		if !found {
			t.Fatalf("%s is not in the catalog", h.Key)
		}
		want := base.Multiplier
		if h.Key == "pair" {
			want += StoneValue(base, 1)
		}
		if h.Multiplier != want {
			t.Errorf("%s pays %d, want %d", h.Key, h.Multiplier, want)
		}
	}
}

// A stone naming a rung the catalog has not got is refused rather than landing on seat zero,
// which is the No Hand — the failure the bool on HandSlot exists to prevent.
func TestAStoneOnANonexistentRungIsRefused(t *testing.T) {
	if _, ok := HandSlot("no-such-hand"); ok {
		t.Fatal("HandSlot found a rung that does not exist")
	}
	d, ok := Duelist{}.WithHandStone("no-such-hand")
	if ok {
		t.Error("a stone was accepted for a rung the catalog does not hold")
	}
	if !reflect.DeepEqual(d, Duelist{}) {
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
		t.Fatalf("%s is not a rung the catalog holds", pair.Key)
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
		(pair.Multiplier + StoneValue(pair, 1)) / multiplierScale
	if before != wantBefore || after != wantAfter {
		t.Errorf("dealt %d then %d, want %d then %d", before, after, wantBefore, wantAfter)
	}
}

// twoOfAKind is the card pair, which is the rung the test above builds.
func twoOfAKind() Hand {
	h, _ := HandByID(mustHandID("pair"))
	return h
}

func mustHandID(key string) HandID {
	id, _ := HandIDForKey(key)
	return id
}

// blowDamage resolves one round of two identical Bashes and reports what landed.
func blowDamage(t *testing.T, d Duelist) int {
	t.Helper()

	target := Duelist{DMG: 10, Actions: 6, MaxLife: 500, CurrentLife: 500}
	cards := []Card{Plain(Bash), Plain(Bash)}

	_, _, after := ResolveRound(d, target, cards, nil, 1, Sources{})
	return target.CurrentLife - after.CurrentLife
}

// blowBase is what the two Bashes are worth before any multiplier.
func blowBase(d Duelist) int {
	return 2 * Plain(Bash).Damage(d.DMG)
}

// A shape is the rung's groups with the axis left out, so every axis' version of one hand shares
// it and no two different hands do.
func TestAShapeGathersEveryAxisOfOneHand(t *testing.T) {
	if got, want := HandsShaped("3"), []string{
		"concept-three-of-a-kind", "form-three-of-a-kind", "element-three-of-a-kind",
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("shape 3 is %v, want %v", got, want)
	}
	if got := HandsShaped("2"); !reflect.DeepEqual(got, []string{"pair"}) {
		t.Errorf("shape 2 is %v, want only the merged pair", got)
	}
	if got := HandsShaped("9"); len(got) != 0 {
		t.Errorf("a shape no rung carries names %v", got)
	}

	for _, shape := range HandShapes() {
		keys := HandsShaped(shape)
		if len(keys) == 0 {
			t.Errorf("shape %s is listed and carries no rung", shape)
		}
		for _, key := range keys {
			for _, h := range handTable {
				if h.Key == key && !reflect.DeepEqual(ShapeOf(h.Groups), shape) {
					t.Errorf("%s is filed under %s and its groups spell %s", key, shape, ShapeOf(h.Groups))
				}
			}
		}
	}
}
