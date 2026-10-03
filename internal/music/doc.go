// Package music plays the game's score.
//
// The score ships as a Standard MIDI File — `assets/sounds/duello.mid`, a kilobyte of
// notes — and is synthesized to PCM once at startup by the pure-Go code in smf.go and
// synth.go. Ebitengine cannot play MIDI; its audio package decodes MP3, Ogg Vorbis and
// WAV and nothing else. The three ways past that were to convert the file to Ogg
// offline, to embed a SoundFont and a synthesizer library, or to generate the audio
// here, and the third was chosen **to keep the tune in the diff**: a SoundFont is
// megabytes and a rendered Ogg is a binary where a kilobyte of notes will do, so
// editing either means editing something nobody can read.
//
// What it costs is fidelity. This is an oscillator, so the score sounds like a chiptune
// rather than like General MIDI. The file is two synth basses and a drum part, so the
// distance is short — but a score wanting strings would not survive the trip, and that
// is the moment to revisit the decision rather than to add oscillators.
//
// Editing the tune is still just editing the MIDI file. Nothing is baked.
//
// # Tracks
//
// Beside the score this package plays named **tracks**: recorded loops, handed in as whole files
// by whoever loaded them. A track is named by the file's stem and decoded by its extension —
// WAV, Ogg Vorbis or MP3, the three Ebitengine takes — so the format is a property of the file
// rather than a decision the game makes. `main` takes them out of `privateassets/`; nothing here
// knows that, and a track and the score are the same kind of thing once they are in. See
// track.go.
//
// **Which music a screen gets is not decided here.** This package knows how to hold several
// pieces and which one is sounding; `internal/game/score.go` owns the table that says a screen
// plays one, for the reason the settings cog lives in the frame. A name nothing was loaded under
// falls back to the score, which is what lets a build with no bundle synced play everywhere
// rather than fall silent somewhere.
//
// # Working on it
//
// `main` starts the score after assets load, and it loops for the whole session across every screen.
// `oto`, first-party to Ebitengine, is the only dependency the synthesizer needs.
//
//   - **`smf.go` and `synth.go` may not import Ebitengine**, exactly like `internal/combat`. That is
//     what makes them testable, and `music_test.go` pins the shape of the real file — its note count,
//     its channels, its loop length, and a render that is byte-identical twice. **Generated output fails
//     quietly**: a synth handed a file it half-understands plays something, and what it plays is wrong in
//     a way nobody notices.
//   - **No `math/rand`**, per the determinism rules. The drum noise is a 15-bit shift register seeded
//     from each note's start frame, so two renders cannot differ.
//   - **The score resumes and a track restarts.** The score runs for the whole session and dropping back
//     into it mid-phrase is what it is for. A track belongs to a *place*, so arriving there starts it.
//   - **`SetLevel` sets every player, not the one that is sounding.** A paused track keeps the volume it
//     was set to, so a level changed while a loop is up would otherwise be the level the score came back
//     at — the bar and the music disagreeing about the number the bar is the only control for.
//   - **There is no mute, only a level.** `SetLevel(0..1)` is the whole control and zero is the only
//     silence there is; a latch beside a bar would be two controls over one number that then have to be
//     kept from disagreeing. The bar is on the settings screen — a control, never a hotkey.
//   - **`fullVolume` is the ceiling the bar's 1 actually means**, a third of the device's range because
//     this is background music under combat sounds that do not exist yet. "How loud may the score get"
//     stays one decision here rather than a figure typed into a scene.
//   - **Volume, not Pause.** Pausing would hold the score at the bar it was on, so coming back from
//     silence mid-duel would drop the player into a phrase they had already heard.
//   - **Failing to open the audio device is logged, never fatal.** A machine with no sound card still
//     plays the game, and `Available` reports it so the settings screen's volume bar disables itself
//     rather than silently doing nothing.
//   - **The game boots silent for a new player** — a fresh `profile.Settings` has `MusicVolume: 0`, and
//     `main` applies the saved settings *before* `Start` opens the device, so a returning player gets the
//     level they chose rather than a moment of the wrong one.
//
// **Normalize a bought loop on the way in, and never fix a quiet one with the ceiling.** `fullVolume`
// caps the score and every track *together*, so raising it to rescue one loop makes the chiptune
// harsh. The score sits at **-1.7 dBFS peak, -14.7 dBFS RMS**; peak-normalize a track to **-1.0 dBFS**
// and the two land within a decibel or two, which is what stops a player reaching for the bar when the
// screen changes. The gain applied goes in the pack's `Note`, because it means the bytes that ship are
// not the bytes that were bought.
//
// **A track is decoded by its extension, and the size is the thing to watch.** `decoders` is the table
// — the same three formats `privateassets.Audio` hands over, which is why `LoadTrack` is given a
// *filename* rather than a name: a loader accepting less than the embed carries is a manifested sound
// that ships and is never heard. `sampleRate` is 44100 and the loops are 16-bit stereo at exactly that,
// so nothing is resampled — but a minute of WAV is about ten megabytes in the binary where the Ogg
// would be a few hundred kilobytes. **Convert on the way into `privateassets/audio/`, not in the
// loader**, and read the size against the catalog before adding a loop.
package music
