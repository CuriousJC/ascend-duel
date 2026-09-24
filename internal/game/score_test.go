package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/music"
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

// TestEveryNamedTrackIsInTheBundle holds score.go against the manifest, which is the only other
// place a track name is written down.
//
// **It reads the manifest rather than the directory**, and that is the whole reason it can exist:
// the WAVs are not in git, so a test looking for files would fail on every clean clone, every
// fork and every CI run. The manifest *is* committed, so what is checkable everywhere is that a
// name score.go asks for is a name the bundle carries.
//
// **The reverse is reported and not failed.** A loop in the bundle that no screen plays is a
// perfectly legitimate state — it is bought, filed and waiting for a screen — but it is invisible
// otherwise, and a loop nobody remembers owning is one nobody spends.
func TestEveryNamedTrackIsInTheBundle(t *testing.T) {
	var man struct {
		Files []struct{ Name string }
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "privateassets", "audio", "manifest.json"))
	if err != nil {
		t.Fatalf("reading the bundle manifest: %v", err)
	}
	if err := json.Unmarshal(raw, &man); err != nil {
		t.Fatalf("reading the bundle manifest: %v", err)
	}

	// The manifest names files and score.go names tracks, and a track's name is its filename
	// stem — the same cut assets.embedFamily and privateassets.Audio make.
	bundled := make(map[string]bool, len(man.Files))
	for _, f := range man.Files {
		bundled[strings.TrimSuffix(f.Name, filepath.Ext(f.Name))] = true
	}

	played := make(map[string]bool)
	for screen, tn := range screenTunes {
		if tn.keep || tn.track == music.Score {
			continue
		}
		played[tn.track] = true
		if !bundled[tn.track] {
			t.Errorf("screen %v plays %q, which the bundle manifest does not carry", screen, tn.track)
		}
	}

	var spare []string
	for name := range bundled {
		if !played[name] {
			spare = append(spare, name)
		}
	}
	sort.Strings(spare)
	for _, name := range spare {
		t.Logf("%q is in the bundle and no screen plays it", name)
	}
}
