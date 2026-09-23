package crashlog

// **How big a report is allowed to be, and what it gives up to fit.**
//
// A report exists to be sent, and a file too big to send is a file nobody reads. So the document
// has a ceiling and sheds tiers from the bottom until it is under one — which is the same rule the
// guarded readers in internal/game are already under, with bytes as the currency instead of trust.

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
)

// maxReport is the ceiling on the report document, and maxShot the ceiling on the picture beside
// it.
//
// **Two numbers rather than one budget, because they are two files.** The picture is a companion
// with its own name and its own line in the directory, so a large screenshot has no way to push
// the ledger out of the report — which is the tier a bug report can least afford to lose.
//
// A screen of flat fills and card art encodes to a few hundred kilobytes, so the picture's ceiling
// only ever catches something pathological. The document's is generous for the same reason: a run
// that crashes on floor one carries almost nothing, and the number is here to bound the worst case
// rather than to trim the ordinary one.
const (
	maxReport = 1 << 20 // 1 MiB
	maxShot   = 2 << 20 // 2 MiB
)

// The names a shed tier is recorded under, in Shed. **Words rather than ordinals**, on the rule
// every file the game writes is under: this one outlives the build that wrote it.
const (
	shedScene    = "scene"
	shedProblems = "problems"
	shedLedger   = "ledger"
)

// Encode is the report as it goes to the file, shedding tiers until it fits.
//
// **It decides before writing rather than writing and retrying**, which is the whole reason this
// is a function and not a loop around a file write: a crash handler gets one chance at the disk,
// and a report deleted and rewritten twice is two chances for the second attempt to be the one
// that fails. The retrying happens in memory, and the file is written once.
//
// **The common path is exactly one marshal.** Measuring each tier separately to avoid a second
// marshal would cost one marshal *per tier* on every report, to save one marshal on the rare
// report that is over — so the cheap case pays for the expensive one.
//
// **A report that cannot be got under the ceiling is written anyway.** There is nothing left to
// drop but the identity and the stack, which are the two things a bug report is actually made of,
// and a file too big to send still beats no file at all.
func Encode(r Report) ([]byte, error) {
	for {
		raw, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return nil, err
		}
		raw = append(raw, '\n')
		if len(raw) <= maxReport || !shed(&r) {
			return raw, nil
		}
	}
}

// shed drops the next tier and says whether there was one.
//
// **Bottom up, in the order the tiers go in the file**, so a truncated report still opens with the
// identity a bug report needs most. The run snapshot is never shed — it is small, it is already a
// save-safe schema, and it is what says which run this was.
//
// **The ledger goes a fight at a time, oldest first**, rather than whole. A crash mid-duel has the
// room's start state on disk and nothing since, so the fight being played is the one tier the
// snapshot cannot give; dropping the lot to save bytes would throw away the most valuable thing in
// the file to keep the least.
func shed(r *Report) bool {
	switch {
	case r.Scene != nil:
		r.Scene = nil
		r.Shed = append(r.Shed, shedScene)
	case r.Problems != nil:
		r.Problems = nil
		r.Shed = append(r.Shed, shedProblems)
	case len(r.Ledger) > 0:
		r.Ledger = r.Ledger[1:]
		// **One note however many fights go.** Shed is what tells a reader the report is partial;
		// a line per dropped fight would be the shedding taking up the room it was making.
		if !shedAlready(r.Shed, shedLedger) {
			r.Shed = append(r.Shed, shedLedger)
		}
	default:
		return false
	}
	return true
}

func shedAlready(shed []string, name string) bool {
	for _, s := range shed {
		if s == name {
			return true
		}
	}
	return false
}

// EncodeShot is the screen as a PNG, or nil if there is nothing to file.
//
// **Encoded once and measured once.** PNG has no quality dial, so a picture over the ceiling
// cannot be re-encoded smaller — the only two answers are this one or a resample, and resampling a
// screenshot in a crash handler is work done at the worst possible moment for a picture that is
// already only corroboration.
//
// **It takes a plain Go image rather than an *ebiten.Image**, on the rule this package is under:
// nothing here may link Ebitengine. The readback is the caller's, on the game goroutine, and what
// crosses the boundary is pixels. See internal/game.
func EncodeShot(img image.Image) []byte {
	if img == nil {
		return nil
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		Note("could not encode the crash screenshot: %v", err)
		return nil
	}
	if buf.Len() > maxShot {
		Note("the crash screenshot is %d bytes and the ceiling is %d; leaving it out", buf.Len(), maxShot)
		return nil
	}
	return buf.Bytes()
}
