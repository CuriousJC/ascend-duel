package game

// **Where the player is, written into the journal once a frame.**
//
// Every other choice in the game is a click with a function behind it, and that function is where
// its journal line goes. These two are not: a screen change and a phase change happen from a dozen
// places — a button, a run advancing, a crash, a scenario opening the game halfway up a tower — and
// a call beside each is a list the next one gets left off.
//
// **So they are diffed rather than announced**, which is screens.RunWatch's rule and
// combat_handmorph.go's one screen over: what changed is read off what is true now, so nothing
// anywhere needs a case, and a new way to reach a screen cannot arrive without a line.

import (
	"github.com/curiousjc/ascend-duel/internal/journal"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// journalWatch remembers where the player was a frame ago.
type journalWatch struct {
	// started says the first frame has been through, so the opening screen and the opening phase
	// are written rather than skipped as "no change from the zero value".
	started bool

	screen state.ActiveScreen
	phase  session.Phase

	// hadRun is whether there was a run last frame, so the phase of a run that has just ended is
	// not diffed against the phase of the one that replaces it.
	hadRun bool
	fight  int
}

// note writes a line for anything that moved since the last frame.
//
// **Guarded, like every other reader that touches a run from the frame.** This runs after the
// scene, which is the code most likely to have left something in a state its own accessors do not
// survive; a journal that panicked would turn a bad frame into a crash.
func (w *journalWatch) note(gs *state.GlobalState) {
	defer func() { _ = recover() }()
	if gs == nil {
		return
	}

	phase, fight, hasRun := session.Phase(0), 0, gs.Run != nil
	if hasRun {
		phase, fight = gs.Run.Phase(), gs.Run.Fight()
	}

	first := !w.started
	w.started = true

	if first || gs.ActiveScreen != w.screen {
		gs.Journal.Write(journal.Record{
			Kind:   journal.KindScreen,
			Screen: gs.ActiveScreen.String(),
		})
	}

	// **A run that has just begun writes its phase and a run that has just ended writes nothing.**
	// The station a fresh run opens at is worth a line; the absence of one is already said by the
	// run record endRun wrote.
	if hasRun && (!w.hadRun || phase != w.phase || fight != w.fight) {
		gs.Journal.Write(journal.Record{
			Kind:  journal.KindPhase,
			Phase: phase.String(),
			Fight: fight,
		})
	}

	w.screen, w.phase, w.fight, w.hadRun = gs.ActiveScreen, phase, fight, hasRun
}
