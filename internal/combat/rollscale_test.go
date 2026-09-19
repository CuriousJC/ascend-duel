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

// shockedDuelist is a duelist carrying whatever status the file registers as the miss chance, so
// the test names the effect kind rather than a record key that could be renamed.
func shockedDuelist(t *testing.T, d Duelist) Duelist {
	t.Helper()

	for _, id := range AllStatuses() {
		if StatusOf(id).Effect != EffectMissChance {
			continue
		}
		out, _, ok := applyStatus(d, id, Duelist{})
		if !ok {
			t.Fatalf("%v would not apply", StatusOf(id).Key)
		}
		return out
	}
	t.Fatal("no status in the file is a miss chance")
	return d
}

func rollsRelic(t *testing.T, key string, pct int) RelicID {
	t.Helper()

	return relic(t, key, RelicRule{
		When: MomentFightStart,
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

func TestTheShockRollDoublesAgainstItsOwnWearer(t *testing.T) {
	id := rollsRelic(t, "rolls.shock", 200)

	d := shockedDuelist(t, duelist(10, 3, 100))
	bare := d.MissChance()

	lucky := d.Wearing(WornRelic{Relic: id})
	if got, want := lucky.MissChance(), bare*2; got != want {
		t.Errorf("a shocked duelist wearing a doubling relic misses %d%% of the time, wanted %d%%", got, want)
	}
}

func TestTheMissChanceIsStillCapped(t *testing.T) {
	// **Nothing misses every time**, which is maxStatusPct's line — and the scale is applied
	// before the cap, or doubling a figure already held at 99 would do nothing at all.
	id := rollsRelic(t, "rolls.cap", 100000)

	d := shockedDuelist(t, duelist(10, 3, 100)).Wearing(WornRelic{Relic: id})

	if got := d.MissChance(); got > maxStatusPct {
		t.Errorf("miss chance reached %d%%, and the ceiling is %d%%", got, maxStatusPct)
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
