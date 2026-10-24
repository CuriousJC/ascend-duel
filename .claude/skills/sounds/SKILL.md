---
name: sounds
description: How bought audio reaches the game - the library of packs in Google Drive, the three audio catalogs (data/music.json for loops, data/sounds.json for effects, data/audio_packs.json for credits) and the per-sound gain and quality they record, building game-ready/music and game-ready/sounds from the library, the scripts in privateassets/mgmt that mirror them into privateassets/audio/ and the bucket, the tool that holds the directory to the catalogs, naming a track for a screen in score.go, a sound cue a screen raises, and the S3 bucket a release fetches from. Load before adding, moving, renaming or replacing any sound, changing a gain, adding a cue, building game-ready or running the sync, touching privateassets/, tools/privateassets, any of the three catalogs, internal/game/score.go's track names, internal/music's loader or effect player, or the release workflow's private-asset steps.
---

# Sounds

**No sound is in git.** Everything under `assets/` is committed; every sound the game makes —
the bought loops and the game's own MIDI score, `duello.mid` — lives in `privateassets/audio/`,
which is gitignored except for its README. **Three committed catalogs are the record**:
`data/music.json` for the loops, `data/sounds.json` for the effects, one entry per sound that ships,
and `data/audio_packs.json` for the packs they came from. The *why* of the arrangement — the separate directory, the embed, the bucket,
the gating, the levels — is [privateassets/README.md](../../../privateassets/README.md),
`go doc ./privateassets`, `go doc ./data` on `ParseAudio`, and the header of
`tools/privateassets/main.go`. This file is the procedure.

## Where a sound lives, in order

1. **The library**, `<SourcePath>/sounds/<pack>/` — the owner's `ascend_assets/` in Google Drive,
   one folder per pack, every file as it was delivered. **It is only ever read.** `SourcePath` is
   in `privateassets/mgmt/local.psd1`, the one gitignored file in `mgmt/` and the only place the
   path is written down.
2. **The catalogs**, which name what ships and say how each is made from the library: its
   `Source`, its own `GainDB`, its own `Quality`, the pack it came from, and the generated `Bytes`
   and `SHA256` of the game-ready file — plus the `Screens` a loop may play on in `music.json`, and
   the `Cues` an effect answers in `sounds.json`. `audio_packs.json` credits every pack either uses.
3. **`<SourcePath>/game-ready/music/` and `game-ready/sounds/`**, flat, one file per record of
   that catalog, built from the library by `privateassets/mgmt/build-game-ready.ps1`. They are
   derived: rebuilding overwrites them, and **the encode is bit-exact**, so rebuilding an unchanged
   record leaves its hash alone.
4. **`privateassets/audio/`**, a mirror of `game-ready/` by
   `privateassets/mgmt/sync-from-drive.ps1`, which then records each file's hash in its catalog. `//go:embed audio` compiles every file there into the binary, so nothing but the
   game-ready tracks and the committed README belongs in it.
5. **The S3 bucket**, a mirror of the whole source folder by `push-sounds.ps1`: the tracks at
   `s3://<bucket>/game-ready/`, which is all a release fetches, and the library at `sounds/`, a
   backup the release role cannot read.

**The source folder is in this workspace**, and its path is `SourcePath` in `local.psd1` — read it
from there rather than writing it down anywhere committed. Being in the workspace is what lets the
file tools and the shell reach it; that is for reading the library and building `game-ready/`, not
for reorganizing the owner's folder or editing a delivered file.

**`privateassets/mgmt/` is the management kit.** `config.ps1` is dot-sourced by every script and
reads `local.psd1` — the source path, the bucket, the region, the AWS profile — and a script's own
parameter overrides the file. `local.example.psd1` is the committed template. **A new script here
reads its machine values through `config.ps1`, never as a literal**, which is what keeps the whole
directory committable.

## Adding a sound

```powershell
powershell -File privateassets/mgmt/build-game-ready.ps1 -Measure <pack>/<file>.wav  # peaks, and a gain
#   ...author the record in data/music.json or data/sounds.json, and its pack...
powershell -File privateassets/mgmt/build-game-ready.ps1 -Track <name>              # into game-ready/<folder>/
powershell -File privateassets/mgmt/sync-from-drive.ps1                             # into audio/, hash recorded
go test ./data ./internal/game                                                      # the records, and score.go against them
```

- **The record comes first.** A file in `game-ready/` or `audio/` that no record names is refused by
  `-require`, and `-write` will not invent a record for it: a record is what the file is built from.
- **A loop or an effect?** A loop plays on screens, one at a time, and goes in `music.json`; an
  effect plays once on a cue, over everything, and goes in `sounds.json`. **A name is unique across
  both**, and a new pack needs a record in `audio_packs.json` with the purchase as its `Name`.
- **Every sound's gain is its own, and so is its quality.** There is no default for either, and a
  record without a `GainDB` is refused — 0 is a gain somebody chose. `-Measure` reports the sample
  and true peaks and suggests the gain that puts the **true peak** at -1.0 dBFS; that is a starting
  figure for the owner to review, not a rule to apply unasked. Quality is libvorbis's 0–10, and 6 is
  what the shipped tracks use.
- **Measure the true peak.** A file can sit under full scale by its samples and go over between
  them, and the resample to 44100 Hz and the encode bring that out. The build refuses an Ogg peaking
  above -0.1 dBFS; the answer is a lower `GainDB`, not a different measurement.
- **What the build does**: a MIDI file, or a source already in its game-ready format at no gain, is
  copied; anything else is encoded to Ogg Vorbis at 44100 Hz with the record's gain and quality,
  carrying the text tags. **What does not survive**: embedded cover art and the WAV-only chunks —
  `bext`, `iXML`, editor blocks. The library keeps them.
- **A loop's length is checked.** The build refuses an Ogg whose duration differs from the source's
  by more than a millisecond, because `music.LoadTrack` loops the stream's whole length and any
  difference is a gap or a cut at the seam.
- **`Pack` must be credited in `audio_packs.json`, `Source` must be under that pack's folder**, and
  every credited pack must ship something from one catalog or the other — a pack's `Name` is the purchase as it is credited, and `Note` is for anything
  about the pack as a whole.
- **Only `.ogg`, `.wav`, `.mp3` and `.mid` play** — `data.MusicExts`, `privateassets.audioExts` and
  `music.decoders` must agree. Ebitengine's WAV decoder takes 8- and 16-bit only, so a sound ships
  as Ogg. **An effect is never MIDI.** **`.mid` is the score alone**: `main` takes `duello.mid` out and starts it with
  `music.Start`, and any other MIDI file would reach `music.LoadTrack` and be refused.
- **The score's own tests read the real file** — `internal/music/music_test.go` skips them on a
  clone without the sounds, CI included. Run `go test ./internal/music` on a synced machine after
  touching `duello.mid` or the synthesizer.

## Changing a sound

Edit the record and rebuild that sound — `-Track <name>` — then sync. **Never edit a file in the
library or in `game-ready/` by hand**: the library is the delivered file a rebuild starts from, and
`game-ready/` is overwritten by the next build. With no arguments, `build-game-ready.ps1` prints
every sound's state — not built, built and unrecorded, differing from its record, or matching — and
any file in `game-ready/` no record names. `sync-from-drive.ps1` ends with the same list; read it
out to the owner rather than acting on it. `-Missing` builds what is not built yet and `-All`
rebuilds everything.

**The tools are ffmpeg and ffprobe**, `winget install -e --id Gyan.FFmpeg`.

## Naming: a sound is its stem

`combat-ambience-1.ogg` is the track `combat-ambience-1`. So:

- **Two files sharing a stem fail the launch**, across both folders, and `ParseAudio` refuses two
  records sharing one across both catalogs.
- **A stem is what `internal/game/score.go` writes down**, so renaming a track is editing the
  record's `File` and the constant that names it.

## Making a screen play a sound

A file being present plays nothing. `internal/game/score.go` says what each screen does, read every
frame; a name with no loaded file falls back to the score. **A record's `Screens` is the pool it
may play in**, and a screen either names one fixed track, which must be in its pool, or is a **pool
screen** (`{pool: true}`) that plays one track of it, rolled per fight off `seeds.CombatMusic` when
the screen is entered — so a fight and its retry play the same track. Each pool screen reads its
own pool, by screen name. `music.Default` is the synthesized score: the fallback, the music of
any screen that names it, and **the studio splash's**, started by the loader in `main.go` before
`score.go` runs at all.
`TestEveryTrackPlaysWhereItMay` fails on a fixed track outside its pool, a pool screen with an empty
pool, or a track pooled on a screen that always plays something else, and logs every track no
screen plays. Screen names are `state.ActiveScreen.String`. **A clone without the sounds has no
score either and is silent** — that is the supported state, not a bug. Load `combat-screen` too if
the sound is about a round.

## Making a moment play an effect

An effect plays on a **cue**, a moment a screen raises: `music.PlayCue(data.CueRelicToast)`. `main`
loads each record of `sounds.json` under its `Cues`, so the screen never names a file. **The cues are
closed** — `data.SoundCues` — and a cue the game does not raise is refused at load. **A cue may be
answered by several sounds**, which take turns in `sounds.json`'s order (`music.PlayCue`) — a
rotation rather than a roll, so what a player hears spends no stream.

- **A new cue is a constant in `data` and the one call site that raises it**, never something the
  file can assert into existence.
- **Each play is its own player, so effects layer**, and `music.maxLiveSounds` bounds how many at
  once. Effects have their own level and their own bar on the settings screen, beside the music's.
- **`data.CueRelicToast` is raised wherever a relic toasts in a fight**: `shakeRelicAt` and each beat
  of the deal's cascade (`internal/screens/combat_sound.go`). Toasts raised on one frame are spread
  a sixth of a beat apart in the sound only, so two relics firing into one figure are two sounds
  rather than one twice as loud.
- **`data.CueBoosterOpened` is raised once a booster is paid for**, in `ShopScene.openBooster`
  beside `openBoosters`, so a refused purchase opens nothing and sounds nothing.
- **A cue may never change an outcome**: it is raised by playback, after the round is resolved, or
  after the decision it marks has already been made.

## Setting up AWS

The owner runs the AWS and GitHub steps — they change things outside the repo — unless they ask
for a specific one to be run:

1. Fill in `Bucket` (and `AwsProfile` if not the default) in `privateassets/mgmt/local.psd1`.
2. `powershell -File privateassets/mgmt/setup-aws.ps1` — needs credentials that can create a
   bucket, an IAM OIDC provider and a role. It stops early, saying so, if `aws sts
   get-caller-identity` fails, and it is safe to re-run: re-running is how a change to the role's
   policies reaches AWS. It prints the role ARN and three `gh variable set` commands.
3. `powershell -File privateassets/mgmt/push-sounds.ps1 -DryRun`, read the plan, then without
   `-DryRun`.
4. Commit `data/music.json` and `data/sounds.json` so the committed hashes match the bucket.
5. Run the three `gh variable set` commands — `SOUNDS_BUCKET`, `SOUNDS_ROLE_ARN`, `SOUNDS_REGION`.
   **This arms the release**: from then on a release fetches the sounds and refuses to build
   without them.
6. Cut a release and confirm the build log shows the sync and `-require` passing.

The role trusts only `repo:CuriousJC/ascend-duel` on `refs/heads/main` and `refs/tags/v*`, so a
fork's pull request cannot assume it; anyone with write access to the repository can, by pushing a
`v*` tag. It reads `game-ready/*` and nothing else. Nothing in `mgmt/` holds a secret — the account
ID is derived at run time and lives only in the `SOUNDS_ROLE_ARN` variable.

## Releasing

Both release build jobs run `.github/scripts/fetch-private-assets.sh` and then
`go run ./tools/privateassets -require`, **both gated on the `SOUNDS_BUCKET` repository
variable**. Unset, a release fetches nothing and ships with no sound; set, the fetch syncs
`s3://$SOUNDS_BUCKET/game-ready/` and the guard refuses a build missing a recorded sound, holding
different bytes, or holding a file no record names. CI runs neither, and passes on a clone with no
sounds at all. The fetch takes OIDC credentials, never stored keys — see the script's header.

**The catalogs a release builds are the ones at that commit**, so the bucket has to hold exactly
what they record. `push-sounds.ps1` (`-DryRun` first) is the whole push: it runs the sync and
`-require` first, mirrors `ascend_assets/` to the bucket with `--delete`, and says when
a catalog is left uncommitted. Push and commit them together. **There is no version prefix**:
the bucket is the current set, so an old commit rebuilt against it fails `-require` — accepted,
since only new releases fetch sounds. The bucket is versioned, so a deleted file is recoverable.

## What not to do

- **Never `git add` anything under `privateassets/audio/` but its README.** The ignore rules make
  that the default; `git check-ignore -v` confirms a path.
- **Never put a script, an original, or a working file inside `audio/` or `game-ready/`** — the one
  embeds and the other is mirrored into it.
- **Never write a count of sounds or packs** into a doc, a skill or CLAUDE.md. The catalogs are the
  list and `go run ./tools/privateassets` reads them.
- **Never reason about where a sound came from** beyond what `audio_packs.json` says — provenance
  is the owner's business, and audio is the one asset that records it.
