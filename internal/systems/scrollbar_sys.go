package systems

// The behaviour half of models.Scrollbar: dragging a thumb down a track, and drawing it.
//
// **Everything here is a drag, which is already in the input vocabulary.** There is no wheel and
// no keyboard — see CLAUDE.md — so the thumb is the whole of how a long list is reached, and it
// therefore has to be forgiving: a press anywhere on the track jumps the thumb to the cursor, and
// a drag that wanders off the control keeps its grip until the button comes up. Both are the
// slider's rules, for the slider's reason.
//
// **It works in rows and converts to pixels here.** A panel says how many rows it has and how
// many fit; this decides how tall the thumb is and where it sits. The panel never sees a pixel of
// the bar, which is what stops a list and its scrollbar disagreeing about where the list is.

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	// scrollThumbMin is the shortest the thumb may draw. A thousand-row list would otherwise
	// compute a thumb a pixel and a half tall, which is a control the cursor cannot catch.
	scrollThumbMin = 28

	// scrollTrackInset is how far the thumb sits inside the track at each side, so it reads as
	// something running in a groove rather than as the groove itself.
	scrollTrackInset = 2
)

// The thumb's ramp, matching the button's and the slider's: resting has somewhere to climb to and
// a live drag is the full colour.
const (
	scrollRestStrength  = 62
	scrollHoverStrength = 80
	scrollDragStrength  = 100
)

// defaultScrollColor is the thumb for a bar that names no colour: the chrome's slate, because a
// scrollbar is the program rather than the fight.
var defaultScrollColor = color.RGBA{R: 92, G: 96, B: 108, A: 255}

// defaultScrollGround is what the track is toned toward for a bar that names no ground.
var defaultScrollGround = color.RGBA{R: 46, G: 48, B: 56, A: 255}

// ScrollbarRect is where a bar sits, derived from its centre exactly as a button's is.
func ScrollbarRect(s *models.Scrollbar) image.Rectangle {
	left := s.ScreenX - s.Width/2
	top := s.ScreenY - s.Height/2
	return image.Rect(left, top, left+s.Width, top+s.Height)
}

// ScrollbarNeeded reports whether there is anything to scroll. A bar with everything on show is
// drawn as an empty groove rather than as a thumb filling the whole track, so the player is never
// offered a control that cannot move.
func ScrollbarNeeded(s *models.Scrollbar) bool { return s.Total > s.Visible && s.Visible > 0 }

// ClampScroll brings the offset into range and returns it. Called by UpdateScrollbar every frame,
// and exported because a panel that has just grown or shrunk its list wants the same arithmetic
// without waiting a frame to be told.
func ClampScroll(s *models.Scrollbar) int {
	max := s.Total - s.Visible
	if max < 0 {
		max = 0
	}
	if s.Offset > max {
		s.Offset = max
	}
	if s.Offset < 0 {
		s.Offset = 0
	}
	return s.Offset
}

// scrollThumb is where the thumb stands for the current offset.
func scrollThumb(s *models.Scrollbar) image.Rectangle {
	r := ScrollbarRect(s)
	if !ScrollbarNeeded(s) {
		return image.Rectangle{}
	}

	height := r.Dy() * s.Visible / s.Total
	if height < scrollThumbMin {
		height = scrollThumbMin
	}
	if height > r.Dy() {
		height = r.Dy()
	}

	travel := r.Dy() - height
	top := r.Min.Y
	if max := s.Total - s.Visible; max > 0 {
		top += travel * s.Offset / max
	}
	return image.Rect(r.Min.X+scrollTrackInset, top, r.Max.X-scrollTrackInset, top+height)
}

// UpdateScrollbar runs one frame: hit testing, dragging, and the offset that comes out of it.
func UpdateScrollbar(gs *state.GlobalState, s *models.Scrollbar) {
	ClampScroll(s)

	if !ScrollbarNeeded(s) {
		s.Dragging, s.Hovered = false, false
		return
	}

	r := ScrollbarRect(s)
	at := image.Pt(gs.MouseX, gs.MouseY)
	s.Hovered = at.In(r) && gs.CursorAllowed()

	if s.Hovered && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		thumb := scrollThumb(s)
		s.Dragging = true

		// **A press on the thumb keeps its grip; a press on the track jumps.** Picking the thumb
		// up where it was grabbed is what stops it snapping its centre under the cursor, and a
		// press anywhere else is the player pointing at where they want to be.
		if at.In(thumb) {
			s.SetGrab(at.Y - thumb.Min.Y)
		} else {
			s.SetGrab(thumb.Dy() / 2)
			s.Offset = ScrollOffsetAt(s, at.Y)
		}
	}

	if s.Dragging {
		// Read off the cursor even when it has left the track: a drag belongs to this bar until
		// the button comes up, exactly as the slider's does.
		s.Offset = ScrollOffsetAt(s, gs.MouseY)
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		s.Dragging = false
	}

	ClampScroll(s)
}

// ScrollOffsetAt converts a screen y — the cursor's, with the grab already accounted for — into a
// row offset, clamped. Exported so a test can ask where a drag would land without a cursor.
func ScrollOffsetAt(s *models.Scrollbar, y int) int {
	r := ScrollbarRect(s)
	thumb := scrollThumb(s)
	travel := r.Dy() - thumb.Dy()
	max := s.Total - s.Visible
	if travel <= 0 || max <= 0 {
		return 0
	}

	top := y - s.Grab() - r.Min.Y
	switch {
	case top < 0:
		top = 0
	case top > travel:
		top = travel
	}

	// **Rounded rather than truncated**, so the last row is reachable by dragging the thumb to
	// the bottom of the track rather than one row short of it.
	return (top*max + travel/2) / travel
}

// DrawScrollbar paints the groove and the thumb.
//
// **The whole control is one cached image**, repainted only when the offset, the list or the
// cursor state changed — Button's and Slider's rule, and it matters more here: this is drawn over
// a scrim every frame a panel is open.
func DrawScrollbar(gs *state.GlobalState, screen *ebiten.Image, s *models.Scrollbar) {
	r := ScrollbarRect(s)

	if !s.PaintedIsCurrent() {
		if s.Image == nil || s.Image.Bounds().Dx() != s.Width || s.Image.Bounds().Dy() != s.Height {
			s.Image = ebiten.NewImage(s.Width, s.Height)
		}
		s.Image.Clear()

		ground := s.Ground
		if ground.A == 0 {
			ground = defaultScrollGround
		}
		base := s.BaseColor
		if base.A == 0 {
			base = defaultScrollColor
		}

		// The groove is sunken: this is a track cut into the panel, not a panel floating on it.
		// See systems.BevelRect and the sunken-is-a-meaning note in CLAUDE.md.
		BevelRect(s.Image, 0, 0, s.Width, s.Height, PaneBevelWidth, ground, true)

		if thumb := scrollThumb(s); !thumb.Empty() {
			strength := scrollRestStrength
			switch {
			case s.Dragging:
				strength = scrollDragStrength
			case s.Hovered:
				strength = scrollHoverStrength
			}
			// **Toward the groove rather than scaled toward black**, so the resting thumb is
			// quieter than the live one on a light panel as well as a dark one. See
			// systems.ColorToward.
			face := ColorToward(base, ground, 100-strength)
			BevelRect(s.Image, thumb.Min.X-r.Min.X, thumb.Min.Y-r.Min.Y,
				thumb.Dx(), thumb.Dy(), BevelWidth, face, false)
		}

		s.RecordPainted()
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	screen.DrawImage(s.Image, op)
}
