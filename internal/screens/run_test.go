package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// TestAbandoningARunLeavesNothingToResume is the whole point of the control: giving up has to mean
// the next launch starts over, and the only thing that decides what the next launch does is whether
// a snapshot is on disk.
func TestAbandoningARunLeavesNothingToResume(t *testing.T) {
	gs := saveState(t)
	gs.Resumed = true

	// Put something on disk the way the game does, by moving the run to a new station.
	gs.Run.WonFight(40)
	advanceRun(gs)
	if _, ok, _ := profile.LoadRun(gs.Store); !ok {
		t.Fatal("the run should be on disk before it is abandoned")
	}

	AbandonRun(gs)

	if _, ok, _ := profile.LoadRun(gs.Store); ok {
		t.Error("abandoning a run must delete the snapshot")
	}
	if gs.Run != nil {
		t.Error("abandoning a run must leave no run standing")
	}
	if gs.Resumed {
		t.Error("there is nothing left to have resumed")
	}
	if gs.ActiveScreen != state.RunOver {
		t.Errorf("abandoning a run shows what it came to, not %v", gs.ActiveScreen)
	}
	if gs.Summary == nil {
		t.Fatal("abandoning a run must leave a summary to draw")
	}
	if gs.Summary.Ended != session.EndedByChoice {
		t.Errorf("the summary says %q, want %q", gs.Summary.Ended, session.EndedByChoice)
	}
}

// TestANewRunRollsANewTower is the seed rule: New Run means a new climb, so the seed has to move.
// **Unless it was pinned**, which the next test holds.
func TestANewRunRollsANewTower(t *testing.T) {
	gs := saveState(t)
	before := gs.RunSeed

	NewRun(gs)

	if gs.RunSeed == before {
		t.Error("a new run must not replay the tower the last one was on")
	}
	if gs.Run == nil {
		t.Fatal("a new run must leave a run standing")
	}
	if gs.Resumed {
		t.Error("a run built here was not resumed off the disk")
	}
}

// TestAPinnedSeedSurvivesANewRun is why state.SeedPinned exists. A pin is a debugging session where
// the same tower in the same order is the whole point, and a menu button must not undo from the
// title screen what was set in main.
func TestAPinnedSeedSurvivesANewRun(t *testing.T) {
	gs := saveState(t)
	gs.SeedPinned = true

	// **Taught, and the pin loses.** The tutorial's script carries its own run code because the
	// lesson promises the player the hand they are holding, and that outranks every other way a
	// seed can be chosen — see buildRun. This test is about the other case, so the player is one
	// who has already been taught.
	gs.Profile.TutorialSeen = true

	before := gs.RunSeed

	NewRun(gs)

	if gs.RunSeed != before {
		t.Errorf("a pinned seed must survive New Run: %d became %d", before, gs.RunSeed)
	}
}

// TestANewRunClearsTheOldSaveBeforeItStarts is the ordering that matters: a player who presses New
// Run and then closes the window on the title screen must not come back to the climb they just gave
// up on.
func TestANewRunClearsTheOldSaveBeforeItStarts(t *testing.T) {
	gs := saveState(t)
	gs.Run.WonFight(40)
	advanceRun(gs)
	if _, ok, _ := profile.LoadRun(gs.Store); !ok {
		t.Fatal("the run should be on disk before New Run is pressed")
	}

	NewRun(gs)

	if _, ok, _ := profile.LoadRun(gs.Store); ok {
		t.Error("New Run must clear the snapshot it is replacing")
	}
}

// TestARunIsEnteredOnTheScreenThatDrawsItsStation is what Continue promises: a run saved at the
// shop comes back to the shop, not to a fresh duel.
func TestARunIsEnteredOnTheScreenThatDrawsItsStation(t *testing.T) {
	for _, tc := range []struct {
		phase session.Phase
		want  state.ActiveScreen
	}{
		{session.PhaseFight, state.Combat},
		{session.PhaseReward, state.PostBattle},
		{session.PhaseShop, state.Shop},
	} {
		gs := saveState(t)
		gs.Run.SetPhase(tc.phase)
		gs.ActiveScreen = state.Title

		ContinueRun(gs)

		if gs.ActiveScreen != tc.want {
			t.Errorf("%v should be drawn by %v, got %v", tc.phase, tc.want, gs.ActiveScreen)
		}
		if !gs.NewScreen {
			t.Errorf("%v: entering a run must run the incoming scene's Init", tc.phase)
		}
	}
}

// TestContinueIsDeadWithNothingToContinue holds the title screen's one piece of logic. A menu entry
// that works and does nothing is worse than one that says it has nothing to do.
func TestContinueIsDeadWithNothingToContinue(t *testing.T) {
	gs := saveState(t)
	scene := &TitleScene{}
	scene.Init(gs)

	// A fresh run standing is not a run to *continue* — it was built at boot, not resumed.
	gs.Resumed = false
	if err := scene.Update(gs); err != nil {
		t.Fatal(err)
	}
	if scene.continueButton.State != models.ButtonStateDisabled {
		t.Error("Continue must be dead when nothing was resumed off the disk")
	}

	gs.Resumed = true
	if err := scene.Update(gs); err != nil {
		t.Fatal(err)
	}
	if scene.continueButton.State == models.ButtonStateDisabled {
		t.Error("Continue must be live when a run was resumed off the disk")
	}
}

// TestNewRunOnlyAsksWhenThereIsSomethingToLose is the courtesy half of the confirm. A player on a
// clean install pressing New Run gets a run, not a question about a climb they have never taken.
func TestNewRunOnlyAsksWhenThereIsSomethingToLose(t *testing.T) {
	gs := saveState(t)
	scene := &TitleScene{}
	scene.Init(gs)

	gs.Resumed = false
	scene.startNewRun(gs)
	if scene.confirm.isOpen() {
		t.Error("a run nobody has entered is not worth a dialog")
	}
	if gs.ActiveScreen != state.Combat {
		t.Errorf("New Run with nothing to lose should start the run, got %v", gs.ActiveScreen)
	}

	scene.Init(gs)
	gs.ActiveScreen = state.Title
	gs.Resumed = true
	scene.startNewRun(gs)
	if !scene.confirm.isOpen() {
		t.Error("New Run over a resumed climb must ask first")
	}
	if gs.ActiveScreen != state.Title {
		t.Error("asking the question must not also answer it")
	}
}

// TestTheConfirmAnswersTheQuestionItWasLastAsked is why the callbacks are rebound every frame. One
// dialog serves two callers, and a Cancel that ran the previous caller's answer would abandon a run
// the player was declining to abandon.
func TestTheConfirmAnswersTheQuestionItWasLastAsked(t *testing.T) {
	gs := saveState(t)

	first, second := 0, 0
	var d confirmDialog

	d.ask("one", "", "One", func() { first++ })
	d.update(gs)
	d.close()

	d.ask("two", "", "Two", func() { second++ })
	d.update(gs)
	d.yes.OnClick()

	if first != 0 || second != 1 {
		t.Errorf("the dialog answered the wrong question: first=%d second=%d", first, second)
	}
	if d.isOpen() {
		t.Error("answering a question closes it")
	}
}

// TestAQuestionCancelsWithoutAnswering is the other half: Cancel is the safe answer and it must not
// touch the run.
func TestAQuestionCancelsWithoutAnswering(t *testing.T) {
	gs := saveState(t)

	fired := 0
	var d confirmDialog
	d.ask("gone?", "", "Yes", func() { fired++ })
	d.update(gs)
	d.no.OnClick()

	if fired != 0 {
		t.Error("Cancel must not run the destructive answer")
	}
	if d.isOpen() {
		t.Error("Cancel closes the question")
	}
}

// TestAbandonIsDeadWithNoRunToGiveUp keeps the settings screen honest on the one screen it can be
// opened from with no climb in progress: the title.
func TestAbandonIsDeadWithNoRunToGiveUp(t *testing.T) {
	gs := saveState(t)
	scene := &SettingsScene{}
	scene.Init(gs)

	gs.Run = nil
	if err := scene.Update(gs); err != nil {
		t.Fatal(err)
	}
	if scene.abandon.State != models.ButtonStateDisabled {
		t.Error("Abandon Run must be dead with no run to abandon")
	}

	// And it refuses to raise the question even if something reaches the callback anyway.
	scene.askAbandon(gs)
	if scene.confirm.isOpen() {
		t.Error("there is nothing to ask about")
	}
}

// TestAbandonAsksBeforeItDestroys is the guard on the only irreversible control in the game.
func TestAbandonAsksBeforeItDestroys(t *testing.T) {
	gs := saveState(t)
	scene := &SettingsScene{}
	scene.Init(gs)

	scene.askAbandon(gs)

	if !scene.confirm.isOpen() {
		t.Fatal("Abandon Run must ask first")
	}
	if gs.Run == nil {
		t.Error("asking the question must not also answer it")
	}
}

// TestADeathEndsTheRun is the roguelike rule, and it is the one thing about a defeat that must not
// be quietly softened later: there is no retry, so a loss has to leave nothing to come back to.
func TestADeathEndsTheRun(t *testing.T) {
	gs := saveState(t)
	gs.Run.WonFight(40)
	advanceRun(gs)
	if _, ok, _ := profile.LoadRun(gs.Store); !ok {
		t.Fatal("the run should be on disk before the duelist falls")
	}

	EndRunInDefeat(gs)

	if _, ok, _ := profile.LoadRun(gs.Store); ok {
		t.Error("a death must leave nothing to resume")
	}
	if gs.Run != nil {
		t.Error("a death must leave no run standing")
	}
	if gs.ActiveScreen != state.RunOver {
		t.Errorf("a death shows what the run came to, not %v", gs.ActiveScreen)
	}
	if gs.Summary == nil {
		t.Fatal("a death must leave a summary to draw")
	}
	if gs.Summary.Ended != session.EndedInDefeat {
		t.Errorf("the summary says %q, want %q", gs.Summary.Ended, session.EndedInDefeat)
	}
}

// TestTheSummaryIsTakenBeforeTheRunIsDestroyed is the ordering the whole splash depends on. The
// numbers come off the ledger the run was carrying, so a clear-then-summarise would leave the page
// with nothing — and nothing about drawing an empty page fails.
func TestTheSummaryIsTakenBeforeTheRunIsDestroyed(t *testing.T) {
	gs := saveState(t)

	gs.Run.BeginFight(1, "GiantBat")
	gs.Run.RecordRound([]session.LedgerLine{session.Line(session.VoicePlain, "hit")}, 40)
	gs.Run.EndFight(session.OutcomeWon)
	gs.Run.BeginFight(1, "Cave Troll")
	gs.Run.RecordRound([]session.LedgerLine{session.Line(session.VoicePlain, "hit")}, 12)
	gs.Run.EndFight(session.OutcomeLost)

	EndRunInDefeat(gs)

	if gs.Summary == nil {
		t.Fatal("no summary was taken")
	}
	if gs.Summary.Defeated != 1 {
		t.Errorf("one enemy fell, the summary says %d", gs.Summary.Defeated)
	}
	if gs.Summary.Dealt != 52 {
		t.Errorf("the run dealt 52, the summary says %d", gs.Summary.Dealt)
	}
	if gs.Summary.Rooms != 2 {
		t.Errorf("two rooms were entered, the summary says %d", gs.Summary.Rooms)
	}
}

// TestTheSummaryCarriesTheSeedThatDealtTheRun is the field the page exists for. **It must be the
// run's own code**, not whatever the seed happens to be by the time the splash is drawn.
func TestTheSummaryCarriesTheSeedThatDealtTheRun(t *testing.T) {
	gs := saveState(t)
	want := seeds.Code(gs.RunSeed)

	EndRunInDefeat(gs)

	if gs.Summary == nil {
		t.Fatal("no summary was taken")
	}
	if gs.Summary.Seed != want {
		t.Errorf("the summary names run %q, want %q", gs.Summary.Seed, want)
	}
}

// TestLeavingTheSplashDropsTheSummary stops a finished run's numbers being reachable later. A stale
// summary on the state is a page that can be opened again showing two runs ago.
func TestLeavingTheSplashDropsTheSummary(t *testing.T) {
	gs := saveState(t)
	EndRunInDefeat(gs)

	scene := &RunOverScene{}
	scene.Init(gs)
	scene.leave(gs)

	if gs.Summary != nil {
		t.Error("leaving the splash must drop the summary it was drawing")
	}
	if gs.ActiveScreen != state.Title {
		t.Errorf("the splash goes to the title, not %v", gs.ActiveScreen)
	}
}

// TestTheSplashLeavesByItselfWithNothingToSay is the guard on the one state it must not sit in. A
// blank page with a button on it is worse than not going there.
func TestTheSplashLeavesByItselfWithNothingToSay(t *testing.T) {
	gs := saveState(t)
	gs.Summary = nil
	gs.ActiveScreen = state.RunOver

	scene := &RunOverScene{}
	scene.Init(gs)
	if err := scene.Update(gs); err != nil {
		t.Fatal(err)
	}
	if gs.ActiveScreen != state.Title {
		t.Errorf("a splash with no summary should leave, got %v", gs.ActiveScreen)
	}
}

// TestEveryEndingHasWordsForIt keeps the two constants and the page's phrase table in step. An
// ending with no entry still draws its numbers, but it loses the sentence — which is a silent
// downgrade rather than a failure, so this is what notices.
func TestEveryEndingHasWordsForIt(t *testing.T) {
	for _, ending := range []string{session.EndedInDefeat, session.EndedByChoice} {
		w, ok := runOverWords[ending]
		if !ok {
			t.Errorf("no words for the ending %q", ending)
			continue
		}
		if w.title == "" || w.how == "" {
			t.Errorf("the ending %q is missing a title or a line", ending)
		}
	}
}

// TestEveryMenuRowIsOnScreen is the layout guard the title screen grew when it went from three rows
// to six. **A button drawn off the bottom of the screen is still clickable in principle and
// invisible in practice**, and nothing about drawing one fails.
func TestEveryMenuRowIsOnScreen(t *testing.T) {
	gs := saveState(t)
	gs.ScreenWidth, gs.ScreenHeight = 1280, 960

	scene := &TitleScene{}
	scene.Init(gs)

	for i, b := range scene.menu() {
		top := b.ScreenY - b.Height/2
		bottom := b.ScreenY + b.Height/2
		if top < 0 || bottom > gs.ScreenHeight {
			t.Errorf("menu row %d (%q) spans %d..%d, off a %d-tall screen",
				i, b.Text, top, bottom, gs.ScreenHeight)
		}
	}
}

// TestTheMenuRowsDoNotOverlap holds the other half of the same layout: the gap has to be at least
// the button height, or two rows share pixels and the upper one eats the lower one's clicks.
func TestTheMenuRowsDoNotOverlap(t *testing.T) {
	gs := saveState(t)
	gs.ScreenWidth, gs.ScreenHeight = 1280, 960

	scene := &TitleScene{}
	scene.Init(gs)

	rows := scene.menu()
	for i := 1; i < len(rows); i++ {
		prevBottom := rows[i-1].ScreenY + rows[i-1].Height/2
		top := rows[i].ScreenY - rows[i].Height/2
		if top < prevBottom {
			t.Errorf("menu rows %d and %d overlap: %d then %d", i-1, i, prevBottom, top)
		}
	}
}

// TestEveryMenuRowIsBuilt catches the failure the menu() list exists to prevent: a field added to
// the struct and left nil is a button that is drawn as a nil dereference rather than not drawn.
func TestEveryMenuRowIsBuilt(t *testing.T) {
	gs := saveState(t)
	scene := &TitleScene{}
	scene.Init(gs)

	for i, b := range scene.menu() {
		if b == nil {
			t.Fatalf("menu row %d was never built", i)
		}
		if b.Text == "" {
			t.Errorf("menu row %d has no label", i)
		}
	}
}
