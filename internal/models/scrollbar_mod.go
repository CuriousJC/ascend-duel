package models

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Scrollbar is a dragged thumb on a vertical track, saying which slice of a long list is being
// looked at.
//
// **It is the third widget in the game's vocabulary, and it is a drag for the reason the slider
// is one** *(owner's call, 2026-09-02)*: the input vocabulary is clicks, drags and hover, and a
// wheel would be a fourth verb rather than a widget. A panel that cannot reach its own earlier
// rows is a panel that keeps a record nobody can read — which is what the fight log was until
// the ledger, and what the deck overlay's `+N more not shown` still is.
//
// **It counts rows, not pixels.** Offset is the index of the first row on show, so the panel
// pouring rows into itself does not have to convert a scroll position into a line number and
// cannot land half a line off. The pixel arithmetic is entirely inside internal/systems.
//
// It follows Button and Slider exactly: a plain struct here, behaviour in internal/systems,
// owned by whoever draws the list. Nothing about it knows what the rows say.
type Scrollbar struct {
	// Width and Height are the whole track. ScreenX and ScreenY are its **centre**, the
	// convention Button and Slider both use.
	Width, Height    int
	ScreenX, ScreenY int

	// Total is how many rows there are and Visible is how many fit. **Both are set by the panel
	// every frame**, because either can change while the bar is on screen — a fight ends, a
	// collapsed record is opened — and a bar carrying a stale total is a bar that scrolls past
	// the end of a list or refuses to reach it.
	Total, Visible int

	// Offset is the first row on show, clamped into 0..Total-Visible by the systems side. The
	// panel reads it to know where to start drawing and writes it to jump somewhere.
	Offset int

	// BaseColor is the thumb at full strength, following the one-colour rule. The zero value
	// means "use the default".
	BaseColor color.RGBA

	// Ground is what the track is dimmed toward, for a bar on a light surface.
	//
	// **It exists for the reason Slider.Ink does**: the game has two grounds, and
	// systems.ColorAtStrength scales toward black, which reads as louder rather than quieter on
	// cream. The zero value means the dark default.
	Ground color.RGBA

	// Dragging is whether the thumb is being carried by the cursor. Written by the systems side.
	Dragging bool

	// Hovered is whether the cursor is over the track.
	Hovered bool

	// grab is where inside the thumb the drag started, in pixels from its top edge, so a thumb
	// picked up by its bottom does not jump under the cursor.
	grab int

	// Image is the cached face, repainted only when something visible changed — Button's and
	// Slider's reasoning, and the same hazard: this one is drawn over a scrim every frame it is
	// up.
	Image   *ebiten.Image
	painted paintedScrollbar
}

type paintedScrollbar struct {
	valid                  bool
	total, visible, offset int
	hovered, dragging      bool
	color, ground          color.RGBA
	width, height          int
}

// PaintedIsCurrent reports whether the cached face still matches the bar.
func (s *Scrollbar) PaintedIsCurrent() bool {
	return s.painted == s.snapshot() && s.Image != nil &&
		s.Image.Bounds().Dx() == s.Width && s.Image.Bounds().Dy() == s.Height
}

// RecordPainted stores what was just drawn.
func (s *Scrollbar) RecordPainted() { s.painted = s.snapshot() }

// Grab and SetGrab are how the systems side carries the offset into the thumb across frames.
// Unexported state with exported accessors, because nothing outside internal/systems has any
// business setting it and the two packages cannot share a field otherwise.
func (s *Scrollbar) Grab() int     { return s.grab }
func (s *Scrollbar) SetGrab(n int) { s.grab = n }

func (s *Scrollbar) snapshot() paintedScrollbar {
	return paintedScrollbar{
		valid:    true,
		total:    s.Total,
		visible:  s.Visible,
		offset:   s.Offset,
		hovered:  s.Hovered,
		dragging: s.Dragging,
		color:    s.BaseColor,
		ground:   s.Ground,
		width:    s.Width,
		height:   s.Height,
	}
}

// NewScrollbar makes a bar of a given size, at the top of an empty list.
func NewScrollbar(width, height int) *Scrollbar {
	return &Scrollbar{
		Width:  width,
		Height: height,
		Image:  ebiten.NewImage(width, height),
	}
}
