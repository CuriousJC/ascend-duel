// Package assets embeds every image, font and sound into the binary and hands them out.
//
// # Adding one
//
// A named asset is three edits: the file, an //go:embed var, and a map entry in the loader. The
// map key is the lookup name used everywhere else and is deliberately independent of where the
// file sits — assets/ is grouped by what a file is for, and tying a key to a path would mean a
// data migration every time something was refiled.
//
// # The exception, and what it costs
//
// The enemy portraits are a family rather than named assets. There are 96 of them, so a glob pulls
// the directory in as an embed.FS and the loader keys each by filename stem. The consequence is
// exactly what the three-edit rule protects against: a portrait's key is tied to its filename, so
// renaming one means editing data/enemies.json. That is the price of not hand-maintaining 192
// lines nobody could review. Reach for the glob only when a set of files is being added.
//
// # Two kinds of loader, and why
//
// LoadAssets and LoadFonts hand back Ebitengine types. LoadImageData and LoadFontData hand back
// raw bytes, because internal/cards renders into a plain Go image with no graphics context and
// tools/cardsheet calls it with no window. Both read the same embedded bytes, so the game and the
// contact sheet cannot end up showing different pictures.
//
// The portraits are bytes only. Decoding 96 at startup would cost around 20 MB of resident memory
// for pictures most runs never show.
//
// # Provenance
//
// This game is source-available and will be sold, so every asset needs a provable licence. The
// enemy portraits are PVGames creature art from a Humble bundle whose licence permits shipping
// inside a game; everything else here is first-party or generated at runtime. "Found it online" is
// not sufficient. That constraint is the reason the glyphs and the score are generated rather than
// loaded — see internal/systems and internal/music.
package assets
