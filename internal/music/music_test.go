package music

import (
	"math"
	"testing"

	"github.com/curiousjc/ascend-duel/assets"
)

// These tests exist because the score is *generated*, and generated output fails
// quietly. A recorded track that fails to load is silent and obvious; a synthesiser
// handed a file it half-understands plays something, and what it plays is wrong in a
// way nobody notices until they go looking. So the shape of the real file is pinned
// here, and every one of these numbers is a fact about assets/ascending.mid.
//
// Nothing below opens an audio device — parsing and rendering are pure arithmetic over
// byte slices, which is what keeps them testable at all. Only music.go touches
// Ebitengine, and nothing here calls it.

func score0(t *testing.T) *score {
	t.Helper()
	sc, err := parseSMF(assets.LoadMusic()["ascending_mid"])
	if err != nil {
		t.Fatalf("parsing the score: %v", err)
	}
	return sc
}

func TestScoreParsesToTheExpectedShape(t *testing.T) {
	sc := score0(t)

	if sc.division != 480 {
		t.Errorf("division is %d ticks per quarter note, want 480", sc.division)
	}
	if sc.barTicks != 1920 {
		t.Errorf("bar is %d ticks, want 1920 — the file declares 4/4", sc.barTicks)
	}
	if sc.lastTick != 25020 {
		t.Errorf("the last note lands on tick %d, want 25020", sc.lastTick)
	}

	tempos := sc.tempoChanges()
	if len(tempos) != 1 || tempos[0].a != 500000 {
		t.Errorf("tempo map is %v, want one change to 500000us per quarter (120bpm)", tempos)
	}
}

func TestEveryNoteOnIsPairedAndOrdered(t *testing.T) {
	sc := score0(t)
	notes := collectNotes(sc)

	if len(notes) != 85 {
		t.Errorf("collected %d notes, want 85", len(notes))
	}

	// An unpaired note-on is the failure this whole pairing pass exists to prevent: it
	// would sustain to the end of the loop and drone under everything after it.
	for i, n := range notes {
		if n.end <= n.start {
			t.Errorf("note %d (channel %d key %d) ends at frame %d, at or before its start %d",
				i, n.ch, n.key, n.end, n.start)
		}
	}

	// Events are sorted before collection, so notes come out in start order. Playback
	// does not depend on this, but a failure here means the sort stopped working.
	for i := 1; i < len(notes); i++ {
		if notes[i].start < notes[i-1].start {
			t.Fatalf("note %d starts at frame %d, before note %d at %d — the event sort is broken",
				i, notes[i].start, i-1, notes[i-1].start)
		}
	}
}

func TestChannelsCarryTheirProgramsAndPercussion(t *testing.T) {
	sc := score0(t)

	// The two bass parts have to reach the synth with their programs attached, or both
	// collapse onto the same default voice and the arrangement loses its second line.
	programs := map[uint8]int{}
	percussion := 0
	for _, n := range collectNotes(sc) {
		if n.ch == percussionChannel {
			percussion++
			continue
		}
		if p, seen := programs[n.ch]; seen && p != n.program {
			t.Errorf("channel %d carries programs %d and %d", n.ch, p, n.program)
		}
		programs[n.ch] = n.program
	}

	if programs[0] != 38 {
		t.Errorf("channel 0 is program %d, want 38 (Synth Bass 1)", programs[0])
	}
	if programs[1] != 39 {
		t.Errorf("channel 1 is program %d, want 39 (Synth Bass 2)", programs[1])
	}
	if percussion != 35 {
		t.Errorf("%d percussion hits, want 35", percussion)
	}
}

func TestLoopIsAWholeNumberOfBars(t *testing.T) {
	sc := score0(t)

	// The last note-off sits 60 ticks past the thirteenth barline. A loop that ended
	// there would drift the beat every time it came round, so it is rounded back to
	// the bar — see loopTicks.
	got := loopTicks(sc)
	if got != 13*1920 {
		t.Errorf("loop is %d ticks, want %d (13 bars)", got, 13*1920)
	}
	if got%sc.barTicks != 0 {
		t.Errorf("loop of %d ticks is not a whole number of %d-tick bars", got, sc.barTicks)
	}
}

func TestRenderProducesAWholeLoopOfAudibleStereo(t *testing.T) {
	sc := score0(t)
	pcm, loopBytes := render(sc)

	if loopBytes%frameBytes != 0 {
		t.Errorf("loop length %d is not a whole number of %d-byte frames", loopBytes, frameBytes)
	}
	// Thirteen bars at 120bpm is 26 seconds exactly.
	if want := int64(26 * sampleRate * frameBytes); loopBytes != want {
		t.Errorf("loop is %d bytes, want %d (26 seconds)", loopBytes, want)
	}
	// The render is longer than the loop: the surplus is the blend material
	// audio.NewInfiniteLoop crossfades the joint with. Without it the loop still plays,
	// but the seam clicks.
	if int64(len(pcm)) <= loopBytes {
		t.Errorf("render is %d bytes with a loop of %d — no blend material follows the loop end",
			len(pcm), loopBytes)
	}

	// Normalisation targets a fixed peak, so a silent render and a clipped one are both
	// caught by looking at the loudest sample.
	peak := 0
	for i := 0; i+1 < len(pcm); i += 2 {
		s := int(int16(uint16(pcm[i]) | uint16(pcm[i+1])<<8))
		if s < 0 {
			s = -s
		}
		if s > peak {
			peak = s
		}
	}
	if want := int(math.Round(targetPeak * 32767)); math.Abs(float64(peak-want)) > 64 {
		t.Errorf("loudest sample is %d, want about %d — the mix is silent, quiet or clipping", peak, want)
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	// The same reason internal/combat pins this: a run has to be reproducible. The
	// synth's only randomness is a shift register seeded from each note's start frame,
	// precisely so two renders of one file cannot differ.
	a, aLoop := render(score0(t))
	b, bLoop := render(score0(t))

	if aLoop != bLoop {
		t.Fatalf("two renders disagree on the loop length: %d and %d", aLoop, bLoop)
	}
	if len(a) != len(b) {
		t.Fatalf("two renders produced %d and %d bytes", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("two renders of the same score differ at byte %d: %d and %d", i, a[i], b[i])
		}
	}
}

func TestRejectsWhatItCannotPlay(t *testing.T) {
	valid := assets.LoadMusic()["ascending_mid"]

	// A file the reader cannot understand has to be an error rather than a best guess.
	// Silence with a log line is recoverable; a score with holes in it is not, because
	// it sounds like a composition choice.
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"not a midi file", []byte("this is not a midi file at all")},
		{"truncated mid-track", valid[:len(valid)/2]},
		{"header only", valid[:14]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if sc, err := parseSMF(tt.data); err == nil {
				t.Errorf("parsed %d bytes into a score of %d events, want an error", len(tt.data), len(sc.events))
			}
		})
	}

	// A good file must still parse after all that, so a broken test fixture cannot pass
	// this file by making everything fail.
	if _, err := parseSMF(valid); err != nil {
		t.Errorf("the real score no longer parses: %v", err)
	}
}

func TestMutingSurvivesHavingNoAudioDevice(t *testing.T) {
	// **Start is allowed to fail** — a machine with no sound card still plays the game — and
	// then `player` is nil. The mute state has to be recorded anyway rather than dropped on
	// the floor, or a control that comes back with the device would disagree with what it
	// shows. Tests never call Start, so this is that case exactly.
	if Available() {
		t.Skip("an audio device was opened; this covers the case where none was")
	}

	defer SetMuted(false)

	if Muted() {
		t.Fatal("the score starts muted")
	}
	SetMuted(true)
	if !Muted() {
		t.Error("muting with no player did not record the state")
	}
	SetMuted(false)
	if Muted() {
		t.Error("unmuting with no player did not record the state")
	}
}
