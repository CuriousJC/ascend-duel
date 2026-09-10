package cards

// Style is a card's geometry at one size.
//
// **A *generated* glyph cannot be made smaller**, and that is still true: its rim is derived one
// pixel thick, so a fractional scale drops pixels out of the only edge it has. GlyphScale must
// stay a whole number and 1 is the floor.
//
// **Drawn art is the exception** *(2026-08-23)*, which is what lets Mini carry a form mark at all.
// A painting has interior detail to average, so `systems.RenderGlyphAt` will halve one — and the
// four form marks are drawings. See that function for the line between the two.
type Style struct {
	Width, Height int

	// CornerRadius and BorderWidth are the shape. Named here rather than written inline
	// at the draw site so both are tunable in one place, which is the whole reason the
	// contact sheet is worth having.
	CornerRadius int
	BorderWidth  int

	// What this size is big enough to show.
	ShowName bool
	ShowForm bool

	TextLeft int
	NameTop  int
	NameSize float64

	// NameCentered centres the name across the card instead of starting it at TextLeft.
	// Rings use it: with no glyph column down the left there is nothing for a
	// left-aligned name to line up with, and it reads as having slipped off centre.
	NameCentered bool

	// NameWordPerLine breaks the name at every space, one word to a line, and
	// NameLinePitch is how far apart those lines sit.
	//
	// **A break at every space rather than a wrap at the card's width** *(owner's call,
	// 2026-08-21)*. Rings use it. Their names are one or two words and the two-word ones are
	// what a width-wrap handles worst: "Frozen Lightning" either fits by a hair and reads as a
	// sentence squeezed into a card, or misses by a hair and breaks anyway — and which of those
	// happens depends on the font, so the same catalogue lays out differently for a change
	// nothing about rings caused. Breaking always is a layout that cannot drift.
	//
	// **The style has to leave room for the lines it allows.** Nothing clamps a name to two
	// words, so a three-word ring runs into whatever is under it; `TestARingNameClearsItsArt`
	// checks the longest name the file actually holds rather than a hypothetical one.
	NameWordPerLine bool
	NameLinePitch   int

	// The form mark, above the cost stack: the box its art is centred in.
	//
	// **The box is a number here rather than `systems.SizeOf`** *(2026-08-15)*. It was, while the
	// mark was a generated glyph and the glyph's own size was the authority — assuming one was
	// how a 22-pixel shape got a 64-pixel hole. The layout names the space instead, and centring
	// by ink means the mark fills it whatever the drawing leaves as margin. A glyph bigger than
	// this box overflows it rather than resizing it, so the box is the one number to change —
	// see `systems.formArtSize`, which is the size the four marks are authored at to match.
	FormTop  int
	FormSize int

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

	// The form mark's left edge, and the scale a glyph would be blown up by.
	//
	// **There is no damage badge at all any more** *(2026-08-14)*. It went in two steps: the
	// 64-pixel generated sword first, because it said what the corner mark already says
	// while taking the room the effect text needed, and then the figure beside it — because
	// the text says what the card does, and "Deal 2x DMG" is the same fact stated once instead
	// of twice. `systems.GlyphDamage` still exists and is still on the glyph sheet; nothing on
	// a card draws it.
	//
	// GlyphScale is the whole-number pixel repeat a mark is blitted at, applied by placeInk on
	// top of whatever size the art came back at. It is 1 on every style: a form mark is already
	// rendered to FormSize, so the repeat has nothing left to do and a fractional one would drop
	// pixels out of a one-pixel rim. See the note at the top of this file.
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

	// Spec.Effects drawn as a centred row of squares along the bottom edge. A zero
	// EffectSize means the style shows none, which is every style but the two fighter cards.
	//
	// **A square box each, and the badge is fitted into it** — the art is 500px and this is
	// twenty, so it is scaled like the portrait rather than blitted like a glyph. The row is
	// centred on the card and closes up as badges come and go, so two statuses sit in the
	// middle rather than clinging to the left.
	EffectSize int
	EffectTop  int
	EffectGap  int

	// Spec.Counter drawn as a badge in the bottom-right corner. A zero CounterHeight means the
	// style shows none, which is every style but RingStyle.
	//
	// **It is measured from the card's own right and bottom edges**, unlike everything else here,
	// which is measured from the top-left. The badge belongs to the corner: a top-left offset
	// would have to be recomputed by hand every time the card's size changed, and the one thing
	// this badge must never do is drift off the card it is counting.
	//
	// **CounterHeight is the band the figure is centred in, not a box drawn around it.** There is
	// nothing behind the text; the height is fixed so a two-character figure and a four-character
	// one sit on the same line, which is what lets a row of rings be read across.
	CounterHeight int
	CounterRight  int
	CounterBottom int
	CounterSize   float64

	// CounterRadius is the disc behind the figure. A zero radius draws none, which is what every
	// style but RingStyle has, and it is the same box: the disc is `2*CounterRadius` square,
	// measured in from CounterRight and up from CounterBottom.
	CounterRadius int

	// Bleed is how far past the card’s right and bottom edges the rendered image extends.
	//
	// **A card image is the card’s own size everywhere else, and that is worth keeping true**:
	// every caller draws it at a top-left point and measures its hit box off the style, so an image
	// that is quietly larger than the card is a thing to be able to point at. This is the one
	// reason there is a field for it rather than a constant — the accumulator badge is centred on
	// the bottom-right corner, so three quarters of it lies outside the card, and a badge clipped
	// to the card would be a quarter disc filling the corner instead.
	//
	// **It grows the image, never the card.** The face is still drawn at the origin at Width x
	// Height, so nothing about the layout moves; what changes is that there are pixels to the right
	// of it and below it for a corner ornament to live in.
	Bleed int
}

// NameLinesAbove is how many lines of name this style can draw before its ink would reach
// `floor` — the top of whatever sits under the name — given a line's height in the font the
// caller measured.
//
// **The line height is passed in rather than derived**, because the only honest source of one
// is a parsed font at the style's point size, and this package's geometry is deliberately
// readable without a font in hand. The caller has the Faces; this has the offsets.
//
// A style that does not break its name gets 1, which is the truth: it draws one line whatever
// is under it.
func (st Style) NameLinesAbove(floor, lineHeight int) int {
	if !st.NameWordPerLine || st.NameLinePitch <= 0 {
		return 1
	}
	n := 0
	for st.NameTop+n*st.NameLinePitch+lineHeight <= floor {
		n++
	}
	return n
}

// Hand is the card as the hand draws it, and the size every constant here is written
// for. 200x280 — five by seven, exactly a playing card's proportions.
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
	Width: 200, Height: 280,

	CornerRadius: 15,
	BorderWidth:  4,

	ShowName: true,
	ShowForm: true,

	TextLeft:     15,
	NameTop:      18,
	NameSize:     25,
	NameCentered: true,

	// The mark is centred on its ink in this box, so the box is what the layout tests hold.
	//
	// **It sits inside the card rather than hanging off the corner** *(2026-08-15)*. A glyph
	// placed at 0,0 is cropped by the card's own curve, which a silhouette survives — it loses a
	// corner and still reads as itself — but which costs a mark carrying detail its top-left
	// quarter. The clip in blitGlyph still applies; this is the box moving, not the crop going.
	//
	// **FormSize is 32 because that is what the art is**, not because it is a proportion of the
	// card: the four form marks are PNGs authored at 32x32 — see systems.formArtSize — and a box
	// asking for more would upsample a one-pixel rim. The box around it is free to move; this
	// number is not, until the marks are re-exported.
	FormTop:  10,
	FormSize: 32,

	// The cost ticks, hamburger-style, under the form mark. A tick rather than a bar: four of
	// them have to stack without reading as a hatched block, which is what the gap holds.
	//
	// **20 is the width the column has, not a round number.** DashLeft is 10 and TextColumnLeft
	// is 33, so 23 is the wall; TestTheCostColumnStaysOutOfTheTextColumn is what fails on a wider
	// one, and widening past it means moving the text column rather than nudging this.
	DashLeft:   10,
	DashTop:    60,
	DashWidth:  20,
	DashHeight: 5,
	DashGap:    6,

	GlyphScale: 1,
	GlyphInset: 13,

	TextColumnLeft: 33,
	TextInset:      10,
	TextBandTop:    55,
	TextBandBottom: 268,
	TextSize:       22.5,
	TextLineHeight: 28,
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

// Scaled is this style at a fraction of its size: every measurement multiplied by num/den,
// every flag carried across unchanged.
//
// **It exists so a smaller card is the same card** *(2026-08-23)*. Mini was authored by hand
// beside Hand and drifted from it field by field — a name at 14 where half of 20 is 10, dashes
// at 7x4 where half of 13x8 is 6x4, a form box left at the full 32 on a card half the size, and
// a mark that consequently sat a third of the way down the face instead of in the corner. Every
// one of those was defensible on its own and together they made the overlay a second design of a
// card rather than a small copy of one. **The owner's call was that the deck panel shows exactly
// the hand's card, only smaller**, and a derivation is the only way that stays true: a field
// added to Hand now reaches Mini without anyone remembering to halve it.
//
// **Rational rather than a float**, so the arithmetic is exact and reviewable: half of 224 is 112
// and not 111.99. Rounding is to nearest, which is what keeps a 13px dash at 6 rather than at 7.
//
// **It is a geometry scale and cannot judge legibility.** Halving TextSize gives a real number
// that a font may not be able to set readably; that is a question for whoever looks at the sheet,
// not something this can decide. What it guarantees is proportion.
func (st Style) Scaled(num, den int) Style {
	i := func(v int) int {
		if v == 0 {
			return 0
		}
		// Rounded to nearest rather than truncated: a truncating scale walks every offset
		// upward and to the left, so the whole face creeps off centre as the factor shrinks.
		return (v*num*2 + den) / (den * 2)
	}
	f := func(v float64) float64 { return v * float64(num) / float64(den) }

	out := st
	out.Width, out.Height = i(st.Width), i(st.Height)
	out.CornerRadius, out.BorderWidth = i(st.CornerRadius), i(st.BorderWidth)

	out.TextLeft, out.NameTop, out.NameSize = i(st.TextLeft), i(st.NameTop), f(st.NameSize)
	out.NameLinePitch = i(st.NameLinePitch)

	out.FormTop, out.FormSize = i(st.FormTop), i(st.FormSize)

	out.DashLeft, out.DashTop = i(st.DashLeft), i(st.DashTop)
	out.DashWidth, out.DashHeight, out.DashGap = i(st.DashWidth), i(st.DashHeight), i(st.DashGap)

	out.GlyphInset = i(st.GlyphInset)

	out.TextColumnLeft, out.TextInset = i(st.TextColumnLeft), i(st.TextInset)
	out.TextBandTop, out.TextBandBottom = i(st.TextBandTop), i(st.TextBandBottom)
	out.TextSize, out.TextLineHeight = f(st.TextSize), i(st.TextLineHeight)

	out.ArtTop, out.ArtInset, out.ArtMaxH = i(st.ArtTop), i(st.ArtInset), i(st.ArtMaxH)

	out.StatsTop, out.StatRowPitch, out.StatSize = i(st.StatsTop), i(st.StatRowPitch), f(st.StatSize)

	out.HealthBarInset, out.HealthBarTop = i(st.HealthBarInset), i(st.HealthBarTop)
	out.HealthBarHeight, out.HealthTextTop = i(st.HealthBarHeight), i(st.HealthTextTop)
	out.HealthTextSize = f(st.HealthTextSize)

	out.EffectSize, out.EffectTop, out.EffectGap = i(st.EffectSize), i(st.EffectTop), i(st.EffectGap)

	// **The counter was missing from this list until 2026-09-09**, which meant the one field block
	// on Style authored in one space and drawn in another: the offsets are measured from the card's
	// own edges, so an unscaled CounterBottom on a card a quarter taller put the figure a quarter
	// of the growth further from the bottom than it was authored to be. The numbers in ringAuthored
	// were re-tuned in the same commit, so the comments there now describe what is drawn.
	out.CounterHeight, out.CounterRight = i(st.CounterHeight), i(st.CounterRight)
	out.CounterBottom, out.CounterRadius = i(st.CounterBottom), i(st.CounterRadius)
	out.CounterSize = f(st.CounterSize)

	out.Bleed = i(st.Bleed)

	// GlyphScale is a whole-number pixel repeat, not a measurement. Scaling it would ask for a
	// fractional repeat, which is the one thing a derived rim cannot survive.
	return out
}

// Mini is the deck overlay's card: **Hand at exactly half size**, 100x140.
//
// **It is derived rather than authored** *(2026-08-23)*. See Scaled for what that fixed and why
// the owner asked for it. The consequence to know: there is no longer a place to tune the small
// card on its own. A change wanted here is a change to Hand, or it is a second field on Style —
// which is the right cost, because the last version of this comment spent four paragraphs
// explaining differences that turned out to be drift rather than design.
//
// **What half size does not fix is legibility, and one thing is genuinely marginal**: the effect
// text lands at 9pt in a 64-pixel measure. The panel is a list of what you own rather than a
// place cards are played from, so the name and the mark carry it; look at `tools/cardsheet`
// before assuming the sentence can be read.
// The card is 200x280 — five by seven, a playing card's proportions.
//
// **Every number in this file is the number that is drawn**, and a card size is changed by
// editing these rather than by multiplying them on the way out. A scale between the authored
// figure and the drawn one makes the border width and the corner radius rounding artifacts
// nobody chose, and every asset drawn at the card's real size — the form marks, the ring art —
// then needs an exemption from it.
//
// **The height is what fixes the size, and 280 is near the ceiling.** The combat screen stacks
// three card heights — the top row, the table row and the hand — plus the furniture above,
// between and below them, leaving `3h + 200 <= 1080`. The cap is 293 and 280 clears it by 13.
// The width has never been the binding constraint: a hand of eight fits flat well past this.
//
// **`Style.Scaled` stays**, because Mini and Stack are genuinely derived sizes and a field added
// to Hand has to reach them without anyone remembering to halve it.

// centringItsMark puts the glyph inset back on the centre line of a card that has no text column
// to line a mark up against.
//
// **Token is the only style whose `GlyphInset` is an arithmetic rather than a measurement** — it is
// authored as `(Width - FormSize) / 2` — so holding `FormSize` back while the card grows leaves the
// mark off centre by the difference. Hand's inset is a left margin and means what it says at any
// size; this one has to be recomputed. TestATokenCentresItsColumn is what fails otherwise.
func centringItsMark(st Style) Style {
	st.GlyphInset = (st.Width - st.FormSize) / 2
	return st
}

var Mini = Hand.Scaled(1, 2)

// Stack is the draw pile's card, derived from Hand — see stackOf.
var Stack = stackOf(Hand)

// Token is the hands panel's card, centred on its mark — see tokenBase and centringItsMark.
var Token = centringItsMark(tokenBase)

// Stack is the draw pile's card: a back, and nothing else, at **three quarters of Hand**
// *(2026-09-04, owner's call)*.
//
// **The size is a proportion of the card now, not of the strip it used to stand in.** It was a
// fifth — 41x56 — because the pile hung off the bottom edge in the band under the action-point
// bar, and that band was about 86 pixels deep; anything larger reached up into the bar. The pile
// moved into the duelist card's column on 2026-09-04 and the constraint left with it. A full-size
// card was tried first and read as the biggest thing on the screen, which is the opposite of what
// a pile nobody is playing from should say.
//
// **Derived from Hand rather than authored** *(2026-09-04)*, the way Mini is, so the pile reads as
// the same object as the cards in the row seen smaller and cannot drift out of proportion with
// them — see TestStackStyleKeepsTheCardsProportions. What stays authored is the chrome, which is
// not a proportion: no name, no form mark, no border, because a back has none of them.
//
// What makes any of these sizes survivable is the thing that makes a back different from a face:
// **there is no detail to lose.** A back is a dark rounded rectangle with a triangle on it, and
// the triangle is sized as a proportion of the card, so it is the same drawing here as at hand
// size.
func stackOf(st Style) Style {
	out := st.Scaled(3, 4)
	out.ShowName, out.ShowForm = false, false
	out.BorderWidth = 0
	return out
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
//	 15  name              centred   (15..46 at 25pt)
//	 55  portrait          55..195   (Spec.Art, scaled to fit and centred)
//	201  health bar        201..219
//	225  hit points        "42/60", centred
//	246  status badges     246..271  (Spec.Effects, a centred row)
//	272  inside of the bottom border
//
// **The badges are on this card and not the duelist's** *(2026-08-16)*, which breaks the
// twins rule everywhere except where that rule actually bites — the bar and the fraction are
// still at identical offsets, and the band under them is the same free strip on both. The
// reason was that nothing could put a status on the player: the enemy wears no rings and a ring
// is what makes a status happen. **`DuelistStyle` gained the row on 2026-08-31**, in the three
// lines this comment promised, and what fills it is not a status — it is the shield count, one pip
// per shield. The band is at the same offsets on both cards, so the two still read as twins.
//
// **The strip they sit in is what was left, not what was wanted.** The fraction's ink ends
// around y=246 and the border starts at 272, so the badges get twenty-five pixels — small
// for a 500-pixel drawing, and legible because what a badge has to say is a colour and a rough
// shape rather than a picture. `TestStatusBadgesClearTheHealthTextAndTheBorder` holds both
// ends of that strip; making them bigger means moving the fraction on *both* fighter cards.
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
	Width: 200, Height: 280,

	CornerRadius: 15,
	BorderWidth:  4,

	ShowName: true,
	ShowForm: false,

	TextLeft:     15,
	NameTop:      15,
	NameSize:     25,
	NameCentered: true,

	ArtTop:   55,
	ArtInset: 15,
	ArtMaxH:  140,

	HealthBarInset:  15,
	HealthBarTop:    201,
	HealthBarHeight: 18,
	HealthTextTop:   225,
	HealthTextSize:  22.5,

	EffectSize: 25,
	EffectTop:  246,
	EffectGap:  8,
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
//	 18  name              centred   (18..48 at 25pt)
//	 70  DMG               70..96     label left, figure right
//	108  AP               108..134
//	146  Vitae            146..172
//	201  health bar        201..219
//	225  hit points        "42/60", centred
//	246  shield pips       246..271  (Spec.Effects, a centred row)
//	272  inside of the bottom border
//
// **The shield row is the enemy's badge row, seat for seat** *(2026-08-31)*. It holds five, which
// is `combat`'s cap on a duelist's shields for the same reason — a turn is five cards, so a sixth
// shield could never be spent. The row closes up as shields are eaten, so three pips sit centred
// rather than clinging to the left, and an unshielded duelist draws nothing at all.
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
	Width: 200, Height: 280,

	CornerRadius: 15,
	BorderWidth:  4,

	ShowName: true,
	ShowForm: false,

	TextLeft:     15,
	NameTop:      18,
	NameSize:     25,
	NameCentered: true,

	StatsTop:     70,
	StatRowPitch: 38,
	StatSize:     21.25,

	HealthBarInset:  15,
	HealthBarTop:    201,
	HealthBarHeight: 18,
	HealthTextTop:   225,
	HealthTextSize:  22.5,

	EffectSize: 25,
	EffectTop:  246,
	EffectGap:  8,
}

// WormStyle is a worm, in the card format *(2026-08-22)*.
//
// **A picture with its text under it**, which is what the owner asked for and is the shape a
// creature card wants: the worm is a thing you are catching, so the face is mostly the thing. It
// borrowed `Hand` until now, which drew a worm as a cost column with no cost and a paragraph
// beside the empty space where a form mark was not.
//
// What it drops is everything describing a *play* — no form mark, no cost dashes — for the reason
// RingStyle drops them: a worm is not played from a hand and resolves in no phase. What it gains
// over RingStyle is the text band, because a worm's whole content is the sentence saying what it
// does to a card.
//
// **The art is a placeholder for every worm today.** `Spec.Art` is filled from the shared default
// image, so the box is the seat the art goes into rather than a box that will have to be invented
// when there is some.
var WormStyle = Style{
	Width: 200, Height: 280,

	CornerRadius: 15,
	BorderWidth:  4,

	ShowName: true,
	ShowForm: false,

	TextLeft:     15,
	NameTop:      18,
	NameSize:     25,
	NameCentered: true,

	// Between the name and the text band. What is left after a line of name above and five lines
	// of text below is a 75-pixel box, which is why the art is the smallest thing on this card.
	ArtTop:   55,
	ArtInset: 33,
	ArtMaxH:  75,

	// The full width, unlike Hand — there is no cost column to leave room for. Centred in the band
	// under the art for the same reason Hand centres in its own: a one-line worm and a two-line one
	// should look like the same card.
	TextColumnLeft: 15,
	TextInset:      10,
	TextBandTop:    140,
	TextBandBottom: 265,
	TextSize:       21.25,
	TextLineHeight: 25,
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
	Width: 200, Height: 280,

	CornerRadius: 15,
	BorderWidth:  4,

	ShowName: true,
	ShowForm: false,

	TextLeft:     15,
	NameTop:      18,
	NameSize:     25,
	NameCentered: true,

	// **One word to a line** *(2026-08-21)*, which is what buys the art box below its room:
	// a two-word ring is two lines of 25pt, and 28 is that size plus the gap that keeps two
	// capitals from touching.
	NameWordPerLine: true,
	NameLinePitch:   28,

	// The artwork is fitted to this box rather than drawn at its own size. **The art is square, so
	// its height is set by the 160-pixel width of the box rather than by ArtMaxH**; what ArtMaxH
	// does is state the floor a two-line name has to clear, which is what TestARingNameClearsItsArt
	// holds it to.
	ArtTop:   78,
	ArtInset: 20,
	ArtMaxH:  150,

	// The accumulator figure, on a disc **tucked into the bottom-right corner** — flush to both
	// edges, so the disc's own curve meets the card's rather than sitting a margin inside it. That
	// is what the zero offsets mean, and it is why drawCounter clips the disc to the card's
	// silhouette: a circle tangent to both edges overlaps the corner curve unless its radius
	// happens to equal the card's, and an unclipped one would square the corner off.
	//
	// **The band is what pays for the figure, and the art pays for the band** *(owner's call,
	// 2026-09-09)*. This is the ring saying how big it has grown, on the one card whose whole job
	// is to be read at a glance, so it is set large enough not to be leaned into. The art is square
	// and fitted, so trimming ArtMaxH trims every side of it and nothing else on the card moves;
	// the box ends at 228 and the disc starts at 245, so the two still do not meet.
	//
	// **Only rings have one**, because only rings grow. Nothing else on the card is displaced by
	// it: the corner it takes was empty on every ring in the file.
	CounterHeight: 35,
	CounterRight:  0,
	CounterBottom: 0,
	CounterSize:   26.25,
	CounterRadius: 18,

	// Enough for the disc's overhang and for the widest figure past it: the disc is 36 across and
	// centred on the corner, so this is a little over the half of it. A figure wider than the bleed
	// is pulled back inside it rather than cut.
	Bleed: 28,
}

// Token is a card reduced to the three things a hand is counted on: its **element**, its
// **form** and what it **costs**. 40x56 — a quarter of Hand in each dimension, a sixteenth of
// its area — and it carries no name, no effect text and no picture.
//
// **It is authored rather than derived, and that is the one place it departs from Mini's rule**
// *(2026-08-24)*. A quarter of Hand gives an 8px form mark and a 5x1 tick, which is a mark with
// its detail averaged away and a tick that reads as a scratch — the same floor the glyph rules
// describe. The mark therefore stays at Mini's 16, which the drawn art can be halved to twice and
// still be read, and the ticks stay at Hand's own 20x5. **So this is not a small card; it is a
// different object**, which is why deriving it would be claiming something untrue.
//
// **The left column, standing on its own.** Everything it draws — a tinted form mark with the
// cost ticks under it — is exactly what a Hand card puts down its left edge, so a row of these
// is the same reading in the same colours, and nothing here can drift from the card it stands
// for except by that column moving.
//
// The hands panel is the caller: eighteen rungs, each shown as the cards that build it, is a
// hundred-odd cards on one screen, and at Mini's 100x140 that is a panel of cards with no room
// left for the ladder.
var tokenBase = Style{
	Width: 50, Height: 70,

	CornerRadius: 8,
	BorderWidth:  3,

	ShowName: false,
	ShowForm: true,

	// **The mark stays at 16 and the ticks at Hand's own footprint**, which is the whole reason
	// this style is written out rather than derived: a quarter of Hand gives an 8px mark with its
	// detail averaged away and a tick that reads as a scratch.
	FormTop:  8,
	FormSize: 16,

	DashLeft:   15,
	DashTop:    33,
	DashWidth:  20,
	DashHeight: 5,
	DashGap:    4,

	GlyphScale: 1,
	GlyphInset: 17,
}
