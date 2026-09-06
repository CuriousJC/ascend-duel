package screens

// **The achievement toast: the game telling the player they did something, and waiting to be
// dismissed.**
//
// It is the confirm dialog's shape with one answer instead of two, and that is deliberate rather
// than a fourth shape *(owner's call, 2026-09-06)*. A confirm is a question, a modal panel is a
// page, and this is neither — but it is the same *size* of thing as a question: a few words in a
// small centred box, which is exactly what confirm.go argues a near-full-screen panel must not be
// used for. Same scrim, same bevelled panel, same pink stroke.
//
// **It is clicked out of rather than timed out.** A notice that fades has to be looked at while it
// is up, and the one moment an achievement lands is the moment the player is watching something
// else — a killing blow, a card flying to the discard. A dismissal is also the only input verb the
// game has for "I have read this".
//
// **It is chrome, not a scene's dialog**, driven by `internal/game` on exactly the terms the ledger
// panel is: an achievement can land during a duel, on the post-battle screen, or on the transition
// between them, and no scene owns any of that. While it is up the active scene is not updated at
// all, which freezes pacing and — like every other dialog in the game — cannot change an outcome,
// because ResolveRound decided the round before a frame of it was drawn.
//
// **The queue is drained one at a time.** A five-element turn earns Prism, Elementalist and
// Spectrum together, and three boxes in a row is the honest reading of three achievements: a single
// box listing them would have to pick one name to put at the top.

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/achieve"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The box. **Sized to its longest line rather than to a percentage**, on confirm.go's argument: a
// notice that grew with the screen would be a page again.
const (
	toastWidth  = 820
	toastHeight = 340

	// The eyebrow is the word that says what kind of thing this is, over the name that says which.
	// Small and quiet: the player reads the name, and the eyebrow is what tells them why a box has
	// appeared at all.
	toastEyebrow     = "ACHIEVEMENT"
	toastEyebrowSize = 16
	toastEyebrowTop  = 48

	toastNameSize = 38
	toastNameTop  = 92

	// The commentary. **Every line at once**, so this is a stack rather than one string — see
	// data.AchievementData.Said for why there is no picking.
	toastSaidSize = 19
	toastSaidTop  = 156
	toastSaidStep = 30

	toastButtonWidth  = 300
	toastButtonHeight = 68
	toastButtonBottom = 52

	// toastButtonLabel is the one answer. **Not "OK"** — the box is not asking anything, and a
	// dismissal that reads as agreement makes it a question.
	toastButtonLabel = "GOT IT"
)

// toastEyebrowInk is the quiet word over the name, and toastSaidInk the commentary under it. Both
// are pulled toward the panel rather than given a hue: the colour wheel belongs to the elements,
// and this box has nothing elemental to say.
var (
	toastPanelFill  = color.RGBA{R: 30, G: 30, B: 38, A: 255}
	toastEyebrowInk = color.RGBA{R: 150, G: 150, B: 164, A: 255}
	toastSaidInk    = color.RGBA{R: 198, G: 198, B: 208, A: 255}
)

// AchievementToast is the frame's notice. **It holds no queue of its own** — the queue is
// `state.EarnedThisSession`, written by whoever awarded the achievement — so this is a view over
// state rather than a second place an achievement can be remembered.
type AchievementToast struct {
	dismiss *models.Button

	// shown is the key currently up, so a redraw does not have to re-read the queue's head to know
	// what it is drawing, and so dismissing pops exactly what was shown.
	shown string
}

// IsOpen reports whether there is anything to show. **Derived from the queue rather than latched**,
// which is what makes an empty queue and a closed toast the same state — there is no way to leave
// one flag saying yes while the other says no.
func (t *AchievementToast) IsOpen(gs *state.GlobalState) bool {
	return gs != nil && len(gs.EarnedThisSession) > 0
}

// Update runs the one answer.
func (t *AchievementToast) Update(gs *state.GlobalState) {
	if !t.IsOpen(gs) {
		t.shown = ""
		return
	}
	t.shown = gs.EarnedThisSession[0]

	if t.dismiss == nil {
		t.dismiss = models.NewButton(toastButtonWidth, toastButtonHeight, toastButtonLabel, nil)
		t.dismiss.BaseColor = apBarColor
	}

	// **Rebound every frame**, for confirm.go's reason: one widget serves every achievement, so a
	// callback wired once at build time would pop the queue on behalf of a box shown ten minutes
	// ago.
	t.dismiss.OnClick = func() { t.pop(gs) }

	box := toastRect(gs)
	t.dismiss.ScreenX = box.Min.X + box.Dx()/2
	t.dismiss.ScreenY = box.Max.Y - toastButtonBottom - toastButtonHeight/2

	// The frame owns the screen while this is up, so nothing behind it may be reached. Set on the
	// same terms every other dialog sets it.
	gs.ModalOpen = true
	gs.InputGated = false
	systems.UpdateButton(gs, t.dismiss)
}

// pop takes the shown achievement off the queue.
func (t *AchievementToast) pop(gs *state.GlobalState) {
	if len(gs.EarnedThisSession) == 0 {
		return
	}
	gs.EarnedThisSession = gs.EarnedThisSession[1:]
	t.shown = ""
}

// Draw puts the scrim, the box and the words over whatever was underneath.
func (t *AchievementToast) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	if !t.IsOpen(gs) || t.dismiss == nil {
		return
	}

	// **An unknown key still draws a box.** A profile written by a build that had an achievement
	// this one does not is exactly the case profile.LoadProfile is built around, and a notice that
	// silently showed nothing would leave the queue's head un-poppable and the game stuck behind an
	// invisible dialog.
	a, ok := achieve.Loaded().Find(t.shown)
	if !ok {
		a = achieve.Achievement{Name: "SOMETHING"}
	}

	modalScrim(screen)
	box := toastRect(gs)

	systems.BevelRect(screen, box.Min.X, box.Min.Y, box.Dx(), box.Dy(),
		systems.PaneBevelWidth, toastPanelFill, false)
	vector.StrokeRect(screen, float32(box.Min.X), float32(box.Min.Y),
		float32(box.Dx()), float32(box.Dy()), 2, apBarColor, false)

	centre := float64(box.Min.X + box.Dx()/2)

	eyebrow := &text.DrawOptions{}
	eyebrow.GeoM.Translate(centre, float64(box.Min.Y+toastEyebrowTop))
	eyebrow.PrimaryAlign = text.AlignCenter
	eyebrow.SecondaryAlign = text.AlignCenter
	eyebrow.ColorScale.ScaleWithColor(toastEyebrowInk)
	text.Draw(screen, toastEyebrow,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: toastEyebrowSize}, eyebrow)

	name := &text.DrawOptions{}
	name.GeoM.Translate(centre, float64(box.Min.Y+toastNameTop))
	name.PrimaryAlign = text.AlignCenter
	name.SecondaryAlign = text.AlignCenter
	text.Draw(screen, a.Name,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: toastNameSize}, name)

	for i, line := range a.Said {
		said := &text.DrawOptions{}
		said.GeoM.Translate(centre, float64(box.Min.Y+toastSaidTop+i*toastSaidStep))
		said.PrimaryAlign = text.AlignCenter
		said.SecondaryAlign = text.AlignCenter
		said.ColorScale.ScaleWithColor(toastSaidInk)
		text.Draw(screen, line,
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: toastSaidSize}, said)
	}

	systems.DrawButton(gs, screen, t.dismiss)
}

// toastRect is the box: centred, at a fixed size. Same reasoning as confirmRect.
func toastRect(gs *state.GlobalState) image.Rectangle {
	left := gs.PctX(50) - toastWidth/2
	top := gs.PctY(50) - toastHeight/2
	return image.Rect(left, top, left+toastWidth, top+toastHeight)
}
