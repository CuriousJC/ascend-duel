# privateassets

**The audio kept out of git** — every sound the game makes, the bought loops and its own MIDI
score (`custom/duello.mid`) alike. Everything under [assets/](../assets/) is committed; nothing in
here is, and a release syncs it from a private S3 bucket instead.

**Two directories rather than one with a special corner**, because the split is the thing being
enforced. A rule saying "this one subdirectory of `assets/` is different" is a rule somebody has
to keep in their head; a directory outside `assets/` is a rule `git add assets/` obeys without
being told.

## What is here

```
privateassets/
  README.md            this file
  embed.go             package privateassets — the embed and the loader
  mgmt/                committed: the sync, push and AWS setup scripts
    local.psd1         ignored: this machine's source path, bucket and profile
  audio/
    manifest.json      committed: the checksum list
    bundle-names.txt   committed: which purchase each pack folder came from
    <pack>/            ignored: one folder per pack
      *.ogg *.wav *.mp3 *.mid  the sounds
```

**A folder is a pack.** `go run ./tools/privateassets -write` records a file under `audio/<pack>/`
as belonging to that pack and adds the pack to `Packs` for its `Note` to be written. **The folder
never reaches a track's name**: `<pack>/dark-fantasy-01.wav` is the track
`dark-fantasy-01`, so two files sharing a stem in different packs fail the launch.

**`manifest.json` is committed and is load-bearing twice over.** It is the checksum list — what
the bundle holds and what each file hashes to — and it is also what keeps `audio/` real:
`//go:embed audio` is resolved at compile time and a pattern matching nothing is a *build* error,
so without a committed file in there a clean clone would fail to compile rather than merely play
silent.

## Getting the files

They live in a private S3 bucket under `sounds/`, one folder per pack, as a mirror of the owner's
own copy. A release fetches them; anyone else with read access can too:

```powershell
aws s3 sync "s3://<bucket>/sounds/" privateassets/audio/ --exclude "*" --include "*.ogg" --include "*.wav" --include "*.mp3" --include "*.mid"
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

## Filing a new batch

The packs are kept outside the repository and mirrored in by the scripts in [mgmt/](mgmt/), which
read this machine's paths and bucket from a gitignored `mgmt/local.psd1` (template:
`mgmt/local.example.psd1`); the procedure is the [`sounds` skill](../.claude/skills/sounds/SKILL.md).

The bucket holds the converted, shipped audio. Ebitengine decodes **MP3, Ogg Vorbis and WAV and
nothing else**, so a file delivered as AIFF or FLAC is converted before it is filed, the same way
generated art is reduced before it is committed. Those three are what `music.decoders` and
`privateassets.Audio` both take; what a conversion is choosing between is size and fidelity, not
whether the game can read it. **MIDI is the score alone** — `custom/duello.mid`, synthesized by
`internal/music`.

**Normalize on the way in, and check it against the score.** A delivered loop is mastered to
whatever the supplier chose, and the dark-fantasy loops arrived at about **-14 dBFS peak** — roughly a
fifth the amplitude of the synthesized score, which sits at -1.7 dBFS peak and -14.7 dBFS RMS. A
quiet file is not something to fix with the volume ceiling, because `music.fullVolume` caps the
score and every loop together: raising it to rescue one makes the chiptune harsh. **Peak-normalize
to -1.0 dBFS** and the two sit within a decibel or two of each other, which is the point — a
player should not reach for the bar when the screen changes.

Record the gain applied in the batch's `Note`, since it means the bytes that ship are not the
bytes that were delivered.

```powershell
go run ./tools/privateassets -write   # rewrite audio/manifest.json from what is in that directory
```

**The bucket is the current set, not a history.** It mirrors the owner's copy, and what a release
may ship is the manifest committed with it — so the bucket and the committed manifest move
together, and an old commit rebuilt against a newer bucket fails `-require` rather than shipping
sounds its manifest never named. The bucket is versioned, so a file deleted from it is
recoverable.
