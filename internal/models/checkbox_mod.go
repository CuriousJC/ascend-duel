package models

// Checkbox is an on-or-off setting, ticked by a click.
//
// It follows Button's split exactly: a plain struct here, behavior in internal/systems, owned by
// the scene that uses it. **It knows nothing about what it switches**: the scene says whether it is
// ticked and what a click does, and the box draws that.
type Checkbox struct {
	// Size is the box's side in pixels, the drawn box rather than the art's canvas around it.
	// ScreenX and ScreenY are its **center**, the convention every widget here uses.
	Size             int
	ScreenX, ScreenY int

	// Checked is whether the box carries its mark. **The scene owns it**, and may set it every
	// frame from whatever the box stands for, so a value changed some other way is still drawn.
	Checked bool

	// OnToggle fires on a click, with the state the click asks for. The box does not flip Checked
	// itself: the scene decides what a click means and sets Checked to match.
	OnToggle func(checked bool)

	// Disabled draws the box in its disabled stone and stops it responding.
	Disabled bool

	// Hovered and PressedInside are set by the systems side, as Button's are: whether the cursor is
	// over the box, and whether the press that is down began on it.
	Hovered, PressedInside bool

	// TickAge counts frames since the mark last arrived, which is what pops it in large and settles
	// it. Set by the systems side.
	TickAge int

	// shown is the Checked the last frame drew, so the systems side can tell a mark arriving.
	shown bool
}

// MarkArrived reports whether the box became ticked since the last call, and records the state.
// Only the systems side calls it.
func (c *Checkbox) MarkArrived() bool {
	arrived := c.Checked && !c.shown
	c.shown = c.Checked
	return arrived
}
