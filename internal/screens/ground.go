package screens

import "image/color"

// The table everything on a between-fight screen is drawn on.
//
// **These live here rather than on the combat screen because every scene stands on them.**
// They were declared in combat.go, which meant the post-battle screen — and the shop and the
// room choice after it — had to reach into the combat screen's own file to find out what
// colour the game is. The ground is not the combat screen's; it is the game's.
//
// Anything drawn on a surface of its own — a card, a panel, a button — takes that surface's
// own colours instead. These two are only for what is painted straight onto the table.

// screenGround is what every screen is painted on, and **it went cream on 2026-08-14**, from
// the {50,50,50} dark grey the combat screen had been since it existed.
//
// **Everything drawn straight onto it had assumed a dark ground**, which is the cost of the
// change and the reason it is a named constant now rather than a literal in Draw. Three
// figures were white and are now `groundInk`; the action-point bar's empty cells were
// `ColorAtStrength(apBarColor, 20)`, which scales toward black and therefore came out *louder*
// than the filled cells on a light ground — exactly the failure `systems.ColorToward` was
// written for; and the ring row's backing had to stop being one step *lighter* than the ground
// and become one step darker.
//
// **It is deeper than the cards stand on** — `cards.Surface` is {240,239,234} and the
// fight log's panel fill is {234,230,224} — because a card, a panel and the table cannot all be
// the same off-white or the objects stop having edges. The warmth is where the separation
// comes from: the ground is the yellowest of the three.
var screenGround = color.RGBA{R: 226, G: 208, B: 176, A: 255}

// groundInk is for text written straight onto the ground rather than onto a card, a pane or a
// button — the action-point figure, the draw pile's count, the ring row's fraction, the
// post-battle screen's heading. Near black and slightly warm, so it belongs to the cream rather
// than sitting on it.
//
// Anything drawn on a surface of its own takes that surface's ink instead; this is only for what
// has nothing behind it.
var groundInk = color.RGBA{R: 44, G: 40, B: 34, A: 255}
