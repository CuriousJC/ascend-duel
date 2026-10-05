# Sort button art prompt

Three **square icon buttons**, each drawn twice — **at rest** and **pressed** — so six files. Each
file is a whole button: the rounded face and the picture on it, drawn together as one image. They
carry no text; the picture is the button's whole meaning.

The three sit side by side as one group, and exactly one of them is pressed at any moment: the
pressed one is the choice currently in force. So the pressed drawing has to read as **"this one is
on"** next to two at rest, at a glance, at small size.

---

## The prompt

High-resolution digital illustration of **three square game-interface buttons, each in two states**,
one per file, for a dark-fantasy card game. Each is a rounded square with one icon on it.

### The button

- **A square with rounded corners**, the corner radius about **a fifth of the side**. Clearly
  rounded, not a circle and not a pill.
- **No heavy outline.** The edge of the button is a **thin, soft dark line — about a fortieth of the
  side**, a darker shade of the button's own color rather than black. No black drop shadow, no
  black ring, no inked comic-book contour.
- **The face is a gently lit surface**: the gray `core` color, lifting toward its `light` value at
  the top and settling toward its `dark` value at the bottom in a **smooth gradient**, with a faint
  lighter rim along the top edge where the light catches it. It reads as a slightly raised, softly
  rounded tile. No gloss streak, no glass reflection, no metal sheen, no texture, no noise.
- **The button fills the canvas**: its edge touches all four sides, with no padding and no shadow
  falling outside it. Everything outside the rounded corners is **fully transparent**.

Light comes from the **upper left** on the button and on its icon alike.

### The icon

- **Centered, filling about 70% of the button's width**, so there is a clear margin of face all
  round it.
- **Pale ivory, against the face**: the icon is light and the button is mid-dark, so it reads by
  contrast at a glance. Smooth shading within the icon — lighter on the faces toward the upper left,
  a little darker away from it — and a **thin darker contour** of its own, about a fiftieth of the
  side: enough to hold its edge, nowhere near heavy.
- **All three icons are the same ivory.** No icon takes a color of its own — the flame is **not**
  orange or red, the axe is **not** steel-blue. They are one set, told apart by silhouette alone.
- **It sits on the face, not above it**: no cast shadow onto the button, no glow, no halo.
- **Chunky shapes only.** Nothing may be thinner than about **a twelfth of the side**: the buttons
  are used at **44 to 68 pixels** across, so a thin line becomes a gray scratch and a small detail
  disappears. Each icon is recognized by its silhouette first.

### The three icons

| Icon         | What it is |
| ------------ | ---------- |
| `sort-cost`    | **A bold upward arrow**: a thick vertical shaft with a broad triangular head on top, the head clearly wider than the shaft — about twice as wide. Squared-off tail, crisp straight edges. A symbol, not a hand-drawn stroke: no curve, no brush texture, no feathering. Pointing **straight up**. |
| `sort-form`    | **A single-bladed battle axe**, held upright and tilted about 30 degrees, handle running from lower left to upper right with the head at the top. The **blade is one broad crescent on one side of the handle** — a wide curved cutting edge, much taller than the handle is thick — with a short blunt back on the other side. The handle is a **thick straight shaft**, with a plain band where the head meets it. Blade and handle are both the same pale ivory, told apart by shading only. Not a double-bladed axe, not a hatchet with a tiny head, not a pickaxe. |
| `sort-element` | **A single flame**, the classic teardrop-shaped fire: a broad rounded base narrowing to **one main tip** at the top, with **two smaller tongues** licking up either side of it, the left one a little taller than the right. Solid and chunky, like a candle flame drawn as an icon, with **one inner flame** — a smaller teardrop inside the base — shaded a touch darker so the shape has depth. Pale ivory, **not orange, not red, not yellow**, no embers, no sparks, no smoke. |

### The two states

Each icon is drawn **twice, on an otherwise identical button**: same size, same corner radius, same
icon in exactly the same place at exactly the same size. Only the lighting changes, so swapping one
file for the other never makes the button jump.

- **At rest** — the button as described above: a softly raised tile, lit from the upper left.
- **Pressed** — the same button **pushed in and darker**:
  - The face is **clearly darker overall** — the gradient runs from the gray `core` at the top down
    to the gray `dark` at the bottom, so even its brightest part is no lighter than the resting
    button's middle.
  - The **light falls the other way**: a soft shadow along the **inside of the top edge**, as if the
    tile has sunk below the frame around it, and the faint lighter rim moves to the **bottom edge**.
    No lit rim along the top.
  - The icon stays **pale and fully readable**, dimmed only slightly — a little less bright than at
    rest, never gray and never lost into the face.
  - Still no glow, no colored light, no outline change. Pressed is darker and sunken, not lit up.

Side by side, a pressed button between two resting ones should look **pushed down and switched
on**, and the icon on it should still be the first thing the eye finds.

### The button colors

| Word  | dark      | core      | light     |
| ----- | --------- | --------- | --------- |
| gray  | `#32343B` | `#5C606C` | `#9D9FA6` |
| ivory | `#9B8B68` | `#DCCFAE` | `#FFF3D2` |

**All six files are the same button**, with a different icon and a different state. Draw the
resting button once and put each icon on a copy of it; draw the pressed button once and do the same.

### The canvas

Each file is **exactly 256 x 256**, a **transparent** PNG with an alpha channel. Work as large as you
like natively and reduce once to 256 on the way out.

Avoid: text, letters, numbers, card suits, labels, a frame or border around the button, a second
button, a background, scenery, sparkle, glow, gloss, bevel edges, colored icons, and any heavy black
outline.

---

## Before you hand anything back

1. **Reduce each to 44 x 44 and look at it.** Is the icon still the thing it is — an up arrow, an
   axe, a flame? Does the axe still read as an axe rather than a flag or a hammer?
2. **Put the three side by side, one pressed and two at rest, three times over** — each icon taking
   its turn as the pressed one. Is the pressed one unmistakable every time, and is its icon still
   clear?
3. **Toggle each rest/pressed pair.** Nothing should move — only the light changes.
4. **Lay all six over a dark picture and a light one.** Do they read as buttons on both, with no
   black ring around them?
5. **Is everything outside the rounded corners fully transparent**, with no halo?

Name the six files exactly as below, each ending `.png`, and give me a single zip file of everything
so I can download it all at once — never a link per file.

| File                          | Icon    | State   |
| ----------------------------- | ------- | ------- |
| `icon-sort-cost`              | arrow   | rest    |
| `icon-sort-cost-pressed`      | arrow   | pressed |
| `icon-sort-form`              | axe     | rest    |
| `icon-sort-form-pressed`      | axe     | pressed |
| `icon-sort-element`           | flame   | rest    |
| `icon-sort-element-pressed`   | flame   | pressed |
