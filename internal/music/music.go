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

import (
	"bytes"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// volume is the score's resting level. It is low on purpose: this is background music
// under a game that will eventually have combat sounds over it, and the player has no
// way to turn it down yet.
const volume = 0.35

// player is held at package level so the garbage collector cannot reach it. An
// audio.Player carries a finalizer that stops playback, so a player kept only in a
// local variable falls silent at an unpredictable moment after Start returns.
var player *audio.Player

// Start decodes the given Standard MIDI File, renders it, and begins looping it.
//
// It is safe to call more than once; the second call does nothing. Errors are worth
// reporting but never worth quitting over — a machine with no sound device should
// still play the game, so callers log the error and carry on.
func Start(midi []byte) error {
	if player != nil {
		return nil
	}

	sc, err := parseSMF(midi)
	if err != nil {
		return fmt.Errorf("music: reading the score: %w", err)
	}
	pcm, loopBytes := render(sc)

	// One context per process, and it panics if made twice. Checking for an existing
	// one means this package does not have to be the thing that owns audio setup.
	ctx := audio.CurrentContext()
	if ctx == nil {
		ctx = audio.NewContext(sampleRate)
	}

	// The reader covers more bytes than loopBytes: the extra is a copy of the opening,
	// which is what InfiniteLoop crossfades the joint with. See blendSeconds.
	p, err := ctx.NewPlayer(audio.NewInfiniteLoop(bytes.NewReader(pcm), loopBytes))
	if err != nil {
		return fmt.Errorf("music: opening the audio device: %w", err)
	}
	p.SetVolume(volume)
	p.Play()

	player = p
	return nil
}
