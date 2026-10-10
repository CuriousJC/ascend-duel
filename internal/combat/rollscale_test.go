package combat

import (
	"math/rand"
	"testing"
)

// The scale-rolls verb: a relic that doubles the numerator of every roll in the rules.
//
// **The two halves that can break independently** are the arithmetic — a wider paying band on the
// same die, never a smaller die — and the determinism, which is that the stream is drawn from
// exactly as often whether the relic is worn or not.

func rollsRelic(t *testing.T, key string, pct int) RelicID {
	t.Helper()

	return relic(t, key, RelicRule{
		When: MomentEquipped,
		Then: []RelicEffect{{Do: DoScaleRolls, Amount: pct}},
	})
}

func TestTheIdentityScaleIsTheDieTheGameAlwaysHad(t *testing.T) {
	// 1 in 5 damage, 1 in 5 life, 3 in 5 nothing — the d5 rollGolden was written as.
	dmg, life := 0, 0
	for seed := int64(0); seed < 5000; seed++ {
		d, l := rollGolden(5, 100, rand.New(rand.NewSource(seed)))
		if d > 0 {
			dmg++
		}
		if l > 0 {
			life++
		}
	}
	if dmg < 900 || dmg > 1100 || life < 900 || life > 1100 {
		t.Errorf("at the identity scale a d5 paid %d DMG and %d life in 5000, wanted about 1000 each", dmg, life)
	}
}

func TestDoublingTheScaleDoublesThePayingBand(t *testing.T) {
	dmg, life := 0, 0
	for seed := int64(0); seed < 5000; seed++ {
		d, l := rollGolden(5, 200, rand.New(rand.NewSource(seed)))
		if d > 0 {
			dmg++
		}
		if l > 0 {
			life++
		}
	}
	if dmg < 1900 || dmg > 2100 || life < 1900 || life > 2100 {
		t.Errorf("at 2x a d5 paid %d DMG and %d life in 5000, wanted about 2000 each", dmg, life)
	}
}

func TestAGambleTakesOneSampleWhateverTheScale(t *testing.T) {
	// **The determinism half, and the reason the die is widened rather than rolled twice.** A
	// second draw would advance the luck cursor, so wearing the relic would reroll every later
	// gamble in the run. Two sources that have taken the same number of samples are still in step.
	bare := rand.New(rand.NewSource(7))
	lucky := rand.New(rand.NewSource(7))

	for i := 0; i < 50; i++ {
		rollGolden(5, 100, bare)
		rollGolden(5, 400, lucky)
	}
	if bare.Int63() != lucky.Int63() {
		t.Error("the two sources have diverged, so the relic changed how often the stream is drawn from")
	}
}

func TestALosingFaceAlwaysSurvives(t *testing.T) {
	// **LuckOutcomes' rule, held at runtime.** A relic wide enough to cover the die would make a
	// gamble a purchase, which is the thing the loader refuses a record for doing.
	for _, scale := range []int{200, 400, 1000, 100000} {
		paid := 0
		for seed := int64(0); seed < 2000; seed++ {
			if d, l := rollGolden(5, scale, rand.New(rand.NewSource(seed))); d > 0 || l > 0 {
				paid++
			}
		}
		if paid == 2000 {
			t.Errorf("at %d%% a golden card paid on every one of 2000 plays, which is a purchase", scale)
		}
	}
}

func TestTwoRollRelicsCompound(t *testing.T) {
	first := rollsRelic(t, "rolls.c1", 200)
	second := rollsRelic(t, "rolls.c2", 150)

	worn := []WornRelic{{Relic: first}, {Relic: second}}
	if got := RollScale(worn); got != 300 {
		t.Errorf("2x then 1.5x came to %d%%, wanted 300%% — a multiplier compounds", got)
	}
}

func TestTheOddsAPlayerReadsAreTheOddsTheDieRolls(t *testing.T) {
	// **LuckOdds is what carddesc prints**, so this is the tripwire on the tooltip lying.
	if num, den := LuckOdds(5, 100, 2); num != 1 || den != 5 {
		t.Errorf("a bare golden card reads %d in %d, wanted 1 in 5", num, den)
	}
	if num, den := LuckOdds(5, 200, 2); num != 2 || den != 5 {
		t.Errorf("a doubled golden card reads %d in %d, wanted 2 in 5", num, den)
	}
	if num, den := LuckOdds(5, 200, 1); num != 2 || den != 5 {
		t.Errorf("a doubled silver card reads %d in %d, wanted 2 in 5", num, den)
	}
}
