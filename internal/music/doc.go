// Package music plays the game's score.
//
// The score ships as a Standard MIDI File — `assets/ascending.mid`, a kilobyte of
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
package music
