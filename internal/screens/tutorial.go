package screens

// **Bob, and the shape he draws.**
//
// `internal/tutorial` decides which step is up and when it gives way; this file is the half of
// the feature that has a window. It owns three things and nothing else:
//
//   - **the rectangle behind an anchor name.** An anchor is a word in `data/tutorial.json`; a
//     rectangle is a fact about a layout, and layouts live here. Each scene answers for its own
//     anchors through `tutorialHost`, and `TestEveryAnchorHasARectangle` fails if a name nobody
//     answers for is added to the enum.
//   - **the bubble** — Bob's card, what he is saying, and the two buttons under it.
//   - **the spotlight**, which is the *same rectangle* the input gate uses. That is the load-
//     bearing part: a lit hole the player cannot click, or a clickable region that is not lit,
//     would each be worse than no tutorial at all.
//
// **It is deliberately not a `modalToggle`.** Every other dialog in the game takes one footprint,
// scrims the whole screen and sets `gs.ModalOpen` — see modal.go, whose argument is that the
// player should learn one shape. This is a second shape on purpose *(owner's call, 2026-08-25)*
// and the reason is structural rather than aesthetic: a panel at the modal footprint covers 91%
// of the screen, and a thing whose whole job is to point at what is underneath cannot be the
// thing covering it. So the bubble is small, it moves out of its own way, and the game beneath
// stays live.

import (
	"image"
	"image/color"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/achieve"
	"github.com/curiousjc/ascend-duel/internal/ui"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// tutorialHost is a scene that can be taught: it says what is true right now, and it answers for
// the anchors that name things it draws.
//
// **Two methods rather than a registration call**, because a scene's rectangles are methods on
// the scene and most of them are only meaningful while it is the one on screen. An anchor asked
// of the wrong scene reports `false` and the bubble simply points at nothing, which is the right
// behavior for the frame or two either side of a scene change.
// tutorialClearance is a scene that knows what the bubble must stay off for an anchor beyond what
// the anchor lights — the reward screen's row of cards, which the essence step asks the player to
// choose from without pointing at it. Optional: a scene without one keeps only the anchor clear.
type tutorialClearance interface {
	tutorialKeepClear(gs *state.GlobalState, a tutorial.Anchor) []image.Rectangle
}

type tutorialHost interface {
	// tutorialFacts is what this scene can say about the run right now. See tutorial.Facts for
	// why the traffic goes this way rather than as events.
	tutorialFacts(gs *state.GlobalState) tutorial.Facts

	// tutorialRects is where this scene draws the thing an anchor names, and whether it knows the
	// anchor at all.
	//
	// **A list, because an anchor may name a set** *(2026-09-08)*. Nearly every anchor is one
	// control in one place and hands back one rectangle; `matching-cards` and `shattered-cards`
	// name a set of cards that need not be adjacent, and the bounding box round them is not the
	// set — it is the set plus whatever is sitting between two of them. That distinction is
	// load-bearing here and nowhere else in the game: this rectangle is the *click gate* as well
	// as the spotlight, so a box was a license to click a card the step had not named. See
	// state.GlobalState.InputFocus, where the bug it caused is written down.
	tutorialRects(gs *state.GlobalState, a tutorial.Anchor) ([]image.Rectangle, bool)

	// tutorialCovered is whether one of this scene's own dialogs is over the top of everything.
	//
	// **A spotlight cannot point through a panel** *(2026-08-25)*. The step naming the hands ladder
	// invites the player to open it, and the moment they do, the button being pointed at is behind
	// a full-screen dialog — so the square was drawn around a control nobody could see and the
	// leader ran off to a corner of the panel. The rectangle is still perfectly correct, which is
	// what makes this worth a method rather than a special case: the *screen* knows something is
	// covering it and nothing else can.
	tutorialCovered(gs *state.GlobalState) bool
}

// The bubble's footprint and the furniture inside it.
const (
	// Bob is drawn at RelicStyle's 200x280 — a card with a name and a picture and nothing else,
	// which is exactly what he is. EnemyStyle would give him a health bar.
	tutorialCardW = 200
	tutorialCardH = 280

	tutorialPad      = 20
	tutorialTextW    = 400
	tutorialTextSize = systems.TextMedium

	// tutorialLinePitch is the text size plus the gap that keeps two lines of Kubasta from
	// touching. Prose rather than the cards' clipped register, so this is the one place in the
	// game setting a paragraph.
	tutorialLinePitch = 27

	tutorialButtonW   = 110
	tutorialButtonH   = ui.ButtonSmall
	tutorialButtonGap = 12

	tutorialPanelW = tutorialPad*3 + tutorialCardW + tutorialTextW
	tutorialPanelH = tutorialPad*2 + tutorialCardH

	// tutorialMargin is how far the bubble keeps off the screen edges, and off the thing it is
	// pointing at.
	tutorialMargin = 24
)

// tutorialInk is the bubble's colors. It is a dark panel over a light table, matching the fight
// log and the deck overlay rather than the cards — Bob is chrome, not something in play.
//
// **The surface and the prose ink are the tooltip's** *(owner's call, 2026-09-18)*. This panel and
// that one are the same object — a dark box of type over a light table — and they were two
// hand-picked palettes a degree apart, which is the drift the color rule exists to stop. What is
// still this file's is the *alpha*: a tooltip is faintly transparent because it covers the thing
// it explains, and the bubble is opaque because the spotlight is what points at things here.
var (
	tutorialPanel = opaque(systems.PanelSurface)
	tutorialText  = systems.PanelSpeech

	// tutorialWaiting is the line standing where a Next button would be. **Dimmer than the prose
	// it sits under**, because it is a state and not something Bob is saying.
	//
	// **Derived rather than picked**, so it stays one step off whatever the panel is: a hand-picked
	// gray stops being a step off the surface the moment the surface moves.
	tutorialWaiting = systems.ColorToward(systems.PanelInk, tutorialPanel, 40)

	// tutorialAction is the waiting line when it is something the player has to do. **Amber, which
	// sits between fire's orange and lightning's yellow** — the wheel is full — so it is set in
	// capitals in the interface's own lettering, where no element word ever appears, to read as an
	// instruction rather than as an element.
	tutorialAction = color.RGBA{R: 255, G: 190, B: 40, A: 255}

	// tutorialGlow is the square drawn around whatever is being pointed at, and the leader line
	// running to it from the bubble. **Red** *(owner's call, 2026-08-25)*.
	//
	// **It is the one place red is not a control**, and that is worth knowing rather than
	// discovering: `modalCloseColor` is the dialog X and CLAUDE.md records red as belonging to it
	// alone, so that a red thing on screen always means "this closes something". A highlight does
	// not dilute the *click* meaning — there is nothing here to press — but the color is no longer
	// unique to the exit, and a future red control would now be the third thing wearing it.
	//
	// A shade off the X's own, so the two are not mistaken for the same object on a screen showing
	// both: brighter and slightly orange, which is a mark rather than a button face.
	tutorialGlow = color.RGBA{R: 232, G: 60, B: 48, A: 255}

	// tutorialShade is the scrim laid over everything outside the spotlight. **Lighter than the
	// modal scrim's 190**, because this one is not saying "that is inert", it is saying "not
	// that, this" — and a player still has to be able to read the board they are being taught.
	tutorialShade = color.RGBA{A: 120}
)

// tutorialOverlay is the widget: two buttons and the memory of whether they were pressed.
//
// **One per scene, built lazily**, exactly as the modal closer is. A scene that is never taught
// never builds one.
type tutorialOverlay struct {
	next, skip *models.Button

	// placedFor is the step the bubble was last placed for, and placedPointing whether its anchor
	// had anything to point at then. See update: the bubble settles once per step.
	placedFor      string
	placedText     string
	placedPointing bool

	nextPressed bool
	skipPressed bool

	// finished latches the one frame the lesson ends on. **A latch rather than the `!run.Active()`
	// test alone**, because that test stays true for the rest of the session: without it the
	// tutorial-finished moment would be raised on every frame of every screen after the lesson, and
	// while awarding is idempotent, walking the catalog sixty times a second to be told nothing
	// changed is not.
	finished bool

	// panel is where the bubble was last placed, kept so that draw and update agree about where
	// the buttons are without placing them twice from two call sites.
	panel image.Rectangle
}

// runOf is the teaching run, or nil. Every caller here tolerates nil, because a run nobody is
// teaching is the overwhelmingly common case and asking twice at every site is how one gets
// missed.
func runOf(gs *state.GlobalState) *tutorial.Run {
	if gs.Run == nil {
		return nil
	}
	return gs.Run.Tutorial()
}

// tutorialUp reports whether Bob is on screen for this frame.
func tutorialUp(gs *state.GlobalState) bool { return runOf(gs).Active() }

// update runs the tutorial for one frame: it places the bubble, works the two buttons, sets the
// input gate for everything else on the screen, and advances the step if the scene says what the
// step was waiting for has happened.
//
// **It must be called before the scene's own input**, because the gate it sets is what the rest
// of the screen reads. A scene that ran its widgets first would give the player one live frame
// per step on controls the lesson has closed.
func (t *tutorialOverlay) update(gs *state.GlobalState, host tutorialHost) {
	run := runOf(gs)
	if !run.Active() {
		return
	}
	step, _ := run.Current()

	// The gate goes up before the buttons are worked, and Bob's own two are placed inside the
	// exception below — otherwise the shield would close the only controls that can dismiss it.
	focus, gated := t.focus(gs, host)
	gs.InputGated = gated
	gs.InputFocus = focus

	// **The bubble settles once per step** *(owner's call)*, so a card lifting as it is selected does
	// not move it. It is placed again while the step's anchor has nothing to point at yet — a hand
	// still being dealt — and holds from the first frame it does.
	//
	// **A step repeating the line before it keeps that step's seat** — choosing the cards, pressing
	// DUEL! and watching the round are one sentence across three steps — unless it would cover what
	// the new step points at.
	if step.Key != t.placedFor || !t.placedPointing {
		rs, ok := host.tutorialRects(gs, step.Anchor)
		keep := step.Key != t.placedFor && step.Text == t.placedText && t.placedPointing &&
			!(ok && t.panel.Overlaps(unionOf(rs).Inset(-tutorialMargin)))
		if !keep {
			t.panel = t.place(gs, host, step)
		}
		t.placedFor, t.placedText = step.Key, step.Text
		t.placedPointing = step.Anchor == tutorial.AnchorNone || (ok && len(rs) > 0)
	}
	t.build()
	t.nextPressed, t.skipPressed = false, false

	// **Bob's buttons are worked with the shield down.** They sit outside the anchor by
	// construction — the bubble is placed away from whatever is being pointed at — so leaving the
	// gate up over them would make Skip unreachable on exactly the steps a stuck player wants it.
	wasGated := gs.InputGated
	gs.InputGated = false
	t.placeButtons(step)
	systems.UpdateButton(gs, t.skip)
	if step.Until == tutorial.CondNext {
		systems.UpdateButton(gs, t.next)
	}
	gs.InputGated = wasGated

	if t.skipPressed {
		run.Skip()
		t.finish(gs)
		gs.InputGated = false
		return
	}

	run.Update(host.tutorialFacts(gs), t.nextPressed)

	// **The gate is dropped the moment the script ends**, here rather than being left for the
	// next frame's early return. A tutorial that finished on the frame it also set a focus would
	// leave the screen shielded around a rectangle nothing is drawing any more.
	if !run.Active() {
		gs.InputGated = false
		t.finish(gs)
	}
}

// coversCursor is whether Bob's bubble is under the cursor right now.
//
// **It exists for the reward screen's narration** *(2026-09-08)*, which reads the raw mouse rather
// than a widget and does so with the gate ignored, so that a taught player can fill the payout the
// way an untaught one can. Without this, the click on NEXT would also be a click on the screen
// underneath it, and one press would answer two questions.
func (t *tutorialOverlay) coversCursor(gs *state.GlobalState) bool {
	if !tutorialUp(gs) {
		return false
	}
	x, y := ebiten.CursorPosition()
	return image.Pt(x, y).In(t.panel)
}

// finish records the lesson as over, once. **Skipping counts as finishing** — see markTutorialSeen,
// which makes the same call for the same reason.
func (t *tutorialOverlay) finish(gs *state.GlobalState) {
	if t.finished {
		return
	}
	t.finished = true
	markTutorialSeen(gs)
	earnMoment(gs, achieve.TutorialFinished())
}

// focus is the rectangle input is being held to, and whether it is being held at all.
//
// **An anchor the current scene cannot answer for drops the gate rather than closing the whole
// screen.** That happens for a frame either side of a scene change, and the alternative — a
// shield around an empty rectangle — is a game the player cannot click at all.
func (t *tutorialOverlay) focus(gs *state.GlobalState, host tutorialHost) ([]image.Rectangle, bool) {
	run := runOf(gs)
	anchor, lock := run.Gate()
	if lock == tutorial.LockNone {
		return nil, false
	}

	// **Locked with no hole in it.** An empty focus rectangle contains no point, so
	// `InputAllowed` refuses everything — which is exactly right for a step whose only request is
	// that it be read. Bob's own two buttons are worked with the shield down and stay live; see
	// update.
	if lock == tutorial.LockAll {
		return nil, true
	}
	// **A covered anchor drops the gate**, for the reason an unknown one does: shielding the screen
	// around a rectangle the player cannot see leaves them with one legal click they have no way to
	// find. The dialog's own X is then the only thing to press, which is what it is for.
	if host.tutorialCovered(gs) {
		return nil, false
	}
	rs, ok := host.tutorialRects(gs, anchor)
	if !ok || len(rs) == 0 {
		return nil, false
	}
	return rs, true
}

// unionOf is the bounding box of a set of rectangles, for the two things that genuinely want one:
// where to put the bubble, and where to point the leader line. **Never for the gate** — that is the
// whole distinction this file now keeps.
func unionOf(rs []image.Rectangle) image.Rectangle {
	var out image.Rectangle
	for _, r := range rs {
		if out.Empty() {
			out = r
			continue
		}
		out = out.Union(r)
	}
	return out
}

// build makes the two buttons on first use. The Next button is crimson like DUEL! — it is the
// same "and on with it" slot — and Skip is the quiet gray the sort column uses, because leaving
// the tutorial should be findable without being the loudest thing in the bubble.
func (t *tutorialOverlay) build() {
	if t.next == nil {
		t.next = models.NewButton(tutorialButtonW, tutorialButtonH, "NEXT",
			func() { t.nextPressed = true })
		t.next.BaseColor = ui.ButtonRed
		t.next.TextSize = 36
	}
	if t.skip == nil {
		t.skip = models.NewButton(tutorialButtonW, tutorialButtonH, "SKIP",
			func() { t.skipPressed = true })
		t.skip.BaseColor = ui.ButtonJade
		t.skip.TextSize = 36
	}
}

// placeButtons puts the pair along the bottom of the bubble, under the text column.
//
// **Skip is always there and Next only on a step that waits for it.** A step waiting on the
// player to do something has no Next, because a button that skipped past the one thing the step
// exists to teach would be the fastest route to a player who has read the tutorial and cannot
// play the game.
func (t *tutorialOverlay) placeButtons(step tutorial.Step) {
	y := t.panel.Max.Y - tutorialPad - tutorialButtonH/2
	right := t.panel.Max.X - tutorialPad - tutorialButtonW/2

	if step.Until == tutorial.CondNext {
		t.next.ScreenX, t.next.ScreenY = right, y
		right -= tutorialButtonW + tutorialButtonGap
	}
	t.skip.ScreenX, t.skip.ScreenY = right, y
}

// place decides where the bubble goes: the first candidate seat that does not cover what the step
// is pointing at.
//
// **Moving out of its own way is the whole design.** A fixed bubble would have to be small enough
// never to overlap anything worth pointing at, which on a 1280x960 screen with a card in each
// corner is nowhere.
//
// The candidates are ordered so the common case is stable: bottom-center first, because most of
// what a lesson points at is a card or a corner control, then the four corners.
//
// **A step whose anchor rules out every seat gets the one that covers least of it** *(owner's call,
// 2026-09-06)*, rather than the last seat in the list. The reward screen is what wanted it: its
// essence anchor covers the prizes *and* the row of cards they are aimed at — the whole middle of the
// screen — so every seat overlaps, and falling through to dead center put the bubble squarely over
// the two essences the step was telling the player to choose between. Top-center misses by 27 pixels.
// A bubble overlapping a spotlit anchor is legible where no bubble at all would be a lesson with no
// words, so the fallback still places one; it just stops picking the worst available spot on
// purpose.
func (t *tutorialOverlay) place(gs *state.GlobalState, host tutorialHost,
	step tutorial.Step) image.Rectangle {

	w, h := tutorialPanelW, tutorialPanelH
	left, right := tutorialMargin, gs.ScreenWidth-tutorialMargin-w
	top, bottom := tutorialMargin, gs.ScreenHeight-tutorialMargin-h
	middle := (gs.ScreenWidth - w) / 2

	// **Top-center is second, ahead of the corners** *(owner's call, 2026-08-25)*. Most of what a
	// step points at during a duel spans the screen — the hand, the AP bar, the band the blow is
	// added up in — so the first seat is out and the fallback used to be a top corner, which is
	// where the two fighter cards and their life bars are. The middle of the top row is the relic
	// pane, which is the least costly thing on this screen to cover.
	//
	// The dead center stays last, because it is over the table: it is where a seat lands only when
	// everything else is ruled out.
	seats := []image.Point{
		{X: middle, Y: bottom},
		{X: middle, Y: top},
		{X: left, Y: top},
		{X: right, Y: top},
		{X: left, Y: bottom},
		{X: right, Y: bottom},
		{X: middle, Y: (gs.ScreenHeight - h) / 2},
	}

	targets, pointing := host.tutorialRects(gs, step.Anchor)
	target := unionOf(targets)
	if step.Anchor == tutorial.AnchorNone || !pointing || target.Empty() {
		// **A step waiting on an outcome is one the player watches the game through** — a round
		// playing out, a fight being finished — so it sits at the top, over the panes, and leaves the
		// table and the arithmetic in the middle of it visible. That is a step with neither a Next
		// nor a click to wait for; a round playing back locks the screen, so the lock cannot say it.
		if step.Until != tutorial.CondNext && step.Lock != tutorial.LockToAnchor {
			return image.Rect(middle, top, middle+w, top+h)
		}
		// Nothing to avoid: an opening or closing line belongs in the middle of the screen.
		return image.Rect(middle, (gs.ScreenHeight-h)/2, middle+w, (gs.ScreenHeight-h)/2+h)
	}

	// Kept off the anchor by the same margin it keeps off the screen edge, so a bubble does not
	// end up touching the thing it is pointing at.
	if c, ok := host.(tutorialClearance); ok {
		for _, r := range c.tutorialKeepClear(gs, step.Anchor) {
			target = target.Union(r)
		}
	}
	avoid := target.Inset(-tutorialMargin)

	// **A step that describes a card sits in the middle, straight above it** *(owner's call)*: it
	// is read rather than acted on, the table is empty while the player plans, and the middle is
	// the seat nearest a card in the hand. Only when it is clear of the card.
	if step.Until == tutorial.CondNext && step.Anchor.NamesCards() {
		mid := seats[len(seats)-1]
		r := image.Rect(mid.X, mid.Y, mid.X+w, mid.Y+h)
		clear := !r.Overlaps(avoid)
		extras, _ := alsoRects(gs, host, step)
		for _, e := range extras {
			clear = clear && !r.Overlaps(e.Inset(-tutorialMargin))
		}
		if clear {
			return r
		}
	}

	// **Scored against the anchor itself, not against the margin around it.** What matters in the
	// fallback is how much of the thing being pointed at is hidden; the margin is a courtesy that
	// decides whether a seat is clean, and counting it would let a seat that merely crowds the
	// anchor lose to one that sits on top of it.
	best, bestArea := image.Rectangle{}, -1
	for _, s := range seats {
		r := image.Rect(s.X, s.Y, s.X+w, s.Y+h)
		if !r.Overlaps(avoid) {
			return r
		}
		hidden := r.Intersect(target)
		if area := hidden.Dx() * hidden.Dy(); bestArea < 0 || area < bestArea {
			best, bestArea = r, area
		}
	}
	return best
}

// draw puts the spotlight down, then the bubble on top of it.
//
// **It must be called last in a scene's Draw**, after everything it is pointing at — the scrim
// dims what is already on the screen, so anything drawn afterwards would sit on top of the
// dimming and read as the one lit thing.
func (t *tutorialOverlay) draw(gs *state.GlobalState, screen *ebiten.Image, host tutorialHost) {
	run := runOf(gs)
	if !run.Active() {
		return
	}
	step, _ := run.Current()

	// **Nothing is pointed at while one of the scene's dialogs is up.** The bubble stays, so what
	// Bob is saying is still readable and Skip is still reachable; what goes is the square and the
	// line, because both would be describing something the player cannot see.
	targets, ok := host.tutorialRects(gs, step.Anchor)
	if ok && len(targets) > 0 && step.Anchor != tutorial.AnchorNone && !host.tutorialCovered(gs) {
		inks := frameInks(step, len(targets))
		framed := make([]bool, len(targets))
		for i := range framed {
			framed[i] = !step.Anchor.NamesCards()
		}

		// **The step's Also frames join the same spotlight**, each framed in its own part's ink, so
		// the scrim is cut once round everything the step points at.
		extras, extraInks := alsoRects(gs, host, step)
		holes := append(append([]image.Rectangle(nil), targets...), extras...)
		holeInks := append(append([]color.RGBA(nil), inks...), extraInks...)
		for range extras {
			framed = append(framed, true)
		}
		t.drawSpotlight(screen, gs, holes, holeInks, framed, step.Lock != tutorial.LockNone)
		for i, r := range extras {
			t.drawLeader(screen, r, extraInks[i])
		}

		// **Under the bubble and over the scrim.** The line leaves the bubble's edge, so nothing
		// of it is hidden either way — but drawing it before the panel is what guarantees that
		// stays true if the bubble ever grows a shadow or a tail of its own.
		// **A line to each framed thing**, since a framed anchor can name controls in different
		// corners and a line to the middle of them points at nothing. A row of cards is one subject
		// and gets one line.
		// **Named cards are separate subjects and get a line each** — they need not sit together,
		// and a line to the middle of them lands on a card the step did not name.
		if step.Anchor.NamesCards() && step.Anchor != tutorial.AnchorNamedCards {
			t.drawLeader(screen, unionOf(targets), tutorialGlow)
		} else {
			for i, r := range targets {
				t.drawLeader(screen, r, inks[i])
			}
		}
	}
	t.drawBubble(gs, screen, step)
}

// drawLeader is the line from the bubble to the thing being pointed at.
//
// **It exists because the bubble moves** *(owner's call, 2026-08-25)*. A panel that takes one of
// six seats depending on what it is avoiding is a panel whose position carries no information, so
// on a screen with a card in each corner and a row of controls along the bottom, "the highlighted
// thing" and "the words about it" can end up at opposite corners with nothing joining them. The
// square says what; the line says *these two go together*.
//
// **Both ends are clipped to their own rectangle's edge**, so the line touches the bubble and the
// square rather than starting inside one and burying its first thirty pixels. The dot at the
// target end is what makes it read as pointing rather than as a stray rule — a bare line meets the
// square at a right angle and looks like part of the frame.
func (t *tutorialOverlay) drawLeader(screen *ebiten.Image, target image.Rectangle, ink color.RGBA) {
	// The same hole the spotlight cuts, so the line lands on the square and not inside it.
	hole := target.Inset(-8)

	from := edgeToward(t.panel, center(hole))
	to := edgeToward(hole, center(t.panel))

	vector.StrokeLine(screen, float32(from.X), float32(from.Y),
		float32(to.X), float32(to.Y), 3, ink, true)
	vector.FillCircle(screen, float32(to.X), float32(to.Y), 6, ink, true)
}

// center is a rectangle's middle.
func center(r image.Rectangle) image.Point {
	return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
}

// edgeToward is where a ray from the center of `r` aimed at `at` crosses `r`'s border.
//
// **Scaled along the ray rather than picked per edge**, which is what keeps the line pointing at
// the right place near a corner: choosing an edge first and then a point on it has to answer for
// the corners, and every answer puts a kink in a line that should be straight.
func edgeToward(r image.Rectangle, at image.Point) image.Point {
	c := center(r)
	dx, dy := at.X-c.X, at.Y-c.Y
	if dx == 0 && dy == 0 {
		return c
	}

	// The fraction of the ray that stays inside the box, per axis; the smaller one is the edge the
	// ray actually leaves through. Doubled denominators avoid an integer half-width rounding the
	// short side of a thin rectangle to zero.
	const scale = 1 << 12
	tx, ty := scale, scale
	if dx != 0 {
		tx = r.Dx() * scale / (2 * ui.Abs(dx))
	}
	if dy != 0 {
		ty = r.Dy() * scale / (2 * ui.Abs(dy))
	}
	tt := tx
	if ty < tt {
		tt = ty
	}
	if tt > scale {
		tt = scale // `at` is inside the box; stop at it rather than overshooting
	}
	return image.Pt(c.X+dx*tt/scale, c.Y+dy*tt/scale)
}

// drawSpotlight is the pointing.
//
// **A card is tinted, not framed** *(owner's call, 2026-09-08)*. An anchor naming cards gets the
// scrim and no rectangle: the cards themselves are drawn in `cards.MarkHighlit`, which is the same
// red. A frame outside a card is a thing on the screen *near* the card, where a tinted card is the
// card answering — and where the anchor names several, a frame each is a lot of loose rectangles
// while a tint each is just the cards. `tutorial.Anchor.NamesCards` is the closed table that says
// which, and the card rows read the same focus list this function is given, so what is lit and what
// is clickable stay the same set.
//
// **A step that locks anything gets the scrim; one that locks nothing gets only the relic.** The
// darkened area is exactly the area that has stopped accepting clicks, so a player never learns
// that a dimmed thing is still clickable.
//
// **The converse is deliberately not held, and it is worth knowing why.** On a fully locked step
// the anchor is left bright and is *not* clickable — the square is naming the subject of the
// sentence rather than inviting a press. What disambiguates is the bubble: a step wanting a click
// says so where its Next button would be ("take a card", "press it"), and a step wanting to be
// read has an actual Next button to press. Dimming the thing being described would be worse, since
// the player would be reading about something they cannot see.
//
// The scrim is rectangles around the holes rather than a full-screen fill with a cut-out,
// because there is no cut-out — Ebitengine would want a mask and a blend mode for that, and a
// handful of `DrawFilledRect` calls are the same picture with none of the machinery.
//
// **Several holes, and the gaps between them are scrimmed too** *(2026-09-08)*. An anchor naming a
// set of cards can have something sitting between two of them — the tutorial's four taught cards sit
// at seats 0, 1, 2 and 4 — and the whole rule this file keeps is that the lit area and the clickable
// area are the same area. Lighting the bounding box would have made the odd card out look like part
// of the lesson, which is how it came to be queued.
//
// **Every gap is closed, on both axes** — see scrimAround — because an anchor can name things in
// different corners of the screen (the opponent's card, two rows of the duelist's, the clock), and
// the lit area has to be the holes and nothing between them.
func (t *tutorialOverlay) drawSpotlight(screen *ebiten.Image, gs *state.GlobalState,
	targets []image.Rectangle, inks []color.RGBA, framed []bool, locked bool) {

	holes := make([]image.Rectangle, 0, len(targets))
	for _, r := range targets {
		holes = append(holes, r.Inset(-8))
	}
	if len(holes) == 0 {
		return
	}

	if locked {
		for _, r := range scrimAround(image.Rect(0, 0, gs.ScreenWidth, gs.ScreenHeight), holes) {
			vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y),
				float32(r.Dx()), float32(r.Dy()), tutorialShade, false)
		}
	}

	for i, hole := range holes {
		if framed[i] {
			vector.StrokeRect(screen, float32(hole.Min.X), float32(hole.Min.Y),
				float32(hole.Dx()), float32(hole.Dy()), 3, inks[i], false)
		}
	}
}

// alsoRects is the rectangles a step's Also frames, and the ink each is framed in: its part's,
// where the text marks it by name, and the glow otherwise. An Also the scene cannot answer for is
// left out, as an anchor is.
func alsoRects(gs *state.GlobalState, host tutorialHost, step tutorial.Step) ([]image.Rectangle, []color.RGBA) {
	byPart := partInks(step)
	var rects []image.Rectangle
	var inks []color.RGBA
	for _, a := range step.Also {
		rs, ok := host.tutorialRects(gs, a)
		if !ok || len(rs) == 0 {
			continue
		}
		ink, marked := byPart[a.String()]
		if !marked {
			ink = tutorialGlow
		}
		rects = append(rects, unionOf(rs))
		inks = append(inks, ink)
	}
	return rects, inks
}

// partInkOrder is the inks a step's marked parts are drawn in, taken in this order. **Borrowed from
// the rest of the game and strictly the tutorial's** *(owner's call)*: the relic pink first, since
// it is no element, then the element inks — each skipped on a step whose own text names that
// element, so a part never wears the color of an element word beside it.
var partInkOrder = []cards.Element{cards.Relic, cards.Ice, cards.Earth, cards.Arcane, cards.Lightning, cards.Fire}

// partInks is the ink each part a step's text marks is drawn in — the phrase and its frame alike.
func partInks(step tutorial.Step) map[string]color.RGBA {
	taken := map[color.RGBA]bool{}
	for _, span := range cards.ElementSpans(step.Plain()) {
		taken[span.Ink] = true
	}
	// **Every part takes an ink, marked or not** *(owner's call)* — the marked ones first, in the
	// order the text names them, so a phrase and its frame agree; then the rest in the anchor's own
	// order, so four panes are four colors whether or not the sentence names each one.
	order := step.MarkedParts()
	for _, part := range step.Parts() {
		if !slices.Contains(order, part) {
			order = append(order, part)
		}
	}
	out := map[string]color.RGBA{}
	next := 0
	for _, part := range order {
		for next < len(partInkOrder) {
			ink := elementInk(partInkOrder[next])
			next++
			if !taken[ink] {
				out[part] = ink
				break
			}
		}
	}
	return out
}

// elementInk is the ink an element's word is written in, which is what a part borrows; the relic
// pink is its border, since no word names it.
func elementInk(e cards.Element) color.RGBA {
	if e == cards.Relic {
		return cards.BorderOf(cards.Relic)
	}
	if spans := cards.ElementSpans(e.String()); len(spans) > 0 {
		return spans[0].Ink
	}
	return tutorialGlow
}

// frameInks is the ink each of an anchor's n rectangles is framed in: its part's, where the step's
// text marks that part, and the glow everywhere else.
func frameInks(step tutorial.Step, n int) []color.RGBA {
	byPart := partInks(step)
	parts := step.Anchor.Parts()
	out := make([]color.RGBA, n)
	for i := range out {
		out[i] = tutorialGlow
		if i < len(parts) {
			if ink, ok := byPart[parts[i]]; ok {
				out[i] = ink
			}
		}
	}
	return out
}

// bubbleLine is one authored line of a step's text as runs: the element words colored as every
// tooltip colors them, and each marked phrase in its part's ink.
func bubbleLine(authored string, inks map[string]color.RGBA) models.TipLine {
	segs, err := tutorial.Segments(authored)
	if err != nil {
		return ui.TipLine(authored)
	}
	var out models.TipLine
	for _, seg := range segs {
		if ink, ok := inks[seg.Part]; ok {
			out = append(out, models.TextSpan{Text: seg.Text, Ink: ink})
			continue
		}
		out = append(out, ui.TipLine(seg.Text)...)
	}
	return out
}

// scrimAround is the screen minus the holes, as rectangles that do not overlap — so the shade is
// laid once everywhere and never twice anywhere, which a translucent fill would show as a darker
// band.
//
// **Cut into vertical slabs at every hole edge**: inside one slab the holes covering it are a set
// of vertical intervals, and the shade is the gaps between them. Overlapping holes — the hand row
// overlaps when it is full — merge rather than producing a negative gap.
func scrimAround(screen image.Rectangle, holes []image.Rectangle) []image.Rectangle {
	xs := []int{screen.Min.X, screen.Max.X}
	for _, h := range holes {
		h = h.Intersect(screen)
		if !h.Empty() {
			xs = append(xs, h.Min.X, h.Max.X)
		}
	}
	sort.Ints(xs)

	var out []image.Rectangle
	for i := 0; i+1 < len(xs); i++ {
		x0, x1 := xs[i], xs[i+1]
		if x1 <= x0 {
			continue
		}
		var cover []image.Rectangle
		for _, h := range holes {
			if h.Min.X <= x0 && h.Max.X >= x1 {
				cover = append(cover, h)
			}
		}
		sort.Slice(cover, func(a, b int) bool { return cover[a].Min.Y < cover[b].Min.Y })
		y := screen.Min.Y
		for _, h := range cover {
			if h.Min.Y > y {
				out = append(out, image.Rect(x0, y, x1, h.Min.Y))
			}
			if h.Max.Y > y {
				y = h.Max.Y
			}
		}
		if y < screen.Max.Y {
			out = append(out, image.Rect(x0, y, x1, screen.Max.Y))
		}
	}
	return out
}

// drawBubble is Bob's card, what he is saying, and the buttons.
func (t *tutorialOverlay) drawBubble(gs *state.GlobalState, screen *ebiten.Image,
	step tutorial.Step) {

	r := t.panel

	// Raised, for the reason the fight log's panel is: it is in front of the game.
	systems.BevelRect(screen, r.Min.X, r.Min.Y, r.Dx(), r.Dy(),
		systems.PaneBevelWidth, tutorialPanel, false)
	vector.StrokeRect(screen, float32(r.Min.X), float32(r.Min.Y),
		float32(r.Dx()), float32(r.Dy()), 2, tutorialGlow, false)

	ui.BlitCard(gs, screen, image.Pt(r.Min.X+tutorialPad, r.Min.Y+tutorialPad),
		guideSpec(gs), cards.RelicStyle)

	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: tutorialTextSize}
	x := r.Min.X + tutorialPad*2 + tutorialCardW
	y := r.Min.Y + tutorialPad + 6

	// **Bob's prose goes through the game's one coloring door and the game's one wrapper**
	// *(owner's call, 2026-09-18)*. He was the only voice that named an element without writing it
	// in that element's color, which is a gap rather than a style: the lesson teaching a new player
	// what fire is was the one place the word was gray. `ui.TipLine` is the same cut every tooltip
	// takes, so the elements, the metals and the forms all arrive coloured with nothing added here.
	//
	// **An authored break still forces a line and can only ever add one** — each is wrapped on its
	// own — so `data/tutorial.json` can shape a paragraph without having to measure one.
	inks := partInks(step)
	for _, authored := range strings.Split(step.Text, "\n") {
		for _, line := range systems.WrapLine(bubbleLine(authored, inks), face, tutorialTextW) {
			systems.DrawLine(screen, line, face, x, y, tutorialText)
			y += tutorialLinePitch
		}
	}

	systems.DrawButton(gs, screen, t.skip)
	if step.Until == tutorial.CondNext {
		systems.DrawButton(gs, screen, t.next)
		return
	}

	// **A step with no Next says what it is waiting for** *(owner's call, 2026-08-25)*. A bubble
	// carrying nothing but Skip reads as a bubble whose other button failed to draw, which is how
	// it was first reported — and the fix is not to add a Next, because a button skipping past the
	// one thing a step exists to teach is the fastest route to a player who has read the tutorial
	// and cannot play the game. So the slot says why it is empty instead.
	//
	// **On a line of its own, centered in the text column, halfway between the prose and the
	// buttons** *(owner's call)*: tucked beside Skip it read as part of the button row and was
	// missed. It is the interface's own lettering, so it is set in the figure glyphs, in capitals.
	//
	// **An action is amber, an outcome is quiet.** "TAKE THEM ALL" and "PRESS IT" are things the
	// player must do and have to stand out as such; "WATCHING" is not an instruction, and in amber it
	// would read as one.
	hint := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: systems.TextMedium}
	buttonsTop := r.Max.Y - tutorialPad - tutorialButtonH
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x+tutorialTextW/2), float64(y+buttonsTop)/2)
	op.PrimaryAlign, op.SecondaryAlign = text.AlignCenter, text.AlignCenter
	ink := tutorialWaiting
	if step.Lock == tutorial.LockToAnchor {
		ink = tutorialAction
	}
	op.ColorScale.ScaleWithColor(ink)
	systems.DrawUI(screen, strings.ToUpper(waitingHint(step)), hint, op)
}

// waitingFor is what the bubble says in place of a Next button: the thing the player has to do
// before Bob moves on.
//
// **A table with an entry per condition and no default arm**, which is the shape `eventDwells`
// uses and for the same reason: a condition added without a line here would silently inherit
// whatever the last arm said, and the failure — a step waiting for something while telling the
// player to do something else — looks exactly like a step that is simply stuck.
// `TestEveryConditionSaysWhatItIsWaitingFor` fails rather than letting one through.
var waitingWords = map[tutorial.Condition]string{
	tutorial.CondNext:         "", // has a button; this is never read
	tutorial.CondCardsQueued:  "take a card",
	tutorial.CondHandEmptied:  "take them all",
	tutorial.CondMatchQueued:  "select all three",
	tutorial.CondDuelPressed:  "duel!",
	tutorial.CondRoundDone:    "watching",
	tutorial.CondShieldBroke:  "watching",
	tutorial.CondPhaseFight:   "back to the journey",
	tutorial.CondPhaseReward:  "win the fight",
	tutorial.CondPhaseShop:    "take your prize",
	tutorial.CondLedgerOpened: "open the ledger",
	tutorial.CondRelicsWorn:   "buy them",
	tutorial.CondDMGBought:    "drink it",
}

func waitingFor(c tutorial.Condition) string { return waitingWords[c] }

// waitingHint is the line for one step: its condition's, except a queue counting more than one card,
// which names the count — a step asking for three named cards does not say "take a card".
//
// **`select all three` on the matching set is a fact about the taught hand**, which holds exactly
// three Slices; TestTheTutorialsFightPlaysAsTaught fails if it stops.
func waitingHint(step tutorial.Step) string {
	if step.Until == tutorial.CondCardsQueued && step.Count > 1 {
		return "select all " + numberWord(step.Count)
	}
	return waitingFor(step.Until)
}

// numberWord is a small count as the tutorial writes it.
func numberWord(n int) string {
	words := []string{"zero", "one", "two", "three", "four", "five"}
	if n >= 0 && n < len(words) {
		return words[n]
	}
	return strconv.Itoa(n)
}

// guideSpec is Bob as a card: his name, his face, and nothing else.
//
// **RelicStyle's shape rather than an opponent's.** He has no life to draw and no shields to
// carry, and a health bar on the character explaining the game would be the single most confusing
// thing on the screen — the player would spend the tutorial waiting to fight him.
//
// `cards.Basic` is the mid gray every non-elemental card borders in, which is what he should be:
// the pink is a relic, and the four colors are things that can be played.
func guideSpec(gs *state.GlobalState) cards.Spec {
	return cards.Spec{
		Name:    "Bob",
		Element: cards.Basic,
		Art:     ui.Artwork(gs, "guide_png"),
		Enabled: true,
	}
}

// opaque is a color at full alpha. The shared panel surface carries the tooltip's transparency,
// and a bubble that covers nothing has no reason to be see-through.
func opaque(c color.RGBA) color.RGBA { c.A = 255; return c }

// buttonRect is a button's footprint, derived from its center exactly as `systems.UpdateButton`
// derives it for hit testing. **Shared rather than written out per anchor**, so a spotlight and
// the click it invites cannot end up describing two different rectangles.
//
// A nil button — one whose scene has not built it yet — is the empty rectangle, which the overlay
// reads as "no anchor here" and responds to by dropping the gate rather than shielding the screen
// around nothing.
func buttonRect(b *models.Button) image.Rectangle {
	if b == nil {
		return image.Rectangle{}
	}
	return image.Rect(
		b.ScreenX-b.Width/2, b.ScreenY-b.Height/2,
		b.ScreenX+b.Width/2, b.ScreenY+b.Height/2,
	)
}

// one is a single rectangle as the set an anchor hands back. **Nearly every anchor is one control
// in one place**, so this is what most of the three scenes' answers look like — the list exists for
// the two anchors that name a set of cards, not because a control ever needs more than one.
func one(r image.Rectangle) []image.Rectangle { return []image.Rectangle{r} }
