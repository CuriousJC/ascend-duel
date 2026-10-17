package screens

import (
	"strconv"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/achieve"
	"github.com/curiousjc/ascend-duel/internal/actions"

	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// TestEveryAchievementInTheCatalogHasBothHalves is the one thing that can go wrong in a
// catalog nobody looks at: a row with a key and no words. A blank row draws a blank card, which
// reads as a bug in the page rather than as a mistake in the list.
//
// **The shape checks moved to internal/achieve on 2026-09-06**, where the file is parsed and a bad
// record fails the launch. What is left here is the half that is this screen's: that every record
// carries the two strings this page draws.
func TestEveryAchievementInTheCatalogHasBothHalves(t *testing.T) {
	all := achieve.Loaded().All()
	if len(all) == 0 {
		t.Fatal("the catalog is empty, so the page has nothing to say")
	}
	for _, a := range all {
		if a.DisplayName == "" || a.Description == "" {
			t.Errorf("achievement %q is missing a DisplayName or a Description", a.APIName)
		}
	}
}

// TestTheOldFirstStepsKeyIsStillInTheCatalog is the one seam a data move can quietly break. The
// key is the contract on disk — every profile already written holds `first-steps` — and the
// catalog that awards it is now a JSON file, so nothing but this says the two still agree.
//
// **profile.AchievementFirstSteps is the Go side of that contract**, and it is kept for exactly
// this: a constant naming the one key that shipped before the catalog existed.
func TestTheOldFirstStepsKeyIsStillInTheCatalog(t *testing.T) {
	if _, ok := achieve.Loaded().Find(profile.AchievementFirstSteps); !ok {
		t.Fatalf("every profile on disk holds %q; the catalog no longer has it",
			profile.AchievementFirstSteps)
	}
}

// TestTheTallyCountsWhatTheProfileHolds is the number under the heading, which is the whole reason
// the page gets opened.
func TestTheTallyCountsWhatTheProfileHolds(t *testing.T) {
	gs := saveState(t)
	total := strconv.Itoa(len(achieve.Loaded().All()))

	if got, want := achievementTally(gs), "0 of "+total; got != want {
		t.Errorf("a fresh profile should read %q, got %q", want, got)
	}

	gs.Profile.Award(profile.AchievementFirstSteps)
	if got, want := achievementTally(gs), "1 of "+total; got != want {
		t.Errorf("after the award it should read %q, got %q", want, got)
	}
}

// TestAMissingProfileIsNothingEarned holds the rule the whole persistence layer is under: a machine
// that could not read its profile still gets to look at the page.
func TestAMissingProfileIsNothingEarned(t *testing.T) {
	gs := saveState(t)
	gs.Profile = nil

	for _, a := range achieve.Loaded().All() {
		if earned(gs, a.APIName) {
			t.Errorf("%q cannot be earned with no profile to hold it", a.APIName)
		}
	}
	want := "0 of " + strconv.Itoa(len(achieve.Loaded().All()))
	if got := achievementTally(gs); got != want {
		t.Errorf("with no profile the tally should read %q, got %q", want, got)
	}
}

// TestTheMenuScreensGoBackWhereTheyCameFrom is the one job a screen reachable from anywhere has.
// Back pops gs.Returns rather than naming the title, so a screen opened from the shop goes back to
// the shop.
func TestTheMenuScreensGoBackWhereTheyCameFrom(t *testing.T) {
	for _, tc := range []struct {
		name  string
		here  state.ActiveScreen
		leave func(*state.GlobalState)
	}{
		{"achievements", state.Achievements, func(gs *state.GlobalState) {
			(&AchievementsScene{}).leave(gs)
		}},
		{"credits", state.Credits, func(gs *state.GlobalState) {
			(&CreditsScene{}).leave(gs)
		}},
		{"compendium", state.Compendium, func(gs *state.GlobalState) {
			(&CompendiumScene{}).leave(gs)
		}},
	} {
		gs := saveState(t)
		gs.ActiveScreen = state.Shop
		gs.PushReturn()
		gs.ActiveScreen = tc.here

		tc.leave(gs)

		if gs.ActiveScreen != state.Shop {
			t.Errorf("%s: Back should return to the shop, got %v", tc.name, gs.ActiveScreen)
		}
		if !gs.NewScreen {
			t.Errorf("%s: Back must run the incoming scene's Init", tc.name)
		}

		// And with nothing recorded, the zero value already means the title.
		gs = saveState(t)
		gs.ActiveScreen = tc.here
		tc.leave(gs)
		if gs.ActiveScreen != state.Title {
			t.Errorf("%s: with no return recorded, Back goes to the title, got %v",
				tc.name, gs.ActiveScreen)
		}
	}
}

// TestBackWalksANestedPathInReverse is why Returns is a stack: a fight, then Settings, then the
// Compendium, then Achievements, and every Back has to land one step nearer the fight. One slot held
// only the last step, and the fourth Back went to the title instead of the duel.
func TestBackWalksANestedPathInReverse(t *testing.T) {
	gs := saveState(t)
	gs.ActiveScreen = state.Combat
	gs.PushReturn()
	gs.ActiveScreen = state.Settings
	actions.OpenCompendium(gs)
	actions.OpenAchievements(gs)

	(&AchievementsScene{}).leave(gs)
	if gs.ActiveScreen != state.Compendium {
		t.Fatalf("Back from Achievements should reach the Compendium, got %v", gs.ActiveScreen)
	}
	(&CompendiumScene{}).leave(gs)
	if gs.ActiveScreen != state.Settings {
		t.Fatalf("Back from the Compendium should reach Settings, got %v", gs.ActiveScreen)
	}
	(&SettingsScene{}).leave(gs)
	if gs.ActiveScreen != state.Combat {
		t.Fatalf("Back from Settings should reach the fight, got %v", gs.ActiveScreen)
	}
	if len(gs.Returns) != 0 {
		t.Errorf("every step should have been popped, %d left", len(gs.Returns))
	}
}

// TestNoCreditsLineIsBlankByAccident is why creditsGap is a kind rather than an empty string. A
// blank body line and a deliberate gap look identical in the source and lay out identically on the
// page, so the only way to tell them apart is to refuse the first.
func TestNoCreditsLineIsBlankByAccident(t *testing.T) {
	for i, l := range credits {
		if l.kind == creditsGap {
			continue
		}
		if l.text == "" {
			t.Errorf("credits line %d is blank but is not a gap", i)
		}
	}
}

// TestTheCreditsNameTheMakerAndTheLicense is the correctness half of the page: who made the game —
// the studio and its two people *(owner's call)* — and the terms the source is available under.
// Any one dropped is the kind of mistake nobody notices until it has shipped.
func TestTheCreditsNameTheMakerAndTheLicense(t *testing.T) {
	for _, want := range []string{"Ambivalent Systems", "CuriousJC", "KingSherman1820", "PolyForm Noncommercial"} {
		if !creditsMention(want) {
			t.Errorf("the credits do not mention %q", want)
		}
	}
}

// creditsMention reports whether any line of the page contains a string.
func creditsMention(want string) bool {
	for _, l := range credits {
		if strings.Contains(l.text, want) {
			return true
		}
	}
	return false
}
