//go:build demoplay

package screens

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

// The demo driver: **the game plays a scripted round with nobody at the controls and writes
// the screen to PNG files**, so a change to the combat screen can be reviewed by opening
// pictures rather than by asking someone to sit and play it.
//
//	go run -tags demoplay .        # writes demo/*.png and exits
//
// It exists for the same reason `tools/glyphsheet` does. Glyph art was being reviewed by
// launching the game and hunting for a card that used it; the sheet replaced that with a file.
// This is that idea applied to the one thing a sheet cannot capture — a screen mid-playback,
// with a combo line on it and the highlight on the right row.
//
// **It is a build tag rather than a flag, and it must stay deletable in one commit.** A game
// that plays itself is not a thing to ship, and `go build .` carries none of this: no script,
// no PNG encoder, no file writes.
//
// **It may never change an outcome**, the same constraint as trace, idle and playback speed.
// It presses the same buttons a player would — `toggle` for selection, `startRound` for DUEL!
// — and `ResolveRound` neither sees it nor is called by it.
//
// **It picks its own opening hand by name**, via the seed catalogue in seeds.go. That is what
// lets round one be *clicked* rather than forced: `strike-flurry` guarantees three Strikes in
// hand, so the demo selects them through `toggle` exactly as a player would and the combo
// fires off a real selection. Before named seeds the deal held no three-of-a-kind and the only
// way to see a combo was to write the queue directly, which tested the pane but not the path
// to it.
//
// Round two is still written straight into `fighterActions`, because what it is for is putting
// all three verb chips on screen at once and the hand it is dealt after round one's discard is
// not something the seed pins.

// demoPlans is one scripted round each, played in order. **Between them they have to cover
// every verb chip**, because the chips are the thing hardest to check any other way and the
// enemy will not supply them: Monster1 is a brute and only ever attacks.
//
//   - Round 1 spends 6 AP on a Gather, a Riposte and a Strike, which puts a white *prepares*,
//     a blue *defends* and a red *attacks* on screen in one pane.
//   - Round 2 is three Strikes, which forms a Strike Flurry — the combo line, the amber
//     swatch, and a stagger on the opponent's turn. It is affordable because the Gather in
//     round 1 banked +2, so the budget is 8.
//
// The order matters and is not arbitrary: three Strikes is exactly a 6 AP budget, so a Gather
// cannot be added to the combo round, and the combo round cannot come first without the
// Gather having nowhere to go.
// Round one is clicked, not listed — see demoClickRun. Round two is the scripted one.
var demoPlans = [][]combat.ActionKind{
	{combat.Gather, combat.Riposte, combat.Strike},
}

// demoSeedName is the opening hand the demo asks for, overriding whatever a plain launch uses.
// `strike-flurry` puts three Strikes in hand, which is what round one clicks.
const demoSeedName = "strike-flurry"

// demoClickRun is the card round one selects three of. It has to be a card the chosen seed
// actually deals three of, or the click phase quietly selects fewer and no combo forms —
// `tools/seeds` is what keeps those two facts in agreement.
var demoClickRun = combat.Strike

func init() { deckSeed = seedFor(demoSeedName) }

// The script, in ticks at 60 TPS. Held as package state rather than on CombatScene so that
// combat.go gains only its two call sites and nothing else has to know this exists.
var demo struct {
	tick      int
	round     int // how many scripted plans have been sent
	shots     int
	lastShot  int // the tick a capture was last taken on
	mode      demoShotMode
	shotEvery int // ticks between captures, wider in keys mode than in all
	done      bool
}

const (
	demoSelectAt = 40 // click a couple of cards, so the hand is shown mid-selection
	demoShotHand = 70 // capture the planning screen
	demoDuelAt   = 80 // press DUEL!
	demoShotFrom = 86 // start capturing playback

	// **One capture per event, derived from the dwell rather than written down.** It was a
	// flat 22 and went stale the moment playback slowed to a second and a quarter a beat —
	// the same number then took three pictures of every event. Tying it to `eventDwellTicks`
	// means a pacing change cannot leave the harness sampling at the wrong rate.
	demoShotFor = eventDwellTicks

	// How long to hold after a round settles before sending the next plan, so the settled
	// screen is on show long enough to be captured rather than passed straight through.
	demoBetweenRounds = 40

	// How many of one card round one clicks. Named against the combo it is trying to form
	// rather than written as a 3, so it stays honest if the flurry run length ever changes.
	flurryRunCards = 3

	// A stop, because a script that never ends is a window nobody closed. Two rounds of
	// playback are well inside this; reaching it means something hung and the captures taken
	// so far are the evidence.
	//
	// Sized against the dwell rather than as a flat number, so slowing playback down cannot
	// quietly turn the safety net into the thing that ends the run. Sixty events is roughly
	// double what two rounds produce.
	demoGiveUpAt = 60 * eventDwellTicks
)

func (s *CombatScene) demoUpdate(gs *state.GlobalState) {
	if demo.done {
		return
	}
	demo.tick++

	if demo.tick == 1 {
		demo.mode = demoShotsWanted()
		demo.shotEvery = demoShotFor
		if demo.mode == demoShotsKeys {
			demo.shotEvery = demoShotFor * keyShotsPerRound
		}
		if demo.mode == demoShotsOff {
			fmt.Println("demo: captures off (ASCEND_DUEL_DEMO_SHOTS=keys|all to write PNGs)")
		} else {
			demoClean()
		}
	}

	switch demo.tick {
	case demoSelectAt:
		// Round one, clicked: three of one card through the real selection path, so the combo
		// that follows came out of a hand the player could have picked themselves.
		picked := 0
		for i := range s.hand {
			if picked >= flurryRunCards {
				break
			}
			if s.hand[i].actionCard.action == demoClickRun {
				s.toggle(i)
				picked++
			}
		}
		if picked < flurryRunCards {
			fmt.Printf("demo: seed %q dealt only %d %v - no combo will form; re-run tools/seeds\n",
				demoSeedName, picked, demoClickRun)
		}

	case demoDuelAt:
		s.startRound()
	}

	// Playback finished. Either send the next scripted round or close the window, the same
	// way the close button does.
	settled := demo.tick > demoDuelAt+demoBetweenRounds && s.cursor >= len(s.log)
	if settled {
		// s.round rather than demo.round: the first round is clicked rather than scripted, so
		// the two do not line up and the scene's own count is the one that is true.
		s.demoReport(gs, fmt.Sprintf("round %d resolution", s.round))
		if demo.round < len(demoPlans) {
			s.demoSendPlan()
			return
		}
		demo.done = true
		gs.ShouldClose = true
	}

	if demo.tick > demoGiveUpAt {
		fmt.Println("demo: gave up waiting; captures so far are in demo/")
		demo.done = true
		gs.ShouldClose = true
	}
}

// demoSendPlan writes the next scripted plan straight into the queue and presses DUEL!.
// Over-budget plans are refused by startRound rather than resolved, which would leave the
// script waiting on a round that never began — so it says so instead of hanging until the
// give-up tick.
func (s *CombatScene) demoSendPlan() {
	if demo.round >= len(demoPlans) {
		return
	}
	plan := demoPlans[demo.round]
	demo.round++

	s.fighterActions = append([]combat.ActionKind(nil), plan...)
	if s.overBudget() {
		fmt.Printf("demo: plan %d costs %d against a budget of %d and will not resolve\n",
			demo.round, combat.CostOf(plan), s.fighter.ActionPoints())
	}
	s.startRound()
}

// **Captures are off unless asked for**, because a 1280x960 PNG is expensive for whoever ends
// up reading it and most questions do not need one. What the pane *says* is answered by
// demoReport below for almost nothing; a picture is only needed for how it *looks* — colour,
// spacing, alignment.
//
//	go run -tags demoplay .                             # text report only
//	ASCEND_DUEL_DEMO_SHOTS=keys go run -tags demoplay . # a handful of frames
//	ASCEND_DUEL_DEMO_SHOTS=all  go run -tags demoplay . # one frame per event
type demoShotMode int

const (
	demoShotsOff demoShotMode = iota
	demoShotsKeys
	demoShotsAll
)

func demoShotsWanted() demoShotMode {
	switch os.Getenv("ASCEND_DUEL_DEMO_SHOTS") {
	case "keys", "1", "true":
		return demoShotsKeys
	case "all":
		return demoShotsAll
	default:
		return demoShotsOff
	}
}

// keyShotsPerRound is how many frames `keys` mode takes of a round's playback, spread evenly
// across it. Three is enough to see the pane fill, a combo land and the round settle without
// producing a flipbook.
const keyShotsPerRound = 3

func (s *CombatScene) demoDraw(gs *state.GlobalState, screen *ebiten.Image) {
	if demo.done || demo.shots == 0 && demo.mode == demoShotsOff {
		return
	}
	if demo.mode == demoShotsOff || demo.tick == demo.lastShot {
		// Ebiten can call Draw more than once for a tick, and a tick-equality test fires
		// again each time — which is how the first run produced every capture twice.
		return
	}

	switch {
	case demo.tick == demoShotHand:
		demo.lastShot = demo.tick
		demoCapture(screen, "01-planning")

	case demo.tick >= demoShotFrom && (demo.tick-demoShotFrom)%demo.shotEvery == 0:
		demo.lastShot = demo.tick
		demo.shots++
		demoCapture(screen, fmt.Sprintf("%02d-play", demo.shots+1))
	}
}

// demoReport prints the Resolution pane as text: the same rows the pane draws, flattened.
//
// **This is the cheap way to check what a round said**, and it is what the captures were being
// read for most of the time. It cannot answer a question about colour or spacing, and it is
// not meant to — it answers "did the combo fire, in the right place, with the right words",
// which is most of them.
func (s *CombatScene) demoReport(gs *state.GlobalState, label string) {
	fmt.Printf("\n--- %s ---\n", label)
	for _, row := range s.resolutionLines(gs) {
		line := row.prefix + row.verb + row.suffix
		if line == "" {
			continue
		}
		fmt.Printf("  %s\n", line)
	}
}

// demoClean empties the capture directory before a run fills it.
//
// **Without it a run's output is the union of every run before it.** A script that captures
// thirty frames after one that captured fifty leaves twenty stale pictures behind, numbered
// exactly like the live ones and indistinguishable from them — which is the same failure as a
// stale glyph sheet, a picture that lies, and worse here because the file names encourage you
// to read them as a sequence.
//
// It removes only `*.png` in that one directory rather than the directory itself, so a path
// typo cannot take anything else with it.
func demoClean() {
	matches, err := filepath.Glob(filepath.Join(demoDir, "*.png"))
	if err != nil {
		fmt.Printf("demo: %v\n", err)
		return
	}
	for _, m := range matches {
		if err := os.Remove(m); err != nil {
			fmt.Printf("demo: %v\n", err)
		}
	}
	if len(matches) > 0 {
		fmt.Printf("demo: cleared %d capture(s) from a previous run\n", len(matches))
	}
}

// demoDir is where captures go. Gitignored — see .gitignore.
const demoDir = "demo"

// demoCapture writes the screen to demo/<name>.png. A nil screen writes nothing and exists
// only so the settle step can be expressed in demoUpdate beside the state change it belongs
// with; the frame it wants is the one demoDraw already took.
//
// **Synchronous, unlike trace's capture.** trace throttles and hands the encode to a
// goroutine because it runs alongside someone playing and must not cost them a frame. Nobody
// is playing this, the window is about to close, and a capture still being encoded when the
// process exits is a file that does not appear.
func demoCapture(screen *ebiten.Image, name string) {
	if screen == nil {
		return
	}

	b := screen.Bounds()
	img := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	screen.ReadPixels(img.Pix)

	if err := os.MkdirAll(demoDir, 0o755); err != nil {
		fmt.Printf("demo: %v\n", err)
		return
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		fmt.Printf("demo: %v\n", err)
		return
	}

	path := filepath.Join(demoDir, name+".png")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		fmt.Printf("demo: %v\n", err)
		return
	}
	fmt.Printf("demo: wrote %s\n", path)
}
