package screens

import (
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// TestEveryAchievementInTheCatalogueHasBothHalves is the one thing that can go wrong in a table
// nobody validates: a row with a key and no words, or words and no key. A blank row draws a blank
// card, which reads as a bug in the page rather than as a mistake in the list.
func TestEveryAchievementInTheCatalogueHasBothHalves(t *testing.T) {
	if len(achievements) == 0 {
		t.Fatal("the catalogue is empty, so the page has nothing to say")
	}
	seen := map[string]bool{}
	for i, a := range achievements {
		if a.key == "" {
			t.Errorf("achievement %d (%q) has no key, so nothing can ever award it", i, a.name)
		}
		if a.name == "" || a.line == "" {
			t.Errorf("achievement %d (%q) is missing a name or a line", i, a.key)
		}
		if seen[a.key] {
			t.Errorf("achievement %d repeats the key %q", i, a.key)
		}
		seen[a.key] = true
	}
}

// TestTheAchievementCatalogueNamesKeysTheProfileAwards is the seam between the two halves. The key
// is the contract on disk and the catalogue is presentation, so nothing links them but this: a
// catalogue naming a key nothing ever awards is a row that can never light up, and no test in
// `internal/profile` can see it.
func TestTheAchievementCatalogueNamesKeysTheProfileAwards(t *testing.T) {
	awarded := map[string]bool{
		profile.AchievementFirstSteps: true,
	}
	for _, a := range achievements {
		if !awarded[a.key] {
			t.Errorf("the page lists %q, which nothing in the game awards", a.key)
		}
	}
}

// TestTheTallyCountsWhatTheProfileHolds is the number under the heading, which is the whole reason
// the page gets opened.
func TestTheTallyCountsWhatTheProfileHolds(t *testing.T) {
	gs := saveState(t)

	if got := achievementTally(gs); got != "0 of 1" {
		t.Errorf("a fresh profile should read %q, got %q", "0 of 1", got)
	}

	gs.Profile.Award(profile.AchievementFirstSteps)
	if got := achievementTally(gs); got != "1 of 1" {
		t.Errorf("after the award it should read %q, got %q", "1 of 1", got)
	}
}

// TestAMissingProfileIsNothingEarned holds the rule the whole persistence layer is under: a machine
// that could not read its profile still gets to look at the page.
func TestAMissingProfileIsNothingEarned(t *testing.T) {
	gs := saveState(t)
	gs.Profile = nil

	for _, a := range achievements {
		if earned(gs, a) {
			t.Errorf("%q cannot be earned with no profile to hold it", a.key)
		}
	}
	if got := achievementTally(gs); got != "0 of 1" {
		t.Errorf("with no profile the tally should read %q, got %q", "0 of 1", got)
	}
}

// TestTheMenuScreensGoBackWhereTheyCameFrom is the one job a screen reachable from anywhere has.
// **Both of these can be opened from the title today and from anywhere tomorrow**, so Back reading
// ReturnScreen rather than naming the title is what stops that being a change to two files.
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
	} {
		gs := saveState(t)
		gs.ReturnScreen = state.Shop
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
		gs.ReturnScreen = tc.here
		tc.leave(gs)
		if gs.ActiveScreen != state.Title {
			t.Errorf("%s: with no return recorded, Back goes to the title, got %v",
				tc.name, gs.ActiveScreen)
		}
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

// TestTheCreditsNameBothCopyrightHolders is the correctness half of the page. The project is
// source-available and meant to be sold by two people; a credits screen that dropped one of them is
// the kind of mistake nobody notices until it has shipped.
func TestTheCreditsNameBothCopyrightHolders(t *testing.T) {
	for _, want := range []string{"Justin Crosby", "KingSherman1820", "PolyForm Noncommercial"} {
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
