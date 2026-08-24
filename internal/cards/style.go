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
	// EffectSize means the style shows none, which is every style but EnemyStyle.
	//
	// **A square box each, and the badge is fitted into it** — the art is 500px and this is
	// twenty, so it is scaled like the portrait rather than blitted like a glyph. The row is
	// centred on the card and closes up as badges come and go, so two statuses sit in the
	// middle rather than clinging to the left.
	EffectSize int
	EffectTop  int
	EffectGap  int
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
	BorderWidth:  3,

	ShowName: true,
	ShowForm: true,

	TextLeft:     12,
	NameTop:      14,
	NameSize:     20,
	NameCentered: true,

	// The mark is centred on its ink in this box, so the box is what the layout tests hold.
	//
	// **It sits inside the card rather than hanging off the corner** *(2026-08-15)*. A glyph
	// placed at 0,0 is cropped by the card's own curve, which a silhouette survives — it loses a
	// corner and still reads as itself — but which costs a mark carrying detail its top-left
	// quarter. The clip in blitGlyph still applies; this is the box moving, not the crop going.
	FormTop:  8,
	FormSize: 32,

	// **Half the height and a quarter longer, 2026-08-23** *(owner's call)*. 13x8 was a bar; 16x4
	// is a tick, and four of them stack in 31 pixels where they used to take 47 — so the cost
	// column ends higher up the face without its top edge moving. The gap holds at 5, which is
	// what keeps four ticks reading as four rather than as a hatched block.
	//
	// **16 is the width the column has, not a round number.** DashLeft is 8 and TextColumnLeft is
	// 26, so 18 is the wall; TestTheCostColumnStaysOutOfTheTextColumn is what fails on a wider
	// one, and widening past it means moving the text column rather than nudging this.
	DashLeft:   8,
	DashTop:    48,
	DashWidth:  16,
	DashHeight: 4,
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

	// GlyphScale is a whole-number pixel repeat, not a measurement. Scaling it would ask for a
	// fractional repeat, which is the one thing a derived rim cannot survive.
	return out
}

// Mini is the deck overlay's card: **Hand at exactly half size**, 81x112.
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
var Mini = Hand.Scaled(1, 2)

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

	ShowName: false,
	ShowForm: false,
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
//	197  status badges     197..217  (Spec.Effects, a centred row)
//	218  inside of the bottom border
//
// **The badges are on this card and not the duelist's** *(2026-08-16)*, which breaks the
// twins rule everywhere except where that rule actually bites — the bar and the fraction are
// still at identical offsets, and the band under them is the same free strip on both. The
// reason is that nothing can put a status on the player: the enemy wears no rings and a ring
// is what makes a status happen. Drawing an empty band on the duelist card would be reserving
// space for a mechanic that does not exist. `DuelistStyle` gains it in three lines when one
// does.
//
// **The strip they sit in is what was left, not what was wanted.** The fraction's ink ends
// around y=197 at 18pt and the border starts at 218, so the badges get twenty pixels — small
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
	Width: 162, Height: 224,

	CornerRadius: 12,
	BorderWidth:  3,

	ShowName: true,
	ShowForm: false,

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

	EffectSize: 20,
	EffectTop:  197,
	EffectGap:  6,
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
	BorderWidth:  3,

	ShowName: true,
	ShowForm: false,

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
	Width: 162, Height: 224,

	CornerRadius: 12,
	BorderWidth:  3,

	ShowName: true,
	ShowForm: false,

	TextLeft:     12,
	NameTop:      14,
	NameSize:     20,
	NameCentered: true,

	// Between the name and the text band, and squarer than the ring's: what is left after a line
	// of name above and three lines of text below is a 96-pixel box.
	ArtTop:   44,
	ArtInset: 26,
	ArtMaxH:  96,

	// The full width, unlike Hand — there is no cost column to leave room for. Centred in the band
	// under the art for the same reason Hand centres in its own: a one-line worm and a two-line one
	// should look like the same card.
	TextColumnLeft: 12,
	TextInset:      8,
	TextBandTop:    148,
	TextBandBottom: 212,
	TextSize:       17,
	TextLineHeight: 21,
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
	BorderWidth:  3,

	ShowName: true,
	ShowForm: false,

	TextLeft:     12,
	NameTop:      14,
	NameSize:     20,
	NameCentered: true,

	// **One word to a line** *(2026-08-21)*, which is what buys the art box below its room:
	// a two-word ring is two lines of 20pt, and 22 is that size plus the gap that keeps two
	// capitals from touching.
	NameWordPerLine: true,
	NameLinePitch:   22,

	// Scaled with the card, like the enemy's: the artwork is fitted to this box rather than
	// drawn at its own size, so there is nothing here that a smaller card breaks.
	//
	// **The box moved down and shrank when names went to two lines**, from 46..206 to 62..192,
	// and the art did not move: it is square, so its height was already set by the 130-pixel
	// width of the box rather than by ArtMaxH, and it was already being centred at about 61.
	// What changed is that the box now states that, which is what lets a test hold a two-line
	// name off it.
	ArtTop:   62,
	ArtInset: 16,
	ArtMaxH:  130,
}

// Token is a card reduced to the three things a hand is counted on: its **element**, its
// **form** and what it **costs**. 40x56 — a quarter of Hand in each dimension, a sixteenth of
// its area — and it carries no name, no effect text and no picture.
//
// **It is authored rather than derived, and that is the one place it departs from Mini's rule**
// *(2026-08-24)*. `Hand.Scaled(1, 4)` gives an 8px form mark and a 4x1 tick, which is a mark
// with its detail averaged away and a tick that reads as a scratch — the same floor the glyph
// rules describe. The mark therefore stays at Mini's 16, which the drawn art can be halved to
// twice and still be read, and the ticks stay at Hand's own 16x4. **So this is not a small card;
// it is a different object**, which is why deriving it would be claiming something untrue.
//
// **The left column, standing on its own.** Everything it draws — a tinted form mark with the
// cost ticks under it — is exactly what a Hand card puts down its left edge, so a row of these
// is the same reading in the same colours, and nothing here can drift from the card it stands
// for except by that column moving.
//
// The hands panel is the caller: nineteen rungs, each shown as the cards that build it, is a
// hundred-odd cards on one screen, and at Mini's 81x112 that is a panel of cards with no room
// left for the ladder.
var Token = Style{
	Width: 40, Height: 56,

	CornerRadius: 6,
	BorderWidth:  2,

	ShowName: false,
	ShowForm: true,

	// Centred rather than in the corner: with no text column to the right of it there is
	// nothing for a left-aligned mark to line up with, the same reason a ring centres its name.
	FormTop:    6,
	FormSize:   16,
	GlyphInset: (40 - 16) / 2,
	GlyphScale: 1,

	// Hand's own tick, centred under the mark. The gap is 3 rather than 5 so a four-point card
	// still lands its fourth tick inside the border — TestATokenHoldsFourTicks is what fails if
	// it stops doing that, and drawDashes would otherwise drop the tick silently.
	DashLeft:   (40 - 16) / 2,
	DashTop:    26,
	DashWidth:  16,
	DashHeight: 4,
	DashGap:    3,
}
