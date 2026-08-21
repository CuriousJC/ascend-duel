// Package music plays the game's score.
//
// The score ships as a Standard MIDI File — `assets/ascending.mid`, a kilobyte of
// notes — and is synthesised to PCM once at startup by the pure-Go code in smf.go and
// synth.go. Ebitengine cannot play MIDI; its audio package decodes MP3, Ogg Vorbis and
// WAV and nothing else. The three ways past that were to convert the file to Ogg
// offline, to embed a SoundFont and a synthesiser library, or to generate the audio
// here, and the third was chosen for the reason the glyph generator exists: **there is
// no asset whose licence has to be cleared before the game can be sold.** A SoundFont
// would have been megabytes with a provenance question attached, and a rendered Ogg
// would have carried that question inside it invisibly.
//
// What it costs is fidelity. This is an oscillator, so the score sounds like a chiptune
// rather than like General MIDI. The file is two synth basses and a drum part, so the
// distance is short — but a score wanting strings would not survive the trip, and that
// is the moment to revisit the decision rather than to add oscillators.
//
// Editing the tune is still just editing the MIDI file. Nothing is baked.
package music
