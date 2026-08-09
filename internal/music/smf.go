package music

import (
	"encoding/binary"
	"fmt"
	"sort"
)

// A Standard MIDI File reader, cut down to the events a score actually uses.
//
// It is deliberately not a general-purpose MIDI library. It reads format 0 and 1 files
// with metrical (ticks-per-quarter) timing, understands notes, programs, three
// controllers, tempo and time signature, and skips everything else by its declared
// length rather than guessing at it. Anything it cannot read is an error, never a
// silent fallback — a score that half-parses would play as music with holes in it, and
// nobody would know which holes were the composer's.

// Meta-event types. Everything else is skipped.
const (
	metaTempo    = 0x51
	metaTimeSig  = 0x58
	metaEndTrack = 0x2F
)

// Controllers the synth honours. A controller outside this set is parsed and dropped.
const (
	ccVolume     = 7
	ccPan        = 10
	ccExpression = 11
)

// defaultTempo is MIDI's own default of 120 bpm, in microseconds per quarter note. A
// file with no tempo event is playing at this speed whether it says so or not.
const defaultTempo = 500000

type eventKind uint8

// The order of these constants is the order events at the same tick are applied in, so
// a channel's volume lands before a note reads it, and a note-off lands before a
// note-on retriggering the same key. Reordering them changes how the score sounds.
const (
	evTempo eventKind = iota
	evProgram
	evController
	evNoteOff
	evNoteOn
)

// event is one thing happening at one tick. a and b carry whatever the kind needs:
// key and velocity for notes, controller and value for a controller, the program
// number or the tempo in microseconds per quarter note for those.
type event struct {
	tick int
	kind eventKind
	ch   uint8
	a, b int
}

// score is a whole file flattened into one tick-ordered event list. Format 1 files
// split a piece across parallel tracks, but nothing downstream cares which track a
// note came from — only when it sounds and on which channel.
type score struct {
	division int // ticks per quarter note
	barTicks int // ticks in one bar, from the time signature; 4/4 unless told otherwise
	events   []event
	lastTick int // the last tick any note starts or stops on: the musical extent
}

func parseSMF(data []byte) (*score, error) {
	r := &reader{d: data}

	if h := string(r.bytes(4)); h != "MThd" {
		return nil, fmt.Errorf("not a Standard MIDI File: header is %q, want \"MThd\"", h)
	}
	if n := r.u32(); n != 6 {
		return nil, fmt.Errorf("MThd chunk is %d bytes, want 6", n)
	}
	format, tracks, division := r.u16(), r.u16(), r.u16()
	if r.err != nil {
		return nil, r.err
	}
	if format != 0 && format != 1 {
		return nil, fmt.Errorf("SMF format %d is not supported, want 0 or 1", format)
	}
	if division&0x8000 != 0 {
		return nil, fmt.Errorf("SMPTE time division is not supported, want ticks per quarter note")
	}
	if division == 0 {
		return nil, fmt.Errorf("time division is zero")
	}

	// 4/4 until a time signature says otherwise. Only the loop length reads this, and a
	// wrong guess there costs a rounded loop point rather than a wrong-sounding score.
	sc := &score{division: division, barTicks: 4 * division}
	for t := 0; t < tracks; t++ {
		if err := sc.readTrack(r, t); err != nil {
			return nil, err
		}
	}
	if r.err != nil {
		return nil, r.err
	}

	// Stable, so two events on the same tick and of the same kind keep the order the
	// tracks were read in. Every tie has to break the same way on every run, because a
	// score that renders differently twice is the audio version of a nondeterministic
	// duel.
	sort.SliceStable(sc.events, func(i, j int) bool {
		if sc.events[i].tick != sc.events[j].tick {
			return sc.events[i].tick < sc.events[j].tick
		}
		return sc.events[i].kind < sc.events[j].kind
	})
	if sc.lastTick == 0 {
		return nil, fmt.Errorf("the file contains no notes")
	}
	return sc, nil
}

func (sc *score) readTrack(r *reader, index int) error {
	if h := string(r.bytes(4)); h != "MTrk" {
		return fmt.Errorf("track %d: chunk header is %q, want \"MTrk\"", index, h)
	}
	length := r.u32()
	if r.err != nil {
		return r.err
	}
	if r.i+length > len(r.d) {
		return fmt.Errorf("track %d: declares %d bytes but only %d remain", index, length, len(r.d)-r.i)
	}
	end := r.i + length

	tick := 0
	status := 0
	for r.i < end && r.err == nil {
		tick += r.varint()
		if r.i >= end {
			break
		}

		// Running status: a byte with the top bit clear is the first data byte of
		// another message of whatever kind came last, with the status byte left out.
		if b := r.d[r.i]; b&0x80 != 0 {
			status = int(b)
			r.i++
		} else if status == 0 {
			return fmt.Errorf("track %d: data byte %#02x before any status byte", index, b)
		}

		switch {
		case status == 0xFF:
			kind := r.u8()
			payload := r.bytes(r.varint())
			switch kind {
			case metaTempo:
				if len(payload) == 3 {
					us := int(payload[0])<<16 | int(payload[1])<<8 | int(payload[2])
					sc.events = append(sc.events, event{tick: tick, kind: evTempo, a: us})
				}
			case metaTimeSig:
				// Numerator over two to the power of the second byte, in quarter notes.
				if len(payload) >= 2 && payload[1] < 8 {
					sc.barTicks = int(payload[0]) * sc.division * 4 / (1 << payload[1])
				}
			case metaEndTrack:
				r.i = end
			}

		case status == 0xF0 || status == 0xF7:
			r.bytes(r.varint()) // system exclusive, skipped whole

		default:
			ch := uint8(status & 0x0F)
			switch status & 0xF0 {
			case 0x80:
				key := r.u8()
				r.u8() // release velocity, unused
				sc.note(event{tick: tick, kind: evNoteOff, ch: ch, a: key})
			case 0x90:
				key, vel := r.u8(), r.u8()
				// A note-on at zero velocity is a note-off. Most files written by a
				// sequencer use this form exclusively, so it is the common path here
				// rather than the special case it looks like.
				if vel == 0 {
					sc.note(event{tick: tick, kind: evNoteOff, ch: ch, a: key})
				} else {
					sc.note(event{tick: tick, kind: evNoteOn, ch: ch, a: key, b: vel})
				}
			case 0xB0:
				num, val := r.u8(), r.u8()
				sc.events = append(sc.events, event{tick: tick, kind: evController, ch: ch, a: num, b: val})
			case 0xC0:
				sc.events = append(sc.events, event{tick: tick, kind: evProgram, ch: ch, a: r.u8()})
			case 0xA0:
				r.bytes(2) // polyphonic aftertouch
			case 0xD0:
				r.bytes(1) // channel aftertouch
			case 0xE0:
				r.bytes(2) // pitch bend
			default:
				return fmt.Errorf("track %d: unknown status byte %#02x at byte %d", index, status, r.i)
			}
		}
	}

	// Trust the chunk length over the end-of-track marker: a track that ran short or
	// long has already been read as far as its header said it went.
	r.i = end
	return nil
}

// note records an event and extends the score's musical extent. Only notes move
// lastTick — a trailing controller or an end-of-track marker sitting a bar past the
// final chord would otherwise stretch the loop into silence.
func (sc *score) note(e event) {
	sc.events = append(sc.events, e)
	if e.tick > sc.lastTick {
		sc.lastTick = e.tick
	}
}

// tempoChanges pulls the tempo map out in tick order, for converting ticks to time.
func (sc *score) tempoChanges() []event {
	var out []event
	for _, e := range sc.events {
		if e.kind == evTempo {
			out = append(out, e)
		}
	}
	return out
}

// reader is a bounds-checked cursor that latches its first error instead of returning
// one from every call. The parse above reads a dozen fields in a row; threading an
// error through each would bury the file format in error handling.
type reader struct {
	d   []byte
	i   int
	err error
}

func (r *reader) fail(format string, args ...any) {
	if r.err == nil {
		r.err = fmt.Errorf(format, args...)
	}
}

// bytes returns n bytes, or n zero bytes once the reader has failed. Returning a slice
// of the right length even on failure is what lets callers index it without checking.
func (r *reader) bytes(n int) []byte {
	if r.err != nil || n < 0 || r.i+n > len(r.d) {
		if r.err == nil {
			r.fail("unexpected end of file at byte %d, wanted %d more", r.i, n)
		}
		if n < 0 {
			n = 0
		}
		return make([]byte, n)
	}
	b := r.d[r.i : r.i+n]
	r.i += n
	return b
}

func (r *reader) u8() int  { return int(r.bytes(1)[0]) }
func (r *reader) u16() int { return int(binary.BigEndian.Uint16(r.bytes(2))) }
func (r *reader) u32() int { return int(binary.BigEndian.Uint32(r.bytes(4))) }

// varint reads MIDI's variable-length quantity: seven bits per byte, top bit set on
// every byte but the last. Four bytes is the format's own maximum.
func (r *reader) varint() int {
	v := 0
	for n := 0; n < 4; n++ {
		b := r.u8()
		v = v<<7 | b&0x7F
		if b&0x80 == 0 {
			return v
		}
	}
	r.fail("variable-length quantity longer than four bytes at byte %d", r.i)
	return 0
}
