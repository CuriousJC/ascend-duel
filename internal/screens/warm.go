package screens

// Painting the run's card faces ahead of time, a few a frame.
//
// **What it is buying is the first deal.** Every card face is painted pixel by pixel in Go and then
// uploaded to the graphics card the first time it is drawn, and a fresh duel wants dozens of them
// on the frames its hand is flying out of the pile — which is a deal that hangs and lurches. The
// title screen spends that work up front, behind a turning swirl, before its menu appears.
//
// **The work is spread across frames by a time budget**, so whatever is moving keeps moving while it
// happens. The clock read is presentation — how long to keep painting this frame — and reaches
// nothing a rule reads.

import (
	"time"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// warmBudget is how long a frame may spend painting faces, which leaves the rest of a 60Hz frame
// for everything else.
const warmBudget = 8 * time.Millisecond

// faceWarmer is the list of faces still to paint.
type faceWarmer struct {
	jobs []func()
}

// start gathers every face the run will want first: its deck at the hand's size and the deck
// panel's, and every worn relic. A face already painted costs a cache lookup, so gathering again
// on a later visit is cheap.
func (w *faceWarmer) start(gs *state.GlobalState) {
	ui.UseImages(gs)
	w.jobs = nil
	if gs.Run == nil {
		return
	}
	w.jobs = append(w.jobs, ui.DeckWarmJobs(gs)...)
	counters := runCounters(gs)
	for _, record := range wornRelics(gs) {
		spec := ui.RelicSpec(gs, record, counters[record.RelicRecord], true, false)
		w.jobs = append(w.jobs, func() { ui.WarmFace(gs, spec, cards.RelicStyle) })
	}
}

// step paints until this frame's budget is spent, and reports whether anything is left.
func (w *faceWarmer) step() bool {
	start := time.Now()
	for len(w.jobs) > 0 && time.Since(start) < warmBudget {
		w.jobs[0]()
		w.jobs = w.jobs[1:]
	}
	return len(w.jobs) > 0
}
