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
	if muted {
		p.SetVolume(0)
	} else {
		p.SetVolume(volume)
	}
	p.Play()

	player = p
	return nil
}

// muted is whether the score is currently silenced. Package state alongside the player,
// because it is a property of the one thing this package owns.
//
// **It starts true: the game boots silent and the player turns the score on.** Music that
// begins on its own is the first thing a new player reaches for a control to stop, and there
// is no settings screen to stop it from — the mute button in the chrome is the whole vocabulary.
// Start reads this when it opens the device, so nothing is heard before the button is touched.
var muted = true

// Available reports whether there is anything to mute.
//
// **Opening the audio device is allowed to fail** — a machine with no sound card still plays
// the game — and when it does, `player` is nil and every call below is a no-op. A mute
// control that silently did nothing would be worse than none, so the caller asks this and
// disables it instead.
func Available() bool { return player != nil }

// Muted reports whether the score is silenced.
func Muted() bool { return muted }

// SetMuted silences the score or restores it.
//
// **Volume, not Pause, and the difference is what a mute button means.** Pausing would hold
// the score at the bar it was on and resume from there, so unmuting halfway through a duel
// would drop the player back into a phrase they had already heard. Muting a track that keeps
// running puts them wherever the music would have got to, which is the behaviour every other
// mute control in the world has.
//
// The flag is set whether or not there is a player, so the state survives a machine with no
// audio device — see Available.
func SetMuted(m bool) {
	muted = m
	if player == nil {
		return
	}
	if m {
		player.SetVolume(0)
		return
	}
	player.SetVolume(volume)
}
