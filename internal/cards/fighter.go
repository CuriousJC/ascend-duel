package cards

import (
	"image"
	"image/color"
)

// The block along the bottom of a fighter card: the badge row, the figure under it, and the
// health bar.
//
// **A ground is painted for type and for nothing else.** The card that writes a figure under its
// portrait gets a scrim from just above that figure to the bottom edge, because light type on a
// picture with no ground under it is a figure you have to hunt for. The duelist card writes no
// figure there, gets no scrim, and stays off-white — a black stripe across it would be a second
// surface on a card that already has one. That difference falls out of the band being derived from
// the type it covers rather than from a field either card could set.
//
// **The two cards share the badge seat and the bar and nothing else.** Those are the rows read
// *across* the table against each other, so a bar at a different height on each would turn a
// comparison into an act of measurement. What sits between them is each card's own business: the
// opponent writes its DMG there because it has nowhere else to put a figure, and the duelist has a
// ladder of stat rows up top and leaves it empty.
//
// The offsets exist once, here, which is what stops an edit to one card moving only that one —
// see TestTheTwoFighterCardsShareTheirHealthGeometry, which holds the property from the other
// side.
//
//	200  the badge row       200..220  statuses on a creature, shields on the duelist
//	216  the scrim's top edge          on the opponent's card only, six above the figure
//	222  the DMG row         222..247  the opponent's only figure; the duelist writes none here
//	250  the health bar      250..274  with the fraction written across it
//	276  the inside of the bottom border
//
// **The opponent's portrait is composed against what is left**, which is 200x200 rather than the
// whole card. `docs/art/creature_art_prompt.MD` states that as the picture's own constraint: the
// scrim covers most of the strip and the badge row sits on bare picture above it, so nothing the
// creature is recognized by may live down there either way.
const (
	FighterBlockTop = 200

	fighterBadgeTop  = 200
	fighterBadgeSize = 20
	fighterBadgeGap  = 8

	// **The figure is bigger than a stat row.** It is the only number on the opponent's card and
	// it is read across the table against the duelist's own DMG, so it is set above the size the
	// duelist's ladder uses rather than matching it — one figure on a picture can afford what
	// five stacked figures cannot.
	fighterStatTop  = 222
	fighterStatSize = 23

	fighterBarInset  = 15
	fighterBarTop    = 250
	fighterBarHeight = 24
	fighterBarText   = 20
)

// drawPortraitStat writes the one figure a card with a portrait has nowhere else to put.
//
// **It is a row rather than an entry in Spec.Stats** because it is not on that ladder: the stat
// rows stack from the top of the card at one pitch, and this sits in the block at the bottom with
// a badge row above it and the bar below.
func drawPortraitStat(dst *image.RGBA, s Spec, st Style, f *Faces, ink func(color.RGBA) color.RGBA) error {
	if st.PortraitStatSize <= 0 || (s.PortraitStat.Label == "" && s.PortraitStat.Value == "") {
		return nil
	}

	if err := drawText(dst, f, st.PortraitStatSize, s.PortraitStat.Label,
		st.TextLeft, st.PortraitStatTop, ink(LabelInk)); err != nil {
		return err
	}
	value := NumberInk
	if s.PortraitStat.ValueInk.A > 0 {
		value = s.PortraitStat.ValueInk
	}
	return drawTextRightAligned(dst, f, st.PortraitStatSize, s.PortraitStat.Value,
		st.Width-st.TextLeft, st.PortraitStatTop, ink(value))
}
