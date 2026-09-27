package screens

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/seeds"
)

// TestAChosenSeedProgressesNothing is the rule the new-run dialog's checkbox reports: a run started
// on a code the player chose is a journey that could have been looked up, so nothing it does moves
// the profile — no tally, no award, no unlock, no toast. The same code rolled progresses as normal.
func TestAChosenSeedProgressesNothing(t *testing.T) {
	gs := saveState(t)
	seed, err := seeds.Parse("M9079R")
	if err != nil {
		t.Fatal(err)
	}
	turn := []combat.Card{
		{Concept: combat.Bash, Element: combat.Fire},
		{Concept: combat.Jab, Element: combat.Ice},
	}
	play := func() {
		bumpCounters(gs, turn)
		settleCounters(gs)
		earn(gs, []string{profile.AchievementFirstSteps})
	}

	NewRunOn(gs, seed, true)
	before := *gs.Profile
	before.Counters = map[string]int{}
	for k, v := range gs.Profile.Counters {
		before.Counters[k] = v
	}
	play()
	if !reflect.DeepEqual(gs.Profile.Counters, before.Counters) && len(gs.Profile.Counters)+len(before.Counters) > 0 {
		t.Errorf("a chosen-seed run moved the tallies: %v, was %v", gs.Profile.Counters, before.Counters)
	}
	if len(gs.Profile.Achievements) != len(before.Achievements) || len(gs.EarnedThisSession) != 0 {
		t.Error("a chosen-seed run earned an achievement")
	}
	if len(gs.Profile.Unlocks) != len(before.Unlocks) {
		t.Error("a chosen-seed run unlocked something")
	}

	NewRunOn(gs, seed, false)
	play()
	if !earned(gs, profile.AchievementFirstSteps) {
		t.Error("a rolled run on the same code must still earn it")
	}
	moved := false
	for _, n := range gs.Profile.Counters {
		moved = moved || n > 0
	}
	if !moved {
		t.Error("a rolled run's turn must still move the tallies")
	}
}

// profileWrites are the calls that move a player's progress. A new kind of progress is a new method
// on profile.Profile, and it belongs on this list the day it is written.
var profileWrites = regexp.MustCompile(`\.Profile\.(Bump|Award|Unlock|Discover)\(|\.Profile\.(Counters|Achievements|Unlocks|HandsDiscovered)(\[[^\]]*\])?\s*(=[^=]|\+=|-=|\+\+|--)`)

// TestOnlyAchieveGoWritesProgress is what makes the chosen-seed rule hold for progress nobody has
// written yet. achieve.go asks progresses before every write, so a write anywhere else in the
// program is a write that can leak past a chosen seed — the fix is to route it through achieve.go,
// never to add a file here. See the achievements skill.
func TestOnlyAchieveGoWritesProgress(t *testing.T) {
	root := filepath.Join("..", "..")
	var offenders []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", ".scratch", "artreview", "docs", "assets", "privateassets", "profile":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if filepath.Base(path) == "achieve.go" && filepath.Base(filepath.Dir(path)) == "screens" {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if loc := profileWrites.FindIndex(src); loc != nil {
			offenders = append(offenders, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range offenders {
		t.Errorf("%s writes player progress outside internal/screens/achieve.go, past the chosen-seed gate", f)
	}
}
