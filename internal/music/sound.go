package music

// sound.go — the effects: a short sound played once, over whatever music is sounding.
//
// **An effect is loaded under the cue it plays on, not under its own name.** A screen raises a cue
// — `data.CueRelicToast` — and never learns which file answers it, so `main` reads data/sounds.json
// and hands each sound in under its cues. This package knows neither the catalog nor the screens.
//
// **Every play is its own player, so effects layer.** Three relics toasting in one beat are three
// sounds, each from the start, overlapping as the beats do. The decoded bytes are shared and only
// the player is new, which is what Ebitengine's NewPlayerFromBytes is for.
//
// **Its own level, beside the score's and not under it** *(owner's call)*: a player who wants the
// music low and the relics loud, or the reverse, has two bars. Same rules as the score's: 0..1,
// zero is the only silence, and the ceiling is decided here.
//
// **It may never change an outcome.** A cue is raised by playback, after the round is resolved.

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// soundCeiling is what a sounds bar at the top of its travel sets an effect to. **Above the
// music's**: an effect is a short mark over the bed the music lays down, and one that could only
// ever be as loud as the loop under it would be lost in it.
const soundCeiling = 0.6

// maxLiveSounds bounds how many effects can be sounding at once. **A guard, not a mix**: the toasts
// a round can raise are a handful, and this only stops a screen that raised a cue every frame from
// opening players without end. Past it, a new play is skipped.
const maxLiveSounds = 16

var (
	// cuePCM is each cue's sounds, decoded once at the game's rate, in the order they were loaded.
	cuePCM = map[string][][]byte{}

	// cueTurn is which of a cue's sounds plays next. **Turns rather than a roll**: what a player
	// hears may never spend a stream, and four sounds in rotation already never repeat back to back.
	cueTurn = map[string]int{}

	// liveSounds are the effects still playing, held so the garbage collector cannot stop them —
	// the same reason the score's player is held at package level.
	liveSounds []*audio.Player

	// soundLevel is how loud the effects are, 0 to 1. **It starts at zero**, for the score's
	// reason: a fresh profile is silent and the player turns it up.
	soundLevel float64

	// soundMuted silences the effects without forgetting their level, as muted does the score's.
	soundMuted bool
)

// soundVolume is what an effect is played at: the level, or nothing while muted.
func soundVolume() float64 {
	if soundMuted {
		return 0
	}
	return soundLevel * soundCeiling
}

// SoundMuted reports whether the effects are muted.
func SoundMuted() bool { return soundMuted }

// SetSoundMuted silences the effects or brings them back, the ones already sounding included,
// leaving the level as it was.
func SetSoundMuted(m bool) {
	soundMuted = m
	for _, p := range liveSounds {
		p.SetVolume(soundVolume())
	}
}

// LoadCue decodes one effect and files it under a cue, after any already filed there — a cue with
// several sounds plays them in turn, in loading order. `file` is the sound's filename, whose
// extension picks the decoder.
//
// Errors are worth reporting and never worth quitting over, exactly as LoadTrack's are.
func LoadCue(cue, file string, raw []byte) error {
	ext := strings.ToLower(filepath.Ext(file))
	decode, known := decoders[ext]
	if !known {
		return fmt.Errorf("music: %s is a %q, which nothing here decodes", file, ext)
	}
	s, err := decode(sampleRate, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("music: reading the sound %s: %w", file, err)
	}
	pcm, err := io.ReadAll(s)
	if err != nil {
		return fmt.Errorf("music: reading the sound %s: %w", file, err)
	}
	if audio.CurrentContext() == nil {
		audio.NewContext(sampleRate)
	}
	cuePCM[cue] = append(cuePCM[cue], pcm)
	return nil
}

// PlayCue plays the cue's next sound once, over everything else. **A cue with no sound is
// silence**, and so is a level of zero or a mute: none is an error, since a build without the sounds is a
// build. A silent play does not take a turn.
func PlayCue(cue string) {
	sounds := cuePCM[cue]
	ctx := audio.CurrentContext()
	if len(sounds) == 0 || ctx == nil || soundVolume() == 0 {
		return
	}
	pcm := sounds[cueTurn[cue]%len(sounds)]
	cueTurn[cue]++
	pruneSounds()
	if len(liveSounds) >= maxLiveSounds {
		return
	}
	p := ctx.NewPlayerFromBytes(pcm)
	p.SetVolume(soundVolume())
	p.Play()
	liveSounds = append(liveSounds, p)
}

// pruneSounds lets go of the effects that have finished.
func pruneSounds() {
	kept := liveSounds[:0]
	for _, p := range liveSounds {
		if p.IsPlaying() {
			kept = append(kept, p)
			continue
		}
		_ = p.Close()
	}
	liveSounds = kept
}

// SoundLevel reports how loud the effects are, from 0 to 1.
func SoundLevel() float64 { return soundLevel }

// SetSoundLevel sets how loud the effects are, clamped to 0..1, the ones already sounding included
// — a bar dragged while a toast rings should be heard moving it.
func SetSoundLevel(l float64) {
	soundLevel = clamp01(l)
	for _, p := range liveSounds {
		p.SetVolume(soundVolume())
	}
}
