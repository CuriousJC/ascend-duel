---
name: sounds
description: How bought audio reaches the game - the packs in Google Drive, the committed scripts in privateassets/mgmt that mirror them into privateassets/audio/ and the bucket, the manifest and the tool that keeps it honest, naming a track for a screen in score.go, and the S3 bucket a release fetches from. Load before adding, moving, renaming or replacing any sound, running the sync, touching privateassets/, tools/privateassets, manifest.json, bundle-names.txt, internal/game/score.go's track names, internal/music's loader, or the release workflow's private-asset steps.
---

# Sounds

**No sound is in git.** Everything under `assets/` is committed; every sound the game makes —
the bought loops and the game's own MIDI score, `custom/duello.mid` — lives in
`privateassets/audio/`, which is gitignored except for `manifest.json` and `bundle-names.txt`.
The *why* of the arrangement — the separate directory, the embed, the bucket, the gating — is
[privateassets/README.md](../../../privateassets/README.md), `go doc ./privateassets` and the
header of `tools/privateassets/main.go`. This file is the procedure.

## Where a sound lives, in order

1. **The source folder**, the owner's `ascend_assets/` in Google Drive — `SourcePath` in
   `privateassets/mgmt/local.psd1`, the one gitignored file in `mgmt/` and the only place the path
   is written down. Its `sounds/` holds one folder per pack — the purchase a file came from — plus
   `bundle-names.txt`, one `folder:full purchase name` line per pack. The owner files sounds here;
   it is the source everything else is mirrored from. **Only the sounds are outside git**; every
   script that moves them is committed.
2. **`privateassets/audio/<pack>/`**, mirrored from the source by
   `privateassets/mgmt/sync-from-drive.ps1`. The scripts live in `mgmt/`, *beside* `audio/`, on
   purpose: `//go:embed audio` compiles every file under that directory into the binary, so nothing
   that is not a shipped sound or one of the two committed text files may be put there.
3. **The S3 bucket**, a mirror of the whole source folder, so the packs are at
   `s3://<bucket>/sounds/<pack>/`. `privateassets/mgmt/push-sounds.ps1` pushes it; the release
   jobs fetch it. See *Releasing* below.

**`privateassets/mgmt/` is the management kit.** `config.ps1` is dot-sourced by every script and
reads `local.psd1` — the source path, the bucket, the region, the AWS profile — and a script's own
parameter overrides the file. `local.example.psd1` is the committed template. **A new script here
reads its machine values through `config.ps1`, never as a literal**, which is what keeps the whole
directory committable.

**A pack folder is the pack.** `-write` records a file under `audio/<pack>/` as belonging to that
pack and adds the pack to `Packs`; `-require` refuses an entry whose `Pack` disagrees with its
folder. A pack folder should match a line in `bundle-names.txt`, which is how a sound is traced
to the purchase it came from. Nothing checks that, so check it by eye when a folder is added.

## Adding or changing sounds

```powershell
powershell -File privateassets/mgmt/sync-from-drive.ps1   # mirror the source, then -write, then report
go run ./tools/privateassets                          # what is present, what matches, what is unrecorded
go test ./internal/game                               # score.go's names against the manifest
```

- **Drive is the source and the sync is a mirror** (`robocopy /MIR`). A file removed or renamed
  in Drive is deleted from `audio/`, and anything filed straight into `audio/` is deleted on the
  next sync — **file in Drive, never in the repo**. `manifest.json` is the one exception, kept by
  the script's `/XF` and rebuilt by `-write`. The Drive folder is also the shape the S3 bundle is
  uploaded in, so Drive, `audio/` and the bucket hold the same tree.
- **Normalize before a sound ships: peak to -1.0 dBFS**, and record the gain in the pack's `Note`
  — the bytes that ship are then not the bytes that were delivered, and the note is the only
  record of it. Bought audio arrives quieter than the synthesized score; never rescue a quiet file
  by raising `music.fullVolume`, which caps everything. The README has the measurements.
  Normalizing is done with the owner, file by file — do not batch-process a pack unasked.
- **Normalize in Drive, not only locally.** The sync overwrites `audio/` from Drive, so a file
  normalized only in `privateassets/audio/` is undone by the next sync.
- **`Note` is hand-written.** `-write` keeps existing notes and adds a new pack with none; fill it
  in rather than leaving it blank.
- **Only `.ogg`, `.wav`, `.mp3` and `.mid` load** — `privateassets.audioExts` and
  `tools/privateassets`' copy must agree, and the first three must also be `music.decoders`.
  **`.mid` is the score alone**: `main` takes `duello.mid` out and starts it with `music.Start`,
  and any other MIDI file would reach `music.LoadTrack` and be refused. Anything else is converted
  first.
- **The score's own tests read the real file** — `internal/music/music_test.go` skips them on a
  clone without the bundle, CI included. Run `go test ./internal/music` on a synced machine after
  touching `duello.mid` or the synthesizer.

## Naming: the folder never reaches a track

A track is its **filename stem**: `humble-bundle-game-audio-sfx-treasure-trove/dark-fantasy-01.wav`
is the track `dark-fantasy-01`, wherever it is filed. So:

- **Two files sharing a stem anywhere under `audio/` fail the launch** — across packs as well as
  `hit.ogg` beside `hit.wav`. Rename one in Drive.
- **Moving a file between packs changes nothing a screen plays**, only the manifest.
- **A stem is what `internal/game/score.go` writes down**, so renaming a file is editing the
  constant that names it. `TestEveryNamedTrackIsInTheBundle` fails on a name the manifest does
  not carry, and logs every bundled track no screen plays.

## Making a screen play a sound

A file being present plays nothing. `internal/game/score.go` picks a track per screen, read every
frame; a name with no loaded file falls back to the score. **A clone without the bundle has no
score either and is silent** — that is the supported state, not a bug. Load `combat-screen` too
if the sound is about a round. **There is no sound-effects system yet** — `score.go` is music
only, so a one-shot effect is a design conversation before it is a file.

## Setting up AWS, the first time

In order, and the owner runs the AWS and GitHub steps — they change things outside the repo:

1. Fill in `Bucket` (and `AwsProfile` if not the default) in `privateassets/mgmt/local.psd1`.
   `SourcePath` is already the Drive folder.
2. `powershell -File privateassets/mgmt/setup-aws.ps1` — needs credentials that can create a
   bucket, an IAM OIDC provider and a role. It stops early, saying so, if `aws sts
   get-caller-identity` fails. It prints the role ARN and three `gh variable set` commands.
3. `powershell -File privateassets/mgmt/push-sounds.ps1 -DryRun`, read the plan, then without
   `-DryRun`.
4. Commit `privateassets/audio/manifest.json` (and `bundle-names.txt`) so the committed manifest
   matches the bucket.
5. Run the three `gh variable set` commands — `SOUNDS_BUCKET`, `SOUNDS_ROLE_ARN`, `SOUNDS_REGION`.
   **This arms the release**: from then on a release fetches the sounds and refuses to build
   without them. It is a remote change, so the owner makes it.
6. Cut a release and confirm the build log shows the sync and `-require` passing.

The role trusts only `repo:CuriousJC/ascend-duel` on `refs/heads/main` and `refs/tags/v*`, so a
fork's pull request cannot assume it; anyone with write access to the repository can, by pushing a
`v*` tag. Nothing in `mgmt/` holds a secret — the account ID is derived at run time and lives
only in the `SOUNDS_ROLE_ARN` variable.

## Releasing

Both release build jobs run `.github/scripts/fetch-private-assets.sh` and then
`go run ./tools/privateassets -require`, **both gated on the `SOUNDS_BUCKET` repository
variable**. Unset, a release fetches nothing and ships with no sound; set, the fetch syncs
`s3://$SOUNDS_BUCKET/sounds/` (recursive, so the pack folders come down as they are) and the
guard refuses a build missing a manifested file or holding different bytes. CI runs neither, and
passes on a clone with no sounds at all. The fetch takes OIDC credentials, never stored keys —
see the script's header.

**The manifest a release builds is the manifest at that commit**, so the bucket has to hold
exactly what the committed manifest lists. `privateassets/mgmt/push-sounds.ps1` (`-DryRun` first) is
the whole push: it runs the Drive sync and `-require` first, mirrors `ascend_assets/` to the bucket
with `--delete`, and says when `manifest.json` is left uncommitted. Push and commit the manifest
together. **There is no version prefix**: the bucket is the current set, so an old commit
rebuilt against it fails `-require` — accepted, since only new releases fetch sounds. The bucket
is versioned, so a deleted file is recoverable. `privateassets/mgmt/setup-aws.ps1` creates the
bucket and the read-only release role, and is safe to re-run; the role's trust and read policies
are in it, so a change to who can read the sounds is a reviewed diff.

## What not to do

- **Never `git add` anything under `privateassets/audio/` but the two text files.** The ignore
  rules make that the default; `git check-ignore -v` confirms a path.
- **Never put the script, an original, or a working file inside `audio/`** — it embeds.
- **Never write a count of sounds or packs** into a doc, a skill or CLAUDE.md. The manifest is the
  list and `go run ./tools/privateassets` reads it.
- **Never reason about where a sound came from** beyond what `bundle-names.txt` and the pack names
  say — provenance is the owner's business.
