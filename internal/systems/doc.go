// Package systems is the behaviour for the structs in internal/models, plus the procedural art
// generator.
//
// The split is deliberate: a widget is a plain struct in models and a pair of Update* and Draw*
// free functions here, taking (gs, ...). Nothing in models has a method that draws.
//
// # Colour
//
// ColorAtStrength and ColorToward are the two ways a colour is dimmed, and they are not
// interchangeable. ColorAtStrength scales toward black, which reads as quieter only against a dark
// ground; ColorToward moves a colour toward whatever it actually sits on. The combat screen's
// ground is cream, so on it ColorAtStrength is the exception rather than the default, and reaching
// for it to dim something drawn straight onto the table is a bug waiting to be seen. It still
// governs buttons, because a button paints its own dark face and its label is white.
//
// The rule both serve: a widget names the colour it wants at full strength and scales down from it
// for its other states. Scale a colour, never add to it — adding a fixed step to every channel
// walks a saturated colour toward white, and a channel already near 255 has nowhere to go.
//
// # Glyphs
//
// glyphs.go generates the pixel-art glyphs in code rather than loading them from files, because
// generated art has no provenance question at all — which is the whole reason to prefer this
// pattern for interface art in a game that will be sold.
//
// It is a generator, not a bitmap. A glyph is a filled silhouette described by horizontal spans;
// the rim is derived by asking which filled pixels touch empty space, and the interior shading is
// computed from where a pixel sits across its row and down the sprite. Nothing is hand-placed, so
// a shape can be nudged without repainting it.
//
// Constraints that fall out of the technique and drive every span in the file:
//
//   - Nothing in a silhouette may be thinner than about five pixels. The derived rim takes one
//     pixel off each side, so a three-pixel crossguard renders as two rows of outline around one
//     row of metal and reads as a scratch.
//   - A glyph cannot be resized, so a smaller one is a different drawing. The rim is derived one
//     pixel thick however big the shape is, so a third-size copy is a third-size copy of its
//     outline with nothing inside.
//   - GlyphKind is append-only. The cache keys on the ordinal, so inserting a kind mid-enum
//     silently re-points every existing entry.
//   - SizeOf is the authority on how big one is, never an assumed 64.
//   - Glyphs carry a five-value palette and are the deliberate exception to the scale-one-colour
//     rule, because a bevel cannot be made from one colour scaled down. They are drawn untinted; a
//     disabled card dims them by alpha, so the shading survives and only the weight changes.
//
// RenderGlyph returns a plain Go image and is free of Ebitengine on purpose: creating an
// *ebiten.Image needs a graphics context, and the review tool has no window. Glyph wraps and
// caches it for the game.
//
// Run `go run ./tools/glyphsheet` after changing any of it. The sheet is committed so a change to
// a silhouette shows up in review as a picture, and a stale sheet is worse than none because it is
// a picture that lies.
package systems
