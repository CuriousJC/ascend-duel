# Gear art prompt

**One image.** The settings control in the game's chrome corner — a cog, on a 44px square button,
in the bottom-left of every screen. It is the last generated glyph in the game; drawing it retires
`internal/systems/glyphs.go`'s silhouette generator entirely.

**It is a chrome control, not a card mark.** The form marks and cost ticks sit on a pale card face
and say something about the card; this sits on a **dark button face** and says "settings". So it is
hueless where they are colored, and it is the one thing on this page that is not part of that set.

---

## The prompt

High-resolution digital illustration of a single **cog**, rendered as a **user-interface icon**.

Canvas: **exactly 256x256**, transparent background, PNG with an alpha channel. Work as large as
you like natively and reduce once to 256 on the way out. The game draws it at **32 pixels** and
reduces from 256 itself, so anything finer than about an eighth of the canvas width is lost.

Framing: the cog is centred and fills about **90% of the canvas**, with only a thin margin of empty
space around it. It faces the viewer square on, flat, with no perspective and no tilt.

Shape: a classic gear — a round body with **eight square teeth around the rim, four on the axes and
four on the diagonals**, and a **round hole through the middle** about a third of the body's width.
The tooth count is not a detail: at 32 pixels a gear is recognized by *how many* teeth it has
before any one of them is legible, and four teeth read as a compass rose rather than as a cog.
Teeth are blocky rectangles, not tapered wedges — a taper is the part that disappears first.

**Nothing may be thinner than a twelfth of the canvas width.** Thin teeth, a thin rim or a delicate
hub all vanish at card size or survive as a single gray pixel that reads as a scratch. Make the
teeth chunky, the rim thick, and the hole big enough to still be a hole at 32 pixels.

Color: **neutral gray and nothing else.** Mid-gray across the body, lighter gray on the faces
turned toward the upper left, darker gray on the faces turned away, a near-white hot spot where the
light catches hardest, and a **heavy near-black contour** all the way around the outside and around
the inside of the hole. **No hue at all** — not a warm gray, not a cool gray, not a tint of
anything, and no metal color: this sits on a dark button and its job is to be a shape, not a
material.

**Make the contour thick: at least 8 pixels of the 256, and 10 is better.** It is reduced by a
factor of eight, so a thin outline averages away and leaves a soft shape with nothing holding it
against the button face.

Light: from the **upper left** — highlights on the upper-left faces, shadow falling away to the
lower right. Everything in this game is lit from that corner.

Shading: smooth and continuous within each plane, with real material weight rather than a flat
vector fill. Anti-aliasing, soft gradients and blended edges are all wanted. Do not posterize, do
not dither, do not quantize to a block grid, and **do not draw pixel art**.

Background: **fully transparent everywhere the cog is not.** No border, no frame, no rectangle, no
box, no panel, no circle behind it, no plaque, no button, no ground, no cast shadow, no glow, no
vignette. The game draws the button underneath, so anything at all behind the cog becomes a visible
patch sitting on that button. **The only dark lines in the image are the two contours on the cog's
own edges.**

Avoid: text of any kind, other machinery, screws, bolts, scenery, props, motion lines, sparkle,
and every kind of clutter. One cog, nothing else.

## Before you hand it back

1. **Reduce it to 32x32 and look at it.** That is the size it is used at. Count the teeth: there
   should still be eight. If the rim has gone soft or the dark ring has vanished, the contour was
   too thin.
2. **Is the hole still a hole** at 32 pixels, or has it filled in?
3. **Is the background actually transparent** — not black, not white, not a dark square?
4. **Is it genuinely hueless?** Hold it at arm's length: no part of it should read as warm or cool.

Name it `gear.png` and give me a link I can click to download it.
