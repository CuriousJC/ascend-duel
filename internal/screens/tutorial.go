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
	"strings"

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
// behaviour for the frame or two either side of a scene change.
type tutorialHost interface {
	// tutorialFacts is what this scene can say about the run right now. See tutorial.Facts for
	// why the traffic goes this way rather than as events.
	tutorialFacts(gs *state.GlobalState) tutorial.Facts

	// tutorialRect is where this scene draws the thing an anchor names, and whether it knows the
	// anchor at all.
	tutorialRect(gs *state.GlobalState, a tutorial.Anchor) (image.Rectangle, bool)

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
	// Bob is drawn at RingStyle's 162x224 — a card with a name and a picture and nothing else,
	// which is exactly what he is. EnemyStyle would give him a health bar.
	tutorialCardW = 162
	tutorialCardH = 224

	tutorialPad      = 20
	tutorialTextW    = 400
	tutorialTextSize = 20

	// tutorialLinePitch is the text size plus the gap that keeps two lines of Kubasta from
	// touching. Prose rather than the cards' clipped register, so this is the one place in the
	// game setting a paragraph.
	tutorialLinePitch = 27

	tutorialButtonW   = 110
	tutorialButtonH   = 40
	tutorialButtonGap = 12

	tutorialPanelW = tutorialPad*3 + tutorialCardW + tutorialTextW
	tutorialPanelH = tutorialPad*2 + tutorialCardH

	// tutorialMargin is how far the bubble keeps off the screen edges, and off the thing it is
	// pointing at.
	tutorialMargin = 24
)

// tutorialInk is the bubble's colours. It is a dark panel over a cream table, matching the fight
// log and the deck overlay rather than the cards — Bob is chrome, not something in play.
var (
	tutorialPanel = color.RGBA{R: 30, G: 30, B: 38, A: 255}
	tutorialText  = color.RGBA{R: 236, G: 232, B: 226, A: 255}

	// tutorialWaiting is the line standing where a Next button would be. **Dimmer than the prose
	// it sits under**, because it is a state and not something Bob is saying.
	tutorialWaiting = color.RGBA{R: 150, G: 146, B: 140, A: 255}

	// tutorialGlow is the square drawn around whatever is being pointed at, and the leader line
	// running to it from the bubble. **Red** *(owner's call, 2026-08-25)*.
	//
	// **It is the one place red is not a control**, and that is worth knowing rather than
	// discovering: `modalCloseColor` is the dialog X and CLAUDE.md records red as belonging to it
	// alone, so that a red thing on screen always means "this closes something". A highlight does
	// not dilute the *click* meaning — there is nothing here to press — but the colour is no longer
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

	nextPressed bool
	skipPressed bool

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

	t.panel = t.place(gs, host, step)
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
		gs.InputGated = false
		return
	}

	run.Update(host.tutorialFacts(gs), t.nextPressed)

	// **The gate is dropped the moment the script ends**, here rather than being left for the
	// next frame's early return. A tutorial that finished on the frame it also set a focus would
	// leave the screen shielded around a rectangle nothing is drawing any more.
	if !run.Active() {
		gs.InputGated = false
	}
}

// focus is the rectangle input is being held to, and whether it is being held at all.
//
// **An anchor the current scene cannot answer for drops the gate rather than closing the whole
// screen.** That happens for a frame either side of a scene change, and the alternative — a
// shield around an empty rectangle — is a game the player cannot click at all.
func (t *tutorialOverlay) focus(gs *state.GlobalState, host tutorialHost) (image.Rectangle, bool) {
	run := runOf(gs)
	anchor, lock := run.Gate()
	if lock == tutorial.LockNone {
		return image.Rectangle{}, false
	}

	// **Locked with no hole in it.** An empty focus rectangle contains no point, so
	// `InputAllowed` refuses everything — which is exactly right for a step whose only request is
	// that it be read. Bob's own two buttons are worked with the shield down and stay live; see
	// update.
	if lock == tutorial.LockAll {
		return image.Rectangle{}, true
	}
	// **A covered anchor drops the gate**, for the reason an unknown one does: shielding the screen
	// around a rectangle the player cannot see leaves them with one legal click they have no way to
	// find. The dialog's own X is then the only thing to press, which is what it is for.
	if host.tutorialCovered(gs) {
		return image.Rectangle{}, false
	}
	r, ok := host.tutorialRect(gs, anchor)
	if !ok || r.Empty() {
		return image.Rectangle{}, false
	}
	return r, true
}

// build makes the two buttons on first use. The Next button is crimson like DUEL! — it is the
// same "and on with it" slot — and Skip is the quiet grey the sort column uses, because leaving
// the tutorial should be findable without being the loudest thing in the bubble.
func (t *tutorialOverlay) build() {
	if t.next == nil {
		t.next = models.NewButton(tutorialButtonW, tutorialButtonH, "Next",
			func() { t.nextPressed = true })
		t.next.BaseColor = color.RGBA{R: 220, G: 20, B: 60, A: 255}
		t.next.TextSize = 18
	}
	if t.skip == nil {
		t.skip = models.NewButton(tutorialButtonW, tutorialButtonH, "Skip",
			func() { t.skipPressed = true })
		t.skip.BaseColor = sortButtonColor
		t.skip.TextSize = 18
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
// The candidates are ordered so the common case is stable: bottom-centre first, because most of
// what a lesson points at is a card or a corner control, then the four corners. A step whose
// anchor covers all five gets the last one anyway — with the anchor spotlit through the scrim, a
// bubble overlapping it is legible where no bubble at all would be a lesson with no words.
func (t *tutorialOverlay) place(gs *state.GlobalState, host tutorialHost,
	step tutorial.Step) image.Rectangle {

	w, h := tutorialPanelW, tutorialPanelH
	left, right := tutorialMargin, gs.ScreenWidth-tutorialMargin-w
	top, bottom := tutorialMargin, gs.ScreenHeight-tutorialMargin-h
	middle := (gs.ScreenWidth - w) / 2

	// **Top-centre is second, ahead of the corners** *(owner's call, 2026-08-25)*. Most of what a
	// step points at during a duel spans the screen — the hand, the AP bar, the band the blow is
	// added up in — so the first seat is out and the fallback used to be a top corner, which is
	// where the two fighter cards and their life bars are. The middle of the top row is the ring
	// pane, which is the least costly thing on this screen to cover.
	//
	// The dead centre stays last, because it is over the table: it is where a seat lands only when
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

	target, pointing := host.tutorialRect(gs, step.Anchor)
	if step.Anchor == tutorial.AnchorNone || !pointing {
		// Nothing to avoid: an opening or closing line belongs in the middle of the screen.
		return image.Rect(middle, (gs.ScreenHeight-h)/2, middle+w, (gs.ScreenHeight-h)/2+h)
	}

	// Kept off the anchor by the same margin it keeps off the screen edge, so a bubble does not
	// end up touching the thing it is pointing at.
	avoid := target.Inset(-tutorialMargin)
	for _, s := range seats {
		r := image.Rect(s.X, s.Y, s.X+w, s.Y+h)
		if !r.Overlaps(avoid) {
			return r
		}
	}
	last := seats[len(seats)-1]
	return image.Rect(last.X, last.Y, last.X+w, last.Y+h)
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
	target, ok := host.tutorialRect(gs, step.Anchor)
	if ok && step.Anchor != tutorial.AnchorNone && !host.tutorialCovered(gs) {
		t.drawSpotlight(screen, gs, target, step.Lock != tutorial.LockNone)

		// **Under the bubble and over the scrim.** The line leaves the bubble's edge, so nothing
		// of it is hidden either way — but drawing it before the panel is what guarantees that
		// stays true if the bubble ever grows a shadow or a tail of its own.
		t.drawLeader(screen, target)
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
func (t *tutorialOverlay) drawLeader(screen *ebiten.Image, target image.Rectangle) {
	// The same hole the spotlight cuts, so the line lands on the square and not inside it.
	hole := target.Inset(-8)

	from := edgeToward(t.panel, center(hole))
	to := edgeToward(hole, center(t.panel))

	vector.StrokeLine(screen, float32(from.X), float32(from.Y),
		float32(to.X), float32(to.Y), 3, tutorialGlow, true)
	vector.DrawFilledCircle(screen, float32(to.X), float32(to.Y), 6, tutorialGlow, true)
}

// center is a rectangle's middle.
func center(r image.Rectangle) image.Point {
	return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
}

// edgeToward is where a ray from the centre of `r` aimed at `at` crosses `r`'s border.
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
		tx = r.Dx() * scale / (2 * abs(dx))
	}
	if dy != 0 {
		ty = r.Dy() * scale / (2 * abs(dy))
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
// **A step that locks anything gets the scrim; one that locks nothing gets only the ring.** The
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
// The scrim is four rectangles around the hole rather than a full-screen fill with a cut-out,
// because there is no cut-out — Ebitengine would want a mask and a blend mode for that, and four
// `DrawFilledRect` calls are the same picture with none of the machinery.
func (t *tutorialOverlay) drawSpotlight(screen *ebiten.Image, gs *state.GlobalState,
	target image.Rectangle, locked bool) {

	hole := target.Inset(-8)

	if locked {
		w, h := float32(gs.ScreenWidth), float32(gs.ScreenHeight)
		top, bot := float32(hole.Min.Y), float32(hole.Max.Y)
		l, r := float32(hole.Min.X), float32(hole.Max.X)

		vector.DrawFilledRect(screen, 0, 0, w, top, tutorialShade, false)
		vector.DrawFilledRect(screen, 0, bot, w, h-bot, tutorialShade, false)
		vector.DrawFilledRect(screen, 0, top, l, bot-top, tutorialShade, false)
		vector.DrawFilledRect(screen, r, top, w-r, bot-top, tutorialShade, false)
	}

	vector.StrokeRect(screen, float32(hole.Min.X), float32(hole.Min.Y),
		float32(hole.Dx()), float32(hole.Dy()), 3, tutorialGlow, false)
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

	blitCard(gs, screen, image.Pt(r.Min.X+tutorialPad, r.Min.Y+tutorialPad),
		guideSpec(gs), cards.RingStyle)

	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: tutorialTextSize}
	x := r.Min.X + tutorialPad*2 + tutorialCardW
	y := r.Min.Y + tutorialPad + 6

	for _, line := range wrapTutorialText(face, step.Text, tutorialTextW) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		op.ColorScale.ScaleWithColor(tutorialText)
		text.Draw(screen, line, face, op)
		y += tutorialLinePitch
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
	// **Left of Skip, not where Next would be.** Skip slides right into the empty slot when there
	// is no Next — see placeButtons — so a hint pinned to the panel's edge was drawn underneath it,
	// which is how it first shipped: "take them all" and "Skip" in the same pixels.
	//
	// It is measured off the button rather than off the panel, so the two cannot drift apart.
	hint := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 17}
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(t.skip.ScreenX-tutorialButtonW/2-tutorialButtonGap),
		float64(r.Max.Y-tutorialPad-tutorialButtonH/2-9))
	op.PrimaryAlign = text.AlignEnd
	op.ColorScale.ScaleWithColor(tutorialWaiting)
	text.Draw(screen, waitingFor(step.Until), hint, op)
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
	tutorial.CondNext:        "", // has a button; this is never read
	tutorial.CondCardsQueued: "take a card",
	tutorial.CondHandEmptied: "take them all",
	tutorial.CondDuelPressed: "press it",
	tutorial.CondRoundDone:   "watching",
	tutorial.CondPhaseFight:  "back to the tower",
	tutorial.CondPhaseReward: "win the fight",
	tutorial.CondPhaseShop:   "take your prize",
}

func waitingFor(c tutorial.Condition) string { return waitingWords[c] }

// guideSpec is Bob as a card: his name, his face, and nothing else.
//
// **RingStyle's shape rather than an opponent's.** He has no life to draw and no statuses to
// carry, and a health bar on the character explaining the game would be the single most confusing
// thing on the screen — the player would spend the tutorial waiting to fight him.
//
// `cards.Basic` is the mid grey every non-elemental card borders in, which is what he should be:
// the pink is a ring, and the four colours are things that can be played.
func guideSpec(gs *state.GlobalState) cards.Spec {
	return cards.Spec{
		Name:    "Bob",
		Element: cards.Basic,
		Art:     artwork(gs, "guide_png"),
		Enabled: true,
	}
}

// wrapTutorialText breaks a paragraph to the column, honouring an authored `\n` the way the cards
// do — see `cards.WrapText`, whose rule this follows: a break can only ever add a line, since an
// authored line too wide for the column still wraps.
func wrapTutorialText(face *text.GoTextFace, s string, width int) []string {
	var out []string
	for _, authored := range strings.Split(s, "\n") {
		words := strings.Fields(authored)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := words[0]
		for _, w := range words[1:] {
			try := line + " " + w
			if adv, _ := text.Measure(try, face, 0); int(adv) > width {
				out = append(out, line)
				line = w
				continue
			}
			line = try
		}
		out = append(out, line)
	}
	return out
}

// buttonRect is a button's footprint, derived from its centre exactly as `systems.UpdateButton`
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
