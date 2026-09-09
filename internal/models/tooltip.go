package models

import (
	"image"
	"image/color"
)

// TipLine is one line of a tooltip, in the runs it is drawn as. A line with one run is a line in
// one colour, which is what most of them are.
type TipLine []TextRun

// Text is the line's words with the colouring dropped — what a caller keying on a tooltip's
// identity compares, and what a test asserts against.
func (l TipLine) Text() string {
	out := ""
	for _, run := range l {
		out += run.Text
	}
	return out
}

// TextRun is one stretch of a line drawn in its own colour.
//
// **A zero-alpha Ink means the panel's own ink**, which is the convention every optional colour in
// this codebase follows — so a caller that never thinks about colour builds a line of one run and
// nothing changes.
type TextRun struct {
	Text string
	Ink  color.RGBA
}

// Tooltip is the panel that explains whatever the cursor is resting on.
//
// **It is a widget rather than a screen's own drawing** for the reason `Button` is: four scenes
// want it, they want it to behave identically, and the thing that varies between them is only the
// words. Behaviour lives in `internal/systems`, exactly like every other widget here.
//
// **A scene points it at something every tick, and a tick with nothing pointed hides it.** That is
// the same shape `state.ModalOpen` takes — cleared by the frame, re-asserted by whoever still means
// it — and it is what keeps a tooltip from surviving the thing it was describing. A scene never has
// to remember to hide one.
//
// **Hover is the desktop gesture and long press is the touch one** *(owner's call, 2026-08-21)*,
// which reverses the split MECHANICS.md recorded when hover was rejected: hover now *explains*.
// Nothing here knows which input asked, so the day a press can ask, `Point` is what it calls.
type Tooltip struct {
	// Title is the bold first line — the name of the thing under the cursor — and Lines are what it
	// has to say. Short strings: the panel does not wrap, so the caller decides where a line breaks.
	// That is deliberate, because every line in a tooltip here is an authored phrase or one term of
	// an arithmetic, and both know their own shape better than a wrapper would.
	//
	// **A line is runs rather than a string** *(owner's call, 2026-09-08)*, so a word naming an
	// element can be drawn in that element's colour where the rest of the line is not. This package
	// knows nothing about why a run has a colour — the caller decides that, exactly as it decides
	// where a line breaks.
	//
	// **The title is runs too, as of 2026-09-09** *(owner's call)*. It was a plain string, which
	// made it the one place in the game an element word was not written in its element's colour —
	// and a card's title is `FIRE JAB`, so it was the place that mattered most. The two now take
	// the same type and go through the same drawing.
	Title TipLine
	Lines []TipLine

	// Anchor is the thing being explained. The panel is placed beside it rather than under the
	// cursor, so the tooltip does not sit on top of the card it is about and does not jitter as the
	// hand moves inside one card.
	Anchor image.Rectangle

	// Dwell is how long the cursor has rested, in ticks, and DwellTicks is how long it has to before
	// the panel appears. **A delay rather than an instant panel**, because a cursor crossing a row of
	// eight cards would otherwise strobe eight tooltips on its way somewhere else.
	Dwell      int
	DwellTicks int

	// pointed is set by Point and cleared by the systems update every tick. Unexported because it is
	// the handshake between the two, and a scene reading it would be reading its own last frame.
	pointed bool

	// key is what the tooltip is currently about, so that moving from one card to the next restarts
	// the dwell rather than continuing it. **The title plus the anchor**: two cards can carry the
	// same name in different seats, and one card can change what it says without moving.
	key string
}

// Point aims the tooltip at something, and is called every tick the cursor is still on it. It
// restarts the dwell when the thing under the cursor changes.
//
// **The key is the title's *words*, not its runs.** Recolouring a title without changing what it
// says is not a different thing under the cursor, and restarting the dwell on it would make a
// tooltip flicker at whatever recoloured it.
func (t *Tooltip) Point(at image.Rectangle, title TipLine, lines []TipLine) {
	key := title.Text() + at.String()
	if key != t.key {
		t.key, t.Dwell = key, 0
	}
	t.Title, t.Lines, t.Anchor, t.pointed = title, lines, at, true
}

// Pointed reports whether a scene aimed this tooltip on the tick just gone. For the systems update,
// which is the only thing that should ask.
func (t *Tooltip) Pointed() bool { return t.pointed }

// Release clears the tick's aim. The systems update calls it after reading Pointed, so the next tick
// starts from nothing pointed.
func (t *Tooltip) Release() { t.pointed = false }

// Showing reports whether the panel has waited long enough to be drawn.
func (t *Tooltip) Showing() bool {
	return t.key != "" && t.Dwell >= t.DwellTicks && (len(t.Title) > 0 || len(t.Lines) > 0)
}

// Forget hides it immediately, for a scene that has just done something the tooltip was describing —
// a card bought out from under the cursor, say.
func (t *Tooltip) Forget() {
	t.key, t.Dwell, t.Title, t.Lines = "", 0, nil, nil
}
