package systems

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/models"
)

// The slider's arithmetic, which is what the widget actually is — the drawing is a picture of it.
//
// **This package links Ebitengine**, so on Linux the test binary needs a display even though
// nothing here draws; CI runs the whole test step under `xvfb-run -a`. See CLAUDE.md.

// bar is a slider placed like the settings screen's, without building an image for it: nothing
// below draws, and NewSlider would want a graphics context.
func bar() *models.Slider {
	return &models.Slider{
		Width: 460, Height: 56,
		ScreenX: 640, ScreenY: 400,
		Label: "Music",
	}
}

func TestBothEndsOfTheTravelAreReachable(t *testing.T) {
	// **The failure this exists for**: without the half-knob inset, a value of 1 needs the cursor
	// past the right-hand edge of the control, so a bar that looks full-width can never be turned
	// all the way up.
	s := bar()
	r := SliderRect(s)

	if got := SliderValueAt(s, r.Min.X); got != 0 {
		t.Errorf("the left edge of the control is %v, want 0", got)
	}
	if got := SliderValueAt(s, r.Max.X); got != 1 {
		t.Errorf("the right edge of the control is %v, want 1", got)
	}
}

func TestTheKnobStaysOnTheControlAtBothExtremes(t *testing.T) {
	s := bar()
	r := SliderRect(s)

	for _, v := range []float64{0, 0.5, 1} {
		s.Value = v
		knob := sliderKnob(s)
		if knob.Min.X < r.Min.X || knob.Max.X > r.Max.X {
			t.Errorf("at %v the knob runs from x=%d to x=%d, outside the control's %d..%d",
				v, knob.Min.X, knob.Max.X, r.Min.X, r.Max.X)
		}
	}
}

func TestTheValueRisesLeftToRightAndIsClamped(t *testing.T) {
	s := bar()
	r := SliderRect(s)

	mid := SliderValueAt(s, r.Min.X+r.Dx()/2)
	if mid <= 0 || mid >= 1 {
		t.Errorf("the middle of the control is %v, want something strictly between the ends", mid)
	}

	// Read off the cursor even when it has left the control: a drag that has started belongs to
	// this slider until the button comes up, which is what stops a fast drag leaving the knob.
	if got := SliderValueAt(s, r.Min.X-500); got != 0 {
		t.Errorf("a cursor well left of the control is %v, want 0", got)
	}
	if got := SliderValueAt(s, r.Max.X+500); got != 1 {
		t.Errorf("a cursor well right of the control is %v, want 1", got)
	}
}

func TestTheTrackLeavesRoomForTheLabel(t *testing.T) {
	// A label drawn over its own track is the layout bug this catches. The band above is only
	// reserved when there is something to put in it, so a bare slider still fills its height.
	s := bar()
	if got := sliderTrack(s).Min.Y - SliderRect(s).Min.Y; got < sliderLabelHeight {
		t.Errorf("the track starts %dpx down the control, into the %dpx label band",
			got, sliderLabelHeight)
	}

	s.Label, s.Readout = "", ""
	if got := sliderTrack(s).Max.Y; got != SliderRect(s).Max.Y {
		t.Errorf("an unlabelled slider's track ends at %d, want the control's own bottom %d",
			got, SliderRect(s).Max.Y)
	}
}
