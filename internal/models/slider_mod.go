package models

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Slider is a value between 0 and 1 set by dragging a knob along a track.
//
// **It is the second widget in the game's whole vocabulary, and it exists because the input
// vocabulary already had drag in it.** CLAUDE.md's rule is that a settings value is a row of
// buttons or a slider and never a number typed in; a row of buttons says "one of these three"
// and a bar says "anywhere along here", which is the honest shape for a volume.
//
// It follows Button's split exactly: a plain struct here, behaviour in internal/systems, owned
// by the scene that uses it. Nothing about it knows what it is setting.
type Slider struct {
	// Width and Height are the whole control, track and knob together. ScreenX and ScreenY are
	// its **centre**, the same convention Button uses, so both hit testing and drawing re-derive
	// the top-left from one point.
	Width, Height    int
	ScreenX, ScreenY int

	// Value is where the knob is, 0 at the left end of the travel and 1 at the right.
	//
	// **It is always 0..1 whatever the setting means.** A slider that carried its own minimum and
	// maximum would put a game decision — how slow is too slow — inside a widget shared by
	// everything; the scene maps this onto the range the setting actually has.
	Value float64

	// Label is drawn above the track, left-aligned. Empty draws nothing.
	Label string

	// Readout is drawn above the track, right-aligned: the value in whatever units the setting is
	// read in. The scene formats it, because the widget does not know whether 0.5 is half volume
	// or half speed.
	Readout string

	// BaseColor is the filled part of the track at full strength, following Button's rule that a
	// widget names one colour and its other tones are scaled from it. The zero value means "use
	// the default".
	BaseColor color.RGBA

	// Ink is the colour of the label and the readout. The zero value means "use the default",
	// which is a near-white for a control on a dark ground.
	//
	// **It exists because the game has two grounds.** The combat screen and everything that
	// followed it are painted on cream, where a near-white label is invisible; the title screen
	// is not. A widget that assumed one of them would be unusable on the other, and the repo's
	// colour rule is already that dimming means different things on the two — see
	// systems.ColorToward.
	Ink color.RGBA

	// OnChange fires on every frame the value moves, which is what makes a volume bar audible
	// while it is being dragged rather than only after it is let go.
	OnChange func(float64)

	// OnCommit fires once, when the drag ends. **It is where a setting is written to disk**: a
	// save per frame of a drag is a hundred writes for one decision.
	OnCommit func(float64)

	// Dragging is whether the knob is currently being carried by the cursor. Set by the systems
	// side; a scene reads it to know a drag is in progress and never writes it.
	Dragging bool

	// Hovered is whether the cursor is over the control, so the face can light up the way a
	// button's does.
	Hovered bool

	// Disabled greys the control out and stops it responding, for a setting there is no hardware
	// to apply — a volume bar on a machine whose audio device would not open.
	Disabled bool

	// Image is the cached face, repainted only when something visible changes. Same reasoning as
	// Button.Image: sixty new GPU textures a second to draw a bar that mostly does not move.
	Image   *ebiten.Image
	painted paintedSlider
}

// paintedSlider is what Image currently holds, so the systems side can tell a stale face from a
// current one. Unexported because only DrawSlider writes it and nothing else has any business
// reading it.
type paintedSlider struct {
	valid                   bool
	value                   float64
	label, readout          string
	hovered, dragging, dead bool
	color, ink              color.RGBA
	width, height           int
}

// PaintedIsCurrent reports whether the cached face still matches the slider.
func (s *Slider) PaintedIsCurrent() bool {
	return s.painted == s.snapshot() && s.Image != nil &&
		s.Image.Bounds().Dx() == s.Width && s.Image.Bounds().Dy() == s.Height
}

// RecordPainted stores what was just drawn.
func (s *Slider) RecordPainted() { s.painted = s.snapshot() }

func (s *Slider) snapshot() paintedSlider {
	return paintedSlider{
		valid:    true,
		value:    s.Value,
		label:    s.Label,
		readout:  s.Readout,
		hovered:  s.Hovered,
		dragging: s.Dragging,
		dead:     s.Disabled,
		color:    s.BaseColor,
		ink:      s.Ink,
		width:    s.Width,
		height:   s.Height,
	}
}

// NewSlider makes a slider at a starting value.
func NewSlider(width, height int, label string, value float64) *Slider {
	return &Slider{
		Width:  width,
		Height: height,
		Label:  label,
		Value:  value,
		Image:  ebiten.NewImage(width, height),
	}
}
