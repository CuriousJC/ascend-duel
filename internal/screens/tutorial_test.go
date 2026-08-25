package screens

import (
	"image"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
)

// **An anchor nobody answers for is a spotlight around the empty rectangle at the origin**, which
// is the failure this whole feature cannot survive: the input gate and the lit hole are the same
// rectangle, so a step naming an anchor no scene knows would shield the entire screen around a
// corner of it and leave the player unable to click their way out.
//
// The three scenes are asked together, because an anchor belongs to whichever screen draws the
// thing it names and no one scene knows them all. It is the same tripwire an `EventKind` has with
// its choreography entry, and it exists for the same reason: the enum is the easy half to add to.
func TestEveryAnchorHasARectangle(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: 1280, ScreenHeight: 960}

	// Each scene is asked with the fields its rect functions read already filled, since a rect
	// that reports false only because a row is empty would hide a missing case.
	combat := stubCombat()

	reward := &PostBattleScene{prizes: make([]prize, 2)}
	shop := &ShopScene{shelf: make([]shelfItem, shelfSize)}

	hosts := []struct {
		name string
		host tutorialHost
	}{
		{"combat", combat},
		{"reward", reward},
		{"shop", shop},
	}

	for _, a := range tutorial.Anchors() {
		if a == tutorial.AnchorNone {
			continue // points at nothing, by definition
		}

		// **More than one scene may answer, but they must agree.** `build-card` is the case: the
		// duelist card in the build band is literally the same card in the same place on the reward
		// screen and in the shop, and both defer to `buildCardRect`. What must never happen is two
		// scenes returning *different* rectangles for one name, which is two spotlights that can
		// disagree about where a thing is.
		answered := ""
		var agreed image.Rectangle
		for _, h := range hosts {
			r, ok := h.host.tutorialRect(gs, a)
			if !ok {
				continue
			}
			if answered != "" && r != agreed {
				t.Errorf("anchor %q is %v on %s and %v on %s; one name, two places",
					a, agreed, answered, r, h.name)
			}
			answered, agreed = h.name, r
		}
		if answered == "" {
			t.Errorf("anchor %q has no rectangle on any scene", a)
		}
	}
}

// The shipped script may only name anchors that exist on the screen its phase is drawn by. A step
// pointing at the shop's shelf while the run is in a duel would drop its gate every frame and read
// as a lesson that does nothing.
//
// It is a weaker check than it sounds — a step's phase is not written down — so what it actually
// holds is that every anchor the script uses is one some scene answers for. The stronger version
// would need the script to declare its screen, which is a field nothing needs yet.
func TestTheShippedScriptOnlyNamesRealAnchors(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: 1280, ScreenHeight: 960}

	combat := stubCombat()
	hosts := []tutorialHost{
		combat,
		&PostBattleScene{prizes: make([]prize, 2)},
		&ShopScene{shelf: make([]shelfItem, shelfSize)},
	}

	for _, step := range tutorial.Load() {
		if step.Anchor == tutorial.AnchorNone {
			continue
		}
		found := false
		for _, h := range hosts {
			if _, ok := h.tutorialRect(gs, step.Anchor); ok {
				found = true
			}
		}
		if !found {
			t.Errorf("step %q points at %q, which no scene draws", step.Key, step.Anchor)
		}
	}
}

// A nil button is a scene that has not built its widgets yet, which is every scene for the frame
// before its first Init. It must read as "no anchor here" rather than as a rectangle at the
// origin — the overlay drops the gate on the first and shields the screen on the second.
func TestANilButtonIsNoRectangle(t *testing.T) {
	if r := buttonRect(nil); !r.Empty() {
		t.Errorf("a nil button gave the rectangle %v", r)
	}
}

// The bubble has to fit on the screen at every seat it can take, or a lesson ends up half off the
// edge. The panel is a fixed size, so this is arithmetic rather than a rendering check.
func TestTheBubbleFitsTheScreen(t *testing.T) {
	const w, h = 1280, 960
	if tutorialPanelW+tutorialMargin*2 > w {
		t.Errorf("the bubble is %dpx wide and the screen is %d", tutorialPanelW, w)
	}
	if tutorialPanelH+tutorialMargin*2 > h {
		t.Errorf("the bubble is %dpx tall and the screen is %d", tutorialPanelH, h)
	}
}

// The two buttons have to fit across the bottom of the bubble beside each other, since a step
// waiting on Next draws both.
func TestBothButtonsFitTheBubble(t *testing.T) {
	need := tutorialButtonW*2 + tutorialButtonGap + tutorialPad*2
	if need > tutorialPanelW {
		t.Errorf("Next and Skip need %dpx and the bubble is %d", need, tutorialPanelW)
	}
}

// The bubble moves out of the way of what it is pointing at. That is the whole reason it is not a
// modal, so it is worth a test rather than a comment.
func TestTheBubbleAvoidsWhatItPointsAt(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: 1280, ScreenHeight: 960}
	var t0 tutorialOverlay

	// A target in the bottom-centre, which is where the bubble would rather be.
	target := image.Rect(500, 700, 800, 900)
	host := fixedHost{rect: target}

	got := t0.place(gs, host, tutorial.Step{Anchor: tutorial.AnchorHand})
	if got.Overlaps(target) {
		t.Errorf("the bubble at %v covers the thing it is pointing at, %v", got, target)
	}
}

// fixedHost answers every anchor with one rectangle, for the placement test.
type fixedHost struct{ rect image.Rectangle }

func (f fixedHost) tutorialFacts(*state.GlobalState) tutorial.Facts { return tutorial.Facts{} }
func (f fixedHost) tutorialCovered(*state.GlobalState) bool         { return false }
func (f fixedHost) tutorialRect(*state.GlobalState, tutorial.Anchor) (image.Rectangle, bool) {
	return f.rect, true
}

// stubButton is a placed button, so a rect function has something to measure. The size and the
// seat do not matter to any test here — what is being checked is that the anchor is answered at
// all.
func stubButton() *models.Button {
	b := models.NewButton(88, 44, "x", func() {})
	b.ScreenX, b.ScreenY = 200, 200
	return b
}

// The leader line has to actually touch both ends. A line starting inside the bubble buries its
// first stretch under the panel and reads as shorter than it is; one stopping short of the square
// reads as pointing at nothing in particular.
func TestTheLeaderTouchesBothEnds(t *testing.T) {
	bubble := image.Rect(100, 600, 700, 860)

	for _, target := range []image.Rectangle{
		image.Rect(900, 100, 1100, 300),  // up and to the right
		image.Rect(40, 60, 240, 200),     // up and to the left
		image.Rect(560, 40, 760, 160),    // straight up
		image.Rect(1000, 700, 1200, 820), // level with it, to the right
	} {
		from := edgeToward(bubble, center(target))
		to := edgeToward(target, center(bubble))

		if !onBorder(bubble, from) {
			t.Errorf("target %v: the line leaves the bubble at %v, which is not on its edge",
				target, from)
		}
		if !onBorder(target, to) {
			t.Errorf("target %v: the line lands at %v, which is not on its edge", target, to)
		}
	}
}

// A ray aimed at a point inside the box stops at that point rather than shooting out the far
// side. It is the degenerate case that happens when the bubble ends up overlapping its own
// target — the last seat `place` falls back to.
func TestTheLeaderDoesNotOvershootAnOverlap(t *testing.T) {
	r := image.Rect(0, 0, 100, 100)
	inside := image.Pt(60, 55)
	if got := edgeToward(r, inside); got != inside {
		t.Errorf("a ray at a point inside the box went to %v, wanted %v", got, inside)
	}
}

// A target dead-centre on the bubble has no direction to point in, and must not divide by zero or
// fly off to the origin.
func TestTheLeaderSurvivesACoincidentCentre(t *testing.T) {
	r := image.Rect(100, 100, 300, 200)
	if got := edgeToward(r, center(r)); got != center(r) {
		t.Errorf("a ray at its own centre went to %v", got)
	}
}

// onBorder reports whether p sits on r's frame, within the pixel the integer arithmetic can lose.
func onBorder(r image.Rectangle, p image.Point) bool {
	const slack = 1
	nearX := abs(p.X-r.Min.X) <= slack || abs(p.X-r.Max.X) <= slack
	nearY := abs(p.Y-r.Min.Y) <= slack || abs(p.Y-r.Max.Y) <= slack

	inX := p.X >= r.Min.X-slack && p.X <= r.Max.X+slack
	inY := p.Y >= r.Min.Y-slack && p.Y <= r.Max.Y+slack

	return (nearX && inY) || (nearY && inX)
}

// A step with no Next button has to say what it is waiting for, or the bubble reads as one whose
// other button failed to draw — which is exactly how it was first reported.
//
// The table has no default arm, so a condition added without a line here would silently inherit
// nothing and print an empty hint. This is that table's tripwire.
func TestEveryConditionSaysWhatItIsWaitingFor(t *testing.T) {
	for _, step := range tutorial.Load() {
		if step.Until == tutorial.CondNext {
			continue // it has a button
		}
		if waitingFor(step.Until) == "" {
			t.Errorf("step %q waits on %q, which has no waiting line",
				step.Key, step.Until)
		}
	}
}

// And the same over the whole vocabulary rather than only the conditions the shipped script
// happens to use, since the failure arrives with the *next* script.
func TestEveryConditionInTheVocabularyHasAWaitingLine(t *testing.T) {
	for c, want := range map[tutorial.Condition]bool{
		tutorial.CondNext:        false, // has a button
		tutorial.CondCardsQueued: true,
		tutorial.CondHandEmptied: true,
		tutorial.CondDuelPressed: true,
		tutorial.CondRoundDone:   true,
		tutorial.CondPhaseFight:  true,
		tutorial.CondPhaseReward: true,
		tutorial.CondPhaseShop:   true,
	} {
		if got := waitingFor(c) != ""; got != want {
			t.Errorf("condition %q: has a waiting line = %v, wanted %v", c, got, want)
		}
	}
}

// The waiting line has to fit the slot the Next button would have taken, or it runs back under
// Skip.
func TestTheWaitingLineFitsItsSlot(t *testing.T) {
	for c := range waitingWords {
		if n := len(waitingFor(c)); n > 20 {
			t.Errorf("the waiting line for %q is %d characters, which will not fit the slot", c, n)
		}
	}
}

// **A spotlight cannot point through a panel.** The step naming the hands ladder invites the
// player to open it, and the moment they do, the button being pointed at is behind a full-screen
// dialog — the square was drawn around a control nobody could see and the leader ran off to a
// corner of the panel. The rectangle stays perfectly correct, which is why the scene has to be the
// one to say it is covered.
func TestACoveredSceneIsNotPointedAt(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: 1280, ScreenHeight: 960}

	s := stubCombat()

	if s.tutorialCovered(gs) {
		t.Fatal("a scene with no dialog up reported itself covered")
	}

	// Opening the hands ladder is exactly what the tutorial invites, and it covers the button the
	// step is pointing at.
	s.hands.open = true
	if !s.tutorialCovered(gs) {
		t.Error("the hands panel is up and the scene does not report itself covered")
	}
	s.hands.open = false

	s.showDeck = true
	if !s.tutorialCovered(gs) {
		t.Error("the deck overlay is up and the scene does not report itself covered")
	}
}

// A covered scene must drop the input gate too, not merely stop drawing the square. Shielding the
// screen around a rectangle the player cannot see leaves them one legal click with no way to find
// it — and the dialog's own X, which is what actually gets them out, would be outside the shield.
func TestACoveredSceneDropsTheGate(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: 1280, ScreenHeight: 960}
	gs.Run = session.New(nil)
	gs.Run.Teach(tutorial.Script{
		{Key: "press", Text: "press it", Anchor: tutorial.AnchorDuelButton,
			Lock: tutorial.LockToAnchor, Until: tutorial.CondDuelPressed},
	})

	s := stubCombat()

	var overlay tutorialOverlay
	if _, gated := overlay.focus(gs, s); !gated {
		t.Fatal("an uncovered gating step did not gate")
	}

	s.showDeck = true
	if _, gated := overlay.focus(gs, s); gated {
		t.Error("a gating step still shielded the screen from behind a dialog")
	}
}

// stubCombat is a combat scene with enough on it for every anchor it answers for to resolve.
//
// **The hand is part of that, and the omission was the bug the test caught**: `first-card` is
// measured off the leftmost card, so a scene with no hand reports it has no such anchor — which is
// correct behaviour and looks exactly like an anchor nobody wrote a case for.
func stubCombat() *CombatScene {
	s := &CombatScene{}
	s.duelButton = stubButton()
	s.hands.button = stubButton()
	s.hand = make([]paletteCard, 5)
	return s
}

// **A read step locks the screen, and this is the end-to-end version of it.** The unit test in
// internal/tutorial pins that the Lock is derived correctly; this pins that the overlay actually
// shields the game with it. The bug was visible on screen — Bob describing the tower while the
// player queued two cards behind him — so it is worth checking at the layer where it was visible.
func TestAReadStepShieldsTheWholeScreen(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: 1280, ScreenHeight: 960}
	gs.Run = session.New(nil)
	gs.Run.Teach(tutorial.Script{
		{Key: "rooms", Text: "eight floors", Anchor: tutorial.AnchorTowerPlace,
			Lock: tutorial.LockAll, Until: tutorial.CondNext},
	})

	var overlay tutorialOverlay
	focus, gated := overlay.focus(gs, stubCombat())
	if !gated {
		t.Fatal("a read step left the screen live")
	}

	// An empty focus contains no point, so nothing outside Bob's own buttons can be clicked.
	gs.InputGated, gs.InputFocus = gated, focus
	for _, at := range []image.Point{
		{X: 250, Y: 750}, // a card in the hand
		{X: 640, Y: 480}, // the middle of the table
		{X: 30, Y: 930},  // the mute button in the corner
	} {
		if gs.InputAllowed(at) {
			t.Errorf("a read step still accepts a click at %v", at)
		}
	}
}

// And an outcome step must leave the screen alone, or the tutorial deadlocks against its own
// condition: winning a fight needs clicks on controls the step never names.
func TestAnOutcomeStepLeavesTheScreenLive(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: 1280, ScreenHeight: 960}
	gs.Run = session.New(nil)
	gs.Run.Teach(tutorial.Script{
		{Key: "win", Text: "again and again", Lock: tutorial.LockNone,
			Until: tutorial.CondPhaseReward},
	})

	var overlay tutorialOverlay
	if _, gated := overlay.focus(gs, stubCombat()); gated {
		t.Error("a step waiting for a fight to be won shielded the screen")
	}
}

// **The waiting line may not sit on top of the Skip button.** Skip slides right into the slot Next
// would have taken when there is no Next, so a hint pinned to the panel's right edge lands
// underneath it — which is exactly how it first shipped, with "take them all" drawn through
// "Skip".
func TestTheWaitingLineClearsTheSkipButton(t *testing.T) {
	var overlay tutorialOverlay
	overlay.panel = image.Rect(100, 600, 100+tutorialPanelW, 600+tutorialPanelH)
	overlay.build()

	// A step with no Next: Skip takes the rightmost slot and the hint has to go to its left.
	overlay.placeButtons(tutorial.Step{Until: tutorial.CondHandEmptied})

	skipLeft := overlay.skip.ScreenX - tutorialButtonW/2
	hintRight := overlay.skip.ScreenX - tutorialButtonW/2 - tutorialButtonGap

	if hintRight >= skipLeft {
		t.Errorf("the hint's right edge is %d and Skip starts at %d", hintRight, skipLeft)
	}
	if hintRight <= overlay.panel.Min.X+tutorialPad+tutorialCardW {
		t.Errorf("the hint at %d runs back over Bob's card", hintRight)
	}
}
