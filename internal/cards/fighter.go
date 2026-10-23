package cards

// The block along the bottom of the opponent's card: its two orbs and the health bar.
//
// **The orbs stand in the portrait's bottom corners, on the bar**, DMG left and AP right — the two
// figures the fighters share, in the pictures the duelist card wears them in. No ground is painted
// under them: an orb carries its own contour and its figure its own outline, so a scrim would be a
// second surface on a card whose face is a picture.
//
// **The two fighter cards share the bar**: the same height, inset and fraction on both, so the two
// are read across the table against each other without measuring.
//
//	162  the orbs            162..246  DMG left, AP right, standing on the bar
//	250  the health bar      250..274  with the fraction written across it
//	276  the inside of the bottom border
//
// **The opponent's portrait is composed against the top 200x200**, which
// `docs/art/creature_art_prompt.MD` states as the picture's own constraint; the orbs reach above that,
// into the corners, where nothing the creature is recognized by lives.
const (
	FighterBlockTop = 200

	enemyOrbSize  = duelistOrbSize
	enemyOrbInset = 12
	enemyOrbY     = fighterBarTop - 4 - enemyOrbSize/2

	// The duelist card's square of orbs: each orb's side, the pitch between centers both ways, and
	// the first row's center. Its room row stands under the square and its bar under the rooms.
	duelistOrbSize  = 84
	duelistOrbPitch = 88
	duelistOrbTop   = 52

	fighterBarInset  = 15
	fighterBarTop    = 250
	fighterBarHeight = 24
	fighterBarText   = 20
)
