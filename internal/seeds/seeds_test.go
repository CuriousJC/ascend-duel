package seeds

import "testing"

// TestEverySaltIsDistinct is the check that could not exist before this package did. The salts
// used to live in three packages, so nothing ever saw them all at once and a duplicate would
// have been found by noticing that two streams dealt suspiciously similar things.
func TestEverySaltIsDistinct(t *testing.T) {
	seen := map[int64]Stream{}
	for _, s := range All() {
		salt := lookup(s).salt
		if salt == 0 {
			t.Errorf("%s has no salt, so it is seeded from the bare run seed", s)
		}
		if other, dup := seen[salt]; dup {
			t.Errorf("%s and %s share salt %#x, so they are the same sequence", s, other, salt)
		}
		seen[salt] = s
	}
}

// TestEveryStreamIsNamed guards the table against a stream added as a bare constant, which would
// index a zero entry — no salt, no name, and seeded from the run seed itself.
func TestEveryStreamIsNamed(t *testing.T) {
	for _, s := range All() {
		if lookup(s).name == "" {
			t.Errorf("stream %d has no name, so it is missing from the table", int(s))
		}
	}
}

// TestNoTwoSeedsCollide walks every stream against a spread of fights and run seeds. A collision
// means two concerns drawing the identical sequence, which is the failure the salts exist to
// prevent and the one that is invisible in play.
func TestNoTwoSeedsCollide(t *testing.T) {
	for _, runSeed := range []int64{0, 1, -1, 20260817, 0x2545F4914F6CDD1D} {
		seen := map[int64]string{}
		for _, s := range All() {
			if !lookup(s).perFight {
				record(t, seen, For(runSeed, s), s.String())
				continue
			}
			for fight := 0; fight < 24; fight++ {
				record(t, seen, ForFight(runSeed, s, fight), s.String())
			}
		}
	}
}

func record(t *testing.T, seen map[int64]string, seed int64, what string) {
	t.Helper()
	if prev, dup := seen[seed]; dup {
		t.Errorf("seed %d is dealt to both %s and %s", seed, prev, what)
	}
	seen[seed] = what
}

// TestAFightIsAFunctionOfItsIndex pins the property that makes a retry a replay: the same fight
// of the same run seeds identically however many times it is played, and neighbouring fights do
// not. Nothing re-rolls a run until Session exists, and a defeat must deal that fight again
// rather than a different one.
func TestAFightIsAFunctionOfItsIndex(t *testing.T) {
	const run = 20260817

	if a, b := ForFight(run, PlayerDeck, 3), ForFight(run, PlayerDeck, 3); a != b {
		t.Errorf("the same fight seeded differently twice: %d then %d", a, b)
	}
	if a, b := ForFight(run, PlayerDeck, 3), ForFight(run, PlayerDeck, 4); a == b {
		t.Errorf("fights 3 and 4 share seed %d", a)
	}

	// Fight zero must not cancel the stride and hand back the per-run seed. That is what the
	// +1 inside ForFight is for, and it is the arithmetic most likely to be "simplified" away.
	if ForFight(run, PlayerDeck, 0) == run^lookup(PlayerDeck).salt {
		t.Error("fight zero seeds as if it had no fight index at all")
	}
}

// TestTheStrideIsOdd guards the one property the stride's size is chosen for. An even stride
// would leave the low bit of every fight's seed decided by the run seed alone.
func TestTheStrideIsOdd(t *testing.T) {
	if fightStride%2 == 0 {
		t.Errorf("fightStride %#x is even", fightStride)
	}
}

// TestScopeIsEnforced pins that asking for the wrong kind of seed fails loudly. Seeding a
// per-fight stream as if it were per-run would hand every fight the same shuffle — which looks
// like it worked, which is why it panics rather than being merely discouraged.
func TestScopeIsEnforced(t *testing.T) {
	assertPanics(t, "For(PlayerDeck)", func() { For(1, PlayerDeck) })
	assertPanics(t, "ForFight(CombatRoll)", func() { ForFight(1, CombatRoll, 0) })
	assertPanics(t, "For(unknown stream)", func() { For(1, Stream(len(streams))) })
}

func assertPanics(t *testing.T, what string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s did not panic", what)
		}
	}()
	fn()
}
