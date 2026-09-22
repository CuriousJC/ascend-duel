package screens

// **Where a run begins and where it ends.**
//
// This is the run's lifecycle: the one the game boots with, the fresh one the title screen's
// **New Run** builds, the saved one **Continue** walks back into, and the abandonment the settings
// screen offers. All four used to be one function in `main`, which was fine while a run existed
// before the player had been asked anything — the game booted straight into a duel.
//
// **A title screen is what forced the move** *(owner's call, 2026-09-03)*. Once "start a new one"
// and "carry on with the old one" are two buttons, building a run is something a *screen* does, and
// `main` cannot be imported by one. So the logic came here, where `session`, `profile`, `tutorial`,
// `seeds` and `scenario` are all already in reach; `main` keeps only the seed it rolled and one
// call to BootRun.
//
// **Nothing here may fail a launch.** A snapshot that cannot be read is a new run and says so in
// the log, which is the rule the whole of `internal/profile` is under.

import (
	"log"
	"time"

	"github.com/curiousjc/ascend-duel/internal/journal"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/scenario"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
)

// BootRun is the run the game opens with: the one saved on disk if there is one, otherwise a fresh
// one from the seed main has already chosen.
//
// **It sets gs.Run and gs.Resumed and nothing else** — in particular it does not touch
// ActiveScreen, because the game now boots to the title and the player says which run they want.
// A resumed run is built and left standing until Continue is pressed.
//
// **A scenario never resumes.** A fixture describes a run it is putting together itself, and a
// saved tower would be the one thing it could not override.
func BootRun(gs *state.GlobalState) {
	if !scenario.Active() {
		if snap, ok, err := profile.LoadRun(gs.Store); err != nil {
			log.Printf("saved run: %v — starting a new one", err)
		} else if ok {
			run, seed, err := session.Resume(gs.Motifs, gs.Tower, snap)
			if err != nil {
				log.Printf("saved run: %v — starting a new one", err)
			} else {
				// **The saved run's seed wins outright.** Everything random in a run derives from
				// it, the climb included, so resuming under the seed rolled at the top of main
				// would put a different tower under the same deck.
				gs.RunSeed = seed
				gs.Resumed = true
				gs.Run = run
				resumeJournal(gs)
				log.Printf("resuming run %s at room %d, %s", snap.Seed, snap.Fight, snap.Phase)
				bootPastTheTitle(gs)
				return
			}
		}
	}
	gs.Resumed = false
	gs.Run = buildRun(gs)
	beginJournal(gs)
	bootPastTheTitle(gs)
}

// beginJournal opens a journal for a run that is starting, and resumeJournal for one already being
// climbed.
//
// **The seed has to be settled before either is called.** A taught run is dealt the script's own
// code and a resumed one brings its own, so a header written at the moment a seed was rolled would
// name a tower nobody is playing — the same ordering main's own log line is under.
//
// **They are two calls rather than one taking a flag** because they do opposite things to the file
// on disk: starting a run truncates it and resuming one appends to it. See internal/journal.
func beginJournal(gs *state.GlobalState) {
	gs.Journal.Begin(journalHeader(gs))
	gs.Journal.Write(journal.Record{
		Kind:   journal.KindRun,
		Action: journal.RunStarted,
		Key:    seeds.Code(gs.RunSeed),
	})
}

func resumeJournal(gs *state.GlobalState) {
	gs.Journal.Resume(journalHeader(gs))
	gs.Journal.Write(journal.Record{
		Kind:   journal.KindRun,
		Action: journal.RunResumed,
		Key:    seeds.Code(gs.RunSeed),
		Fight:  gs.Run.Fight(),
	})
}

// journalHeader is what opens a journal file: which build wrote it and which run it is about.
//
// **The install id is the only thing in it that is about whom**, and it is copied rather than
// looked up because a copy of this file travels beside a crash report — see profile.Profile.
func journalHeader(gs *state.GlobalState) journal.Header {
	h := journal.Header{Version: gs.Version, RunCode: seeds.Code(gs.RunSeed)}
	if gs.Profile != nil {
		h.Install = gs.Profile.InstallID
	}
	return h
}

// bootPastTheTitle walks straight into the run for the one build that cannot press a button to get
// there: the scripted demo, which drives the combat scene rather than navigating to it.
//
// **Called for a resumed run too**, above, because a demo build has no business stopping anywhere.
func bootPastTheTitle(gs *state.GlobalState) {
	if DemoPlaysItself() && gs.Run != nil {
		enterRun(gs)
	}
}

// buildRun makes a new run out of the seed currently on the state, teaching it if this player has
// never been taught.
//
// **A taught run is dealt the script's own seed, and that has to happen before the run is built**
// *(2026-08-25)*. Bob's lesson describes the hand the player is holding and promises a kill in one
// blow; both are facts about one deal against one creature, so the script carries the run code and
// session.Enemy reads the opponent off it. Teaching on whatever the clock rolled is exactly the
// bug this fixes — the lesson said five matching cards over a hand holding two.
func buildRun(gs *state.GlobalState) *session.Session {
	script, teaching := tutorialForThisRun(gs)
	if teaching && script.Seed != "" {
		seed, err := seeds.Parse(script.Seed)
		if err != nil {
			// A script naming a seed that is not a run code is a broken lesson, and the tutorial
			// is the one feature whose audience cannot tell a bug from the game. It fails the
			// launch, exactly as an unresolvable step does.
			log.Fatalf("tutorial.json: seed %q: %v", script.Seed, err)
		}
		gs.RunSeed = seed
	}

	run := session.Start(gs.Motifs, gs.Tower, gs.RunSeed)
	if teaching {
		run.Teach(script)
		log.Printf("teaching this run: %d steps, seed %s, first room %s",
			script.Len(), script.Seed, script.Enemy)
	}
	return run
}

// tutorialForThisRun is the script to teach and whether to teach it.
//
// **It answers no to everything, because the lesson has no fight to be taught in.** The script
// names a creature and a run code together — the taught hand, the blow it lands and the answering
// turn were one tuned set — and the roster it names no longer exists. A lesson that opened on a
// creature the script had not measured would describe a hand it had not dealt, which is worse than
// no lesson: the tutorial is the one feature whose audience cannot tell a bug from the game.
//
// Re-teaching it is a motif, an element and a fresh run code that satisfy all of it at once; see
// TODO.md, and the tutorial section of CLAUDE.md for the constraints a replacement has to meet.
func tutorialForThisRun(gs *state.GlobalState) (tutorial.Script, bool) {
	return tutorial.Script{}, false
}

// NewRun throws away whatever run was in progress and starts one from the beginning.
//
// **The seed is rerolled unless it was pinned** *(2026-09-03)*. A pinned seed is a debugging
// session where the same tower in the same order is the whole point — fixedRunSeed, or a
// scenario's own Seed — and a New Run that quietly rolled a different one would break the pin
// from a button. Everything else gets a new tower, which is what "new run" means.
//
// **The saved run is deleted before the new one is built**, so a game closed on the title screen
// straight afterwards does not come back to the climb the player just abandoned.
func NewRun(gs *state.GlobalState) {
	discardSavedRun(gs)

	if !gs.SeedPinned {
		gs.RunSeed = seeds.Normalize(time.Now().UnixNano())
	}
	gs.Resumed = false
	gs.Run = buildRun(gs)
	beginJournal(gs)
	log.Printf("new run %s", seeds.Code(gs.RunSeed))

	enterRun(gs)
}

// ContinueRun walks back into the run already standing, on whichever screen draws the station it
// was saved at.
func ContinueRun(gs *state.GlobalState) {
	if gs.Run == nil {
		return
	}
	enterRun(gs)
}

// AbandonRun gives the climb up from the settings screen, and EndRunInDefeat is what a death does.
//
// **They are the same event and they are one function** *(owner's call, 2026-09-03)*. There is no
// retry — a death killing the run is what makes a roguelike one — and the only difference between
// dying and walking away is which screen the player was standing on and which word the summary
// prints. Two paths from "this climb is finished" to "the file is gone" is one path that can be got
// wrong.
func AbandonRun(gs *state.GlobalState)     { endRun(gs, session.EndedByChoice) }
func EndRunInDefeat(gs *state.GlobalState) { endRun(gs, session.EndedInDefeat) }

// endRun closes a climb: it is added up, the file goes, the run goes, and the player lands on the
// splash that says what it came to.
//
// **The summary is taken before anything is destroyed**, which is the whole reason this is one
// function rather than a line in each caller. `gs.Run` is nil a moment later and the numbers are
// derived from the ledger it was carrying, so an order that cleared first would leave the screen
// with nothing to draw and nothing to say about it.
//
// **It leaves gs.Run nil rather than building a replacement**, which is the honest state to be in —
// nothing is climbing. The frame's ledger button already goes dead with no run to account for, so
// nothing has to learn a new case.
//
// **The seed is not rerolled here.** NewRun does that at the moment a tower is actually built, so
// the code the summary prints is the code that dealt the run being summarized.
func endRun(gs *state.GlobalState, ended string) {
	if gs.Run != nil {
		summary := gs.Run.Summarize(gs.RunSeed, ended)
		gs.Summary = &summary

		// **The journal is not truncated here, unlike the snapshot on disk.** A run that ended
		// badly is the one somebody is going to ask about, so the choices that got it there stay
		// until the next run starts and takes the file. See internal/journal.
		gs.Journal.Write(journal.Record{
			Kind:   journal.KindRun,
			Action: ended,
			Key:    seeds.Code(gs.RunSeed),
			Fight:  gs.Run.Fight(),
		})
	}

	discardSavedRun(gs)
	gs.Run = nil
	gs.Resumed = false

	// **The title screen is the fallback, for a run that was never there.** Nothing today calls
	// this without a run, but a splash drawn over no summary would be a blank page with a button on
	// it, which is worse than not going there.
	gs.ActiveScreen = state.RunOver
	if gs.Summary == nil {
		gs.ActiveScreen = state.Title
	}
	gs.NewScreen = true
}

// discardSavedRun removes the snapshot on disk, if there is one and if there is anywhere to remove
// it from.
//
// **A failure is logged and stepped over**, like every other write in the game: a machine that
// cannot delete the file still gets to start the new run, and the worst that follows is a stale
// Continue on the next launch.
func discardSavedRun(gs *state.GlobalState) {
	if gs.Store.Dir() == "" {
		return
	}
	if err := profile.DeleteRun(gs.Store); err != nil {
		log.Printf("could not clear the saved run: %v", err)
	}
}

// enterRun puts the game on whichever scene draws the run's current station.
//
// **The combat screen is the fallback**, for a phase with no scene of its own — the room choice is
// in the loop and has none. A run saved at one would otherwise land the player on a screen that is
// not there; advance walks past such a phase during play, and this is the same courtesy at the
// door.
func enterRun(gs *state.GlobalState) {
	next := state.Combat
	if s, ok := screenFor(gs.Run.Phase()); ok {
		next = s
	}
	gs.ActiveScreen = next
	gs.NewScreen = true
}
