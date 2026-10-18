# privateassets

**The audio kept out of git** — every sound the game makes, the bought loops and its own MIDI
score (`duello.mid`) alike. Everything under [assets/](../assets/) is committed; of this directory
only the code, the scripts and two READMEs are, and a release syncs the sounds from a private S3
bucket instead.

**Two directories rather than one with a special corner**, because the split is the thing being
enforced. A rule saying "this one subdirectory of `assets/` is different" is a rule somebody has
to keep in their head; a directory outside `assets/` is a rule `git add assets/` obeys without
being told.

## What is here

```
privateassets/
  README.md            this file
  embed.go             package privateassets — the embed and the loader
  mgmt/                committed: the build, sync, push and AWS setup scripts
    local.psd1         ignored: this machine's source path, bucket and profile
  audio/
    README.md          committed: keeps the directory real for //go:embed
    music/             ignored: the game-ready loops and the score, flat
    sounds/            ignored: the game-ready effects, flat
```

**Two committed catalogs say what belongs in `audio/`**: [data/music.json](../data/music.json) for
`music/` and [data/sounds.json](../data/sounds.json) for `sounds/`. Each entry carries the pack it came
from, the delivered file it was made from, the gain and quality it was made at, where it plays — the
screens a loop may play on, the cues an effect answers — and the size and hash of the game-ready
file. The packs themselves, and the purchase each is credited to, are
[data/audio_packs.json](../data/audio_packs.json), because one pack can supply both. They are
committed; the audio is not. `audio/README.md` is committed for a duller reason: `//go:embed audio` is resolved
at compile time and a directory with nothing embeddable in it is a *build* error, so without it a
clean clone would fail to compile rather than merely play silent.

**A sound's name is its filename stem** — `combat-ambience-1.ogg` is the track `combat-ambience-1`
— and a name is unique across both folders, so two files sharing a stem fail the launch. Each folder
is flat because nothing about where a file came from reaches the game but its record.

## Where the sounds come from

The owner's source folder (`SourcePath` in `mgmt/local.psd1`) holds two things:

```
ascend_assets/
  sounds/<pack>/...   the library: every pack as it was delivered, never edited
  game-ready/music/   built from the library, one file per record in data/music.json
  game-ready/sounds/  built from the library, one file per record in data/sounds.json
```

`mgmt/build-game-ready.ps1` reads each record's source out of the library and writes its
game-ready file into the folder of its catalog: a MIDI file copied, anything else encoded to Ogg Vorbis at 44100 Hz with the
record's own gain and quality. **The library is only ever read**, so a gain is changed by editing
the record and rebuilding, and the delivered file is always there to rebuild from.
`mgmt/sync-from-drive.ps1` mirrors `game-ready/` into `audio/` and records each file's hash in
its catalog. **The encode is bit-exact**, so rebuilding an unchanged record builds the same bytes and
leaves its hash alone. The procedure is the [`sounds` skill](../.claude/skills/sounds/SKILL.md).

## Getting the files

The bucket mirrors the whole source folder, so the game-ready tracks are under `game-ready/` and
the library under `sounds/`. A release fetches `game-ready/` and nothing else — the role it assumes
cannot read the library — and anyone else with read access can too:

```powershell
aws s3 sync "s3://<bucket>/game-ready/" privateassets/audio/ --exclude "*" --include "*.ogg" --include "*.wav" --include "*.mp3" --include "*.mid"
go run ./tools/privateassets          # what is present, what matches, what is unrecorded
```

**A build with none of them present is a supported state and is what a clean clone gets.** The
game loads whatever is there and nothing fails. What is *not* supported is a **configured
release** built that way, which is why both release build jobs run
`go run ./tools/privateassets -require` before they compile anything.

**Both that step and the sync before it are gated on the `SOUNDS_BUCKET` repository variable.**
With no bucket configured a release does neither and ships with no sound at all;
setting the variable turns the sync on and the guard with it, in one switch rather than two that
can disagree.

## Levels

Ebitengine decodes **MP3, Ogg Vorbis and WAV and nothing else**, and its WAV decoder takes 8- and
16-bit files only, which is one more reason a track ships as Ogg. **MIDI is the score alone** —
`duello.mid`, synthesized by `internal/music`.

**Every track carries its own gain, and -1.0 dBFS true peak is the level the score sits beside.**
A delivered loop is mastered to whatever the supplier chose, and the dark-fantasy loops arrived at
about **-14 dBFS peak** — roughly a fifth the amplitude of the synthesized score, which sits at -1.7
dBFS peak and -14.7 dBFS RMS. A quiet file is not something to fix with the volume ceiling, because
`music.fullVolume` caps the score and every loop together: raising it to rescue one makes the
chiptune harsh. At -1.0 dBFS the two sit within a decibel or two of each other, which is the point
— a player should not reach for the bar when the screen changes.

**Measure the true peak, not the sample peak.** A master whose samples stay under full scale can
still go over between them, and a resample or a lossy encode is what brings that out: a file
reading -0.3 dBFS by its samples can come out of the encoder at -0.3 after a -0.7 dB cut.
`build-game-ready.ps1 -Measure` reports both and suggests a gain from the true peak, and a build
refuses an encode that peaks above -0.1 dBFS.

**The bucket is the current set, not a history.** What a release may ship is the catalogs
committed with it — so the bucket and the committed hashes move together, and an old commit rebuilt
against a newer bucket fails `-require` rather than shipping a sound its record never named. The
bucket is versioned, so a file deleted from it is recoverable.
