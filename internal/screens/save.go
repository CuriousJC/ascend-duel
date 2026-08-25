package screens

// **The three moments the game writes to disk, and nothing else knows they exist.**
//
// Persistence is deliberately not a thing a scene does. `internal/profile` owns the files,
// `internal/session` owns turning a run into a snapshot, and this file is the handful of call sites
// that connect them — so the whole feature is three functions to find rather than a `Save()`
// sprinkled through six screens.
//
// **Nothing here may fail a launch or interrupt play.** A failed write is logged once and the game
// carries on: the machine that cannot save is the machine that gets to keep playing, the same rule
// the audio device is under. See profile's doc.go.

import (
	"log"

	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// saveRun writes the run in progress.
//
// **Called at every phase transition and nowhere inside a duel** *(owner's call, 2026-08-25)*. The
// run is quiescent between stations — no piles dealt, no queued actions, no hidden hand — so a
// snapshot is the ten-odd fields `session.Snapshot` reads and not the combat screen's whole working
// state. The cost is stated rather than discovered: quitting mid-duel loses that duel and resumes at
// the start of the room, which is exactly what `Retry` already does after a defeat.
func saveRun(gs *state.GlobalState) {
	if gs == nil || gs.Run == nil {
		return
	}
	// **A machine with nowhere to write is silent, not noisy.** This fires at every phase
	// transition, so reporting "nowhere to save to" here would be a line per station for the whole
	// of a run on a machine that cannot help it. A store that *has* a directory and still fails is
	// a real problem and still says so.
	if gs.Store.Dir() == "" {
		return
	}
	if err := profile.SaveRun(gs.Store, gs.Run.Snapshot(gs.RunSeed)); err != nil {
		log.Printf("could not save the run: %v", err)
	}
}

// awardFirstSteps records the achievement for winning a duel.
//
// **It fires on every win and the profile keeps one**, which is what makes "defeat the first enemy"
// mean the first one the player ever beats rather than the first of a particular run: a player who
// loses room one fifty times gets it on the fifty-first.
func awardFirstSteps(gs *state.GlobalState) {
	award(gs, profile.AchievementFirstSteps)
}

// award records an achievement and saves the profile if anything changed.
//
// **Nothing is shown to the player.** There is no toast and no achievements screen yet — see
// TODO.md — so today the file is the whole of the record. The boolean `Award` reports is what a
// toast will hang off when one exists.
func award(gs *state.GlobalState, key string) {
	if gs == nil || gs.Profile == nil || !gs.Profile.Award(key) {
		return
	}
	saveProfile(gs)
}

// markTutorialSeen records that the teaching run is over.
//
// **Skipping counts as seeing it.** A player who dismissed Bob does not want him back next launch,
// and there is nowhere to put a control that would bring him back — the same argument that makes
// `tutorial.Run.Skip` irreversible for the session.
func markTutorialSeen(gs *state.GlobalState) {
	if gs == nil || gs.Profile == nil || gs.Profile.TutorialSeen {
		return
	}
	gs.Profile.TutorialSeen = true
	saveProfile(gs)
}

// saveProfile writes the profile, unless what was loaded may not be written over.
func saveProfile(gs *state.GlobalState) {
	if !gs.ProfileWritable {
		return
	}
	if err := profile.SaveProfile(gs.Store, gs.Profile); err != nil {
		log.Printf("could not save the profile: %v", err)
	}
}
