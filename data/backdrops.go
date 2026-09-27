package data

// The backdrops: **the painted place a duel is fought in front of**, authored per motif.
//
// A motif's backdrops live beside its creatures, in `motifs/<motif>/backdrops.json`, and a
// backdrop belongs to one of the floor's three rooms by the same `Tier` a creature does. So a
// goblin floor's outer chamber is fought in a goblin outer room, and the three rooms of one floor
// are three different places — a player can tell at a glance which room they are standing in,
// because the tier layer of docs/art/background_art_prompt.MD puts a different door in each.
//
// **One record is one room drawn once per element.** `Draw` is the room and `ElementDraw` is what
// each element does to *this* room, exactly as a creature's picture is briefed: a tinker studio in
// fire runs with molten metal and the same studio in ice is buried in frost. The picture is
// `<Art>-<element>.jpg`, one per affinity.
//
// **Nothing in the rules reads any of this.** A backdrop is a picture, and it may never change an
// outcome; `internal/screens` is its only reader.

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
)

// Backdrop is one room a motif's fights can be drawn in front of.
type Backdrop struct {
	// Backdrop is the key, unique across every motif, and it must read `<motif>-<tier>-<slug>` —
	// checked, as a creature record's is.
	Backdrop string `json:"Backdrop"`

	// Name is what a review page calls the place. Nothing in the game prints it.
	Name string `json:"Name"`

	// Tier is which of the floor's three rooms this is: outer, inner or boss. The portal room is
	// `boss`, the same word the creature standing on it carries.
	Tier string `json:"Tier"`

	// Art is the stem of this room's picture family, and it must read `<motif>-<tier>-<slug>`
	// like the key: the tier's door is painted into the picture, so the name says which tier it was
	// drawn for. The picture drawn is `<Art>-<element>`, so a missing one falls back to the default
	// backdrop rather than drawing nothing.
	Art string `json:"Art"`

	// Affinities is which elements this room is drawn in — a non-empty subset of
	// AffinityElements, no repeats. **The game reads this list and never reads ElementDraw**,
	// which is what keeps the brief an authored-and-ignored field.
	Affinities []string `json:"Affinities"`

	// Draw is the room itself, whatever element it is drawn in: what the place is and what is lying
	// about in it, never who is in it. Ignored by the game.
	Draw string `json:"Draw"`

	// ElementDraw is what each element does to this room, keyed by element name. Ignored by the
	// game. A key the backdrop does not take as an affinity is refused — direction for a picture
	// nothing will ever draw — and a missing one is a brief nobody has written yet.
	ElementDraw map[string]string `json:"ElementDraw"`
}

// ArtKey is the picture this backdrop draws in an element.
func (b Backdrop) ArtKey(element string) string {
	if b.Art == "" || element == "" {
		return DefaultBackgroundArt
	}
	return b.Art + "-" + element
}

// HasAffinity reports whether this room is drawn in an element.
func (b Backdrop) HasAffinity(element string) bool {
	for _, a := range b.Affinities {
		if a == element {
			return true
		}
	}
	return false
}

// Brief is the authored art direction for one backdrop drawn in one element: the room, then what
// the element does to it. The style and the tier's door are layers one and two and live in
// docs/art/background_art_prompt.MD; these are three and four.
//
// **One function so a review sheet and a generated prompt cannot assemble it differently**, as
// MotifData.Brief is for a creature. An unwritten layer is left out rather than printed as a gap.
func (b Backdrop) Brief(element string) []string {
	var out []string
	for _, part := range []string{b.Draw, b.ElementDraw[element]} {
		if written(part) {
			out = append(out, part)
		}
	}
	return out
}

// DefaultBackgroundArt is what a fight with no backdrop to fit it draws: a room whose motif has
// none authored for its tier and element, or one whose picture has not been painted yet.
//
// **It carries no door on purpose.** The three tiers are told apart by what stands at the far side
// of the room, so a fight drawn on this is one the owner can see has no backdrop of its own.
const DefaultBackgroundArt = "default-background"

// BackdropFor is the picture a fight is drawn in front of: one of this motif's backdrops for the
// tier and the element, or the default when it has none.
//
// **The choice is derived from the run, the floor and the room, never rolled.** A run code always
// shows the same places, with no stream to advance, because a picture may never change an outcome
// and so has nothing to share a cursor with. It is the crack pattern's rule: a presentation choice
// that must be stable is a function of what it is about.
//
// **Candidates are walked in file order**, which is authored and fixed, so the hash lands on the
// same room however the map above this was iterated.
func (m MotifData) BackdropFor(tier, element string, runSeed int64, floor int) string {
	var candidates []Backdrop
	for _, b := range m.Backdrops {
		if b.Tier == tier && b.HasAffinity(element) {
			candidates = append(candidates, b)
		}
	}
	if len(candidates) == 0 {
		return DefaultBackgroundArt
	}

	h := fnv.New64a()
	h.Write([]byte(strconv.FormatInt(runSeed, 10) + "/" + strconv.Itoa(floor) + "/" + tier + "/" + element))
	return candidates[h.Sum64()%uint64(len(candidates))].ArtKey(element)
}

// BackdropCoverage is how many backdrops can be drawn behind each fight of a motif, as
// [tier][element] — Coverage's shape, for the rooms rather than the creatures.
//
// **Nothing refuses a hole here.** A fight with no backdrop draws the default, which is correct
// and visible; this is what a review page reads to say what is left to paint.
func BackdropCoverage(m MotifData) Coverage {
	c := Coverage{Motif: m.Motif}
	for _, b := range m.Backdrops {
		ti, ok := TierIndex(b.Tier)
		if !ok {
			continue
		}
		for _, a := range b.Affinities {
			if ai, ok := AffinityIndex(a); ok {
				c.Counts[ti][ai]++
			}
		}
	}
	return c
}

// checkBackdrop is everything refusable about one backdrop.
func checkBackdrop(m MotifData, b Backdrop, file string) {
	where := file + ": " + b.Backdrop

	if b.Backdrop == "" {
		panic(file + ": a backdrop in " + m.Motif + " has no key")
	}
	if b.Name == "" {
		panic(where + " has no name")
	}
	if _, ok := TierIndex(b.Tier); !ok {
		panic(where + " is tier " + b.Tier + ", which is not one of outer, inner or boss")
	}
	if want := m.Motif + "-" + b.Tier + "-"; !strings.HasPrefix(b.Backdrop, want) {
		panic(where + " should be keyed " + want + "<slug>")
	}
	if b.Art == "" {
		panic(where + " names no art family")
	}
	// **The tier is painted into a room** — the door or the portals at the far side — so a picture
	// belongs to one tier and its name says which. A room moved to another tier fails here until its
	// art is renamed, which is the reminder that it has to be repainted.
	if want := m.Motif + "-" + b.Tier + "-"; !strings.HasPrefix(b.Art, want) {
		panic(where + " draws the art family " + b.Art + ", which should read " + want + "<slug>")
	}
	checkAffinities(b.Affinities, where)
	for element := range b.ElementDraw {
		if !b.HasAffinity(element) {
			panic(fmt.Sprintf("%s writes direction for %s, which it is never drawn in — it takes %s",
				where, element, strings.Join(b.Affinities, ", ")))
		}
	}
}
