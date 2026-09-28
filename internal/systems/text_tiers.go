package systems

// The three sizes every line of reading text is set at.

// **Every line of prose is one of three sizes, and a new one takes a tier rather than a figure** —
// the button heights' rule, applied to text. A screen wanting a line a little bigger takes the next
// tier and moves its layout; three sizes a point apart read as three typefaces that nearly match.
//
// These are point sizes of the interface font today. They are also the three sizes the prose glyph
// sheet is reduced to, so moving a line onto the glyphs is a change to how it is drawn and never to
// how big it is.
//
// Headings, titles and the loud figures are not prose and are not tiered here: they are the display
// glyphs' business.
const (
	TextSmall  = 16 // notes, captions, eyebrows, the small print under a control
	TextMedium = 20 // a dialog's body, narration, the tutorial's bubble, a toast's line
	TextLarge  = 26 // the tooltip's lines, the reward screen's prose, a panel's row names
)
