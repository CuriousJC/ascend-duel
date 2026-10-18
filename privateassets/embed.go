// Package privateassets carries the music loops, which are kept out of git and synced from a
// private S3 bucket instead.
//
// # Why it is not in assets/
//
// **A separate directory is what enforces the split**, rather than a rule about one corner of
// `assets/` that somebody has to remember. `git add assets/` cannot sweep up a loop, because a
// loop is not under `assets/`.
//
// It is a package rather than a bare directory because `//go:embed` cannot reach outside the
// directory its Go file sits in, so a top-level split *is* a package boundary — which turns out
// to be useful rather than a cost: nothing reaches these files by accident through the asset
// loader, since a caller has to name this package.
//
// # An empty directory is a supported state
//
// A clone with no bundle synced **builds**, and the game plays silent. That is not a courtesy:
// it is every clean clone, every fork and every pull-request CI run, and a build that failed on
// them would make the repository unbuildable for everyone who does not have the bucket.
//
// **It is also how a release could ship quiet without anything going red**, which is why
// `tools/privateassets -require` runs in both release build jobs before anything is compiled —
// gated, with the sync before it, on the SOUNDS_BUCKET repository variable, so a release cut
// before the bucket exists ships silent rather than failing. **The score is here too**, in the
// `custom` pack, so a build without the bundle has no sound at all.
//
// See privateassets/README.md for how a batch is filed and where the bundle lives.
package privateassets

import (
	"embed"
	"io/fs"
	"log"
	"path"
	"strings"
)

// **The directory is embedded, never globbed**, and that is the single most breakable decision
// here. A pattern like `audio/*.ogg` matching no file is a *compile* error, so a glob would turn
// "this clone has no sounds" into "this clone does not build". Embedding `audio` matches the
// committed manifest whatever else is beside it, so the build always resolves.
//
//go:embed audio
var audio embed.FS

// audioExts is what the game can play: MP3, Ogg Vorbis and WAV, which Ebitengine decodes, and
// MIDI, which internal/music synthesizes — the score, `duello.mid`. It matches the filter in
// tools/privateassets, and the two have to stay in step — a file the
// tool records and this skips would be a manifested sound that is not in the binary.
var audioExts = map[string]bool{".ogg": true, ".wav": true, ".mp3": true, ".mid": true}

// Audio returns the raw bytes of every sound that is present, keyed by **filename** —
// `audio/dark-fantasy/dark-fantasy-01.wav` is `dark-fantasy-01.wav`.
//
// **A folder under audio/ is a pack and is walked into**, so a batch can be filed as one
// directory and still be told apart from the others on disk. **The folder never reaches the key**:
// a track is played by its stem, wherever it is filed, which is the same cut assets.EmbedTree makes
// for the motif pictures.
//
// **The extension is kept because it is what says how to decode the bytes.** The name a caller
// plays a sound by is still the stem, and music.LoadTrack is where that cut is made: it needs
// both halves, and a map that had already thrown the extension away would leave it sniffing
// bytes to work out what it had been handed.
//
// Bytes rather than a decoded stream, for the reason assets hands back bytes: the
// decoder belongs to whatever plays it, and this package sits below anything that might.
//
// **Anything that is not decodable audio is skipped rather than keyed** — the manifest, a stray
// original somebody left beside its converted copy — so filing a source file next to its
// conversion cannot quietly put an undecodable blob in the map.
//
// **A stem claimed twice is fatal, across folders as well as within one.** `hit.ogg` and
// `hit.wav`, or `a/hit.ogg` and `b/hit.ogg`, are two files that a caller can only ask for as
// `hit`, and picking either silently is worse than refusing to start.
//
// **Absence is never fatal.** An empty map is a build that has not synced the bundle.
func Audio() map[string][]byte {
	sounds := make(map[string][]byte)
	// stems is only for the collision check; the map itself is keyed by filename.
	stems := make(map[string]string)

	err := fs.WalkDir(audio, "audio", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		ext := strings.ToLower(path.Ext(name))
		if !audioExts[ext] {
			return nil
		}
		raw, err := audio.ReadFile(p)
		if err != nil {
			return err
		}
		stem := strings.TrimSuffix(name, path.Ext(name))
		if first, taken := stems[stem]; taken {
			log.Fatalf("%s and %s share the stem %q; one name cannot mean two files", first, p, stem)
		}
		stems[stem] = p
		sounds[name] = raw
		return nil
	})
	if err != nil {
		// Impossible in a built binary, since the files are compiled in. A panic is the honest
		// response rather than a silently empty set.
		log.Fatalf("failed to read the embedded audio directory: %v", err)
	}

	return sounds
}
