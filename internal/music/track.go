package music

// track.go — the recorded loops, beside the synthesized score.
//
// **A track is a looping player with a name and nothing else.** Where the bytes came from is the
// caller's business: `main` loads the loops out of `privateassets/`, this package never learns
// that, and a track and the score are the same kind of thing once they are here.
//
// **A track that was never loaded falls back to the score.** That is what makes a machine with
// no bundle synced play the game rather than sit in silence on one screen — absent audio is a
// supported state.

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// Score is what the synthesized score is called.
//
// **A name rather than the empty string**, because "the music that plays nearly everywhere" is
// the most important entry in `internal/game`'s screen table and it was the one entry that could
// not be written down there. A zero value standing for a real thing is the pattern this project
// rejects everywhere else — a run seed of zero is the run 000000 rather than "unset".
//
// It is not a loadable name: LoadTrack refuses it, so nothing can take the score's seat.
const Score = "ascending"

// tracks holds every loaded loop by name. Package state beside `player` and `level`, for the
// reason `player` is: an audio.Player carries a finalizer that stops playback, so one the
// garbage collector can reach falls silent at an unpredictable moment.
var tracks = map[string]*audio.Player{}

// current names what is sounding, and starts at the score because the score starts playing.
// **There is no "nothing" state**: something is always sounding unless the device failed to
// open, and a separate name for silence would be a second way of saying the volume is zero.
var current = Score

// stream is what the three decoders have in common: bytes to read, and a length to loop at.
// Each package returns its own *Stream type, so the dispatch below needs a shape rather than a
// type.
type stream interface {
	io.ReadSeeker
	Length() int64
}

// decoders is every format this can take, by lower-case extension. **It is the same three
// Ebitengine decodes and the same three privateassets.Audio hands over**, and the three lists
// have to agree: a file the loader skips and the embed carries is a manifested sound that is
// shipped and never heard — which is silence on one screen and nothing going red anywhere.
//
// The loops ship as WAV today, which costs size — a minute of 16-bit stereo is about ten
// megabytes in the binary where the Ogg of it is a few hundred kilobytes. That is a conversion
// on the way *in* to privateassets/ rather than a second code path here, and this table is what
// makes it a conversion rather than a change to the game.
var decoders = map[string]func(int, io.Reader) (stream, error){
	".wav": func(rate int, r io.Reader) (stream, error) { return wav.DecodeWithSampleRate(rate, r) },
	".ogg": func(rate int, r io.Reader) (stream, error) { return vorbis.DecodeWithSampleRate(rate, r) },
	".mp3": func(rate int, r io.Reader) (stream, error) { return mp3.DecodeWithSampleRate(rate, r) },
}

// LoadTrack decodes one audio file and makes a looping player for it, without playing it.
//
// **It takes a filename rather than a name**, because the extension is what picks the decoder
// and the stem is what the track is called — `dark-fantasy-01.wav` is the track
// `dark-fantasy-01`. That is the same cut privateassets.Audio makes for its collision check and
// the same one internal/game's screen table is written against.
//
// **The file's rate has to be the context's** — 44100, see sampleRate. A file at another rate is
// resampled by the decoder, which is correct but is work done at startup for something a
// conversion should have settled.
//
// **The whole file is the loop.** These are authored as loops, so there is no crossfade and no
// tail: unlike the score, which is rendered with a blended joint because a synthesized phrase
// does not end where it began.
//
// Errors are worth reporting and never worth quitting over, exactly as Start's are.
func LoadTrack(file string, w []byte) error {
	ext := strings.ToLower(filepath.Ext(file))
	name := strings.TrimSuffix(file, filepath.Ext(file))

	if name == Score {
		return fmt.Errorf("music: %q is the synthesized score and cannot be loaded over", name)
	}
	decode, known := decoders[ext]
	if !known {
		return fmt.Errorf("music: %s is a %q, which nothing here decodes", file, ext)
	}
	if _, taken := tracks[name]; taken {
		return nil
	}

	s, err := decode(sampleRate, bytes.NewReader(w))
	if err != nil {
		return fmt.Errorf("music: reading the track %q: %w", name, err)
	}

	// One context per process, and it panics if made twice. Same check Start makes, so neither
	// has to be the one that owns audio setup.
	ctx := audio.CurrentContext()
	if ctx == nil {
		ctx = audio.NewContext(sampleRate)
	}

	p, err := ctx.NewPlayer(audio.NewInfiniteLoop(s, s.Length()))
	if err != nil {
		return fmt.Errorf("music: opening the audio device for %q: %w", name, err)
	}
	p.SetVolume(level * fullVolume)

	tracks[name] = p
	return nil
}

// PlayTrack switches to a loaded track, or to the synthesized score when handed Score.
//
// **Safe to call every frame**, which is what lets the caller be a table read off the active
// screen rather than a pair of hooks on entering and leaving one. A screen that has just
// changed and a screen that has not are the same call; only a change does anything.
//
// **A name nothing was loaded under plays the score.** A build with no private assets synced
// therefore keeps its music instead of falling silent wherever a track was meant to be — the
// same rule that makes an empty privateassets/audio/ a build rather than a failure.
//
// **The score resumes and a track restarts**, and the two are different on purpose. The score
// runs for the whole session and dropping back into it mid-phrase is what it is for — the same
// argument SetLevel is under, that a track which kept running puts you where the music would
// have got to. A track belongs to a *place*, so arriving there should start it.
func PlayTrack(name string) {
	if _, ok := tracks[name]; name != Score && !ok {
		name = Score
	}
	if name == current {
		return
	}

	if now := sounding(); now != nil {
		now.Pause()
	}

	current = name

	next := sounding()
	if next == nil {
		return
	}
	if name != Score {
		if err := next.Rewind(); err != nil {
			// Not worth stopping for: a loop that failed to rewind plays from wherever it was,
			// which is music rather than silence.
			log.Printf("music: rewinding %q: %v", name, err)
		}
	}
	next.SetVolume(level * fullVolume)
	next.Play()
}

// Playing reports what is sounding, by name. Score is the synthesized one.
func Playing() string { return current }

// sounding is the player behind `current`. It is nil when the audio device never opened, which
// every caller has to survive — see Available.
func sounding() *audio.Player {
	if current == Score {
		return player
	}
	return tracks[current]
}
