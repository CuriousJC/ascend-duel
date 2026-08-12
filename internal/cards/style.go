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
// with it and deliberately so: the column is a stack of fixed-size art — a 32-pixel category
// glyph, four dashes, a 64-pixel damage badge — and none of those can be scaled, so what the
// height loses is the empty strip under the badge and nothing else. That strip was ~94px and
// is now ~54, which is still room for the long-press description it is being kept for.
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
	Width: 162, Height: 224,

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
// **Half of the new Hand since 2026-08-11** — 81x112, down from 90x132. `deckStackPitch` in
// internal/screens came down with it, because the pitch is the width less the overlap and a
// pitch wider than the card is a row with gaps in it.
//
// **It shows the name now, and that was the last thing missing.** The overlay's rows
// group by element and the glyph and dashes give phase and cost, so the only fact a card
// was not stating was which concept it is. At 14pt the longest name in the deck —
// "Riposte" — measures 35 pixels against the usable width, so it was never a question of
// room; it was a question of the cards being overlapped so tightly that only 29 pixels
// showed. Widening the row (see deckStackPitch) is what made the space real.
//
// The one thing still absent is the damage badge, and that is forced: a 64-pixel sword
// under a name, a glyph and four dashes does not fit in this height. Damage is a
// function of the concept, so a named card implies it.
//
// The reading order matches Hand deliberately — name centred across the top, then the
// glyph, then the dashes under it — so the two sizes are the same card and not two
// designs.
var Mini = Style{
	Width: 81, Height: 112,

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
// The proportions are Hand's — 40x54 against 162x224 — so the stack reads as the same object
// as the cards in the row, seen smaller. It came down with the rest on 2026-08-11; the strip
// it has to fit did not change, so this one had room to spare either way.
var Stack = Style{
	Width: 40, Height: 54,

	CornerRadius: 4,
	BorderWidth:  0,

	ShowName:     false,
	ShowCategory: false,
	ShowDamage:   false,
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

	ShowName:     true,
	ShowCategory: false,
	ShowDamage:   false,

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

	ShowName:     true,
	ShowCategory: false,
	ShowDamage:   false,

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

	ShowName:     true,
	ShowCategory: false,
	ShowDamage:   false,

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
