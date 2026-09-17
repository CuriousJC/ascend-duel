// Package systems is the behavior for the structs in internal/models, plus the door authored art
// reaches the drawing through.
//
// The split is deliberate: a widget is a plain struct in models and a pair of Update* and Draw*
// free functions here, taking (gs, ...). Nothing in models has a method that draws.
//
// # Color
//
// ColorAtStrength and ColorToward are the two ways a color is dimmed, and they are not
// interchangeable. ColorAtStrength scales toward black, which reads as quieter only against a dark
// ground; ColorToward moves a color toward whatever it actually sits on. **Every screen's ground is
// light** — a light slate blue since 2026-09-07, cream before that — so ColorAtStrength is the
// exception rather than the default, and reaching for it to dim something drawn straight onto the
// table is a bug waiting to be seen. It still governs buttons, because a button paints its own dark
// face and its label is white.
//
// The rule both serve: a widget names the color it wants at full strength and scales down from it
// for its other states. Scale a color, never add to it — adding a fixed step to every channel
// walks a saturated color toward white, and a channel already near 255 has nowhere to go.
//
// # Art
//
// artmark.go fetches authored art by asset key and reduces it to whatever size a caller asks for.
// It is the whole of how a picture reaches the drawing.
//
// **There was a generator here until 2026-09-16** (owner's call). glyphs.go described pixel-art
// silhouettes in code — horizontal spans, a rim derived by asking which filled pixels touch empty
// space, shading computed from where a pixel sat in its row — because generated art has no
// provenance question, which is a real argument in a game that will be sold.
//
// What retired it was every one of its pictures becoming a drawing. The form marks went from four
// tinted silhouettes to one authored picture per form per element plus a neutral set plus a tick
// per element, which is a multiplication rather than a list; the rest had drawn nothing for a
// month; and the settings cog, the last one standing, became assets/game/gear.png. The provenance
// argument is answered differently now: the art comes from a prompt this repo owns, in docs/art/,
// so there is still nothing to clear.
//
// Two properties of the old technique are worth remembering, because they are why it could not
// hold this set:
//
//   - A generated glyph could not be resized. The rim was derived one pixel thick however big the
//     shape was, so a smaller one was a *different drawing*. A painting has interior detail to
//     average and survives being reduced from 256 to 32 and to 16.
//   - Its pictures were keyed by an append-only enum indexing a cache. Thirty-one marks would have
//     been thirty-one ordinals naming pictures the rules know nothing about, where an asset key is
//     a string the caller builds from a card's own form and element.
//
// ArtMark returns a plain Go image and is free of Ebitengine on purpose: creating an *ebiten.Image
// needs a graphics context, and the review tools have no window. ArtMarkImage wraps and caches it
// for a screen.
//
// Run `go run ./tools/marksheet` after changing any art. The sheet is committed so a change shows
// up in review as a picture, and a stale sheet is worse than none because it is a picture that
// lies.
package systems
