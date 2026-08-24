// Package cards draws an action card to a plain Go image.
//
// **It creates no Ebitengine images**, which is the whole reason it exists as its own
// package. `systems.RenderGlyph` already established the pattern — art that returns an
// `image.Image` can be rendered by the game *and* by a command-line tool with no window,
// so a card can be reviewed by opening a file instead of by launching the game and
// dealing yourself the hand that shows it. `tools/cardsheet` is that tool.
//
// The important consequence is that the sheet cannot lie. Both callers run this same
// code and get the same pixels; there is no second drawing path to drift out of sync,
// which is what went wrong with the first version of the glyph sheet when it previewed
// glyphs at a scale the cards did not use.
//
// # What a card looks like, and what changed
//
// The surface is a constant off-white for every card and **the left column carries the
// element** — the tinted form mark in the corner and the cost ticks under it *(owner's
// call, 2026-08-23)*. The border carried it from
// 2026-08-09 until then, which itself reversed the decision recorded in CLAUDE.md's
// colour section on 2026-08-03, where the surface was the element. Three things follow
// from where it has landed:
//
//   - **The border is state and nothing else now.** Every card of every element draws the
//     same neutral grey, and rest, selected, dragging and disabled are distances from it
//     toward the surface. See borderBase for why the swap was made: the corner mark is the
//     thing a hand is counted on, so it is what the one remaining colour channel buys the
//     most on. BorderOf still holds the element colours and is still what the mark, the
//     row labels and the arithmetic panel read.
//   - **The border and the ticks share one state switch.** Spec.atState carries a colour
//     from full strength to whatever the card's state wants; the border hands it the
//     neutral grey and the ticks hand it the element. They are different colours and the
//     same mark, so they must dim and light together.
//   - **Ring keeps its pink.** Pink was never an element — it is the "you cannot play
//     this" signal — so it survives a change that neutralises the four element borders.
//   - **The form marks are drawn art, tinted rather than repainted.** tintInk maps each
//     pixel's own brightness onto a ramp between a dark and a light version of the
//     element's colour, so the drawing keeps its outline and its bevel and only the hue
//     moves. A flat silhouette in the element colour would throw away the interior detail
//     that made drawn marks worth having over generated ones.
//
// # Rounded corners are rasterised here, not masked
//
// **This is the only rounding approach in the tree** *(2026-08-24)*. The screen used to
// round with `CreateRoundedRecMask` + `ebiten.BlendSourceIn`, and for a while the two
// coexisted — that path could never be used here, since it takes an `*ebiten.Image`, its
// body is `vector.DrawFilledCircle`, and `BlendSourceIn` is a GPU blend mode, none of
// which exist without a graphics context. Being window-free is the requirement that makes
// the review tool possible at all, so it wins.
//
// The mask path went when both fighters became cards and their health bars came in here
// with them, so nothing on screen needs it any more. **A new rounded shape belongs in
// shape.go**, whatever is drawing it: a second GPU-side rasteriser would put two
// silhouettes of the same corner back in the tree, and the one that cannot be reached
// without a window is the one that would spread.
//
// Corners are hard-edged — no antialiasing — because the cards were already drawn that
// way (`vector.DrawFilledRect(..., false)`) and because the glyphs sitting on them are
// 1:1 pixel art with a one-pixel rim. An antialiased card edge around pixel-art contents
// reads as two different pictures.
//
// # Why it creates no Ebitengine images
//
// Same pattern and same reason as systems.RenderGlyph: it is what lets tools/cardsheet render a
// card with no window, and it is why text is set through golang.org/x/image rather than
// Ebitengine's text/v2, which can only draw into an *ebiten.Image. Both the game and the contact
// sheet go through this package, so the sheet cannot drift from what is drawn.
//
// internal/screens/card_art.go is the bridge and holds the cache — rendering writes every pixel in
// Go and is far too slow to do per frame.
//
// # Spec must stay comparable
//
// The screen's cache keys on the whole struct. That is why Stats is a fixed array and not a slice.
//
// Spec is plain data — a name, a category, a cost, an element, optional artwork, a state — rather
// than a rules type, so the sheet can draw combinations the rules cannot produce: a border colour
// nothing uses, a ring.
//
// # The border carries the element, not the surface
//
// A card is a constant off-white surface with a thick coloured border. Three things follow and
// have each been re-broken once:
//
//   - A near-white border on an off-white card is invisible, which is why basic is a mid grey and
//     a test fails if it is set back to a near-white.
//   - systems.ColorAtStrength is the wrong tool on a light card. It scales toward black, so a
//     border scaled down comes out darker than the surface and therefore louder than the live card
//     beside it. Use systems.ColorToward. Card state is expressed as distance to the surface.
//   - Cost is dash marks and the form is a corner mark, never text and never a numeral. Every card
//     in the game runs 1..3; a fourth dash grows the stack further down the card and is a layout
//     change rather than a bigger number. A card declares its own cost now, so nothing stops a
//     data file writing 5 — which is a reason to read this before authoring one, not a reason for
//     the renderer to clamp.
package cards
