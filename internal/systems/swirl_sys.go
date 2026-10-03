package systems

import (
	"image"
	"math"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// How a swirl turns, in radians a tick at the game's sixty a second.
const (
	swirlRestSpin  = 0.010
	swirlHoverSpin = 0.032

	// swirlSpinEase is how much of the gap to the wanted speed is closed each tick.
	swirlSpinEase = 0.08

	// The second layer: the same picture smaller, fainter and turning the other way, which is what
	// stops one rigid picture reading as a disc on a spindle.
	swirlInnerScale = 0.72
	swirlInnerAlpha = 0.55
	// SwirlInnerSpin is the inner layer's speed against the outer one's: slower, and the other way.
	SwirlInnerSpin = -0.6

	// What hover and press do to the face: brighter under the cursor, a little smaller when held.
	swirlHoverLight = 1.15
	swirlPressScale = 0.93
)

// UpdateSwirl runs one tick of a swirl: the press and release, its state, and its turning.
//
// **The input rules are UpdateButton's** — a click is a press and a release inside, the tutorial's
// gate is folded into the hover so a gated swirl rests rather than freezing, and a disabled one
// neither hovers nor fires. Only the hit test differs, being a circle.
func UpdateSwirl(gs *state.GlobalState, s *models.Swirl) {
	if s.State != models.ButtonStateDisabled {
		over := SwirlCovers(s, image.Pt(gs.MouseX, gs.MouseY)) && gs.CursorAllowed()
		if over && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			s.PressedInside = true
		}
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			if over && s.PressedInside && s.OnClick != nil {
				s.OnClick()
			}
			s.PressedInside = false
		}
		switch {
		case over && s.PressedInside:
			s.State = models.ButtonStatePressed
		case over:
			s.State = models.ButtonStateHovered
		default:
			s.State = models.ButtonStateNormal
		}
	}

	want := swirlRestSpin
	if s.State == models.ButtonStateHovered || s.State == models.ButtonStatePressed {
		want = swirlHoverSpin
	}
	s.Spin += (want - s.Spin) * swirlSpinEase
	s.Angle = math.Mod(s.Angle+s.Spin, 2*math.Pi)
	s.InnerAngle = math.Mod(s.InnerAngle+s.Spin*SwirlInnerSpin, 2*math.Pi)
}

// SwirlTurnTicks is how long one whole turn takes at rest. **It is not on the game-speed setting**:
// the turning is the swirl idling, not a beat of anything the player is waiting on.
func SwirlTurnTicks() int {
	turn := 2 * math.Pi / swirlRestSpin
	return int(turn + 0.5)
}

// SwirlCovers reports whether a point is on the part of the swirl that answers a click.
func SwirlCovers(s *models.Swirl, at image.Point) bool {
	dx, dy := float64(at.X-s.ScreenX), float64(at.Y-s.ScreenY)
	r := float64(s.Radius) * models.SwirlHitShare
	return dx*dx+dy*dy <= r*r
}

// SwirlBounds is the square the swirl is drawn in.
func SwirlBounds(s *models.Swirl) image.Rectangle {
	return image.Rect(s.ScreenX-s.Radius, s.ScreenY-s.Radius, s.ScreenX+s.Radius, s.ScreenY+s.Radius)
}

// DrawSwirl draws the swirl as two layers of one picture turning against each other. A nil picture
// draws nothing.
func DrawSwirl(screen *ebiten.Image, s *models.Swirl, picture *ebiten.Image) {
	if picture == nil {
		return
	}
	size := float64(2 * s.Radius)
	light := 1.0
	switch s.State {
	case models.ButtonStateHovered:
		light = swirlHoverLight
	case models.ButtonStatePressed:
		light = swirlHoverLight
		size *= swirlPressScale
	}
	drawSwirlLayer(screen, picture, s, size, s.Angle, light, 1)
	drawSwirlLayer(screen, picture, s, size*swirlInnerScale, s.InnerAngle, light, swirlInnerAlpha)
}

func drawSwirlLayer(screen, picture *ebiten.Image, s *models.Swirl, size, angle, light, alpha float64) {
	pw, ph := float64(picture.Bounds().Dx()), float64(picture.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-pw/2, -ph/2)
	op.GeoM.Rotate(angle)
	op.GeoM.Scale(size/pw, size/ph)
	op.GeoM.Translate(float64(s.ScreenX), float64(s.ScreenY))
	op.ColorScale.Scale(float32(light*alpha), float32(light*alpha), float32(light*alpha), float32(alpha))
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(picture, op)
}
