package data

// The roster: sixteen-odd motifs, each a file, each holding the creatures of one themed floor.
//
// **A floor is a motif and an element**, and it holds three fights — the outer chamber, the inner
// chamber and the stairway. So a motif has to be able to field all three of those at whichever
// element the floor took, which is the one thing this file checks that no other catalog loader
// does: see MustCover.
//
// **The records of one motif live in one file, bosses included.** The coverage question is asked
// of a motif, so a motif split across two files is a question neither half can answer and a
// reader has to join by hand.

import (
	"embed"
	"fmt"
	"path"
	"sort"
	"strings"
)

//go:embed motifs/*.json
var motifsFS embed.FS

// The three fights a floor holds, outermost first.
//
// **Written as names and indexed as ordinals.** A tier is a string in the JSON because a file
// outlives the build that wrote it; it is an index in Go because the ascent curve is a function
// of how far into the tower a fight stands, and the tier is the last term of that. TierIndex is
// the one crossing.
const (
	TierOuter = "outer"
	TierInner = "inner"
	TierBoss  = "boss"
)

// TierOrder is the three tiers in the order they are fought. The index into this is the tier's
// contribution to the ascent step, so it is the order that matters rather than the list.
var TierOrder = []string{TierOuter, TierInner, TierBoss}

// TierIndex is where a tier stands within its floor, and whether the name is one at all.
func TierIndex(tier string) (int, bool) {
	for i, t := range TierOrder {
		if t == tier {
			return i, true
		}
	}
	return 0, false
}

// AffinityElements is every element a creature can be instantiated as.
//
// **`basic` is deliberately absent.** A creature takes its floor's element, and a floor has one;
// a creature with no element would be a floor with no theme. The player's deck still holds basic
// cards — that is a different question about a different deck.
//
// **This is the five names written down a second time**, the first being combat.AllElements, and
// the duplication is what the dependency graph costs: this package is the bottom and may not
// import the rules. `internal/decks` holds the test that fails when the two disagree.
var AffinityElements = []string{"fire", "ice", "lightning", "earth", "arcane"}

// AffinityIndex is an element's column in a coverage report, and whether the name is an element
// a creature may take.
func AffinityIndex(element string) (int, bool) {
	for i, e := range AffinityElements {
		if e == element {
			return i, true
		}
	}
	return 0, false
}

// MinCoverage is how many records must be able to field any one (tier, element) fight.
//
// **Two, so no floor is ever the same fight twice.** A motif that can field an ice inner chamber
// with exactly one record deals that creature every time an ice floor of that motif comes up,
// and the floor stops being a draw. What it costs is that a motif is nine records rather than
// three; what it buys is that picking a floor is picking a pool.
const MinCoverage = 2

// MotifData is one themed floor's worth of creatures: the name, the band of floors it may theme,
// and every record that can stand in one of its three rooms.
type MotifData struct {
	// Motif is the key, unique across every file, and it is also the art family and the name of
	// the generator prompt this motif's pictures come from.
	Motif string `json:"Motif"`

	// Name is what a player would call them.
	Name string `json:"Name"`

	// Draw is the art direction every record in this motif shares: the body plan, the silhouette,
	// the materials — what makes the family a family. **Nothing in the game reads it.**
	//
	// **It is here rather than in docs/art/ because it is about these records.** A prompt file
	// holds what is true of no record at all — the canvas, the ground, the light — and a brief
	// kept apart from the records it governs is a brief that gets deleted when the pictures it
	// produced are filed.
	//
	// **It is here rather than on each record because it is true of all of them.** Written nine
	// times it drifts; written nowhere, nine prompts each reinvent what a goblin looks like, which
	// is how a floor's three rooms come to look unrelated.
	Draw string `json:"Draw"`

	// ElementDraw is how each element changes *this* motif, keyed by element name.
	//
	// **Per motif rather than once for the whole roster**, because an element does not mean the
	// same thing to two body plans: an ice slime *is* ice all the way through, and an ice goblin is
	// a goblin wearing frost. One block written globally would make the second one wrong.
	//
	// A key that is not an element is refused. A **missing** one is not: it is a brief nobody has
	// written yet, and the review sheet marks it rather than the launch failing over it.
	ElementDraw map[string]string `json:"ElementDraw"`

	// ValidFloors is the inclusive band of tower floors this motif may theme, as [low, high].
	// A zero band means any floor.
	//
	// **It is motif-level rather than per-record** because a floor takes a whole motif: a motif
	// whose outer creatures were valid on floors 1 to 3 and whose boss was valid on 4 to 6 could
	// never theme a floor at all.
	ValidFloors [2]int `json:"ValidFloors"`

	Records []MotifRecord `json:"Records"`
}

// MotifRecord is one creature: which room it can stand in, which elements it can be dealt as,
// what it is worth at the bottom of the tower, and the cards it holds.
type MotifRecord struct {
	// Record is the key, unique across every motif file, and it must read
	// `<motif>-<tier>-<slug>`. The prefix is checked rather than trusted: a key that disagrees
	// with its own tier is a record the coverage report counts in the wrong column, and the
	// report is what the floor generator trusts.
	Record string `json:"Record"`

	// Name is what the creature is called. It is drawn in the tooltip rather than on the card,
	// because the card is a picture.
	Name string `json:"Name"`

	// Tier is which of the floor's three rooms this record can stand in.
	Tier string `json:"Tier"`

	// Title is the boss's epithet — "the Toll-Taker". Empty on a creature, and refused on one.
	Title string `json:"Title"`

	// Art is the stem of this record's picture family. The face drawn is `<Art>-<element>.png`,
	// so one record carries a picture per element it can be dealt as, and a missing one falls
	// back to the placeholder rather than drawing nothing.
	Art string `json:"Art"`

	// Draw is the subject paragraph an art generator is given for this record. Nothing in the
	// game reads it.
	Draw string `json:"Draw"`

	// Affinities is which elements this record can be instantiated as — a non-empty subset of
	// AffinityElements, no repeats.
	//
	// **A record is dealt as exactly one of them**, chosen by the floor, and its whole deck takes
	// that element. There is no element anywhere on a card: a card is a concept, and the colour
	// belongs to the creature holding it.
	Affinities []string `json:"Affinities"`

	// HP and DMG are the bases, and they are **step-zero quantities** — what this creature is
	// worth in floor one's outer chamber, whatever floor it is actually met on. The ascent curve
	// puts it where it stands; see pyramid.ScaleToFight.
	//
	// So a creature that only appears high in the tower is not written as a high stat line. It is
	// written as the multiple of its neighbours it is meant to be, and the curve does the rest —
	// which is also what keeps the ratio between two motifs fixed however the curve is retuned.
	HP  int `json:"HP"`
	DMG int `json:"DMG"`

	// Actions is the action-point budget a turn is spent out of. **Authored and never scaled**:
	// growing it would hand a creature more cards rather than a harder version of its own.
	Actions int `json:"Actions"`

	// Cards is this record's own deck, authored per record. Two creatures of one motif are two
	// different fights, so they hold different cards rather than the same cards at different
	// weights.
	Cards []CardData `json:"Cards"`
}

// AllowsFloor reports whether this motif may theme a given floor. A zero band means every floor,
// so a motif authored without one is placeable rather than unreachable.
func (m MotifData) AllowsFloor(floor int) bool {
	if m.ValidFloors == [2]int{} {
		return true
	}
	return floor >= m.ValidFloors[0] && floor <= m.ValidFloors[1]
}

// Candidates is every record of one tier that can be dealt as one element, in file order.
//
// **File order rather than sorted**, because the file is authored in an order somebody chose and
// the caller picking from this shuffles it against a seeded stream anyway.
func (m MotifData) Candidates(tier, element string) []MotifRecord {
	var out []MotifRecord
	for _, r := range m.Records {
		if r.Tier == tier && r.HasAffinity(element) {
			out = append(out, r)
		}
	}
	return out
}

// HasAffinity reports whether this record can be dealt as an element.
func (r MotifRecord) HasAffinity(element string) bool {
	for _, a := range r.Affinities {
		if a == element {
			return true
		}
	}
	return false
}

// ArtKey is the picture this record wears when dealt as an element.
func (r MotifRecord) ArtKey(element string) string {
	if r.Art == "" || element == "" {
		return DefaultEnemyArt
	}
	return r.Art + "-" + element
}

// FullName is the name and the title as one string, so the hover and the review sheet cannot
// join the two halves differently.
func (r MotifRecord) FullName() string {
	if r.Title == "" {
		return r.Name
	}
	return r.Name + " " + r.Title
}

// DefaultEnemyArt is the face a record draws when its own picture has not been generated yet.
// One placeholder for every motif and both kinds of record: a per-motif placeholder is a picture
// somebody has to draw before the motif can be looked at.
const DefaultEnemyArt = "default-enemy"

// Coverage is how many records can field each fight of a motif, as [tier][element].
//
// **It is the one report the floor generator trusts.** A floor picks a motif and an element and
// then needs a record for each of the three rooms; a hole here is a floor that cannot be built,
// and the whole point of checking it at init is that the generator never has to ask.
type Coverage struct {
	Motif  string
	Counts [TierCount][AffinityCount]int
}

// The widths of the two closed sets above.
//
// **Constants because Coverage is a value that is copied and compared**, and a slice inside it
// would make two identical reports different objects. `init` below fails the launch if either
// list stops being this long, so appending an element to one place and not the other cannot
// silently shrink the report.
const (
	TierCount     = 3
	AffinityCount = 5
)

func init() {
	if len(TierOrder) != TierCount {
		panic(fmt.Sprintf("data: TierOrder holds %d tiers and Coverage is %d wide", len(TierOrder), TierCount))
	}
	if len(AffinityElements) != AffinityCount {
		panic(fmt.Sprintf("data: AffinityElements holds %d elements and Coverage is %d wide", len(AffinityElements), AffinityCount))
	}
}

// CoverageOf counts every record of a motif into its (tier, element) cells.
func CoverageOf(m MotifData) Coverage {
	c := Coverage{Motif: m.Motif}
	for _, r := range m.Records {
		ti, ok := TierIndex(r.Tier)
		if !ok {
			continue
		}
		for _, a := range r.Affinities {
			ai, ok := AffinityIndex(a)
			if !ok {
				continue
			}
			c.Counts[ti][ai]++
		}
	}
	return c
}

// Holes is every fight this motif cannot field MinCoverage ways, as readable phrases, in tier
// then element order. Empty means the motif covers.
func (c Coverage) Holes() []string {
	var out []string
	for ti, tier := range TierOrder {
		for ai, element := range AffinityElements {
			if n := c.Counts[ti][ai]; n < MinCoverage {
				out = append(out, fmt.Sprintf("%s %s: %d record(s), needs %d", element, tier, n, MinCoverage))
			}
		}
	}
	return out
}

// LoadMotifs reads every file under motifs/ and refuses anything the floor generator could not
// build a tower out of.
//
// **Every check here is a panic**, like the rest of this package. A motif with a hole in its
// coverage is not a record short — it is a floor the generator can offer and then fail to build,
// and the failure would land in front of a player mid-climb rather than in front of whoever
// authored the file.
func LoadMotifs() map[string]MotifData {
	entries, err := motifsFS.ReadDir("motifs")
	if err != nil {
		panic("motifs: " + err.Error())
	}

	// Sorted, because embed.FS walks in its own order and every check below reports the first
	// failure it finds — an unsorted walk would name a different file each launch.
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	out := make(map[string]MotifData, len(names))
	records := map[string]string{}

	for _, name := range names {
		file := path.Join("motifs", name)
		raw, err := motifsFS.ReadFile(file)
		if err != nil {
			panic(file + ": " + err.Error())
		}
		m := parseOne[MotifData](raw, file)

		checkMotif(m, file, name)
		if _, clash := out[m.Motif]; clash {
			panic(file + ": two files claim the motif " + m.Motif)
		}
		for _, r := range m.Records {
			if where, clash := records[r.Record]; clash {
				panic(file + ": record " + r.Record + " is also in " + where)
			}
			records[r.Record] = file
		}
		out[m.Motif] = m
	}

	if len(out) == 0 {
		panic("motifs: no motif files, so no floor can be built")
	}
	return out
}

// checkMotif is everything refusable about one file.
func checkMotif(m MotifData, file, name string) {
	if m.Motif == "" {
		panic(file + ": the file names no motif")
	}
	if stem := strings.TrimSuffix(name, ".json"); stem != m.Motif {
		panic(file + ": the file is named " + stem + " and the motif inside it is " + m.Motif)
	}
	if m.Name == "" {
		panic(file + ": " + m.Motif + " has no name")
	}
	if lo, hi := m.ValidFloors[0], m.ValidFloors[1]; lo < 0 || hi < 0 || (m.ValidFloors != [2]int{} && lo > hi) {
		panic(fmt.Sprintf("%s: %s has the floor band %v, which is not a band", file, m.Motif, m.ValidFloors))
	}
	if len(m.Records) == 0 {
		panic(file + ": " + m.Motif + " has no records")
	}
	for element := range m.ElementDraw {
		if _, ok := AffinityIndex(element); !ok {
			panic(file + ": " + m.Motif + " writes art direction for " + element + ", which is not an element")
		}
	}

	for _, r := range m.Records {
		checkRecord(m, r, file)
	}

	if holes := CoverageOf(m).Holes(); len(holes) > 0 {
		panic(fmt.Sprintf("%s: %s cannot field every fight — %s", file, m.Motif, strings.Join(holes, "; ")))
	}
}

// checkRecord is everything refusable about one creature.
func checkRecord(m MotifData, r MotifRecord, file string) {
	where := file + ": " + r.Record

	if r.Record == "" {
		panic(file + ": a record in " + m.Motif + " has no key")
	}
	if r.Name == "" {
		panic(where + " has no name")
	}
	if _, ok := TierIndex(r.Tier); !ok {
		panic(where + " is tier " + r.Tier + ", which is not one of outer, inner or boss")
	}
	if want := m.Motif + "-" + r.Tier + "-"; !strings.HasPrefix(r.Record, want) {
		panic(where + " should be keyed " + want + "<slug>")
	}
	if r.Title != "" && r.Tier != TierBoss {
		panic(where + " carries a title and is not a boss")
	}
	if r.Art == "" {
		panic(where + " names no art family")
	}

	if len(r.Affinities) == 0 {
		panic(where + " can be dealt as no element")
	}
	seen := map[string]bool{}
	for _, a := range r.Affinities {
		if _, ok := AffinityIndex(a); !ok {
			panic(where + " names the affinity " + a + ", which is not an element")
		}
		if seen[a] {
			panic(where + " names the affinity " + a + " twice")
		}
		seen[a] = true
	}

	if r.HP <= 0 || r.DMG <= 0 || r.Actions <= 0 {
		panic(fmt.Sprintf("%s has the stat line HP %d, DMG %d, Actions %d, and every one of them has to be positive", where, r.HP, r.DMG, r.Actions))
	}
	if len(r.Cards) == 0 {
		panic(where + " holds no cards, so it cannot fight")
	}
	for _, c := range r.Cards {
		if c.Label == "" {
			panic(where + " has a card with no label")
		}
		if c.Copies <= 0 {
			panic(where + ": " + c.Label + " has no copies")
		}
		if c.Cost < 1 || c.Cost > MaxCardCost {
			panic(fmt.Sprintf("%s: %s costs %d, and a card costs 1 to %d", where, c.Label, c.Cost, MaxCardCost))
		}
		if len(c.Elements) > 0 {
			panic(where + ": " + c.Label + " names its own elements, and a creature's colour is the floor's")
		}
	}
}

// MaxCardCost is the most action points one card may ask for. The cost column on a card face is
// tick marks stacked down a fixed band, so a fourth tick is a layout change rather than a bigger
// number.
const MaxCardCost = 3

// Brief is the whole art direction for one record dealt as one element, in the order it is read:
// what the motif shares, what the element does to this motif, and what this record is.
//
// **One function so a review sheet and a generated prompt cannot assemble it differently.** The
// four layers a picture is briefed from are the prompt file under docs/art/, which is about no
// record at all, and these three.
//
// Any layer may be empty — a brief nobody has written yet — and an empty one is left out rather
// than printed as a gap.
func (m MotifData) Brief(r MotifRecord, element string) []string {
	var out []string
	for _, part := range []string{m.Draw, m.ElementDraw[element], r.Draw} {
		if part != "" && part != DrawUnwritten {
			out = append(out, part)
		}
	}
	return out
}

// DrawUnwritten is what an unwritten brief says. It is a value rather than an empty string so a
// record that has been *looked at* and left reads differently from one nobody has reached, and
// the review sheet counts both.
const DrawUnwritten = "TO BE DETERMINED"

// MotifOrder is every motif key, sorted.
//
// **Sorted because LoadMotifs returns a map and Go randomizes that order.** A tower is built by
// drawing motifs against a seeded stream, so an unsorted walk would make a run code deal a
// different climb on every launch.
func MotifOrder(recs map[string]MotifData) []string {
	return sortedKeys(recs)
}

// MotifRecords is every record in every motif, keyed by its own key.
//
// The flat view, for the callers that hold a record key and need the record back — a saved run,
// a scenario fixture, a deck builder.
func MotifRecords(recs map[string]MotifData) map[string]MotifRecord {
	out := map[string]MotifRecord{}
	for _, key := range MotifOrder(recs) {
		for _, r := range recs[key].Records {
			out[r.Record] = r
		}
	}
	return out
}

// MotifOf is the motif a record key belongs to, which is the part of the key before its tier.
// It exists so a caller holding only a key can find the file it came from.
func MotifOf(recs map[string]MotifData, record string) (MotifData, bool) {
	for _, key := range MotifOrder(recs) {
		for _, r := range recs[key].Records {
			if r.Record == record {
				return recs[key], true
			}
		}
	}
	return MotifData{}, false
}

// MustBeClimbable refuses a roster that cannot give every floor of the tower a motif of its own.
//
// **A matching problem rather than a per-floor one**, and that is the whole reason it is a
// function. Three motifs that each say `[1, 2]` satisfy "floor 1 has a motif" and "floor 2 has a
// motif" while still leaving floor 3 empty — and a run never repeats a motif, so what has to
// hold is that the floors can be given *distinct* ones. This walks the floors and takes the
// motif with the fewest remaining choices first, which is exact for bands this shape.
func MustBeClimbable(recs map[string]MotifData, floors int) {
	if floors <= 0 {
		return
	}

	type slot struct {
		floor      int
		candidates []string
	}
	slots := make([]slot, 0, floors)
	for f := 1; f <= floors; f++ {
		var can []string
		for _, key := range MotifOrder(recs) {
			if recs[key].AllowsFloor(f) {
				can = append(can, key)
			}
		}
		if len(can) == 0 {
			panic(fmt.Sprintf("motifs: no motif may theme floor %d", f))
		}
		slots = append(slots, slot{floor: f, candidates: can})
	}

	// Fewest choices first, so a floor only one motif can theme takes it before a floor that
	// could have had anything spends it.
	sort.SliceStable(slots, func(i, j int) bool {
		return len(slots[i].candidates) < len(slots[j].candidates)
	})

	taken := map[string]int{}
	for _, s := range slots {
		placed := false
		for _, key := range s.candidates {
			if _, used := taken[key]; !used {
				taken[key] = s.floor
				placed = true
				break
			}
		}
		if !placed {
			panic(fmt.Sprintf("motifs: %d floors cannot each be given a motif of their own — floor %d has nothing left", floors, s.floor))
		}
	}
}
