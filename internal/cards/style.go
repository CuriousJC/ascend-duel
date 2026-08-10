package cards

// Style is a card's geometry at one size.
//
// **A glyph cannot be made smaller.** systems.GlyphSize is 64 and CardGlyphScale is 1:
// the art is authored at exactly the size it is displayed, and a fractional scale drops
// pixels out of a rim that is one pixel thick. GlyphScale must stay a whole number, and 1
// is already the floor. That makes 64 pixels the hard floor on a card that shows a glyph
// at all — which is why Mini shows none.
type Style struct {
	Width, Height int

	// CornerRadius and BorderWidth are the shape. Named here rather than written inline
	// at the draw site so both are tunable in one place, which is the whole reason the
	// contact sheet is worth having.
	CornerRadius int
	BorderWidth  int

	// What this size is big enough to show. A Mini card is 59 pixels wide and cannot
	// hold a 64-pixel glyph or legible text at any size, so it shows neither — see Mini.
	ShowName     bool
	ShowCategory bool
	ShowDamage   bool

	TextLeft int
	NameTop  int
	NameSize float64

	// NameCentered centres the name across the card instead of starting it at TextLeft.
	// Rings use it: with no glyph column down the left there is nothing for a
	// left-aligned name to line up with, and it reads as having slipped off centre.
	NameCentered bool

	// The category glyph, above the cost stack. It replaced the category *word*: a
	// picture where a line of text used to be.
	//
	// **It is not 64, and it is not one number either** — the drawn sword and shield are
	// 32 and the generated book is 22. systems.SizeOf is the authority and the layout must
	// ask per glyph rather than assume; see systems/glyphs.go for why the two halves of
	// the set are sized differently.
	CategoryGlyphTop int

	// The cost dashes, hamburger-style, below the category glyph.
	//
	// **The left column is a stack now, and everything in it is load-bearing.** Category
	// glyph, then dashes, then the damage badge, one under the other down the same 64
	// pixels of width. A cost above four grows the stack downward into the damage badge;
	// costs run 1..4 today, and TestLeftColumnDoesNotCollide fails rather than letting
	// them overlap.
	DashLeft   int
	DashTop    int
	DashWidth  int
	DashHeight int
	DashGap    int

	// The damage badge: the sword and the figure beside it, at the bottom of the column.
	GlyphScale     int
	GlyphInset     int
	GlyphColumnTop int
	GlyphNumberGap int
	NumberSize     float64

	// ArtTop and ArtInset frame Spec.Art, used by rings. The art is scaled to fit the
	// box they describe and centred in it.
	ArtTop   int
	ArtInset int
	ArtMaxH  int
}

// Hand is the card as the hand draws it, and the size every constant here is written
// for. 180x264 is roughly a playing card's proportions.
//
// The face reads: the name centred across the top, and a left column starting in the
// corner beside it — category glyph, cost dashes, damage badge.
//
//	 12  category glyph     12..44    (32px at the largest, top-left corner)
//	 14  name               centred
//	 48  cost dashes        48..95    (four at 8 on a 5 gap)
//	100  damage badge      100..164
//	258  inside of the bottom border
//
// **The dashes moved down when the drawn glyphs arrived**, from 42 to 48: Sherman's sword
// and shield are 32 pixels where the generated ones were 22, and the ten pixels had to come
// from somewhere. The damage badge did not move — four dashes now end at y=95 against its
// top at y=100, which is five pixels of slack rather than the fifteen there used to be.
//
// **The name is centred and the glyph is in the corner beside it, not under it.** Those
// two go together: a left-aligned name would sit directly on top of the glyph now that
// the glyph has moved up into the corner, and centring it is what clears the space. The
// name is centred on the *card*, not on the room left over beside the glyph, so a long
// enough name would still reach back into it — TestNameClearsTheCategoryGlyph checks
// every concept in the deck against that.
//
// **Only one 64-pixel badge is left.** Cost became dashes and the category became a
// 22-pixel glyph, so a column that was two full badges deep is now a small glyph, a stack
// of bars and one badge, all finished by y=164. The ~94 pixels below are deliberately
// unfilled — that is where a long-press description or a status line goes.
var Hand = Style{
	Width: 180, Height: 264,

	CornerRadius: 12,
	BorderWidth:  6,

	ShowName:     true,
	ShowCategory: true,
	ShowDamage:   true,

	TextLeft:     12,
	NameTop:      14,
	NameSize:     20,
	NameCentered: true,

	CategoryGlyphTop: 12,

	DashLeft:   12,
	DashTop:    48,
	DashWidth:  13,
	DashHeight: 8,
	DashGap:    5,

	GlyphScale:     1,
	GlyphInset:     12,
	GlyphColumnTop: 100,
	GlyphNumberGap: 10,
	NumberSize:     26,
}

// Mini is the deck overlay's card: half the hand's size, kept to the same proportions.
//
// **It shows the name now, and that was the last thing missing.** The overlay's rows
// group by element and the glyph and dashes give phase and cost, so the only fact a card
// was not stating was which concept it is. At 14pt the longest name in the deck —
// "Riposte" — measures 35 pixels against 78 usable, so it was never a question of room;
// it was a question of the cards being overlapped so tightly that only 29 pixels showed.
// Widening the row (see deckStackPitch) is what made the space real.
//
// The one thing still absent is the damage badge, and that is forced: a 64-pixel sword
// under a name, a glyph and four dashes does not fit in 132 pixels of height. Damage is a
// function of the concept, so a named card implies it.
//
// The reading order matches Hand deliberately — name centred across the top, then the
// glyph, then the dashes under it — so the two sizes are the same card and not two
// designs.
var Mini = Style{
	Width: 90, Height: 132,

	CornerRadius: 6,
	BorderWidth:  3,

	ShowName:     true,
	ShowCategory: true,
	ShowDamage:   false,

	TextLeft:     8,
	NameTop:      8,
	NameSize:     14,
	NameCentered: true,

	GlyphScale:       1,
	GlyphInset:       8,
	CategoryGlyphTop: 36,

	// 72 rather than 66, for the same reason Hand's moved: the drawn glyphs are 32 pixels
	// and end at y=68. A mini card carries a 32px glyph without shrinking it, because
	// GlyphScale is whole-number only and 1 is the floor — the overlay's cards are half
	// size and their glyphs are not.
	DashLeft:   8,
	DashTop:    72,
	DashWidth:  7,
	DashHeight: 4,
	DashGap:    3,
}

// Stack is the draw pile's card: a back, and nothing else, at the size the screen has room
// for rather than at any proportion of Hand.
//
// **It is smaller than Mini and that is forced by the layout, not chosen.** The strip below
// the action-point bar is about 86 pixels tall, and a 132-pixel Mini card does not fit in it
// — see the deck stack in internal/screens/combat_flight.go for the arithmetic.
//
// What makes that survivable is the thing that makes a back different from a face: **there
// is no detail to lose.** A face at this size would be illegible, which is exactly why Mini
// already drops the damage badge and why nothing smaller than Mini exists for one. A back is
// a dark rounded rectangle with a triangle on it, and the triangle is sized as a proportion
// of the card, so it is the same drawing here as at hand size.
//
// The proportions are Hand's — 44x64 against 180x264 — so the stack reads as the same object
// as the cards in the row, seen smaller.
var Stack = Style{
	Width: 44, Height: 64,

	CornerRadius: 4,
	BorderWidth:  0,

	ShowName:     false,
	ShowCategory: false,
	ShowDamage:   false,
}

// RingStyle is a ring, in the card format.
//
// Same footprint, corners and border treatment as Hand, so the two read as one game.
// What it drops is everything that describes a *play*: no category glyph, because a ring
// has no phase; no cost dashes, because it is not played from a hand; no damage badge.
// What it gains is Spec.Art across the face.
//
// **Not wired into the game.** Nothing builds one of these yet — it exists so the design
// can be looked at on the contact sheet before rings become real.
var RingStyle = Style{
	Width: 180, Height: 264,

	CornerRadius: 12,
	BorderWidth:  6,

	ShowName:     true,
	ShowCategory: false,
	ShowDamage:   false,

	TextLeft:     12,
	NameTop:      14,
	NameSize:     20,
	NameCentered: true,

	ArtTop:   52,
	ArtInset: 16,
	ArtMaxH:  190,
}
