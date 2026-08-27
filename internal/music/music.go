package music

import (
	"bytes"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// fullVolume is what a slider at the top of its travel actually sets the player to.
//
// **The scale is not linear from 0 to 1 because the ceiling is not 1.** This is background
// music under a game that will eventually have combat sounds over it, so the loudest the score
// is ever meant to be is a third of the device's range. The settings screen offers 0..1 and
// this is the number it is multiplied by, which is what keeps "how loud is the music allowed to
// get" one decision in one place rather than a figure typed into a scene.
const fullVolume = 0.35

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
	p.SetVolume(level * fullVolume)
	p.Play()

	player = p
	return nil
}

// level is how loud the score is, from 0 (silent) to 1 (fullVolume). Package state alongside
// the player, because it is a property of the one thing this package owns.
//
// **It starts at zero: the game boots silent and the player turns the score up.** Music that
// begins on its own is the first thing a new player reaches for a control to stop. Start reads
// this when it opens the device, so nothing is heard before the setting is touched — and the
// settings screen writes the saved level in before the device is opened, so a returning player
// gets back the level they chose rather than silence.
//
// **There is no separate mute flag** *(owner's call, 2026-08-27)*. A mute latch and a volume bar
// are two controls over one number that have to be kept from disagreeing; zero on the bar is
// silence, and it is the only way to be silent.
var level float64

// Available reports whether there is anything to hear.
//
// **Opening the audio device is allowed to fail** — a machine with no sound card still plays
// the game — and when it does, `player` is nil and every call below is a no-op. A volume
// control that silently did nothing would be worse than none, so the caller asks this and
// disables it instead.
func Available() bool { return player != nil }

// Level reports how loud the score is, from 0 to 1.
func Level() float64 { return level }

// SetLevel sets how loud the score is, clamped to 0..1.
//
// **Volume, not Pause, and the difference is what turning the music down means.** Pausing would
// hold the score at the bar it was on and resume from there, so coming back from silence halfway
// through a duel would drop the player into a phrase they had already heard. A track that keeps
// running puts them wherever the music would have got to, which is the behaviour every other
// volume control in the world has.
//
// The level is recorded whether or not there is a player, so the setting survives a machine with
// no audio device — see Available — and so that a level restored from the profile before Start
// is what Start opens the device at.
func SetLevel(l float64) {
	level = clamp01(l)
	if player == nil {
		return
	}
	player.SetVolume(level * fullVolume)
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}
