# Achievement icon prompt

High-resolution digital illustration of **square achievement icons**, one per file, for a
dark-fantasy card game. Each is a square of **polished jade** with one pale ivory icon on it — the
same stone and the same icon treatment as the game's own icon buttons, so the achievements read as
one set with the interface.

No text anywhere in the image — no title, no label, no caption, no number, no letter, no signature,
no watermark.

## The stone

- **Jade fills the whole canvas, edge to edge**, with **square corners**. No rounded corners, no
  transparency anywhere, no border, no frame, no padding: every pixel of the square is stone or icon.
  The icon is shown by Steam as well as by the game, and both put their own edge round it.
- **Polished jade**: smooth and dense, with **faint soft clouding** inside it — paler and darker
  drifts of the stone's own color, like real carved jade. The clouding is gentle and low-contrast; it
  must never compete with the icon. Not glass, not plastic, not metal.
- **One smooth gradient**: the stone's `light` at the top, its `core` across the middle, its `dark`
  at the bottom. A soft satin polish, as if lit from the upper left.
- **Plain apart from the icon.** Do not draw any line, arc, band, rim, ridge, groove, highlight
  stripe, inner border or second outline anywhere on the stone. No hard gloss, no glass reflection,
  no sparkle.
- **The same stone on every file.** Draw the jade square once and put each icon on a copy of it, so
  the set reads as one.

## The icon

- **Centered, filling about 70% of the width and height**, with a clear margin of stone all round.
  Nothing touches or runs off an edge.
- **Pale ivory, against the stone**: the icon is light and the stone is mid-dark, so it reads by
  contrast at a glance. Smooth shading within the icon — lighter toward the upper left, a little
  darker away from it — and a **thin darker contour** of its own, about a fiftieth of the side:
  enough to hold its edge, nowhere near heavy.
- **Every icon is the same ivory.** No icon takes a color of its own — a flame is not orange, a blade
  is not steel, a card is not red. Where a brief says a part is different from the part beside it,
  tell the two apart by **shading and contour**: a slightly darker ivory, a line between them, never
  a hue.
- **It sits on the stone, not above it**: no cast shadow, no glow, no halo.
- **Chunky shapes only.** Nothing may be thinner than about **a twelfth of the side**: the icon is
  judged at **64 x 64**, which is how Steam's pop-up and the game's achievements page both show it,
  so a thin line becomes a scratch and a small detail disappears. Each icon is recognized by its
  silhouette first.

**It must also read in grayscale.** Every icon is shown with its color taken out, as the locked
version of the achievement. Ivory on jade holds by lightness, which is why this treatment is used —
so keep the icon light and the stone mid-dark, and do not let a dark part of the icon sink into the
stone.

## The colors

| Word  | dark      | core      | light     |
| ----- | --------- | --------- | --------- |
| jade  | `#1E4636` | `#3D7A62` | `#86BFA4` |
| ivory | `#9B8B68` | `#DCCFAE` | `#FFF3D2` |

**Jade** is a deep, slightly blue-leaning green: not emerald, not grass green, not lime.

## The canvas

Each file is **square, exactly 1:1**, an opaque PNG. Work as large as you like natively; it is
reduced to 256 x 256.

Avoid: text, letters, numbers, card suits, pips, labels, colored icons, rounded corners,
transparency, a frame or inner border, a rim highlight, a background other than the stone, scenery,
sparkle, glow, hard gloss, bevel edges, a drop shadow, and any heavy black outline.

## The subject

The block pasted below this prompt is one record from `data/achievements.json`, or a list of them.
With a list, make one image per record, each on its own brief — nothing carries over from one record
to the next except the stone.

**Read `Draw` and nothing else as the subject.** Every other field — `DisplayName`, `Description`
and the rest — is context for what the achievement is and is never to be rendered. If `Draw` is
empty, stop and say so rather than inventing a subject.

**Draw only the achieved icon.** The unachieved one is made from it by taking the color out, so do
not draw a gray, dimmed or locked version.

## Before you hand anything back

1. **Reduce each to 64 x 64 and look at it.** Is the icon still the thing its brief names?
2. **Take the color out of each.** Is the icon still plainly lighter than the stone, all the way
   round its edge?
3. **Is every pixel opaque, and does the jade reach all four corners?**
4. **Lay them side by side.** Is it one stone, one ivory, and one weight of contour across the set?

Name each image after its record's own key — the `APIName` value, `first-steps.png` — and give me
the file itself if there is only one, and a single zip file of everything if there is more than one,
so I can download it all at once — never a link per file.
