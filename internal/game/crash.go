package game

// **Where a panic is caught, and what happens to the frame after one.**
//
// Ebitengine calls Update and Draw; nothing sits between them and the runtime, so a panic anywhere
// in a scene takes the process down with a stack trace into a console a player does not have. The
// two recovers here are the whole of the difference between that and a bug report — see
// internal/crashlog for what gets written.
//
// **A crash is not a dialog.** Every other box in the game is drawn over the scene underneath it,
// and the scene underneath this one is the one that has just panicked: drawing it again is how one
// crash becomes two. So the frame changes screens, the crashed scene is never asked for again, and
// the chrome stands down — see chromeShowing.
//
// **A second panic quits.** If the crash screen itself goes wrong there is nothing left to try, and
// a loop of reports written about the screen that reports them is worse than an exit. The second
// one is noted and the game closes through ShouldClose, the same door the window's close button
// uses.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/crashlog"
	"github.com/curiousjc/ascend-duel/internal/journal"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// recoverFrame is the deferred handler both Update and Draw install.
//
// **One handler rather than two**, because what a panic means does not depend on which of the two
// halves of a frame it happened in: the report is the same, the screen is the same, and the only
// thing Update does that Draw does not is hand an error back to the loop, which it does by leaving
// err alone.
func (g *Game) recoverFrame() { g.recovered(recover(), nil) }

// recoverDraw is the handler Draw installs, and it is the one with a picture in it.
//
// **A panic raised while the screen was being drawn can be photographed, and one raised in Update
// cannot** *(2026-09-23)*. The image Draw is handed holds exactly as much of the frame as got drawn
// before the fault, which is the most diagnostic picture there is — it shows how far the drawing
// got. A panic in Update happens between two frames, with the screen already cleared, so there is
// nothing to read.
//
// **The alternative was a rolling last-good frame and it was declined** *(owner's call,
// 2026-09-23)*. Keeping one means a ReadPixels into a persistent 8.29 MB buffer on a throttle —
// internal/trace's one every two seconds, since a readback is a GPU-to-CPU transfer that stalls the
// frame it happens on. What that buys is a picture of the game up to two seconds *before* anything
// went wrong, which is a picture of it working. The report already carries the screen, the phase,
// the tick, the run and the ledger, all of them current.
func (g *Game) recoverDraw(screen *ebiten.Image) { g.recovered(recover(), screen) }

// recovered is what both handlers do with a cause, or with nothing.
//
// **One body rather than two**, because what a panic means does not depend on which of the two
// halves of a frame it happened in: the report is the same, the screen is the same, and the only
// thing Update does that Draw does not is hand an error back to the loop, which it does by leaving
// err alone.
func (g *Game) recovered(cause any, screen *ebiten.Image) {
	if cause == nil {
		return
	}
	stack := crashlog.Stack()

	if g.crashed {
		// **Noted rather than reported.** A second report would be about the screen written to
		// report the first, and the file already on disk is the one worth reading.
		crashlog.Note("the crash screen itself panicked: %v", cause)
		g.GlobalState.ShouldClose = true
		return
	}
	g.crashed = true
	g.crash(cause, stack, shotOf(screen))
}

// shotOf reads the frame back off the GPU, or hands back nothing when there is no frame to read.
//
// **The readback has to happen here**, on the game goroutine inside the Draw call that panicked,
// because it is a GPU operation; the encoding and the ceiling are crashlog's, which is a package
// with tests. Same split internal/trace already makes, and the same one systems.ArtMark makes for
// a different reason.
//
// **It is guarded like every other reader in this file.** A graphics context in a state bad enough
// to panic a draw is one that can panic a readback, and a report without a picture is worth a great
// deal more than no report.
func shotOf(screen *ebiten.Image) (shot []byte) {
	if screen == nil {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			shot = nil
		}
	}()
	b := screen.Bounds()
	if b.Empty() {
		return nil
	}
	img := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	screen.ReadPixels(img.Pix)
	return crashlog.EncodeShot(img)
}

// crash writes the report and puts the game on the screen that says so.
func (g *Game) crash(cause any, stack []byte, shot []byte) {
	gs := g.GlobalState

	report := crashlog.Build(crashlog.State{
		Version:   gs.Version,
		Screen:    gs.ActiveScreen.String(),
		Phase:     runPhase(gs),
		Tick:      gs.Count,
		RunSeed:   gs.RunSeed,
		InstallID: installID(gs),
		Run:       runSnapshot(gs),
		Ledger:    runLedger(gs),
		Scene:     sceneState(g.scene()),
	}, cause, stack)

	// **The journal goes with it.** There is one journal and the next run truncates it, so the run
	// that blew up — exactly the run worth retracing — would otherwise be overwritten by the next
	// launch. It lands under the report's own name with its own extension, so the two sort
	// together and are pruned together. See internal/journal and crashlog.Write.
	path, err := crashlog.Write(gs.Store, report, shot, journal.FileName)
	if err != nil {
		crashlog.Note("could not write the crash report: %v", err)
	}

	gs.Crash = &state.CrashInfo{
		Panic: report.Panic,
		Code:  report.RunCode,
		Path:  path,
	}

	// **Everything that could take the frame is stood down first.** The toast, the ledger and any
	// scene's dialog all draw over whatever is underneath them, and what is underneath is the thing
	// that just went wrong. The crash screen has to be the only thing on the screen.
	if g.ledger.IsOpen() {
		g.ledger.Toggle()
	}
	gs.EarnedThisSession = nil
	gs.ModalOpen = false
	gs.InputGated = false
	gs.InputFocus = nil

	gs.ActiveScreen = state.Crashed
	gs.NewScreen = true
}

// The four readers below all touch a run that has just panicked, so each one is guarded: a run in a
// state bad enough to crash the game is a run whose Snapshot may crash the crash handler. **A tier
// that cannot be read is left out of the report**, which is exactly what the plan's shedding rule
// says a report does with a tier it cannot afford — here the currency is trust rather than bytes.

// runPhase is the station the run was standing at, as a word.
func runPhase(gs *state.GlobalState) (phase string) {
	defer func() { _ = recover() }()
	if gs.Run == nil {
		return ""
	}
	return gs.Run.Phase().String()
}

// runSnapshot is the run as it would have been saved.
func runSnapshot(gs *state.GlobalState) (snap *profile.RunSnapshot) {
	defer func() {
		if r := recover(); r != nil {
			snap = nil
		}
	}()
	if gs.Run == nil {
		return nil
	}
	return gs.Run.Snapshot(gs.RunSeed)
}

// runLedger is the run's account of itself so far.
//
// **This is the tier the snapshot cannot give.** A run is written to disk only at phase boundaries,
// so a crash mid-duel has the room's start state on disk and nothing since; these records are the
// only thing that knows what happened in the fight being played.
func runLedger(gs *state.GlobalState) (fights []session.LedgerFight) {
	defer func() {
		if r := recover(); r != nil {
			fights = nil
		}
	}()
	if gs.Run == nil {
		return nil
	}
	return gs.Run.LedgerFights()
}

// sceneState is whatever the screen that panicked has to say about itself, or nil.
//
// **Optional, so a new screen is not broken by not having one.** A scene that does not implement
// ui.Reporter contributes nothing and the tier is simply absent — which is the same answer a run
// that cannot be read gives above.
//
// **Guarded for the sharper version of this file's reason.** Every other reader here touches a run
// that has just panicked; this one calls a method on the exact object that did. See ui.Reporter for
// the rule that keeps the implementation to plain field reads, which is what makes the recover a
// backstop rather than the plan.
func sceneState(scene ui.Scene) (m map[string]any) {
	defer func() {
		if r := recover(); r != nil {
			m = nil
		}
	}()
	r, ok := scene.(ui.Reporter)
	if !ok {
		return nil
	}
	return r.Report()
}

// installID is what groups several reports from one player, or "" on a machine with no profile.
func installID(gs *state.GlobalState) string {
	if gs.Profile == nil {
		return ""
	}
	return gs.Profile.InstallID
}

// noticeAllowed reports whether a non-fatal notice may take the frame this tick.
//
// **Never during a duel** *(2026-09-22)*. A box in front of a round in playback stops a fight to
// talk about a file, and what it has to say — a run that is not being saved — is not something the
// player can act on until the fight is over. So it waits for a phase boundary: the run standing
// anywhere but the fight, or no run at all.
//
// **And never on top of a crash**, which is the one screen that may not be drawn over.
func noticeAllowed(gs *state.GlobalState) bool {
	if gs.ActiveScreen == state.Crashed || gs.ActiveScreen == state.Combat {
		return false
	}
	if gs.Run != nil && runPhase(gs) == session.PhaseFight.String() {
		return false
	}
	return true
}
