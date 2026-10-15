// Package motifs embeds every creature's picture and every duel's backdrop, and files them into
// assets.LoadImageData when it is imported. **Import it for its side effect and nothing else** —
// `import _ "github.com/curiousjc/ascend-duel/assets/motifs"` — and only from a program that draws
// a creature or a room: the game and the roster tools.
//
// **It is its own package because it is most of the embedded weight.** Everything that links
// `assets` — every test binary and every review sheet — would otherwise carry the whole roster's
// portraits and backdrops whether it draws one or not. A program that does not import this package
// draws the placeholder wherever a creature's key is named and the plain backdrop wherever a room's
// is, which is what a picture nobody has made looks like everywhere.
//
// **One directory per motif, mirroring `data/motifs/<motif>/`**, so what a motif has been drawn
// is one folder to open: `goblins/creature/` and `goblins/backdrop/`. The backdrop placeholder sits
// at the top of the tree, `default-background.jpg`, because it belongs to no motif; a creature with
// no picture draws the shared placeholder in `assets/game/`.
//
// **Keyed by filename stem, and the directory is not part of the key.**
// `goblins/creature/goblins-serf-fire.png` is `goblins-serf-fire`, which is what
// `data.MotifRecord.ArtKey` builds out of the record's `Art` field and the element the realm dealt
// it as; `goblins/backdrop/goblins-outer-tinker-studio-fire.jpg` is
// `goblins-outer-tinker-studio-fire`, which is what `data.Backdrop.ArtKey` builds the same way. So
// a file can be refiled without touching a record, and two files anywhere in the tree sharing a
// stem are one lookup with two answers — `assets.EmbedTree` refuses that at load.
//
// **Embedded as a tree rather than one var each, which is a deliberate exception to the
// three-edit rule** in assets/embed.go. That rule — the file, an //go:embed var, a map entry — is
// right for a handful of named assets and absurd for a roster of this size: it would be hundreds of
// lines no reviewer could check, drifting the first time a creature was renamed. The consequence,
// stated because it is the thing the rule was protecting: **a picture's key is tied to its
// filename**, so renaming one means editing the `Art` field of the record that names it.
//
// **A missing picture falls back to a placeholder**, so a blank face means art nobody has made
// rather than a name nobody spelled right, and the plain backdrop in a duel means no backdrop
// fits that room. One placeholder of each kind for every motif: a per-motif placeholder is a
// picture somebody has to draw before the motif can be looked at.
//
// **A creature is a PNG and a backdrop is a JPEG.** A backdrop is opaque edge to edge at the
// screen's own 1920x1080, and a painted scene is about a sixth the size that way. Both are handed
// out as bytes and decoded by whoever draws them, because a decoded 1920x1080 picture is eight
// megabytes and a run only shows a handful.
//
// Provenance: generated from the prompts under `docs/art/`, like everything else in `assets/`.
package motifs
