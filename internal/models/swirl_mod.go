package models

// Swirl is a round control drawn as a picture turning in place: the portal screen's way through a
// portal. **It has no label**, so the turning is what says it can be pressed — it turns faster under
// the cursor and gives a little when pressed.
//
// It is the Button split applied to a different shape: this struct holds the state, and
// `systems.UpdateSwirl` / `systems.DrawSwirl` do the work. The picture is handed to the draw rather
// than held here, so the struct stays free of anything a test would need a graphics context for.
type Swirl struct {
	// ScreenX and ScreenY are the center, the convention every button in the game uses.
	ScreenX int
	ScreenY int

	// Radius is how big it is drawn. **The click lands on the inner part only** — see
	// SwirlHitShare — because the picture fades to nothing at its rim, and a press on the faded
	// edge is a press on what looks like the backdrop.
	Radius int

	State         ButtonState
	PressedInside bool
	OnClick       func()

	// Angle is how far it has turned, in radians, and Spin how fast it is turning now, in radians a
	// tick. Spin eases toward the speed its state asks for rather than jumping to it, so the cursor
	// arriving reads as the swirl being stirred rather than switched.
	Angle float64
	Spin  float64

	// InnerAngle is the counter-turning layer's own angle. **It is kept rather than derived from
	// Angle**: Angle wraps at a whole turn, which the outer layer cannot show, but a fraction of it
	// jumps when it wraps — a derived inner layer snaps once a turn.
	InnerAngle float64
}

// SwirlHitShare is how much of the drawn radius answers a click.
const SwirlHitShare = 0.8
