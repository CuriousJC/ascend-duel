# privateassets

**The audio kept out of git.** Everything under [assets/](../assets/) is committed; the music
loops in here are not, and are synced from a private S3 bucket instead.

**Two directories rather than one with a special corner**, because the split is the thing being
enforced. A rule saying "this one subdirectory of `assets/` is different" is a rule somebody has
to keep in their head; a directory outside `assets/` is a rule `git add assets/` obeys without
being told.

## What is here

```
privateassets/
  README.md            this file
  embed.go             package privateassets — the embed and the loader
  audio/
    manifest.json      committed: the checksum list
    *.ogg *.wav *.mp3  ignored: the loops
```

**`manifest.json` is committed and is load-bearing twice over.** It is the checksum list — what
the bundle holds and what each file hashes to — and it is also what keeps `audio/` real:
`//go:embed audio` is resolved at compile time and a pattern matching nothing is a *build* error,
so without a committed file in there a clean clone would fail to compile rather than merely play
silent.

## Getting the files

They live in a private S3 bucket, one prefix per bundle version, and the version is the `Bundle`
field in `audio/manifest.json`.

```powershell
aws s3 sync "s3://$env:ASCEND_DUEL_SOUNDS_BUCKET/sounds/v1/" privateassets/audio/ --exclude "*" --include "*.ogg" --include "*.wav" --include "*.mp3"
go run ./tools/privateassets          # what is present, what matches, what is unrecorded
```

**A build with none of them present is a supported state and is what a clean clone gets.** The
game loads whatever is there and nothing fails. What is *not* supported is a **configured
release** built that way, which is why both release build jobs run
`go run ./tools/privateassets -require` before they compile anything.

**Both that step and the sync before it are gated on the `SOUNDS_BUCKET` repository variable.**
With no bucket configured a release does neither and ships the synthesized score everywhere;
setting the variable turns the sync on and the guard with it, in one switch rather than two that
can disagree.

## Filing a new batch

The bucket holds exactly what embeds — the converted, shipped audio. Ebitengine decodes **MP3,
Ogg Vorbis and WAV and nothing else**, so a file delivered as AIFF or FLAC is converted before it
is filed, the same way generated art is reduced before it is committed. Those three are what
`music.decoders` and `privateassets.Audio` both take, so any of them can be filed here and play;
what a conversion is choosing between is size and fidelity, not whether the game can read it.

**Normalize on the way in, and check it against the score.** A delivered loop is mastered to
whatever the supplier chose, and the two in here arrived at about **-14 dBFS peak** — roughly a
fifth the amplitude of the synthesized score, which sits at -1.7 dBFS peak and -14.7 dBFS RMS. A
quiet file is not something to fix with the volume ceiling, because `music.fullVolume` caps the
score and every loop together: raising it to rescue one makes the chiptune harsh. **Peak-normalize
to -1.0 dBFS** and the two sit within a decibel or two of each other, which is the point — a
player should not reach for the bar when the screen changes.

Record the gain applied in the batch's `Note`, since it means the bytes that ship are not the
bytes that were delivered.

```powershell
go run ./tools/privateassets -write   # rewrite audio/manifest.json from what is in that directory
aws s3 sync privateassets/audio/ "s3://$env:ASCEND_DUEL_SOUNDS_BUCKET/sounds/v2/" --exclude "*" --include "*.ogg" --include "*.wav" --include "*.mp3"
```

**A bundle prefix is never overwritten in place.** Cut `v2` beside `v1` and move the `Bundle`
field; an overwritten prefix means two releases claiming one bundle shipped different audio,
which is the one thing the manifest exists to make impossible.
