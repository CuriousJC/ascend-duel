package systems

import (
	"image/color"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/models"
)

// **A bevel's two edges have to differ from the face and from each other**, or the widget is a
// flat rectangle with extra drawing in it. Asserted on the colours rather than on a rendered face,
// so the check needs no graphics context — and the four fills are the cases the derivation can
// fail on: a saturated colour has nowhere to climb by scaling, and the two ends of the range have
// little room left in one direction.
func TestABevelIsLitOnOneSideAndShadowedOnTheOther(t *testing.T) {
	fills := []color.RGBA{
		{R: 95, G: 95, B: 40, A: 255},
		{R: 198, G: 46, B: 46, A: 255},
		{R: 20, G: 20, B: 24, A: 255},
		{R: 240, G: 240, B: 246, A: 255},
	}

	for _, fill := range fills {
		light, shade := BevelEdges(fill)

		if sum(light) <= sum(fill) {
			t.Errorf("fill %v: the lit edge %v is no brighter than the face", fill, light)
		}
		if sum(shade) >= sum(fill) {
			t.Errorf("fill %v: the shadowed edge %v is no darker than the face", fill, shade)
		}
		if light.A != fill.A || shade.A != fill.A {
			t.Errorf("fill %v: a bevel changed the alpha — %d and %d against %d",
				fill, light.A, shade.A, fill.A)
		}
	}
}

// **Pressed and latched are the sunken states; hover and rest are not.** A button the cursor is
// merely resting on has not moved, and conflating the two is what left press and hover competing
// for the single signal the brightness ramp had.
func TestOnlyTheInStatesAreSunken(t *testing.T) {
	want := map[models.ButtonState]bool{
		models.ButtonStateNormal:  false,
		models.ButtonStateHovered: false,
		models.ButtonStatePressed: true,
		models.ButtonStateLatched: true,
	}

	for state, sunken := range want {
		if got := buttonSunken(&models.Button{State: state}); got != sunken {
			t.Errorf("state %v draws sunken=%v, want %v", state, got, sunken)
		}
	}
}

func sum(c color.RGBA) int { return int(c.R) + int(c.G) + int(c.B) }
