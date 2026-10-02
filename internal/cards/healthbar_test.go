package cards

import (
	"image"
	"testing"
)

// TestTheMarksFallOnTheThirds. A bar cut in three is marked at a third and two thirds of its own
// width, and nothing marks a bar that asks for fewer than two parts.
func TestTheMarksFallOnTheThirds(t *testing.T) {
	bar := image.Rect(15, 250, 185, 274)
	got := HealthMarkXs(bar, 3)
	if len(got) != 2 || got[0] != 15+170/3 || got[1] != 15+170*2/3 {
		t.Errorf("a 170-wide bar cut in three was marked at %v, want [%d %d]", got, 15+170/3, 15+170*2/3)
	}
	for _, parts := range []int{0, 1} {
		if got := HealthMarkXs(bar, parts); len(got) != 0 {
			t.Errorf("a bar in %d parts was marked at %v", parts, got)
		}
	}
}

// TestOnlyAMarkedBarDrawsMarks. The marks are the duelist's, so a bar that does not ask for them
// must be the same picture it always was, and one that does must differ on each line it names.
func TestOnlyAMarkedBarDrawsMarks(t *testing.T) {
	plain := render(t, duelist(30, 120), DuelistStyle)
	spec := duelist(30, 120)
	spec.LifeMarks = 3
	marked := render(t, spec, DuelistStyle)

	bar := HealthBarRect(DuelistStyle)
	for _, x := range HealthMarkXs(bar, 3) {
		differs := false
		for y := bar.Min.Y; y < bar.Max.Y && !differs; y++ {
			differs = plain.RGBAAt(x, y) != marked.RGBAAt(x, y)
		}
		if !differs {
			t.Errorf("the line at x=%d draws nothing on a bar cut in three", x)
		}
	}
}
