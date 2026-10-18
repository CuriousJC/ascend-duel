package game

import (
	"slices"
	"sort"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// TestEveryScreenNamesItsMusic is what makes "every screen is listed" in score.go a fact rather
// than an intention.
//
// **A screen left out is not a crash and not a silence** — updateScore falls back to the score,
// which is usually the right answer, so the omission would work correctly and say nothing. That
// is exactly the kind of gap this file exists to close: the table is the page somebody reads to
// find out what plays where, and a screen missing from it is a screen that page lies about.
//
// The walk relies on String returning "Unknown" past the end of the enum, which is also what
// makes adding a screen to state.ActiveScreen enough to fail this.
func TestEveryScreenNamesItsMusic(t *testing.T) {
	for s := state.ActiveScreen(0); s.String() != "Unknown"; s++ {
		if _, listed := screenTunes[s]; !listed {
			t.Errorf("screen %v is not in screenTunes: say which music it plays, or {keep: true} if it is opened from somewhere and returns there", s)
		}
	}
}

// TestEveryTrackPlaysWhereItMay holds score.go against data/music.json, the other place a track
// name is written down.
//
// **It reads the catalog rather than the directory**, and that is the whole reason it can exist:
// the audio is not in git, so a test looking for files would fail on every clean clone, every fork
// and every CI run. The catalog *is* committed, so what is checkable everywhere is that a name
// score.go asks for is a track that ships, on a screen its record says it may play on.
//
// **The reverse is reported and not failed.** A track no screen plays is a perfectly legitimate
// state — it is bought, made game-ready and waiting for a screen — but it is invisible otherwise,
// and a track nobody remembers owning is one nobody spends.
func TestEveryTrackPlaysWhereItMay(t *testing.T) {
	screens := map[string]bool{}
	for s := state.ActiveScreen(0); s.String() != "Unknown"; s++ {
		screens[s.String()] = true
	}

	pool := map[string][]string{}
	for _, tr := range data.LoadAudio().Music {
		pool[tr.Name()] = tr.Screens
		for _, s := range tr.Screens {
			if !screens[s] {
				t.Errorf("music.json: %s may play on %q, which is no screen", tr.Name(), s)
			}
		}
	}

	played := map[string]bool{}
	for screen, tn := range screenTunes {
		if tn.keep {
			continue
		}
		if tn.pool {
			// A pool screen plays whatever music.json lets it, so the check is only that something
			// does: an empty pool is a screen quietly playing the score.
			if len(musicPools[screen.String()]) == 0 {
				t.Errorf("screen %v plays from its pool, and no track in data/music.json names it", screen)
			}
			for _, name := range musicPools[screen.String()] {
				played[name] = true
			}
			continue
		}
		played[tn.track] = true
		may, ships := pool[tn.track]
		switch {
		case !ships:
			t.Errorf("screen %v plays %q, which data/music.json does not name", screen, tn.track)
		case !slices.Contains(may, screen.String()):
			t.Errorf("screen %v plays %q, whose Screens in data/music.json do not include it", screen, tn.track)
		}
	}

	// A pool may only be on a pool screen: a track listing a screen that names one fixed track would
	// be a pool nothing draws from.
	for screenName, names := range musicPools {
		for s, tn := range screenTunes {
			if s.String() == screenName && !tn.pool && !tn.keep {
				for _, name := range names {
					if name != tn.track {
						t.Errorf("music.json: %s may play on %s, which always plays %q", name, screenName, tn.track)
					}
				}
			}
		}
	}

	var spare []string
	for name := range pool {
		if !played[name] {
			spare = append(spare, name)
		}
	}
	sort.Strings(spare)
	for _, name := range spare {
		t.Logf("%q ships and no screen plays it", name)
	}
}

// TestAFightPlaysOneTrackOfItsPool holds the pick to the pool and to the fight: the same run and
// fight always land on the same track, and across fights every track in the pool comes up.
func TestAFightPlaysOneTrackOfItsPool(t *testing.T) {
	pool := musicPools[state.Combat.String()]
	seen := map[string]bool{}
	for fight := 0; fight < 64; fight++ {
		got := poolPick(state.Combat, 12345, fight)
		if !slices.Contains(pool, got) {
			t.Fatalf("fight %d plays %q, which is not in the combat pool %v", fight, got, pool)
		}
		if again := poolPick(state.Combat, 12345, fight); again != got {
			t.Fatalf("fight %d played %q and then %q", fight, got, again)
		}
		seen[got] = true
	}
	if len(seen) != len(pool) {
		t.Errorf("64 fights played %d of the combat pool's %d tracks", len(seen), len(pool))
	}
}
