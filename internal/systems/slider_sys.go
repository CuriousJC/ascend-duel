package systems

// The behaviour half of models.Slider: dragging a knob along a track, and drawing the result.
//
// **A slider is a drag, and drag is already in the game's input vocabulary** — the action box and
// the ring row are built on it. What is new here is that the thing being dragged never leaves its
// track, so there is no drop target and no way to fail: wherever the cursor goes while the button
// is held, the knob follows along one axis and the value follows the knob.
//
// **The cursor is what is dragged with, not the knob.** Once a drag has started the value is read
// off the cursor's x even when it has wandered off the control, which is what stops a fast drag
// leaving the knob behind. It is the same reason UpdateButton fires on the release rather than the
// press: the player is allowed to be imprecise.

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Slider geometry. The control is a labelled row: a line of text along the top and the track
// under it, with the knob standing on the track.
const (
	// sliderTrackHeight is how thick the groove is. Thin enough to read as a track rather than as
	// a bar, thick enough for the two-pixel pane bevel that sinks it to be visible.
	sliderTrackHeight = 14

	// sliderKnobWidth is how wide the thing being dragged is. **It is what bounds the travel**:
	// the knob's centre runs from half a knob in on the left to half a knob in on the right, so a
	// value of 0 and a value of 1 both draw a knob that is fully on the control.
	sliderKnobWidth = 22

	// sliderKnobOverhang is how far the knob stands proud of the groove at each end, so it reads
	// as something sitting on the track rather than something filling it.
	sliderKnobOverhang = 6

	// sliderLabelHeight is the band above the track that the label and the readout sit in.
	sliderLabelHeight = 26

	// sliderTextSize is the point size of both strings.
	sliderTextSize = 18
)

// defaultSliderColor is the filled part of the track for a slider that names no colour. The same
// slate the chrome uses: this is the program rather than the fight.
var defaultSliderColor = color.RGBA{R: 92, G: 96, B: 108, A: 255}

// sliderGrooveColor is the empty part of the track. Deliberately not a scaled version of the
// slider's own colour — the groove is the hole the knob runs in, and it reads as one surface
// whatever the fill in front of it happens to be.
var sliderGrooveColor = color.RGBA{R: 46, G: 48, B: 56, A: 255}

// sliderInkColor is the label and the readout for a slider that names no ink: a near-white, for a
// control on a dark ground. See models.Slider.Ink for why it is not the only answer.
var sliderInkColor = color.RGBA{R: 232, G: 236, B: 242, A: 255}

// How bright the knob and the filled track draw. Same ramp as a button: resting has somewhere to
// climb to, and a drag in progress is the full colour.
const (
	sliderRestStrength  = 65
	sliderHoverStrength = 82
	sliderDragStrength  = 100
)

// SliderRect is where a slider sits, derived from its centre exactly as a button's is.
func SliderRect(s *models.Slider) image.Rectangle {
	left := s.ScreenX - s.Width/2
	top := s.ScreenY - s.Height/2
	return image.Rect(left, top, left+s.Width, top+s.Height)
}

// sliderTrack is the groove inside the control: the full width, below the label band.
func sliderTrack(s *models.Slider) image.Rectangle {
	r := SliderRect(s)
	top := r.Max.Y - sliderTrackHeight
	if s.Label != "" || s.Readout != "" {
		top = r.Min.Y + sliderLabelHeight + (r.Dy()-sliderLabelHeight-sliderTrackHeight)/2
	}
	return image.Rect(r.Min.X, top, r.Max.X, top+sliderTrackHeight)
}

// sliderKnob is where the knob stands for the current value.
func sliderKnob(s *models.Slider) image.Rectangle {
	track := sliderTrack(s)
	lo, hi := sliderTravel(s)
	centre := lo + int(float64(hi-lo)*s.Value+0.5)
	return image.Rect(
		centre-sliderKnobWidth/2, track.Min.Y-sliderKnobOverhang,
		centre+sliderKnobWidth/2, track.Max.Y+sliderKnobOverhang,
	)
}

// sliderTravel is the two screen x values the knob's centre runs between.
//
// **The track is inset by half a knob at each end**, which is what makes both extremes reachable:
// without the inset a value of 1 would need the cursor half a knob past the control's right edge.
func sliderTravel(s *models.Slider) (lo, hi int) {
	track := sliderTrack(s)
	return track.Min.X + sliderKnobWidth/2, track.Max.X - sliderKnobWidth/2
}

// UpdateSlider runs one frame of a slider: hit testing, dragging, and the two callbacks.
//
// **A press anywhere on the control jumps the knob there and starts a drag.** Requiring the press
// to land on the knob itself would make a 22-pixel target of a control whose whole point is being
// easy to move, and jumping to the click is what every volume bar does.
func UpdateSlider(gs *state.GlobalState, s *models.Slider) {
	if s.Disabled {
		s.Dragging, s.Hovered = false, false
		return
	}

	// The grab band is the whole control rather than just the groove, so the label above the
	// track is a live part of the target and a click that lands a few pixels high still works.
	over := image.Pt(gs.MouseX, gs.MouseY).In(SliderRect(s)) && gs.CursorAllowed()
	s.Hovered = over

	if over && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		s.Dragging = true
	}

	if s.Dragging {
		// Read off the cursor even when it has left the control: a drag that has started belongs
		// to this slider until the button comes up.
		setSliderValue(s, SliderValueAt(s, gs.MouseX))
	}

	if s.Dragging && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		s.Dragging = false
		if s.OnCommit != nil {
			s.OnCommit(s.Value)
		}
	}
}

// SliderValueAt converts a screen x into a value, clamped to the travel. Exported so a test can
// ask where a click would land without driving a cursor.
func SliderValueAt(s *models.Slider, x int) float64 {
	lo, hi := sliderTravel(s)
	if hi <= lo {
		return 0
	}
	v := float64(x-lo) / float64(hi-lo)
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}

// setSliderValue moves the knob and reports the move, but only if it actually moved. A drag that
// holds still would otherwise fire OnChange sixty times a second with the same number.
func setSliderValue(s *models.Slider, v float64) {
	if v == s.Value {
		return
	}
	s.Value = v
	if s.OnChange != nil {
		s.OnChange(v)
	}
}

// DrawSlider blits the slider's cached face, repainting it first if anything visible changed.
func DrawSlider(gs *state.GlobalState, screen *ebiten.Image, s *models.Slider) {
	if !s.PaintedIsCurrent() {
		paintSlider(gs, s)
	}
	r := SliderRect(s)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	screen.DrawImage(s.Image, op)
}

// paintSlider renders the whole control into its own image, in the control's own coordinates.
func paintSlider(gs *state.GlobalState, s *models.Slider) {
	if s.Image == nil ||
		s.Image.Bounds().Dx() != s.Width || s.Image.Bounds().Dy() != s.Height {
		s.Image = ebiten.NewImage(s.Width, s.Height)
	}
	s.Image.Clear()

	origin := SliderRect(s).Min
	track := sliderTrack(s).Sub(origin)
	knob := sliderKnob(s).Sub(origin)

	// **The groove is sunken and the knob is raised**, which is the geometry saying what the
	// control is before any colour does: a hole with something standing in it. Same split
	// BevelFace makes for a pressed button.
	BevelRect(s.Image, track.Min.X, track.Min.Y, track.Dx(), track.Dy(),
		PaneBevelWidth, sliderGrooveColor, true)

	// The filled part stops at the knob's centre, so the fill and the knob read as one travelled
	// distance rather than as two things that nearly line up.
	fill := sliderFill(s)
	if width := knob.Min.X + knob.Dx()/2 - track.Min.X; width > 0 {
		vector.DrawFilledRect(s.Image,
			float32(track.Min.X), float32(track.Min.Y),
			float32(width), float32(track.Dy()), fill, false)
	}

	if s.Disabled {
		// Flat, with no light on it at all, exactly as a disabled button is drawn — unavailable
		// first, itself second.
		vector.DrawFilledRect(s.Image,
			float32(knob.Min.X), float32(knob.Min.Y),
			float32(knob.Dx()), float32(knob.Dy()), disabledButtonColor, false)
	} else {
		// **Sunken while dragging**, the same way a pressed button is: the knob is being held
		// down, and brightness alone cannot say that on a ramp whose bright end is already hover.
		BevelRect(s.Image, knob.Min.X, knob.Min.Y, knob.Dx(), knob.Dy(),
			BevelWidth, fill, s.Dragging)
	}

	paintSliderText(gs, s)
	s.RecordPainted()
}

// sliderFill is the colour of the filled track and the knob at the current state.
func sliderFill(s *models.Slider) color.RGBA {
	full := s.BaseColor
	if full.A == 0 {
		full = defaultSliderColor
	}
	switch {
	case s.Dragging:
		return ColorAtStrength(full, sliderDragStrength)
	case s.Hovered:
		return ColorAtStrength(full, sliderHoverStrength)
	default:
		return ColorAtStrength(full, sliderRestStrength)
	}
}

// paintSliderText draws the label at the left of the band above the track and the readout at the
// right of it, which is the arrangement that keeps a column of sliders aligned on both edges
// however long either string is.
func paintSliderText(gs *state.GlobalState, s *models.Slider) {
	if s.Label == "" && s.Readout == "" {
		return
	}
	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: sliderTextSize}
	ink := s.Ink
	if ink.A == 0 {
		ink = sliderInkColor
	}
	if s.Disabled {
		ink = ColorAtStrength(ink, 45)
	}

	if s.Label != "" {
		op := &text.DrawOptions{}
		op.GeoM.Translate(0, float64(sliderLabelHeight)/2)
		op.SecondaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(ink)
		text.Draw(s.Image, s.Label, face, op)
	}
	if s.Readout != "" {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(s.Width), float64(sliderLabelHeight)/2)
		op.PrimaryAlign = text.AlignEnd
		op.SecondaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(ink)
		text.Draw(s.Image, s.Readout, face, op)
	}
}
