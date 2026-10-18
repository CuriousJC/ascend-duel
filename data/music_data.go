package data

// The music catalog: **every sound the game ships, where it came from, and how it was made
// game-ready.**
//
// The audio itself is not in git. The owner's library — every pack as it was delivered — sits
// outside the repository, and `privateassets/mgmt/build-game-ready.ps1` builds each record here into
// one flat `game-ready/` folder beside it: `Source` read from the library, `GainDB` applied,
// encoded at `Quality`. That folder is what is mirrored into `privateassets/audio/` and what a
// release fetches, so **only what this file names is ever built into the game**.
//
// **Two kinds of field, and one file on purpose.** `Packs`, `Source`, `GainDB`, `Quality` and
// `Screens` are authored. `Bytes` and `SHA256` are written by `go run ./tools/privateassets -write`
// from the game-ready file, and `-require` refuses a release whose files do not hash to them — the
// bucket holds the current set rather than a history, so the hash committed here is what stops an
// older commit shipping a file re-made since. The tool rewrites the whole file in this struct's
// field order and touches only those two fields.
//
// **Every gain is the record's own.** There is no default level: a track is made game-ready at the
// gain written on it, and a record without one is refused, so every figure is one somebody chose
// and can review. -1.0 dBFS true peak is the level the score sits beside — see
// privateassets/README.md — and `build-game-ready.ps1 -Measure` says what gain would put a source
// there.
//
// **This is the one catalog that carries provenance**, because bought audio is the one asset with a
// supplier to credit: a pack's `Name` is the purchase it came from, and is what a credits screen
// can read.
//
// **`Screens` is the pool a track may play in**, named as `state.ActiveScreen.String` writes them
// and checked there, since `data` cannot see that vocabulary. Which track of a pool a screen plays
// is still `internal/game/score.go`'s table, and a test holds every choice it makes to this list.

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"
)

//go:embed music.json
var musicJSON []byte

// MusicCatalog is the whole of data/music.json.
type MusicCatalog struct {
	// Packs are the purchases a track can come from, plus `custom` for the game's own.
	Packs []MusicPack `json:"Packs"`

	// Tracks are the sounds that ship, in the order the file writes them.
	Tracks []MusicTrack `json:"Tracks"`
}

// MusicPack is one source of audio.
type MusicPack struct {
	// Pack is the key: the pack's folder in the library, which every Source under it starts with.
	Pack string `json:"Pack"`

	// Name is the purchase as it is credited.
	Name string `json:"Name"`

	// Note is anything about the pack as a whole worth keeping beside it.
	Note string `json:"Note,omitempty"`
}

// MusicTrack is one shipped sound.
type MusicTrack struct {
	// File is the game-ready filename, extension included. **Its stem is the track's name** —
	// what score.go plays it by and what privateassets.Audio keys it under.
	File string `json:"File"`

	// Pack names the record in Packs this track came from.
	Pack string `json:"Pack"`

	// Source is the delivered file, as a path under the library's `sounds/` — `<pack>/<file>`.
	Source string `json:"Source"`

	// GainDB is applied to Source on the way to game-ready, in decibels. Required for every track
	// but the MIDI score, whose level is the synthesizer's; refused on that.
	GainDB *float64 `json:"GainDB,omitempty"`

	// Quality is the libvorbis quality, 0 to 10, the track was encoded at. Required whenever the
	// track is encoded — a source in another format, or any gain — and refused on a straight copy.
	Quality *int `json:"Quality,omitempty"`

	// Screens are the screens this track may play on.
	Screens []string `json:"Screens"`

	// Bytes and SHA256 describe the game-ready file. **Generated** by tools/privateassets -write.
	Bytes  int64  `json:"Bytes"`
	SHA256 string `json:"SHA256"`
}

// MusicExts is what the game can play: MP3, Ogg Vorbis and WAV, which Ebitengine decodes, and MIDI,
// which internal/music synthesizes. privateassets.Audio and music.decoders take the same set.
var MusicExts = []string{".ogg", ".wav", ".mp3", ".mid"}

// Name is the track's name: its file's stem.
func (t MusicTrack) Name() string { return strings.TrimSuffix(t.File, path.Ext(t.File)) }

// Encoded says whether making the track game-ready re-encodes its source, rather than copying it.
func (t MusicTrack) Encoded() bool {
	return path.Ext(t.File) != path.Ext(t.Source) || (t.GainDB != nil && *t.GainDB != 0)
}

// LoadMusic parses the embedded catalog.
func LoadMusic() MusicCatalog { return ParseMusic(musicJSON) }

// ParseMusic parses and validates a music catalog. tools/privateassets reads the file off disk
// through it, so the tool and the game refuse the same things.
//
// **Unknown fields are refused**, unlike the other catalogs, because the tool rewrites this file:
// a misspelled field would otherwise be read as absent and dropped by the next -write.
//
// It refuses a track with no file or a file the game cannot play, two tracks sharing a name, a pack
// nothing describes, a source outside its pack's folder, a gain missing or misplaced, a quality
// missing, misplaced or off the scale, an encode to anything but Ogg, and a pack no track uses — a
// credit for something that does not ship.
func ParseMusic(raw []byte) MusicCatalog {
	const file = "music.json"
	var cat MusicCatalog
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cat); err != nil {
		panic(file + ": " + err.Error())
	}

	mustBeUniquelyKeyed(cat.Packs, file, func(p MusicPack) string { return p.Pack })
	packs := map[string]bool{}
	for _, p := range cat.Packs {
		if p.Name == "" {
			panic(fmt.Sprintf("%s: pack %s has no Name to credit it by", file, p.Pack))
		}
		packs[p.Pack] = true
	}

	mustBeUniquelyKeyed(cat.Tracks, file, func(t MusicTrack) string { return t.Name() })
	used := map[string]bool{}
	for _, t := range cat.Tracks {
		name := t.Name()
		ext := path.Ext(t.File)
		switch {
		case !slices.Contains(MusicExts, ext):
			panic(fmt.Sprintf("%s: %s is not a format the game plays (%v)", file, t.File, MusicExts))
		case strings.ContainsAny(t.File, `/\`):
			panic(fmt.Sprintf("%s: %s is in a folder; game-ready is flat", file, t.File))
		case !packs[t.Pack]:
			panic(fmt.Sprintf("%s: %s names pack %q, which Packs does not describe", file, name, t.Pack))
		case !strings.HasPrefix(t.Source, t.Pack+"/"):
			panic(fmt.Sprintf("%s: %s comes from %q, which is not under its pack's folder %s/", file, name, t.Source, t.Pack))
		case ext == ".mid" && (t.GainDB != nil || t.Quality != nil):
			panic(fmt.Sprintf("%s: %s is MIDI, whose level is the synthesizer's; it takes no GainDB or Quality", file, name))
		case ext != ".mid" && t.GainDB == nil:
			panic(fmt.Sprintf("%s: %s has no GainDB; every track's level is its own, 0 included", file, name))
		case t.Encoded() && ext != ".ogg":
			panic(fmt.Sprintf("%s: %s is re-encoded from %s, and an encode is to Ogg", file, name, t.Source))
		case t.Encoded() && t.Quality == nil:
			panic(fmt.Sprintf("%s: %s is encoded and has no Quality", file, name))
		case !t.Encoded() && t.Quality != nil:
			panic(fmt.Sprintf("%s: %s is a straight copy of its source, so a Quality says nothing", file, name))
		case t.Quality != nil && (*t.Quality < 0 || *t.Quality > 10):
			panic(fmt.Sprintf("%s: %s has Quality %d; libvorbis takes 0 to 10", file, name, *t.Quality))
		}
		used[t.Pack] = true
	}
	for _, p := range cat.Packs {
		if !used[p.Pack] {
			panic(fmt.Sprintf("%s: pack %s is credited and no track comes from it", file, p.Pack))
		}
	}
	return cat
}
