package data

// The audio catalogs: **every sound the game ships, where it came from, and how it was made
// game-ready.** Three files, one shape:
//
//   - `music.json` is the loops: each plays on the `Screens` it names, one at a time.
//   - `sounds.json` is the effects: each plays on the `Cues` it names — a moment the game raises,
//     a relic toasting — once, over whatever else is sounding.
//   - `audio_packs.json` is where they came from: one record per purchase, credited by `Name`.
//     **Its own file because a pack can supply both** — a credit kept in each catalog would be
//     one line written twice that can drift apart.
//
// The audio itself is not in git. The owner's library — every pack as it was delivered — sits
// outside the repository, and `privateassets/mgmt/build-game-ready.ps1` builds each record into
// `game-ready/music/` or `game-ready/sounds/` beside it: `Source` read from the library, `GainDB`
// applied, encoded at `Quality`. Those folders are what is mirrored into `privateassets/audio/` and
// what a release fetches, so **only what these files name is ever built into the game**.
//
// **Two kinds of field, and one file each on purpose.** `Source`, `GainDB`, `Quality`, `Screens`
// and `Cues` are authored. `Bytes` and `SHA256` are written by `go run ./tools/privateassets -write`
// from the game-ready file, and `-require` refuses a release whose files do not hash to them — the
// bucket holds the current set rather than a history, so the hash committed here is what stops an
// older commit shipping a file re-made since. The tool rewrites each catalog in these structs'
// field order and touches only those two fields.
//
// **Every gain is the record's own.** There is no default level: a sound is made game-ready at the
// gain written on it, and a record without one is refused, so every figure is one somebody chose
// and can review. `build-game-ready.ps1 -Measure` says what gain would put a source's true peak at
// -1.0 dBFS — see privateassets/README.md.
//
// **These are the one catalog that carries provenance**, because bought audio is the one asset with
// a supplier to credit.
//
// **`Screens` is the pool a track may play in**, named as `state.ActiveScreen.String` writes them
// and checked there, since `data` cannot see that vocabulary. **`Cues` is a closed vocabulary that
// lives here**, `SoundCues`, because a cue is a name the screens raise and this file is the one
// place both sides can read. **A cue may be answered by several sounds**, and plays them in turn in
// this file's order — variety without a roll, so no stream is spent on what a player hears.

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"
)

//go:embed audio_packs.json
var audioPacksJSON []byte

//go:embed music.json
var musicJSON []byte

//go:embed sounds.json
var soundsJSON []byte

// The two folders the game-ready audio is filed in, under `game-ready/` at the source and under
// `privateassets/audio/` in the repository. A folder is which catalog a file belongs to.
const (
	MusicDir  = "music"
	SoundsDir = "sounds"
)

// AudioCatalog is all three files.
type AudioCatalog struct {
	Packs  []AudioPack
	Music  []MusicTrack
	Sounds []SoundTrack
}

// AudioPack is one source of audio: a purchase, or `custom` for the game's own.
type AudioPack struct {
	// Pack is the key: the pack's folder in the library, which every Source under it starts with.
	Pack string `json:"Pack"`

	// Name is the purchase as it is credited.
	Name string `json:"Name"`

	// Note is anything about the pack as a whole worth keeping beside it.
	Note string `json:"Note,omitempty"`
}

// AudioFile is what every shipped sound has, music or effect: the game-ready file, where it came
// from, how it was made, and what it hashes to.
type AudioFile struct {
	// File is the game-ready filename, extension included. **Its stem is the sound's name**, and a
	// name is unique across both catalogs: one stem meaning two files is a confusion in the logs and
	// in the library whichever call plays it.
	File string `json:"File"`

	// Pack names the record in audio_packs.json this sound came from.
	Pack string `json:"Pack"`

	// Source is the delivered file, as a path under the library's `sounds/` — `<pack>/<file>`.
	Source string `json:"Source"`

	// GainDB is applied to Source on the way to game-ready, in decibels. Required for everything but
	// the MIDI score, whose level is the synthesizer's; refused on that.
	GainDB *float64 `json:"GainDB,omitempty"`

	// Quality is the libvorbis quality, 0 to 10, the file was encoded at. Required whenever the file
	// is encoded — a source in another format, or any gain — and refused on a straight copy.
	Quality *int `json:"Quality,omitempty"`

	// Bytes and SHA256 describe the game-ready file. **Generated** by tools/privateassets -write.
	Bytes  int64  `json:"Bytes"`
	SHA256 string `json:"SHA256"`
}

// MusicTrack is one loop, and the screens it may play on.
type MusicTrack struct {
	AudioFile
	Screens []string `json:"Screens"`
}

// SoundTrack is one effect, and the cues it plays on.
type SoundTrack struct {
	AudioFile
	Cues []string `json:"Cues"`
}

// The cues: the moments a sound can play on. **Closed, like every word a catalog may write**, so a
// misspelled cue fails the launch rather than producing an effect nothing ever plays.
const (
	// CueRelicToast is a worn relic toasting in a fight: firing into a hit's arithmetic, or as a
	// hand is dealt. Once per relic per toast.
	CueRelicToast = "relic-toast"

	// CueBoosterOpened is a booster bought off the shop's shelf and opened.
	CueBoosterOpened = "booster-opened"
)

// SoundCues is every cue there is.
var SoundCues = []string{CueRelicToast, CueBoosterOpened}

// MusicExts is what a track can be: MP3, Ogg Vorbis and WAV, which Ebitengine decodes, and MIDI,
// which internal/music synthesizes. privateassets.audioExts and music.decoders take the same set.
// **An effect is never MIDI**: the synthesizer renders the score and nothing else.
var MusicExts = []string{".ogg", ".wav", ".mp3", ".mid"}

// Name is the sound's name: its file's stem.
func (f AudioFile) Name() string { return strings.TrimSuffix(f.File, path.Ext(f.File)) }

// Encoded says whether making the file game-ready re-encodes its source, rather than copying it.
func (f AudioFile) Encoded() bool {
	return path.Ext(f.File) != path.Ext(f.Source) || (f.GainDB != nil && *f.GainDB != 0)
}

// Each walks every shipped file, music then sounds, with the folder it is filed in. **By pointer**,
// so tools/privateassets can write the hashes back.
func (c *AudioCatalog) Each(fn func(dir string, f *AudioFile)) {
	for i := range c.Music {
		fn(MusicDir, &c.Music[i].AudioFile)
	}
	for i := range c.Sounds {
		fn(SoundsDir, &c.Sounds[i].AudioFile)
	}
}

// CueSounds is every sound a cue plays, by name, in file order — the order they take turns in.
func (c AudioCatalog) CueSounds(cue string) []string {
	var names []string
	for _, s := range c.Sounds {
		if slices.Contains(s.Cues, cue) {
			names = append(names, s.Name())
		}
	}
	return names
}

// LoadAudio parses the three embedded catalogs.
func LoadAudio() AudioCatalog { return ParseAudio(audioPacksJSON, musicJSON, soundsJSON) }

// ParseAudio parses and validates the three catalogs together. tools/privateassets reads the files
// off disk through it, so the tool and the game refuse the same things.
//
// **Unknown fields are refused**, unlike the other catalogs, because the tool rewrites the music and
// sounds files: a misspelled field would otherwise be read as absent and dropped by the next -write.
//
// **Checked together, because three of the rules span files**: a name unique across both catalogs,
// a pack every record can find, and every credited pack shipping something. Per record, it refuses
// a file the game cannot play, a file in a folder, a source outside its pack's folder, a gain
// missing or misplaced, a quality missing, misplaced or off the scale, an encode to anything but
// Ogg, a MIDI effect, and a cue the game does not raise.
func ParseAudio(packsRaw, musicRaw, soundsRaw []byte) AudioCatalog {
	var cat AudioCatalog
	strict("audio_packs.json", packsRaw, &cat.Packs)
	strict("music.json", musicRaw, &cat.Music)
	strict("sounds.json", soundsRaw, &cat.Sounds)

	mustBeUniquelyKeyed(cat.Packs, "audio_packs.json", func(p AudioPack) string { return p.Pack })
	packs := map[string]bool{}
	for _, p := range cat.Packs {
		if p.Name == "" {
			panic(fmt.Sprintf("audio_packs.json: pack %s has no Name to credit it by", p.Pack))
		}
		packs[p.Pack] = true
	}

	used := map[string]bool{}
	names := map[string]string{}
	cat.Each(func(dir string, f *AudioFile) {
		file := dir + ".json" // each catalog is named for its folder
		name := f.Name()
		if name == "" {
			panic(file + ": a record has no File")
		}
		if other, taken := names[name]; taken {
			panic(fmt.Sprintf("%s: %s is a name %s already uses; one stem is one sound", file, name, other))
		}
		names[name] = file
		checkAudioFile(file, *f, packs, dir == SoundsDir)
		used[f.Pack] = true
	})

	for _, s := range cat.Sounds {
		for _, cue := range s.Cues {
			if !slices.Contains(SoundCues, cue) {
				panic(fmt.Sprintf("sounds.json: %s plays on %q, which is no cue (%v)", s.Name(), cue, SoundCues))
			}
		}
	}

	for _, p := range cat.Packs {
		if !used[p.Pack] {
			panic(fmt.Sprintf("audio_packs.json: pack %s is credited and nothing ships from it", p.Pack))
		}
	}
	return cat
}

// strict decodes one catalog, refusing a field the struct does not have.
func strict(file string, raw []byte, into any) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		panic(file + ": " + err.Error())
	}
}

// checkAudioFile is the per-record half of ParseAudio.
func checkAudioFile(file string, f AudioFile, packs map[string]bool, effect bool) {
	name := f.Name()
	ext := path.Ext(f.File)
	switch {
	case !slices.Contains(MusicExts, ext):
		panic(fmt.Sprintf("%s: %s is not a format the game plays (%v)", file, f.File, MusicExts))
	case effect && ext == ".mid":
		panic(fmt.Sprintf("%s: %s is MIDI, and the synthesizer renders the score alone", file, name))
	case strings.ContainsAny(f.File, `/\`):
		panic(fmt.Sprintf("%s: %s is in a folder; a catalog's files sit in its own folder, flat", file, f.File))
	case !packs[f.Pack]:
		panic(fmt.Sprintf("%s: %s names pack %q, which audio_packs.json does not credit", file, name, f.Pack))
	case !strings.HasPrefix(f.Source, f.Pack+"/"):
		panic(fmt.Sprintf("%s: %s comes from %q, which is not under its pack's folder %s/", file, name, f.Source, f.Pack))
	case ext == ".mid" && (f.GainDB != nil || f.Quality != nil):
		panic(fmt.Sprintf("%s: %s is MIDI, whose level is the synthesizer's; it takes no GainDB or Quality", file, name))
	case ext != ".mid" && f.GainDB == nil:
		panic(fmt.Sprintf("%s: %s has no GainDB; every sound's level is its own, 0 included", file, name))
	case f.Encoded() && ext != ".ogg":
		panic(fmt.Sprintf("%s: %s is re-encoded from %s, and an encode is to Ogg", file, name, f.Source))
	case f.Encoded() && f.Quality == nil:
		panic(fmt.Sprintf("%s: %s is encoded and has no Quality", file, name))
	case !f.Encoded() && f.Quality != nil:
		panic(fmt.Sprintf("%s: %s is a straight copy of its source, so a Quality says nothing", file, name))
	case f.Quality != nil && (*f.Quality < 0 || *f.Quality > 10):
		panic(fmt.Sprintf("%s: %s has Quality %d; libvorbis takes 0 to 10", file, name, *f.Quality))
	}
}
