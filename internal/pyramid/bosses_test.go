package pyramid

import (
	"math/rand"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// The stairway placement. Like roster_test.go these build their own records rather than loading
// `data/bosses.json`, so they test the placement rule and not whatever the pool happens to hold.

func testBosses() map[string]data.BossData {
	recs := map[string]data.BossData{}
	for _, b := range []struct {
		name  string
		floor int
	}{
		{"tollman", 1}, {"gatekeeper", 1},
		{"watchman", 2},
		{"warden", 3}, {"hexwright", 3},
	} {
		recs[b.name] = data.BossData{BossRecord: b.name, Floor: b.floor}
	}
	return recs
}

func isBoss(name string, recs map[string]data.BossData) bool {
	_, ok := recs[name]
	return ok
}

func TestABossStandsOnEveryStairwayAndNowhereElse(t *testing.T) {
	enemies, bosses := testRoster(), testBosses()
	p := New(enemies, bosses, rand.New(rand.NewSource(7)))

	for fight := 0; fight < 24; fight++ {
		got := p.EnemyAt(fight)
		wantBoss := RoomOf(fight) == RoomStairway
		if isBoss(got, bosses) != wantBoss {
			t.Fatalf("fight %d (room %d) drew %q; boss=%v, wanted boss=%v",
				fight, RoomOf(fight), got, isBoss(got, bosses), wantBoss)
		}
	}
}

// A boss standing in a room must not also consume a creature: the roster index skips the
// stairways, so the two ordinary rooms of floor 2 carry on from floor 1's rather than starting
// three records later.
func TestAStairwayDoesNotEatARosterEntry(t *testing.T) {
	enemies, bosses := testRoster(), testBosses()
	p := New(enemies, bosses, rand.New(rand.NewSource(3)))

	var ordinary []string
	for fight := 0; fight < 8; fight++ {
		if RoomOf(fight) != RoomStairway {
			ordinary = append(ordinary, p.EnemyAt(fight))
		}
	}

	for i, name := range ordinary {
		if name != p.order[i] {
			t.Fatalf("ordinary room %d drew %q, want %q — the roster index skipped one", i, name, p.order[i])
		}
	}
}

// The pool is authored for floors 1 to 8 and nothing caps the fight index, so a run played past
// the top has to keep finding somebody rather than reaching an empty stairway.
func TestTheBossesWrapPastTheTopFloor(t *testing.T) {
	p := New(testRoster(), testBosses(), rand.New(rand.NewSource(11)))

	if got, want := p.BossAt(4), p.BossAt(1); got != want {
		t.Fatalf("floor 4 drew %q, want floor 1's %q — the pool covers three floors", got, want)
	}
	if p.BossAt(0) != p.BossAt(1) {
		t.Fatalf("a floor below the first should answer as the first")
	}
}

// A run picks one of a floor's bosses, and two runs off different seeds should be able to
// disagree — otherwise the pool's extra entries are unreachable.
func TestTwoRunsCanDrawDifferentBosses(t *testing.T) {
	enemies, bosses := testRoster(), testBosses()

	first := New(enemies, bosses, rand.New(rand.NewSource(1))).BossAt(1)
	for seed := int64(2); seed < 40; seed++ {
		if New(enemies, bosses, rand.New(rand.NewSource(seed))).BossAt(1) != first {
			return
		}
	}
	t.Fatal("forty seeds all drew the same boss for floor 1")
}

// The shipped file is the one this is really about: a floor with nobody authored for it would
// hand the run the floor above's boss, which is a fallback and not a design.
func TestTheShippedPoolCoversEveryFloor(t *testing.T) {
	bosses := data.LoadBosses()
	seen := map[int]bool{}
	for _, b := range bosses {
		seen[b.Floor] = true
	}
	for floor := 1; floor <= 8; floor++ {
		if !seen[floor] {
			t.Errorf("no boss guards floor %d's stairway", floor)
		}
	}
}

// heaviestBlow is the hardest single hit a deck can land at this DMG, which is what a duelist
// actually feels. A card's Amount is a percentage of the wielder's DMG — see data.CardData — so
// neither number means anything on its own: a DMG of 6 behind a 300 card hits harder than a DMG
// of 9 behind a 200 one.
func heaviestBlow(dmg int, cards []data.CardData) int {
	worst := 0
	for _, c := range cards {
		if c.Verb != "attack" {
			continue
		}
		if blow := dmg * c.Amount / 100; blow > worst {
			worst = blow
		}
	}
	return worst
}

// A stairway that is easier than the room before it is not a boss. This is the only place the
// two files are compared, and it is deliberately a floor-by-floor check rather than a curve: the
// ascent multiplies both sides by the same amount, so the authored numbers are what decide it.
//
// **It compares blows rather than the DMG stat** *(2026-09-13)*. It read `b.DMG` against the
// enemy's for as long as the two pools were authored on one scale, and stopped being true the
// moment they were not: boss cards sit at 250-300 where a creature's sit at 100-200, so halving
// every boss's DMG left all thirty hitting at least as hard as their floor and failed this test
// thirty times over. Half of a two-factor product is not the product. HP takes no such treatment
// because HP is HP.
func TestABossIsToughAgainstTheFloorItGuards(t *testing.T) {
	enemies := data.LoadEnemies()
	bosses := data.LoadBosses()

	for _, key := range data.BossOrder(bosses) {
		b := bosses[key]

		hp, blow, found := 0, 0, false
		for _, e := range enemies {
			if !e.AllowsFloor(b.Floor) {
				continue
			}
			found = true
			if e.HP > hp {
				hp = e.HP
			}
			if w := heaviestBlow(e.DMG, e.Cards); w > blow {
				blow = w
			}
		}
		if !found {
			t.Errorf("no enemy is valid on floor %d, so %s guards a floor nothing else reaches", b.Floor, key)
			continue
		}

		if b.HP <= hp {
			t.Errorf("%s has %d HP against the toughest floor-%d enemy's %d", key, b.HP, b.Floor, hp)
		}
		// **Not-worse rather than strictly harder.** A boss ties the floor's hardest hitter on
		// floors 1 and 3 and outclasses it everywhere else, and what makes those two bosses is
		// the HP above and the deck full of heavy cards behind the one blow — a strict test here
		// would be tuning the roster to satisfy the check.
		if got := heaviestBlow(b.DMG, b.Cards); got < blow {
			t.Errorf("%s lands %d at its heaviest against the hardest-hitting floor-%d enemy's %d",
				key, got, b.Floor, blow)
		}
	}
}
