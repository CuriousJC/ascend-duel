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
	ShowName   bool
	ShowFamily bool

	TextLeft int
	NameTop  int
	NameSize float64

	// NameCentered centres the name across the card instead of starting it at TextLeft.
	// Rings use it: with no glyph column down the left there is nothing for a
	// left-aligned name to line up with, and it reads as having slipped off centre.
	NameCentered bool

	// The family mark, above the cost stack: the box it is centred in, and the point size the
	// placeholder letter is set at.
	//
	// **The box is a number here rather than `systems.SizeOf`** *(2026-08-15)*. It was, while the
	// mark was a generated glyph and the glyph's own size was the authority — assuming one was
	// how a 22-pixel shape got a 64-pixel hole. A letter has no intrinsic size, so the layout has
	// to name the space it gets, and centring by ink means the letter fills it whatever the font
	// does. When the families get silhouettes, a glyph bigger than this box overflows it rather
	// than resizing it, and the box is the one number to change.
	FamilyTop        int
	FamilySize       int
	FamilyLetterSize float64

	// The cost dashes, hamburger-style, below the category glyph.
	//
	// **They are the whole of the cost column now.** A card used to stack a category glyph, the
	// dashes and a damage badge down the same 64 pixels; the badge is gone and the glyph has
	// moved into the corner above the column, so what is left is one stack of bars whose width
	// sets how much of the card the effect text gets. A cost above four grows the stack
	// downward, and TestLeftColumnDoesNotCollide fails rather than letting it off the card.
	DashLeft   int
	DashTop    int
	DashWidth  int
	DashHeight int
	DashGap    int

	// The family mark's left edge, and the scale a glyph would be blown up by.
	//
	// **There is no damage badge at all any more** *(2026-08-14)*. It went in two steps: the
	// 64-pixel generated sword first, because it said what the corner mark already says
	// while taking the room the effect text needed, and then the figure beside it — because
	// the text says what the card does, and "Deal 2x DMG" is the same fact stated once instead
	// of twice. `systems.GlyphDamage` still exists and is still on the glyph sheet; nothing on
	// a card draws it.
	//
	// GlyphScale is unread while the mark is a letter — a typeface scales by asking for a bigger
	// size, not by repeating pixels — and is kept for the glyph path it was written for.
	GlyphScale int
	GlyphInset int

	// Spec.Text, wrapped and set as a block **centred in the space the left column leaves**.
	// A zero TextLineHeight means the style has none, which is every style but Hand — the
	// block exists only on a card big enough to read.
	//
	// **The left column is treated as a column and the text gets the rest** *(2026-08-14)*.
	// TextColumnLeft is where the text may start; everything left of it belongs to the category
	// glyph, the cost dashes and the damage figure. Inside that column the block is centred
	// horizontally, and vertically it is centred in TextBandTop..TextBandBottom — under the
	// name, down to the inside of the bottom border. Short text sits in the middle of the card
	// rather than clinging to the bottom edge.
	//
	// **TextLines() is what the card fits, not a limit on the writing.** Render draws every line
	// it wraps to, so text too long for the band runs off the card rather than being silently
	// cut; TestEveryCardTextFitsItsBand is what fails first. Same rule as MaxStatLines and the
	// cost dashes.
	TextColumnLeft int
	TextInset      int
	TextBandTop    int
	TextBandBottom int
	TextSize       float64
	TextLineHeight int

	// ArtTop and ArtInset frame Spec.Art, used by rings. The art is scaled to fit the
	// box they describe and centred in it.
	ArtTop   int
	ArtInset int
	ArtMaxH  int

	// The stat rows: Spec.Stats drawn one per row, label against the left margin and
	// figure against the right. Zero StatRowPitch means the style has none, which is
	// every style but DuelistStyle.
	//
	// **One size for both halves of a row.** A quieter label and a louder figure was the
	// character block's shape and it worked there because the two were stacked; on one
	// baseline, two sizes read as a typo rather than as a hierarchy. The label is set in
	// LabelInk instead, which is the same distinction made with colour.
	StatsTop     int
	StatRowPitch int
	StatSize     float64

	// The health bar and the fraction under it, drawn from Spec.Life and Spec.MaxLife.
	// Zero HealthBarHeight means the style has no health and neither is drawn.
	//
	// HealthBarInset is the bar's side margin. **It is its own field rather than a reuse
	// of ArtInset**, which is what it was until the duelist card arrived: that card has a
	// health bar and no art at all, so borrowing the art box's margin would have made a
	// style state a measurement for something it does not draw.
	HealthBarInset  int
	HealthBarTop    int
	HealthBarHeight int
	HealthTextTop   int
	HealthTextSize  float64
}

// Hand is the card as the hand draws it, and the size every constant here is written
// for. 162x224 — roughly a playing card's proportions, a little squarer than one.
//
// **It was 180x264 until 2026-08-11**, when every size came down by a tenth across and by
// 15% down the face to give the screen back some room. The y offsets below did *not* scale
// with it and deliberately so: what the column holds is fixed-size art that cannot be scaled,
// so the height came off the empty strip and nothing else. That strip is what the effect text
// occupies now.
//
// The face reads: the name centred across the top, the category glyph in the corner, a stack
// of cost dashes down the left edge, and the effect text filling everything else.
//
//	  0  category glyph      0..32    (32px at the largest, in the corner, cropped by the curve)
//	 14  name               centred
//	 48  cost dashes        48..95    (four at 8 on a 5 gap)
//	 44  effect text        44..214   (x=26..154, block centred both ways)
//	218  inside of the bottom border
//
// **The cost column is 26px and the glyph is not in it.** The glyph is 32 wide and would set
// the column's width single-handed, so it sits in the corner *above* the text band, cropped by
// the card's own curve. Below it the column holds one thing — the 13px dash marks — which is
// what a "cost column" ought to mean and is why 26 is nearly the floor.
//
// **The dashes moved down when the drawn glyphs arrived**, from 42 to 48: Sherman's sword
// and shield are 32 pixels where the generated ones were 22, and the ten pixels had to come
// from somewhere. They are still at 48 with the glyph hard in the corner, which leaves 16
// pixels of air between the two rather than a join.
//
// **The name is centred and the glyph is in the corner beside it, not under it.** Those
// two go together: a left-aligned name would sit directly on top of the glyph, and centring it
// is what clears the space. The name is centred on the *card*, not on the room left over
// beside the glyph, so a long enough name would still reach back into it —
// TestNameClearsTheCategoryGlyph checks every concept in the deck against that.
//
// **The card is a column and a paragraph** *(2026-08-14)*. There is no damage badge at all any
// more — not the 64-pixel sword and not the figure that briefly replaced it — because the text
// says what the card deals and a number beside it was the same fact multiplied out. The column
// is the cost dashes; the text takes everything right of them and is centred in it, both ways,
// in the band running from under the name to the inside of the bottom border.
//
// **Centred rather than top-left because the block is the card's whole right-hand side.** A
// two-line effect pinned to the top of a 170-pixel band reads as a caption that has come
// unstuck; centred, a short card and a long one look like the same design.
//
// What that buys is size: 18pt against the 13 the text was set in when it ran the full width
// under a badge. What it costs is measure — 128 pixels, and the wording has to be short words.
// `TestEveryCardTextFitsItsBand` holds the line count against the band and
// `TestNoEffectTextWordIsWiderThanItsColumn` holds each word against the measure.
var Hand = Style{
	Width: 162, Height: 224,

	CornerRadius: 12,
	BorderWidth:  6,

	ShowName:   true,
	ShowFamily: true,

	TextLeft:     12,
	NameTop:      14,
	NameSize:     20,
	NameCentered: true,

	// 40pt in a 32px box: the letter is centred on its ink, so the point size only has to be
	// large enough to fill the box and the box is what the layout tests hold.
	//
	// **It sits inside the card rather than hanging off the corner** *(2026-08-15)*. The glyph
	// before it was placed at 0,0 and deliberately cropped by the card's own curve — a
	// silhouette loses a corner and still reads as itself. A letter does not: clipped at 0,0 the
	// S came out as a shape with its top-left quarter missing, which reads as a rendering fault
	// rather than as a mark. The clip in blitGlyph still applies and is still what a glyph will
	// want; this is the box moving, not the crop going.
	FamilyTop:        8,
	FamilySize:       32,
	FamilyLetterSize: 40,

	DashLeft:   8,
	DashTop:    48,
	DashWidth:  13,
	DashHeight: 8,
	DashGap:    5,

	GlyphScale: 1,
	GlyphInset: 10,

	TextColumnLeft: 26,
	TextInset:      8,
	TextBandTop:    44,
	TextBandBottom: 214,
	TextSize:       18,
	TextLineHeight: 22,
}

// TextLines is how many lines of effect text the band holds at this style's line height.
//
// **Derived rather than a field**, so a band that moves cannot leave a constant behind
// claiming a capacity the card does not have — the same rule `resolutionCapacity` follows on
// the combat screen.
func (st Style) TextLines() int {
	if st.TextLineHeight <= 0 {
		return 0
	}
	return (st.TextBandBottom - st.TextBandTop) / st.TextLineHeight
}

// Mini is the deck overlay's card: half the hand's size, kept to the same proportions.
//
// **Half of the new Hand since 2026-08-11** — 81x112, down from 90x132. `deckStackPitch` in
// internal/screens came down with it, because the pitch is the width less the overlap and a
// pitch wider than the card is a row with gaps in it.
//
// **It shows the name now, and that was the last thing missing.** The overlay's rows
// group by element and the glyph and dashes give phase and cost, so the only fact a card
// was not stating was which concept it is. At 14pt the longest name in the deck —
// "Prepare" — measures 35 pixels against the usable width, so it was never a question of
// room; it was a question of the cards being overlapped so tightly that only 29 pixels
// showed. Widening the row (see deckStackPitch) is what made the space real.
//
// The one thing still absent is the effect text, and that is forced: an 81-pixel card less a
// cost column leaves about 55 pixels of measure, which is four or five characters a line at
// any size legible here. What a card does is a function of the concept, so a named card
// implies it — and the overlay is a list of what you own, not a place you play from.
//
// The reading order matches Hand deliberately — name centred across the top, then the
// glyph, then the dashes under it — so the two sizes are the same card and not two
// designs.
var Mini = Style{
	Width: 81, Height: 112,

	CornerRadius: 6,
	BorderWidth:  3,

	ShowName:   true,
	ShowFamily: true,

	TextLeft:     8,
	NameTop:      8,
	NameSize:     14,
	NameCentered: true,

	GlyphScale:       1,
	GlyphInset:       8,
	FamilyTop:        36,
	FamilySize:       32,
	FamilyLetterSize: 34,

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
// The proportions are Hand's — 40x54 against 162x224 — so the stack reads as the same object
// as the cards in the row, seen smaller. It came down with the rest on 2026-08-11; the strip
// it has to fit did not change, so this one had room to spare either way.
var Stack = Style{
	Width: 40, Height: 54,

	CornerRadius: 4,
	BorderWidth:  0,

	ShowName:   false,
	ShowFamily: false,
}

// EnemyStyle is the opponent, in the card format *(2026-08-11)*.
//
// **The enemy was a bare sprite with a health bar hanging under it** until this landed —
// the last thing on the combat screen still drawn as a loose picture floating on the
// background. Putting it in the card format says the obvious thing: everything the duel is
// made of is a card, and the one you are fighting is one too.
//
// The face reads top to bottom: name, portrait, bar, numbers.
//
//	 12  name              centred   (12..36 at 20pt)
//	 44  portrait          44..156   (Spec.Art, scaled to fit and centred)
//	161  health bar        161..175
//	180  hit points        "42/60", centred
//	218  inside of the bottom border
//
// **The name moved above the portrait on 2026-08-12**, having sat between the portrait and
// the bar since the card was built. It puts the name where every other card in the game
// carries it — Hand, Mini and RingStyle all name themselves across the top — so the enemy
// reads as one of the set rather than as a card with its own reading order. What it costs is
// the portrait's proximity to its name; they are still adjacent, only the other way round.
//
// **Every offset here scaled with the card on 2026-08-11**, unlike Hand's — nothing on this
// face is fixed-size art. The portrait is scaled to fit its box and the bar is drawn to the
// width it is given, so the whole layout is a proportion of the card and stays one.
//
// **The portrait gets the middle and the rest shares the bottom**, which is the layout the
// owner asked for and also the one the art wants: the vendor portraits are wider than they
// are tall once cropped, so a box 138 wide by 112 gives them their width rather than letting
// height decide the scale.
//
// What it drops is everything describing a *play* — no category glyph, no cost dashes, no
// damage badge — for the same reason RingStyle does: none of them are things an enemy card
// is. `Element` is Basic, so the border is the neutral mid grey rather than claiming the
// opponent is made of fire.
var EnemyStyle = Style{
	Width: 162, Height: 224,

	CornerRadius: 12,
	BorderWidth:  6,

	ShowName:   true,
	ShowFamily: false,

	TextLeft:     12,
	NameTop:      12,
	NameSize:     20,
	NameCentered: true,

	ArtTop:   44,
	ArtInset: 12,
	ArtMaxH:  112,

	HealthBarInset:  12,
	HealthBarTop:    161,
	HealthBarHeight: 14,
	HealthTextTop:   180,
	HealthTextSize:  18,
}

// DuelistStyle is the player, in the card format *(2026-08-12)*.
//
// **It replaced the character block**, which was a framed box of stacked captions and figures
// in the top-left corner. The argument is the enemy card's, one seat further round the table:
// everything the duel is made of is a card, the opponent became one on 2026-08-11, and the
// player was the last thing on the screen still drawn as furniture. The two now sit in
// opposite corners in the same format, which is what makes them read as the two sides of one
// fight rather than as a HUD and a monster.
//
// The face reads top to bottom: name, stat rows, bar, numbers.
//
//	 14  name              centred   (14..38 at 20pt)
//	 56  DMG               56..77     label left, figure right
//	 86  AP                86..107
//	116  Vitae            116..137
//	161  health bar        161..175
//	180  hit points        "42/60", centred
//	218  inside of the bottom border
//
// **The bar and the fraction are at exactly the enemy card's offsets**, deliberately: the two
// cards face each other across the screen and a health bar that sat at a different height on
// each would make comparing them an act of measurement. Everything above the bar is free to
// differ, because that is where the two cards say different things — a portrait against three
// numbers.
//
// What it drops is everything describing a *play*, like EnemyStyle and RingStyle: a duelist
// is not something you put down from a hand. `Element` is Basic, so the border is the neutral
// mid grey — the same as the enemy's, since neither card is made of an element. If the two
// corners ever need telling apart by colour, that is one entry in the Element enum and not a
// change here.
var DuelistStyle = Style{
	Width: 162, Height: 224,

	CornerRadius: 12,
	BorderWidth:  6,

	ShowName:   true,
	ShowFamily: false,

	TextLeft:     12,
	NameTop:      14,
	NameSize:     20,
	NameCentered: true,

	StatsTop:     56,
	StatRowPitch: 30,
	StatSize:     17,

	HealthBarInset:  12,
	HealthBarTop:    161,
	HealthBarHeight: 14,
	HealthTextTop:   180,
	HealthTextSize:  18,
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
	Width: 162, Height: 224,

	CornerRadius: 12,
	BorderWidth:  6,

	ShowName:   true,
	ShowFamily: false,

	TextLeft:     12,
	NameTop:      14,
	NameSize:     20,
	NameCentered: true,

	// Scaled with the card, like the enemy's: the artwork is fitted to this box rather than
	// drawn at its own size, so there is nothing here that a smaller card breaks.
	ArtTop:   46,
	ArtInset: 16,
	ArtMaxH:  160,
}
