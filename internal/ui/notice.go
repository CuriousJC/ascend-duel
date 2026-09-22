package ui

// **The non-fatal notice: the game admitting something went wrong and carrying on.**
//
// It is the confirm dialog's shape with one answer, which is the achievement toast's shape — and
// that is deliberate rather than a fourth thing *(2026-09-22)*. A confirm is a question, a modal
// panel is a page, and this is neither; what it is the same *size* of thing as is a question, which
// is exactly what confirm.go argues a near-full-screen panel must not be used for. Same scrim, same
// bevelled panel, same stroke.
//
// **It is not a toast, and the difference is the point.** A toast means the player did something
// good and the box is a reward for it. This is the opposite piece of news, so it takes the modal
// red rather than the panel blue, and its eyebrow says PROBLEM rather than ACHIEVEMENT. Sharing the
// geometry and differing in the two things that carry meaning is what keeps the game looking like
// one game while still saying two different things.
//
// **It never raises during a duel.** A box in front of a round in playback is a box that stops a
// fight to talk about a file, and the news it carries — a run that is not being saved — is not news
// the player can act on until the fight is over anyway. So the frame holds it until the run is
// between stations. See internal/game.
//
// **Its queue is internal/crashlog's**, not a field here: the failures are noted from wherever they
// happen, which is several packages away from anything that draws, and a view over that queue is
// better than a second place a problem can be remembered. Same shape as the achievement toast over
// state.EarnedThisSession.

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/crashlog"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The box. Sized to its longest line rather than to a percentage, on confirm.go's argument.
const (
	noticeWidth  = 820
	noticeHeight = 300

	noticeEyebrow     = "PROBLEM"
	noticeEyebrowSize = 16
	noticeEyebrowTop  = 52

	// noticeWhatSize is the failure itself, and noticeWhatTop where it sits. **Set smaller than an
	// achievement's name**, because what is written here is a sentence rather than a title.
	noticeWhatSize = 22
	noticeWhatTop  = 112

	// The reassurance. **Every notice carries one**, because the reason this box exists at all is
	// that the failure did *not* stop the game, and a player shown a red box with no such line
	// reasonably assumes it did.
	noticeCalm     = "The game is still running. This is a note, not a crash."
	noticeCalmSize = 17
	noticeCalmTop  = 160

	noticeButtonWidth  = 300
	noticeButtonHeight = 68
	noticeButtonBottom = 48

	// noticeButtonLabel is the one answer. Not "OK", for the toast's reason: the box is not asking
	// anything, and a dismissal that reads as agreement makes it a question.
	noticeButtonLabel = "GOT IT"
)

var (
	noticePanelFill  = color.RGBA{R: 30, G: 30, B: 38, A: 255}
	noticeEyebrowInk = color.RGBA{R: 216, G: 140, B: 144, A: 255}
	noticeCalmInk    = color.RGBA{R: 198, G: 198, B: 208, A: 255}
)

// ProblemNotice is the frame's bad news. **It holds no queue of its own** — the queue is
// crashlog's, so this is a view over what has gone wrong rather than a second record of it.
type ProblemNotice struct {
	dismiss *models.Button

	// shown is the sentence currently up, so a redraw does not have to re-read the queue and so
	// dismissing drops exactly what was read.
	shown string
}

// Waiting reports whether there is anything to say. **Derived from the queue rather than latched**,
// which is what makes an empty queue and a closed box the same state.
//
// **It does not decide whether the box may be up** — that is the frame's call and depends on where
// the run is standing. See internal/game.
func (n *ProblemNotice) Waiting() bool { return crashlog.Notice() != "" }

// Update runs the one answer.
func (n *ProblemNotice) Update(gs *state.GlobalState) {
	n.shown = crashlog.Notice()
	if n.shown == "" {
		return
	}

	if n.dismiss == nil {
		n.dismiss = models.NewButton(noticeButtonWidth, noticeButtonHeight, noticeButtonLabel, nil)
		n.dismiss.BaseColor = ModalCloseColor
	}

	// Rebound every frame, for confirm.go's reason: one widget serves every notice.
	n.dismiss.OnClick = func() { crashlog.Dismiss() }

	box := noticeRect(gs)
	n.dismiss.ScreenX = box.Min.X + box.Dx()/2
	n.dismiss.ScreenY = box.Max.Y - noticeButtonBottom - noticeButtonHeight/2

	// The frame owns the screen while this is up, on the same terms every other dialog sets it.
	gs.ModalOpen = true
	gs.InputGated = false
	systems.UpdateButton(gs, n.dismiss)
}

// Draw puts the scrim, the box and the words over whatever was underneath.
func (n *ProblemNotice) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	if n.shown == "" || n.dismiss == nil {
		return
	}

	ModalScrim(screen)
	box := noticeRect(gs)

	systems.BevelRect(screen, box.Min.X, box.Min.Y, box.Dx(), box.Dy(),
		systems.PaneBevelWidth, noticePanelFill, false)
	vector.StrokeRect(screen, float32(box.Min.X), float32(box.Min.Y),
		float32(box.Dx()), float32(box.Dy()), 2, ModalCloseColor, false)

	center := float64(box.Min.X + box.Dx()/2)

	line := func(s string, size float64, top int, ink color.Color) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(center, float64(box.Min.Y+top))
		op.PrimaryAlign = text.AlignCenter
		op.SecondaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(ink)
		text.Draw(screen, s, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: size}, op)
	}

	line(noticeEyebrow, noticeEyebrowSize, noticeEyebrowTop, noticeEyebrowInk)
	line(n.shown, noticeWhatSize, noticeWhatTop, color.White)
	line(noticeCalm, noticeCalmSize, noticeCalmTop, noticeCalmInk)

	systems.DrawButton(gs, screen, n.dismiss)
}

// noticeRect is the box: centered, at a fixed size. Same reasoning as confirmRect.
func noticeRect(gs *state.GlobalState) image.Rectangle {
	left := gs.PctX(50) - noticeWidth/2
	top := gs.PctY(50) - noticeHeight/2
	return image.Rect(left, top, left+noticeWidth, top+noticeHeight)
}
