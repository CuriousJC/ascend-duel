package data

// The backdrops: **the painted place a duel is fought in front of.**
//
// A record is a place belonging to one element, and a floor's element picks among that element's
// records — so a fire floor is fought in a fire place, and a catalog with several fire places
// gives two fire floors two different rooms. **A floor keeps one backdrop for all three of its
// rooms**: the climb reads as moving from one place to the next, rather than as the scenery
// changing under a player standing still.
//
// **Nothing in the rules reads this file.** A backdrop is a picture, and it may never change an
// outcome; `internal/screens` is its only reader.
//
// `Art` and `Draw` are the two fields every art-bearing catalog carries — see the data skill on the
// fields no catalog's rules read. The generic prompt is docs/art/background_art_prompt.MD.

import (
	_ "embed"
	"hash/fnv"
	"slices"
	"strconv"
)

//go:embed backgrounds.json
var backgroundsJSON []byte

// BackgroundData is one backdrop as written in the file.
type BackgroundData struct {
	// BackgroundRecord is the key. Kebab-case, like every other catalog's.
	BackgroundRecord string `json:"BackgroundRecord"`

	// Name is what a review page calls the place. Nothing in the game prints it.
	Name string `json:"Name"`

	// Element is which floors may be fought here, by the element's name. **One of
	// AffinityElements, refused at load otherwise** — a backdrop whose element no floor carries is
	// a picture nothing can ever draw, and nothing else would notice.
	Element string `json:"Element"`

	// Art is an assets.LoadImageData key. **Empty means the catalog's default** — see ArtKey.
	Art string `json:"Art"`

	// Draw is the subject paragraph an art generator is given: what the place is and what is lying
	// about in it, never who is in it. Ignored by the game.
	Draw string `json:"Draw"`
}

// LoadBackgrounds returns the catalog keyed by record, refusing a record whose element no floor
// can carry.
func LoadBackgrounds() map[string]BackgroundData {
	out := keyed(backgroundsJSON, "backgrounds.json",
		func(b BackgroundData) string { return b.BackgroundRecord })
	for key, b := range out {
		if !slices.Contains(AffinityElements, b.Element) {
			panic("backgrounds.json: " + key + " names element " + strconv.Quote(b.Element) +
				", which no floor carries")
		}
	}
	return out
}

// BackgroundOrder is every key, sorted — the walk anything whose result depends on order uses.
func BackgroundOrder(m map[string]BackgroundData) []string { return sortedKeys(m) }

// DefaultBackgroundArt is what a backdrop with no picture of its own draws, and what a floor
// whose element has no backdrop at all draws.
const DefaultBackgroundArt = "default-background"

// ArtKey is the picture this backdrop actually draws: its own if it has one, the default otherwise.
func (b BackgroundData) ArtKey() string {
	if b.Art == "" {
		return DefaultBackgroundArt
	}
	return b.Art
}

// BackdropFor is the picture a floor is fought in front of: one of its element's backdrops, or the
// default when the catalog has none for that element.
//
// **The choice is derived from the run and the floor, never rolled.** Every room on a floor asks
// the same question and gets the same answer, and a run code always shows the same places — with
// no stream to advance, because a picture may never change an outcome and so has nothing to share a
// cursor with. It is the crack pattern's rule: a presentation choice that must be stable is a
// function of what it is about.
func BackdropFor(catalog map[string]BackgroundData, element string, runSeed int64, floor int) string {
	var candidates []string
	for _, key := range BackgroundOrder(catalog) {
		if catalog[key].Element == element {
			candidates = append(candidates, key)
		}
	}
	if len(candidates) == 0 {
		return DefaultBackgroundArt
	}

	h := fnv.New64a()
	h.Write([]byte(strconv.FormatInt(runSeed, 10) + "/" + strconv.Itoa(floor) + "/" + element))
	return catalog[candidates[h.Sum64()%uint64(len(candidates))]].ArtKey()
}
