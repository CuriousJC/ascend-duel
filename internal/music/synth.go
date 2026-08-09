package music

import "math"

// The synthesiser. It turns a parsed score into signed 16-bit stereo PCM, which is the
// one audio format Ebitengine will take without a decoder.
//
// **There is no recorded audio and no SoundFont anywhere in this.** Every sample is
// computed from the notes in the file, the same argument that makes the generated
// glyphs in internal/systems worth their code: generated output has no provenance
// question, and this is a game that has to be sellable. The cost is that it sounds
// like a synthesiser rather than an orchestra, which for a score of two synth basses
// and a drum part is close to what was asked for anyway.
//
// This file may not import Ebitengine. Rendering is pure arithmetic over a byte slice
// and wants to stay testable without a window; only music.go touches the audio device.

const (
	// sampleRate is deliberately untyped so it can be an int for the audio context and
	// a float64 in the arithmetic below without a conversion at every use.
	sampleRate = 44100

	frameBytes = 4 // two channels, two bytes each

	// percussionChannel is MIDI channel 10, counting from zero. Its notes are drum
	// names rather than pitches, which is why it gets a different voice entirely.
	percussionChannel = 9

	// tailSeconds is how far past the loop end the render runs, so the release of a
	// note that finishes on the last beat is captured rather than cut. It is folded
	// back over the start, so that release rings on across the loop joint.
	tailSeconds = 1.5

	// blendSeconds is a copy of the loop's opening appended after the loop end.
	// audio.NewInfiniteLoop crossfades the joint using whatever follows it; giving it
	// the loop start means it crossfades the material with itself and cannot smear.
	blendSeconds = 0.1

	// targetPeak leaves headroom below full scale. The whole mix is normalised to it,
	// so adding a voice changes the balance rather than the loudness.
	targetPeak = 0.82
)

// General MIDI percussion keys used by this score. Anything else falls back to a
// generic hit rather than going silent — a drum nobody hears is harder to notice than
// one that sounds wrong.
const (
	keyElectricSnare = 40
	keyLowFloorTom   = 41
)

// note is one sounding note with its channel's state captured at the moment it
// started, so a later volume or pan change cannot reach back and alter it.
type note struct {
	start, end int // sample frames; end is where the key was released
	ch         uint8
	program    int
	key, vel   int
	volume     float64 // channel volume times expression, 0..1
	pan        float64 // -1 hard left, +1 hard right
	seed       uint16  // deterministic noise seed, derived from the start frame
}

// render turns a score into the PCM bytes to loop, plus the loop length in bytes. The
// returned slice is longer than that length: the extra is the blend material described
// at blendSeconds.
func render(sc *score) (pcm []byte, loopBytes int64) {
	loopFrames := sc.frameAt(loopTicks(sc))
	tailFrames := int(tailSeconds * sampleRate)

	left := make([]float64, loopFrames+tailFrames)
	right := make([]float64, loopFrames+tailFrames)
	for _, n := range collectNotes(sc) {
		n.renderInto(left, right)
	}

	// Fold the tail over the start. A note still decaying when the loop wraps should
	// keep decaying over the opening bar, so the tail is *added* to the front rather
	// than crossfaded with it — the two sounds genuinely overlap in a live performance
	// of a loop, and summing is what overlapping means.
	for i := loopFrames; i < len(left); i++ {
		left[i-loopFrames] += left[i]
		right[i-loopFrames] += right[i]
	}
	left, right = left[:loopFrames], right[:loopFrames]

	normalise(left, right)

	blendFrames := int(blendSeconds * sampleRate)
	if blendFrames > loopFrames {
		blendFrames = loopFrames
	}
	out := make([]byte, (loopFrames+blendFrames)*frameBytes)
	for i := 0; i < loopFrames+blendFrames; i++ {
		f := i % loopFrames
		putSample(out[i*frameBytes:], left[f])
		putSample(out[i*frameBytes+2:], right[f])
	}
	return out, int64(loopFrames * frameBytes)
}

// loopTicks rounds the score's extent to the nearest whole bar. A loop has to end on a
// barline to come round in time, and a composer's last note-off usually sits a few
// ticks either side of one — rounding to the *nearest* bar trims a stray tail where
// rounding up would insert a bar of silence and rounding down could lose a beat.
func loopTicks(sc *score) int {
	if sc.barTicks <= 0 {
		return sc.lastTick
	}
	bars := (sc.lastTick + sc.barTicks/2) / sc.barTicks
	if bars < 1 {
		bars = 1
	}
	return bars * sc.barTicks
}

// frameAt converts an absolute tick into a sample frame, walking the tempo map.
func (sc *score) frameAt(tick int) int {
	seconds, prev, us := 0.0, 0, float64(defaultTempo)
	for _, t := range sc.tempoChanges() {
		if t.tick >= tick {
			break
		}
		seconds += float64(t.tick-prev) / float64(sc.division) * us / 1e6
		prev, us = t.tick, float64(t.a)
	}
	seconds += float64(tick-prev) / float64(sc.division) * us / 1e6
	return int(seconds * sampleRate)
}

// collectNotes replays the event list to pair every note-on with its note-off, keeping
// per-channel program, volume, expression and pan as it goes.
func collectNotes(sc *score) []note {
	type channel struct {
		program            int
		volume, expression float64
		pan                float64
	}
	var chans [16]channel
	for i := range chans {
		// MIDI's own power-on defaults: volume 100, expression 127, pan centred.
		chans[i] = channel{volume: 100.0 / 127.0, expression: 1}
	}

	var notes []note
	open := map[uint16]int{} // channel and key to an index in notes
	for _, e := range sc.events {
		c := &chans[e.ch]
		switch e.kind {
		case evProgram:
			c.program = e.a
		case evController:
			switch e.a {
			case ccVolume:
				c.volume = float64(e.b) / 127
			case ccExpression:
				c.expression = float64(e.b) / 127
			case ccPan:
				c.pan = (float64(e.b) - 64) / 63
			}
		case evNoteOn:
			id := uint16(e.ch)<<8 | uint16(e.a)
			// A second note-on for a key already sounding ends the first one, so an
			// unmatched note-off cannot leave a voice droning to the end of the loop.
			if i, ok := open[id]; ok {
				notes[i].end = sc.frameAt(e.tick)
			}
			start := sc.frameAt(e.tick)
			open[id] = len(notes)
			notes = append(notes, note{
				start:   start,
				end:     -1,
				ch:      e.ch,
				program: c.program,
				key:     e.a,
				vel:     e.b,
				volume:  c.volume * c.expression,
				pan:     c.pan,
				seed:    uint16(start),
			})
		case evNoteOff:
			id := uint16(e.ch)<<8 | uint16(e.a)
			if i, ok := open[id]; ok {
				notes[i].end = sc.frameAt(e.tick)
				delete(open, id)
			}
		}
	}

	// Anything still held when the file runs out is released at the end of it.
	end := sc.frameAt(sc.lastTick)
	for i := range notes {
		if notes[i].end < 0 {
			notes[i].end = end
		}
	}
	return notes
}

// renderInto generates a note and adds it to the mix. It is additive rather than
// assigning, which is the whole of the polyphony: overlapping notes are summed and the
// normalise pass afterwards deals with the level.
func (n note) renderInto(left, right []float64) {
	var mono []float64
	if n.ch == percussionChannel {
		mono = n.drum()
	} else {
		mono = n.pulse()
	}

	// Velocity squared, because loudness is perceived closer to the square of
	// amplitude than to it — a linear map makes soft notes vanish.
	v := float64(n.vel) / 127
	amp := n.volume * v * v

	// Equal-power panning: a sound panned centre keeps the same energy as one panned
	// hard over, where halving both channels would leave a hole in the middle.
	angle := (n.pan + 1) * math.Pi / 4
	gl, gr := math.Cos(angle), math.Sin(angle)

	for i, s := range mono {
		j := n.start + i
		if j < 0 {
			continue
		}
		if j >= len(left) {
			break
		}
		left[j] += s * amp * gl
		right[j] += s * amp * gr
	}
}

// timbre is a pulse voice: how wide the pulse is, how bright, and how it opens and
// closes. Duty is what separates one synth bass from another — a square is hollow, a
// narrow pulse is nasal and reedy.
type timbre struct {
	duty    float64
	cutoff  float64 // one-pole low pass, in Hz
	gain    float64
	attack  float64
	decay   float64
	sustain float64
	release float64
}

// timbreFor maps a General MIDI program onto a voice. Only the two synth basses this
// score uses are distinguished; everything else gets the square, which is the safest
// thing to be wrong with.
func timbreFor(program int) timbre {
	base := timbre{duty: 0.5, cutoff: 3200, gain: 0.55, attack: 0.004, decay: 0.06, sustain: 0.72, release: 0.08}
	switch program {
	case 38: // Synth Bass 1 — the square, carrying the line
		return base
	case 39: // Synth Bass 2 — narrower and darker, so it reads as a second voice
		base.duty, base.cutoff, base.gain = 0.28, 2000, 0.45
		return base
	}
	return base
}

// pulse generates one pitched note as a band-limited pulse wave.
func (n note) pulse() []float64 {
	t := timbreFor(n.program)

	held := n.end - n.start
	if held < 1 {
		held = 1
	}
	out := make([]float64, held+int(t.release*sampleRate)+1)

	// Concert A at MIDI key 69, twelve equal steps to the octave.
	freq := 440 * math.Pow(2, float64(n.key-69)/12)
	dt := freq / sampleRate

	// One-pole low pass, expressed as how far the filter moves toward its input each
	// sample. It rounds the corners a pulse wave is nothing but.
	a := 1 - math.Exp(-2*math.Pi*t.cutoff/sampleRate)

	phase, lp := 0.0, 0.0
	for i := range out {
		s := 1.0
		if phase >= t.duty {
			s = -1
		}
		// PolyBLEP: a pulse built from hard steps has infinite bandwidth and folds
		// back over Nyquist as a metallic buzz that tracks the note wrongly. Rounding
		// each of the two edges with a polynomial over one sample removes most of it,
		// which additive synthesis would also do at fifty times the cost.
		s += polyBlep(phase, dt)
		s -= polyBlep(frac(phase+1-t.duty), dt)

		lp += a * (s - lp)
		out[i] = lp * t.gain * t.envelope(i, held)
		phase = frac(phase + dt)
	}
	return out
}

// envelope is the note's shape over time: a fast rise, a fall to a sustained level
// while the key is held, then a release once it is let go.
func (t timbre) envelope(i, held int) float64 {
	attack := int(t.attack * sampleRate)
	if attack > held {
		attack = held
	}
	decay := int(t.decay * sampleRate)
	release := int(t.release * sampleRate)

	switch {
	case i < attack:
		return float64(i) / float64(attack)
	case i < held:
		if d := i - attack; d < decay {
			return 1 - float64(d)/float64(decay)*(1-t.sustain)
		}
		return t.sustain
	default:
		p := float64(i-held) / float64(release)
		if p >= 1 {
			return 0
		}
		return t.sustain * (1 - p)
	}
}

// polyBlep returns the correction to add at a waveform discontinuity, given the phase
// and how far the phase advances per sample. It is zero everywhere except within one
// sample either side of the edge, so most of the loop above pays nothing for it.
func polyBlep(t, dt float64) float64 {
	switch {
	case t < dt:
		t /= dt
		return t + t - t*t - 1
	case t > 1-dt:
		t = (t - 1) / dt
		return t*t + t + t + 1
	}
	return 0
}

func frac(x float64) float64 {
	return x - math.Floor(x)
}

// drum generates one percussion hit. Percussion ignores the note-off entirely: a drum
// decays on its own and holding the key does not sustain it.
func (n note) drum() []float64 {
	switch n.key {
	case keyLowFloorTom:
		return tom(n.seed)
	default:
		return snare(n.seed)
	}
}

// snare is filtered noise with a short tonal body under it — the noise is the snares
// rattling, the body is the drum head.
func snare(seed uint16) []float64 {
	out := make([]float64, int(0.22*sampleRate))
	rng := newNoise(seed)
	phase, hp, prev := 0.0, 0.0, 0.0
	for i := range out {
		t := float64(i) / sampleRate
		s := rng.next()

		// One-pole high pass, so the rattle sits above the bass rather than fighting it.
		hp = 0.85 * (hp + s - prev)
		prev = s

		body := math.Sin(2 * math.Pi * phase)
		phase = frac(phase + 185.0/sampleRate)

		out[i] = hp*0.80*math.Exp(-t/0.055) + body*0.35*math.Exp(-t/0.030)
	}
	return out
}

// tom is a sine whose pitch falls away as it decays. That fall is the whole character
// of a tom; at a fixed pitch the same envelope is just a beep.
func tom(seed uint16) []float64 {
	out := make([]float64, int(0.35*sampleRate))
	rng := newNoise(seed)
	phase := 0.0
	for i := range out {
		t := float64(i) / sampleRate
		f := 62 + 58*math.Exp(-t/0.06)
		phase = frac(phase + f/sampleRate)
		out[i] = math.Sin(2*math.Pi*phase)*math.Exp(-t/0.16) + rng.next()*0.12*math.Exp(-t/0.012)
	}
	return out
}

// noise is a fifteen-bit linear feedback shift register, the same trick the NES sound
// hardware used. It is here rather than math/rand because the project's determinism
// rule forbids the package-level generator outright, and threading a *rand.Rand into
// the synth would buy nothing — a shift register is already reproducible, needs no
// seeding ceremony, and sounds right for the job.
type noise struct{ reg uint16 }

func newNoise(seed uint16) *noise {
	if seed == 0 {
		seed = 1 // an all-zero register only ever produces zero
	}
	return &noise{reg: seed}
}

func (n *noise) next() float64 {
	bit := (n.reg ^ (n.reg >> 1)) & 1
	n.reg = n.reg>>1 | bit<<14
	if n.reg&1 == 1 {
		return 1
	}
	return -1
}

// normalise scales the mix so its loudest moment sits at targetPeak. Both channels are
// scaled by the same factor, or the stereo image would move.
func normalise(left, right []float64) {
	peak := 0.0
	for i := range left {
		if v := math.Abs(left[i]); v > peak {
			peak = v
		}
		if v := math.Abs(right[i]); v > peak {
			peak = v
		}
	}
	if peak == 0 {
		return
	}
	scale := targetPeak / peak
	for i := range left {
		left[i] *= scale
		right[i] *= scale
	}
}

// putSample writes one sample as signed 16-bit little endian, the format Ebitengine's
// NewPlayer takes. Clamping is belt and braces after normalise, but a sample that
// wrapped instead of clipping would be an audible crack rather than a soft edge.
func putSample(b []byte, v float64) {
	s := int(v * 32767)
	if s > 32767 {
		s = 32767
	}
	if s < -32768 {
		s = -32768
	}
	b[0] = byte(s)
	b[1] = byte(s >> 8)
}
