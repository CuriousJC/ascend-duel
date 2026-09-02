package systems

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/models"
)

func testBar(total, visible int) *models.Scrollbar {
	s := models.NewScrollbar(18, 600)
	s.ScreenX, s.ScreenY = 100, 400 // centre, so the track runs y=100..700
	s.Total, s.Visible = total, visible
	return s
}

// **Both ends of the list have to be reachable by dragging**, which is the scrollbar's whole job:
// a bar that cannot land on the last row is a panel whose last row does not exist. The slider has
// the same test for the same reason — see TestBothEndsOfTheTravelAreReachable.
func TestBothEndsOfTheListAreReachable(t *testing.T) {
	s := testBar(500, 40)
	r := ScrollbarRect(s)

	if got := ScrollOffsetAt(s, r.Min.Y-50); got != 0 {
		t.Errorf("dragged above the track the offset is %d, want 0", got)
	}
	if got, want := ScrollOffsetAt(s, r.Max.Y+50), s.Total-s.Visible; got != want {
		t.Errorf("dragged below the track the offset is %d, want %d", got, want)
	}
}

// A list that fits needs no bar, and offering one would be a control that cannot move.
func TestAListThatFitsHasNoThumb(t *testing.T) {
	s := testBar(10, 40)
	if ScrollbarNeeded(s) {
		t.Error("a list shorter than the panel wants a scrollbar")
	}
	if !scrollThumb(s).Empty() {
		t.Error("a list shorter than the panel draws a thumb")
	}
}

// The thumb never leaves its track, at either end, and never shrinks below the size a cursor can
// catch. A thumb a pixel and a half tall is what a very long list computes without the floor.
func TestTheThumbStaysOnTheTrackAndStaysCatchable(t *testing.T) {
	s := testBar(4000, 30)
	r := ScrollbarRect(s)

	for _, offset := range []int{0, 1, 2000, s.Total - s.Visible} {
		s.Offset = offset
		thumb := scrollThumb(s)
		if thumb.Dy() < scrollThumbMin {
			t.Errorf("at offset %d the thumb is %dpx tall, under the %dpx floor",
				offset, thumb.Dy(), scrollThumbMin)
		}
		if thumb.Min.Y < r.Min.Y || thumb.Max.Y > r.Max.Y {
			t.Errorf("at offset %d the thumb %v is off the track %v", offset, thumb, r)
		}
	}
}

// An offset past the end is brought back rather than left: the list under a bar grows and shrinks
// while it is on screen — a fight ends, a folded record is opened — and a stale offset would draw
// past the end of it.
func TestAnOffsetPastTheEndIsBroughtBack(t *testing.T) {
	s := testBar(100, 40)
	s.Offset = 900
	if got := ClampScroll(s); got != 60 {
		t.Errorf("an offset past the end clamped to %d, want 60", got)
	}
	s.Offset = -5
	if got := ClampScroll(s); got != 0 {
		t.Errorf("a negative offset clamped to %d, want 0", got)
	}
}
